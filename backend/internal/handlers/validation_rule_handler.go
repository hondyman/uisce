package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/hondyman/uisce/backend/internal/models"
)

// ValidationRuleHandler exposes CRUD + evaluation for validation-rule
// catalog nodes - the unified-engine replacement for the retired
// catalog_validation_rules table (see docs/validation_rules_migration_report.json).
// Mirrors PreAggregationHandler's shape: both are catalog_node-backed
// domain objects with the same CRUD/DDL-or-evaluate pattern.
type ValidationRuleHandler struct {
	svc *analytics.ValidationRuleService
}

func NewValidationRuleHandler(svc *analytics.ValidationRuleService) *ValidationRuleHandler {
	return &ValidationRuleHandler{svc: svc}
}

// RegisterRoutes adds the validation-rule-node routes to the given
// chi.Router. Deliberately not /validation-rules - that path is retired
// (410 Gone, see internal/api/validation_rules_routes.go) and reusing it
// would blur the two systems in logs/docs during the transition.
func (h *ValidationRuleHandler) RegisterRoutes(r chi.Router) {
	r.Route("/validation-rule-nodes", func(r chi.Router) {
		r.Post("/", h.handleUpsert)
		r.Get("/", h.handleListByBO)
		r.Get("/{id}", h.handleGetByID)
		r.Post("/{id}/evaluate", h.handleEvaluate)
	})
}

func (h *ValidationRuleHandler) handleUpsert(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	var req models.UpsertValidationRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.TenantID = tenantID.String()

	desc, err := h.svc.UpsertValidationRule(r.Context(), req)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("validation-rule-nodes: upsert failed: %v", err)
		http.Error(w, "failed to upsert validation rule: "+err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(desc)
}

func (h *ValidationRuleHandler) handleListByBO(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	boName := r.URL.Query().Get("bo_name")
	list, err := h.svc.ListByBO(r.Context(), tenantID.String(), boName)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("validation-rule-nodes: list failed: %v", err)
		http.Error(w, "failed to list validation rules", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"validationRules": list})
}

func (h *ValidationRuleHandler) handleGetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	desc, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "validation rule not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(desc)
}

// handleEvaluate runs the rule against a caller-supplied data payload via
// the unified engine (internal/rules/vm.AdvancedEvaluator) - the server-side
// counterpart to the browser wasm preview, used for manual testing from
// the editor and for the oracle-rule verification against a live DB
// constraint.
func (h *ValidationRuleHandler) handleEvaluate(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	result, err := h.svc.Evaluate(r.Context(), id, data)
	if err != nil {
		http.Error(w, "evaluation failed: "+err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"result": result})
}
