package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/pkg/workflows"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"go.temporal.io/sdk/client"
)

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Get Temporal address
	temporalAddr := os.Getenv("TEMPORAL_ADDRESS")
	if temporalAddr == "" {
		temporalAddr = "localhost:7233"
	}

	// Get namespace
	namespace := os.Getenv("TEMPORAL_NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}

	log.Printf("Connecting to Temporal at %s (namespace: %s)...", temporalAddr, namespace)

	// Create Temporal client
	c, err := client.Dial(client.Options{
		HostPort:  temporalAddr,
		Namespace: namespace,
	})
	if err != nil {
		log.Fatalf("Failed to create Temporal client: %v", err)
	}
	defer c.Close()

	log.Println("Connected to Temporal successfully")

	// Connect to Postgres for DB-backed activities (currently just
	// ActivityCalculation, the centralized calc engine — see
	// pkg/workflows/calc_activities.go). Without this, "Calculation"
	// workflow nodes fail at runtime rather than silently no-op.
	var deps *workflows.ActivityDeps
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		db, err := sqlx.Open("postgres", dbURL)
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
		defer db.Close()
		if err := db.Ping(); err != nil {
			log.Fatalf("Database ping failed: %v", err)
		}
		log.Println("Connected to PostgreSQL database")
		deps = &workflows.ActivityDeps{CalcService: analytics.NewSemanticCalculationService(db)}
	} else {
		log.Println("WARNING: DATABASE_URL not set — \"Calculation\" workflow nodes will fail at runtime")
	}

	// Create and start the BP worker
	worker := workflows.NewBPWorker(c, deps)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start worker in goroutine
	go func() {
		if err := worker.Start(); err != nil {
			log.Fatalf("Worker failed: %v", err)
		}
	}()

	log.Println("BP Framework Worker is running. Press Ctrl+C to stop.")

	// Wait for interrupt
	<-sigChan
	log.Println("Received interrupt signal, shutting down...")

	worker.Stop()
	log.Println("BP Framework Worker stopped")
}
