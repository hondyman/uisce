package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/services"
)

// MetricRegistryHandler handles metric registry and orchestration endpoints
type MetricRegistryHandler struct {
	service *services.MetricRegistryService
}

// NewMetricRegistryHandler creates a new handler
func NewMetricRegistryHandler(service *services.MetricRegistryService) *MetricRegistryHandler {
	return &MetricRegistryHandler{service: service}
}

// RegisterRoutes registers all metric registry routes
func (h *MetricRegistryHandler) RegisterRoutes(r chi.Router) {
	r.Route("/metrics-registry", func(r chi.Router) {
		// Metric discovery & info
		r.Get("/", h.ListMetricRegistry)
		r.Get("/{metricID}", h.GetMetricRegistry)
		r.Get("/{metricID}/history", h.GetExecutionHistory)

		// Orchestration & lanes
		r.Post("/refresh-atomic", h.RefreshAtomicMetrics)
		r.Post("/{metricID}/compute-pop", h.ComputeMonthlyPoP)
		r.Post("/{metricID}/compute-comparisons", h.ComputeComparisonPeriods)
		r.Post("/{metricID}/detect-anomalies", h.DetectAnomalies)

		// Governance
		r.Post("/{metricID}/promote-golden", h.PromoteToGoldenPath)
		r.Get("/golden-path/readiness", h.GetGoldenPathReadiness)
	})
}

// ListMetricRegistry lists all metrics in the registry
func (h *MetricRegistryHandler) ListMetricRegistry(w http.ResponseWriter, r *http.Request) {
	domain := r.URL.Query().Get("domain")
	goldenOnly := r.URL.Query().Get("golden_only") == "true"

	var domainPtr *string
	if domain != "" {
		domainPtr = &domain
	}

	metrics, err := h.service.ListMetricRegistry(r.Context(), domainPtr, goldenOnly)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count":   len(metrics),
		"metrics": metrics,
	})
}

// GetMetricRegistry retrieves a specific metric
func (h *MetricRegistryHandler) GetMetricRegistry(w http.ResponseWriter, r *http.Request) {
	metricIDStr := chi.URLParam(r, "metricID")
	metricID, err := uuid.Parse(metricIDStr)
	if err != nil {
		http.Error(w, "invalid metric ID", http.StatusBadRequest)
		return
	}

	metric, err := h.service.GetMetricRegistry(r.Context(), metricID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metric)
}

// GetExecutionHistory retrieves execution history for a metric
func (h *MetricRegistryHandler) GetExecutionHistory(w http.ResponseWriter, r *http.Request) {
	metricIDStr := chi.URLParam(r, "metricID")
	metricID, err := uuid.Parse(metricIDStr)
	if err != nil {
		http.Error(w, "invalid metric ID", http.StatusBadRequest)
		return
	}

	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 500 {
			limit = l
		}
	}

	logs, err := h.service.GetExecutionHistory(r.Context(), metricID, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count": len(logs),
		"logs":  logs,
	})
}

// RefreshAtomicMetricsRequest is the request for atomic refresh
type RefreshAtomicMetricsRequest struct {
	MetricID *uuid.UUID `json:"metric_id"`
}

// RefreshAtomicMetrics triggers the real-time atomic refresh lane
func (h *MetricRegistryHandler) RefreshAtomicMetrics(w http.ResponseWriter, r *http.Request) {
	var req RefreshAtomicMetricsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	logs, err := h.service.RefreshAtomicMetrics(r.Context(), req.MetricID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":       "queued",
		"execution_id": logs[0].ExecutionID,
		"logs":         logs,
	})
}

// ComputeMonthlyPoPRequest is the request for PoP computation
type ComputeMonthlyPoPRequest struct {
	MetricID    *uuid.UUID `json:"metric_id"`
	PeriodStart *time.Time `json:"period_start"`
	PeriodEnd   *time.Time `json:"period_end"`
}

// ComputeMonthlyPoP triggers the batch PoP computation
func (h *MetricRegistryHandler) ComputeMonthlyPoP(w http.ResponseWriter, r *http.Request) {
	var req ComputeMonthlyPoPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	log, err := h.service.ComputeMonthlyPoP(r.Context(), req.MetricID, req.PeriodStart, req.PeriodEnd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(log)
}

// ComputeComparisonPeriodsRequest is the request for comparison periods
type ComputeComparisonPeriodsRequest struct {
	MetricID *uuid.UUID `json:"metric_id"`
}

// ComputeComparisonPeriods triggers comparison period computation
func (h *MetricRegistryHandler) ComputeComparisonPeriods(w http.ResponseWriter, r *http.Request) {
	var req ComputeComparisonPeriodsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	log, err := h.service.ComputeComparisonPeriods(r.Context(), req.MetricID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(log)
}

// DetectAnomaliesRequest is the request for anomaly detection
type DetectAnomaliesRequest struct {
	MetricID        *uuid.UUID `json:"metric_id"`
	ZScoreThreshold float64    `json:"zscore_threshold,omitempty"`
	WindowDays      int        `json:"window_days,omitempty"`
	MinDataPoints   int        `json:"min_data_points,omitempty"`
}

// DetectAnomalies triggers z-score anomaly detection
func (h *MetricRegistryHandler) DetectAnomalies(w http.ResponseWriter, r *http.Request) {
	var req DetectAnomaliesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Set defaults
	if req.ZScoreThreshold == 0 {
		req.ZScoreThreshold = 2.5
	}
	if req.WindowDays == 0 {
		req.WindowDays = 90
	}
	if req.MinDataPoints == 0 {
		req.MinDataPoints = 7
	}

	anomalies, err := h.service.DetectZScoreAnomalies(r.Context(), req.MetricID, req.ZScoreThreshold, req.WindowDays, req.MinDataPoints)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "queued",
		"count":     len(anomalies),
		"anomalies": anomalies,
	})
}

// PromoteToGoldenPath promotes a metric to golden path status
func (h *MetricRegistryHandler) PromoteToGoldenPath(w http.ResponseWriter, r *http.Request) {
	metricIDStr := chi.URLParam(r, "metricID")
	metricID, err := uuid.Parse(metricIDStr)
	if err != nil {
		http.Error(w, "invalid metric ID", http.StatusBadRequest)
		return
	}

	if err := h.service.PromoteToGoldenPath(r.Context(), metricID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "promoted_to_golden_path",
		"metric_id": metricID,
	})
}

// GetGoldenPathReadiness retrieves golden path metrics readiness status
func (h *MetricRegistryHandler) GetGoldenPathReadiness(w http.ResponseWriter, r *http.Request) {
	readiness, err := h.service.GetGoldenPathReadiness(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count":     len(readiness),
		"readiness": readiness,
	})
}
