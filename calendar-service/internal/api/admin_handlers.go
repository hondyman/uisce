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

// AdminHandler handles admin API endpoints
type AdminHandler struct {
	hasuraClient   *hasura.Client
	healthHandlers *HealthHandlers
	auditService   services.AuditService
	logger         *logrus.Entry
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(hc *hasura.Client, hh *HealthHandlers, audit services.AuditService, logger *logrus.Entry) *AdminHandler {
	return &AdminHandler{
		hasuraClient:   hc,
		healthHandlers: hh,
		auditService:   audit,
		logger:         logger.WithField("handler", "admin"),
	}
}

// GetAdminStats returns admin statistics
// @Summary Get admin statistics
// @Tags admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/stats [get]
func (h *AdminHandler) GetAdminStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check admin role
	if !middleware.HasRole(ctx, "admin") {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Query various stats
	stats := make(map[string]interface{})

	// Total users
	var userCount struct {
		Count int `json:"count"`
	}
	h.hasuraClient.QueryRaw(ctx, "query { count: user_settings_aggregate { aggregate { count } } }", nil, &userCount)
	stats["total_users"] = userCount.Count

	// Active syncs
	var syncCount struct {
		Count int `json:"count"`
	}
	h.hasuraClient.QueryRaw(ctx, "query { count: google_calendar_connections_aggregate(where: {sync_enabled: {_eq: true}}) { aggregate { count } } }", nil, &syncCount)
	stats["active_syncs"] = syncCount.Count

	// Pending conflicts
	var conflictCount struct {
		Count int `json:"count"`
	}
	h.hasuraClient.QueryRaw(ctx, "query { count: sync_conflicts_aggregate(where: {resolution_status: {_eq: \"pending\"}}) { aggregate { count } } }", nil, &conflictCount)
	stats["pending_conflicts"] = conflictCount.Count

	// Errors in last 24h
	var errorCount struct {
		Count int `json:"count"`
	}
	h.hasuraClient.QueryRaw(ctx, "query { count: error_logs_aggregate(where: {created_at: {_gte: \"now() - interval '24 hours'\"}}) { aggregate { count } } }", nil, &errorCount)
	stats["errors_24h"] = errorCount.Count

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// ListUsers returns list of users
// @Summary List users
// @Tags admin
// @Produce json
// @Param page query int false "Page"
// @Param limit query int false "Limit"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/users [get]
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !middleware.HasRole(ctx, "admin") {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	page := 1
	limit := 20

	query := `
	query ListUsers($limit: Int!, $offset: Int!) {
		user_settings(
			limit: $limit,
			offset: $offset,
			order_by: {created_at: desc}
		) {
			user_id tenant_id display_name email
			sync_frequency auto_sync_enabled
			created_at updated_at
		}
		user_settings_aggregate {
			aggregate {
				count
			}
		}
	}
	`

	var result struct {
		Settings []struct {
			UserID          string    `json:"user_id"`
			TenantID        string    `json:"tenant_id"`
			DisplayName     string    `json:"display_name"`
			Email           string    `json:"email"`
			SyncFrequency   string    `json:"sync_frequency"`
			AutoSyncEnabled bool      `json:"auto_sync_enabled"`
			CreatedAt       time.Time `json:"created_at"`
			UpdatedAt       time.Time `json:"updated_at"`
		} `json:"user_settings"`
		Aggregate struct {
			Aggregate struct {
				Count int `json:"count"`
			} `json:"aggregate"`
		} `json:"user_settings_aggregate"`
	}

	if err := h.hasuraClient.QueryRaw(ctx, query, map[string]interface{}{
		"limit":  limit,
		"offset": (page - 1) * limit,
	}, &result); err != nil {
		http.Error(w, "Failed to list users", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"users": result.Settings,
		"total": result.Aggregate.Aggregate.Count,
	})
}

// UpdateUserRole updates user role
// @Summary Update user role
// @Tags admin
// @Accept json
// @Produce json
// @Param user_id path string true "User ID"
// @Param request body map[string]string true "Role"
// @Success 200 {object} map[string]string
// @Router /api/v1/admin/users/{user_id}/role [put]
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

	// Update user role in user_settings table
	mutation := `
	mutation UpdateUserRole($user_id: uuid!, $role: String!) {
		update_user_settings(
			where: {user_id: {_eq: $user_id}},
			_set: {role: $role}
		) {
			affected_rows
		}
	}
	`

	if err := h.hasuraClient.QueryRaw(ctx, mutation, map[string]interface{}{
		"user_id": userID,
		"role":    req.Role,
	}, nil); err != nil {
		h.logger.WithError(err).Error("Failed to update user role")
		http.Error(w, "Failed to update user role", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "updated",
	})
}

// DeleteUser deletes a user
// @Summary Delete user
// @Tags admin
// @Produce json
// @Param user_id path string true "User ID"
// @Success 200 {object} map[string]string
// @Router /api/v1/admin/users/{user_id} [delete]
func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !middleware.HasRole(ctx, "admin") {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	userID := mux.Vars(r)["user_id"]

	// Soft delete user by setting deleted_at
	mutation := `
	mutation DeleteUser($user_id: uuid!) {
		update_user_settings(
			where: {user_id: {_eq: $user_id}},
			_set: {deleted_at: "now()"}
		) {
			affected_rows
		}
	}
	`

	if err := h.hasuraClient.QueryRaw(ctx, mutation, map[string]interface{}{
		"user_id": userID,
	}, nil); err != nil {
		h.logger.WithError(err).Error("Failed to delete user")
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "deleted",
	})
}

