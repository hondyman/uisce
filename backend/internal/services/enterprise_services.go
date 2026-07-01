package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// ============================================================================
// TRANSLATION SERVICE (i18n)
// ============================================================================

// TranslationService provides multilingual support
type TranslationService struct {
	db     *sqlx.DB
	logger *zap.Logger
	cache  map[string]map[string]string // locale -> namespace.key -> value
}

// NewTranslationService creates a new translation service
func NewTranslationService(db *sqlx.DB) *TranslationService {
	logger, _ := zap.NewProduction()
	return &TranslationService{
		db:     db,
		logger: logger,
		cache:  make(map[string]map[string]string),
	}
}

// GetTranslation fetches a translation for a key
func (s *TranslationService) GetTranslation(ctx context.Context, locale, namespace, key, defaultValue string) string {
	// Check cache
	cacheKey := fmt.Sprintf("%s.%s.%s", locale, namespace, key)
	if translations, ok := s.cache[locale]; ok {
		if value, exists := translations[cacheKey]; exists {
			return value
		}
	}

	var value string
	err := s.db.GetContext(ctx, &value, `
		SELECT value FROM translations
		WHERE locale_code = $1 AND namespace = $2 AND key = $3
		LIMIT 1
	`, locale, namespace, key)

	if err != nil {
		return defaultValue
	}

	// Update cache
	if s.cache[locale] == nil {
		s.cache[locale] = make(map[string]string)
	}
	s.cache[locale][cacheKey] = value
	return value
}

// GetTranslations fetches multiple translations for a namespace
func (s *TranslationService) GetTranslations(ctx context.Context, locale, namespace string) (map[string]string, error) {
	type row struct {
		Key   string `db:"key"`
		Value string `db:"value"`
	}
	var rows []row
	err := s.db.SelectContext(ctx, &rows, `
		SELECT key, value FROM translations
		WHERE locale_code = $1 AND namespace = $2
	`, locale, namespace)

	if err != nil {
		return nil, err
	}

	result := make(map[string]string, len(rows))
	for _, r := range rows {
		result[r.Key] = r.Value
	}
	return result, nil
}

// ============================================================================
// AUDIT SERVICE
// ============================================================================

// AuditService provides comprehensive audit logging
type AuditService struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewAuditService creates a new audit service
func NewAuditService(db *sqlx.DB) *AuditService {
	logger, _ := zap.NewProduction()
	return &AuditService{db: db, logger: logger}
}

// AuditEntry represents an audit log entry
type AuditEntry struct {
	EntityType string                 `json:"entity_type"`
	EntityID   string                 `json:"entity_id"`
	EntityName string                 `json:"entity_name,omitempty"`
	Action     string                 `json:"action"`
	Actor      string                 `json:"actor"`
	ActorType  string                 `json:"actor_type,omitempty"`
	IPAddress  string                 `json:"ip_address,omitempty"`
	UserAgent  string                 `json:"user_agent,omitempty"`
	Changes    map[string]interface{} `json:"changes,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Severity   string                 `json:"severity,omitempty"`
}

// LogAudit creates an audit log entry
func (s *AuditService) LogAudit(ctx context.Context, entry AuditEntry) error {
	changesJSON, _ := json.Marshal(entry.Changes)
	metadataJSON, _ := json.Marshal(entry.Metadata)

	id := uuid.New().String()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO audit_log (
			id, entity_type, entity_id, entity_name, action, actor,
			actor_type, ip_address, user_agent, changes, metadata, severity, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11, $12, NOW()
		)
	`, id, entry.EntityType, entry.EntityID, entry.EntityName, entry.Action, entry.Actor,
		entry.ActorType, entry.IPAddress, entry.UserAgent,
		string(changesJSON), string(metadataJSON), entry.Severity)

	if err != nil {
		s.logger.Error("Failed to create audit log", zap.Error(err))
		return err
	}
	return nil
}

// GetAuditHistory fetches audit history for an entity
func (s *AuditService) GetAuditHistory(ctx context.Context, entityType, entityID string, limit int) ([]map[string]interface{}, error) {
	type row struct {
		ID        string `db:"id"`
		Action    string `db:"action"`
		Actor     string `db:"actor"`
		ActorType string `db:"actor_type"`
		Changes   string `db:"changes"`
		Metadata  string `db:"metadata"`
		CreatedAt string `db:"created_at"`
	}

	var rows []row
	err := s.db.SelectContext(ctx, &rows, `
		SELECT id, action, actor, COALESCE(actor_type,'') as actor_type,
		       COALESCE(changes,'{}') as changes, COALESCE(metadata,'{}') as metadata,
		       created_at::text as created_at
		FROM audit_log
		WHERE entity_type = $1 AND entity_id = $2
		ORDER BY created_at DESC
		LIMIT $3
	`, entityType, entityID, limit)

	if err != nil {
		return nil, err
	}

	history := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		history = append(history, map[string]interface{}{
			"id":         r.ID,
			"action":     r.Action,
			"actor":      r.Actor,
			"actor_type": r.ActorType,
			"changes":    r.Changes,
			"metadata":   r.Metadata,
			"created_at": r.CreatedAt,
		})
	}
	return history, nil
}

