package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Tenant represents tenant metadata
type Tenant struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Plan          string    `json:"plan"` // enterprise, standard
	SLOStatus     string    `json:"slo_status"`
	QueueDepth    int       `json:"queue_depth"`
	CacheBudgetMB int       `json:"cache_budget_mb"`
	LastRefresh   time.Time `json:"last_refresh"`
}

// ListTenantsHandler returns all tenants with their SLO status
func ListTenantsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Query tenants from database
		query := `
			SELECT 
				t.id,
				t.name,
				t.plan,
				COALESCE(m.slo_status, 'unknown') as slo_status,
				COALESCE(m.queue_depth, 0) as queue_depth,
				COALESCE(m.cache_budget_mb, 128) as cache_budget_mb,
				COALESCE(m.last_refresh, NOW()) as last_refresh
			FROM tenant_datasources t
			LEFT JOIN (
				SELECT tenant_id, 
					MAX(last_build) as last_refresh,
					COUNT(CASE WHEN status='healthy' THEN 1 END) > 0 as slo_status,
					0 as queue_depth,
					128 as cache_budget_mb
				FROM semantic_layer.rollups
				GROUP BY tenant_id
			) m ON t.id = m.tenant_id
			WHERE t.is_active = true
		`

		rows, err := db.Query(query)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to query tenants: %v", err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var tenants []Tenant
		for rows.Next() {
			var t Tenant
			var sloStatus bool
			err := rows.Scan(&t.ID, &t.Name, &t.Plan, &sloStatus, &t.QueueDepth, &t.CacheBudgetMB, &t.LastRefresh)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if sloStatus {
				t.SLOStatus = "healthy"
			} else {
				t.SLOStatus = "breached"
			}
			tenants = append(tenants, t)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tenants)
	}
}

// Worker represents worker pool metadata
type Worker struct {
	ID              string   `json:"id"`
	Lane            string   `json:"lane"` // hot, cold
	QueueDepth      int      `json:"queue_depth"`
	Failures        int      `json:"failures"`
	Status          string   `json:"status"`
	AssignedTenants []string `json:"assigned_tenants,omitempty"`
}

// ListWorkersHandler returns worker pool status
func ListWorkersHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Mock worker data for now - would integrate with Cube.js orchestrator
		workers := []Worker{
			{
				ID:         "worker_hot_1",
				Lane:       "hot",
				QueueDepth: 5,
				Failures:   0,
				Status:     "healthy",
			},
			{
				ID:         "worker_cold_1",
				Lane:       "cold",
				QueueDepth: 2,
				Failures:   0,
				Status:     "healthy",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(workers)
	}
}

// Rollup represents rollup metadata
type Rollup struct {
	ID               string    `json:"id"`
	CubeName         string    `json:"cube_name"`
	TenantID         string    `json:"tenant_id"`
	FreshnessMinutes int       `json:"freshness_minutes"`
	Status           string    `json:"status"`
	LastBuild        time.Time `json:"last_build"`
}

// ListRollupsHandler returns rollup status for tenant(s)
func ListRollupsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.URL.Query().Get("tenantId")

		query := `
			SELECT rollup_id, cube_name, tenant_id, 
				   COALESCE(freshness_minutes, 999) as freshness_minutes,
				   COALESCE(status, 'unknown') as status,
				   COALESCE(last_build, NOW()) as last_build
			FROM semantic_layer.rollups
			WHERE date = CURRENT_DATE
		`

		args := []interface{}{}
		if tenantID != "" {
			query += " AND tenant_id = ?"
			args = append(args, tenantID)
		}

		rows, err := db.Query(query, args...)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to query rollups: %v", err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var rollups []Rollup
		for rows.Next() {
			var r Rollup
			err := rows.Scan(&r.ID, &r.CubeName, &r.TenantID, &r.FreshnessMinutes, &r.Status, &r.LastBuild)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			rollups = append(rollups, r)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rollups)
	}
}

// RefreshRollupRequest represents a rollup refresh request
type RefreshRollupRequest struct {
	RollupID string `json:"rollup_id"`
	TenantID string `json:"tenant_id"`
}

// RefreshRollupResponse represents the refresh response
type RefreshRollupResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// RefreshRollupHandler triggers a targeted rollup refresh
func RefreshRollupHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RefreshRollupRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Log audit event
		actor := r.Header.Get("X-User-Id")
		logAuditEvent(db, actor, "refresh_rollup", fmt.Sprintf("%s:%s", req.TenantID, req.RollupID), "requested")

		// TODO: Trigger Cube.js refresh via API
		// For now, just log the request
		resp := RefreshRollupResponse{
			Success: true,
			Message: fmt.Sprintf("Refresh queued for %s", req.RollupID),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// logAuditEvent records admin actions
func logAuditEvent(db *sql.DB, actor, action, scope, result string) {
	_, err := db.Exec(`
		INSERT INTO semantic_layer.audit_log (actor, action, scope, result, timestamp)
		VALUES (?, ?, ?, ?, ?)
	`, actor, action, scope, result, time.Now())

	if err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to log audit event: %v\n", err)
	}
}
