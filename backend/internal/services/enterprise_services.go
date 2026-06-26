package services

import (
	"context"
	"encoding/json"
	"fmt"

	hasuraclient "github.com/hondyman/semlayer/libs/hasura-client"
	"go.uber.org/zap"
)

// ============================================================================
// TRANSLATION SERVICE (i18n)
// ============================================================================

// TranslationService provides multilingual support
type TranslationService struct {
	hasuraClient *hasuraclient.HasuraClient
	logger       *zap.Logger
	cache        map[string]map[string]string // locale -> namespace.key -> value
}

// NewTranslationService creates a new translation service
func NewTranslationService(hasuraClient *hasuraclient.HasuraClient) *TranslationService {
	logger, _ := zap.NewProduction()
	return &TranslationService{
		hasuraClient: hasuraClient,
		logger:       logger,
		cache:        make(map[string]map[string]string),
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

	// Query from database
	query := `
		query GetTranslation($locale: String!, $namespace: String!, $key: String!) {
			translations(
				where: {
					locale_code: { _eq: $locale },
					namespace: { _eq: $namespace },
					key: { _eq: $key }
				},
				limit: 1
			) {
				value
			}
		}
	`

	result, err := s.hasuraClient.Query(query, map[string]interface{}{
		"locale":    locale,
		"namespace": namespace,
		"key":       key,
	})

	if err != nil || result == nil {
		return defaultValue
	}

	translations, ok := result["translations"].([]interface{})
	if !ok || len(translations) == 0 {
		return defaultValue
	}

	data := translations[0].(map[string]interface{})
	if value, ok := data["value"].(string); ok {
		// Update cache
		if s.cache[locale] == nil {
			s.cache[locale] = make(map[string]string)
		}
		s.cache[locale][cacheKey] = value
		return value
	}

	return defaultValue
}

// GetTranslations fetches multiple translations for a namespace
func (s *TranslationService) GetTranslations(ctx context.Context, locale, namespace string) (map[string]string, error) {
	query := `
		query GetTranslations($locale: String!, $namespace: String!) {
			translations(
				where: {
					locale_code: { _eq: $locale },
					namespace: { _eq: $namespace }
				}
			) {
				key
				value
			}
		}
	`

	result, err := s.hasuraClient.Query(query, map[string]interface{}{
		"locale":    locale,
		"namespace": namespace,
	})

	if err != nil {
		return nil, err
	}

	translations := make(map[string]string)
	items, ok := result["translations"].([]interface{})
	if !ok {
		return translations, nil
	}

	for _, item := range items {
		data := item.(map[string]interface{})
		key := getString(data, "key")
		value := getString(data, "value")
		translations[key] = value
	}

	return translations, nil
}

// ============================================================================
// AUDIT SERVICE
// ============================================================================

// AuditService provides comprehensive audit logging
type AuditService struct {
	hasuraClient *hasuraclient.HasuraClient
	logger       *zap.Logger
}

// NewAuditService creates a new audit service
func NewAuditService(hasuraClient *hasuraclient.HasuraClient) *AuditService {
	logger, _ := zap.NewProduction()
	return &AuditService{
		hasuraClient: hasuraClient,
		logger:       logger,
	}
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

	mutation := `
		mutation InsertAuditLog($object: audit_log_insert_input!) {
			insert_audit_log_one(object: $object) {
				id
			}
		}
	`

	object := map[string]interface{}{
		"entity_type": entry.EntityType,
		"entity_id":   entry.EntityID,
		"action":      entry.Action,
		"actor":       entry.Actor,
	}

	if entry.EntityName != "" {
		object["entity_name"] = entry.EntityName
	}
	if entry.ActorType != "" {
		object["actor_type"] = entry.ActorType
	}
	if entry.IPAddress != "" {
		object["ip_address"] = entry.IPAddress
	}
	if entry.UserAgent != "" {
		object["user_agent"] = entry.UserAgent
	}
	if len(entry.Changes) > 0 {
		object["changes"] = string(changesJSON)
	}
	if len(entry.Metadata) > 0 {
		object["metadata"] = string(metadataJSON)
	}
	if entry.Severity != "" {
		object["severity"] = entry.Severity
	}

	_, err := s.hasuraClient.Mutate(mutation, map[string]interface{}{
		"object": object,
	})

	if err != nil {
		s.logger.Error("Failed to create audit log", zap.Error(err))
		return err
	}

	return nil
}

// GetAuditHistory fetches audit history for an entity
func (s *AuditService) GetAuditHistory(ctx context.Context, entityType, entityID string, limit int) ([]map[string]interface{}, error) {
	query := `
		query GetAuditHistory($entityType: String!, $entityID: String!, $limit: Int!) {
			audit_log(
				where: { entity_type: { _eq: $entityType }, entity_id: { _eq: $entityID } }
				order_by: { created_at: desc }
				limit: $limit
			) {
				id
				action
				actor
				actor_type
				changes
				metadata
				created_at
			}
		}
	`

	result, err := s.hasuraClient.Query(query, map[string]interface{}{
		"entityType": entityType,
		"entityID":   entityID,
		"limit":      limit,
	})

	if err != nil {
		return nil, err
	}

	logs, ok := result["audit_log"].([]interface{})
	if !ok {
		return []map[string]interface{}{}, nil
	}

	history := make([]map[string]interface{}, 0, len(logs))
	for _, log := range logs {
		history = append(history, log.(map[string]interface{}))
	}

	return history, nil
}

// ============================================================================
// NOTIFICATION SERVICE
// ============================================================================

// NotificationService provides notification management
type NotificationService struct {
	hasuraClient *hasuraclient.HasuraClient
	logger       *zap.Logger
}

// NewNotificationService creates a new notification service
func NewNotificationService(hasuraClient *hasuraclient.HasuraClient) *NotificationService {
	logger, _ := zap.NewProduction()
	return &NotificationService{
		hasuraClient: hasuraClient,
		logger:       logger,
	}
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

	mutation := `
		mutation InsertNotification($object: notifications_insert_input!) {
			insert_notifications_one(object: $object) {
				id
			}
		}
	`

	object := map[string]interface{}{
		"recipient_id": notif.RecipientID,
		"channel":      notif.Channel,
		"category":     notif.Category,
		"title":        notif.Title,
		"message":      notif.Message,
	}

	if notif.RecipientType != "" {
		object["recipient_type"] = notif.RecipientType
	}
	if notif.Priority != "" {
		object["priority"] = notif.Priority
	}
	if len(notif.Data) > 0 {
		object["data"] = string(dataJSON)
	}
	if notif.LinkURL != "" {
		object["link_url"] = notif.LinkURL
	}
	if notif.LinkText != "" {
		object["link_text"] = notif.LinkText
	}

	result, err := s.hasuraClient.Mutate(mutation, map[string]interface{}{
		"object": object,
	})

	if err != nil {
		return "", err
	}

	data := result["insert_notifications_one"].(map[string]interface{})
	return data["id"].(string), nil
}

// SendFromTemplate sends a notification using a template
func (s *NotificationService) SendFromTemplate(ctx context.Context, templateKey, recipientID string, variables map[string]string) error {
	// Get template
	query := `
		query GetTemplate($key: String!) {
			notification_templates(where: { key: { _eq: $key } }, limit: 1) {
				subject_template
				body_template
				category
				channels
			}
		}
	`

	result, err := s.hasuraClient.Query(query, map[string]interface{}{"key": templateKey})
	if err != nil {
		return err
	}

	templates, ok := result["notification_templates"].([]interface{})
	if !ok || len(templates) == 0 {
		return fmt.Errorf("template not found: %s", templateKey)
	}

	template := templates[0].(map[string]interface{})
	subject := getString(template, "subject_template")
	body := getString(template, "body_template")
	category := getString(template, "category")

	// Replace variables
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		subject = replaceAll(subject, placeholder, value)
		body = replaceAll(body, placeholder, value)
	}

	// Send notification
	channels, _ := template["channels"].([]interface{})
	for _, ch := range channels {
		channel := ch.(string)
		_, err := s.SendNotification(ctx, NotificationMessage{
			RecipientID: recipientID,
			Channel:     channel,
			Category:    category,
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
	query := `
		query GetUnreadNotifications($recipientID: String!) {
			notifications(
				where: {
					recipient_id: { _eq: $recipientID },
					status: { _nin: ["read"] }
				}
				order_by: { created_at: desc }
				limit: 50
			) {
				id
				category
				title
				message
				link_url
				link_text
				priority
				created_at
			}
		}
	`

	result, err := s.hasuraClient.Query(query, map[string]interface{}{
		"recipientID": recipientID,
	})

	if err != nil {
		return nil, err
	}

	notifs, ok := result["notifications"].([]interface{})
	if !ok {
		return []map[string]interface{}{}, nil
	}

	notifications := make([]map[string]interface{}, 0, len(notifs))
	for _, n := range notifs {
		notifications = append(notifications, n.(map[string]interface{}))
	}

	return notifications, nil
}

// MarkAsRead marks a notification as read
func (s *NotificationService) MarkAsRead(ctx context.Context, notificationID string) error {
	mutation := `
		mutation MarkAsRead($id: String!) {
			update_notifications_by_pk(
				pk_columns: { id: $id }
				_set: { status: "read", read_at: "now()" }
			) {
				id
			}
		}
	`

	_, err := s.hasuraClient.Mutate(mutation, map[string]interface{}{
		"id": notificationID,
	})

	return err
}

// Helper function for string replacement (already defined in other services)
func replaceAll(s, old, new string) string {
	result := s
	for {
		i := -1
		for j := 0; j <= len(result)-len(old); j++ {
			if result[j:j+len(old)] == old {
				i = j
				break
			}
		}
		if i < 0 {
			break
		}
		result = result[:i] + new + result[i+len(old):]
	}
	return result
}
