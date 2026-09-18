// Package legislation — main entrypoint. Wires up dependencies and starts
// the HTTP server. In production this calls the legislation application
// service via internal HTTP or NATS. For local testing it can run standalone.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	addr := os.Getenv("LEGISLATION_SERVICE_ADDR")
	if addr == "" {
		addr = ":9001"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"legislation"}`))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ready","service":"legislation"}`))
	})

	srv := &http.Server{Addr: addr, Handler: mux, ReadTimeout: 10 * time.Second, WriteTimeout: 30 * time.Second}

	go func() {
		log.Printf("legislation service listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("legislation service shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
