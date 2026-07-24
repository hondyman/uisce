package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ResourceGroup represents a StarRocks resource group
type ResourceGroup struct {
	ID               string `json:"id"`
	Database         string `json:"database"`
	CPUShare         int    `json:"cpu_share"`
	MemLimit         int    `json:"mem_limit"`
	ConcurrencyLimit int    `json:"concurrency_limit"`
}

// UpdateResourceGroupRequest represents update request
type UpdateResourceGroupRequest struct {
	ID               string `json:"id"`
	CPUShare         int    `json:"cpu_share"`
	MemLimit         int    `json:"mem_limit"`
	ConcurrencyLimit int    `json:"concurrency_limit"`
}

// UpdateResourceGroupResponse represents the update response
type UpdateResourceGroupResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// UpdateResourceGroupHandler updates StarRocks resource group quotas
func UpdateResourceGroupHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor := r.Header.Get("X-User-Id")

		var req UpdateResourceGroupRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Execute ALTER RESOURCE GROUP
		sqlStmt := fmt.Sprintf(`
			ALTER RESOURCE GROUP %s
			WITH (
				'cpu_share' = '%d',
				'mem_limit' = '%d%%',
				'concurrency_limit' = '%d'
			)
		`, req.ID, req.CPUShare, req.MemLimit, req.ConcurrencyLimit)

		_, err := db.Exec(sqlStmt)

		result := "success"
		if err != nil {
			result = fmt.Sprintf("failure: %v", err)
		}

		// Log audit event
		logAuditEvent(db, actor, "update_resource_group", req.ID, result)

		resp := UpdateResourceGroupResponse{
			Success: err == nil,
			Message: result,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// CreateResourceGroupRequest represents create request
type CreateResourceGroupRequest struct {
	ID               string `json:"id"`
	Database         string `json:"database"`
	CPUShare         int    `json:"cpu_share"`
	MemLimit         int    `json:"mem_limit"`
	ConcurrencyLimit int    `json:"concurrency_limit"`
}

// CreateResourceGroupResponse represents create response
type CreateResourceGroupResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// CreateResourceGroupHandler creates a new StarRocks resource group
func CreateResourceGroupHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor := r.Header.Get("X-User-Id")

		var req CreateResourceGroupRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		sqlStmt := fmt.Sprintf(`
			CREATE RESOURCE GROUP %s
			TO ('%s')
			WITH (
				'cpu_share' = '%d',
				'mem_limit' = '%d%%',
				'concurrency_limit' = '%d'
			)
		`, req.ID, req.Database, req.CPUShare, req.MemLimit, req.ConcurrencyLimit)

		_, err := db.Exec(sqlStmt)

		result := "success"
		if err != nil {
			result = fmt.Sprintf("failure: %v", err)
		}

		logAuditEvent(db, actor, "create_resource_group", req.ID, result)

		resp := CreateResourceGroupResponse{
			Success: err == nil,
			Message: result,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// ListResourceGroupsHandler lists all resource groups
func ListResourceGroupsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Mock data - would query StarRocks system tables in production
		groups := []ResourceGroup{
			{
				ID:               "semantic_rollups",
				Database:         "semantic_layer",
				CPUShare:         50,
				MemLimit:         60,
				ConcurrencyLimit: 50,
			},
			{
				ID:               "calc_engine",
				Database:         "calc_engine",
				CPUShare:         30,
				MemLimit:         30,
				ConcurrencyLimit: 100,
			},
			{
				ID:               "default_group",
				Database:         "default",
				CPUShare:         20,
				MemLimit:         10,
				ConcurrencyLimit: 20,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(groups)
	}
}

// UpdateTenantConcurrencyRequest represents concurrency update request
type UpdateTenantConcurrencyRequest struct {
	Concurrency   int `json:"concurrency"`
	CacheBudgetMB int `json:"cache_budget_mb"`
}

// UpdateTenantConcurrencyHandler updates per-tenant orchestration settings
func UpdateTenantConcurrencyHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor := r.Header.Get("X-User-Id")
		tenantID := r.URL.Query().Get("id")

		var req UpdateTenantConcurrencyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Store tenant settings (would persist to database)
		// TODO: Integrate with Cube.js orchestrator

		logAuditEvent(db, actor, "update_tenant_concurrency", tenantID,
			fmt.Sprintf("concurrency=%d, cache=%dMB", req.Concurrency, req.CacheBudgetMB))

		resp := UpdateResourceGroupResponse{
			Success: true,
			Message: "Tenant settings updated",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// AuditEvent represents an audit log entry
type AuditEvent struct {
	ID        int       `json:"id"`
	Actor     string    `json:"actor"`
	Action    string    `json:"action"`
	Scope     string    `json:"scope"`
	Timestamp time.Time `json:"timestamp"`
	Result    string    `json:"result"`
}

// ListAuditEventsHandler returns audit trail
func ListAuditEventsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.URL.Query().Get("tenantId")
		limit := 20 // default limit

		query := `
			SELECT id, actor, action, scope, timestamp, result
			FROM semantic_layer.audit_log
			WHERE 1=1
		`
		args := []interface{}{}

		if tenantID != "" {
			query += " AND scope LIKE ?"
			args = append(args, tenantID+"%")
		}

		query += " ORDER BY timestamp DESC LIMIT ?"
		args = append(args, limit)

		rows, err := db.Query(query, args...)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to query audit log: %v", err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var events []AuditEvent
		for rows.Next() {
			var e AuditEvent
			err := rows.Scan(&e.ID, &e.Actor, &e.Action, &e.Scope, &e.Timestamp, &e.Result)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			events = append(events, e)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(events)
	}
}
