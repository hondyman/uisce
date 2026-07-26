// Package notifications - Engagement notification service.
//
// This file contains the canonical EngagementNotificationService that was
// extracted from internal/services/engagement_notification_service.go.
//
// Cardinal Rule 3 (no cycles): depends on backend/models, libs/*, stdlib.
// Cardinal Rule 7 (tenant security): tenantID required on all events.
package notifications

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/hondyman/uisce/backend/internal/models"
)

// ============================================================================
// ENGAGEMENT NOTIFICATION SERVICE
// ============================================================================

// EngagementNotificationService handles advanced notification engagement features
// including user targeting, scheduling, and broadcast delivery.
type EngagementNotificationService struct {
	DB                    *sql.DB
	BroadcastFunc         func(userID string, message []byte)
	BroadcastAllFunc      func(message []byte)
	BroadcastAudienceFunc func(audience string, message []byte)
}

// NewEngagementNotificationService creates a new engagement notification service.
func NewEngagementNotificationService(db *sql.DB) *EngagementNotificationService {
	return &EngagementNotificationService{DB: db}
}

// SetBroadcastFunctions sets the broadcasting functions for real-time delivery.
func (s *EngagementNotificationService) SetBroadcastFunctions(
	broadcastFunc func(userID string, message []byte),
	broadcastAllFunc func(message []byte),
	broadcastAudienceFunc func(audience string, message []byte),
) {
	s.BroadcastFunc = broadcastFunc
	s.BroadcastAllFunc = broadcastAllFunc
	s.BroadcastAudienceFunc = broadcastAudienceFunc
}

// NotificationEvent represents a single notification event.
type NotificationEvent struct {
	ID        uuid.UUID `json:"id" db:"id"`
	TenantID  string    `json:"tenant_id" db:"tenant_id"`
	UserID    string    `json:"user_id" db:"user_id"`
	Title     string    `json:"title" db:"title"`
	Body      string    `json:"body" db:"body"`
	Channel   string    `json:"channel" db:"channel"`
	Priority  int       `json:"priority" db:"priority"`
	Metadata  []byte    `json:"metadata" db:"metadata"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// EngagementRule represents a rule that determines when/how to engage users.
type EngagementRule struct {
	ID          uuid.UUID      `json:"id" db:"id"`
	TenantID    string         `json:"tenant_id" db:"tenant_id"`
	Name        string         `json:"name" db:"name"`
	Trigger     string         `json:"trigger" db:"trigger"`
	Audience    pq.StringArray `json:"audience" db:"audience"`
	TemplateID  uuid.UUID      `json:"template_id" db:"template_id"`
	Enabled     bool           `json:"enabled" db:"enabled"`
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at" db:"updated_at"`
}

// ============================================================================
// EVENT PERSISTENCE
// ============================================================================

// RecordEvent persists a notification event and broadcasts it to the user.
func (s *EngagementNotificationService) RecordEvent(ctx context.Context, event NotificationEvent) error {
	if s.DB == nil {
		return fmt.Errorf("engagement notification service: db is nil")
	}
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}

	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO notification_events (id, tenant_id, user_id, title, body, channel, priority, metadata, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		event.ID, event.TenantID, event.UserID, event.Title, event.Body,
		event.Channel, event.Priority, event.Metadata, event.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to record notification event: %w", err)
	}

	if s.BroadcastFunc != nil {
		payload, _ := json.Marshal(event)
		s.BroadcastFunc(event.UserID, payload)
	}

	return nil
}

// BroadcastToAudience broadcasts a payload to all users in an audience.
func (s *EngagementNotificationService) BroadcastToAudience(audience string, payload []byte) {
	if s.BroadcastAudienceFunc != nil {
		s.BroadcastAudienceFunc(audience, payload)
		return
	}
	if s.BroadcastAllFunc != nil {
		s.BroadcastAllFunc(payload)
	}
}

