package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"calendar-service/internal/hasura"
	"calendar-service/internal/middleware"
	"calendar-service/internal/services"
)

type NotificationPreferencesHandler struct {
	hasuraClient *hasura.Client
	auditService services.AuditService
	logger       *logrus.Entry
}

func NewNotificationPreferencesHandler(client *hasura.Client, audit services.AuditService, logger *logrus.Entry) *NotificationPreferencesHandler {
	return &NotificationPreferencesHandler{
		hasuraClient: client,
		auditService: audit,
		logger:       logger.WithField("component", "notification_prefs_handler"),
	}
}

type NotificationPreferences struct {
	EmailSyncComplete     bool   `json:"email_sync_complete"`
	EmailSyncFailed       bool   `json:"email_sync_failed"`
	EmailConflictDetected bool   `json:"email_conflict_detected"`
	EmailTokenExpiring    bool   `json:"email_token_expiring"`
	PushSyncComplete      bool   `json:"push_sync_complete"`
	PushSyncFailed        bool   `json:"push_sync_failed"`
	PushConflictDetected  bool   `json:"push_conflict_detected"`
	DigestFrequency       string `json:"digest_frequency"`
}

func (h *NotificationPreferencesHandler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	_, err := middleware.ExtractTenantIDFromContextStrict(ctx)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID, err := middleware.ExtractUserIDFromContextStrict(ctx)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	reqUserID := vars["user_id"]

	if reqUserID != userID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	query := `
	query GetNotificationPrefs($user_id: uuid!) {
		user_notification_settings(
			where: {user_id: {_eq: $user_id}},
			limit: 1
		) {
			email_sync_complete
			email_sync_failed
			email_conflict_detected
			email_token_expiring
			push_sync_complete
			push_sync_failed
			push_conflict_detected
			digest_frequency
		}
	}
	`

	type Result struct {
		Settings []NotificationPreferences `json:"user_notification_settings"`
	}

	var result Result
	if err := h.hasuraClient.QueryRaw(ctx, query, map[string]interface{}{"user_id": userID}, &result); err != nil {
		h.logger.WithError(err).Error("Failed to get notification preferences")
		http.Error(w, "Failed to get settings", http.StatusInternalServerError)
		return
	}

	if len(result.Settings) == 0 {
		// Default preferences
		defaultPrefs := NotificationPreferences{
			EmailSyncComplete:     true,
			EmailSyncFailed:       true,
			EmailConflictDetected: true,
			EmailTokenExpiring:    true,
			PushSyncComplete:      false,
			PushSyncFailed:        true,
			PushConflictDetected:  true,
			DigestFrequency:       "weekly",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(defaultPrefs)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result.Settings[0])
}

func (h *NotificationPreferencesHandler) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, err := middleware.ExtractTenantIDFromContextStrict(ctx)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID, err := middleware.ExtractUserIDFromContextStrict(ctx)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	reqUserID := vars["user_id"]

	if reqUserID != userID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var prefs NotificationPreferences
	if err := json.NewDecoder(r.Body).Decode(&prefs); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	mutation := `
	mutation UpsertNotificationPrefs($object: user_notification_settings_insert_input!) {
		insert_user_notification_settings_one(
			object: $object,
			on_conflict: {
				constraint: user_notification_settings_user_id_key,
				update_columns: [
					email_sync_complete, email_sync_failed, email_conflict_detected, email_token_expiring,
					push_sync_complete, push_sync_failed, push_conflict_detected, digest_frequency, updated_at
				]
			}
		) {
			id
		}
	}
	`

	object := map[string]interface{}{
		"user_id":                 userID,
		"tenant_id":               tenantID,
		"email_sync_complete":     prefs.EmailSyncComplete,
		"email_sync_failed":       prefs.EmailSyncFailed,
		"email_conflict_detected": prefs.EmailConflictDetected,
		"email_token_expiring":    prefs.EmailTokenExpiring,
		"push_sync_complete":      prefs.PushSyncComplete,
		"push_sync_failed":        prefs.PushSyncFailed,
		"push_conflict_detected":  prefs.PushConflictDetected,
		"digest_frequency":        prefs.DigestFrequency,
		"updated_at":              time.Now().Format(time.RFC3339),
	}

	var res interface{}
	if err := h.hasuraClient.QueryRaw(ctx, mutation, map[string]interface{}{"object": object}, &res); err != nil {
		h.logger.WithError(err).Error("Failed to update notification preferences")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	_ = h.auditService.Record(ctx, services.AuditEntry{
		TenantID:   tenantID,
		EntityType: "user_notification_settings",
		EntityID:   userID,
		Action:     "UPDATE",
		ChangedBy:  userID,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prefs)
}
