package api

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"calendar-service/internal/database"
	"calendar-service/internal/middleware"
	"calendar-service/internal/services"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type SettingsHandler struct {
	dbClient     *database.Client
	auditService services.AuditService
	logger       *logrus.Entry
}

func NewSettingsHandler(db *database.Client, audit services.AuditService, logger *logrus.Entry) *SettingsHandler {
	return &SettingsHandler{
		dbClient:     db,
		auditService: audit,
		logger:       logger.WithField("handler", "settings"),
	}
}

type UserSettings struct {
	UserID                   string `json:"user_id"`
	TenantID                 string `json:"tenant_id"`
	DisplayName              string `json:"display_name"`
	Email                    string `json:"email"`
	AvatarURL                string `json:"avatar_url,omitempty"`
	Timezone                 string `json:"timezone"`
	Language                 string `json:"language"`
	SyncFrequency            string `json:"sync_frequency"`
	AutoSyncEnabled          bool   `json:"auto_sync_enabled"`
	DefaultCalendarID        string `json:"default_calendar_id,omitempty"`
	SyncConflictsAutoResolve bool   `json:"sync_conflicts_auto_resolve"`
	SyncConflictsStrategy    string `json:"sync_conflicts_strategy"`
	EmailNotifications       bool   `json:"email_notifications"`
	PushNotifications        bool   `json:"push_notifications"`
	SyncCompleteNotification bool   `json:"sync_complete_notification"`
	ConflictNotification     bool   `json:"conflict_notification"`
	ErrorNotification        bool   `json:"error_notification"`
	DataRetentionDays        int    `json:"data_retention_days"`
	ShareAnalytics           bool   `json:"share_analytics"`
	CreatedAt                string `json:"created_at"`
	UpdatedAt                string `json:"updated_at"`
}

func (h *SettingsHandler) GetUserSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, err := middleware.ExtractTenantIDFromContextStrict(ctx)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	userID := vars["user_id"]

	query := `
		SELECT user_id, tenant_id, display_name, email, avatar_url, timezone, language,
			sync_frequency, auto_sync_enabled, default_calendar_id,
			sync_conflicts_auto_resolve, sync_conflicts_strategy,
			email_notifications, push_notifications,
			sync_complete_notification, conflict_notification, error_notification,
			data_retention_days, share_analytics,
			created_at, updated_at
		FROM user_settings
		WHERE user_id = $1
	`

	var settings UserSettings
	var displayName, email, avatarURL, timezone, language, syncFreq, defaultCalID, syncStrategy sql.NullString
	var autoSync, syncAutoResolve, emailNotif, pushNotif, syncComplete, conflictNotif, errorNotif, shareAnalytics sql.NullBool
	var dataRetention sql.NullInt64
	var createdAt, updatedAt sql.NullString

	err = h.dbClient.Pool().QueryRow(ctx, query, userID).Scan(
		&settings.UserID, &settings.TenantID, &displayName, &email, &avatarURL, &timezone, &language,
		&syncFreq, &autoSync, &defaultCalID,
		&syncAutoResolve, &syncStrategy,
		&emailNotif, &pushNotif,
		&syncComplete, &conflictNotif, &errorNotif,
		&dataRetention, &shareAnalytics,
		&createdAt, &updatedAt,
	)

	if err == sql.ErrNoRows {
		defaultSettings := UserSettings{
			UserID:                   userID,
			TenantID:                 tenantID,
			Timezone:                 "UTC",
			Language:                 "en",
			SyncFrequency:            "hourly",
			AutoSyncEnabled:          true,
			SyncConflictsAutoResolve: false,
			SyncConflictsStrategy:    "manual",
			EmailNotifications:       true,
			PushNotifications:        false,
			SyncCompleteNotification: true,
			ConflictNotification:     true,
			ErrorNotification:        true,
			DataRetentionDays:        365,
			ShareAnalytics:           false,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(defaultSettings)
		return
	}

	if err != nil {
		h.logger.WithError(err).Error("Failed to get user settings")
		http.Error(w, "Failed to get settings", http.StatusInternalServerError)
		return
	}

	settings.DisplayName = displayName.String
	settings.Email = email.String
	settings.AvatarURL = avatarURL.String
	settings.Timezone = timezone.String
	settings.Language = language.String
	settings.SyncFrequency = syncFreq.String
	settings.AutoSyncEnabled = autoSync.Bool
	settings.DefaultCalendarID = defaultCalID.String
	settings.SyncConflictsAutoResolve = syncAutoResolve.Bool
	settings.SyncConflictsStrategy = syncStrategy.String
	settings.EmailNotifications = emailNotif.Bool
	settings.PushNotifications = pushNotif.Bool
	settings.SyncCompleteNotification = syncComplete.Bool
	settings.ConflictNotification = conflictNotif.Bool
	settings.ErrorNotification = errorNotif.Bool
	settings.DataRetentionDays = int(dataRetention.Int64)
	settings.ShareAnalytics = shareAnalytics.Bool
	settings.CreatedAt = createdAt.String
	settings.UpdatedAt = updatedAt.String

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}

