package api

import (
	"context"

	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/hondyman/uisce/backend/internal/services"
)

// noopEventBus is a minimal EventBus implementation used in local/dev runs.
type noopEventBus struct{}

func (n *noopEventBus) Emit(ctx context.Context, event string, data interface{}) error {
	return nil
}

// notificationAdapter adapts the EngagementNotificationService to the
// NotificationService interface expected by the TriggerEngine.
type notificationAdapter struct {
	svc *services.EngagementNotificationService
}

func (a *notificationAdapter) Send(ctx context.Context, channel string, payload *NotificationPayload) error {
	userID := ""
	if len(payload.Recipients) > 0 {
		userID = payload.Recipients[0]
	}

	notification := &models.EngagementNotification{
		UserID:    userID,
		Title:     payload.Subject,
		Message:   payload.Body,
		Channels:  []string{channel},
		CreatedBy: "system",
	}

	if err := a.svc.CreateNotification(ctx, notification); err != nil {
		return err
	}
	return a.svc.SendNotification(ctx, notification.ID)
}
