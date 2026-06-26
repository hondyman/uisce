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

// AuditEvent represents the structure of an audit log.
type AuditEvent struct {
	ID           uuid.UUID         `json:"id"`
	EventType    string            `json:"event_type"`
	TenantID     uuid.UUID         `json:"tenant_id"`
	ActorID      string            `json:"actor_id"`
	Action       string            `json:"action"`
	ResourceID   string            `json:"resource_id"`
	ResourceType string            `json:"resource_type"`
	Metadata     map[string]string `json:"metadata"`
	Timestamp    time.Time         `json:"timestamp"`
}

func main() {
	log.Println("Starting Audit Worker...")

	// Kafka configuration
	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		kafkaBrokers = "localhost:9092"
	}
	topic := "Audit.Event" // Subscribe to generic audit events
	groupID := "audit-worker"

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

	// Trino configuration (Placeholder for now)
	// trinoDSN := os.Getenv("TRINO_DSN")
	// For MVP, we will just log the audit events. Real implementation would write to Iceberg via Trino.

	log.Println("Audit Worker ready to process events")

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

		if err := processAuditEvent(ctx, m.Value); err != nil {
			log.Printf("Error processing audit event: %v", err)
		}
	}
}

func processAuditEvent(ctx context.Context, data []byte) error {
	var event AuditEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return err
	}

	// Logic to write to Trino/Iceberg would go here.
	// For now, we simulate by logging structured data.
	log.Printf("[AUDIT] Tenant: %s | User: %s | Action: %s | Resource: %s/%s",
		event.TenantID, event.ActorID, event.Action, event.ResourceType, event.ResourceID)

	return nil
}
