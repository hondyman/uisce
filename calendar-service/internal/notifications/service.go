package notifications

import (
	"context"

	"github.com/sirupsen/logrus"
)

// NotificationEvent represents an event that triggers a notification
type NotificationEvent struct {
	UserID         string
	TenantID       string
	Type           string // "SYNC_COMPLETE", "SYNC_ERROR", "CONFLICT_DETECTED"
	Title          string
	Message        string
	RecipientEmail string
	RecipientName  string
	Data           map[string]interface{}
}

// NotificationService handles sending notifications to users
type NotificationService interface {
	SendNotification(ctx context.Context, event NotificationEvent) error
}

// MockNotificationService is a basic implementation that logs notifications
type MockNotificationService struct {
	logger *logrus.Entry
	// In a real implementation this would hold an SMTP client, SendGrid client, etc.
}

// NewMockNotificationService creates a new mock notification service
func NewMockNotificationService(logger *logrus.Entry) *MockNotificationService {
	return &MockNotificationService{
		logger: logger.WithField("service", "mock_notification"),
	}
}

// SendNotification logs the notification
func (s *MockNotificationService) SendNotification(ctx context.Context, event NotificationEvent) error {
	s.logger.WithFields(logrus.Fields{
		"user_id":   event.UserID,
		"tenant_id": event.TenantID,
		"type":      event.Type,
		"title":     event.Title,
	}).Infof("SENDING NOTIFICATION: %s", event.Message)

	// Here you would check the user's settings (from DB) to see if they have
	// EmailNotifications or PushNotifications enabled for this specific event type,
	// and then route to the appropriate sender.

	return nil
}
