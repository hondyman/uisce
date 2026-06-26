package workers

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hondyman/semlayer/backend/internal/events"
	"github.com/jmoiron/sqlx"
)

// SecurityEventWorker processes security events from outbox to Kafka.
type SecurityEventWorker struct {
	db        *sqlx.DB
	publisher *events.KafkaSecurityPublisher
	interval  time.Duration
}

// NewSecurityEventWorker creates a new security event worker.
func NewSecurityEventWorker(db *sqlx.DB, kafkaBrokers string) *SecurityEventWorker {
	return &SecurityEventWorker{
		db:        db,
		publisher: events.NewKafkaSecurityPublisher(kafkaBrokers),
		interval:  5 * time.Second, // Process every 5 seconds
	}
}

// Start begins processing events in the background.
func (w *SecurityEventWorker) Start(ctx context.Context) error {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	fmt.Println("Security event worker started")

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Security event worker stopping...")
			return ctx.Err()

		case <-ticker.C:
			if err := w.processEvents(ctx); err != nil {
				fmt.Printf("Error processing security events: %v\n", err)
				// Continue processing despite errors
			}
		}
	}
}

// processEvents processes a batch of security events from the outbox.
func (w *SecurityEventWorker) processEvents(ctx context.Context) error {
	return events.ProcessSecurityOutbox(ctx, w.db, w.publisher)
}

// RunSecurityEventWorker starts the security event worker and blocks until interrupted.
func RunSecurityEventWorker(db *sqlx.DB, kafkaBrokers string) error {
	worker := NewSecurityEventWorker(db, kafkaBrokers)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	errChan := make(chan error, 1)
	go func() {
		errChan <- worker.Start(ctx)
	}()

	select {
	case <-sigChan:
		fmt.Println("Received interrupt signal, shutting down...")
		cancel()
		return nil
	case err := <-errChan:
		return err
	}
}
