package integrations

import (
	"context"
	"encoding/json"
	"net/http"

	"calendar-service/internal/hasura"

	"github.com/sirupsen/logrus"
)

// ZapierWebhook handles Zapier webhook integration
type ZapierWebhook struct {
	hasuraClient *hasura.Client
	logger       *logrus.Entry
}

// NewZapierWebhook creates a new Zapier webhook handler
func NewZapierWebhook(hc *hasura.Client, logger *logrus.Entry) *ZapierWebhook {
	return &ZapierWebhook{
		hasuraClient: hc,
		logger:       logger.WithField("component", "zapier_webhook"),
	}
}

// ZapierTrigger represents a Zapier trigger payload
type ZapierTrigger struct {
	Action     string                 `json:"action"`
	TenantID   string                 `json:"tenant_id"`
	UserID     string                 `json:"user_id"`
	CalendarID string                 `json:"calendar_id"`
	EventData  map[string]interface{} `json:"event_data"`
}

// HandleTrigger handles Zapier trigger webhook
func (z *ZapierWebhook) HandleTrigger(w http.ResponseWriter, r *http.Request) {
	var trigger ZapierTrigger
	if err := json.NewDecoder(r.Body).Decode(&trigger); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// Process trigger based on action
	switch trigger.Action {
	case "calendar.created":
		z.handleCalendarCreated(r.Context(), trigger)
	case "event.created":
		z.handleEventCreated(r.Context(), trigger)
	case "sync.completed":
		z.handleSyncCompleted(r.Context(), trigger)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "received",
	})
}

func (z *ZapierWebhook) handleCalendarCreated(ctx context.Context, t ZapierTrigger) {}
func (z *ZapierWebhook) handleEventCreated(ctx context.Context, t ZapierTrigger)    {}
func (z *ZapierWebhook) handleSyncCompleted(ctx context.Context, t ZapierTrigger)   {}

// SubscribeToEvents subscribes to calendar events via Zapier
func (z *ZapierWebhook) SubscribeToEvents(ctx context.Context, userID, webhookURL string) error {
	mutation := `
    mutation CreateWebhookSubscription($object: webhook_subscriptions_insert_input!) {
        insert_webhook_subscriptions_one(object: $object) {
            id
        }
    }
    `

	object := map[string]interface{}{
		"user_id":     userID,
		"webhook_url": webhookURL,
		"events":      []string{"sync.completed", "conflict.detected", "calendar.updated"},
		"is_active":   true,
	}

	return z.hasuraClient.Mutate(ctx, mutation, map[string]interface{}{
		"object": object,
	}, &struct{}{})
}