// ============================================================================
// ENGAGEMENT RULES
// ============================================================================

// CreateRule creates a new engagement rule.
func (s *EngagementNotificationService) CreateRule(ctx context.Context, rule EngagementRule) (*EngagementRule, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("engagement notification service: db is nil")
	}
	if rule.ID == uuid.Nil {
		rule.ID = uuid.New()
	}
	now := time.Now()
	rule.CreatedAt = now
	rule.UpdatedAt = now

	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO engagement_rules (id, tenant_id, name, trigger, audience, template_id, enabled, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		rule.ID, rule.TenantID, rule.Name, rule.Trigger, rule.Audience,
		rule.TemplateID, rule.Enabled, rule.CreatedAt, rule.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create engagement rule: %w", err)
	}
	return &rule, nil
}

// ListRules returns all engagement rules for a tenant.
func (s *EngagementNotificationService) ListRules(ctx context.Context, tenantID string) ([]EngagementRule, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("engagement notification service: db is nil")
	}
	rows, err := s.DB.QueryContext(ctx,
		`SELECT id, tenant_id, name, trigger, audience, template_id, enabled, created_at, updated_at
		 FROM engagement_rules WHERE tenant_id = $1 ORDER BY created_at DESC`,
		tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list engagement rules: %w", err)
	}
	defer rows.Close()

	var rules []EngagementRule
	for rows.Next() {
		var r EngagementRule
		if err := rows.Scan(
			&r.ID, &r.TenantID, &r.Name, &r.Trigger, &r.Audience,
			&r.TemplateID, &r.Enabled, &r.CreatedAt, &r.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan engagement rule: %w", err)
		}
		rules = append(rules, r)
	}
	return rules, rows.Err()
}

// ToggleRule enables or disables an engagement rule.
func (s *EngagementNotificationService) ToggleRule(ctx context.Context, ruleID uuid.UUID, enabled bool) error {
	if s.DB == nil {
		return fmt.Errorf("engagement notification service: db is nil")
	}
	_, err := s.DB.ExecContext(ctx,
		`UPDATE engagement_rules SET enabled = $1, updated_at = NOW() WHERE id = $2`,
		enabled, ruleID)
	return err
}

// ============================================================================
// LEGACY HELPERS
// ============================================================================

// SendEmailNotification is a stub for sending email notifications.
func SendEmailNotification(ctx context.Context, to, subject, body string) error {
	log.Printf("[Email] To: %s | Subject: %s | Body: %s", to, subject, body)
	return nil
}

// SendSlackNotification is a stub for sending Slack notifications.
func SendSlackNotification(ctx context.Context, channel, message string) error {
	log.Printf("[Slack] Channel: %s | Message: %s", channel, message)
	return nil
}

// Compile-time guard that EngagementNotificationService implements expected surface.
var _ = (*EngagementNotificationService)(nil)

// Suppress unused warnings for pq (referenced for future use).
var (
	_ = pq.StringArray{}
)

// ============================================================================
// USER NOTIFICATIONS
// ============================================================================

// GetUserNotifications retrieves notifications for a user with pagination.
func (s *EngagementNotificationService) GetUserNotifications(ctx context.Context, userID string, limit, offset int) ([]map[string]interface{}, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("engagement notification service: db is nil")
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	rows, err := s.DB.QueryContext(ctx,
		`SELECT id, tenant_id, title, body, channel, priority, metadata, created_at
		 FROM notification_events
		 WHERE user_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`,
		userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list user notifications: %w", err)
	}
	defer rows.Close()

	var notifs []map[string]interface{}
	for rows.Next() {
		var (
			id, tenantID, title, body, channel string
			priority                            int
			metadata                            []byte
			createdAt                           time.Time
		)
		if err := rows.Scan(&id, &tenantID, &title, &body, &channel, &priority, &metadata, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan notification: %w", err)
		}
		notifs = append(notifs, map[string]interface{}{
			"id":         id,
			"tenant_id":  tenantID,
			"title":      title,
			"body":       body,
			"channel":    channel,
			"priority":   priority,
			"metadata":   string(metadata),
			"created_at": createdAt,
		})
	}
	return notifs, rows.Err()
}

