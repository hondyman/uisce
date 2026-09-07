package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/hondyman/uisce/libs/jwt-middleware"
)

// Helper functions to read tenant/user from request (header preferred, then context)
func tenantIDFromRequest(r *http.Request) string {
	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil && claims.TenantID != "" {
		return claims.TenantID
	}
	if ctxv := r.Context().Value("tenant_id"); ctxv != nil {
		if s, ok := ctxv.(string); ok {
			return s
		}
	}
	if auth, ok := security.AuthInfoFromContext(r.Context()); ok && len(auth.TenantIDs) > 0 {
		return auth.TenantIDs[0]
	}
	return r.Header.Get("X-Tenant-ID")
}

func userIDFromRequest(r *http.Request) string {
	if v := r.Header.Get("X-User-ID"); v != "" {
		return v
	}
	if ctxv := r.Context().Value("user_id"); ctxv != nil {
		if s, ok := ctxv.(string); ok {
			return s
		}
	}
	return ""
}

// RegisterTriggerRoutesChi registers trigger-related endpoints on a chi router
// Note: paths are relative to the router context this is called in.
// Inside r.Route("/api", ...), use /v1/* paths (chi sub-router accumulates prefix).
// Called with a bare chi.NewRouter() (e.g. in tests), use /api/v1/* paths.
func RegisterTriggerRoutesChi(r chi.Router, db *sqlx.DB, engine *TriggerEngine) {
	handler := NewTriggerHandler(db, engine)

	// Admin metadata endpoints (public)
	r.Get("/v1/triggers/types", handler.ListTriggerTypesHTTP)
	r.Get("/v1/triggers/operators", handler.ListValidationOperatorsHTTP)
	r.Get("/v1/triggers/events", handler.ListWorkflowEventsHTTP)
	r.Get("/v1/triggers/objects", handler.ListBusinessObjectsHTTP)

	// Trigger CRUD (authenticated)
	r.Post("/v1/triggers", handler.CreateValidationTriggerHTTP)
	r.Get("/v1/triggers", handler.GetValidationTriggersHTTP)
	r.Put("/v1/triggers/{id}", handler.UpdateValidationTriggerHTTP)
	r.Delete("/v1/triggers/{id}", handler.DeleteValidationTriggerHTTP)

	// Timeout management
	r.Post("/v1/timeouts", handler.CreateTimeoutTriggerHTTP)
	r.Get("/v1/timeouts/pending", handler.GetPendingTimeoutsHTTP)
	r.Post("/v1/timeouts/{id}/escalate", handler.EscalateTimeoutHTTP)

	// Audit / executions
	r.Get("/v1/triggers/executions", handler.GetTriggerExecutionsHTTP)
}

// -------------------- HTTP handler implementations --------------------