// GetErrorLogs returns error logs
// @Summary Get error logs
// @Tags admin
// @Produce json
// @Param page query int false "Page"
// @Param limit query int false "Limit"
// @Param level query string false "Level"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/error-logs [get]
func (h *AdminHandler) GetErrorLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !middleware.HasRole(ctx, "admin") {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Query error logs from your error_logs table
	query := `
	query GetErrorLogs {
		error_logs(order_by: {created_at: desc}, limit: 100) {
			id level message component stack_trace created_at
		}
		error_logs_aggregate {
			aggregate { count }
		}
	}
	`

	var result struct {
		Logs []map[string]interface{} `json:"error_logs"`
		Agg  struct {
			Agg struct {
				Count int `json:"count"`
			} `json:"aggregate"`
		} `json:"error_logs_aggregate"`
	}

	if err := h.hasuraClient.QueryRaw(ctx, query, nil, &result); err != nil {
		h.logger.WithError(err).Error("Failed to get error logs")
		http.Error(w, "Failed to get error logs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"logs":  result.Logs,
		"total": result.Agg.Agg.Count,
	})
}

// GetSystemHealth returns system health status
// @Summary Get system health
// @Tags admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/health [get]
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

// GetAuditLogs returns audit logs
// @Summary Get audit logs
// @Tags admin
// @Produce json
// @Param page query int false "Page"
// @Param limit query int false "Limit"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/audit-logs [get]
func (h *AdminHandler) GetAuditLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !middleware.HasRole(ctx, "admin") {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Query audit logs from calendar.audit_log table
	query := `
	query GetAuditLogs {
		audit_logs(order_by: {created_at: desc}, limit: 100) {
			id tenant_id entity_type entity_id action changed_by created_at
		}
		audit_logs_aggregate {
			aggregate { count }
		}
	}
	`

	var result struct {
		Logs []map[string]interface{} `json:"audit_logs"`
		Agg  struct {
			Agg struct {
				Count int `json:"count"`
			} `json:"aggregate"`
		} `json:"audit_logs_aggregate"`
	}

	if err := h.hasuraClient.QueryRaw(ctx, query, nil, &result); err != nil {
		h.logger.WithError(err).Error("Failed to get audit logs")
		http.Error(w, "Failed to get audit logs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"logs":  result.Logs,
		"total": result.Agg.Agg.Count,
	})
}

// GetSyncStats returns detailed sync statistics
// @Summary Get sync statistics
// @Tags admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/sync-stats [get]
func (h *AdminHandler) GetSyncStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !middleware.HasRole(ctx, "admin") {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	query := `
	query GetSyncStats {
		google_sync_aggregated: google_calendar_connections_aggregate {
			aggregate {
				count
			}
			nodes {
				last_sync_status
			}
		}
		sync_errors: error_logs_aggregate(where: {component: {_eq: "sync_processor"}}) {
			aggregate { count }
		}
	}
	`

	var result struct {
		GoogleSync struct {
			Agg struct {
				Count int `json:"count"`
			} `json:"aggregate"`
			Nodes []struct {
				Status string `json:"last_sync_status"`
			} `json:"nodes"`
		} `json:"google_sync_aggregated"`
		SyncErrors struct {
			Agg struct {
				Count int `json:"count"`
			} `json:"aggregate"`
		} `json:"sync_errors"`
	}

	if err := h.hasuraClient.QueryRaw(ctx, query, nil, &result); err != nil {
		h.logger.WithError(err).Error("Failed to get sync stats")
		http.Error(w, "Failed to get sync stats", http.StatusInternalServerError)
		return
	}

	// Calculate breakdown
	successCount := 0
	failedCount := 0
	for _, n := range result.GoogleSync.Nodes {
		if n.Status == "success" || n.Status == "synced" {
			successCount++
		} else if n.Status == "failed" {
			failedCount++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_connections": result.GoogleSync.Agg.Count,
		"success_count":     successCount,
		"failed_count":      failedCount,
		"error_logs_count":  result.SyncErrors.Agg.Count,
		"timestamp":         time.Now().UTC(),
	})
}
