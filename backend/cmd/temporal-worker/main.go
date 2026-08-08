package main

import (
	"database/sql"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/lib/pq"
	"go.temporal.io/sdk/worker"
	"go.uber.org/zap"

	"github.com/hondyman/uisce/backend/internal/temporal"
)

func main() {
	// Parse flags
	temporalAddr := flag.String("temporal-address", "localhost:7233", "Temporal server address")
	namespace := flag.String("namespace", "default", "Temporal namespace")
	taskQueue := flag.String("task-queue", "analytics-worker", "Temporal task queue")
	dbURL := flag.String("database-url", "", "PostgreSQL database URL")
	controlDBURL := flag.String("control-database-url", "", "Control database URL (for tenant management)")
	flag.Parse()

	log.Printf("Starting Temporal Worker for analytics orchestration")
	log.Printf("  Temporal: %s", *temporalAddr)
	log.Printf("  Namespace: %s", *namespace)
	log.Printf("  Task Queue: %s", *taskQueue)

	// Get database URL from environment if not provided via flag
	databaseURL := *dbURL
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}
	controlDBURLVal := *controlDBURL
	if controlDBURLVal == "" {
		controlDBURLVal = os.Getenv("CONTROL_DATABASE_URL")
	}

	// Connect to PostgreSQL for activity operations
	var db *sql.DB
	var controlDB *sql.DB
	var logger *zap.SugaredLogger

	if databaseURL != "" {
		var err error
		db, err = sql.Open("postgres", databaseURL)
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
		defer db.Close()
		if err = db.Ping(); err != nil {
			log.Fatalf("Database ping failed: %v", err)
		}
		log.Println("Connected to PostgreSQL database")
	}

	if controlDBURLVal != "" {
		var err error
		controlDB, err = sql.Open("postgres", controlDBURLVal)
		if err != nil {
			log.Fatalf("Failed to connect to control database: %v", err)
		}
		defer controlDB.Close()
		if err = controlDB.Ping(); err != nil {
			log.Fatalf("Control database ping failed: %v", err)
		}
		log.Println("Connected to control database")
	}

	// Create logger
	zapLogger, _ := zap.NewProduction()
	logger = zapLogger.Sugar()
	defer logger.Sync()

	// Start worker
	w, err := temporal.StartWorker(temporal.WorkerConfig{
		TemporalServerAddress: *temporalAddr,
		Namespace:             *namespace,
		TaskQueue:             *taskQueue,
		DB:                    db,
		ControlDB:             controlDB,
		Logger:                logger,
	})
	if err != nil {
		log.Fatalf("Failed to start worker: %v", err)
	}

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Run worker in goroutine
	go func() {
		log.Println("Worker started, listening for tasks...")
		if err := w.Run(worker.InterruptCh()); err != nil {
			log.Fatalf("Worker error: %v", err)
		}
	}()

	// Wait for shutdown signal
	sig := <-sigCh
	log.Printf("Received signal: %v, shutting down...", sig)

	w.Stop()
	log.Println("Worker stopped gracefully")
}
