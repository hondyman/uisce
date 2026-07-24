package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"strings"

	"github.com/google/uuid"
	kafka "github.com/segmentio/kafka-go"
)

// ============================================================================
// COMMAND BUS - Microservices Command Bus Pattern
// ============================================================================
// This service implements the command bus pattern where all CRUD operations
// for Business Objects are performed through RabbitMQ message queues instead
// of direct HTTP endpoints. This enables:
//
// 1. Loose coupling between API gateway and microservices
// 2. Async request/reply pattern for long-running operations
// 3. Easy scale-out of command handlers
// 4. Audit trail of all commands
// 5. Replay and testing capabilities
//
// Architecture:
// API Gateway → publishes Command → semlayer.commands exchange → BO Service
// BO Service → executes command + publishes Event → semlayer.events exchange
// BO Service → publishes CommandResponse → reply queue
// API Gateway ← receives CommandResponse ← reply queue
//
// ============================================================================

// CommandPublisher publishes commands to the command bus (Kafka/Redpanda)
type CommandPublisher struct {
	writer       *kafka.Writer
	commandTopic string
	replyTopic   string
	enabled      bool
}

// IsEnabled returns true if the command bus is enabled
func (cp *CommandPublisher) IsEnabled() bool {
	return cp.enabled
}

// NewCommandPublisher creates a new command publisher (Kafka/Redpanda).
// Accepts either a Kafka brokers list or legacy AMQP URL (deprecated).
func NewCommandPublisher(brokersOrURL string) (*CommandPublisher, error) {
	if brokersOrURL == "" {
		log.Println("⚠️  Command bus not configured - disabled")
		return &CommandPublisher{enabled: false}, nil
	}

	// Detect legacy AMQP URL and disable (encourage migration)
	if strings.HasPrefix(brokersOrURL, "amqp://") {
		log.Printf("⚠️  Detected legacy AMQP URL %s - command bus disabled. Set KAFKA_BROKERS instead.", brokersOrURL)
		return &CommandPublisher{enabled: false}, nil
	}

	brokers := strings.Split(brokersOrURL, ",")
	w := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Balancer: &kafka.LeastBytes{},
	}

	log.Println("✅ Kafka-based command bus initialized")

	return &CommandPublisher{
		writer:       w,
		commandTopic: "semlayer.commands",
		replyTopic:   "semlayer.replies",
		enabled:      true,
	}, nil
}

// PublishCommand publishes a command to the command bus (Kafka). Returns the correlation ID for tracking.
func (cp *CommandPublisher) PublishCommand(ctx context.Context, commandType CommandType, tenantID, userID string, data interface{}) (string, error) {
	if !cp.enabled {
		return "", fmt.Errorf("command bus not enabled")
	}

	correlationID := uuid.New().String()
	command := &Command{
		ID:            uuid.New().String(),
		Type:          commandType,
		TenantID:      tenantID,
		UserID:        userID,
		Data:          data,
		Timestamp:     time.Now(),
		CorrelationID: correlationID,
	}

	body, err := json.Marshal(command)
	if err != nil {
		return "", fmt.Errorf("failed to marshal command: %w", err)
	}

	routingKey := string(commandType)
	msg := kafka.Message{
		Topic: cp.commandTopic,
		Key:   []byte(routingKey),
		Value: body,
		Time:  time.Now(),
	}

	if err := cp.writer.WriteMessages(ctx, msg); err != nil {
		return "", fmt.Errorf("failed to publish command: %w", err)
	}

	log.Printf("📤 Command published: %s (correlation: %s)", commandType, correlationID)

	return correlationID, nil
}

// Close closes the publisher resources
func (cp *CommandPublisher) Close() error {
	if !cp.enabled {
		return nil
	}
	if cp.writer != nil {
		return cp.writer.Close()
	}
	return nil
}

// ============================================================================
// COMMAND RESPONSE
// ============================================================================

// CommandStatus indicates the result of command execution
type CommandStatus string

const (
	CommandStatusSuccess CommandStatus = "success"
	CommandStatusFailed  CommandStatus = "failed"
	CommandStatusPending CommandStatus = "pending"
)

