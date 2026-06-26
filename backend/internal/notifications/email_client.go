package notifications

import (
	"context"
	"fmt"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

type SendGridClient struct {
	client    *sendgrid.Client
	fromEmail string
	fromName  string
}

func NewSendGridClient(apiKey, fromEmail, fromName string) *SendGridClient {
	return &SendGridClient{
		client:    sendgrid.NewSendClient(apiKey),
		fromEmail: fromEmail,
		fromName:  fromName,
	}
}

func (c *SendGridClient) Send(ctx context.Context, toEmail, subject, htmlBody string) error {
	if c.client == nil || c.fromEmail == "" {
		// No-op / mock if not configured
		return nil
	}

	from := mail.NewEmail(c.fromName, c.fromEmail)
	to := mail.NewEmail("", toEmail)

	message := mail.NewSingleEmail(from, subject, to, subject, htmlBody)
	response, err := c.client.SendWithContext(ctx, message)

	if err != nil {
		return err
	}

	if response.StatusCode >= 400 {
		return fmt.Errorf("sendgrid error: %d", response.StatusCode)
	}

	return nil
}
