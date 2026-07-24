package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pusher/pusher-http-go/v5"
)

// PusherService handles real-time notifications via Pusher
type PusherService struct {
	appID   string
	key     string
	secret  string
	cluster string
	client  *pusher.Client
}

// NewPusherService creates a new Pusher service
func NewPusherService(appID, key, secret, cluster string) *PusherService {
	client := &pusher.Client{
		AppID:   appID,
		Key:     key,
		Secret:  secret,
		Cluster: cluster,
		Secure:  true,
	}

	return &PusherService{
		appID:   appID,
		key:     key,
		secret:  secret,
		cluster: cluster,
		client:  client,
	}
}

// PusherEvent represents a real-time event
type PusherEvent struct {
	Channel string                 `json:"channel"`
	Event   string                 `json:"event"`
	Data    map[string]interface{} `json:"data"`
}

// TriggerEvent sends a real-time event to a channel
func (s *PusherService) TriggerEvent(ctx context.Context, channel, event string, data map[string]interface{}) error {
	err := s.client.Trigger(channel, event, data)
	if err != nil {
		return fmt.Errorf("failed to trigger Pusher event: %w", err)
	}
	return nil
}

// NotifyTradeExecution notifies user of trade execution
func (s *PusherService) NotifyTradeExecution(ctx context.Context, userID string, trade map[string]interface{}) error {
	channel := fmt.Sprintf("private-user-%s", userID)
	return s.TriggerEvent(ctx, channel, "trade-executed", trade)
}

// NotifyPriceAlert notifies user of price alert
func (s *PusherService) NotifyPriceAlert(ctx context.Context, userID string, alert map[string]interface{}) error {
	channel := fmt.Sprintf("private-user-%s", userID)
	return s.TriggerEvent(ctx, channel, "price-alert", alert)
}

// NotifyPortfolioUpdate notifies user of portfolio update
func (s *PusherService) NotifyPortfolioUpdate(ctx context.Context, userID string, update map[string]interface{}) error {
	channel := fmt.Sprintf("private-user-%s", userID)
	return s.TriggerEvent(ctx, channel, "portfolio-update", update)
}

// TriggerBatch sends multiple events in a batch
func (s *PusherService) TriggerBatch(ctx context.Context, events []PusherEvent) error {
	batch := make([]pusher.Event, 0, len(events))

	for _, evt := range events {
		data, _ := json.Marshal(evt.Data)
		batch = append(batch, pusher.Event{
			Channel: evt.Channel,
			Name:    evt.Event,
			Data:    string(data),
		})
	}

	_, err := s.client.TriggerBatch(batch)
	if err != nil {
		return fmt.Errorf("failed to trigger batch events: %w", err)
	}

	return nil
}

// AuthenticatePrivateChannel authenticates a user for a private channel
func (s *PusherService) AuthenticatePrivateChannel(ctx context.Context, socketID, channelName string) ([]byte, error) {
	auth, err := s.client.AuthenticatePrivateChannel([]byte(fmt.Sprintf("socket_id=%s&channel_name=%s", socketID, channelName)))
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate channel: %w", err)
	}
	return auth, nil
}

// NotifyWorkflowUpdate sends workflow status update
func (s *PusherService) NotifyWorkflowUpdate(ctx context.Context, userID, workflowID, status string, progress float64) error {
	channel := fmt.Sprintf("private-user-%s", userID)
	data := map[string]interface{}{
		"workflow_id": workflowID,
		"status":      status,
		"progress":    progress,
		"timestamp":   ctx.Value("timestamp"),
	}
	return s.TriggerEvent(ctx, channel, "workflow-update", data)
}
