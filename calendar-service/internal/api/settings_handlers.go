package api

import (
	"encoding/json"
	"net/http"
	"time"

	"calendar-service/internal/hasura"
	"calendar-service/internal/middleware"
	"calendar-service/internal/services"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// SettingsHandler handles user settings API endpoints
type SettingsHandler struct {
	hasuraClient *hasura.Client
	auditService services.AuditService
	logger       *logrus.Entry
}

// NewSettingsHandler creates a new settings handler
func NewSettingsHandler(hc *hasura.Client, audit services.AuditService, logger *logrus.Entry) *SettingsHandler {
	return &SettingsHandler{
		hasuraClient: hc,
		auditService: audit,
		logger:       logger.WithField("handler", "settings"),
	}
}

// UserSettings represents user settings payload
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
}

// GetUserSettings returns user settings
// @Summary Get user settings
// @Tags settings
// @Produce json
// @Param user_id path string true "User ID"
// @Success 200 {object} UserSettings
// @Router /api/v1/settings/user/{user_id} [get]
func (h *SettingsHandler) GetUserSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract tenant from JWT
	tenantID, err := middleware.ExtractTenantIDFromContextStrict(ctx)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	userID := vars["user_id"]

	query := `
	query GetUserSettings($user_id: uuid!) {
		user_settings(
			where: {user_id: {_eq: $user_id}},
			limit: 1
		) {
			user_id tenant_id display_name email avatar_url timezone language
			sync_frequency auto_sync_enabled default_calendar_id
			sync_conflicts_auto_resolve sync_conflicts_strategy
			email_notifications push_notifications
			sync_complete_notification conflict_notification error_notification
			data_retention_days share_analytics
			created_at updated_at
		}
	}
	`

	var result struct {
		Settings []UserSettings `json:"user_settings"`
	}

	if err := h.hasuraClient.QueryRaw(ctx, query, map[string]interface{}{"user_id": userID}, &result); err != nil {
		h.logger.WithError(err).Error("Failed to get user settings")
		http.Error(w, "Failed to get settings", http.StatusInternalServerError)
		return
	}

	if len(result.Settings) == 0 {
		// Return default settings
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result.Settings[0])
}

// UpdateUserSettings updates user settings
// @Summary Update user settings
// @Tags settings
// @Accept json
// @Produce json
// @Param user_id path string true "User ID"
// @Param settings body UserSettings true "Settings"
// @Success 200 {object} UserSettings
// @Router /api/v1/settings/user/{user_id} [put]
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

	// Verify user can only update their own settings
	if userID != requestUserID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var settings UserSettings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update settings
	mutation := `
	mutation UpdateUserSettings($user_id: uuid!, $settings: user_settings_set_input!) {
		update_user_settings(
			where: {user_id: {_eq: $user_id}},
			_set: $settings
		) {
			affected_rows
			returning {
				user_id updated_at
			}
		}
	}
	`

	updateData := map[string]interface{}{}
	if settings.DisplayName != "" {
		updateData["display_name"] = settings.DisplayName
	}
	if settings.Email != "" {
		updateData["email"] = settings.Email
	}
	if settings.AvatarURL != "" {
		updateData["avatar_url"] = settings.AvatarURL
	}
	if settings.Timezone != "" {
		updateData["timezone"] = settings.Timezone
	}
	if settings.Language != "" {
		updateData["language"] = settings.Language
	}
	if settings.SyncFrequency != "" {
		updateData["sync_frequency"] = settings.SyncFrequency
	}
	updateData["auto_sync_enabled"] = settings.AutoSyncEnabled
	updateData["sync_conflicts_auto_resolve"] = settings.SyncConflictsAutoResolve
	if settings.SyncConflictsStrategy != "" {
		updateData["sync_conflicts_strategy"] = settings.SyncConflictsStrategy
	}
	updateData["email_notifications"] = settings.EmailNotifications
	updateData["push_notifications"] = settings.PushNotifications
	updateData["sync_complete_notification"] = settings.SyncCompleteNotification
	updateData["conflict_notification"] = settings.ConflictNotification
	updateData["error_notification"] = settings.ErrorNotification
	updateData["data_retention_days"] = settings.DataRetentionDays
	updateData["share_analytics"] = settings.ShareAnalytics

	var result struct {
		Update struct {
			AffectedRows int `json:"affected_rows"`
			Returning    []struct {
				UserID    string    `json:"user_id"`
				UpdatedAt time.Time `json:"updated_at"`
			} `json:"returning"`
		} `json:"update_user_settings"`
	}

	if err := h.hasuraClient.QueryRaw(ctx, mutation, map[string]interface{}{
		"user_id":  userID,
		"settings": updateData,
	}, &result); err != nil {
		h.logger.WithError(err).Error("Failed to update user settings")
		return
	}

	// Audit log
	_ = h.auditService.Record(ctx, services.AuditEntry{
		TenantID:   tenantID,
		EntityType: "user_settings",
		EntityID:   userID,
		Action:     "UPDATE",
		ChangedBy:  userID,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result.Update.Returning[0])
}