// ============================================================================
// NOTIFICATION SERVICE
// ============================================================================

// NotificationService provides notification management
type NotificationService struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewNotificationService creates a new notification service
func NewNotificationService(db *sqlx.DB) *NotificationService {
	logger, _ := zap.NewProduction()
	return &NotificationService{db: db, logger: logger}
}

// NotificationMessage represents a notification message
type NotificationMessage struct {
	RecipientID   string                 `json:"recipient_id"`
	RecipientType string                 `json:"recipient_type,omitempty"`
	Channel       string                 `json:"channel"`
	Priority      string                 `json:"priority,omitempty"`
	Category      string                 `json:"category"`
	Title         string                 `json:"title"`
	Message       string                 `json:"message"`
	Data          map[string]interface{} `json:"data,omitempty"`
	LinkURL       string                 `json:"link_url,omitempty"`
	LinkText      string                 `json:"link_text,omitempty"`
}

// SendNotification creates a notification
func (s *NotificationService) SendNotification(ctx context.Context, notif NotificationMessage) (string, error) {
	dataJSON, _ := json.Marshal(notif.Data)
	id := uuid.New().String()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO notifications (
			id, recipient_id, recipient_type, channel, priority,
			category, title, message, data, link_url, link_text,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10, $11,
			NOW(), NOW()
		)
	`, id, notif.RecipientID, notif.RecipientType, notif.Channel, notif.Priority,
		notif.Category, notif.Title, notif.Message, string(dataJSON),
		notif.LinkURL, notif.LinkText)

	if err != nil {
		return "", err
	}
	return id, nil
}

// SendFromTemplate sends a notification using a template
func (s *NotificationService) SendFromTemplate(ctx context.Context, templateKey, recipientID string, variables map[string]string) error {
	var tmpl struct {
		SubjectTemplate string `db:"subject_template"`
		BodyTemplate    string `db:"body_template"`
		Category        string `db:"category"`
		Channels        string `db:"channels"`
	}

	err := s.db.GetContext(ctx, &tmpl, `
		SELECT subject_template, body_template, category, channels::text
		FROM notification_templates
		WHERE key = $1
		LIMIT 1
	`, templateKey)

	if err != nil {
		return fmt.Errorf("template not found: %s", templateKey)
	}

	subject := tmpl.SubjectTemplate
	body := tmpl.BodyTemplate
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		subject = strings.ReplaceAll(subject, placeholder, value)
		body = strings.ReplaceAll(body, placeholder, value)
	}

	// channels stored as JSON array string e.g. ["email","sms"]
	var channels []string
	_ = json.Unmarshal([]byte(tmpl.Channels), &channels)

	for _, channel := range channels {
		_, err := s.SendNotification(ctx, NotificationMessage{
			RecipientID: recipientID,
			Channel:     channel,
			Category:    tmpl.Category,
			Title:       subject,
			Message:     body,
		})
		if err != nil {
			s.logger.Error("Failed to send notification", zap.Error(err))
		}
	}
	return nil
}

// GetUnreadNotifications fetches unread notifications for a user
func (s *NotificationService) GetUnreadNotifications(ctx context.Context, recipientID string) ([]map[string]interface{}, error) {
	type row struct {
		ID        string `db:"id"`
		Category  string `db:"category"`
		Title     string `db:"title"`
		Message   string `db:"message"`
		LinkURL   string `db:"link_url"`
		LinkText  string `db:"link_text"`
		Priority  string `db:"priority"`
		CreatedAt string `db:"created_at"`
	}

	var rows []row
	err := s.db.SelectContext(ctx, &rows, `
		SELECT id, COALESCE(category,'') as category, COALESCE(title,'') as title,
		       COALESCE(message,'') as message, COALESCE(link_url,'') as link_url,
		       COALESCE(link_text,'') as link_text, COALESCE(priority,'') as priority,
		       created_at::text as created_at
		FROM notifications
		WHERE recipient_id = $1 AND (status IS NULL OR status != 'read')
		ORDER BY created_at DESC
		LIMIT 50
	`, recipientID)

	if err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		result = append(result, map[string]interface{}{
			"id":         r.ID,
			"category":   r.Category,
			"title":      r.Title,
			"message":    r.Message,
			"link_url":   r.LinkURL,
			"link_text":  r.LinkText,
			"priority":   r.Priority,
			"created_at": r.CreatedAt,
		})
	}
	return result, nil
}

// MarkAsRead marks a notification as read
func (s *NotificationService) MarkAsRead(ctx context.Context, notificationID string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE notifications SET status = 'read', read_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`, notificationID)
	return err
}

// Helper function for string replacement — kept for compatibility
func replaceAll(s, old, new string) string {
	return strings.ReplaceAll(s, old, new)
}
