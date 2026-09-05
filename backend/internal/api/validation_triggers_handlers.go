package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/identity"
	"github.com/hondyman/uisce/backend/internal/validation"
	"github.com/hondyman/uisce/libs/jwt-middleware"
)

// ValidationTriggersHandler handles validation trigger requests
type ValidationTriggersHandler struct {
	triggerEngine *validation.TriggerValidationEngine
	db            *sql.DB
}

// NewValidationTriggersHandler creates a new handler
func NewValidationTriggersHandler(db *sql.DB, triggerEngine *validation.TriggerValidationEngine) *ValidationTriggersHandler {
	if triggerEngine == nil {
		triggerEngine = validation.NewTriggerValidationEngine(db, &validation.SimpleLogger{})
	}
	return &ValidationTriggersHandler{
		triggerEngine: triggerEngine,
		db:            db,
	}
}

// HandleValidateField validates a single field and returns pass/fail for quick client-side feedback.
// POST /api/validate/field
// Body: { entity: "orders", field: "total", value: 100, record: {...} }
// Response: { status: "pass" } or { error: "error message" }
func (h *ValidationTriggersHandler) HandleValidateField(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header required", http.StatusBadRequest)
		return
	}

	var req struct {
		Entity string                 `json:"entity"`
		Field  string                 `json:"field"`
		Value  interface{}            `json:"value"`
		Record map[string]interface{} `json:"record"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Entity == "" || req.Field == "" {
		http.Error(w, "entity and field are required", http.StatusBadRequest)
		return
	}

	tid, err := uuid.Parse(tenantID)
	if err != nil {
		http.Error(w, "invalid tenant ID", http.StatusBadRequest)
		return
	}

	// Perform field validation
	err = h.triggerEngine.ValidateField(ctx, tid, req.Entity, req.Field, req.Value)

	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "pass"})
}

// HandleTriggerValidation is a middleware/helper that enforces trigger validation for create/save/delete.
// Call this from your handlers before committing DB changes.
// Parameters:
//   - triggerType: "create", "save", "delete", "field_change", "workflow_step", etc.
//   - entity: target entity name (e.g. "orders", "customers")
//   - stepName: optional, for workflow_step triggers
//   - data: the full payload being saved
//
// Returns an error if validation fails, nil on pass.
func (h *ValidationTriggersHandler) TriggerValidate(ctx context.Context, tenantID uuid.UUID, triggerType, entity, stepName string, data map[string]interface{}) error {
	return h.triggerEngine.TriggerValidate(ctx, tenantID, triggerType, entity, stepName, data)
}

// HandleListTriggers lists all validation triggers for a tenant/entity (for admin/debug)
// GET /api/admin/validation-triggers?entity=orders
func (h *ValidationTriggersHandler) HandleListTriggers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header required", http.StatusBadRequest)
		return
	}

	// RBAC: require admin permission
	actorID, ok := identity.ActorIDFromContext(ctx)
	if !ok || actorID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	entity := r.URL.Query().Get("entity")
	if entity == "" {
		http.Error(w, "entity query param required", http.StatusBadRequest)
		return
	}

	q := `
    SELECT vt.id, vt.tenant_id, vt.trigger_type, vt.target_entity, vt.step_name, vt.rule_ids,
           vt.is_active,
           lr.last_fired_at
    FROM validation_triggers vt
    LEFT JOIN LATERAL (
        SELECT MAX(r.start_time) AS last_fired_at
        FROM data_pipeline_runs r
        WHERE r.trigger_id = vt.id
    ) lr ON true
    WHERE vt.tenant_id = $1 AND vt.target_entity = $2
    ORDER BY vt.created_at DESC
  `

	rows, err := h.db.QueryContext(ctx, q, tenantID, entity)
	if err != nil {
		log.Printf("HandleListTriggers: query error: %v", err)
		http.Error(w, "query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var triggers []map[string]interface{}
	for rows.Next() {
		var id, tid, triggerType, targetEntity string
		var stepName sql.NullString
		var ruleIDsJSON []byte
		var isActive bool
		var lastFiredAt sql.NullTime

		if err := rows.Scan(&id, &tid, &triggerType, &targetEntity, &stepName, &ruleIDsJSON, &isActive, &lastFiredAt); err != nil {
			log.Printf("HandleListTriggers: scan error: %v", err)
			continue
		}

		t := map[string]interface{}{
			"id":            id,
			"tenant_id":     tid,
			"trigger_type":  triggerType,
			"target_entity": targetEntity,
			"step_name":     stepName.String,
			"rule_ids":      string(ruleIDsJSON),
			"is_active":     isActive,
		}
		if lastFiredAt.Valid {
			t["last_fired_at"] = lastFiredAt.Time
		}
		triggers = append(triggers, t)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(triggers)
}

// HandleCreateTrigger creates a new validation trigger (admin endpoint).
// POST /api/admin/validation-triggers
// Body: { trigger_type: "save", target_entity: "orders", rule_ids: [...] }
func (h *ValidationTriggersHandler) HandleCreateTrigger(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header required", http.StatusBadRequest)
		return
	}

	actorID, ok := identity.ActorIDFromContext(ctx)
	if !ok || actorID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		TriggerType   string   `json:"trigger_type"`
		TargetEntity string   `json:"target_entity"`
		StepName     *string  `json:"step_name"`
		RuleIDs      []string `json:"rule_ids"`
		PipelineID   *string  `json:"pipeline_id"`
		DispatchMode string   `json:"dispatch_mode"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.TriggerType == "" || req.TargetEntity == "" || len(req.RuleIDs) == 0 {
		http.Error(w, "trigger_type, target_entity, and rule_ids are required", http.StatusBadRequest)
		return
	}

	if req.DispatchMode == "" {
		req.DispatchMode = "sync"
	}
	if req.DispatchMode != "sync" && req.DispatchMode != "async" {
		http.Error(w, "dispatch_mode must be 'sync' or 'async'", http.StatusBadRequest)
		return
	}

	triggerID := uuid.New().String()

	ruleIDsJSON, err := json.Marshal(req.RuleIDs)
	if err != nil {
		http.Error(w, "invalid rule_ids", http.StatusBadRequest)
		return
	}

	var pipelineID *uuid.UUID
	if req.PipelineID != nil && *req.PipelineID != "" {
		parsed, err := uuid.Parse(*req.PipelineID)
		if err != nil {
			http.Error(w, "invalid pipeline_id", http.StatusBadRequest)
			return
		}
		pipelineID = &parsed
	}

	q := `
    INSERT INTO validation_triggers (id, tenant_id, trigger_type, target_entity, step_name, rule_ids, pipeline_id, dispatch_mode, created_by, created_at)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
  `

	_, err = h.db.ExecContext(ctx, q, triggerID, tenantID, req.TriggerType, req.TargetEntity, req.StepName, ruleIDsJSON, pipelineID, req.DispatchMode, actorID)
	if err != nil {
		log.Printf("HandleCreateTrigger: exec error: %v", err)
		http.Error(w, "create failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": triggerID, "status": "created"})
}

// RegisterValidationTriggersRoutes registers trigger-related routes
func RegisterValidationTriggersRoutes(r chi.Router, db *sql.DB, triggerEngine *validation.TriggerValidationEngine) {
	handler := NewValidationTriggersHandler(db, triggerEngine)

	r.Route("/validate", func(r chi.Router) {
		r.Post("/field", handler.HandleValidateField)
	})

	r.Route("/admin/validation-triggers", func(r chi.Router) {
		r.Get("/", handler.HandleListTriggers)
		r.Post("/", handler.HandleCreateTrigger)
	})
}
