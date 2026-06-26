package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type EventType string

const (
	EventWorkflowStarted   EventType = "WORKFLOW_STARTED"
	EventStepCompleted     EventType = "STEP_COMPLETED"
	EventWorkflowCompleted EventType = "WORKFLOW_COMPLETED"
	EventWorkflowFailed    EventType = "WORKFLOW_FAILED"
)

type RealtimeEvent struct {
	TenantID   string      `json:"tenantId"`
	PipelineID string      `json:"pipelineId"`
	RunID      string      `json:"runId"`
	Type       EventType   `json:"type"`
	Timestamp  time.Time   `json:"timestamp"`
	Payload    interface{} `json:"payload"`
}

type Publisher struct {
	client *redis.Client
}

func NewPublisher(addr string, password string) *Publisher {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password, // no password set
		DB:       0,        // use default DB
	})

	return &Publisher{
		client: rdb,
	}
}

// Publish emits an event to a channel specific to the tenant or pipeline
func (p *Publisher) Publish(ctx context.Context, event RealtimeEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Channel strategy: notifications:tenant:{tenantID}
	channel := fmt.Sprintf("notifications:tenant:%s", event.TenantID)

	return p.client.Publish(ctx, channel, data).Err()
}

func (p *Publisher) Close() error {
	return p.client.Close()
}
