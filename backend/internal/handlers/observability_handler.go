package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/observability"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// ObservabilityHandler handles observability API requests
type ObservabilityHandler struct {
	metricsCollector *observability.MetricsCollector
	sloService       *observability.SLOService
}

// NewObservabilityHandler creates a new observability handler
func NewObservabilityHandler(
	metricsCollector *observability.MetricsCollector,
	sloService *observability.SLOService,
) *ObservabilityHandler {
	return &ObservabilityHandler{
		metricsCollector: metricsCollector,
		sloService:       sloService,
	}
}

// RegisterRoutes registers observability routes
func (h *ObservabilityHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/observability", func(r chi.Router) {
		// Metrics endpoints
		r.Get("/metrics", h.QueryMetrics)
		r.Post("/metrics", h.RecordMetric)
		r.Get("/metrics/{name}/latest", h.GetLatestMetric)

		// SLO endpoints
		r.Get("/slos", h.ListSLOs)
		r.Post("/slos", h.CreateSLO)
		r.Get("/slos/{id}", h.GetSLO)
		r.Put("/slos/{id}", h.UpdateSLO)
		r.Delete("/slos/{id}", h.DeleteSLO)
		r.Get("/slos/{id}/status", h.GetSLOStatus)

		// Alert endpoints
		r.Get("/alerts/active", h.GetActiveAlerts)
		r.Post("/alerts/{id}/acknowledge", h.AcknowledgeAlert)
		r.Post("/alerts/{id}/resolve", h.ResolveAlert)

		// Alert rules
		r.Post("/slos/{sloId}/rules", h.CreateAlertRule)
		r.Get("/slos/{sloId}/rules", h.GetAlertRules)

		// Dashboard
		r.Get("/dashboard", h.GetDashboard)

		// Evaluation trigger (for testing/manual evaluation)
		r.Post("/evaluate", h.EvaluateAllSLOs)
	})
}

