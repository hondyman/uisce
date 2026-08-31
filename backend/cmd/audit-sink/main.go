package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hondyman/uisce/backend/internal/audit"
)

func main() {
	log.Println("Starting Audit Sink Consumer...")

	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		kafkaBrokers = os.Getenv("REDPLANDA_BROKERS")
	}
	if kafkaBrokers == "" {
		kafkaBrokers = "localhost:19092"
	}
	groupID := os.Getenv("KAFKA_GROUP_ID")
	if groupID == "" {
		groupID = "audit-sink-consumer"
	}

	s3Endpoint := os.Getenv("S3_ENDPOINT")
	if s3Endpoint == "" {
		s3Endpoint = "http://localhost:9000"
	}
	s3AccessKey := os.Getenv("S3_ACCESS_KEY")
	if s3AccessKey == "" {
		s3AccessKey = "minioadmin"
	}
	s3SecretKey := os.Getenv("S3_SECRET_KEY")
	if s3SecretKey == "" {
		s3SecretKey = "minioadmin"
	}
	s3Bucket := os.Getenv("S3_BUCKET")
	if s3Bucket == "" {
		s3Bucket = "iceberg-warehouse"
	}
	catalogURI := os.Getenv("POLARIS_URI")
	if catalogURI == "" {
		catalogURI = os.Getenv("ICEBERG_CATALOG_URI")
		if catalogURI == "" {
			catalogURI = "http://localhost:8181"
		}
	}

	topics := []string{
		audit.TopicSchedulerJobRuns,
		audit.TopicSchedulerDAGRuns,
		audit.TopicGovernanceChangeSets,
		audit.TopicSemanticSnapshots,
		audit.TopicOrchestrationEvents,
		audit.TopicComplianceViolations,
		audit.TopicAISuggestions,
		audit.TopicExceptionLifecycle,
	}

	log.Printf("[Audit-Sink] Kafka Brokers: %s", kafkaBrokers)
	log.Printf("[Audit-Sink] S3 Endpoint: %s", s3Endpoint)
	log.Printf("[Audit-Sink] S3 Bucket: %s", s3Bucket)
	log.Printf("[Audit-Sink] Iceberg Catalog: %s", catalogURI)
	log.Printf("[Audit-Sink] Subscribing to topics: %v", topics)

	s3Client, err := audit.NewMinIOClient(s3Endpoint, s3AccessKey, s3SecretKey)
	if err != nil {
		log.Fatalf("[Audit-Sink] Failed to create MinIO client: %v", err)
	}
	log.Printf("[Audit-Sink] MinIO client connected to %s", s3Endpoint)

	icebergWriter := &audit.IcebergWriter{
		S3Client:   s3Client,
		BucketName: s3Bucket,
		CatalogURI: catalogURI,
	}

	consumer, err := audit.NewIcebergSinkConsumer(kafkaBrokers, groupID, topics, icebergWriter)
	if err != nil {
		log.Fatalf("[Audit-Sink] Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	log.Println("[Audit-Sink] Consumer created successfully")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	errChan := make(chan error, 1)
	go func() {
		errChan <- consumer.Start(ctx)
	}()

	log.Printf("[Audit-Sink] Running. Press Ctrl+C to stop.")

	select {
	case <-sigChan:
		log.Println("[Audit-Sink] Received shutdown signal")
		cancel()
		time.Sleep(2 * time.Second)
	case err := <-errChan:
		if err != nil && err != context.Canceled {
			log.Printf("[Audit-Sink] Consumer error: %v", err)
		}
	}

	log.Println("[Audit-Sink] Stopped")
}
