package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/config"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/observability"
)

// Config holds the ingestion service's runtime configuration.
type Config struct {
	HTTPAddr        string        `env:"HTTP_ADDR" default:":8082"`
	NATSURL         string        `env:"NATS_URL" default:"nats://127.0.0.1:4222"`
	DBURL           string        `env:"DATABASE_URL" default:"postgres://civic:civic@localhost:5432/civic?sslmode=disable"`
	LogLevel        string        `env:"LOG_LEVEL" default:"info"`
	WorkerCount     int           `env:"WORKER_COUNT" default:"4"`
	FetchBatchSize  int           `env:"FETCH_BATCH_SIZE" default:"50"`
	UserAgent       string        `env:"USER_AGENT" default:""`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" default:"15s"`
}

func main() {
	var cfg Config
	if err := config.Load(&cfg); err != nil {
		_, _ = os.Stderr.WriteString("config: " + err.Error() + "\n")
		os.Exit(1)
	}
	log := observability.NewLogger(os.Stderr, cfg.LogLevel, "ingestion")
	log.Info("starting ingestion service",
		observability.String("addr", cfg.HTTPAddr),
		observability.Int("workers", cfg.WorkerCount))

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})

	srv := &http.Server{Addr: cfg.HTTPAddr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("http server error", err)
		os.Exit(1)
	}
	log.Info("ingestion service stopped")
}
