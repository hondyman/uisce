package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/tenant_intelligence/behavior"
	"github.com/hondyman/semlayer/backend/internal/tenant_intelligence/compliance"
	"github.com/hondyman/semlayer/backend/internal/tenant_intelligence/performance"
	"github.com/hondyman/semlayer/backend/internal/tenant_intelligence/ux"
)

type TenantIntelligenceHandler struct {
	behaviorModeler    *behavior.BehaviorModeler
	tenantOptimizer    *performance.TenantOptimizer
	uxPersonalizer     *ux.UXPersonalizer
	complianceProfiler *compliance.ComplianceProfiler
}

func NewTenantIntelligenceHandler(
	beh *behavior.BehaviorModeler,
	opt *performance.TenantOptimizer,
	uxp *ux.UXPersonalizer,
	comp *compliance.ComplianceProfiler,
) *TenantIntelligenceHandler {
	return &TenantIntelligenceHandler{
		behaviorModeler:    beh,
		tenantOptimizer:    opt,
		uxPersonalizer:     uxp,
		complianceProfiler: comp,
	}
}

func (h *TenantIntelligenceHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Behavior Modeling
	r.Get("/behavior/{tenantId}", h.GetBehaviorModel)
	r.Get("/suggestions/{tenantId}", h.GetSuggestions)

	// Performance Optimization
	r.Get("/performance/{tenantId}", h.GetOptimizations)

	// UX Personalization
	r.Get("/ux/{tenantId}", h.GetPersonalizations)

	// Compliance Profiling
	r.Get("/compliance/{tenantId}", h.GetComplianceProfile)

	return r
}

func (h *TenantIntelligenceHandler) GetBehaviorModel(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	model, _ := h.behaviorModeler.Model(r.Context(), tenantID)
	json.NewEncoder(w).Encode(model)
}

func (h *TenantIntelligenceHandler) GetSuggestions(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	model, _ := h.behaviorModeler.Model(r.Context(), tenantID)
	suggestions, _ := h.behaviorModeler.Suggest(r.Context(), model)
	json.NewEncoder(w).Encode(suggestions)
}

func (h *TenantIntelligenceHandler) GetOptimizations(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	strategies, _ := h.tenantOptimizer.Optimize(r.Context(), tenantID)
	json.NewEncoder(w).Encode(strategies)
}

func (h *TenantIntelligenceHandler) GetPersonalizations(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	personalizations, _ := h.uxPersonalizer.Personalize(r.Context(), tenantID)
	json.NewEncoder(w).Encode(personalizations)
}

func (h *TenantIntelligenceHandler) GetComplianceProfile(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	profile, _ := h.complianceProfiler.Profile(r.Context(), tenantID)
	json.NewEncoder(w).Encode(profile)
}
