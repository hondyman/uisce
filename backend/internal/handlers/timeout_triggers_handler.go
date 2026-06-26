//go:build !bp_versioned
// +build !bp_versioned

package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/auth"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/jmoiron/sqlx"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// TimeoutAction represents an action to execute when timeout threshold is reached
type TimeoutAction struct {
	Percent int    `json:"percent" binding:"required,min=1,max=100"`
	Type    string `json:"type" binding:"required,oneof=escalate notify log cancel"`
	Target  string `json:"target" binding:"required"`
	Message string `json:"message"`
}

// TimeoutTrigger represents a workflow timeout trigger configuration
type TimeoutTrigger struct {
	ID                 string          `json:"id" db:"id"`
	TenantID           string          `json:"tenant_id" db:"tenant_id"`
	WorkflowName       string          `json:"workflow_name" db:"workflow_name" binding:"required"`
	StepName           string          `json:"step_name" db:"step_name" binding:"required"`
	DueHours           int             `json:"due_hours" db:"due_hours" binding:"required,min=1,max=999"`
	TriggerPercentages []int           `json:"trigger_percentages" db:"trigger_percentages"`
	Actions            []TimeoutAction `json:"actions" binding:"required"`
	IsActive           bool            `json:"is_active" db:"is_active"`
	CreatedAt          time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at" db:"updated_at"`
}

// TimeoutTriggersHandler encapsulates handlers for timeout triggers management
type TimeoutTriggersHandler struct {
	db *sqlx.DB
}

// NewTimeoutTriggersHandler creates a new timeout triggers handler
func NewTimeoutTriggersHandler(db *sqlx.DB) *TimeoutTriggersHandler {
	return &TimeoutTriggersHandler{db: db}
}

// RegisterRoutes adds the timeout trigger management routes to the router
func (h *TimeoutTriggersHandler) RegisterRoutes(r chi.Router) {
	r.Route("/workflow-timeout-triggers", func(r chi.Router) {
		r.Get("/", h.listTimeoutTriggers)
		r.Post("/", h.createTimeoutTrigger)
		r.Route("/{triggerId}", func(r chi.Router) {
			r.Get("/", h.getTimeoutTrigger)
			r.Put("/", h.updateTimeoutTrigger)
			r.Delete("/", h.deleteTimeoutTrigger)
			r.Post("/test", h.testTimeoutTrigger)
		})
	})
}

// getTenantID extracts tenant ID from request context
func (h *TimeoutTriggersHandler) getTenantID(r *http.Request) (string, error) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		return "", errors.New("X-Tenant-ID header is required")
	}
	return tenantID, nil
}

// getUser extracts user from context or returns default
func (h *TimeoutTriggersHandler) getUser(r *http.Request) models.User {
	if u, ok := auth.GetUserFromContext(r.Context()); ok {
		return u
	}
	return models.User{
		ID:           "system",
		Email:        "system@semlayer.io",
		Name:         "System",
		Role:         "Admin",
		Organization: "Semlayer",
		Permissions:  []string{"read", "write", "admin"},
		IsCoreAdmin:  true,
		IsActive:     true,
	}
}

// listTimeoutTriggers retrieves all timeout triggers for the tenant
func (h *TimeoutTriggersHandler) listTimeoutTriggers(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	query := `
		SELECT id, tenant_id, workflow_name, step_name, due_hours,
		       trigger_percentages, actions_json, is_active, created_at, updated_at
		FROM workflow_timeout_triggers
		WHERE tenant_id = $1
		ORDER BY workflow_name, step_name
	`

	rows, err := h.db.QueryContext(r.Context(), query, tenantID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to query triggers"})
		return
	}
	defer rows.Close()

	triggers := []TimeoutTrigger{}
	for rows.Next() {
		var t TimeoutTrigger
		var percentagesJSON []byte
		var actionsJSON []byte

		err := rows.Scan(
			&t.ID, &t.TenantID, &t.WorkflowName, &t.StepName, &t.DueHours,
			&percentagesJSON, &actionsJSON, &t.IsActive, &t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			continue
		}

		if err := json.Unmarshal(percentagesJSON, &t.TriggerPercentages); err != nil {
			t.TriggerPercentages = []int{80, 100}
		}
		if err := json.Unmarshal(actionsJSON, &t.Actions); err != nil {
			t.Actions = []TimeoutAction{}
		}

		triggers = append(triggers, t)
	}

	respondJSON(w, http.StatusOK, triggers)
}

// getTimeoutTrigger retrieves a specific timeout trigger
func (h *TimeoutTriggersHandler) getTimeoutTrigger(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	triggerId := chi.URLParam(r, "triggerId")

	query := `
		SELECT id, tenant_id, workflow_name, step_name, due_hours,
		       trigger_percentages, actions_json, is_active, created_at, updated_at
		FROM workflow_timeout_triggers
		WHERE id = $1 AND tenant_id = $2
	`

	var t TimeoutTrigger
	var percentagesJSON []byte
	var actionsJSON []byte

	err = h.db.QueryRowContext(r.Context(), query, triggerId, tenantID).Scan(
		&t.ID, &t.TenantID, &t.WorkflowName, &t.StepName, &t.DueHours,
		&percentagesJSON, &actionsJSON, &t.IsActive, &t.CreatedAt, &t.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Trigger not found"})
		return
	}
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Database error"})
		return
	}

	if err := json.Unmarshal(percentagesJSON, &t.TriggerPercentages); err != nil {
		t.TriggerPercentages = []int{80, 100}
	}
	if err := json.Unmarshal(actionsJSON, &t.Actions); err != nil {
		t.Actions = []TimeoutAction{}
	}

	respondJSON(w, http.StatusOK, t)
}

