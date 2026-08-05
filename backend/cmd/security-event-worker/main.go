package main

import (
	"fmt"
	"log"
	"os"

	"github.com/hondyman/uisce/backend/internal/workers"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		panic("DATABASE_URL environment variable is required")
	}

	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		kafkaBrokers = "localhost:9092"
	}

	// Connect to database
	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	fmt.Printf("Starting security event worker...\n")
	fmt.Printf("Database: %s\n", dbURL)
	fmt.Printf("Kafka: %s\n", kafkaBrokers)
	fmt.Printf("Publishing to topics: security.audit, security.snapshot\n")

	// Run worker
	if err := workers.RunSecurityEventWorker(db, kafkaBrokers); err != nil {
		log.Fatalf("Worker error: %v", err)
	}
}
