package notifications

import (
	"context"
	"fmt"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/sirupsen/logrus"
)

// SendGridSender implements NotificationService using SendGrid
type SendGridSender struct {
	client    *sendgrid.Client
	fromEmail string
	fromName  string
	logger    *logrus.Entry
}

// SendGridConfig holds SendGrid configuration
type SendGridConfig struct {
	APIKey    string
	FromEmail string
	FromName  string
}

// NewSendGridSender creates a new SendGrid email sender
func NewSendGridSender(cfg SendGridConfig, logger *logrus.Entry) *SendGridSender {
	client := sendgrid.NewSendClient(cfg.APIKey)
	return &SendGridSender{
		client:    client,
		fromEmail: cfg.FromEmail,
		fromName:  cfg.FromName,
		logger:    logger.WithField("service", "sendgrid_sender"),
	}
}

// SendNotification sends an email based on the event
func (s *SendGridSender) SendNotification(ctx context.Context, event NotificationEvent) error {
	toEmail, ok := event.Data["email"].(string)
	if !ok || toEmail == "" {
		s.logger.Warnf("No email provided for user %s, skipping email notification", event.UserID)
		return nil
	}

	from := mail.NewEmail(s.fromName, s.fromEmail)
	to := mail.NewEmail(event.UserID, toEmail) // Using UserID as name if real name not available

	// Build email content (simple HTML for now, matching the sprint plan)
	content := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<body style="font-family: Arial, sans-serif; padding: 20px;">
			<h2>%s</h2>
			<p>Hi,</p>
			<p>%s</p>
			<p>Best regards,<br>Calendar Sync Team</p>
		</body>
		</html>
	`, event.Title, event.Message)

	message := mail.NewSingleEmail(from, event.Title, to, "", content)

	response, err := s.client.Send(message)
	if err != nil {
		s.logger.WithError(err).Error("Failed to send SendGrid email")
		return err
	}

	if response.StatusCode >= 400 {
		return fmt.Errorf("sendgrid error: %d", response.StatusCode)
	}

	s.logger.Infof("Successfully sent %s email to %s via SendGrid", event.Type, toEmail)
	return nil
}