// CommandResponse is the response to a command
type CommandResponse struct {
	ID            string        `json:"id"`
	CorrelationID string        `json:"correlation_id"`
	Status        CommandStatus `json:"status"`
	Message       string        `json:"message"`
	Data          interface{}   `json:"data,omitempty"`
	Error         string        `json:"error,omitempty"`
	Timestamp     time.Time     `json:"timestamp"`
}

// ============================================================================
// COMMAND CONSUMER (Microservice Side)
// ============================================================================

// CommandHandler is a function that handles a command
type CommandHandler func(ctx context.Context, command *Command) (*CommandResponse, error)

// CommandConsumer consumes commands from the command bus (Kafka)
type CommandConsumer struct {
	reader      *kafka.Reader
	replyWriter *kafka.Writer
	serviceName string
	handlers    map[CommandType]CommandHandler
	enabled     bool
}

// NewCommandConsumer creates a new command consumer
func NewCommandConsumer(brokersOrURL, serviceName string) (*CommandConsumer, error) {
	if brokersOrURL == "" {
		return &CommandConsumer{enabled: false}, nil
	}

	if strings.HasPrefix(brokersOrURL, "amqp://") {
		log.Printf("⚠️  Detected legacy AMQP URL %s - command consumer disabled. Set KAFKA_BROKERS instead.", brokersOrURL)
		return &CommandConsumer{enabled: false}, nil
	}

	brokers := strings.Split(brokersOrURL, ",")
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  serviceName,
		Topic:    "semlayer.commands",
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})

	rw := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Balancer: &kafka.LeastBytes{},
	}

	return &CommandConsumer{
		reader:      r,
		replyWriter: rw,
		serviceName: serviceName,
		handlers:    make(map[CommandType]CommandHandler),
		enabled:     true,
	}, nil
}

// RegisterHandler registers a handler for a command type
func (cc *CommandConsumer) RegisterHandler(commandType CommandType, handler CommandHandler) {
	cc.handlers[commandType] = handler
	log.Printf("✅ Handler registered for command: %s", commandType)
}

func (cc *CommandConsumer) Subscribe(ctx context.Context, pattern string) error {
	if !cc.enabled {
		return fmt.Errorf("command consumer not enabled")
	}

	log.Printf("📥 Command consumer starting (pattern=%s)", pattern)

	// Start consuming in a goroutine
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				m, err := cc.reader.FetchMessage(ctx)
				if err != nil {
					continue
				}

				// Process message
				var command Command
				if err := json.Unmarshal(m.Value, &command); err != nil {
					log.Printf("❌ Failed to unmarshal command: %v", err)
					cc.reader.CommitMessages(ctx, m)
					continue
				}

				handler, ok := cc.handlers[command.Type]
				if !ok {
					log.Printf("❌ No handler for command type: %s", command.Type)
					cc.reader.CommitMessages(ctx, m)
					continue
				}

				log.Printf("⚙️  Executing command: %s (correlation: %s)", command.Type, command.CorrelationID)
				response, err := handler(ctx, &command)
				if err != nil {
					log.Printf("❌ Command failed: %v", err)
					response = &CommandResponse{
						ID:            uuid.New().String(),
						CorrelationID: command.CorrelationID,
						Status:        CommandStatusFailed,
						Error:         err.Error(),
						Timestamp:     time.Now(),
					}
				}

				// Send response to reply topic with correlation id as key
				respBody, _ := json.Marshal(response)
				if cc.replyWriter != nil {
					msg := kafka.Message{Topic: "semlayer.replies", Key: []byte(response.CorrelationID), Value: respBody, Time: time.Now()}
					if err := cc.replyWriter.WriteMessages(ctx, msg); err != nil {
						log.Printf("❌ Failed to publish command response: %v", err)
					}
				}

				// Commit the consumed message
				cc.reader.CommitMessages(ctx, m)
			}
		}
	}()

	return nil
}

// Close closes the consumer connection
func (cc *CommandConsumer) Close() error {
	if !cc.enabled {
		return nil
	}
	if cc.reader != nil {
		cc.reader.Close()
	}
	if cc.replyWriter != nil {
		cc.replyWriter.Close()
	}
	return nil
}
