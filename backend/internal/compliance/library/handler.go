package library

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
)

// Handler handles HTTP requests for the compliance rule library and tenant activation matrix.
type Handler struct {
	service *Service
}

// NewHandler creates a new rule library Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers library and activation routes on the provided chi Router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/api/compliance/library", func(r chi.Router) {
		r.Get("/rules", h.HandleListCoreRules)
		r.Get("/rules/{id}", h.HandleGetRuleDetails)
		r.Get("/rulesets", h.HandleListRulesets)
	})

	r.Route("/api/compliance/tenants/{tenant_id}/activations", func(r chi.Router) {
		r.Get("/", h.HandleGetTenantActivationMatrix)
		r.Put("/rules/{rule_id}", h.HandleUpdateRuleActivation)
		r.Post("/rules/{rule_id}/repin", h.HandleRepinRule)
		r.Post("/rulesets/{ruleset_code}/toggle", h.HandleToggleRuleset)
	})
}

func (h *Handler) authorizeTenant(r *http.Request, requestedTenantID string) error {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		return nil
	}
	return jwtmiddleware.ValidateTenantAccess(claims, requestedTenantID)
}

func (h *Handler) getActor(r *http.Request) string {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims != nil && claims.Email != "" {
		return claims.Email
	}
	if userHdr := r.Header.Get("X-User-Email"); userHdr != "" {
		return userHdr
	}
	return "compliance_steward"
}

// HandleListCoreRules returns filterable Gold-Copy core library rules.
func (h *Handler) HandleListCoreRules(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()

	filter := ListRulesFilter{
		Domain:        q.Get("domain"),
		RulesetCode:   q.Get("ruleset_code"),
		LibraryStatus: q.Get("library_status"),
		RulePhase:     q.Get("rule_phase"),
		Severity:      q.Get("severity"),
		Jurisdiction:  q.Get("jurisdiction"),
		SearchQuery:   q.Get("search_query"),
	}

	rules, err := h.service.ListCoreRules(ctx, filter)
	if err != nil {
		http.Error(w, "failed to list core rules: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data":        rules,
		"total_count": len(rules),
	})
}

// HandleGetRuleDetails returns AST, parameters, timeline, and scenario vectors for a rule.
func (h *Handler) HandleGetRuleDetails(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	ruleID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid rule ID format", http.StatusBadRequest)
		return
	}

	details, err := h.service.GetRuleDetails(ctx, ruleID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get rule details: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(details)
}

// HandleListRulesets returns licensable ruleset packages and their constituent rules.
func (h *Handler) HandleListRulesets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rulesets, err := h.service.ListRulesets(ctx)
	if err != nil {
		http.Error(w, "failed to list rulesets: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data":        rulesets,
		"total_count": len(rulesets),
	})
}

// HandleGetTenantActivationMatrix returns a tenant's compliance configuration, active rules, and drift statuses.
func (h *Handler) HandleGetTenantActivationMatrix(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantIDStr := chi.URLParam(r, "tenant_id")
	if err := h.authorizeTenant(r, tenantIDStr); err != nil {
		http.Error(w, "forbidden: unauthorized tenant access", http.StatusForbidden)
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		http.Error(w, "invalid tenant ID format", http.StatusBadRequest)
		return
	}

	matrix, err := h.service.GetTenantActivationMatrix(ctx, tenantID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get tenant activation matrix: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(matrix)
}

// HandleUpdateRuleActivation updates activation toggle, inheritance mode, and parameter overrides.
func (h *Handler) HandleUpdateRuleActivation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantIDStr := chi.URLParam(r, "tenant_id")
	if err := h.authorizeTenant(r, tenantIDStr); err != nil {
		http.Error(w, "forbidden: unauthorized tenant access", http.StatusForbidden)
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		http.Error(w, "invalid tenant ID format", http.StatusBadRequest)
		return
	}

	ruleIDStr := chi.URLParam(r, "rule_id")
	ruleID, err := uuid.Parse(ruleIDStr)
	if err != nil {
		http.Error(w, "invalid rule ID format", http.StatusBadRequest)
		return
	}

	var req UpdateRuleActivationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.ActorID == "" {
		req.ActorID = h.getActor(r)
	}

	if err := h.service.UpdateTenantRuleActivation(ctx, tenantID, ruleID, req); err != nil {
		http.Error(w, "failed to update rule activation: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": "rule activation updated successfully",
	})
}

// HandleRepinRule repins an extended rule to the target core version and reconciles drift.
func (h *Handler) HandleRepinRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantIDStr := chi.URLParam(r, "tenant_id")
	if err := h.authorizeTenant(r, tenantIDStr); err != nil {
		http.Error(w, "forbidden: unauthorized tenant access", http.StatusForbidden)
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		http.Error(w, "invalid tenant ID format", http.StatusBadRequest)
		return
	}

	ruleIDStr := chi.URLParam(r, "rule_id")
	ruleID, err := uuid.Parse(ruleIDStr)
	if err != nil {
		http.Error(w, "invalid rule ID format", http.StatusBadRequest)
		return
	}

	var req RepinRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.ActorID == "" {
		req.ActorID = h.getActor(r)
	}

	if req.TargetVersion <= 0 {
		http.Error(w, "target_version must be > 0", http.StatusBadRequest)
		return
	}

	if err := h.service.RepinTenantRule(ctx, tenantID, ruleID, req); err != nil {
		http.Error(w, "failed to repin rule: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": "rule successfully repinned and drift reconciled",
	})
}

// HandleToggleRuleset bulk toggles all rules in a ruleset for a tenant.
func (h *Handler) HandleToggleRuleset(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantIDStr := chi.URLParam(r, "tenant_id")
	if err := h.authorizeTenant(r, tenantIDStr); err != nil {
		http.Error(w, "forbidden: unauthorized tenant access", http.StatusForbidden)
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		http.Error(w, "invalid tenant ID format", http.StatusBadRequest)
		return
	}

	rulesetCode := chi.URLParam(r, "ruleset_code")
	if rulesetCode == "" {
		http.Error(w, "missing ruleset_code", http.StatusBadRequest)
		return
	}

	var req ToggleRulesetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.ActorID == "" {
		req.ActorID = h.getActor(r)
	}

	if err := h.service.ToggleRulesetActivation(ctx, tenantID, rulesetCode, req.Enabled, req.ActorID); err != nil {
		http.Error(w, "failed to toggle ruleset activation: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": "ruleset activation toggled successfully",
	})
}
