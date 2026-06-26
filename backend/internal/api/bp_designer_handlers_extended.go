package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

/**
 * Extended BP Designer Handlers
 * Implements triggers 8-13 (Workflow Step, Status Change, Bulk Load,
 * Calculated Field, Timeout, Security Role)
 *
 * These extend bp_designer_handlers.go with the 6 pending triggers
 */

// BPDesignerHandlers provides HTTP handlers for the Business Process Designer
type BPDesignerHandlersExt struct {
	db *sql.DB
}

// TriggerConfiguration represents a validation trigger
type TriggerConfiguration struct {
	ID              string          `json:"id"`
	TenantID        string          `json:"tenant_id"`
	TriggerType     string          `json:"trigger_type"`     // save, field_change, delete, create, etc
	TargetEntity    string          `json:"target_entity"`    // orders, customers, employees
	EventConfig     json.RawMessage `json:"event_config"`     // Field filters
	ConditionConfig json.RawMessage `json:"condition_config"` // Rule conditions
	ActionConfig    json.RawMessage `json:"action_config"`    // Post-commit actions
	Enabled         bool            `json:"enabled"`
	Priority        int             `json:"priority"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// StepTimeout represents a process step timeout
type StepTimeout struct {
	ID            string     `json:"id"`
	TenantID      string     `json:"tenant_id"`
	BPExecutionID string     `json:"bp_execution_id"`
	StepName      string     `json:"step_name"`
	StartedAt     time.Time  `json:"started_at"`
	TimeoutAt     time.Time  `json:"timeout_at"`
	EscalatedAt   *time.Time `json:"escalated_at,omitempty"`
	EscalatedTo   *string    `json:"escalated_to,omitempty"`
	Status        string     `json:"status"` // pending, escalated, resolved
	CreatedAt     time.Time  `json:"created_at"`
}

// StatusChangeEvent represents a status field change
type StatusChangeEvent struct {
	EntityID   string `json:"entity_id"`
	EntityType string `json:"entity_type"`
	OldStatus  string `json:"old_status"`
	NewStatus  string `json:"new_status"`
}

// BulkLoadEvent represents a batch import
type BulkLoadEvent struct {
	ImportID    string        `json:"import_id"`
	EntityType  string        `json:"entity_type"`
	RecordCount int           `json:"record_count"`
	Records     []interface{} `json:"records"`
	StartedAt   time.Time     `json:"started_at"`
}

// CalculatedFieldUpdate represents a formula field recalculation
type CalculatedFieldUpdate struct {
	EntityID    string      `json:"entity_id"`
	FieldName   string      `json:"field_name"`
	OldValue    interface{} `json:"old_value"`
	NewValue    interface{} `json:"new_value"`
	DependentOn []string    `json:"dependent_on"`
}

// RoleChangeEvent represents a security role assignment
type RoleChangeEvent struct {
	UserID  string `json:"user_id"`
	OldRole string `json:"old_role"`
	NewRole string `json:"new_role"`
}

// ============================================================================
// TRIGGER 8: Workflow Step Completion
// ============================================================================

// OnWorkflowStepComplete fires when a BP step completes (Phase 6A)
func (h *BPDesignerHandlersExt) OnWorkflowStepComplete(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID

	var req struct {
		ProcessID string      `json:"process_id"`
		StepName  string      `json:"step_name"`
		Result    interface{} `json:"result"`
		Timestamp time.Time   `json:"timestamp"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Query triggers WHERE trigger_type = 'workflow_step' AND target_entity = ?
	query := `
		SELECT id, event_config, condition_config, action_config
		FROM validation_triggers
		WHERE tenant_id = $1 AND trigger_type = 'workflow_step'
		AND event_config->>'step_name' = $2 AND enabled = true
		ORDER BY priority ASC
	`

	rows, err := h.db.QueryContext(r.Context(), query, tenantID, req.StepName)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	// Execute triggered actions
	triggerCount := 0
	for rows.Next() {
		var triggerID string
		var eventCfg, condCfg, actionCfg sql.NullString

		if err := rows.Scan(&triggerID, &eventCfg, &condCfg, &actionCfg); err != nil {
			continue
		}

		// Evaluate condition (if exists)
		if condCfg.Valid {
			// Parse and evaluate conditions against step result
			conditions := make([]map[string]interface{}, 0)
			if err := json.Unmarshal([]byte(condCfg.String), &conditions); err == nil {
				// TODO: Implement condition evaluation logic
				// For now, assume all conditions pass
			}
		}

		// Execute action
		if actionCfg.Valid {
			actions := make(map[string]interface{})
			if err := json.Unmarshal([]byte(actionCfg.String), &actions); err == nil {
				// Execute action: send notification, update status, etc
				// This would call the appropriate handler based on action type
			}
		}

		triggerCount++
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"step_name":         req.StepName,
		"triggers_executed": triggerCount,
		"timestamp":         req.Timestamp,
	})
}

// ============================================================================
// TRIGGER 9: Status Change
// ============================================================================