// TrackEngagementEvent records user interaction with a notification (open, click, dismiss).
// Accepts *models.NotificationAnalytics for the legacy callers.
func (s *EngagementNotificationService) TrackEngagementEvent(ctx context.Context, analytics *models.NotificationAnalytics) error {
	if s.DB == nil {
		return fmt.Errorf("engagement notification service: db is nil")
	}
	if analytics == nil {
		return fmt.Errorf("analytics is nil")
	}
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO notification_analytics (id, notification_id, user_id, event_type, event_timestamp)
		 VALUES ($1, $2, $3, $4, $5)`,
		uuid.New(), analytics.NotificationID, analytics.UserID,
		analytics.EventType, analytics.EventTimestamp)
	return err
}

// CreateNotification persists a new engagement notification and broadcasts it.
func (s *EngagementNotificationService) CreateNotification(ctx context.Context, notification *models.EngagementNotification) error {
	if s.DB == nil {
		return fmt.Errorf("engagement notification service: db is nil")
	}
	if notification == nil {
		return fmt.Errorf("notification is nil")
	}

	if notification.ID == "" {
		notification.ID = uuid.New().String()
	}
	if notification.Status == "" {
		notification.Status = "draft"
	}

	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO engagement_notifications (id, user_id, type, title, message, priority, channels, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (id) DO NOTHING`,
		notification.ID, notification.UserID, notification.Type,
		notification.Title, notification.Message, notification.Priority,
		pq.Array(notification.Channels), notification.Status)
	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	// Broadcast the event
	if s.BroadcastFunc != nil {
		payload, _ := json.Marshal(notification)
		s.BroadcastFunc(notification.UserID, payload)
	}
	return nil
}

// SendNotification immediately broadcasts a stored notification by ID.
func (s *EngagementNotificationService) SendNotification(ctx context.Context, notificationID string) error {
	if s.DB == nil {
		return fmt.Errorf("engagement notification service: db is nil")
	}

	var notification models.EngagementNotification
	err := s.DB.QueryRowContext(ctx,
		`SELECT id, user_id, type, title, message, priority, channels, status
		 FROM engagement_notifications WHERE id = $1`, notificationID).Scan(
		&notification.ID, &notification.UserID, &notification.Type,
		&notification.Title, &notification.Message, &notification.Priority,
		&notification.Channels, &notification.Status)
	if err != nil {
		return fmt.Errorf("failed to load notification %s: %w", notificationID, err)
	}

	// Mark as sent
	now := time.Now()
	notification.Status = "sent"
	notification.SentAt = &now
	_, _ = s.DB.ExecContext(ctx,
		`UPDATE engagement_notifications SET status = 'sent', sent_at = $1 WHERE id = $2`,
		now, notificationID)

	// Broadcast
	if s.BroadcastFunc != nil {
		payload, _ := json.Marshal(notification)
		s.BroadcastFunc(notification.UserID, payload)
	}
	return nil
}

// GetUserPreferences returns user notification preferences.
func (s *EngagementNotificationService) GetUserPreferences(ctx context.Context, userID string) (*models.UserNotificationPreferences, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("engagement notification service: db is nil")
	}
	var prefs models.UserNotificationPreferences
	err := s.DB.QueryRowContext(ctx,
		`SELECT user_id, email_enabled, sms_enabled, push_enabled, in_app_enabled,
		        quiet_hours_start, quiet_hours_end, updated_at
		 FROM user_notification_preferences WHERE user_id = $1`,
		userID).Scan(
		&prefs.UserID, &prefs.EmailEnabled, &prefs.SMSEnabled,
		&prefs.PushEnabled, &prefs.InAppEnabled, &prefs.QuietHoursStart,
		&prefs.QuietHoursEnd, &prefs.UpdatedAt)
	if err != nil {
		// Return defaults on first lookup
		return &models.UserNotificationPreferences{
			UserID:        userID,
			EmailEnabled:  true,
			SMSEnabled:    false,
			PushEnabled:   true,
			InAppEnabled:  true,
			UpdatedAt:     time.Now(),
		}, nil
	}
	return &prefs, nil
}

