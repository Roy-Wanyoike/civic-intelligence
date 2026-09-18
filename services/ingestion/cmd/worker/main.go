//
// cmd/worker is the Temporal worker process for the Bill Processing Pipeline.
//
// It connects to a Temporal server (Temporal CLI / Temporalite locally, the
// shared Temporal cluster in staging/prod), registers the workflow + its
// activities, and starts polling the `bill-processing` task queue.
//
// Run locally:
//
//      # 1. Start Temporal (dev server, web UI at http://localhost:8080):
//      temporal server start-dev
//
//      # 2. Start the worker:
//      go run ./cmd/worker
//
//      # 3. Trigger a workflow (separate terminal):
//      temporal workflow execute \
//        --task-queue bill-processing \
//        --type BillProcessingWorkflow \
//        --input '{"country_code":"KE","bill_id":"00000000-0000-0000-0000-000000000001","source_url":"https://www.parliament.go.ke/bills/14-2024","identifier":"Bill No. 14 of 2024","year":2024}'
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

        // 1. Temporal client
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

        // 2. Wire activities with their dependencies. In a real deployment these
        // concrete dependencies are constructed from infrastructure clients
        // (Postgres pool, NATS conn, S3 client, AI gateway HTTP client). They are
        // intentionally left as nil-zero values here because the wiring is
        // environment-specific — see README.md "Wiring dependencies" section.
        wf := temporal.NewBillProcessingWorkflow(
                &temporal.DiscoverActivity{Registry: nil /* TODO: wire SourceRegistry */},
                &temporal.FetchActivity{Fetcher: nil, Storage: nil, IDGen: uuidGen{}},
                &temporal.ParseActivity{Documents: nil},
                &temporal.ExtractActivity{AI: nil},
                &temporal.ValidateActivity{Validator: nil, Evidence: nil},
                &temporal.PublishActivity{Publisher: nil, Legislation: nil},
        )

        // 3. Worker
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
