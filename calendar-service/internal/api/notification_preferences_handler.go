package api

import (
	"encoding/json"
	"net/http"
	"time"

	"calendar-service/internal/database"
	"calendar-service/internal/middleware"
	"calendar-service/internal/services"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type NotificationPreferencesHandler struct {
	dbClient     *database.Client
	auditService services.AuditService
	logger       *logrus.Entry
}

func NewNotificationPreferencesHandler(db *database.Client, audit services.AuditService, logger *logrus.Entry) *NotificationPreferencesHandler {
	return &NotificationPreferencesHandler{
		dbClient:     db,
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
		SELECT email_sync_complete, email_sync_failed, email_conflict_detected, email_token_expiring,
			push_sync_complete, push_sync_failed, push_conflict_detected, digest_frequency
		FROM user_notification_settings
		WHERE user_id = $1
	`

	var prefs NotificationPreferences
	var emailSyncComplete, emailSyncFailed, emailConflict, emailToken, pushSyncComplete, pushSyncFailed, pushConflict bool
	var digestFreq string

	err = h.dbClient.Pool().QueryRow(ctx, query, userID).Scan(
		&emailSyncComplete, &emailSyncFailed, &emailConflict, &emailToken,
		&pushSyncComplete, &pushSyncFailed, &pushConflict, &digestFreq,
	)

	if err != nil {
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

	prefs.EmailSyncComplete = emailSyncComplete
	prefs.EmailSyncFailed = emailSyncFailed
	prefs.EmailConflictDetected = emailConflict
	prefs.EmailTokenExpiring = emailToken
	prefs.PushSyncComplete = pushSyncComplete
	prefs.PushSyncFailed = pushSyncFailed
	prefs.PushConflictDetected = pushConflict
	prefs.DigestFrequency = digestFreq

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prefs)
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

	query := `
		INSERT INTO user_notification_settings (user_id, tenant_id, email_sync_complete, email_sync_failed,
			email_conflict_detected, email_token_expiring, push_sync_complete, push_sync_failed,
			push_conflict_detected, digest_frequency, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (user_id) DO UPDATE SET
			email_sync_complete = EXCLUDED.email_sync_complete,
			email_sync_failed = EXCLUDED.email_sync_failed,
			email_conflict_detected = EXCLUDED.email_conflict_detected,
			email_token_expiring = EXCLUDED.email_token_expiring,
			push_sync_complete = EXCLUDED.push_sync_complete,
			push_sync_failed = EXCLUDED.push_sync_failed,
			push_conflict_detected = EXCLUDED.push_conflict_detected,
			digest_frequency = EXCLUDED.digest_frequency,
			updated_at = EXCLUDED.updated_at
	`

	_, err = h.dbClient.Pool().Exec(ctx, query,
		userID, tenantID, prefs.EmailSyncComplete, prefs.EmailSyncFailed,
		prefs.EmailConflictDetected, prefs.EmailTokenExpiring,
		prefs.PushSyncComplete, prefs.PushSyncFailed,
		prefs.PushConflictDetected, prefs.DigestFrequency, time.Now(),
	)

	if err != nil {
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
