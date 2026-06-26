package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// ============================================================================
// TRIGGER MANAGEMENT HANDLERS (Admin API)
// ============================================================================

type TriggerHandler struct {
	db     *sqlx.DB
	engine *TriggerEngine
}

func NewTriggerHandler(db *sqlx.DB, engine *TriggerEngine) *TriggerHandler {
	return &TriggerHandler{
		db:     db,
		engine: engine,
	}
}

// ListTriggerTypes returns all available trigger types
func (h *TriggerHandler) ListTriggerTypes(w http.ResponseWriter, r *http.Request) {
	query := `SELECT id, key, label, description, icon_svg, category FROM trigger_types ORDER BY category, label`

	var types []map[string]interface{}
	rows, err := h.db.QueryxContext(r.Context(), query)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
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

// ListValidationOperators returns all available operators
func (h *TriggerHandler) ListValidationOperators(w http.ResponseWriter, r *http.Request) {
	query := `SELECT id, key, label, value_type FROM validation_operators ORDER BY label`

	var operators []map[string]interface{}
	rows, err := h.db.QueryxContext(r.Context(), query)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
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

// ListWorkflowEvents returns all available events
func (h *TriggerHandler) ListWorkflowEvents(w http.ResponseWriter, r *http.Request) {
	// Enforce tenant scope via X-Tenant-ID header so the frontend receives a
	// helpful error when tenant/datasource is not selected in the UI.
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "tenant_id required"})
		return
	}

	query := `SELECT id, key, label, description, event_type, config, created_at, updated_at
		FROM workflow_events
		WHERE tenant_id IS NULL OR tenant_id = $1
		ORDER BY label`

	rows, err := h.db.QueryxContext(r.Context(), query, tenantID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
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

// ListBusinessObjects returns all available business objects (entities)
func (h *TriggerHandler) ListBusinessObjects(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID

	query := `SELECT id, name, display_name, description, fields FROM business_objects WHERE tenant_id = $1 ORDER BY display_name`

	var objects []map[string]interface{}
	rows, err := h.db.QueryxContext(r.Context(), query, tenantID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
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

// ============================================================================
// TRIGGER CRUD
// ============================================================================

// CreateValidationTrigger creates a new validation trigger
func (h *TriggerHandler) CreateValidationTrigger(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	userID := r.Header.Get("X-User-ID")

	var req struct {
		TriggerTypeID   string          `json:"trigger_type_id" binding:"required"`
		TargetEntity    string          `json:"target_entity" binding:"required"`
		EventID         *string         `json:"event_id"`
		EventConfig     json.RawMessage `json:"event_config"`
		ConditionConfig json.RawMessage `json:"condition_config" binding:"required"`
		ActionConfig    json.RawMessage `json:"action_config"`
		ABACPolicyID    *string         `json:"abac_policy_id"`
		Enabled         bool            `json:"enabled"`
		Priority        int             `json:"priority"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	query := `
		INSERT INTO validation_triggers 
		(tenant_id, trigger_type_id, target_entity, event_id, event_config, 
		 condition_config, action_config, abac_policy_id, enabled, priority, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at`

	var id string
	var createdAt time.Time
	err := h.db.QueryRowxContext(r.Context(), query,
		tenantID, req.TriggerTypeID, req.TargetEntity, req.EventID,
		req.EventConfig, req.ConditionConfig, req.ActionConfig,
		req.ABACPolicyID, req.Enabled, req.Priority, userID,
	).Scan(&id, &createdAt)

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"id":         id,
		"created_at": createdAt,
	})
}

// GetValidationTriggers lists all validation triggers for a tenant
func (h *TriggerHandler) GetValidationTriggers(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	targetEntity := r.URL.Query().Get("target_entity")

	query := `
		SELECT vt.id, vt.trigger_type_id, vt.target_entity, vt.event_id,
		       vt.event_config, vt.condition_config, vt.action_config,
		       vt.abac_policy_id, vt.enabled, vt.priority, vt.created_at,
		       tt.key as trigger_key, tt.label as trigger_label
		FROM validation_triggers vt
		JOIN trigger_types tt ON vt.trigger_type_id = tt.id
		WHERE vt.tenant_id = $1`

	args := []interface{}{tenantID}
	if targetEntity != "" {
		query += ` AND vt.target_entity = $2`
		args = append(args, targetEntity)
	}
	query += ` ORDER BY vt.priority ASC, vt.created_at DESC`

	var triggers []map[string]interface{}
	rows, err := h.db.QueryxContext(r.Context(), query, args...)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
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

// UpdateValidationTrigger updates an existing trigger
func (h *TriggerHandler) UpdateValidationTrigger(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	userID := r.Header.Get("X-User-ID")
	triggerID := chi.URLParam(r, "id")

	var req struct {
		EventConfig     json.RawMessage `json:"event_config"`
		ConditionConfig json.RawMessage `json:"condition_config"`
		ActionConfig    json.RawMessage `json:"action_config"`
		Enabled         *bool           `json:"enabled"`
		Priority        *int            `json:"priority"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	query := `
		UPDATE validation_triggers
		SET event_config = COALESCE($1, event_config),
		    condition_config = COALESCE($2, condition_config),
		    action_config = COALESCE($3, action_config),
		    enabled = COALESCE($4, enabled),
		    priority = COALESCE($5, priority),
		    updated_by = $6,
		    updated_at = NOW()
		WHERE id = $7 AND tenant_id = $8
		RETURNING id, updated_at`

	var updatedID string
	var updatedAt time.Time
	err := h.db.QueryRowxContext(r.Context(), query,
		req.EventConfig, req.ConditionConfig, req.ActionConfig,
		req.Enabled, req.Priority, userID, triggerID, tenantID,
	).Scan(&updatedID, &updatedAt)

	if err == sql.ErrNoRows {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "trigger not found"})
		return
	}
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"id":         updatedID,
		"updated_at": updatedAt,
	})
}

// DeleteValidationTrigger deletes a trigger
func (h *TriggerHandler) DeleteValidationTrigger(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	triggerID := chi.URLParam(r, "id")

	result, err := h.db.ExecContext(r.Context(),
		`DELETE FROM validation_triggers WHERE id = $1 AND tenant_id = $2`,
		triggerID, tenantID,
	)

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "trigger not found"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"deleted": triggerID})
}

