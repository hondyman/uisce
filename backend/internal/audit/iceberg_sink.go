package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

// IcebergSinkConsumer consumes audit events from Redpanda and writes to Iceberg storage
type IcebergSinkConsumer struct {
	reader        *kafka.Reader
	topics        []string
	icebergWriter *IcebergWriter
	eventBuffer   []KafkaEventEnvelope
	batchSize     int
	flushInterval time.Duration
	mu            sync.RWMutex
	stopChan      chan struct{}
	running       bool
}

// IcebergWriter handles writing Parquet files to Iceberg storage
type IcebergWriter struct {
	S3Client   interface{} // S3-compatible client (e.g., *minio.Client)
	BucketName string
	CatalogURI string
}

// NewIcebergSinkConsumer creates a new Iceberg sink consumer that reads from Redpanda
// bootstrapServers format: "host1:9092,host2:9092"
// groupID: consumer group for offset management
// topics: list of topics to consume from
// icebergWriter: configured writer for Iceberg storage
func NewIcebergSinkConsumer(bootstrapServers, groupID string, topics []string, icebergWriter *IcebergWriter) (*IcebergSinkConsumer, error) {
	if bootstrapServers == "" {
		return nil, fmt.Errorf("bootstrap servers cannot be empty")
	}
	if groupID == "" {
		return nil, fmt.Errorf("group ID cannot be empty")
	}
	if len(topics) == 0 {
		return nil, fmt.Errorf("at least one topic must be specified")
	}
	if icebergWriter == nil {
		return nil, fmt.Errorf("iceberg writer cannot be nil")
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{bootstrapServers},
		GroupID:        groupID,
		GroupTopics:    topics,
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
		CommitInterval: 1 * time.Second,
		StartOffset:    kafka.FirstOffset,
	})

	return &IcebergSinkConsumer{
		reader:        reader,
		topics:        topics,
		icebergWriter: icebergWriter,
		eventBuffer:   make([]KafkaEventEnvelope, 0, 1000),
		batchSize:     1000,
		flushInterval: 30 * time.Second,
		stopChan:      make(chan struct{}),
		running:       false,
	}, nil
}

// Start begins consuming messages and writing to Iceberg
func (c *IcebergSinkConsumer) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return fmt.Errorf("consumer already running")
	}
	c.running = true
	c.mu.Unlock()

	// Create ticker for periodic flushes
	flushTicker := time.NewTicker(c.flushInterval)
	defer flushTicker.Stop()

	// Consumer loop - process messages from Redpanda
	for {
		select {
		case <-ctx.Done():
			// Flush remaining events before shutting down
			_ = c.flush(ctx)
			c.mu.Lock()
			c.running = false
			c.mu.Unlock()
			return ctx.Err()

		case <-c.stopChan:
			// Flush remaining events before shutting down
			_ = c.flush(ctx)
			c.mu.Lock()
			c.running = false
			c.mu.Unlock()
			return nil

		case <-flushTicker.C:
			// Periodically flush buffered events
			if err := c.flush(ctx); err != nil {
				fmt.Printf("Error flushing events: %v\n", err)
				// Continue processing despite flush errors
			}

		default:
			// Fetch a message
			msg, err := c.reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				fmt.Printf("Consumer error: %v\n", err)
				continue
			}

			// Parse the message
			var envelope KafkaEventEnvelope
			if err := json.Unmarshal(msg.Value, &envelope); err != nil {
				fmt.Printf("Failed to unmarshal message: %v\n", err)
				// Commit message even on parse error to avoid reprocessing
				if commitErr := c.reader.CommitMessages(ctx, msg); commitErr != nil {
					fmt.Printf("Failed to commit bad message: %v\n", commitErr)
				}
				continue
			}

			// Add to buffer
			c.mu.Lock()
			c.eventBuffer = append(c.eventBuffer, envelope)
			shouldFlush := len(c.eventBuffer) >= c.batchSize
			c.mu.Unlock()

			// Auto-flush if batch size reached
			if shouldFlush {
				if err := c.flush(ctx); err != nil {
					fmt.Printf("Error flushing batch: %v\n", err)
				}
			}

			// Commit offset after successful processing
			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				fmt.Printf("Failed to commit message: %v\n", err)
			}
		}
	}
}

