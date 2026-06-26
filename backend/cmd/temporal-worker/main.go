package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.temporal.io/sdk/worker"

	"github.com/hondyman/semlayer/backend/internal/temporal"
)

func main() {
	// Parse flags
	temporalAddr := flag.String("temporal-address", "localhost:7233", "Temporal server address")
	namespace := flag.String("namespace", "default", "Temporal namespace")
	taskQueue := flag.String("task-queue", "analytics-worker", "Temporal task queue")
	flag.Parse()

	log.Printf("Starting Temporal Worker for analytics orchestration")
	log.Printf("  Temporal: %s", *temporalAddr)
	log.Printf("  Namespace: %s", *namespace)
	log.Printf("  Task Queue: %s", *taskQueue)

	// Start worker
	w, err := temporal.StartWorker(temporal.WorkerConfig{
		TemporalServerAddress: *temporalAddr,
		Namespace:             *namespace,
		TaskQueue:             *taskQueue,
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
