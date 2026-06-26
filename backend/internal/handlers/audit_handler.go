package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/hondyman/semlayer/backend/internal/logging"
)

// AuditLogEntry represents an audit log record
type AuditLogEntry struct {
	ID           string                 `json:"id"`
	TenantID     string                 `json:"tenantId"`
	Timestamp    time.Time              `json:"timestamp"`
	UserName     string                 `json:"userName"`
	UserEmail    string                 `json:"userEmail"`
	Action       string                 `json:"action"`
	Resource     string                 `json:"resource"`
	ResourceType string                 `json:"resourceType"`
	Details      map[string]interface{} `json:"details"`
}

// AuditLogResponse is the API response structure
type AuditLogResponse struct {
	Entries []AuditLogEntry `json:"entries"`
	Total   int             `json:"total"`
}

// HandleGetAuditLogs returns audit log entries for a tenant/datasource
func HandleGetAuditLogs(w http.ResponseWriter, r *http.Request) {
	// Parse query params
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	startDateStr := r.URL.Query().Get("startDate")
	endDateStr := r.URL.Query().Get("endDate")
	tenantID := r.URL.Query().Get("tenantId")

	limit := 10
	offset := 0
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			offset = o
		}
	}

	// Connect to Trino
	dsn := "http://admin@trino:8080?catalog=iceberg&schema=audit"
	db, err := sql.Open("trino", dsn)
	if err != nil {
		http.Error(w, "Failed to connect to Trino: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Build Query
	baseQuery := " FROM iceberg.audit.audit_logs WHERE 1=1"
	var args []interface{}

	if startDateStr != "" {
		// Expect ISO format or simple YYYY-MM-DD
		// Trino expects TIMESTAMP type, so we might need casting if passing string directly,
		// but Go driver might handle time.Time. Let's try passing string as timestamp literal or compatible format.
		// Safe bet: cast to timestamp in SQL
		baseQuery += " AND timestamp >= CAST(? AS TIMESTAMP)"
		args = append(args, startDateStr)
	}
	if endDateStr != "" {
		baseQuery += " AND timestamp <= CAST(? AS TIMESTAMP)"
		args = append(args, endDateStr)
	}
	if tenantID != "" {
		baseQuery += " AND tenant_id = ?"
		args = append(args, tenantID)
	}

	// Count Query
	countQuery := "SELECT COUNT(*)" + baseQuery
	var total int
	// Use separate slice for count args to avoid modifying the common args list if we appended limit/offset later
	// actually we haven't appended limit/offset yet.
	if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		logging.GetLogger().Sugar().Warnf("Trino count query failed: %v", err)
		// Don't fail completely, just use 0 or estimate
	}

	sortBy := r.URL.Query().Get("sortBy")
	sortOrder := r.URL.Query().Get("sortOrder")

	// Whitelist allowed sort columns
	allowedSort := map[string]bool{
		"timestamp": true,
		"user_name": true,
		"action":    true,
		"resource":  true,
	}
	if !allowedSort[sortBy] {
		sortBy = "timestamp"
	}
	if sortOrder != "ASC" && sortOrder != "DESC" {
		sortOrder = "DESC"
	}

	// Data Query
	query := fmt.Sprintf("SELECT id, tenant_id, timestamp, user_name, user_email, action, resource, resource_type, details"+baseQuery+" ORDER BY %s %s OFFSET ? LIMIT ?", sortBy, sortOrder)
	args = append(args, offset, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		logging.GetLogger().Sugar().Warnf("Trino query failed: %v", err)
		// Return empty list on error instead of mock data
		response := AuditLogResponse{
			Entries: []AuditLogEntry{},
			Total:   0,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}
	defer rows.Close()

	var entries []AuditLogEntry
	for rows.Next() {
		var e AuditLogEntry
		var detailsStr string
		if err := rows.Scan(&e.ID, &e.TenantID, &e.Timestamp, &e.UserName, &e.UserEmail, &e.Action, &e.Resource, &e.ResourceType, &detailsStr); err != nil {
			logging.GetLogger().Sugar().Errorf("Failed to scan row: %v", err)
			continue
		}
		var detailsMap map[string]interface{}
		if err := json.Unmarshal([]byte(detailsStr), &detailsMap); err == nil {
			e.Details = detailsMap
		} else {
			e.Details = map[string]interface{}{"raw": detailsStr}
		}
		entries = append(entries, e)
	}

	response := AuditLogResponse{
		Entries: entries,
		Total:   total,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
