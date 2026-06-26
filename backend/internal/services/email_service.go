package services

import (
	"context"
	"fmt"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// EmailService handles email notifications via SendGrid
type EmailService struct {
	apiKey    string
	fromEmail string
	fromName  string
	client    *sendgrid.Client
}

// NewEmailService creates a new email service
func NewEmailService(apiKey, fromEmail, fromName string) *EmailService {
	return &EmailService{
		apiKey:    apiKey,
		fromEmail: fromEmail,
		fromName:  fromName,
		client:    sendgrid.NewSendClient(apiKey),
	}
}

// EmailTemplate represents an email template
type EmailTemplate string

const (
	TemplateWelcome          EmailTemplate = "welcome"
	TemplatePasswordReset    EmailTemplate = "password_reset"
	TemplateTradeConfirm     EmailTemplate = "trade_confirmation"
	TemplateMonthlyStatement EmailTemplate = "monthly_statement"
	TemplateAlert            EmailTemplate = "alert"
)

// SendEmail sends a templated email
func (s *EmailService) SendEmail(ctx context.Context, to, subject string, template EmailTemplate, data map[string]interface{}) error {
	from := mail.NewEmail(s.fromName, s.fromEmail)
	toAddr := mail.NewEmail("", to)

	// Render template with data
	htmlContent := s.renderTemplate(template, data)
	plainContent := s.renderPlainText(template, data)

	message := mail.NewSingleEmail(from, subject, toAddr, plainContent, htmlContent)

	response, err := s.client.Send(message)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	if response.StatusCode >= 400 {
		return fmt.Errorf("email send failed with status %d: %s", response.StatusCode, response.Body)
	}

	return nil
}

// SendTransactionalEmail sends a one-off email without a template
func (s *EmailService) SendTransactionalEmail(ctx context.Context, to, subject, htmlBody, plainBody string) error {
	from := mail.NewEmail(s.fromName, s.fromEmail)
	toAddr := mail.NewEmail("", to)

	message := mail.NewSingleEmail(from, subject, toAddr, plainBody, htmlBody)

	response, err := s.client.Send(message)
	if err != nil {
		return fmt.Errorf("failed to send transactional email: %w", err)
	}

	if response.StatusCode >= 400 {
		return fmt.Errorf("transactional email failed with status %d", response.StatusCode)
	}

	return nil
}

// renderTemplate renders an HTML email template
func (s *EmailService) renderTemplate(template EmailTemplate, data map[string]interface{}) string {
	// In production, use a proper template engine
	// For now, return basic HTML structure
	switch template {
	case TemplateWelcome:
		name, _ := data["name"].(string)
		return fmt.Sprintf(`
			<html>
			<body style="font-family: Arial, sans-serif;">
				<h2>Welcome to Semlayer, %s!</h2>
				<p>Thank you for joining us. We're excited to have you on board.</p>
				<p>Get started by logging into your account.</p>
			</body>
			</html>
		`, name)
	case TemplateTradeConfirm:
		return fmt.Sprintf(`
			<html>
			<body style="font-family: Arial, sans-serif;">
				<h2>Trade Confirmation</h2>
				<p>Your trade has been executed successfully.</p>
				<ul>
					<li>Symbol: %v</li>
					<li>Quantity: %v</li>
					<li>Price: $%v</li>
				</ul>
			</body>
			</html>
		`, data["symbol"], data["quantity"], data["price"])
	case TemplateAlert:
		message, _ := data["message"].(string)
		return fmt.Sprintf(`
			<html>
			<body style="font-family: Arial, sans-serif;">
				<h2>⚠️ Account Alert</h2>
				<p>%s</p>
			</body>
			</html>
		`, message)
	default:
		return "<html><body><p>Email notification</p></body></html>"
	}
}

// renderPlainText renders plain text version of email
func (s *EmailService) renderPlainText(template EmailTemplate, data map[string]interface{}) string {
	switch template {
	case TemplateWelcome:
		name, _ := data["name"].(string)
		return fmt.Sprintf("Welcome to Semlayer, %s! Thank you for joining us.", name)
	case TemplateTradeConfirm:
		return fmt.Sprintf("Trade Confirmation: %v %v shares at $%v", data["symbol"], data["quantity"], data["price"])
	case TemplateAlert:
		message, _ := data["message"].(string)
		return fmt.Sprintf("Account Alert: %s", message)
	default:
		return "Email notification from Semlayer"
	}
}