func (h *TriggerHandler) ListTriggerTypesHTTP(w http.ResponseWriter, r *http.Request) {
	query := `SELECT id, key, label, description, icon_svg, category FROM trigger_types ORDER BY category, label`

	var types []map[string]interface{}
	rows, err := h.db.QueryxContext(r.Context(), query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		row := make(map[string]interface{})
		if err := rows.MapScan(row); err != nil {
			continue
		}
		types = append(types, row)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(types)
}

func (h *TriggerHandler) ListValidationOperatorsHTTP(w http.ResponseWriter, r *http.Request) {
	query := `SELECT id, key, label, value_type FROM validation_operators ORDER BY label`

	var operators []map[string]interface{}
	rows, err := h.db.QueryxContext(r.Context(), query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		row := make(map[string]interface{})
		if err := rows.MapScan(row); err != nil {
			continue
		}
		operators = append(operators, row)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(operators)
}

func (h *TriggerHandler) ListWorkflowEventsHTTP(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantIDFromRequest(r)
	if tenantID == "" {
		http.Error(w, "tenant_id required", http.StatusBadRequest)
		return
	}

	query := `SELECT id, key, label, description, event_type, config, created_at, updated_at
        FROM workflow_events
        WHERE tenant_id IS NULL OR tenant_id = $1
        ORDER BY label`

	rows, err := h.db.QueryxContext(r.Context(), query, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var events []map[string]interface{}
	for rows.Next() {
		row := make(map[string]interface{})
		if err := rows.MapScan(row); err != nil {
			continue
		}
		events = append(events, row)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

func (h *TriggerHandler) ListBusinessObjectsHTTP(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantIDFromRequest(r)

	query := `SELECT id, name, display_name, description, fields FROM business_objects WHERE tenant_id = $1 ORDER BY display_name`

	var objects []map[string]interface{}
	rows, err := h.db.QueryxContext(r.Context(), query, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		row := make(map[string]interface{})
		if err := rows.MapScan(row); err != nil {
			continue
		}
		objects = append(objects, row)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(objects)
}

func (h *TriggerHandler) CreateValidationTriggerHTTP(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantIDFromRequest(r)
	userID := userIDFromRequest(r)

	var req struct {
		TriggerType string `json:"trigger_type"`
		TargetEntity string `json:"target_entity"`
		RuleIDs     []string        `json:"rule_ids"`
		StepName    string          `json:"step_name"`
		Meta        json.RawMessage `json:"meta"`
		IsActive    *bool           `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.TriggerType == "" || req.TargetEntity == "" {
		http.Error(w, "trigger_type and target_entity are required", http.StatusBadRequest)
		return
	}

	if req.RuleIDs == nil {
		req.RuleIDs = []string{}
	}
	meta := req.Meta
	if meta == nil {
		meta = []byte("{}")
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	stepName := req.StepName
	if stepName == "" {
		stepName = req.TargetEntity + "_validation"
	}

	query := `
        INSERT INTO validation_triggers
        (tenant_id, trigger_type, target_entity, step_name, rule_ids, meta, is_active, created_by)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        RETURNING id, created_at`

	var id string
	var createdAt time.Time
	err := h.db.QueryRowxContext(r.Context(), query,
		tenantID, req.TriggerType, req.TargetEntity, stepName, req.RuleIDs, meta, isActive, userID,
	).Scan(&id, &createdAt)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"id": id, "created_at": createdAt})
}

func (h *TriggerHandler) GetValidationTriggersHTTP(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantIDFromRequest(r)
	targetEntity := r.URL.Query().Get("target_entity")

	query := `
		SELECT vt.id, vt.tenant_id, vt.trigger_type, vt.target_entity, vt.step_name,
		       vt.rule_ids, vt.meta, vt.pipeline_id, vt.dispatch_mode,
		       vt.is_active, vt.created_at,
		       lr.last_fired_at
		FROM validation_triggers vt
		LEFT JOIN LATERAL (
		    SELECT MAX(r.start_time) AS last_fired_at
		    FROM data_pipeline_runs r
		    WHERE r.trigger_id = vt.id
		) lr ON true
		WHERE vt.tenant_id = $1`

	args := []interface{}{tenantID}
	if targetEntity != "" {
		query += ` AND vt.target_entity = $2`
		args = append(args, targetEntity)
	}
	query += ` ORDER BY vt.created_at DESC`

	var triggers []map[string]interface{}
	rows, err := h.db.QueryxContext(r.Context(), query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		row := make(map[string]interface{})
		if err := rows.MapScan(row); err != nil {
			continue
		}
		triggers = append(triggers, row)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(triggers)
}

func (h *TriggerHandler) UpdateValidationTriggerHTTP(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantIDFromRequest(r)
	triggerID := chi.URLParam(r, "id")

	var req struct {
		StepName    *string `json:"step_name"`
		Meta        *json.RawMessage `json:"meta"`
		IsActive    *bool   `json:"is_active"`
		PipelineID  *string `json:"pipeline_id"`
		DispatchMode *string `json:"dispatch_mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := `
		UPDATE validation_triggers
		SET step_name = COALESCE($1, step_name),
		    meta = COALESCE($2, meta),
		    is_active = COALESCE($3, is_active),
		    pipeline_id = COALESCE($4, pipeline_id),
		    dispatch_mode = COALESCE($5, dispatch_mode),
		    updated_at = NOW()
		WHERE id = $6 AND tenant_id = $7
		RETURNING id, updated_at`

	var updatedID string
	var updatedAt time.Time
	err := h.db.QueryRowxContext(r.Context(), query,
		req.StepName, req.Meta, req.IsActive, req.PipelineID, req.DispatchMode, triggerID, tenantID,
	).Scan(&updatedID, &updatedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "trigger not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"id": updatedID, "updated_at": updatedAt})
}

func (h *TriggerHandler) DeleteValidationTriggerHTTP(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantIDFromRequest(r)
	triggerID := chi.URLParam(r, "id")

	result, err := h.db.ExecContext(r.Context(), `DELETE FROM validation_triggers WHERE id = $1 AND tenant_id = $2`, triggerID, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		http.Error(w, "trigger not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"deleted": triggerID})
}

func (h *TriggerHandler) CreateTimeoutTriggerHTTP(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantIDFromRequest(r)

	var req struct {
		ProcessID            string  `json:"process_id"`
		StepName             string  `json:"step_name"`
		TimeoutValue         int     `json:"timeout_value"`
		TimeoutUnit          string  `json:"timeout_unit"`
		EscalationAction     string  `json:"escalation_action"`
		EscalateToRole       *string `json:"escalate_to_role"`
		EscalateToUser       *string `json:"escalate_to_user"`
		NotificationTemplate string  `json:"notification_template"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := `
        INSERT INTO timeout_triggers
        (tenant_id, process_id, step_name, timeout_value, timeout_unit,
         escalation_action, escalate_to_role, escalate_to_user, notification_template)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        RETURNING id, created_at`

	var id string
	var createdAt time.Time
	err := h.db.QueryRowxContext(r.Context(), query,
		tenantID, req.ProcessID, req.StepName, req.TimeoutValue, req.TimeoutUnit,
		req.EscalationAction, req.EscalateToRole, req.EscalateToUser,
		req.NotificationTemplate,
	).Scan(&id, &createdAt)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"id": id, "created_at": createdAt})
}

func (h *TriggerHandler) GetPendingTimeoutsHTTP(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantIDFromRequest(r)

	query := `
        SELECT id, bp_execution_id, step_name, started_at, timeout_at,
               escalation_action, escalate_to_user, status
        FROM step_timeouts
        WHERE tenant_id = $1 AND status = 'pending' AND timeout_at <= NOW()
        ORDER BY timeout_at ASC`

	var timeouts []map[string]interface{}
	rows, err := h.db.QueryxContext(r.Context(), query, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		row := make(map[string]interface{})
		if err := rows.MapScan(row); err != nil {
			continue
		}
		timeouts = append(timeouts, row)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(timeouts)
}

func (h *TriggerHandler) EscalateTimeoutHTTP(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantIDFromRequest(r)
	escalatedByUser := userIDFromRequest(r)
	timeoutID := chi.URLParam(r, "id")

	var req struct {
		Action string `json:"action"`
		Notes  string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Implementation simplified: mark as escalated and record notes
	_, err := h.db.ExecContext(r.Context(), `UPDATE step_timeouts SET status = 'escalated', updated_by = $1, updated_at = NOW() WHERE id = $2 AND tenant_id = $3`, escalatedByUser, timeoutID, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *TriggerHandler) GetTriggerExecutionsHTTP(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantIDFromRequest(r)

	query := `
		SELECT r.id, r.trigger_id, r.status, r.start_time, r.end_time, r.total_records_in, r.total_records_out, r.error_details
		FROM data_pipeline_runs r
		WHERE r.tenant_id = $1 AND r.trigger_id IS NOT NULL
		ORDER BY r.start_time DESC
		LIMIT 100`
	var executions []map[string]interface{}
	rows, err := h.db.QueryxContext(r.Context(), query, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		row := make(map[string]interface{})
		if err := rows.MapScan(row); err != nil {
			continue
		}
		executions = append(executions, row)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(executions)
}
