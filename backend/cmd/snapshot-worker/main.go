package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// SnapshotEvent represents the structure of a generic snapshot event.
// This is typically emitted after an entity is modified to capture its current state.
type SnapshotEvent struct {
	ID           uuid.UUID       `json:"id"`
	EntityID     string          `json:"entity_id"`
	EntityType   string          `json:"entity_type"`
	TenantID     uuid.UUID       `json:"tenant_id"`
	SnapshotData json.RawMessage `json:"snapshot_data"` // The full entity state
	Version      int64           `json:"version"`
	Timestamp    time.Time       `json:"timestamp"`
}

func main() {
	log.Println("Starting Snapshot Worker...")

	// Kafka configuration
	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		kafkaBrokers = "localhost:9092"
	}
	topic := "Entity.Snapshot" // Generic snapshot topic
	groupID := "snapshot-worker"

	log.Printf("Connecting to Kafka at %s, topic: %s", kafkaBrokers, topic)

	// Create Kafka reader
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{kafkaBrokers},
		GroupID:  groupID,
		Topic:    topic,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})
	defer reader.Close()

	// Trino configuration would go here
	// For MVP, we log the snapshot event.

	log.Println("Snapshot Worker ready to process events")

	// Handle shutdown gracefully
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sig := <-sigchan
		log.Printf("Caught signal %v: terminating", sig)
		cancel()
		reader.Close()
	}()

	for {
		m, err := reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			log.Printf("Error reading message: %v", err)
			time.Sleep(time.Second)
			continue
		}

		if err := processSnapshotEvent(ctx, m.Value); err != nil {
			log.Printf("Error processing snapshot event: %v", err)
		}
	}
}

func processSnapshotEvent(ctx context.Context, data []byte) error {
	var event SnapshotEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return err
	}

	// Logic to write to Trino/Iceberg Bitemporal tables goes here.
	// We would insert into a history table with VT (valid time) and TT (transaction time).

	log.Printf("[SNAPSHOT] Tenant: %s | Entity: %s/%s | Version: %d",
		event.TenantID, event.EntityType, event.EntityID, event.Version)

	return nil
}
