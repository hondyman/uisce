package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/identity"
)

// ============================================================================
// TIMEOUT TRIGGERS HTTP HANDLERS
// ============================================================================
// Provides REST API endpoints for managing workflow step timeout triggers.
// Endpoints:
//   POST   /api/admin/timeout-triggers        - Create new timeout trigger
//   GET    /api/admin/timeout-triggers        - List triggers (optional filter by workflow)
//   GET    /api/admin/timeout-triggers/:id    - Get specific trigger
//   PUT    /api/admin/timeout-triggers/:id    - Update trigger
//   DELETE /api/admin/timeout-triggers/:id    - Delete trigger
// ============================================================================

type TimeoutTriggersHandler struct {
	db *sql.DB
}

// NewTimeoutTriggersHandler creates a new timeout triggers handler
func NewTimeoutTriggersHandler(db *sql.DB) *TimeoutTriggersHandler {
	if db == nil {
		log.Fatal("[TimeoutTriggersHandler] Database connection is nil")
	}
	return &TimeoutTriggersHandler{db: db}
}

// TimeoutTrigger represents a timeout trigger rule
type TimeoutTrigger struct {
	ID           string          `json:"id"`
	TenantID     string          `json:"tenant_id"`
	WorkflowName string          `json:"workflow_name"`
	StepName     string          `json:"step_name"`
	DueHours     int             `json:"due_hours"`
	ActionsJSON  json.RawMessage `json:"actions_json"`
	IsActive     bool            `json:"is_active"`
	CreatedAt    string          `json:"created_at"`
	UpdatedAt    string          `json:"updated_at"`
}

// CreateTimeoutTriggerRequest is the request body for creating a timeout trigger
type CreateTimeoutTriggerRequest struct {
	WorkflowName string          `json:"workflow_name" binding:"required"`
	StepName     string          `json:"step_name" binding:"required"`
	DueHours     int             `json:"due_hours" binding:"required,min=1"`
	ActionsJSON  json.RawMessage `json:"actions_json" binding:"required"`
}

