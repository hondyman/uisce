package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/ai"
	"github.com/hondyman/uisce/backend/internal/platform_intelligence/exceptions"
	"github.com/hondyman/uisce/backend/internal/platform_intelligence/health"
	"github.com/hondyman/uisce/backend/internal/platform_intelligence/optimization"
	"github.com/hondyman/uisce/backend/internal/platform_intelligence/roadmap"
)

type PlatformIntelligenceHandler struct {
	globalOptimizer     *optimization.GlobalOptimizer
	exceptionAggregator *exceptions.ExceptionAggregator
	healthScorer        *health.HealthScorer
	roadmapGenerator    *roadmap.RoadmapGenerator
	exceptionAI         *ai.ExceptionAIService
}

func NewPlatformIntelligenceHandler(
	opt *optimization.GlobalOptimizer,
	exc *exceptions.ExceptionAggregator,
	health *health.HealthScorer,
	road *roadmap.RoadmapGenerator,
) *PlatformIntelligenceHandler {
	return &PlatformIntelligenceHandler{
		globalOptimizer:     opt,
		exceptionAggregator: exc,
		healthScorer:        health,
		roadmapGenerator:    road,
	}
}

// WithExceptionAI attaches the AI-assist service (optional — endpoints that
// need it return 503 if it was never wired, e.g. in environments without an
// Anthropic key configured).
func (h *PlatformIntelligenceHandler) WithExceptionAI(svc *ai.ExceptionAIService) *PlatformIntelligenceHandler {
	h.exceptionAI = svc
	return h
}

func (h *PlatformIntelligenceHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Global Optimization
	r.Get("/optimization/proposals", h.GetOptimizationProposals)

	// Centralized Exceptions & Audits Console
	r.Get("/exceptions/all", h.GetAllExceptions)
	r.Get("/exceptions/summary", h.GetExceptionSummary)
	r.Get("/exceptions/by-type/{type}", h.GetExceptionsByType)
	r.Get("/exceptions/autofix-policy", h.GetAutofixPolicy)
	r.Put("/exceptions/autofix-policy", h.SetAutofixPolicy)
	r.Post("/exceptions/{id}/rerun", h.RerunException)
	r.Post("/exceptions/{id}/close", h.CloseException)
	r.Get("/exceptions/{id}/ai-suggestion", h.GetExceptionAISuggestion)

	// Platform Health Score
	r.Get("/health/score", h.GetHealthScore)
	r.Get("/health/trends", h.GetHealthTrends)

	// AI-Generated Roadmap
	r.Get("/roadmap/suggestions", h.GetRoadmapSuggestions)
	r.Get("/roadmap/prioritized", h.GetPrioritizedRoadmap)

	return r
}

func (h *PlatformIntelligenceHandler) GetOptimizationProposals(w http.ResponseWriter, r *http.Request) {
	proposals, _ := h.globalOptimizer.AnalyzeAndPropose(r.Context())
	json.NewEncoder(w).Encode(proposals)
}

func (h *PlatformIntelligenceHandler) GetAllExceptions(w http.ResponseWriter, r *http.Request) {
	exceptions, _ := h.exceptionAggregator.GetAllExceptions(r.Context())
	json.NewEncoder(w).Encode(exceptions)
}

func (h *PlatformIntelligenceHandler) GetExceptionSummary(w http.ResponseWriter, r *http.Request) {
	summary, _ := h.exceptionAggregator.GetSummary(r.Context())
	json.NewEncoder(w).Encode(summary)
}

func (h *PlatformIntelligenceHandler) GetExceptionsByType(w http.ResponseWriter, r *http.Request) {
	exceptionType := exceptions.ExceptionType(chi.URLParam(r, "type"))
	excs, _ := h.exceptionAggregator.GetByType(r.Context(), exceptionType)
	json.NewEncoder(w).Encode(excs)
}