// QueryMetrics queries metrics based on filters
func (h *ObservabilityHandler) QueryMetrics(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "name parameter required", http.StatusBadRequest)
		return
	}

	// Parse time range
	startTime := time.Now().Add(-24 * time.Hour)
	endTime := time.Now()

	if s := r.URL.Query().Get("start"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			startTime = t
		}
	}
	if e := r.URL.Query().Get("end"); e != "" {
		if t, err := time.Parse(time.RFC3339, e); err == nil {
			endTime = t
		}
	}

	query := observability.MetricQuery{
		Name:      name,
		StartTime: startTime,
		EndTime:   endTime,
		Step:      r.URL.Query().Get("step"),
		Aggregate: r.URL.Query().Get("aggregate"),
	}

	ctx := r.Context()
	series, err := h.metricsCollector.Query(ctx, query, tenantID)
	if err != nil {
		http.Error(w, "Failed to query metrics: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(series)
}

// RecordMetric records a new metric data point
func (h *ObservabilityHandler) RecordMetric(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	var metric observability.Metric
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	metric.TenantID = tenantID
	if metric.Timestamp.IsZero() {
		metric.Timestamp = time.Now()
	}

	h.metricsCollector.Record(metric)

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
}

// GetLatestMetric returns the most recent value for a metric
func (h *ObservabilityHandler) GetLatestMetric(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	name := chi.URLParam(r, "name")

	ctx := r.Context()
	metric, err := h.metricsCollector.GetLatest(ctx, tenantID, name)
	if err != nil {
		http.Error(w, "Metric not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metric)
}

// ListSLOs lists all SLOs for the tenant
func (h *ObservabilityHandler) ListSLOs(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	ctx := r.Context()
	slos, err := h.sloService.ListSLOs(ctx, tenantID)
	if err != nil {
		http.Error(w, "Failed to list SLOs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(slos)
}

// CreateSLO creates a new SLO
func (h *ObservabilityHandler) CreateSLO(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	var slo observability.SLO
	if err := json.NewDecoder(r.Body).Decode(&slo); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	slo.TenantID = tenantID

	ctx := r.Context()
	if err := h.sloService.CreateSLO(ctx, &slo); err != nil {
		http.Error(w, "Failed to create SLO: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(slo)
}

// GetSLO retrieves a specific SLO
func (h *ObservabilityHandler) GetSLO(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid SLO ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	slo, err := h.sloService.GetSLO(ctx, id)
	if err != nil {
		http.Error(w, "SLO not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(slo)
}

// UpdateSLO updates an existing SLO
func (h *ObservabilityHandler) UpdateSLO(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid SLO ID", http.StatusBadRequest)
		return
	}

	var slo observability.SLO
	if err := json.NewDecoder(r.Body).Decode(&slo); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	slo.ID = id

	ctx := r.Context()
	if err := h.sloService.UpdateSLO(ctx, &slo); err != nil {
		http.Error(w, "Failed to update SLO: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(slo)
}

// DeleteSLO deletes an SLO
func (h *ObservabilityHandler) DeleteSLO(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid SLO ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	if err := h.sloService.DeleteSLO(ctx, id); err != nil {
		http.Error(w, "Failed to delete SLO: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetSLOStatus returns the current status of an SLO
func (h *ObservabilityHandler) GetSLOStatus(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid SLO ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	status, err := h.sloService.GetSLOStatus(ctx, id)
	if err != nil {
		http.Error(w, "Failed to get SLO status: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// GetActiveAlerts returns all currently firing alerts
func (h *ObservabilityHandler) GetActiveAlerts(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	ctx := r.Context()
	alerts, err := h.sloService.GetActiveAlerts(ctx, tenantID)
	if err != nil {
		http.Error(w, "Failed to get alerts: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alerts)
}

// AcknowledgeAlert acknowledges a firing alert
func (h *ObservabilityHandler) AcknowledgeAlert(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	_, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid alert ID", http.StatusBadRequest)
		return
	}

	// TODO: Implement acknowledge
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "acknowledged"})
}

// ResolveAlert resolves an alert
func (h *ObservabilityHandler) ResolveAlert(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid alert ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	if err := h.sloService.ResolveAlert(ctx, id); err != nil {
		http.Error(w, "Failed to resolve alert: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "resolved"})
}

// CreateAlertRule creates a new alert rule for an SLO
func (h *ObservabilityHandler) CreateAlertRule(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	sloIDStr := chi.URLParam(r, "sloId")
	sloID, err := uuid.Parse(sloIDStr)
	if err != nil {
		http.Error(w, "Invalid SLO ID", http.StatusBadRequest)
		return
	}

	var rule observability.AlertRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	rule.TenantID = tenantID
	rule.SLOID = sloID

	ctx := r.Context()
	if err := h.sloService.CreateAlertRule(ctx, &rule); err != nil {
		http.Error(w, "Failed to create alert rule: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(rule)
}

// GetAlertRules returns all alert rules for an SLO
func (h *ObservabilityHandler) GetAlertRules(w http.ResponseWriter, r *http.Request) {
	sloIDStr := chi.URLParam(r, "sloId")
	sloID, err := uuid.Parse(sloIDStr)
	if err != nil {
		http.Error(w, "Invalid SLO ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	rules, err := h.sloService.GetAlertRulesForSLO(ctx, sloID)
	if err != nil {
		http.Error(w, "Failed to get alert rules: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rules)
}

// GetDashboard returns aggregated dashboard data
func (h *ObservabilityHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	ctx := r.Context()

	// Get SLO statuses
	sloStatuses, _ := h.sloService.EvaluateAllSLOs(ctx, tenantID)

	// Get active alerts
	activeAlerts, _ := h.sloService.GetActiveAlerts(ctx, tenantID)

	// Get metrics summary
	hours := 24
	if h := r.URL.Query().Get("hours"); h != "" {
		if parsed, err := strconv.Atoi(h); err == nil {
			hours = parsed
		}
	}
	duration := time.Duration(hours) * time.Hour

	totalQueries, _ := h.metricsCollector.GetAggregated(ctx, tenantID, "query_count", "sum", duration)
	avgLatency, _ := h.metricsCollector.GetAggregated(ctx, tenantID, "query_latency_ms", "avg", duration)
	errorCount, _ := h.metricsCollector.GetAggregated(ctx, tenantID, "query_error_count", "sum", duration)
	cacheHits, _ := h.metricsCollector.GetAggregated(ctx, tenantID, "cache_hit_count", "sum", duration)
	cacheMisses, _ := h.metricsCollector.GetAggregated(ctx, tenantID, "cache_miss_count", "sum", duration)

	errorRate := 0.0
	if totalQueries > 0 {
		errorRate = errorCount / totalQueries
	}

	cacheHitRate := 0.0
	if cacheHits+cacheMisses > 0 {
		cacheHitRate = cacheHits / (cacheHits + cacheMisses)
	}

	dashboard := observability.DashboardData{
		SLOStatuses:  sloStatuses,
		ActiveAlerts: activeAlerts,
		MetricsSummary: observability.MetricsSummary{
			TotalQueries:    int64(totalQueries),
			AvgQueryLatency: avgLatency,
			ErrorRate:       errorRate,
			CacheHitRate:    cacheHitRate,
		},
		SystemHealth: observability.SystemHealth{
			Status:          "healthy",
			LastHealthCheck: time.Now(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dashboard)
}

// EvaluateAllSLOs triggers evaluation of all SLOs
func (h *ObservabilityHandler) EvaluateAllSLOs(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	ctx := r.Context()
	statuses, err := h.sloService.EvaluateAllSLOs(ctx, tenantID)
	if err != nil {
		http.Error(w, "Failed to evaluate SLOs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statuses)
}

// Helper function to get tenant ID from request
func getTenantID(r *http.Request) uuid.UUID {
	tenantIDStr := jwtmiddleware.GetClaimsFromContext(r).TenantID
	tenantID, _ := uuid.Parse(tenantIDStr)
	return tenantID
}
