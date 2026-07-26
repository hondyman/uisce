// Package services - Notifications domain facade shim.
//
// This file re-exports types from the new internal/notifications domain package
// so existing services-package consumers (notification_middleware.go,
// notification_handlers.go, trigger_adapters.go) can gradually migrate to
// direct internal/notifications imports without breaking the monolithic package.
//
// Cardinal Rule 3 (no cycles): This shim only depends on internal/notifications,
// which itself does NOT depend on internal/services. Zero back-coupling.
//
// Phase 5 of microservice extraction: existing code keeps using the
// services-package types while new code can import internal/notifications directly.
package services

import (
	"database/sql"

	"github.com/google/uuid"

	"github.com/hondyman/uisce/backend/internal/notifications"
)

// ============================================================================
// NOTIFICATIONS DOMAIN TYPE ALIASES
// ============================================================================

type (
	EngagementNotificationService  = notifications.EngagementNotificationService
	NotificationCampaignService    = notifications.NotificationCampaignService
	NotificationCampaignType       = notifications.NotificationCampaign
	EngagementRuleType             = notifications.EngagementRule
	NotificationEventType          = notifications.NotificationEvent
)

// ============================================================================
// CONSTRUCTOR WRAPPERS
// ============================================================================

// NewEngagementNotificationService delegates to internal/notifications.
func NewEngagementNotificationService(db *sql.DB) *notifications.EngagementNotificationService {
	return notifications.NewEngagementNotificationService(db)
}

// NewNotificationCampaignService delegates to internal/notifications.
func NewNotificationCampaignService(db *sql.DB, notifSvc *notifications.EngagementNotificationService) *notifications.NotificationCampaignService {
	return notifications.NewNotificationCampaignService(db, notifSvc)
}

// NewEmailService delegates to internal/notifications.
// Kept here for backward compat with the original services package API.
func NewEmailService(apiKey, fromEmail, fromName string) *notifications.EmailService {
	return notifications.NewEmailService(apiKey, fromEmail, fromName)
}

// NewSMSService delegates to internal/notifications.
func NewSMSService(accountSID, authToken, fromNumber string) *notifications.SMSService {
	return notifications.NewSMSService(accountSID, authToken, fromNumber)
}

// NewPusherService delegates to internal/notifications.
func NewPusherService(appID, key, secret, cluster string) *notifications.PusherService {
	return notifications.NewPusherService(appID, key, secret, cluster)
}

// Suppress unused warnings for uuid (used by callers, not here).
var _ = uuid.Nil