// Package main is the entrypoint for the Civic Intelligence API / BFF.
//
// This service is the citizen-facing HTTP gateway. It:
//   - validates OIDC JWTs (in production)
//   - enforces RBAC per endpoint
//   - rate-limits per IP and per user
//   - calls the domain services (legislation, search, intelligence, etc.)
//   - NEVER touches the database directly
//
// In Phase 1, this is a minimal health/readiness server. Full REST endpoints
// are tracked in issue #19 (complete the Go backend services).
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
	addr := os.Getenv("API_SERVICE_ADDR")
	if addr == "" {
		addr = ":9000"
	}

	mux := http.NewServeMux()

	// Health/readiness — required for Kubernetes probes and for the
	// Next.js frontend's /api/v1/* rewrite target.
	mux.HandleFunc("/api/v1/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"api","version":"0.1.0"}`))
	})
	mux.HandleFunc("/api/v1/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// TODO(issue #19): check downstream services (legislation, search,
		// AI) before reporting ready.
		w.Write([]byte(`{"status":"ready","service":"api","downstream":"unchecked"}`))
	})

	// TODO(issue #19): implement the full REST API per docs/api/openapi.yaml:
	//   GET  /api/v1/bills
	//   GET  /api/v1/bills/{id}
	//   GET  /api/v1/bills/{id}/timeline
	//   GET  /api/v1/bills/{id}/versions
	//   POST /api/v1/bills/{id}/follow
	//   GET  /api/v1/search
	//   GET  /api/v1/briefing
	//   GET  /api/v1/people/{id}
	//   GET  /api/v1/committees/{id}
	//   GET  /api/v1/institutions/{id}
	//   POST /api/v1/questions (SSE streaming to AI service)

	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("civic-api listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("civic-api shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
