package workers

import (
	"context"
	"encoding/json"
	"log"
	"time"

	kafka "github.com/segmentio/kafka-go"

	"github.com/hondyman/semlayer/backend/services/compliance-engine/internal/engine"
	"github.com/hondyman/semlayer/backend/services/compliance-engine/internal/models"
	"github.com/hondyman/semlayer/backend/services/compliance-engine/internal/queue"
	"github.com/hondyman/semlayer/backend/services/compliance-engine/internal/service"
)

// PostTradeWorker processes post-trade compliance checks asynchronously
type PostTradeWorker struct {
	kafkaClient *queue.KafkaClient
	service     *service.ComplianceService
	engine      *engine.ValidationEngine
}

// NewPostTradeWorker creates a new post-trade worker
func NewPostTradeWorker(
	kafkaClient *queue.KafkaClient,
	service *service.ComplianceService,
	engine *engine.ValidationEngine,
) *PostTradeWorker {
	return &PostTradeWorker{
		kafkaClient: kafkaClient,
		service:     service,
		engine:      engine,
	}
}

// Start begins consuming messages from the post-trade queue (topic)
func (w *PostTradeWorker) Start(ctx context.Context) error {
	log.Println("👷 Post-Trade Worker Starting...")

	reader, err := w.kafkaClient.Consume("q.compliance.post_trade", "post-trade-worker")
	if err != nil {
		return err
	}

	go func() {
		defer reader.Close()
		for {
			select {
			case <-ctx.Done():
				log.Println("Post-Trade Worker shutting down...")
				return
			default:
				msg, err := reader.FetchMessage(ctx)
				if err != nil {
					if ctx.Err() != nil {
						// Context cancelled
						return
					}
					log.Printf("Error fetching message: %v", err)
					time.Sleep(1 * time.Second)
					continue
				}
				w.processMessage(ctx, reader, msg)
			}
		}
	}()

	log.Println("Post-Trade Worker started successfully")
	return nil
}

func (w *PostTradeWorker) processMessage(ctx context.Context, reader *kafka.Reader, msg kafka.Message) {
	var trade models.TradeRequest
	if err := json.Unmarshal(msg.Value, &trade); err != nil {
		log.Printf("⚠️  Failed to unmarshal trade: %v", err)
		// Commit anyway to avoid getting stuck on bad message
		if err := reader.CommitMessages(ctx, msg); err != nil {
			log.Printf("Failed to commit bad message: %v", err)
		}
		return
	}

	log.Printf("Processing post-trade validation for trade %s", trade.ID)

	// Run post-trade validation (heavy checks)
	result, err := w.service.PostTradeValidate(ctx, trade)
	if err != nil {
		log.Printf("⚠️  Post-trade validation failed for %s: %v", trade.ID, err)

		// Publish failure event for audit
		w.publishAuditEvent(ctx, trade, "POST_TRADE", "FAIL", []string{err.Error()})

		// Ack (Commit) even on failure to proceed? Or DLQ?
		// For now, commit to prevent blocking.
		if err := reader.CommitMessages(ctx, msg); err != nil {
			log.Printf("Failed to commit message after processing error: %v", err)
		}
		return
	}

	if result.Status == "REJECTED" {
		log.Printf("⚠️  Post-trade flagged: %s - %v", trade.ID, result.Errors)
		// Trigger alerts, freeze account, etc.
		w.publishAuditEvent(ctx, trade, "POST_TRADE", "FAIL", result.Errors)
	} else {
		log.Printf("✅ Post-trade verified %s", trade.ID)
		w.publishAuditEvent(ctx, trade, "POST_TRADE", "PASS", nil)
	}

	if err := reader.CommitMessages(ctx, msg); err != nil {
		log.Printf("Failed to commit message: %v", err)
	}
}

func (w *PostTradeWorker) publishAuditEvent(ctx context.Context, trade models.TradeRequest, eventType string, status string, errors []string) {
	event := models.ComplianceEvent{
		TraceID:     trade.ID,
		EventType:   eventType,
		Status:      status,
		RuleVersion: "2025", // TODO: get from context
		TradeData: map[string]interface{}{
			"id":        trade.ID,
			"amount":    trade.Amount,
			"currency":  trade.Currency,
			"orderType": trade.OrderType,
			"tradeDate": trade.TradeDate,
		},
		CreatedAt: time.Now(),
	}

	if len(errors) > 0 {
		event.ErrorDetails = map[string]interface{}{
			"errors": errors,
		}
	}

	routingKey := "audit.post_trade." + status
	if err := w.kafkaClient.PublishEvent(ctx, routingKey, event); err != nil {
		log.Printf("Failed to publish audit event: %v", err)
	}
}
