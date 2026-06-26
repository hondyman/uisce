package services

import (
	"context"
	"fmt"
	"text/template"
	"time"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/sirupsen/logrus"
)

// NotificationService handles all user notifications
type NotificationService struct {
	sendgridClient *sendgrid.Client
	fromEmail      string
	fromName       string
	logger         *logrus.Entry
	templates      map[NotificationType]*template.Template
}

// NotificationConfig holds configuration
type NotificationConfig struct {
	SendGridAPIKey string
	FromEmail      string
	FromName       string
	Logger         *logrus.Entry
}

// NewNotificationService creates a new notification service
func NewNotificationService(cfg NotificationConfig) (*NotificationService, error) {
	client := sendgrid.NewSendClient(cfg.SendGridAPIKey)

	// Initialize and parse templates
	parsedTemplates := make(map[NotificationType]*template.Template)
	for notifType, tmplString := range emailTemplates {
		tmpl, err := template.New(string(notifType)).Parse(tmplString)
		if err != nil {
			return nil, fmt.Errorf("failed to parse template %s: %w", notifType, err)
		}
		parsedTemplates[notifType] = tmpl
	}

	return &NotificationService{
		sendgridClient: client,
		fromEmail:      cfg.FromEmail,
		fromName:       cfg.FromName,
		logger:         cfg.Logger.WithField("component", "notifications"),
		templates:      parsedTemplates,
	}, nil
}

var emailTemplates = map[NotificationType]string{
	NotificationSyncComplete: `
		<!DOCTYPE html>
		<html>
		<body style="font-family: Arial, sans-serif; padding: 20px;">
			<h2>Sync Complete</h2>
			<p>Hi {{.ToName}},</p>
			<p>Successfully synced {{.events_synced}} events.</p>
		</body>
		</html>
	`,
	NotificationSyncFailed: `
		<!DOCTYPE html>
		<html>
		<body style="font-family: Arial, sans-serif; padding: 20px;">
			<h2>Sync Failed</h2>
			<p>Hi {{.ToName}},</p>
			<p>Error: {{.error}}</p>
		</body>
		</html>
	`,
	NotificationWeeklyDigest: `
		<!DOCTYPE html>
		<html>
		<body style="font-family: Arial, sans-serif; padding: 20px;">
			<h2>Weekly Report</h2>
			<p>Hi {{.ToName}},</p>
			<p>Total Syncs: {{.total_syncs}}</p>
		</body>
		</html>
	`,
}

// NotificationType represents the type of notification
type NotificationType string

const (
	NotificationSyncComplete        NotificationType = "sync_complete"
	NotificationSyncFailed          NotificationType = "sync_failed"
	NotificationConflictDetected    NotificationType = "conflict_detected"
	NotificationTokenExpiring       NotificationType = "token_expiring"
	NotificationAccountDisconnected NotificationType = "account_disconnected"
	NotificationWeeklyDigest        NotificationType = "weekly_digest"
)

// DigestData represents metrics for the weekly summary
type DigestData struct {
	TotalSyncs        int
	SuccessRate       float64
	EventsSynced      int
	ConflictsResolved int
	ActiveConnections int
}

// Notification represents a notification to send
type Notification struct {
	Type     NotificationType
	ToEmail  string
	ToName   string
	Subject  string
	Data     map[string]interface{}
	Priority string // "high", "normal", "low"
}

// Send sends a notification to the user
func (s *NotificationService) Send(ctx context.Context, notif Notification) error {
	// Get email template based on type
	templateID := s.getTemplateID(notif.Type)

	// Build email content
	content := s.buildEmailContent(notif)

	// Create email
	from := mail.NewEmail(s.fromName, s.fromEmail)
	to := mail.NewEmail(notif.ToName, notif.ToEmail)

	message := mail.NewV3Mail()
	message.SetFrom(from)
	personalization := mail.NewPersonalization()
	personalization.AddTos(to)
	personalization.Subject = notif.Subject
	message.AddPersonalizations(personalization)

	message.TemplateID = templateID
	message.AddContent(mail.NewContent("text/html", content))

	// Send email
	response, err := s.sendgridClient.Send(message)
	if err != nil {
		s.logger.WithError(err).WithField("email", notif.ToEmail).Error("Failed to send notification")
		return fmt.Errorf("send email: %w", err)
	}

	if response.StatusCode >= 400 {
		s.logger.WithField("status", response.StatusCode).Error("SendGrid returned error")
		return fmt.Errorf("sendgrid error: %d", response.StatusCode)
	}

	s.logger.WithFields(logrus.Fields{
		"email": notif.ToEmail,
		"type":  notif.Type,
	}).Debug("Notification sent successfully")

	return nil
}

// SendSyncComplete sends sync completion notification
func (s *NotificationService) SendSyncComplete(ctx context.Context, userEmail, userName string, eventsSynced int) error {
	return s.Send(ctx, Notification{
		Type:     NotificationSyncComplete,
		ToEmail:  userEmail,
		ToName:   userName,
		Subject:  fmt.Sprintf("✅ Sync Complete - %d events synced", eventsSynced),
		Priority: "low",
		Data: map[string]interface{}{
			"events_synced": eventsSynced,
			"timestamp":     time.Now().UTC(),
		},
	})
}

// SendSyncFailed sends sync failure notification
func (s *NotificationService) SendSyncFailed(ctx context.Context, userEmail, userName string, errorMsg string) error {
	return s.Send(ctx, Notification{
		Type:     NotificationSyncFailed,
		ToEmail:  userEmail,
		ToName:   userName,
		Subject:  "⚠️ Sync Failed - Action Required",
		Priority: "high",
		Data: map[string]interface{}{
			"error":     errorMsg,
			"timestamp": time.Now().UTC(),
			"retry_url": "/settings/sync",
		},
	})
}

// SendWeeklyDigest sends a summary of sync activity
func (s *NotificationService) SendWeeklyDigest(ctx context.Context, userEmail, userName string, data DigestData) error {
	return s.Send(ctx, Notification{
		Type:     NotificationWeeklyDigest,
		ToEmail:  userEmail,
		ToName:   userName,
		Subject:  "📊 Your Weekly Calendar Sync Report",
		Priority: "low",
		Data: map[string]interface{}{
			"total_syncs":        data.TotalSyncs,
			"success_rate":       fmt.Sprintf("%.1f%%", data.SuccessRate),
			"events_synced":      data.EventsSynced,
			"conflicts_resolved": data.ConflictsResolved,
			"active_conns":       data.ActiveConnections,
			"timestamp":          time.Now().Format("Jan 02, 2006"),
			"unsubscribe_url":    s.generateUnsubscribeURL(userEmail),
		},
	})
}

func (s *NotificationService) generateUnsubscribeURL(email string) string {
	return fmt.Sprintf("https://calendar.yourcompany.com/api/v1/notifications/unsubscribe?email=%s", email)
}
func (s *NotificationService) getTemplateID(notifType NotificationType) string {
	// In production these would be IDs from SendGrid dashboard
	return "d-placeholder-id"
}

func (s *NotificationService) buildEmailContent(notif Notification) string {
	return fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<body style="font-family: Arial, sans-serif; padding: 20px;">
			<h2>%s</h2>
			<p>Hi %s,</p>
			<p>%s</p>
			<p>Best regards,<br>Calendar Sync Team</p>
		</body>
		</html>
	`, notif.Subject, notif.ToName, notif.Subject)
}
