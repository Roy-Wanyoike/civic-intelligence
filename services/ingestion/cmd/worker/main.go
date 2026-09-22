// cmd/worker is the Temporal worker process for the Bill Processing Pipeline.
//
// It connects to a Temporal server (Temporal CLI / Temporalite locally, the
// shared Temporal cluster in staging/prod), registers the workflow + its
// activities, and starts polling the `bill-processing` task queue.
//
// Run locally:
//
//	# 1. Start Temporal (dev server, web UI at http://localhost:8080):
//	temporal server start-dev
//
//	# 2. Start the worker:
//	go run ./cmd/worker
//
//	# 3. Trigger a workflow (separate terminal):
//	temporal workflow execute \
//	  --task-queue bill-processing \
//	  --type BillProcessingWorkflow \
//	  --input '{"country_code":"KE","bill_id":"00000000-0000-0000-0000-000000000001","source_url":"https://www.parliament.go.ke/bills/14-2024","identifier":"Bill No. 14 of 2024","year":2024}'
//
// See services/ingestion/internal/temporal/README.md for the full local runbook.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/registry"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/config"
	"github.com/Roy-Wanyoike/civic-intelligence/services/ingestion/internal/temporal"
)

// Config is loaded from the environment (see packages/config).
type Config struct {
	TemporalAddress   string        `env:"TEMPORAL_ADDRESS" default:"127.0.0.1:7233"`
	TemporalNamespace string        `env:"TEMPORAL_NAMESPACE" default:"default"`
	TaskQueue         string        `env:"TEMPORAL_TASK_QUEUE" default:"bill-processing"`
	WorkerConcurrent  int           `env:"TEMPORAL_WORKER_CONCURRENT" default:"50"`
	ShutdownTimeout   time.Duration `env:"SHUTDOWN_TIMEOUT" default:"15s"`
	LogLevel          string        `env:"LOG_LEVEL" default:"info"`
}

// TaskQueueBillProcessing is the canonical task queue name for the Bill
// Processing Pipeline. Both workers and starters must agree on this string.
const TaskQueueBillProcessing = "bill-processing"

func main() {
	var cfg Config
	if err := config.Load(&cfg); err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLevel(cfg.LogLevel),
	})).With("service", "ingestion-worker", "task_queue", cfg.TaskQueue)

	// 1. Register every shipped country adapter (Kenya, Uganda, Tanzania,
	// Ghana, Nigeria, South Africa, Rwanda, Zambia, Senegal, Egypt, Morocco,
	// DR Congo, Ethiopia, Malawi) with the central adapters/registry. The
	// Discover activity looks adapters up by country code via this registry,
	// so this MUST happen before the workflow is constructed. Idempotent —
	// safe to call again if another binary in the same process already did.
	registry.MustRegisterDefault()
	log.Info("registered source adapters",
		"count", len(registry.SupportedCountries()))

	// 2. Temporal client
	c, err := client.Dial(client.Options{
		HostPort:  cfg.TemporalAddress,
		Namespace: cfg.TemporalNamespace,
		Logger:    slogAdapter{log},
	})
	if err != nil {
		log.Error("temporal client dial failed", "err", err)
		os.Exit(1)
	}
	defer c.Close()

	// 3. Wire activities with their dependencies. The Discover activity is
	// wired with the global SourceRegistry (backed by adapters/registry);
	// the other activities are still nil-zero because their concrete
	// infrastructure clients are environment-specific — see README.md
	// "Wiring dependencies" section.
	wf := buildWorkflow()

	// 4. Worker
	w := worker.New(c, cfg.TaskQueue, worker.Options{
		MaxConcurrentActivityExecutionSize: cfg.WorkerConcurrent,
	})
	w.RegisterWorkflow(wf.Workflow)

	// Register each activity. Each activity method is named after its step
	// (Discover, Fetch, Parse, Extract, Validate, Publish) so the Temporal
	// Go SDK — which derives the activity type name from the method name —
	// registers them under unique names. The workflow's ExecuteActivity
	// calls use the same method values, so the names match automatically.
	w.RegisterActivity(wf.Discover.Discover)
	w.RegisterActivity(wf.Fetch.Fetch)
	w.RegisterActivity(wf.Parse.Parse)
	w.RegisterActivity(wf.Extract.Extract)
	w.RegisterActivity(wf.Validate.Validate)
	w.RegisterActivity(wf.Publish.Publish)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log.Info("starting temporal worker",
		"address", cfg.TemporalAddress,
		"namespace", cfg.TemporalNamespace,
		"concurrent", cfg.WorkerConcurrent)

	go func() {
		if err := w.Run(worker.InterruptCh()); err != nil {
			log.Error("worker exited with error", "err", err)
			cancel()
		}
	}()

	<-ctx.Done()
	log.Info("shutdown signal received, stopping worker")
	w.Stop()
	log.Info("worker stopped")
}