// tenantIDFromRequest resolves the tenant scope for exception endpoints.
// Mirrors the header convention used elsewhere in this handler package
// (X-Tenant-ID) until a shared auth-context helper lands.
func tenantIDFromRequest(r *http.Request) uuid.UUID {
	tid, err := uuid.Parse(r.Header.Get("X-Tenant-ID"))
	if err != nil {
		return uuid.Nil
	}
	return tid
}

func userIDFromRequest(r *http.Request) *uuid.UUID {
	uid, err := uuid.Parse(r.Header.Get("X-User-ID"))
	if err != nil {
		return nil
	}
	return &uid
}

// GetAutofixPolicy lists the tenant's per-exception-type autofix toggles.
func (h *PlatformIntelligenceHandler) GetAutofixPolicy(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantIDFromRequest(r)
	policies, err := h.exceptionAggregator.ListAutofixPolicies(r.Context(), tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(policies)
}

// SetAutofixPolicy sets a single (tenant, exception_type[, user]) toggle.
// There is intentionally no bulk/global-enable path here.
func (h *PlatformIntelligenceHandler) SetAutofixPolicy(w http.ResponseWriter, r *http.Request) {
	var policy exceptions.AutofixPolicy
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if policy.TenantID == uuid.Nil {
		policy.TenantID = tenantIDFromRequest(r)
	}
	if policy.UpdatedBy == "" {
		policy.UpdatedBy = r.Header.Get("X-User-ID")
	}
	if err := h.exceptionAggregator.SetAutofixPolicy(r.Context(), policy); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// RerunException triggers a manual re-verification of an exception (the
// same verify-before-close check the remediation workflow runs). Wiring to
// the actual per-type verify functions lives in
// backend/temporal/activities/exceptionRemediationActivities.ts; this
// endpoint just flags the exception auto_fix_pending so the workflow (or an
// operator) can pick it up.
func (h *PlatformIntelligenceHandler) RerunException(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.exceptionAggregator.AppendAutofixAttempt(r.Context(), id, exceptions.AutofixAttempt{
		Action: "manual_rerun_requested",
	}, exceptions.StatusAutoFixPending); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

// CloseException marks an exception resolved by a human operator.
func (h *PlatformIntelligenceHandler) CloseException(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	resolvedBy := r.Header.Get("X-User-ID")
	if resolvedBy == "" {
		resolvedBy = "unknown"
	}
	if err := h.exceptionAggregator.Close(r.Context(), id, exceptions.CloseExceptionOptions{
		Status:     exceptions.StatusClosed,
		ResolvedBy: resolvedBy,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GetExceptionAISuggestion returns an AI-generated root-cause explanation
// and fix suggestion for one exception.
func (h *PlatformIntelligenceHandler) GetExceptionAISuggestion(w http.ResponseWriter, r *http.Request) {
	if h.exceptionAI == nil {
		http.Error(w, "exception AI assist not configured", http.StatusServiceUnavailable)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	exc, err := h.exceptionAggregator.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	suggestion, err := h.exceptionAI.SuggestFix(r.Context(), *exc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"suggestion": suggestion})
}

func (h *PlatformIntelligenceHandler) GetHealthScore(w http.ResponseWriter, r *http.Request) {
	score, _ := h.healthScorer.CalculateScore(r.Context())
	json.NewEncoder(w).Encode(score)
}

func (h *PlatformIntelligenceHandler) GetHealthTrends(w http.ResponseWriter, r *http.Request) {
	trends, _ := h.healthScorer.GetTrends(r.Context(), 30)
	json.NewEncoder(w).Encode(trends)
}

func (h *PlatformIntelligenceHandler) GetRoadmapSuggestions(w http.ResponseWriter, r *http.Request) {
	items, _ := h.roadmapGenerator.GenerateRoadmap(r.Context())
	json.NewEncoder(w).Encode(items)
}

func (h *PlatformIntelligenceHandler) GetPrioritizedRoadmap(w http.ResponseWriter, r *http.Request) {
	items, _ := h.roadmapGenerator.GenerateRoadmap(r.Context())
	prioritized, _ := h.roadmapGenerator.PrioritizeRoadmap(r.Context(), items)
	json.NewEncoder(w).Encode(prioritized)
}
