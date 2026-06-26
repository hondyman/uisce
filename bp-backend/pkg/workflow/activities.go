package workflow

import (
	"context"
	"fmt"

	"go.temporal.io/sdk/activity"
)

// SendEmailActivity is a placeholder activity that logs the email details.
func SendEmailActivity(ctx context.Context, to, subject, body string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Sending email", "To", to, "Subject", subject)
	// In a real implementation, you would use an email client here.
	return fmt.Sprintf("Email sent to %s", to), nil
}

// ChargeCreditCardActivity is a placeholder activity that logs the charge details.
func ChargeCreditCardActivity(ctx context.Context, amount float64, cardNumber string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Charging credit card", "Amount", amount, "CardNumber", "ending in "+cardNumber[len(cardNumber)-4:])
	// In a real implementation, you would use a payment gateway here.
	return fmt.Sprintf("Charged %.2f to card", amount), nil
}

// CreateUserActivity is a placeholder activity that logs the user creation details.
func CreateUserActivity(ctx context.Context, username, email string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Creating user", "Username", username, "Email", email)
	// In a real implementation, you would interact with your database here.
	return fmt.Sprintf("User %s created", username), nil
}
