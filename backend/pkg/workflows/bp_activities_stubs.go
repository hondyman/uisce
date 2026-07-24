package workflows

import (
	"context"
	"log"
	"time"

	"go.temporal.io/sdk/activity"
)

// ============================================================================
// Activity Stubs for Integration
// ============================================================================

// ActivityCreateHumanTask creates a human task in the external system
// In a real implementation, this would call a Task Service or create a DB record
func ActivityCreateHumanTask(ctx context.Context, input map[string]interface{}) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Creating human task (stub)", "taskID", input["task_id"], "title", input["title"])

	// Simulate external system call
	time.Sleep(100 * time.Millisecond)

	// In a real system, we would:
	// 1. Create a record in the tasks table
	// 2. Send notifications to assignees
	// 3. Return the task ID

	return nil
}

// Stub activities for publish event broker types
// DEPRECATED: legacy RabbitMQ (AMQP) publish stub. Prefer Kafka/Redpanda publishing activity.
func stubActivityPublishRabbitMQ(ctx context.Context, input interface{}) error {
	log.Printf("DEPRECATED STUB: Publish RabbitMQ (legacy): %v", input)
	return nil
}

func stubActivityPublishKafka(ctx context.Context, input interface{}) error {
	log.Printf("STUB: Publish Kafka: %v", input)
	return nil
}

func stubActivitySendAlert(ctx context.Context, input interface{}) error {
	log.Printf("STUB: Send Alert: %v", input)
	return nil
}

func stubActivityExecuteSteps(ctx context.Context, input interface{}) error {
	log.Printf("STUB: Execute Steps: %v", input)
	return nil
}