// GetConnectedAccounts returns user's connected accounts
// @Summary Get connected accounts
// @Tags settings
// @Produce json
// @Param user_id query string true "User ID"
// @Success 200 {array} ConnectedAccount
// @Router /api/v1/settings/connected-accounts [get]
func (h *SettingsHandler) GetConnectedAccounts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id required", http.StatusBadRequest)
		return
	}

	query := `
	query GetConnectedAccounts($user_id: uuid!) {
		google_calendar_connections(
			where: {user_id: {_eq: $user_id}},
			order_by: {created_at: desc}
		) {
			id user_id google_email sync_enabled
			last_sync_at next_sync_at last_sync_status
			mapped_calendars created_at
		}
	}
	`

	var result struct {
		Connections []struct {
			ID              string                   `json:"id"`
			UserID          string                   `json:"user_id"`
			GoogleEmail     string                   `json:"google_email"`
			SyncEnabled     bool                     `json:"sync_enabled"`
			LastSyncAt      *time.Time               `json:"last_sync_at"`
			NextSyncAt      *time.Time               `json:"next_sync_at"`
			LastSyncStatus  string                   `json:"last_sync_status"`
			MappedCalendars []map[string]interface{} `json:"mapped_calendars"`
			CreatedAt       time.Time                `json:"created_at"`
		} `json:"google_calendar_connections"`
	}

	if err := h.hasuraClient.QueryRaw(ctx, query, map[string]interface{}{"user_id": userID}, &result); err != nil {
		h.logger.WithError(err).Error("Failed to get connected accounts")
		http.Error(w, "Failed to get connected accounts", http.StatusInternalServerError)
		return
	}

	// Transform to ConnectedAccount format
	accounts := make([]map[string]interface{}, 0, len(result.Connections))
	for _, conn := range result.Connections {
		status := "active"
		if conn.LastSyncStatus == "failed" {
			status = "error"
		}
		if !conn.SyncEnabled {
			status = "disconnected"
		}

		accounts = append(accounts, map[string]interface{}{
			"id":             conn.ID,
			"provider":       "google",
			"email":          conn.GoogleEmail,
			"connected_at":   conn.CreatedAt,
			"last_synced_at": conn.LastSyncAt,
			"status":         status,
			"scopes":         []string{"calendar.readonly", "calendar.events"},
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"accounts": accounts,
	})
}

// DisconnectAccount disconnects a connected account
// @Summary Disconnect account
// @Tags settings
// @Produce json
// @Param account_id path string true "Account ID"
// @Success 200 {object} map[string]string
// @Router /api/v1/settings/connected-accounts/{account_id} [delete]
func (h *SettingsHandler) DisconnectAccount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, err := middleware.ExtractTenantIDFromContextStrict(ctx)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	accountID := vars["account_id"]

	// Delete connection
	mutation := `
	mutation DeleteConnection($id: uuid!) {
		delete_google_calendar_connections_by_pk(id: $id) {
			id
		}
	}
	`

	var result struct {
		Delete struct {
			ID string `json:"id"`
		} `json:"delete_google_calendar_connections_by_pk"`
	}

	if err := h.hasuraClient.QueryRaw(ctx, mutation, map[string]interface{}{"id": accountID}, &result); err != nil {
		h.logger.WithError(err).Error("Failed to disconnect account")
		http.Error(w, "Failed to disconnect account", http.StatusInternalServerError)
		return
	}

	// Audit log
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

// GetUserSettingsWithContext returns settings for the current user in context
func (h *SettingsHandler) GetUserSettingsWithContext(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.ExtractUserIDFromContextStrict(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Re-use existing logic
	vars := map[string]string{"user_id": userID}
	r = mux.SetURLVars(r, vars)
	h.GetUserSettings(w, r)
}

// UpdateUserSettingsWithContext updates settings for the current user in context
func (h *SettingsHandler) UpdateUserSettingsWithContext(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.ExtractUserIDFromContextStrict(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Re-use existing logic
	vars := map[string]string{"user_id": userID}
	r = mux.SetURLVars(r, vars)
	h.UpdateUserSettings(w, r)
}
