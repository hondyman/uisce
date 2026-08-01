package services

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"
)

type TranslationService struct {
	logger *zap.Logger
	cache  map[string]map[string]string
}

func NewTranslationService() *TranslationService {
	logger, _ := zap.NewProduction()
	return &TranslationService{
		logger: logger,
		cache:  make(map[string]map[string]string),
	}
}

func (s *TranslationService) GetTranslation(ctx context.Context, locale, namespace, key, defaultValue string) string {
	return defaultValue
}

func (s *TranslationService) GetTranslations(ctx context.Context, locale, namespace string) (map[string]string, error) {
	return nil, fmt.Errorf("GetTranslations: Hasura removed from TranslationService")
}

type AuditService struct {
	logger *zap.Logger
}

func NewAuditService() *AuditService {
	logger, _ := zap.NewProduction()
	return &AuditService{
		logger: logger,
	}
}

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

func (s *AuditService) LogAudit(ctx context.Context, entry AuditEntry) error {
	return fmt.Errorf("LogAudit: Hasura removed from AuditService")
}

func (s *AuditService) GetAuditHistory(ctx context.Context, entityType, entityID string, limit int) ([]map[string]interface{}, error) {
	return nil, fmt.Errorf("GetAuditHistory: Hasura removed from AuditService")
}

type NotificationService struct {
	logger *zap.Logger
}

func NewNotificationService() *NotificationService {
	logger, _ := zap.NewProduction()
	return &NotificationService{
		logger: logger,
	}
}

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

func (s *NotificationService) SendNotification(ctx context.Context, notif NotificationMessage) (string, error) {
	return "", fmt.Errorf("SendNotification: Hasura removed from NotificationService")
}

func (s *NotificationService) SendFromTemplate(ctx context.Context, templateKey, recipientID string, variables map[string]string) error {
	return fmt.Errorf("SendFromTemplate: Hasura removed from NotificationService")
}

func (s *NotificationService) GetUnreadNotifications(ctx context.Context, recipientID string) ([]map[string]interface{}, error) {
	return nil, fmt.Errorf("GetUnreadNotifications: Hasura removed from NotificationService")
}

func (s *NotificationService) MarkAsRead(ctx context.Context, notificationID string) error {
	return fmt.Errorf("MarkAsRead: Hasura removed from NotificationService")
}

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