// uuidGen is a minimal IDGenerator that produces a v4-like hex string.
// In production, swap for github.com/google/uuid. It is duplicated here so
// the worker compiles without dragging in extra deps.
type uuidGen struct{}

func (uuidGen) New() string {
	return fmt.Sprintf("run-%d", time.Now().UnixNano())
}

// buildWorkflow constructs the Bill Processing Pipeline workflow with all
// activities wired to their concrete dependencies. It is extracted from main
// so it can be unit-tested (see main_test.go).
//
// The Discover activity's SourceRegistry is wired to a registrySourceRegistry
// that delegates to the global adapters/registry package — main() calls
// registry.MustRegisterDefault() before this so the country adapters are
// already registered. The other activities still have nil dependencies
// because their concrete infrastructure clients are environment-specific
// (see README.md "Wiring dependencies").
func buildWorkflow() *temporal.BillProcessingWorkflowImpl {
	return temporal.NewBillProcessingWorkflow(
		&temporal.DiscoverActivity{Registry: registrySourceRegistry{}},
		&temporal.FetchActivity{Fetcher: nil, Storage: nil, IDGen: uuidGen{}},
		&temporal.ParseActivity{Documents: nil},
		&temporal.ExtractActivity{AI: nil},
		&temporal.ValidateActivity{Validator: nil, Evidence: nil},
		&temporal.PublishActivity{Publisher: nil, Legislation: nil},
	)
}

// registrySourceRegistry adapts the global adapters/registry package to the
// temporal.SourceRegistry interface. The Discover activity calls
// AdapterFor(countryCode) to confirm the country is supported and Discover()
// to enumerate the country's currently-known source items (bills today;
// extensible to acts, gazettes, etc. tomorrow).
//
// It is a thin value-typed wrapper: no fields, no state — all state lives in
// the global registry package. That keeps the worker bootstrap stateless
// and the registry the single source of truth for "which countries are
// supported" (ADR-0004).
type registrySourceRegistry struct{}

// Compile-time assertion that registrySourceRegistry satisfies the
// temporal.SourceRegistry interface. Catches signature drift at build time
// rather than at activity-execution time.
var _ temporal.SourceRegistry = registrySourceRegistry{}

// AdapterFor implements temporal.SourceRegistry. It looks up the country
// adapter via registry.GetAdapter and returns the adapter's own canonical
// country code (an identity check that confirms the country is registered).
// Unknown country codes propagate registry.ErrUnsupportedCountry.
func (registrySourceRegistry) AdapterFor(countryCode string) (string, error) {
	a, err := registry.GetAdapter(countryCode)
	if err != nil {
		return "", err
	}
	return a.CountryCode(), nil
}

// Discover implements temporal.SourceRegistry. It looks up the country
// adapter and asks it for currently-discovered bills, mapping each
// registry.BillCandidate to a temporal.SourceItem. The sourceURL and
// identifier inputs are preserved on each item's Metadata so downstream
// activities (Fetch, Parse) can correlate the item back to the workflow
// input even when the adapter returns multiple candidates.
func (registrySourceRegistry) Discover(ctx context.Context, countryCode, sourceURL, identifier string) ([]temporal.SourceItem, error) {
	a, err := registry.GetAdapter(countryCode)
	if err != nil {
		return nil, err
	}
	bills, err := a.DiscoverBills(ctx)
	if err != nil {
		return nil, fmt.Errorf("discover bills for %s: %w", countryCode, err)
	}
	items := make([]temporal.SourceItem, 0, len(bills))
	for _, b := range bills {
		items = append(items, temporal.SourceItem{
			URL:        b.SourceURL,
			Identifier: b.Number,
			Kind:       "bill",
			Title:      b.Title,
			Metadata: map[string]string{
				"country_code": b.CountryCode,
				"house":        b.House,
				"stage":        b.Stage,
				"sponsor":      b.Sponsor,
				// Echo the workflow input so downstream activities
				// can correlate items back to the trigger even when
				// the adapter returns several candidates.
				"workflow_source_url": sourceURL,
				"workflow_identifier": identifier,
			},
		})
	}
	return items, nil
}

func parseLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// slogAdapter bridges slog → the Temporal client.Logger interface. Temporal
// expects a logger with keyed args; slog already does that.
type slogAdapter struct{ l *slog.Logger }

func (s slogAdapter) Debug(msg string, keyvals ...interface{}) { s.l.Debug(msg, keyvals...) }
func (s slogAdapter) Info(msg string, keyvals ...interface{})  { s.l.Info(msg, keyvals...) }
func (s slogAdapter) Warn(msg string, keyvals ...interface{})  { s.l.Warn(msg, keyvals...) }
func (s slogAdapter) Error(msg string, keyvals ...interface{}) { s.l.Error(msg, keyvals...) }
