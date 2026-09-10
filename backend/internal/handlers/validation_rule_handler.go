package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/jmoiron/sqlx"
)

// ValidationRuleHandler exposes CRUD + evaluation for validation-rule
// catalog nodes - the unified-engine replacement for the retired
// catalog_validation_rules table (see docs/validation_rules_migration_report.json).
// Mirrors PreAggregationHandler's shape: both are catalog_node-backed
// domain objects with the same CRUD/DDL-or-evaluate pattern.
type ValidationRuleHandler struct {
	svc *analytics.ValidationRuleService
	db  *sqlx.DB
}

func NewValidationRuleHandler(svc *analytics.ValidationRuleService, db *sqlx.DB) *ValidationRuleHandler {
	return &ValidationRuleHandler{svc: svc, db: db}
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
		r.Get("/violations", h.handleListViolations)
		r.Get("/bo-fields", h.handleListSemanticFields)
		r.Get("/health", h.handleRuleHealth)
		r.Patch("/{id}/active", h.handleSetActive)
	})
}

// handleListSemanticFields returns the semantic terms a rule can
// reference for the BO named by ?bo_name= - see ListSemanticFields for
// why this is semantic terms, not physical column names: a rule authored
// against a semantic term stays valid if the BO's physical binding ever
// changes, one authored directly against a column name would not.
func (h *ValidationRuleHandler) handleListSemanticFields(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	boName := r.URL.Query().Get("bo_name")
	if boName == "" {
		http.Error(w, "bo_name is required", http.StatusBadRequest)
		return
	}
	fields, err := h.svc.ListSemanticFields(r.Context(), tenantID.String(), boName)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("validation-rule-nodes/bo-fields: list failed: %v", err)
		http.Error(w, "failed to list semantic fields", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"fields": fields})
}

// handleListViolations returns the most recent persisted rule violations
// (validation_rule_violations), optionally filtered to one BO via
// ?bo_name=, so a violation is something a UI or a curl call can actually
// see rather than only a server log line.
func (h *ValidationRuleHandler) handleListViolations(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	boName := r.URL.Query().Get("bo_name")
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}
	violations, err := analytics.ListViolations(r.Context(), h.db, tenantID.String(), boName, limit)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("validation-rule-nodes/violations: list failed: %v", err)
		http.Error(w, "failed to list violations", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"violations": violations})
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

// handleListByBO returns validation rules. With ?bo_name= set, it's
// scoped to that one BO (only active rules); with it omitted, it returns
// every rule tenant-wide - the system validations page's data source
// (the spec's "unfiltered tenant-wide variant"). ?include_inactive=true
// on the tenant-wide path also returns retired rules, so the page can
// show history/the archived-corpus distinction instead of only ever
// showing the live set.
func (h *ValidationRuleHandler) handleListByBO(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	boName := r.URL.Query().Get("bo_name")
	var list []models.ValidationRuleDescriptor
	var err error
	if boName != "" {
		list, err = h.svc.ListByBO(r.Context(), tenantID.String(), boName)
	} else {
		includeInactive := r.URL.Query().Get("include_inactive") == "true"
		list, err = h.svc.ListAll(r.Context(), tenantID.String(), includeInactive)
	}
	if err != nil {
		logging.GetLogger().Sugar().Errorf("validation-rule-nodes: list failed: %v", err)
		http.Error(w, "failed to list validation rules", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"validationRules": list})
}

// handleRuleHealth returns the per-rule violation/health aggregate (see
// analytics.GetRuleHealthSummary) - the "is this rule possibly broken"
// signal the system validations page and the BO tab both read.
func (h *ValidationRuleHandler) handleRuleHealth(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	summary, err := analytics.GetRuleHealthSummary(r.Context(), h.db, tenantID.String())
	if err != nil {
		logging.GetLogger().Sugar().Errorf("validation-rule-nodes/health: aggregate failed: %v", err)
		http.Error(w, "failed to compute rule health", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"health": summary})
}

// handleSetActive flips a rule's is_active flag - the BO Validations
// tab's active toggle. Body: {"active": true|false}.
func (h *ValidationRuleHandler) handleSetActive(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var body struct {
		Active bool `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if err := h.svc.SetActive(r.Context(), tenantID.String(), id, body.Active); err != nil {
		logging.GetLogger().Sugar().Errorf("validation-rule-nodes/%s/active: set failed: %v", id, err)
		http.Error(w, "failed to update rule active state: "+err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"id": id, "active": body.Active})
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
