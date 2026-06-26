package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/platform_intelligence/exceptions"
	"github.com/hondyman/semlayer/backend/internal/platform_intelligence/health"
	"github.com/hondyman/semlayer/backend/internal/platform_intelligence/optimization"
	"github.com/hondyman/semlayer/backend/internal/platform_intelligence/roadmap"
)

type PlatformIntelligenceHandler struct {
	globalOptimizer     *optimization.GlobalOptimizer
	exceptionAggregator *exceptions.ExceptionAggregator
	healthScorer        *health.HealthScorer
	roadmapGenerator    *roadmap.RoadmapGenerator
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

func (h *PlatformIntelligenceHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Global Optimization
	r.Get("/optimization/proposals", h.GetOptimizationProposals)

	// Centralized Exceptions & Audits Console
	r.Get("/exceptions/all", h.GetAllExceptions)
	r.Get("/exceptions/summary", h.GetExceptionSummary)
	r.Get("/exceptions/by-type/{type}", h.GetExceptionsByType)

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
