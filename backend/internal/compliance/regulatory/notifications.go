package regulatory

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type NotificationService struct {
	db         *sql.DB
	httpClient *http.Client
}

func NewNotificationService(db *sql.DB) *NotificationService {
	return &NotificationService{
		db: db,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// GetUnreadNotifications returns unread blotter notifications for a tenant
func (s *NotificationService) GetUnreadNotifications(ctx context.Context, tenantID uuid.UUID) ([]ComplianceNotification, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, tenant_id, kind, title, payload, is_read, created_at
		FROM compliance.compliance_notification
		WHERE tenant_id = $1 AND is_read = false
		ORDER BY created_at DESC
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query notifications: %w", err)
	}
	defer rows.Close()

	var notifs []ComplianceNotification
	for rows.Next() {
		var n ComplianceNotification
		var payloadJSON []byte
		if err := rows.Scan(&n.ID, &n.TenantID, &n.Kind, &n.Title, &payloadJSON, &n.IsRead, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan notification: %w", err)
		}
		if len(payloadJSON) > 0 {
			_ = json.Unmarshal(payloadJSON, &n.Payload)
		}
		notifs = append(notifs, n)
	}

	return notifs, nil
}

// MarkAsRead marks a notification as read
func (s *NotificationService) MarkAsRead(ctx context.Context, notifID, tenantID uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE compliance.compliance_notification
		SET is_read = true
		WHERE id = $1 AND tenant_id = $2
	`, notifID, tenantID)
	return err
}

// SignWebhookPayload computes a cryptographically verifiable HMAC-SHA256 signature
func SignWebhookPayload(secret []byte, payload []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyWebhookSignature verifies HMAC-SHA256 signature against payload and secret
func VerifyWebhookSignature(secret []byte, payload []byte, signature string) bool {
	expected := SignWebhookPayload(secret, payload)
	return hmac.Equal([]byte(expected), []byte(signature))
}

// DispatchWebhook sends an HMAC-signed webhook to a tenant's registered endpoint
func (s *NotificationService) DispatchWebhook(ctx context.Context, endpoint string, secret []byte, payload []byte) error {
	return s.DispatchWebhookWithRetry(ctx, endpoint, secret, payload, 1, 0)
}

// DispatchWebhookWithRetry delivers an HMAC-signed webhook with retry policy
func (s *NotificationService) DispatchWebhookWithRetry(ctx context.Context, endpoint string, secret []byte, payload []byte, maxAttempts int, initialBackoff time.Duration) error {
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	if initialBackoff <= 0 {
		initialBackoff = 50 * time.Millisecond
	}

	signature := SignWebhookPayload(secret, payload)

	var lastErr error
	backoff := initialBackoff

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
		if err != nil {
			return fmt.Errorf("create webhook request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Uisce-Signature-SHA256", signature)
		req.Header.Set("X-Uisce-Timestamp", fmt.Sprintf("%d", time.Now().Unix()))

		resp, err := s.httpClient.Do(req)
		if err == nil {
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				_ = resp.Body.Close()
				return nil
			}
			_ = resp.Body.Close()
			lastErr = fmt.Errorf("endpoint returned status %d", resp.StatusCode)
		} else {
			lastErr = err
		}

		if attempt < maxAttempts {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				backoff *= 2
			}
		}
	}

	return fmt.Errorf("webhook delivery failed after %d attempts: %w", maxAttempts, lastErr)
}
