package notifications

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/smtp"

	"github.com/sirupsen/logrus"
)

// EmailSender implements NotificationService using standard SMTP
type EmailSender struct {
	host      string
	port      string
	username  string
	password  string
	from      string
	logger    *logrus.Entry
	templates *template.Template
}

// SMTPConfig holds SMTP server configuration
type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

// NewEmailSender creates a new SMTP email sender
func NewEmailSender(cfg SMTPConfig, logger *logrus.Entry) *EmailSender {
	// Parse HTML templates (mocked inline for simplicity, usually loaded from files)
	t := template.New("email")
	t, _ = t.Parse(`
	{{define "SYNC_ERROR"}}
		<h1>Calendar Sync Error</h1>
		<p>Hi,</p>
		<p>We encountered an error syncing your calendar: <b>{{.Message}}</b></p>
		<p>Please check your connection settings.</p>
	{{end}}
	{{define "CONFLICT_DETECTED"}}
		<h1>Calendar Sync Conflict</h1>
		<p>We detected a conflict during sync.</p>
		<p>{{.Message}}</p>
		<p>Please log in to your dashboard to resolve it.</p>
	{{end}}
	{{define "DEFAULT"}}
		<h1>Calendar Notification</h1>
		<p>{{.Message}}</p>
	{{end}}
	`)

	return &EmailSender{
		host:      cfg.Host,
		port:      cfg.Port,
		username:  cfg.Username,
		password:  cfg.Password,
		from:      cfg.From,
		logger:    logger.WithField("service", "email_sender"),
		templates: t,
	}
}

// SendNotification sends an email based on the event
func (s *EmailSender) SendNotification(ctx context.Context, event NotificationEvent) error {
	// In a real app, you would look up the user's email address from the DB using event.UserID
	// For this implementation, we will assume it's passed in event.Data["email"] or mock it.
	toEmail, ok := event.Data["email"].(string)
	if !ok || toEmail == "" {
		s.logger.Warnf("No email provided for user %s, skipping email notification", event.UserID)
		return nil
	}

	// Render template
	var bodyBuffer bytes.Buffer
	tmplName := event.Type
	if s.templates.Lookup(tmplName) == nil {
		tmplName = "DEFAULT"
	}

	if err := s.templates.ExecuteTemplate(&bodyBuffer, tmplName, event); err != nil {
		s.logger.WithError(err).Error("Failed to execute email template")
		return err
	}

	subject := fmt.Sprintf("Subject: %s\r\n", event.Title)
	mime := "MIME-version: 1.0;\r\nContent-Type: text/html; charset=\"UTF-8\";\r\n\r\n"
	msg := []byte(subject + mime + bodyBuffer.String())

	// If no SMTP host is configured, just log it (useful for local dev)
	if s.host == "" {
		s.logger.WithFields(logrus.Fields{
			"to":      toEmail,
			"subject": event.Title,
			"type":    event.Type,
		}).Info("Would send email (SMTP host not configured)")
		return nil
	}

	// Send email
	auth := smtp.PlainAuth("", s.username, s.password, s.host)
	addr := fmt.Sprintf("%s:%s", s.host, s.port)

	err := smtp.SendMail(addr, auth, s.from, []string{toEmail}, msg)
	if err != nil {
		s.logger.WithError(err).Error("Failed to send SMTP email")
		return err
	}

	s.logger.Infof("Successfully sent %s email to %s", event.Type, toEmail)
	return nil
}
