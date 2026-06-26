package main

import (
	"fmt"
	"log"
	"os"

	"github.com/hondyman/semlayer/backend/internal/workers"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	// Get configuration from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable"
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
