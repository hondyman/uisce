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

type AdminHandler struct {
	dbClient       *database.Client
	healthHandlers *HealthHandlers
	auditService   services.AuditService
	logger         *logrus.Entry
}

func NewAdminHandler(db *database.Client, hh *HealthHandlers, audit services.AuditService, logger *logrus.Entry) *AdminHandler {
	return &AdminHandler{
		dbClient:       db,
		healthHandlers: hh,
		auditService:   audit,
		logger:         logger.WithField("handler", "admin"),
	}
}

func (h *AdminHandler) GetAdminStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !middleware.HasRole(ctx, "admin") {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	stats := make(map[string]interface{})

	var userCount int
	h.dbClient.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM user_settings").Scan(&userCount)
	stats["total_users"] = userCount

	var syncCount int
	h.dbClient.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM google_calendar_connections WHERE sync_enabled = true").Scan(&syncCount)
	stats["active_syncs"] = syncCount

	var conflictCount int
	h.dbClient.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM sync_conflicts WHERE resolution_status = 'pending'").Scan(&conflictCount)
	stats["pending_conflicts"] = conflictCount

	var errorCount int
	h.dbClient.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM error_logs WHERE created_at >= NOW() - INTERVAL '24 hours'").Scan(&errorCount)
	stats["errors_24h"] = errorCount

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !middleware.HasRole(ctx, "admin") {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	page := 1
	limit := 20

	query := `
		SELECT user_id, tenant_id, display_name, email, sync_frequency, auto_sync_enabled, created_at, updated_at
		FROM user_settings
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	offset := (page - 1) * limit
	rows, err := h.dbClient.Pool().Query(ctx, query, limit, offset)
	if err != nil {
		http.Error(w, "Failed to list users", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []map[string]interface{}
	for rows.Next() {
		var userID, tenantID, displayName, email, syncFreq string
		var autoSync bool
		var createdAt, updatedAt string
		if err := rows.Scan(&userID, &tenantID, &displayName, &email, &syncFreq, &autoSync, &createdAt, &updatedAt); err != nil {
			continue
		}
		users = append(users, map[string]interface{}{
			"user_id":           userID,
			"tenant_id":         tenantID,
			"display_name":      displayName,
			"email":             email,
			"sync_frequency":    syncFreq,
			"auto_sync_enabled": autoSync,
			"created_at":        createdAt,
			"updated_at":        updatedAt,
		})
	}

	var total int
	h.dbClient.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM user_settings").Scan(&total)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"users": users,
		"total": total,
	})
}

func (h *AdminHandler) UpdateUserRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !middleware.HasRole(ctx, "admin") {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	userID := mux.Vars(r)["user_id"]

	var req struct {
		Role string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	query := `UPDATE user_settings SET role = $2 WHERE user_id = $1`

	_, err := h.dbClient.Pool().Exec(ctx, query, userID, req.Role)
	if err != nil {
		h.logger.WithError(err).Error("Failed to update user role")
		http.Error(w, "Failed to update user role", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "updated",
	})
}

func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !middleware.HasRole(ctx, "admin") {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	userID := mux.Vars(r)["user_id"]

	query := `UPDATE user_settings SET deleted_at = NOW() WHERE user_id = $1`

	_, err := h.dbClient.Pool().Exec(ctx, query, userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to delete user")
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "deleted",
	})
}

func (h *AdminHandler) GetErrorLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !middleware.HasRole(ctx, "admin") {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	query := `
		SELECT id, level, message, component, stack_trace, created_at
		FROM error_logs
		ORDER BY created_at DESC
		LIMIT 100
	`

	rows, err := h.dbClient.Pool().Query(ctx, query)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get error logs")
		http.Error(w, "Failed to get error logs", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var logs []map[string]interface{}
	for rows.Next() {
		var id, level, message, component, stackTrace, createdAt sql.NullString
		if err := rows.Scan(&id, &level, &message, &component, &stackTrace, &createdAt); err != nil {
			continue
		}
		logs = append(logs, map[string]interface{}{
			"id":          id.String,
			"level":       level.String,
			"message":     message.String,
			"component":   component.String,
			"stack_trace": stackTrace.String,
			"created_at":  createdAt.String,
		})
	}

	var total int
	h.dbClient.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM error_logs").Scan(&total)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"logs":  logs,
		"total": total,
	})
}

func (h *AdminHandler) GetSystemHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !middleware.HasRole(ctx, "admin") {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	health := h.healthHandlers.CheckReady(ctx)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func (h *AdminHandler) GetAuditLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !middleware.HasRole(ctx, "admin") {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	query := `
		SELECT id, tenant_id, entity_type, entity_id, action, changed_by, created_at
		FROM audit_logs
		ORDER BY created_at DESC
		LIMIT 100
	`

	rows, err := h.dbClient.Pool().Query(ctx, query)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get audit logs")
		http.Error(w, "Failed to get audit logs", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var logs []map[string]interface{}
	for rows.Next() {
		var id, tenantID, entityType, entityID, action, changedBy, createdAt string
		if err := rows.Scan(&id, &tenantID, &entityType, &entityID, &action, &changedBy, &createdAt); err != nil {
			continue
		}
		logs = append(logs, map[string]interface{}{
			"id":          id,
			"tenant_id":   tenantID,
			"entity_type": entityType,
			"entity_id":   entityID,
			"action":      action,
			"changed_by":  changedBy,
			"created_at":  createdAt,
		})
	}

	var total int
	h.dbClient.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM audit_logs").Scan(&total)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"logs":  logs,
		"total": total,
	})
}

func (h *AdminHandler) GetSyncStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !middleware.HasRole(ctx, "admin") {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var totalConnections int
	h.dbClient.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM google_calendar_connections").Scan(&totalConnections)

	var syncErrors int
	h.dbClient.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM error_logs WHERE component = 'sync_processor'").Scan(&syncErrors)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_connections": totalConnections,
		"error_logs_count":  syncErrors,
	})
}