// flush writes buffered events to Iceberg
func (c *IcebergSinkConsumer) flush(ctx context.Context) error {
	c.mu.Lock()
	if len(c.eventBuffer) == 0 {
		c.mu.Unlock()
		return nil
	}

	// Make a copy of events to process
	eventsToWrite := make([]KafkaEventEnvelope, len(c.eventBuffer))
	copy(eventsToWrite, c.eventBuffer)

	// Clear buffer
	c.eventBuffer = c.eventBuffer[:0]
	c.mu.Unlock()

	// Write events grouped by type
	eventsByTopic := make(map[string][]KafkaEventEnvelope)
	for _, event := range eventsToWrite {
		topic := c.getTopicForEvent(event.EventType)
		eventsByTopic[topic] = append(eventsByTopic[topic], event)
	}

	// Write each topic's events
	for topic, events := range eventsByTopic {
		if err := c.writeToIceberg(ctx, topic, events); err != nil {
			fmt.Printf("Error writing events for topic %s: %v\n", topic, err)
			// In production, would implement retry logic or dead-letter queue
		}
	}

	return nil
}

// writeToIceberg writes events to an Iceberg table via Parquet format
func (c *IcebergSinkConsumer) writeToIceberg(ctx context.Context, topic string, events []KafkaEventEnvelope) error {
	if len(events) == 0 {
		return nil
	}

	// Convert events to Parquet-compatible format
	rows := make([]interface{}, len(events))
	for i, event := range events {
		// Create a struct for Parquet serialization
		row := map[string]interface{}{
			"event_id":   event.EventID,
			"event_type": event.EventType,
			"version":    event.Version,
			"timestamp":  event.Timestamp.Unix(),
			"tenant_id":  event.TenantID,
			"source":     event.Source,
			"payload":    string(event.Payload),
		}
		rows[i] = row
	}

	// Serialize to JSON for demonstration
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)

	for _, row := range rows {
		if err := encoder.Encode(row); err != nil {
			return fmt.Errorf("failed to encode event for Parquet: %w", err)
		}
	}

	// In a full implementation, this would:
	// 1. Use parquet-go library to convert rows to Parquet format
	// 2. Write to S3 via minio client
	// 3. Register with Iceberg catalog via REST API

	fmt.Printf("Would write %d events to Iceberg table for topic %s (%d bytes)\n", len(events), topic, buffer.Len())

	return nil
}

// getTopicForEvent returns the target Iceberg table based on event type
func (c *IcebergSinkConsumer) getTopicForEvent(eventType string) string {
	switch eventType {
	case EventTypeJobRunCompleted:
		return TopicSchedulerJobRuns
	case EventTypeDAGRunCompleted:
		return TopicSchedulerDAGRuns
	case EventTypeChangeSetCreated, EventTypeChangeSetApproved, EventTypeChangeSetApplied:
		return TopicGovernanceChangeSets
	case EventTypeSemanticSnapshot:
		return TopicSemanticSnapshots
	case EventTypeWorkflowStarted, EventTypeWorkflowCompleted, EventTypeWorkflowFailed:
		return TopicOrchestrationEvents
	case EventTypeComplianceViolation:
		return TopicComplianceViolations
	case EventTypeAINarrativeGenerated:
		return TopicAISuggestions
	default:
		return "unknown"
	}
}

// GetBufferStats returns statistics about the current buffer
func (c *IcebergSinkConsumer) GetBufferStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"buffer_size": len(c.eventBuffer),
		"max_size":    c.batchSize,
		"running":     c.running,
	}
}

// Stop stops the consumer gracefully
func (c *IcebergSinkConsumer) Stop(ctx context.Context) error {
	select {
	case c.stopChan <- struct{}{}:
		// Wait for consumer to stop or context to timeout
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			c.mu.RLock()
			if !c.running {
				c.mu.RUnlock()
				return nil
			}
			c.mu.RUnlock()

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
			}
		}

	case <-ctx.Done():
		return ctx.Err()
	}
}

// Close closes the consumer and releases resources
func (c *IcebergSinkConsumer) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := c.Stop(ctx); err != nil && err != context.DeadlineExceeded {
		return err
	}

	if c.reader != nil {
		c.reader.Close()
	}

	return nil
}
