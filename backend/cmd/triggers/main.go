package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/triggers"
	temporalclient "github.com/hondyman/semlayer/libs/temporal-client"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.temporal.io/sdk/client"
)

func main() {
	// Simple wiring: read PG and Temporal URLs from env
	pgURL := os.Getenv("DATABASE_URL")
	if pgURL == "" {
		pgURL = "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable"
	}

	db, err := sqlx.Connect("postgres", pgURL)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer db.Close()

	var temporalClient client.Client
	tc, err := temporalclient.NewClientWithRetry()
	if err != nil {
		log.Printf("⚠️  WARNING: failed to create temporal client: %v", err)
		log.Printf("    To enable workflow execution, start Temporal server and set TEMPORAL_URL")
		log.Printf("    Continuing in test mode (workflows will be logged but not executed)\n")
		temporalClient = nil
	} else {
		temporalClient = tc
		defer temporalClient.Close()
	}

	// Initialize trigger engine
	engine := triggers.NewTriggerEngine(temporalClient, (*sql.DB)(db.DB))

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go func() {
		if err := engine.StartEscalationMonitor(ctx); err != nil {
			log.Printf("escalation monitor stopped: %v", err)
		}
	}()

	go func() {
		if err := engine.StartEventListener(ctx, pgURL); err != nil {
			log.Printf("event listener stopped: %v", err)
		}
	}()

	// Minimal HTTP server for health
	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, r2 *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	})
	srv := &http.Server{Addr: ":29090", Handler: r}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server failed: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down trigger engine")
	ctxShut, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	_ = srv.Shutdown(ctxShut)
}
