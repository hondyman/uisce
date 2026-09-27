package datapipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
)

// AzureServiceBusBroker polls and publishes via Azure Service Bus queues.
type AzureServiceBusBroker struct{}

func (b AzureServiceBusBroker) connString(envName string) string {
	return envOr(envName, "AZURE_SERVICEBUS_CONNECTION_STRING")
}

func (b AzureServiceBusBroker) Poll(ctx context.Context, cfg QueueSourceConfig) ([]map[string]any, error) {
	cs := b.connString(cfg.ConnectionStringEnv)
	if cs == "" {
		return nil, fmt.Errorf("Azure Service Bus connection string not set (env AZURE_SERVICEBUS_CONNECTION_STRING)")
	}
	queue := strings.TrimSpace(cfg.TopicOrQueue)
	if queue == "" {
		return nil, fmt.Errorf("topic_or_queue (queue name) is required for azure_servicebus")
	}
	client, err := azservicebus.NewClientFromConnectionString(cs, nil)
	if err != nil {
		return nil, fmt.Errorf("azure service bus client: %w", err)
	}
	defer client.Close(ctx)

	receiver, err := client.NewReceiverForQueue(queue, nil)
	if err != nil {
		return nil, err
	}
	defer receiver.Close(ctx)

	max := cfg.MaxMessages
	if max <= 0 {
		max = 1000
	}
	idle := time.Duration(cfg.IdleTimeoutMS) * time.Millisecond
	if idle <= 0 {
		idle = 3 * time.Second
	}

	out := make([]map[string]any, 0, max)
	deadline := time.Now().Add(idle * 3)
	for len(out) < max && time.Now().Before(deadline) {
		cctx, cancel := context.WithTimeout(ctx, idle)
		n := max - len(out)
		if n > 50 {
			n = 50
		}
		msgs, err := receiver.ReceiveMessages(cctx, n, nil)
		cancel()
		if err != nil {
			if ctx.Err() != nil {
				return out, ctx.Err()
			}
			break
		}
		if len(msgs) == 0 {
			break
		}
		for _, m := range msgs {
			row, err := decodeQueueJSON(m.Body)
			if err != nil {
				row = map[string]any{"_raw": string(m.Body)}
			}
			out = append(out, row)
			_ = receiver.CompleteMessage(ctx, m, nil)
		}
	}
	return out, nil
}

func (b AzureServiceBusBroker) Publish(ctx context.Context, cfg QueueSinkConfig, rows []map[string]any) (int, error) {
	cs := b.connString(cfg.ConnectionStringEnv)
	if cs == "" {
		return 0, fmt.Errorf("Azure Service Bus connection string not set (env AZURE_SERVICEBUS_CONNECTION_STRING)")
	}
	queue := strings.TrimSpace(cfg.TopicOrQueue)
	if queue == "" {
		return 0, fmt.Errorf("topic_or_queue (queue name) is required for azure_servicebus")
	}
	client, err := azservicebus.NewClientFromConnectionString(cs, nil)
	if err != nil {
		return 0, fmt.Errorf("azure service bus client: %w", err)
	}
	defer client.Close(ctx)

	sender, err := client.NewSender(queue, nil)
	if err != nil {
		return 0, err
	}
	defer sender.Close(ctx)

	n := 0
	for _, row := range rows {
		payload, err := json.Marshal(row)
		if err != nil {
			return n, err
		}
		msg := &azservicebus.Message{Body: payload}
		if err := sender.SendMessage(ctx, msg, nil); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}