// createTimeoutTrigger creates a new timeout trigger
func (h *TimeoutTriggersHandler) createTimeoutTrigger(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	var trigger TimeoutTrigger
	if err := json.NewDecoder(r.Body).Decode(&trigger); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	trigger.TenantID = tenantID
	trigger.IsActive = true
	trigger.CreatedAt = time.Now()
	trigger.UpdatedAt = time.Now()

	if len(trigger.TriggerPercentages) == 0 {
		trigger.TriggerPercentages = []int{80, 100}
	}

	actionsJSON, _ := json.Marshal(trigger.Actions)
	percentsJSON, _ := json.Marshal(trigger.TriggerPercentages)

	query := `
		INSERT INTO workflow_timeout_triggers
		(tenant_id, workflow_name, step_name, due_hours,
		 trigger_percentages, actions_json, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`

	var id string
	err = h.db.QueryRowContext(r.Context(),
		query,
		tenantID, trigger.WorkflowName, trigger.StepName, trigger.DueHours,
		percentsJSON, actionsJSON, true, time.Now(), time.Now(),
	).Scan(&id)

	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create trigger"})
		return
	}

	trigger.ID = id
	respondJSON(w, http.StatusCreated, trigger)
}

// updateTimeoutTrigger updates an existing timeout trigger
func (h *TimeoutTriggersHandler) updateTimeoutTrigger(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	triggerId := chi.URLParam(r, "triggerId")

	var trigger TimeoutTrigger
	if err := json.NewDecoder(r.Body).Decode(&trigger); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	trigger.UpdatedAt = time.Now()

	actionsJSON, _ := json.Marshal(trigger.Actions)
	percentsJSON, _ := json.Marshal(trigger.TriggerPercentages)

	query := `
		UPDATE workflow_timeout_triggers
		SET workflow_name = $1, step_name = $2, due_hours = $3,
		    trigger_percentages = $4, actions_json = $5,
		    is_active = $6, updated_at = $7
		WHERE id = $8 AND tenant_id = $9
	`

	result, err := h.db.ExecContext(r.Context(),
		query,
		trigger.WorkflowName, trigger.StepName, trigger.DueHours,
		percentsJSON, actionsJSON, trigger.IsActive,
		time.Now(), triggerId, tenantID,
	)

	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update trigger"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Trigger not found"})
		return
	}

	trigger.ID = triggerId
	trigger.TenantID = tenantID
	respondJSON(w, http.StatusOK, trigger)
}

// deleteTimeoutTrigger soft-deletes a timeout trigger
func (h *TimeoutTriggersHandler) deleteTimeoutTrigger(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	triggerId := chi.URLParam(r, "triggerId")

	query := `
		UPDATE workflow_timeout_triggers
		SET is_active = false, updated_at = $1
		WHERE id = $2 AND tenant_id = $3
	`

	result, err := h.db.ExecContext(r.Context(), query, time.Now(), triggerId, tenantID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete trigger"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Trigger not found"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Trigger deleted successfully"})
}

// testTimeoutTrigger simulates trigger execution for testing
func (h *TimeoutTriggersHandler) testTimeoutTrigger(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	triggerId := chi.URLParam(r, "triggerId")

	query := `
		SELECT actions_json FROM workflow_timeout_triggers
		WHERE id = $1 AND tenant_id = $2
	`

	var actionsJSON []byte
	err = h.db.QueryRowContext(r.Context(), query, triggerId, tenantID).Scan(&actionsJSON)
	if err == sql.ErrNoRows {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Trigger not found"})
		return
	}
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Database error"})
		return
	}

	var actions []TimeoutAction
	if err := json.Unmarshal(actionsJSON, &actions); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Invalid trigger configuration"})
		return
	}

	// Log test execution to audit
	auditQuery := `
		INSERT INTO workflow_audit_log (workflow_id, workflow_name, step_name, action, details, created_at)
		VALUES ($1, $2, $3, 'timeout_trigger_test', $4, NOW())
	`

	details := map[string]interface{}{
		"trigger_id":   triggerId,
		"test_at":      time.Now(),
		"action_count": len(actions),
	}

	detailsJSON, _ := json.Marshal(details)
	h.db.ExecContext(r.Context(), auditQuery, triggerId, "test_workflow", "test_step", string(detailsJSON))

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Test executed successfully",
		"actions": len(actions),
		"details": actions,
	})
}

// respondJSON writes a JSON response
func respondJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

// GetTimeoutTriggersHandler returns timeout triggers handler instance (for use in main router setup)
func GetTimeoutTriggersHandler(db *sqlx.DB) *TimeoutTriggersHandler {
	return NewTimeoutTriggersHandler(db)
}
