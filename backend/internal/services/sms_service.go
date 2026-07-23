package services

import (
	"context"
	"fmt"

	"github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
)

// SMSService handles SMS notifications via Twilio
type SMSService struct {
	accountSID string
	authToken  string
	fromNumber string
	client     *twilio.RestClient
}

// NewSMSService creates a new SMS service
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

// SendSMS sends an SMS message
func (s *SMSService) SendSMS(ctx context.Context, to, message string) error {
	params := &twilioApi.CreateMessageParams{}
	params.SetTo(to)
	params.SetFrom(s.fromNumber)
	params.SetBody(message)

	resp, err := s.client.Api.CreateMessage(params)
	if err != nil {
		return fmt.Errorf("failed to send SMS: %w", err)
	}

	if resp.ErrorCode != nil {
		return fmt.Errorf("SMS error %d: %s", *resp.ErrorCode, *resp.ErrorMessage)
	}

	return nil
}

// SendSecurityCode sends a security/2FA code via SMS
func (s *SMSService) SendSecurityCode(ctx context.Context, to, code string) error {
	message := fmt.Sprintf("Your Semlayer security code is: %s. Do not share this code with anyone.", code)
	return s.SendSMS(ctx, to, message)
}

// SendTradeAlert sends a trade alert via SMS
func (s *SMSService) SendTradeAlert(ctx context.Context, to, symbol string, quantity int, price float64) error {
	message := fmt.Sprintf("Trade Alert: %s - %d shares at $%.2f", symbol, quantity, price)
	return s.SendSMS(ctx, to, message)
}

// SendAccountAlert sends an account alert via SMS
func (s *SMSService) SendAccountAlert(ctx context.Context, to, alertMessage string) error {
	message := fmt.Sprintf("⚠️ Account Alert: %s", alertMessage)
	return s.SendSMS(ctx, to, message)
}

// SendBulkSMS sends SMS to multiple recipients
func (s *SMSService) SendBulkSMS(ctx context.Context, recipients []string, message string) []error {
	errors := make([]error, 0)

	for _, to := range recipients {
		if err := s.SendSMS(ctx, to, message); err != nil {
			errors = append(errors, fmt.Errorf("failed to send to %s: %w", to, err))
		}
	}

	return errors
}
