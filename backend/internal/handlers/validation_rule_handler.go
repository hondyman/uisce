package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

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
		r.Get("/health", h.handleGetHealth)
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

// handleGetHealth exposes analytics.GetRuleHealthSummary - a fully built
// per-rule violation/error aggregate that had no route wired to it at all
// (the BO-scoped Validations & Triggers tab called GET .../health, which
// with no literal route registered fell through to the /{id} handler and
// 400'd trying to uuid.Parse("health")).
func (h *ValidationRuleHandler) handleGetHealth(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	health, err := analytics.GetRuleHealthSummary(r.Context(), h.db, tenantID.String())
	if err != nil {
		logging.GetLogger().Sugar().Errorf("validation-rule-nodes/health: aggregate failed: %v", err)
		http.Error(w, "failed to compute rule health", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"health": health})
}

// handleSetActive toggles a validation rule's catalog_node.is_active - the
// Validations & Triggers tab's per-rule Switch called PATCH .../{id}/active
// with no handler behind it at all.
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
	var req struct {
		Active bool `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	res, err := h.db.ExecContext(r.Context(), `
		UPDATE catalog_node SET is_active = $1, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3
	`, req.Active, id, tenantID)
	if err != nil {
		http.Error(w, "failed to update rule: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		http.Error(w, "validation rule not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
	domain := r.URL.Query().Get("domain")
	list, err := h.svc.ListByBO(r.Context(), tenantID.String(), boName, domain)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("validation-rule-nodes: list failed: %v", err)
		http.Error(w, "failed to list validation rules", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"validationRules": list})
}

func (h *ValidationRuleHandler) handleGetByID(w http.ResponseWriter, r *http.Request) {
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
	// Tenant-scoped: another tenant's rule is indistinguishable from a missing one.
	desc, err := h.svc.GetByIDForTenant(r.Context(), tenantID.String(), id)
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
	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	result, err := h.svc.EvaluateForTenant(r.Context(), tenantID.String(), id, data)
	if err != nil {
		// A rule the tenant may not see is reported as not found, like a missing one.
		if strings.Contains(err.Error(), "validation rule not found") {
			http.Error(w, "validation rule not found", http.StatusNotFound)
			return
		}
		http.Error(w, "evaluation failed: "+err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"result": result})
}