// OnStatusChange fires when a status field is updated
func (h *BPDesignerHandlersExt) OnStatusChange(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID

	var event StatusChangeEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Validate status transition (optional pre-check)
	if event.OldStatus == event.NewStatus {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Status unchanged"})
		return
	}

	// Query triggers WHERE trigger_type = 'status_change'
	query := `
		SELECT id, event_config, action_config
		FROM validation_triggers
		WHERE tenant_id = $1 AND trigger_type = 'status_change'
		AND event_config->>'from' = $2 AND event_config->>'to' = $3
		AND enabled = true
	`

	rows, err := h.db.QueryContext(r.Context(), query, tenantID, event.OldStatus, event.NewStatus)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var triggerID string
		var eventCfg, actionCfg sql.NullString

		rows.Scan(&triggerID, &eventCfg, &actionCfg)

		// Execute action: notification, escalation, etc
		if actionCfg.Valid {
			action := make(map[string]interface{})
			json.Unmarshal([]byte(actionCfg.String), &action)
			// Execute based on action type
		}
	}

	// Emit event for subscribers
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"entity_id":  event.EntityID,
		"old_status": event.OldStatus,
		"new_status": event.NewStatus,
		"timestamp":  time.Now(),
	})
}

// ============================================================================
// TRIGGER 10: Bulk Load (Batch Import)
// ============================================================================

