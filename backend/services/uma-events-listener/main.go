package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"github.com/hondyman/semlayer/backend/internal/events"
	kafka "github.com/segmentio/kafka-go"
)

// ============================================================================
// UMA EVENT LISTENER
// Listens for UMA events from Kafka and processes them
// ============================================================================

type UMAEventListener struct {
	reader *kafka.Reader
}

// NewUMAEventListener creates a new listener
func NewUMAEventListener(brokers string) (*UMAEventListener, error) {
	brokerList := strings.Split(brokers, ",")
	topic := "uma.events"

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokerList,
		GroupID:  "uma-event-listener-group",
		Topic:    topic,
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})

	return &UMAEventListener{
		reader: r,
	}, nil
}

// ============================================================================
// EVENT HANDLERS
// ============================================================================

// HandleRebalanceRequested processes rebalance requested events
func (l *UMAEventListener) HandleRebalanceRequested(msg kafka.Message) error {
	var event events.UMARebalanceRequestedEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		log.Printf("❌ Failed to unmarshal event: %v", err)
		return err
	}

	log.Printf("📥 Received RebalanceRequested event:")
	log.Printf("  RequestID: %s", event.RequestID)
	log.Printf("  UMAAccountID: %s", event.UMAAccountID)
	log.Printf("  Type: %s", event.RequestType)

	// TODO: Process rebalance request
	return nil
}

// HandleSleeveDriftDetected processes sleeve drift detected events
func (l *UMAEventListener) HandleSleeveDriftDetected(msg kafka.Message) error {
	var event events.SleeveDriftDetectedEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		log.Printf("❌ Failed to unmarshal event: %v", err)
		return err
	}

	log.Printf("📥 Received SleeveDriftDetected event:")
	log.Printf("  UMAAccountID: %s", event.UMAAccountID)
	log.Printf("  SleeveType: %s", event.SleeveType)
	log.Printf("  Drift: %.2f%%", event.DriftPercent*100)

	// TODO: Store drift metric
	return nil
}

// HandleTaxHarvestSimulated processes tax harvest simulated events
func (l *UMAEventListener) HandleTaxHarvestSimulated(msg kafka.Message) error {
	var event events.TaxHarvestSimulatedEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		log.Printf("❌ Failed to unmarshal event: %v", err)
		return err
	}

	log.Printf("📥 Received TaxHarvestSimulated event:")
	log.Printf("  PlanID: %s", event.PlanID)
	log.Printf("  Losses Harvested: $%.2f", event.LossesHarvested)
	log.Printf("  Tax Savings Est: $%.2f", event.TaxSavingsEst)

	// TODO: Record tax opportunity
	return nil
}

// HandleRebalanceCompleted processes rebalance completed events
func (l *UMAEventListener) HandleRebalanceCompleted(msg kafka.Message) error {
	var event events.UMARebalanceCompletedEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		log.Printf("❌ Failed to unmarshal event: %v", err)
		return err
	}

	log.Printf("📥 Received RebalanceCompleted event:")
	log.Printf("  PlanID: %s", event.PlanID)
	log.Printf("  Completed Trades: %d", event.CompletedTradeCount)
	log.Printf("  Failed Trades: %d", event.FailedTradeCount)
	log.Printf("  Tax Impact: $%.2f", event.ActualTaxImpact)

	// TODO: Store rebalance completion
	return nil
}

// ============================================================================
// LISTENER SETUP
// ============================================================================

// ListenForEvents starts listening for UMA events
func (l *UMAEventListener) ListenForEvents(ctx context.Context) error {
	log.Printf("🚀 UMA Event Listener started (topic: uma.events)")

	go func() {
		defer l.reader.Close()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				m, err := l.reader.FetchMessage(ctx)
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					log.Printf("Error fetching Kafka message: %v", err)
					time.Sleep(1 * time.Second)
					continue
				}

				key := string(m.Key)
				log.Printf("📨 Message received on key: %s", key)

				var handlerErr error
				switch key {
				case "uma.rebalance.requested":
					handlerErr = l.HandleRebalanceRequested(m)
				case "uma.sleeve.drift.detected":
					handlerErr = l.HandleSleeveDriftDetected(m)
				case "uma.tax.harvest.simulated":
					handlerErr = l.HandleTaxHarvestSimulated(m)
				case "uma.rebalance.completed":
					handlerErr = l.HandleRebalanceCompleted(m)
				default:
					log.Printf("ℹ️  Unhandled message key: %s", key)
				}

				if handlerErr != nil {
					log.Printf("❌ Error processing message: %v", handlerErr)
					// Commit anyway to avoid block? Or retry?
					// For simple migration: commit.
					if err := l.reader.CommitMessages(ctx, m); err != nil {
						log.Printf("Failed to commit offset: %v", err)
					}
				} else {
					if err := l.reader.CommitMessages(ctx, m); err != nil {
						log.Printf("Failed to commit offset: %v", err)
					}
				}
			}
		}
	}()

	<-ctx.Done()
	return ctx.Err()
}

// Close closes the connection (reader)
func (l *UMAEventListener) Close() error {
	if l.reader != nil {
		return l.reader.Close()
	}
	return nil
}

// ============================================================================
// MAIN
// ============================================================================

func main() {
	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		brokers = "redpanda:9092"
	}

	listener, err := NewUMAEventListener(brokers)
	if err != nil {
		log.Fatalf("❌ Failed to create listener: %v", err)
	}
	defer listener.Close()

	ctx := context.Background()
	if err := listener.ListenForEvents(ctx); err != nil {
		log.Fatalf("❌ Listener error: %v", err)
	}
}
