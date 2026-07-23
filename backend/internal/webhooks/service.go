package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// HasuraClient defines the interface for Hasura GraphQL operations
type HasuraClient interface {
	Query(query string, variables map[string]interface{}) (map[string]interface{}, error)
	Mutate(mutation string, variables map[string]interface{}) (map[string]interface{}, error)
}

// Service coordinates webhook subscription management and delivery tracking.
type Service struct {
	db     *sqlx.DB
	hasura HasuraClient
	client *http.Client
}

// NewService creates a webhook service backed by the provided database handle.
func NewService(db *sqlx.DB) *Service {
	return &Service{
		db:     db,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// NewServiceWithHasura creates a webhook service with Hasura support
func NewServiceWithHasura(db *sqlx.DB, hasura HasuraClient) *Service {
	return &Service{
		db:     db,
		hasura: hasura,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// Subscription mirrors the webhook_subscriptions table and exposes hydrated JSON fields.
type Subscription struct {
	SubscriptionID       uuid.UUID       `db:"subscription_id" json:"subscription_id"`
	WebhookURL           string          `db:"webhook_url" json:"webhook_url"`
	SecretKey            string          `db:"secret_key" json:"-"`
	EventTypes           pq.StringArray  `db:"event_types" json:"event_types"`
	IsActive             bool            `db:"is_active" json:"is_active"`
	FiltersRaw           json.RawMessage `db:"filters" json:"-"`
	Filters              map[string]any  `json:"filters"`
	RetryPolicyRaw       json.RawMessage `db:"retry_policy" json:"-"`
	RetryPolicy          map[string]any  `json:"retry_policy"`
	TotalDeliveries      int             `db:"total_deliveries" json:"total_deliveries"`
	SuccessfulDeliveries int             `db:"successful_deliveries" json:"successful_deliveries"`
	LastDeliveryAt       *time.Time      `db:"last_delivery_at" json:"last_delivery_at"`
	LastFailureAt        *time.Time      `db:"last_failure_at" json:"last_failure_at"`
	LastFailureReason    *string         `db:"last_failure_reason" json:"last_failure_reason"`
	CreatedAt            time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt            time.Time       `db:"updated_at" json:"updated_at"`
	CreatedBy            *uuid.UUID      `db:"created_by" json:"created_by"`
}

// DeliveryLog captures the status of an attempted delivery.
type DeliveryLog struct {
	DeliveryID     uuid.UUID  `db:"delivery_id" json:"delivery_id"`
	SubscriptionID uuid.UUID  `db:"subscription_id" json:"subscription_id"`
	EventType      string     `db:"event_type" json:"event_type"`
	EventID        uuid.UUID  `db:"event_id" json:"event_id"`
	AttemptNumber  int        `db:"attempt_number" json:"attempt_number"`
	Status         string     `db:"status" json:"status"`
	ResponseStatus *int       `db:"response_status" json:"response_status"`
	ResponseBody   *string    `db:"response_body" json:"response_body"`
	ResponseTimeMs *int       `db:"response_time_ms" json:"response_time_ms"`
	ErrorMessage   *string    `db:"error_message" json:"error_message"`
	NextRetryAt    *time.Time `db:"next_retry_at" json:"next_retry_at"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
}

// CreateSubscriptionInput describes a new webhook subscription request.
type CreateSubscriptionInput struct {
	WebhookURL  string         `json:"webhook_url"`
	EventTypes  []string       `json:"event_types"`
	Filters     map[string]any `json:"filters"`
	RetryPolicy map[string]any `json:"retry_policy"`
	IsActive    bool           `json:"is_active"`
	CreatedBy   *uuid.UUID     `json:"created_by"`
}

// Event represents an outbound webhook event payload.
type Event struct {
	ID         uuid.UUID
	Type       string
	Payload    map[string]any
	Attributes map[string]string
}

// RegisterSubscription inserts a new webhook subscription and returns the saved record.
func (s *Service) RegisterSubscription(ctx context.Context, input CreateSubscriptionInput) (*Subscription, error) {
	if input.WebhookURL == "" {
		return nil, fmt.Errorf("webhook URL is required")
	}
	if len(input.EventTypes) == 0 {
		return nil, fmt.Errorf("at least one event type must be specified")
	}

	filtersJSON, err := json.Marshal(input.Filters)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal filters: %w", err)
	}
	retryJSON, err := json.Marshal(input.RetryPolicy)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal retry policy: %w", err)
	}

	secret := generateSecret()
	subID := uuid.New()

	query := `
        INSERT INTO webhook_subscriptions (
            subscription_id, webhook_url, secret_key, event_types, filters,
            is_active, retry_policy, created_by
        ) VALUES (
            $1, $2, $3, $4, $5,
            $6, $7, $8
        )
        RETURNING subscription_id, webhook_url, secret_key, event_types, filters,
                  is_active, retry_policy, total_deliveries, successful_deliveries,
                  last_delivery_at, last_failure_at, last_failure_reason,
                  created_at, updated_at, created_by
    `

	row := s.db.QueryRowxContext(ctx, query,
		subID,
		input.WebhookURL,
		secret,
		pq.StringArray(input.EventTypes),
		filtersJSON,
		input.IsActive,
		retryJSON,
		input.CreatedBy,
	)

	var sub Subscription
	if err := row.StructScan(&sub); err != nil {
		return nil, fmt.Errorf("failed to insert webhook subscription: %w", err)
	}
	sub.Filters = unmarshalMap(sub.FiltersRaw)
	sub.RetryPolicy = unmarshalMap(sub.RetryPolicyRaw)
	return &sub, nil
}

// ListSubscriptions returns all webhook subscriptions, optionally filtered by event type.
func (s *Service) ListSubscriptions(ctx context.Context, eventType string) ([]Subscription, error) {
	query := `
        SELECT subscription_id, webhook_url, secret_key, event_types, filters,
               is_active, retry_policy, total_deliveries, successful_deliveries,
               last_delivery_at, last_failure_at, last_failure_reason,
               created_at, updated_at, created_by
        FROM webhook_subscriptions
    `
	args := []any{}
	if eventType != "" {
		query += " WHERE $1 = ANY(event_types)"
		args = append(args, eventType)
	}
	query += " ORDER BY created_at DESC"

	var subs []Subscription
	if err := s.db.SelectContext(ctx, &subs, query, args...); err != nil {
		return nil, fmt.Errorf("failed to list webhook subscriptions: %w", err)
	}
	for i := range subs {
		subs[i].Filters = unmarshalMap(subs[i].FiltersRaw)
		subs[i].RetryPolicy = unmarshalMap(subs[i].RetryPolicyRaw)
	}
	return subs, nil
}

// RotateSecret generates a new signing secret for a subscription and persists it.
func (s *Service) RotateSecret(ctx context.Context, subscriptionID uuid.UUID) (string, error) {
	newSecret := generateSecret()
	if err := s.rotateSecret(ctx, subscriptionID, newSecret); err != nil {
		return "", fmt.Errorf("failed to rotate webhook secret: %w", err)
	}
	return newSecret, nil
}

// SendTestEvent triggers an immediate delivery for the given subscription to verify configuration.
func (s *Service) SendTestEvent(ctx context.Context, subscriptionID uuid.UUID) error {
	sub, err := s.getSubscription(ctx, subscriptionID)
	if err != nil {
		return err
	}
	evt := Event{
		ID:   uuid.New(),
		Type: "advisor.webhook.test",
		Payload: map[string]any{
			"message":   "Webhook subscription test payload",
			"timestamp": time.Now().UTC(),
		},
		Attributes: map[string]string{
			"subscription_id": subscriptionID.String(),
		},
	}
	return s.dispatchToSubscription(ctx, sub, evt)
}

// DispatchEvent fan-outs an event to all matching subscriptions based on event type and filters.
func (s *Service) DispatchEvent(ctx context.Context, evt Event) error {
	if evt.ID == uuid.Nil {
		evt.ID = uuid.New()
	}
	rows, err := s.db.QueryxContext(ctx, `
        SELECT subscription_id, webhook_url, secret_key, event_types, filters,
               is_active, retry_policy, total_deliveries, successful_deliveries,
               last_delivery_at, last_failure_at, last_failure_reason,
               created_at, updated_at, created_by
        FROM webhook_subscriptions
        WHERE is_active = TRUE AND $1 = ANY(event_types)
    `, evt.Type)
	if err != nil {
		return fmt.Errorf("failed to load webhook subscriptions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var sub Subscription
		if err := rows.StructScan(&sub); err != nil {
			return fmt.Errorf("failed to scan subscription: %w", err)
		}
		sub.Filters = unmarshalMap(sub.FiltersRaw)
		sub.RetryPolicy = unmarshalMap(sub.RetryPolicyRaw)
		if matchesFilters(&sub, evt.Attributes) {
			go func(snapshot Subscription) {
				_ = s.dispatchToSubscription(context.Background(), &snapshot, evt)
			}(sub)
		}
	}
	return nil
}

func (s *Service) getSubscription(ctx context.Context, subscriptionID uuid.UUID) (*Subscription, error) {
	var sub Subscription
	err := s.db.GetContext(ctx, &sub, `
        SELECT subscription_id, webhook_url, secret_key, event_types, filters,
               is_active, retry_policy, total_deliveries, successful_deliveries,
               last_delivery_at, last_failure_at, last_failure_reason,
               created_at, updated_at, created_by
        FROM webhook_subscriptions WHERE subscription_id = $1`, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("subscription not found: %w", err)
	}
	sub.Filters = unmarshalMap(sub.FiltersRaw)
	sub.RetryPolicy = unmarshalMap(sub.RetryPolicyRaw)
	return &sub, nil
}

func (s *Service) dispatchToSubscription(ctx context.Context, sub *Subscription, evt Event) error {
	payloadEnvelope := map[string]any{
		"event_id":   evt.ID,
		"event_type": evt.Type,
		"timestamp":  time.Now().UTC(),
		"payload":    evt.Payload,
		"attributes": evt.Attributes,
	}
	body, err := json.Marshal(payloadEnvelope)
	if err != nil {
		return fmt.Errorf("failed to encode payload: %w", err)
	}

	deliveryID := uuid.New()
	err = s.recordDelivery(ctx, deliveryID, sub.SubscriptionID, evt.Type, evt.ID, body)
	if err != nil {
		return fmt.Errorf("failed to record webhook delivery: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sub.WebhookURL, bytes.NewReader(body))
	if err != nil {
		_ = s.updateDeliveryFailure(ctx, deliveryID, fmt.Sprintf("request build failed: %v", err), nil, nil)
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	signature := signPayload(sub.SecretKey, body)
	req.Header.Set("X-Webhook-Signature", signature)
	req.Header.Set("X-Webhook-Event", evt.Type)

	start := time.Now()
	resp, err := s.client.Do(req)
	if err != nil {
		_ = s.updateDeliveryFailure(ctx, deliveryID, err.Error(), nil, nil)
		return err
	}
	defer resp.Body.Close()

	duration := time.Since(start)
	respBody, _ := io.ReadAll(resp.Body)
	statusCode := resp.StatusCode
	responseMs := int(duration.Milliseconds())

	if statusCode >= 200 && statusCode < 300 {
		if err := s.updateDeliverySuccess(ctx, deliveryID, statusCode, string(respBody), responseMs); err != nil {
			return err
		}
	} else {
		_ = s.updateDeliveryFailure(ctx, deliveryID, fmt.Sprintf("status %d", statusCode), &statusCode, &responseMs)
	}
	return nil
}

func (s *Service) updateDeliverySuccess(ctx context.Context, deliveryID uuid.UUID, status int, body string, latencyMs int) error {
	return s.updateDeliverySuccessRecord(ctx, deliveryID, status, body, latencyMs)
}

func (s *Service) updateDeliveryFailure(ctx context.Context, deliveryID uuid.UUID, message string, status *int, latencyMs *int) error {
	next := time.Now().Add(5 * time.Minute)
	return s.updateDeliveryFailureRecord(ctx, deliveryID, message, status, latencyMs, next)
}

func matchesFilters(sub *Subscription, attributes map[string]string) bool {
	if len(sub.Filters) == 0 {
		return true
	}
	for key, expected := range sub.Filters {
		actual, ok := attributes[key]
		if !ok {
			return false
		}
		if fmt.Sprintf("%v", expected) != actual {
			return false
		}
	}
	return true
}

func unmarshalMap(data json.RawMessage) map[string]any {
	if len(data) == 0 {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func signPayload(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func generateSecret() string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return base64.StdEncoding.EncodeToString([]byte(uuid.NewString()))
	}
	return base64.StdEncoding.EncodeToString(buf)
}

// ============================================================================
// HASURA-FIRST HELPERS
// ============================================================================

// rotateSecret updates the webhook secret key
// TODO: Already has Hasura GraphQL mutation implemented
// SQL fallback: UPDATE secret_key with NOW() for updated_at
func (s *Service) rotateSecret(ctx context.Context, subscriptionID uuid.UUID, newSecret string) error {
	if s.hasura != nil {
		mutation := `
			mutation RotateSecret($id: uuid!, $secret: String!) {
				update_webhook_subscriptions_by_pk(
					pk_columns: {subscription_id: $id}
					_set: {secret_key: $secret, updated_at: "now()"}
				) {
					subscription_id
				}
			}
		`

		variables := map[string]interface{}{
			"id":     subscriptionID,
			"secret": newSecret,
		}

		_, err := s.hasura.Mutate(mutation, variables)
		if err == nil {
			return nil
		}
		// Fall through to SQL on Hasura error
	}

	// SQL fallback
	_, err := s.db.ExecContext(ctx,
		`UPDATE webhook_subscriptions SET secret_key = $2, updated_at = NOW() WHERE subscription_id = $1`,
		subscriptionID, newSecret,
	)
	return err
}

// recordDelivery inserts a new webhook delivery record
// TODO: Already has Hasura GraphQL mutation implemented
// SQL fallback: INSERT webhook_deliveries with status PENDING
func (s *Service) recordDelivery(ctx context.Context, deliveryID, subscriptionID uuid.UUID, eventType string, eventID uuid.UUID, payload []byte) error {
	if s.hasura != nil {
		mutation := `
			mutation RecordDelivery(
				$deliveryID: uuid!
				$subscriptionID: uuid!
				$eventType: String!
				$eventID: uuid!
				$payload: jsonb!
			) {
				insert_webhook_deliveries_one(object: {
					delivery_id: $deliveryID
					subscription_id: $subscriptionID
					event_type: $eventType
					event_id: $eventID
					payload: $payload
					attempt_number: 1
					status: "PENDING"
					created_at: "now()"
				}) {
					delivery_id
				}
			}
		`

		var payloadJSON interface{}
		if err := json.Unmarshal(payload, &payloadJSON); err != nil {
			payloadJSON = string(payload)
		}

		variables := map[string]interface{}{
			"deliveryID":     deliveryID,
			"subscriptionID": subscriptionID,
			"eventType":      eventType,
			"eventID":        eventID,
			"payload":        payloadJSON,
		}

		_, err := s.hasura.Mutate(mutation, variables)
		if err == nil {
			return nil
		}
		// Fall through to SQL on Hasura error
	}

	// SQL fallback
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO webhook_deliveries (
			delivery_id, subscription_id, event_type, event_id, payload,
			attempt_number, status, created_at
		) VALUES ($1, $2, $3, $4, $5, 1, 'PENDING', NOW())
	`, deliveryID, subscriptionID, eventType, eventID, payload)
	return err
}

// updateDeliverySuccessRecord marks a delivery as successful
// TODO: Already has Hasura GraphQL mutation implemented
// SQL fallback: UPDATE status to SUCCESS with response details
func (s *Service) updateDeliverySuccessRecord(ctx context.Context, deliveryID uuid.UUID, status int, body string, latencyMs int) error {
	if s.hasura != nil {
		mutation := `
			mutation UpdateDeliverySuccess($id: uuid!, $status: Int!, $body: String!, $latency: Int!) {
				update_webhook_deliveries_by_pk(
					pk_columns: {delivery_id: $id}
					_set: {
						status: "SUCCESS"
						response_status: $status
						response_body: $body
						response_time_ms: $latency
					}
				) {
					delivery_id
				}
			}
		`

		variables := map[string]interface{}{
			"id":      deliveryID,
			"status":  status,
			"body":    body,
			"latency": latencyMs,
		}

		_, err := s.hasura.Mutate(mutation, variables)
		if err == nil {
			return nil
		}
		// Fall through to SQL on Hasura error
	}

	// SQL fallback
	_, err := s.db.ExecContext(ctx, `
		UPDATE webhook_deliveries
		SET status = 'SUCCESS', response_status = $2, response_body = $3,
			response_time_ms = $4
		WHERE delivery_id = $1
	`, deliveryID, status, body, latencyMs)
	return err
}

// updateDeliveryFailureRecord marks a delivery as failed
// TODO: Already has Hasura GraphQL mutation implemented
// SQL fallback: UPDATE status to FAILED with error details and next_retry_at
func (s *Service) updateDeliveryFailureRecord(ctx context.Context, deliveryID uuid.UUID, message string, status *int, latencyMs *int, nextRetry time.Time) error {
	if s.hasura != nil {
		mutation := `
			mutation UpdateDeliveryFailure(
				$id: uuid!
				$message: String!
				$status: Int
				$latency: Int
				$nextRetry: timestamptz!
			) {
				update_webhook_deliveries_by_pk(
					pk_columns: {delivery_id: $id}
					_set: {
						status: "FAILED"
						error_message: $message
						response_status: $status
						response_time_ms: $latency
						next_retry_at: $nextRetry
					}
				) {
					delivery_id
				}
			}
		`

		variables := map[string]interface{}{
			"id":        deliveryID,
			"message":   message,
			"status":    status,
			"latency":   latencyMs,
			"nextRetry": nextRetry,
		}

		_, err := s.hasura.Mutate(mutation, variables)
		if err == nil {
			return nil
		}
		// Fall through to SQL on Hasura error
	}

	// SQL fallback
	_, err := s.db.ExecContext(ctx, `
		UPDATE webhook_deliveries
		SET status = 'FAILED', error_message = $2, response_status = $3,
			response_time_ms = $4, next_retry_at = $5
		WHERE delivery_id = $1
	`, deliveryID, message, status, latencyMs, nextRetry)
	return err
}
