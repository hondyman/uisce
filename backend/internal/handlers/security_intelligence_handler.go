package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/security_intelligence/entitlements"
	"github.com/hondyman/semlayer/backend/internal/security_intelligence/policies"
	"github.com/hondyman/semlayer/backend/internal/security_intelligence/segmentation"
	"github.com/hondyman/semlayer/backend/internal/security_intelligence/threats"
)

type SecurityIntelligenceHandler struct {
	entitlementAnalyzer *entitlements.EntitlementAnalyzer
	threatDetector      *threats.ThreatDetector
	policyGenerator     *policies.PolicyGenerator
	tenantSegmenter     *segmentation.TenantSegmenter
}

func NewSecurityIntelligenceHandler(
	ent *entitlements.EntitlementAnalyzer,
	thr *threats.ThreatDetector,
	pol *policies.PolicyGenerator,
	seg *segmentation.TenantSegmenter,
) *SecurityIntelligenceHandler {
	return &SecurityIntelligenceHandler{
		entitlementAnalyzer: ent,
		threatDetector:      thr,
		policyGenerator:     pol,
		tenantSegmenter:     seg,
	}
}

func (h *SecurityIntelligenceHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Entitlement Analysis
	r.Get("/entitlements/analyze/{tenantId}", h.AnalyzeEntitlements)

	// Threat Detection
	r.Get("/threats/active", h.GetActiveThreats)

	// Policy Generation
	r.Get("/policies/suggestions", h.GetPolicySuggestions)

	// Tenant Segmentation
	r.Get("/segmentation/clusters", h.GetTenantClusters)
	r.Get("/segmentation/recommend/{tenantId}", h.RecommendPolicies)

	return r
}

func (h *SecurityIntelligenceHandler) AnalyzeEntitlements(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	report, _ := h.entitlementAnalyzer.Analyze(r.Context(), tenantID)
	json.NewEncoder(w).Encode(report)
}

func (h *SecurityIntelligenceHandler) GetActiveThreats(w http.ResponseWriter, r *http.Request) {
	threats, _ := h.threatDetector.DetectThreats(r.Context())
	json.NewEncoder(w).Encode(threats)
}

func (h *SecurityIntelligenceHandler) GetPolicySuggestions(w http.ResponseWriter, r *http.Request) {
	suggestions, _ := h.policyGenerator.Suggest(r.Context())
	json.NewEncoder(w).Encode(suggestions)
}

func (h *SecurityIntelligenceHandler) GetTenantClusters(w http.ResponseWriter, r *http.Request) {
	clusters, _ := h.tenantSegmenter.Segment(r.Context())
	json.NewEncoder(w).Encode(clusters)
}

func (h *SecurityIntelligenceHandler) RecommendPolicies(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	policyBundle, _ := h.tenantSegmenter.RecommendPolicies(r.Context(), tenantID)

	response := map[string]string{
		"tenant_id":     tenantID,
		"policy_bundle": policyBundle,
	}
	json.NewEncoder(w).Encode(response)
}
