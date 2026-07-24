package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	kafka "github.com/segmentio/kafka-go"
	"go.temporal.io/sdk/workflow"
)

// TestWorkflow executes a single activity that publishes an event to Kafka.
func TestWorkflow(ctx workflow.Context, brokers string, routingKey string, payload map[string]interface{}) error {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var result string
	err := workflow.ExecuteActivity(ctx, PublishEventActivity, brokers, routingKey, payload).Get(ctx, &result)
	if err != nil {
		return fmt.Errorf("activity failed: %w", err)
	}
	// return nil to indicate workflow completed successfully
	_ = result
	return nil
}

// PublishEventActivity publishes the supplied payload to the given topic/key on Kafka.
func PublishEventActivity(ctx context.Context, brokers string, routingKey string, payload map[string]interface{}) (string, error) {
	b := strings.Split(brokers, ",")
	w := &kafka.Writer{Addr: kafka.TCP(b...), Balancer: &kafka.LeastBytes{}}
	defer w.Close()

	body, _ := json.Marshal(payload)
	msg := kafka.Message{Topic: "events", Key: []byte(routingKey), Value: body, Time: time.Now()}
	if err := w.WriteMessages(ctx, msg); err != nil {
		return "", fmt.Errorf("publish: %w", err)
	}
	return "published", nil
}