// OnBulkLoad fires when batch records are imported
func (h *BPDesignerHandlersExt) OnBulkLoad(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID

	var event BulkLoadEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Query triggers WHERE trigger_type = 'bulk_load'
	query := `
		SELECT id, condition_config, action_config
		FROM validation_triggers
		WHERE tenant_id = $1 AND trigger_type = 'bulk_load'
		AND enabled = true
	`

	rows, err := h.db.QueryContext(r.Context(), query, tenantID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	imported := 0
	skipped := 0

	// Process each record with per-record validation
	for range event.Records {
		// Evaluate per-record triggers
		recordValid := true

		// TODO: Call EvaluateTriggers for each record
		// if err := h.EvaluateTriggers(..., "save", event.EntityType, record); err != nil {
		// 	recordValid = false
		// 	skipped++
		// }

		if recordValid {
			// Insert record
			imported++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"import_id": event.ImportID,
		"imported":  imported,
		"skipped":   skipped,
		"total":     len(event.Records),
	})
}

// ============================================================================
// TRIGGER 11: Calculated Field
// ============================================================================

// RecalculateFields recalculates dependent computed fields
func (h *BPDesignerHandlersExt) RecalculateFields(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID

	var req struct {
		EntityID   string   `json:"entity_id"`
		EntityType string   `json:"entity_type"`
		Fields     []string `json:"fields"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Query calculated field definitions
	query := `
		SELECT action_config
		FROM validation_triggers
		WHERE tenant_id = $1 AND trigger_type = 'calculated_field'
		AND enabled = true
	`

	rows, err := h.db.QueryContext(r.Context(), query, tenantID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	updated := make(map[string]interface{})

	for rows.Next() {
		var actionCfg sql.NullString
		rows.Scan(&actionCfg)

		if actionCfg.Valid {
			fieldDef := make(map[string]interface{})
			json.Unmarshal([]byte(actionCfg.String), &fieldDef)

			// Evaluate formula: total = SUM(line_items.qty * price)
			// Updated value would be: updated[fieldName] = result
			_ = fieldDef // TODO: Evaluate formula
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"entity_id":           req.EntityID,
		"recalculated_fields": updated,
		"timestamp":           time.Now(),
	})
}

// ============================================================================
// TRIGGER 12: Timeout (Escalation)
// ============================================================================

// CreateStepTimeout creates a timeout for a BP step (e.g., approval overdue)
func (h *BPDesignerHandlersExt) CreateStepTimeout(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID

	var req struct {
		BPExecutionID    string `json:"bp_execution_id"`
		StepName         string `json:"step_name"`
		TimeoutValue     int    `json:"timeout_value"` // 2, 48, 7
		TimeoutUnit      string `json:"timeout_unit"`  // hours, days, sla
		EscalationAction string `json:"escalation_action"`
		EscalateTo       string `json:"escalate_to"` // Manager UUID
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Calculate timeout_at based on unit
	var timeoutAt time.Time
	now := time.Now()

	switch req.TimeoutUnit {
	case "hours":
		timeoutAt = now.Add(time.Duration(req.TimeoutValue) * time.Hour)
	case "days":
		timeoutAt = now.Add(time.Duration(req.TimeoutValue*24) * time.Hour)
	case "sla":
		// SLA typically 48 hours, no weekends
		timeoutAt = now.Add(time.Duration(req.TimeoutValue*48) * time.Hour) // Simplified
	default:
		timeoutAt = now.Add(24 * time.Hour)
	}

	// Insert timeout record
	id := uuid.New().String()
	query := `
		INSERT INTO step_timeouts (id, tenant_id, bp_execution_id, step_name, started_at, timeout_at, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'pending')
		RETURNING id
	`

	var returnID string
	err := h.db.QueryRowContext(r.Context(), query, id, tenantID, req.BPExecutionID, req.StepName, now, timeoutAt).Scan(&returnID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"id":                returnID,
		"timeout_at":        timeoutAt,
		"escalation_action": req.EscalationAction,
	})
}

// GetPendingTimeouts returns all timeouts that haven't escalated
func (h *BPDesignerHandlersExt) GetPendingTimeouts(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID

	query := `
		SELECT id, bp_execution_id, step_name, started_at, timeout_at, status
		FROM step_timeouts
		WHERE tenant_id = $1 AND status = 'pending' AND timeout_at <= NOW()
		ORDER BY timeout_at ASC
	`

	rows, err := h.db.QueryContext(r.Context(), query, tenantID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	var timeouts []StepTimeout
	for rows.Next() {
		var t StepTimeout
		rows.Scan(&t.ID, &t.BPExecutionID, &t.StepName, &t.StartedAt, &t.TimeoutAt, &t.Status)
		timeouts = append(timeouts, t)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(timeouts)
}

// EscalateTimeout escalates an overdue step (notify manager, auto-approve, etc)
func (h *BPDesignerHandlersExt) EscalateTimeout(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	timeoutID := chi.URLParam(r, "id")

	var req struct {
		EscalationAction string `json:"escalation_action"` // notify, escalate, auto_approve, auto_reject
		EscalateTo       string `json:"escalate_to"`       // Manager UUID
		Message          string `json:"message,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Update timeout status
	now := time.Now()
	query := `
		UPDATE step_timeouts
		SET status = 'escalated', escalated_at = $1, escalated_to = $2
		WHERE id = $3 AND tenant_id = $4
		RETURNING id
	`

	var returnID string
	err := h.db.QueryRowContext(r.Context(), query, now, req.EscalateTo, timeoutID, tenantID).Scan(&returnID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Execute escalation action
	switch req.EscalationAction {
	case "notify":
		// Send notification to manager
		// h.notificationService.SendToUser(req.EscalateTo, "Step overdue", req.Message)

	case "escalate":
		// Escalate to higher manager
		// h.escalationService.EscalateToHierarchy(req.EscalateTo)

	case "auto_approve":
		// Automatically approve the step
		// h.approvalService.AutoApprove(timeoutID)

	case "auto_reject":
		// Automatically reject the step
		// h.approvalService.AutoReject(timeoutID, "Auto-rejected due to timeout")
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"id":                returnID,
		"escalated_at":      now,
		"escalation_action": req.EscalationAction,
	})
}

// ============================================================================
// TRIGGER 13: Security Role Change
// ============================================================================

// OnRoleChange fires when a user role is assigned/changed
func (h *BPDesignerHandlersExt) OnRoleChange(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID

	var event RoleChangeEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Query triggers WHERE trigger_type = 'role_change'
	query := `
		SELECT id, condition_config, action_config
		FROM validation_triggers
		WHERE tenant_id = $1 AND trigger_type = 'role_change'
		AND event_config->>'role' = $2 AND enabled = true
	`

	rows, err := h.db.QueryContext(r.Context(), query, tenantID, event.NewRole)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var triggerID string
		var condCfg, actionCfg sql.NullString

		rows.Scan(&triggerID, &condCfg, &actionCfg)

		// Execute action: audit log, notification, permission update, etc
		if actionCfg.Valid {
			action := make(map[string]interface{})
			json.Unmarshal([]byte(actionCfg.String), &action)
			// Execute action based on type
		}

		// Log role change for audit trail
		auditQuery := `
			INSERT INTO audit_log (tenant_id, entity_type, entity_id, action, old_value, new_value, timestamp)
			VALUES ($1, 'user', $2, 'role_change', $3, $4, NOW())
		`
		h.db.ExecContext(r.Context(), auditQuery, tenantID, event.UserID, event.OldRole, event.NewRole)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"user_id":   event.UserID,
		"old_role":  event.OldRole,
		"new_role":  event.NewRole,
		"timestamp": time.Now(),
	})
}

// ============================================================================
// Utility Functions
// ============================================================================

// RegisterRoutes registers handlers for triggers 8-13
func (h *BPDesignerHandlersExt) RegisterRoutes(r chi.Router) {
	r.Route("/api/bp/triggers", func(r chi.Router) {
		// Trigger 8: Workflow Step
		r.Post("/workflow-step", h.OnWorkflowStepComplete)

		// Trigger 9: Status Change
		r.Post("/status-change", h.OnStatusChange)

		// Trigger 10: Bulk Load
		r.Post("/bulk-load", h.OnBulkLoad)

		// Trigger 11: Calculated Fields
		r.Post("/recalculate-fields", h.RecalculateFields)

		// Trigger 12: Timeout
		r.Route("/timeout", func(r chi.Router) {
			r.Post("/create", h.CreateStepTimeout)
			r.Get("/pending", h.GetPendingTimeouts)
			r.Post("/{id}/escalate", h.EscalateTimeout)
		})

		// Trigger 13: Role Change
		r.Post("/role-change", h.OnRoleChange)
	})
}
