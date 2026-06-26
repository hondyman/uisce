package events

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	kafka "github.com/segmentio/kafka-go"
	"go.temporal.io/sdk/client"
)

// KafkaConsumer consumes domain events from Kafka and routes them to Temporal
type KafkaConsumer struct {
	reader         *kafka.Reader
	temporalClient client.Client
	stopChan       chan struct{}
}

// KafkaConfig contains Kafka connection configuration
// Defined in kafka_publisher.go, re-used here implies same package.

// NewKafkaConsumer creates a new Kafka event consumer
func NewKafkaConsumer(
	config KafkaConfig,
	temporalClient client.Client,
) (*KafkaConsumer, error) {
	brokers := strings.Split(config.Brokers, ",")
	groupID := "api-catalog-sync-group"
	topics := []string{
		"api.endpoints",
		"api.mappings",
		"catalog.nodes",
		"catalog.edges",
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		GroupID:     groupID,
		GroupTopics: topics,
		MinBytes:    10e3,
		MaxBytes:    10e6,
	})

	consumer := &KafkaConsumer{
		reader:         r,
		temporalClient: temporalClient,
		stopChan:       make(chan struct{}),
	}

	return consumer, nil
}

// StartConsuming starts consuming events from Kafka
func (c *KafkaConsumer) StartConsuming(ctx context.Context) error {
	go c.processMessages(ctx)
	return nil
}

// processMessages processes messages from the reader
func (c *KafkaConsumer) processMessages(ctx context.Context) {
	for {
		select {
		case <-c.stopChan:
			return
		default:
			m, err := c.reader.FetchMessage(ctx)
			if err != nil {
				// Log error and continue (or backoff)
				time.Sleep(1 * time.Second)
				continue
			}

			// Route event to Temporal
			err = c.routeEventToTemporal(ctx, m)
			if err != nil {
				fmt.Printf("failed to route event to temporal: %v\n", err)
				// TODO: DLQ
			}

			if err := c.reader.CommitMessages(ctx, m); err != nil {
				fmt.Printf("failed to commit message: %v\n", err)
			}
		}
	}
}

// routeEventToTemporal routes an event to Temporal for processing
func (c *KafkaConsumer) routeEventToTemporal(ctx context.Context, msg kafka.Message) error {
	// Parse event based on type (stored in Key)
	eventType := EventType(string(msg.Key))
	if eventType == "" {
		// Fallback: try to infer? For now, strict.
		// If key is missing, we might want to check value structure.
		// But defaulting to error is safe.
		return fmt.Errorf("missing event type in message key")
	}

	var event DomainEvent
	var workflowID string
	workflowType := "CatalogSyncWorkflow"

	switch eventType {
	case APIEndpointCreated, APIEndpointUpdated, APIEndpointDeleted, APIEndpointActivated:
		evt := &APIEndpointEvent{}
		if err := json.Unmarshal(msg.Value, evt); err != nil {
			return fmt.Errorf("failed to unmarshal API endpoint event: %w", err)
		}
		event = evt
		workflowID = fmt.Sprintf("api-endpoint-%s-%s", evt.TenantID, evt.EndpointID)

	case EntityMappingCreated, EntityMappingDeleted:
		evt := &EntityMappingEvent{}
		if err := json.Unmarshal(msg.Value, evt); err != nil {
			return fmt.Errorf("failed to unmarshal entity mapping event: %w", err)
		}
		event = evt
		workflowID = fmt.Sprintf("entity-mapping-%s-%s", evt.TenantID, evt.APIEndpointID)

	case DatasourceMappingCreated, DatasourceMappingDeleted:
		evt := &DatasourceMappingEvent{}
		if err := json.Unmarshal(msg.Value, evt); err != nil {
			return fmt.Errorf("failed to unmarshal datasource mapping event: %w", err)
		}
		event = evt
		workflowID = fmt.Sprintf("datasource-mapping-%s-%s", evt.TenantID, evt.APIEndpointID)

	case CatalogNodeCreated, CatalogNodeUpdated, CatalogNodeDeleted:
		evt := &CatalogNodeEvent{}
		if err := json.Unmarshal(msg.Value, evt); err != nil {
			return fmt.Errorf("failed to unmarshal catalog node event: %w", err)
		}
		event = evt
		workflowID = fmt.Sprintf("catalog-node-%s-%s", evt.TenantID, evt.NodeID)

	case CatalogEdgeCreated, CatalogEdgeDeleted:
		evt := &CatalogEdgeEvent{}
		if err := json.Unmarshal(msg.Value, evt); err != nil {
			return fmt.Errorf("failed to unmarshal catalog edge event: %w", err)
		}
		event = evt
		workflowID = fmt.Sprintf("catalog-edge-%s-%s-%s", evt.TenantID, evt.SourceNodeID, evt.TargetNodeID)

	case GoldCopyConnectionChanged:
		evt := &GoldCopyConnectionEvent{}
		if err := json.Unmarshal(msg.Value, evt); err != nil {
			return fmt.Errorf("failed to unmarshal gold copy event: %w", err)
		}
		event = evt
		workflowID = fmt.Sprintf("gold-copy-conn-prop-%s", evt.EventID)
		workflowType = "GoldCopyConnectionPropagation"

	default:
		return fmt.Errorf("unknown event type: %s", eventType)
	}

	// Execute Temporal workflow
	options := client.StartWorkflowOptions{
		ID:                       workflowID,
		TaskQueue:                "metrics-compute", // Use active worker queue
		WorkflowExecutionTimeout: 5 * time.Minute,
	}

	_, err := c.temporalClient.ExecuteWorkflow(ctx, options, workflowType, event)
	if err != nil {
		return fmt.Errorf("failed to execute workflow: %w", err)
	}

	return nil
}

// Close closes the Kafka reader
func (c *KafkaConsumer) Close() error {
	close(c.stopChan)
	if c.reader != nil {
		return c.reader.Close()
	}
	return nil
}

// Healthcheck verifies the consumer is connected and operational
func (c *KafkaConsumer) Healthcheck() error {
	if c.reader == nil {
		return fmt.Errorf("Kafka reader is nil")
	}
	return nil
}