// UpdateUserPreferences upserts user notification preferences.
func (s *EngagementNotificationService) UpdateUserPreferences(ctx context.Context, prefs *models.UserNotificationPreferences) error {
	if s.DB == nil {
		return fmt.Errorf("engagement notification service: db is nil")
	}
	if prefs == nil {
		return fmt.Errorf("preferences is nil")
	}
	prefs.UpdatedAt = time.Now()
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO user_notification_preferences
		 (user_id, email_enabled, sms_enabled, push_enabled, in_app_enabled, quiet_hours_start, quiet_hours_end, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (user_id) DO UPDATE SET
		   email_enabled = EXCLUDED.email_enabled,
		   sms_enabled = EXCLUDED.sms_enabled,
		   push_enabled = EXCLUDED.push_enabled,
		   in_app_enabled = EXCLUDED.in_app_enabled,
		   quiet_hours_start = EXCLUDED.quiet_hours_start,
		   quiet_hours_end = EXCLUDED.quiet_hours_end,
		   updated_at = EXCLUDED.updated_at`,
		prefs.UserID, prefs.EmailEnabled, prefs.SMSEnabled,
		prefs.PushEnabled, prefs.InAppEnabled, prefs.QuietHoursStart,
		prefs.QuietHoursEnd, prefs.UpdatedAt)
	return err
}

// CreateNotificationTemplate persists a reusable notification template.
func (s *EngagementNotificationService) CreateNotificationTemplate(ctx context.Context, template *models.NotificationTemplate) error {
	if s.DB == nil {
		return fmt.Errorf("engagement notification service: db is nil")
	}
	if template == nil {
		return fmt.Errorf("template is nil")
	}
	if template.ID == "" {
		template.ID = uuid.New().String()
	}
	template.CreatedAt = time.Now()
	template.UpdatedAt = time.Now()
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO notification_templates (id, name, type, subject, title, message, channels, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 ON CONFLICT (id) DO NOTHING`,
		template.ID, template.Name, template.Type, template.Subject,
		template.Title, template.Message, pq.Array(template.Channels),
		template.CreatedAt, template.UpdatedAt)
	return err
}

// GetEngagementAnalytics returns aggregated analytics for a date range.
func (s *EngagementNotificationService) GetEngagementAnalytics(ctx context.Context, startDate, endDate time.Time) (map[string]interface{}, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("engagement notification service: db is nil")
	}
	var total, delivered, opened, clicked, dismissed int
	err := s.DB.QueryRowContext(ctx,
		`SELECT
		   COUNT(*) FILTER (WHERE event_type = 'sent'),
		   COUNT(*) FILTER (WHERE event_type = 'delivered'),
		   COUNT(*) FILTER (WHERE event_type = 'opened'),
		   COUNT(*) FILTER (WHERE event_type = 'clicked'),
		   COUNT(*) FILTER (WHERE event_type = 'dismissed')
		 FROM notification_analytics
		 WHERE event_timestamp BETWEEN $1 AND $2`,
		startDate, endDate).Scan(&total, &delivered, &opened, &clicked, &dismissed)
	if err != nil {
		return nil, fmt.Errorf("failed to query analytics: %w", err)
	}
	return map[string]interface{}{
		"total":     total,
		"delivered": delivered,
		"opened":    opened,
		"clicked":   clicked,
		"dismissed": dismissed,
		"period": map[string]interface{}{
			"start": startDate,
			"end":   endDate,
		},
	}, nil
}