// ============================================================================
// TIMEOUT TRIGGER MANAGEMENT
// ============================================================================

// CreateTimeoutTrigger creates a timeout trigger
func (h *TriggerHandler) CreateTimeoutTrigger(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	// user_id is not required for timeout trigger creation at this time

	var req struct {
		ProcessID            string  `json:"process_id" binding:"required"`
		StepName             string  `json:"step_name" binding:"required"`
		TimeoutValue         int     `json:"timeout_value" binding:"required"`
		TimeoutUnit          string  `json:"timeout_unit" binding:"required"`
		EscalationAction     string  `json:"escalation_action" binding:"required"`
		EscalateToRole       *string `json:"escalate_to_role"`
		EscalateToUser       *string `json:"escalate_to_user"`
		NotificationTemplate string  `json:"notification_template"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"id":         id,
		"created_at": createdAt,
	})
}

// GetPendingTimeouts returns all overdue timeouts
func (h *TriggerHandler) GetPendingTimeouts(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID

	query := `
		SELECT id, bp_execution_id, step_name, started_at, timeout_at,
		       escalation_action, escalate_to_user, status
		FROM step_timeouts
		WHERE tenant_id = $1 AND status = 'pending' AND timeout_at <= NOW()
		ORDER BY timeout_at ASC`

	var timeouts []map[string]interface{}
	rows, err := h.db.QueryxContext(r.Context(), query, tenantID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
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

// EscalateTimeout manually escalates a timeout
func (h *TriggerHandler) EscalateTimeout(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	escalatedByUser := r.Header.Get("X-User-ID")
	timeoutID := chi.URLParam(r, "id")

	var req struct {
		Action string `json:"action" binding:"required"` // notify, escalate, auto_approve, auto_reject
		Notes  string `json:"notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	query := `
		UPDATE step_timeouts
		SET status = 'escalated',
		    escalated_at = NOW(),
		    escalated_to_user = $1,
		    escalation_action = $2,
		    notes = $3
		WHERE id = $4 AND tenant_id = $5
		RETURNING id, escalated_at`

	var escalatedID string
	var escalatedAt time.Time
	err := h.db.QueryRowxContext(r.Context(), query,
		escalatedByUser, req.Action, req.Notes, timeoutID, tenantID,
	).Scan(&escalatedID, &escalatedAt)

	if err == sql.ErrNoRows {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "timeout not found"})
		return
	}
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"id":           escalatedID,
		"escalated_at": escalatedAt,
		"action":       req.Action,
	})
}

// ============================================================================
// TRIGGER EXECUTION HISTORY & AUDIT
// ============================================================================

// GetTriggerExecutions returns execution history for audit
func (h *TriggerHandler) GetTriggerExecutions(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	triggerID := r.URL.Query().Get("trigger_id")
	targetEntity := r.URL.Query().Get("target_entity")
	status := r.URL.Query().Get("status")

	query := `
		SELECT id, trigger_id, trigger_key, target_entity, entity_id,
		       status, error_message, executed_by, executed_at, duration_ms
		FROM trigger_executions
		WHERE tenant_id = $1`

	args := []interface{}{tenantID}
	argCount := 2

	if triggerID != "" {
		query += ` AND trigger_id = $` + strconv.Itoa(argCount)
		args = append(args, triggerID)
		argCount++
	}
	if targetEntity != "" {
		query += ` AND target_entity = $` + strconv.Itoa(argCount)
		args = append(args, targetEntity)
		argCount++
	}
	if status != "" {
		query += ` AND status = $` + strconv.Itoa(argCount)
		args = append(args, status)
	}

	query += ` ORDER BY executed_at DESC LIMIT 100`

	var executions []map[string]interface{}
	rows, err := h.db.QueryxContext(r.Context(), query, args...)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
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

// ============================================================================
// ROUTE REGISTRATION
// ============================================================================

// RegisterRoutes registers all trigger-related routes
func (h *TriggerHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/triggers", func(r chi.Router) {
			// Admin metadata endpoints
			r.Get("/types", h.ListTriggerTypes)
			r.Get("/operators", h.ListValidationOperators)
			r.Get("/events", h.ListWorkflowEvents)
			r.Get("/objects", h.ListBusinessObjects)
			r.Get("/executions", h.GetTriggerExecutions)

			// Trigger CRUD
			r.Post("/", h.CreateValidationTrigger)
			r.Get("/", h.GetValidationTriggers)
			r.Route("/{id}", func(r chi.Router) {
				r.Put("/", h.UpdateValidationTrigger)
				r.Delete("/", h.DeleteValidationTrigger)
			})
		})

		r.Route("/timeouts", func(r chi.Router) {
			r.Post("/", h.CreateTimeoutTrigger)
			r.Get("/pending", h.GetPendingTimeouts)
			r.Post("/{id}/escalate", h.EscalateTimeout)
		})
	})
}
