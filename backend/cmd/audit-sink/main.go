package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hondyman/semlayer/backend/internal/audit"
)

func main() {
	log.Println("Starting Audit Sink Consumer...")

	// Configuration from environment
	kafkaBrokers := getEnv("KAFKA_BROKERS", "localhost:19092")
	groupID := getEnv("KAFKA_GROUP_ID", "audit-sink-consumer")
	s3Endpoint := getEnv("S3_ENDPOINT", "http://localhost:9000")
	// s3AccessKey := getEnv("S3_ACCESS_KEY", "minioadmin")    // For S3 authentication
	// s3SecretKey := getEnv("S3_SECRET_KEY", "minioadmin")    // For S3 authentication
	s3Bucket := getEnv("S3_BUCKET", "audit")
	icebergCatalogURI := getEnv("ICEBERG_CATALOG_URI", "http://localhost:8181")

	// Kafka topics to consume
	topics := []string{
		audit.TopicSchedulerJobRuns,
		audit.TopicSchedulerDAGRuns,
		audit.TopicGovernanceChangeSets,
		audit.TopicSemanticSnapshots,
		audit.TopicOrchestrationEvents,
		audit.TopicComplianceViolations,
		audit.TopicAISuggestions,
	}

	log.Printf("Kafka Brokers: %s", kafkaBrokers)
	log.Printf("S3 Endpoint: %s", s3Endpoint)
	log.Printf("S3 Bucket: %s", s3Bucket)
	log.Printf("Iceberg Catalog: %s", icebergCatalogURI)
	log.Printf("Subscribing to topics: %v", topics)

	// Initialize Iceberg writer
	// For now this is a placeholder - full implementation would use:
	// github.com/minio/minio-go/v7
	// S3Client would be initialized here with MinIO client
	icebergWriter := &audit.IcebergWriter{
		BucketName: s3Bucket,
		CatalogURI: icebergCatalogURI,
	}

	// Create Kafka consumer
	consumer, err := audit.NewIcebergSinkConsumer(kafkaBrokers, groupID, topics, icebergWriter)
	if err != nil {
		log.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	log.Println("Consumer created successfully")

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start consumer in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- consumer.Start(ctx)
	}()

	log.Println("Audit Sink Consumer is running. Press Ctrl+C to stop.")

	// Wait for shutdown signal or error
	select {
	case <-sigChan:
		log.Println("Received shutdown signal")
		cancel()
		time.Sleep(2 * time.Second) // Give consumer time to finish processing
	case err := <-errChan:
		if err != nil {
			log.Printf("Consumer error: %v", err)
		}
	}

	log.Println("Audit Sink Consumer stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
