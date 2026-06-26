package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/analytics"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// TelemetryHandler handles telemetry ingestion.
type TelemetryHandler struct {
	service *analytics.TelemetryService
}

func NewTelemetryHandler(service *analytics.TelemetryService) *TelemetryHandler {
	return &TelemetryHandler{service: service}
}

func (h *TelemetryHandler) RegisterRoutes(r chi.Router) {
	r.Post("/api/telemetry/query", h.IngestQuery)
}

// IngestQuery ingests a query telemetry event.
func (h *TelemetryHandler) IngestQuery(w http.ResponseWriter, r *http.Request) {
	var req models.TelemetryIngestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.TenantID == "" || req.BOName == "" {
		http.Error(w, "tenant_id and bo_name required", http.StatusBadRequest)
		return
	}

	if err := h.service.Ingest(r.Context(), req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// --- Advisor Handler ---

// AdvisorHandler provides performance advisor endpoints.
type AdvisorHandler struct {
	analyzer      *analytics.WorkloadAnalyzer
	recommender   *analytics.PreAggRecommendationEngine
	preAggService *analytics.PreAggregationService
}

func NewAdvisorHandler(
	analyzer *analytics.WorkloadAnalyzer,
	recommender *analytics.PreAggRecommendationEngine,
	preAggService *analytics.PreAggregationService,
) *AdvisorHandler {
	return &AdvisorHandler{
		analyzer:      analyzer,
		recommender:   recommender,
		preAggService: preAggService,
	}
}

func (h *AdvisorHandler) RegisterRoutes(r chi.Router) {
	r.Get("/api/bo/{boName}/advisor", h.GetBOAdvisor)
	r.Get("/api/advisor/global", h.GetGlobalAdvisor)
}

// GetBOAdvisor returns workload profile and recommendations for a BO.
func (h *AdvisorHandler) GetBOAdvisor(w http.ResponseWriter, r *http.Request) {
	boName := chi.URLParam(r, "boName")
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = jwtmiddleware.GetClaimsFromContext(r).TenantID
	}
	if tenantID == "" || boName == "" {
		http.Error(w, "tenant_id and boName required", http.StatusBadRequest)
		return
	}

	windowDays := 7
	if wd := r.URL.Query().Get("window_days"); wd != "" {
		if parsed, err := strconv.Atoi(wd); err == nil && parsed > 0 {
			windowDays = parsed
		}
	}
	window := time.Duration(windowDays*24) * time.Hour

	// Get workload profile
	workload, err := h.analyzer.AnalyzeBO(r.Context(), tenantID, boName, window)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get recommendations
	recs, err := h.recommender.RecommendForBO(r.Context(), tenantID, boName, window)
	if err != nil {
		recs = []models.PreAggRecommendation{}
	}

	// Get existing pre-aggs
	existing, _ := h.preAggService.ListByBO(r.Context(), tenantID, boName)

	resp := models.BOAdvisorResponse{
		Workload:                workload,
		Recommendations:         recs,
		ExistingPreAggregations: existing,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GetGlobalAdvisor returns top recommendations across all BOs.
func (h *AdvisorHandler) GetGlobalAdvisor(w http.ResponseWriter, r *http.Request) {
	windowDays := 7
	if wd := r.URL.Query().Get("window_days"); wd != "" {
		if parsed, err := strconv.Atoi(wd); err == nil && parsed > 0 {
			windowDays = parsed
		}
	}
	window := time.Duration(windowDays*24) * time.Hour

	recs, err := h.recommender.RecommendGlobal(r.Context(), window)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Limit to top 50
	if len(recs) > 50 {
		recs = recs[:50]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recs)
}