// HandleCreateTimeoutTrigger creates a new timeout trigger
// POST /api/admin/timeout-triggers
func (h *TimeoutTriggersHandler) HandleCreateTimeoutTrigger(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract tenant ID from context
	tenantID := getTenantIDFromContext(ctx)
	if tenantID == "" {
		http.Error(w, "Tenant not found in context", http.StatusBadRequest)
		return
	}

	// Check RBAC: Ensure user has temporal.admin role
	if !hasRole(ctx, "temporal.admin") {
		http.Error(w, "Forbidden: temporal.admin role required", http.StatusForbidden)
		return
	}

	// Parse request
	var req CreateTimeoutTriggerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate actions JSON is valid JSON array
	var actions []interface{}
	if err := json.Unmarshal(req.ActionsJSON, &actions); err != nil {
		http.Error(w, "Invalid actions_json: must be valid JSON array", http.StatusBadRequest)
		return
	}

	// Generate ID
	triggerID := uuid.New().String()

	// Insert into database
	query := `
		INSERT INTO workflow_timeout_triggers 
		  (id, tenant_id, workflow_name, step_name, due_hours, actions_json, is_active, created_at, updated_at)
		VALUES 
		  ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING id, tenant_id, workflow_name, step_name, due_hours, actions_json, is_active, created_at, updated_at
	`

	var trigger TimeoutTrigger
	err := h.db.QueryRowContext(
		ctx,
		query,
		triggerID,
		tenantID,
		req.WorkflowName,
		req.StepName,
		req.DueHours,
		req.ActionsJSON,
		true, // is_active
	).Scan(
		&trigger.ID,
		&trigger.TenantID,
		&trigger.WorkflowName,
		&trigger.StepName,
		&trigger.DueHours,
		&trigger.ActionsJSON,
		&trigger.IsActive,
		&trigger.CreatedAt,
		&trigger.UpdatedAt,
	)

	if err != nil {
		log.Printf("[TimeoutTriggersHandler] Error creating trigger: %v", err)
		http.Error(w, "Failed to create timeout trigger", http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(trigger)

	log.Printf("[TimeoutTriggersHandler] Created timeout trigger: %s.%s (id=%s)", req.WorkflowName, req.StepName, triggerID)
}

// HandleListTimeoutTriggers lists all timeout triggers (optionally filtered by workflow)
// GET /api/admin/timeout-triggers?workflow=HireEmployee
func (h *TimeoutTriggersHandler) HandleListTimeoutTriggers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract tenant ID
	tenantID := getTenantIDFromContext(ctx)
	if tenantID == "" {
		http.Error(w, "Tenant not found in context", http.StatusBadRequest)
		return
	}

	// Optional: filter by workflow
	workflow := r.URL.Query().Get("workflow")

	// Build query
	var query string
	var args []interface{}

	if workflow != "" {
		query = `
			SELECT id, tenant_id, workflow_name, step_name, due_hours, actions_json, is_active, created_at, updated_at
			FROM workflow_timeout_triggers
			WHERE tenant_id = $1 AND workflow_name = $2 AND is_active = TRUE
			ORDER BY workflow_name, step_name
		`
		args = []interface{}{tenantID, workflow}
	} else {
		query = `
			SELECT id, tenant_id, workflow_name, step_name, due_hours, actions_json, is_active, created_at, updated_at
			FROM workflow_timeout_triggers
			WHERE tenant_id = $1 AND is_active = TRUE
			ORDER BY workflow_name, step_name
		`
		args = []interface{}{tenantID}
	}

	// Execute query
	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		log.Printf("[TimeoutTriggersHandler] Error querying triggers: %v", err)
		http.Error(w, "Failed to list timeout triggers", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Scan results
	var triggers []TimeoutTrigger
	for rows.Next() {
		var trigger TimeoutTrigger
		if err := rows.Scan(
			&trigger.ID,
			&trigger.TenantID,
			&trigger.WorkflowName,
			&trigger.StepName,
			&trigger.DueHours,
			&trigger.ActionsJSON,
			&trigger.IsActive,
			&trigger.CreatedAt,
			&trigger.UpdatedAt,
		); err != nil {
			log.Printf("[TimeoutTriggersHandler] Error scanning trigger: %v", err)
			continue
		}
		triggers = append(triggers, trigger)
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if triggers == nil {
		triggers = []TimeoutTrigger{} // Return empty array, not null
	}
	json.NewEncoder(w).Encode(triggers)

	log.Printf("[TimeoutTriggersHandler] Listed %d timeout triggers for tenant %s", len(triggers), tenantID)
}

// HandleGetTimeoutTrigger retrieves a specific timeout trigger
// GET /api/admin/timeout-triggers/:id
func (h *TimeoutTriggersHandler) HandleGetTimeoutTrigger(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract tenant ID
	tenantID := getTenantIDFromContext(ctx)
	if tenantID == "" {
		http.Error(w, "Tenant not found in context", http.StatusBadRequest)
		return
	}

	// Get trigger ID from URL
	triggerID := chi.URLParam(r, "id")
	if triggerID == "" {
		http.Error(w, "Trigger ID required", http.StatusBadRequest)
		return
	}

	// Query trigger
	query := `
		SELECT id, tenant_id, workflow_name, step_name, due_hours, actions_json, is_active, created_at, updated_at
		FROM workflow_timeout_triggers
		WHERE id = $1 AND tenant_id = $2
	`

	var trigger TimeoutTrigger
	err := h.db.QueryRowContext(ctx, query, triggerID, tenantID).Scan(
		&trigger.ID,
		&trigger.TenantID,
		&trigger.WorkflowName,
		&trigger.StepName,
		&trigger.DueHours,
		&trigger.ActionsJSON,
		&trigger.IsActive,
		&trigger.CreatedAt,
		&trigger.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "Timeout trigger not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("[TimeoutTriggersHandler] Error querying trigger: %v", err)
		http.Error(w, "Failed to get timeout trigger", http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(trigger)
}

// HandleUpdateTimeoutTrigger updates an existing timeout trigger
// PUT /api/admin/timeout-triggers/:id
func (h *TimeoutTriggersHandler) HandleUpdateTimeoutTrigger(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract tenant ID
	tenantID := getTenantIDFromContext(ctx)
	if tenantID == "" {
		http.Error(w, "Tenant not found in context", http.StatusBadRequest)
		return
	}

	// Check RBAC
	if !hasRole(ctx, "temporal.admin") {
		http.Error(w, "Forbidden: temporal.admin role required", http.StatusForbidden)
		return
	}

	// Get trigger ID
	triggerID := chi.URLParam(r, "id")
	if triggerID == "" {
		http.Error(w, "Trigger ID required", http.StatusBadRequest)
		return
	}

	// Parse request
	var req CreateTimeoutTriggerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update in database
	query := `
		UPDATE workflow_timeout_triggers
		SET workflow_name = $1, step_name = $2, due_hours = $3, actions_json = $4, updated_at = NOW()
		WHERE id = $5 AND tenant_id = $6
		RETURNING id, tenant_id, workflow_name, step_name, due_hours, actions_json, is_active, created_at, updated_at
	`

	var trigger TimeoutTrigger
	err := h.db.QueryRowContext(
		ctx,
		query,
		req.WorkflowName,
		req.StepName,
		req.DueHours,
		req.ActionsJSON,
		triggerID,
		tenantID,
	).Scan(
		&trigger.ID,
		&trigger.TenantID,
		&trigger.WorkflowName,
		&trigger.StepName,
		&trigger.DueHours,
		&trigger.ActionsJSON,
		&trigger.IsActive,
		&trigger.CreatedAt,
		&trigger.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "Timeout trigger not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("[TimeoutTriggersHandler] Error updating trigger: %v", err)
		http.Error(w, "Failed to update timeout trigger", http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(trigger)

	log.Printf("[TimeoutTriggersHandler] Updated timeout trigger: %s", triggerID)
}

// HandleDeleteTimeoutTrigger deletes a timeout trigger (soft delete - sets is_active=false)
// DELETE /api/admin/timeout-triggers/:id
func (h *TimeoutTriggersHandler) HandleDeleteTimeoutTrigger(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract tenant ID
	tenantID := getTenantIDFromContext(ctx)
	if tenantID == "" {
		http.Error(w, "Tenant not found in context", http.StatusBadRequest)
		return
	}

	// Check RBAC
	if !hasRole(ctx, "temporal.admin") {
		http.Error(w, "Forbidden: temporal.admin role required", http.StatusForbidden)
		return
	}

	// Get trigger ID
	triggerID := chi.URLParam(r, "id")
	if triggerID == "" {
		http.Error(w, "Trigger ID required", http.StatusBadRequest)
		return
	}

	// Soft delete (set is_active=false)
	query := `
		UPDATE workflow_timeout_triggers
		SET is_active = FALSE, updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2
	`

	result, err := h.db.ExecContext(ctx, query, triggerID, tenantID)
	if err != nil {
		log.Printf("[TimeoutTriggersHandler] Error deleting trigger: %v", err)
		http.Error(w, "Failed to delete timeout trigger", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		http.Error(w, "Timeout trigger not found", http.StatusNotFound)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})

	log.Printf("[TimeoutTriggersHandler] Deleted timeout trigger: %s", triggerID)
}

// RegisterTimeoutTriggersRoutes registers all timeout trigger routes
// Usage in api.go:
//
//	handler := NewTimeoutTriggersHandler(db)
//	RegisterTimeoutTriggersRoutes(r, handler)
func RegisterTimeoutTriggersRoutes(r *chi.Mux, handler *TimeoutTriggersHandler) {
	r.Route("/api/admin/timeout-triggers", func(r chi.Router) {
		r.Post("/", handler.HandleCreateTimeoutTrigger)       // POST /api/admin/timeout-triggers
		r.Get("/", handler.HandleListTimeoutTriggers)         // GET /api/admin/timeout-triggers
		r.Get("/{id}", handler.HandleGetTimeoutTrigger)       // GET /api/admin/timeout-triggers/:id
		r.Put("/{id}", handler.HandleUpdateTimeoutTrigger)    // PUT /api/admin/timeout-triggers/:id
		r.Delete("/{id}", handler.HandleDeleteTimeoutTrigger) // DELETE /api/admin/timeout-triggers/:id
	})
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// getTenantIDFromContext extracts tenant ID from context
func getTenantIDFromContext(ctx context.Context) string {
	// Try from header first
	tenantID := ctx.Value("tenant_id")
	if tenantID != nil {
		return tenantID.(string)
	}

	// Try from identity context
	identityCtx, ok := ctx.Value("identity").(identity.IdentityContext)
	if ok && identityCtx.TenantID != "" {
		return identityCtx.TenantID
	}

	return ""
}

// hasRole checks if the current user has a specific role
func hasRole(ctx context.Context, role string) bool {
	identityCtx, ok := ctx.Value("identity").(identity.IdentityContext)
	if !ok {
		return false
	}

	for _, r := range identityCtx.Roles {
		if r == role {
			return true
		}
	}
	return false
}
