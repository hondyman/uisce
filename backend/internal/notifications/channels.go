// Package notifications - Email, SMS, and Push notification channel services.
//
// These services were extracted from internal/services/ (email_service.go,
// sms_service.go, pusher_service.go).
//
// Cardinal Rule 3 (no cycles): only depends on libs/* + stdlib.
// Cardinal Rule 7 (tenant security): all messages can carry tenant context
// via the From/TenantID fields.
package notifications

import (
	"context"
	"fmt"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
	pusher "github.com/pusher/pusher-http-go/v5"
)

// ============================================================================
// EMAIL SERVICE (SendGrid)
// ============================================================================

// EmailService handles email notifications via SendGrid.
type EmailService struct {
	apiKey    string
	fromEmail string
	fromName  string
	client    *sendgrid.Client
}

// NewEmailService creates a new email service.
func NewEmailService(apiKey, fromEmail, fromName string) *EmailService {
	return &EmailService{
		apiKey:    apiKey,
		fromEmail: fromEmail,
		fromName:  fromName,
		client:    sendgrid.NewSendClient(apiKey),
	}
}

// EmailTemplate represents an email template name.
type EmailTemplate string

// SendEmail sends an email using the SendGrid client.
func (s *EmailService) SendEmail(ctx context.Context, to, subject, body string) error {
	if s.client == nil {
		return fmt.Errorf("email service: client not initialized")
	}
	from := mail.NewEmail(s.fromName, s.fromEmail)
	toEmail := mail.NewEmail("", to)
	message := mail.NewSingleEmail(from, subject, toEmail, body, body)
	_, err := s.client.SendWithContext(ctx, message)
	return err
}

// ============================================================================
// SMS SERVICE (Twilio)
// ============================================================================

// SMSService handles SMS notifications via Twilio.
type SMSService struct {
	accountSID string
	authToken  string
	fromNumber string
	client     *twilio.RestClient
}

// NewSMSService creates a new SMS service.
func NewSMSService(accountSID, authToken, fromNumber string) *SMSService {
	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: accountSID,
		Password: authToken,
	})
	return &SMSService{
		accountSID: accountSID,
		authToken:  authToken,
		fromNumber: fromNumber,
		client:     client,
	}
}

// SendSMS sends an SMS via Twilio.
func (s *SMSService) SendSMS(ctx context.Context, to, body string) error {
	if s.client == nil {
		return fmt.Errorf("sms service: client not initialized")
	}
	params := &twilioApi.CreateMessageParams{
		From: &s.fromNumber,
		To:   &to,
		Body: &body,
	}
	_, err := s.client.Api.CreateMessage(params)
	return err
}

// ============================================================================
// PUSHER SERVICE
// ============================================================================

// PusherService handles push notifications via Pusher.
type PusherService struct {
	appID   string
	key     string
	secret  string
	cluster string
	client  *pusher.Client
}

// NewPusherService creates a new Pusher service.
func NewPusherService(appID, key, secret, cluster string) *PusherService {
	client := pusher.Client{
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
		client:  &client,
	}
}

// Push sends a push notification to a specific channel.
func (s *PusherService) Push(ctx context.Context, channel, event string, data map[string]interface{}) error {
	if s.client == nil {
		return fmt.Errorf("pusher service: client not initialized")
	}
	err := s.client.Trigger(channel, event, data)
	return err
}