func (h *SettingsHandler) UpdateUserSettings(w http.ResponseWriter, r *http.Request) {
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
	requestUserID := vars["user_id"]

	if userID != requestUserID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var settings UserSettings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	query := `
		UPDATE user_settings SET
			display_name = $2, email = $3, avatar_url = $4, timezone = $5, language = $6,
			sync_frequency = $7, auto_sync_enabled = $8, default_calendar_id = $9,
			sync_conflicts_auto_resolve = $10, sync_conflicts_strategy = $11,
			email_notifications = $12, push_notifications = $13,
			sync_complete_notification = $14, conflict_notification = $15, error_notification = $16,
			data_retention_days = $17, share_analytics = $18,
			updated_at = NOW()
		WHERE user_id = $1
		RETURNING updated_at
	`

	var updatedAt string
	err = h.dbClient.Pool().QueryRow(ctx, query,
		userID, settings.DisplayName, settings.Email, settings.AvatarURL, settings.Timezone, settings.Language,
		settings.SyncFrequency, settings.AutoSyncEnabled, settings.DefaultCalendarID,
		settings.SyncConflictsAutoResolve, settings.SyncConflictsStrategy,
		settings.EmailNotifications, settings.PushNotifications,
		settings.SyncCompleteNotification, settings.ConflictNotification, settings.ErrorNotification,
		settings.DataRetentionDays, settings.ShareAnalytics,
	).Scan(&updatedAt)

	if err != nil {
		h.logger.WithError(err).Error("Failed to update user settings")
		http.Error(w, "Failed to update settings", http.StatusInternalServerError)
		return
	}

	_ = h.auditService.Record(ctx, services.AuditEntry{
		TenantID:   tenantID,
		EntityType: "user_settings",
		EntityID:   userID,
		Action:     "UPDATE",
		ChangedBy:  userID,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id":    userID,
		"updated_at": updatedAt,
	})
}

func (h *SettingsHandler) GetConnectedAccounts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id required", http.StatusBadRequest)
		return
	}

	query := `
		SELECT id, user_id, google_email, sync_enabled,
			last_sync_at, next_sync_at, last_sync_status,
			mapped_calendars, created_at
		FROM google_calendar_connections
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := h.dbClient.Pool().Query(ctx, query, userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get connected accounts")
		http.Error(w, "Failed to get connected accounts", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type ConnectedAccount struct {
		ID             string `json:"id"`
		UserID         string `json:"user_id"`
		GoogleEmail    string `json:"google_email"`
		SyncEnabled    bool   `json:"sync_enabled"`
		LastSyncStatus string `json:"last_sync_status"`
		CreatedAt      string `json:"created_at"`
	}

	var accounts []map[string]interface{}
	for rows.Next() {
		var id, uid, email, status, createdAt string
		var syncEnabled bool
		var mappedCalendars []byte
		if err := rows.Scan(&id, &uid, &email, &syncEnabled, nil, nil, &status, &mappedCalendars, &createdAt); err != nil {
			continue
		}

		accStatus := "active"
		if status == "failed" {
			accStatus = "error"
		}
		if !syncEnabled {
			accStatus = "disconnected"
		}

		accounts = append(accounts, map[string]interface{}{
			"id":       id,
			"provider": "google",
			"email":    email,
			"status":   accStatus,
			"scopes":   []string{"calendar.readonly", "calendar.events"},
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"accounts": accounts,
	})
}

func (h *SettingsHandler) DisconnectAccount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, err := middleware.ExtractTenantIDFromContextStrict(ctx)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	accountID := vars["account_id"]

	query := `DELETE FROM google_calendar_connections WHERE id = $1`

	_, err = h.dbClient.Pool().Exec(ctx, query, accountID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to disconnect account")
		http.Error(w, "Failed to disconnect account", http.StatusInternalServerError)
		return
	}

	userID, _ := middleware.ExtractUserIDFromContextStrict(ctx)
	_ = h.auditService.Record(ctx, services.AuditEntry{
		TenantID:   tenantID,
		EntityType: "google_calendar_connection",
		EntityID:   accountID,
		Action:     "DELETE",
		ChangedBy:  userID,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "disconnected",
	})
}

func (h *SettingsHandler) GetUserSettingsWithContext(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.ExtractUserIDFromContextStrict(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := map[string]string{"user_id": userID}
	r = mux.SetURLVars(r, vars)
	h.GetUserSettings(w, r)
}

func (h *SettingsHandler) UpdateUserSettingsWithContext(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.ExtractUserIDFromContextStrict(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := map[string]string{"user_id": userID}
	r = mux.SetURLVars(r, vars)
	h.UpdateUserSettings(w, r)
}
