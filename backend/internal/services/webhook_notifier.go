package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/hondyman/semlayer/backend/internal/models"
)

// HTTPWebhookNotifier sends job completion notifications via HTTP
type HTTPWebhookNotifier struct {
	client        *http.Client
	timeout       time.Duration
	maxRetries    int
	retryInterval time.Duration
}

// NewHTTPWebhookNotifier creates a new HTTP webhook notifier
func NewHTTPWebhookNotifier() *HTTPWebhookNotifier {
	return &HTTPWebhookNotifier{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		timeout:       30 * time.Second,
		maxRetries:    3,
		retryInterval: 5 * time.Second,
	}
}

// NotifyJobCompletion sends a webhook notification for a completed job
func (n *HTTPWebhookNotifier) NotifyJobCompletion(ctx context.Context, job *models.AsyncJob, payload *models.JobWebhookPayload) error {
	if job.WebhookURL == "" {
		return nil
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[WebhookNotifier] Error marshaling payload: %v", err)
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	// Retry logic
	var lastErr error
	for attempt := 0; attempt < n.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(n.retryInterval):
			}
		}

		err := n.sendWebhook(ctx, job.WebhookURL, job.ID, payloadJSON)
		if err == nil {
			log.Printf("[WebhookNotifier] Successfully sent webhook notification for job %s", job.ID)
			return nil
		}

		lastErr = err
		log.Printf("[WebhookNotifier] Webhook attempt %d/%d failed: %v", attempt+1, n.maxRetries, err)
	}

	return fmt.Errorf("webhook delivery failed after %d attempts: %w", n.maxRetries, lastErr)
}

// sendWebhook performs a single webhook send operation
func (n *HTTPWebhookNotifier) sendWebhook(ctx context.Context, url, jobID string, payload []byte) error {
	ctx, cancel := context.WithTimeout(ctx, n.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Job-ID", jobID)
	req.Header.Set("User-Agent", "SemlayerJobProcessor/1.0")

	// Send request
	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	// Read response body (for logging)
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// MockWebhookNotifier for testing
type MockWebhookNotifier struct {
	notificationsSent []NotificationRecord
}

// NotificationRecord tracks a sent notification
type NotificationRecord struct {
	JobID     string
	Timestamp time.Time
	Payload   *models.JobWebhookPayload
}

// NewMockWebhookNotifier creates a mock notifier for testing
func NewMockWebhookNotifier() *MockWebhookNotifier {
	return &MockWebhookNotifier{
		notificationsSent: make([]NotificationRecord, 0),
	}
}

// NotifyJobCompletion records a notification (mock implementation)
func (m *MockWebhookNotifier) NotifyJobCompletion(ctx context.Context, job *models.AsyncJob, payload *models.JobWebhookPayload) error {
	m.notificationsSent = append(m.notificationsSent, NotificationRecord{
		JobID:     job.ID,
		Timestamp: time.Now(),
		Payload:   payload,
	})
	return nil
}

// GetNotificationsSent returns all notifications sent (mock)
func (m *MockWebhookNotifier) GetNotificationsSent() []NotificationRecord {
	return m.notificationsSent
}
