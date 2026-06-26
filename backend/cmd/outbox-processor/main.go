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
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/segmentio/kafka-go"
)

// OutboxEvent represents a row in the outbox table.
type OutboxEvent struct {
	ID        uuid.UUID       `db:"id"`
	EventType string          `db:"event_type"`
	Payload   json.RawMessage `db:"payload"`
	Published bool            `db:"published"`
	CreatedAt time.Time       `db:"created_at"`
}

func main() {
	log.Println("Starting Outbox Processor...")

	// Connect to PostgreSQL
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Println("Connected to PostgreSQL")

	// Kafka configuration
	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		kafkaBrokers = "localhost:9092"
	}

	log.Printf("Kafka brokers: %s", kafkaBrokers)

	// Writer for publishing messages
	// We use a shared writer for now, or we could create one per topic if needed.
	// segmentio/kafka-go Writer is safe for concurrent use.
	writer := &kafka.Writer{
		Addr:     kafka.TCP(kafkaBrokers),
		Balancer: &kafka.LeastBytes{},
	}
	defer writer.Close()

	// Handle shutdown gracefully
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sig := <-sigchan
		log.Printf("Caught signal %v: terminating", sig)
		cancel()
	}()

	log.Println("Starting processor loop...")

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := processBatch(ctx, db, writer); err != nil {
				log.Printf("Error processing batch: %v", err)
			}
		}
	}
}

func processBatch(ctx context.Context, db *sqlx.DB, writer *kafka.Writer) error {
	// Start transaction
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Select unpublished events with SKIP LOCKED for concurrency
	q := `
		SELECT id, event_type, payload, published, created_at
		FROM outbox
		WHERE published = FALSE
		ORDER BY created_at ASC
		LIMIT 50
		FOR UPDATE SKIP LOCKED
	`

	var events []OutboxEvent
	if err := tx.SelectContext(ctx, &events, q); err != nil {
		return err
	}

	if len(events) == 0 {
		return nil
	}

	log.Printf("Processing %d events", len(events))

	var processedIDs []string
	var messages []kafka.Message

	for _, evt := range events {
		// Prepare Kafka message
		// We use the event type as the topic
		// In a real system, you might map event types to specific topics
		topic := evt.EventType

		// Ensure payload is valid JSON
		payloadBytes := []byte(evt.Payload)

		// Key is usually entity ID but we don't have it easily generic here.
		// We can use Event ID as key or leave empty.
		// For ordering guarantees, we'd want the entity ID as key.
		// For now, let's use the Event ID as key.
		key := evt.ID.String()

		messages = append(messages, kafka.Message{
			Topic: topic,
			Key:   []byte(key),
			Value: payloadBytes,
		})

		processedIDs = append(processedIDs, evt.ID.String())
	}

	// Publish to Kafka
	// We do this *before* committing DB transaction to ensure at-least-once delivery.
	// If Kafka fails, we rollback DB (events stay in outbox).
	// If DB commit fails after Kafka publish, we might re-publish (duplicates), which consumers must handle.

	// Note: segmentio Writer.WriteMessages will try to create topics if allowed by broker.
	if err := writer.WriteMessages(ctx, messages...); err != nil {
		return err
	}

	// Update outbox status
	if len(processedIDs) > 0 {
		query, args, err := sqlx.In(`UPDATE outbox SET published = TRUE, published_at = NOW() WHERE id IN (?)`, processedIDs)
		if err != nil {
			return err
		}

		// sqlx.In expands ? to ?, ?, ? ... we need to replace with $1, $2 etc for postgres
		query = db.Rebind(query)

		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return err
		}
	}

	return tx.Commit()
}
