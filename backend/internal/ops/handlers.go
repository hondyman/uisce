package ops

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler defines HTTP handlers for the ops API
type Handler struct {
	store             Store
	timeline          *TimelineService
	evaluator         *AlertEvaluator
	healthCalc        *HealthCalculator
	heatmapBuilder    *HeatmapBuilder
	fingerprinter     *ErrorFingerprinter
	actionExecutor    *ActionExecutor
	rcaEngine         *CorrelationEngine
	patternMatcher    *PatternMatcher
	rateLimiter       *RateLimiter
	paramValidator    *ParameterValidator
	responseSanitizer *ResponseSanitizer
	auditLogger       *AuditLogger
	regionRegistry    *RegionRegistry // Phase 3.1: Region routing and discovery
}

// NewHandler creates a new ops handler
func NewHandler(store Store) *Handler {
	timeline := NewTimelineService(store)
	return &Handler{
		store:             store,
		timeline:          timeline,
		evaluator:         NewAlertEvaluatorWithTimeline(store, timeline),
		healthCalc:        NewHealthCalculatorWithTimeline(store, timeline),
		heatmapBuilder:    NewHeatmapBuilderWithTimeline(store, timeline),
		fingerprinter:     NewErrorFingerprinterWithTimeline(store, timeline),
		actionExecutor:    NewActionExecutor(store, timeline),
		rcaEngine:         NewCorrelationEngine(store),
		patternMatcher:    NewPatternMatcher(store),
		rateLimiter:       NewRateLimiter(10), // 10 actions per minute
		paramValidator:    NewParameterValidator(),
		responseSanitizer: NewResponseSanitizer(),
		auditLogger:       NewAuditLogger(store),
		regionRegistry:    NewRegionRegistry(store), // Phase 3.1: Region routing
	}
}

// RegisterRoutes registers all ops routes
func (h *Handler) RegisterRoutes(router *chi.Mux) {
	router.Route("/admin/alerts", func(r chi.Router) {
		r.Get("/", h.listAlerts)
		r.Post("/", h.createAlert)
		r.Get("/{alertID}", h.getAlert)
		r.Put("/{alertID}", h.updateAlert)
		r.Delete("/{alertID}", h.deleteAlert)
		r.Get("/{alertID}/events", h.getAlertEvents)
		r.Post("/evaluate", h.evaluateAlerts)
	})

	router.Route("/admin/tenants/{tenantID}/health", func(r chi.Router) {
		r.Get("/", h.getTenantHealth)
	})

	router.Get("/admin/endpoints/health", h.getEndpointHealthList)
	router.Get("/admin/endpoints/{endpoint}/health", h.getEndpointHealth)

	router.Get("/admin/latency/heatmap", h.getLatencyHeatmap)
	router.Get("/admin/latency/heatmap/regions", h.getRegionHeatmap)
	router.Get("/admin/latency/heatmap/tenants", h.getTenantHeatmap)
	router.Get("/admin/latency/heatmap/endpoints", h.getEndpointHeatmap)

	router.Get("/admin/errors/fingerprints", h.listFingerprints)
	router.Get("/admin/errors/fingerprints/{fingerprintID}", h.getFingerprintHistory)

	// Timeline and Incident Management
	router.Route("/admin/ops", func(r chi.Router) {
		r.Get("/timeline", h.GetTimeline)
		r.Get("/incidents/{incidentID}", h.GetIncident)
		r.Post("/incidents/{incidentID}/close", h.CloseIncident)
		r.Post("/incidents/{incidentID}/execute-action", h.ExecuteAction)
		r.Get("/incidents/{incidentID}/rca", h.ComputeRCA)
		r.Get("/incidents/{incidentID}/similar", h.GetSimilarIncidents)
		r.Get("/incidents/{incidentID}/pattern", h.GetIncidentPattern)
		r.Get("/incidents/{incidentID}/audit", h.GetIncidentAuditLog)
		r.Get("/audit", h.ListAuditLogs)
		r.Get("/audit/{auditID}", h.GetAuditLog)

		// Region Management (Phase 3.1)
		r.Get("/regions", h.ListRegions)
		r.Get("/regions/{regionCode}", h.GetRegion)
		r.Route("/tenants/{tenantID}/regions", func(tr chi.Router) {
			tr.Get("/", h.GetTenantRegions)
			tr.Post("/", h.ConfigureTenantRegion)
			tr.Get("/{region}", h.GetTenantRegionRouting)
		})

		// Phase 3.13: Advanced Chain Management API
		r.Route("/chains", func(ch chi.Router) {
			// Chain State Management
			ch.Post("/{chainID}/state", h.InitializeChainState)
			ch.Get("/{chainID}/state", h.GetChainState)
			ch.Get("/states", h.ListChainStates)

			// Conflict Management
			ch.Get("/{chainID}/conflicts", h.GetConflict)
			ch.Put("/conflicts/{conflictID}", h.ResolveConflict)

			// Metrics & SLA
			ch.Get("/{chainID}/metrics", h.GetChainMetrics)
			ch.Get("/", h.ListChainsBySLACompliance)

			// Phase 3.14: Chain Search, Filtering & Health
			ch.Post("/filter", h.FilterChains)
			ch.Get("/search", h.SearchChains)
			ch.Get("/{chainID}/health", h.GetChainHealth)
		})

		// Priority Queue Management
		r.Route("/chain-queues", func(q chi.Router) {
			q.Post("/", h.CreateChainExecutionQueue)
			q.Get("/", h.ListPendingChainQueues)
			q.Get("/{executionID}", h.GetChainExecutionQueue)
			q.Put("/{executionID}", h.UpdateChainExecutionQueue)
		})

		// Phase 3.14: Analytics
		r.Route("/analytics", func(a chi.Router) {
			a.Get("/sla-trends", h.ListSLAComplianceTrends)
			a.Get("/conflict-trends", h.GetConflictResolutionTrend)
		})

		// Phase 3.14: Batch Operations
		r.Route("/batch", func(b chi.Router) {
			b.Route("/conflicts", func(bc chi.Router) {
				bc.Post("/resolve", h.BatchResolveConflicts)
				bc.Get("/{batchID}", h.GetBatchOperation)
			})
		})
	})
}

// ========== Alerts ==========

// listAlerts handles GET /admin/alerts
func (h *Handler) listAlerts(w http.ResponseWriter, r *http.Request) {
	var enabledFilter *bool
	if enabledStr := r.URL.Query().Get("enabled"); enabledStr != "" {
		enabled := enabledStr == "true"
		enabledFilter = &enabled
	}

	alerts, err := h.store.ListAlerts(r.Context(), enabledFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data": alerts,
	})
}

// createAlert handles POST /admin/alerts
func (h *Handler) createAlert(w http.ResponseWriter, r *http.Request) {
	var req Alert
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	alert, err := h.store.CreateAlert(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"data": alert,
	})
}

// getAlert handles GET /admin/alerts/{alertID}
func (h *Handler) getAlert(w http.ResponseWriter, r *http.Request) {
	alertID := chi.URLParam(r, "alertID")
	id, err := uuid.Parse(alertID)
	if err != nil {
		http.Error(w, "invalid alert id", http.StatusBadRequest)
		return
	}

	alert, err := h.store.GetAlert(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if alert == nil {
		http.Error(w, "alert not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data": alert,
	})
}

// updateAlert handles PUT /admin/alerts/{alertID}
func (h *Handler) updateAlert(w http.ResponseWriter, r *http.Request) {
	alertID := chi.URLParam(r, "alertID")
	id, err := uuid.Parse(alertID)
	if err != nil {
		http.Error(w, "invalid alert id", http.StatusBadRequest)
		return
	}

	var req Alert
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.store.UpdateAlert(r.Context(), id, req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	alert, _ := h.store.GetAlert(r.Context(), id)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data": alert,
	})
}

// deleteAlert handles DELETE /admin/alerts/{alertID}
func (h *Handler) deleteAlert(w http.ResponseWriter, r *http.Request) {
	alertID := chi.URLParam(r, "alertID")
	id, err := uuid.Parse(alertID)
	if err != nil {
		http.Error(w, "invalid alert id", http.StatusBadRequest)
		return
	}

	if err := h.store.DeleteAlert(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// getAlertEvents handles GET /admin/alerts/{alertID}/events
func (h *Handler) getAlertEvents(w http.ResponseWriter, r *http.Request) {
	alertID := chi.URLParam(r, "alertID")
	id, err := uuid.Parse(alertID)
	if err != nil {
		http.Error(w, "invalid alert id", http.StatusBadRequest)
		return
	}

	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	events, err := h.store.GetAlertEvents(r.Context(), id, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data": events,
	})
}

// evaluateAlerts handles POST /admin/alerts/evaluate
func (h *Handler) evaluateAlerts(w http.ResponseWriter, r *http.Request) {
	if err := h.evaluator.EvaluateAll(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "alerts evaluated",
	})
}

// ========== Tenant Health ==========

// getTenantHealth handles GET /admin/tenants/{tenantID}/health
func (h *Handler) getTenantHealth(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantID")
	id, err := uuid.Parse(tenantID)
	if err != nil {
		http.Error(w, "invalid tenant id", http.StatusBadRequest)
		return
	}

	window := 1 * time.Hour
	if windowStr := r.URL.Query().Get("window"); windowStr != "" {
		if w, err := time.ParseDuration(windowStr); err == nil {
			window = w
		}
	}

	health, err := h.healthCalc.ComputeTenantHealth(r.Context(), id, window)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data": health,
	})
}

// ========== Endpoint Health ==========

// getEndpointHealthList handles GET /admin/endpoints/health
func (h *Handler) getEndpointHealthList(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 500 {
			limit = l
		}
	}

	healths, err := h.store.GetEndpointHealths(r.Context(), limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data": healths,
	})
}

// getEndpointHealth handles GET /admin/endpoints/{endpoint}/health
func (h *Handler) getEndpointHealth(w http.ResponseWriter, r *http.Request) {
	endpoint := chi.URLParam(r, "endpoint")

	window := 30 * time.Minute
	if windowStr := r.URL.Query().Get("window"); windowStr != "" {
		if w, err := time.ParseDuration(windowStr); err == nil {
			window = w
		}
	}

	health, err := h.healthCalc.ComputeEndpointHealth(r.Context(), endpoint, window)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data": health,
	})
}

// ========== Latency Heatmaps ==========

// getLatencyHeatmap handles GET /admin/latency/heatmap
func (h *Handler) getLatencyHeatmap(w http.ResponseWriter, r *http.Request) {
	dimensionType := r.URL.Query().Get("group_by")
	if dimensionType == "" {
		dimensionType = "region"
	}

	heatmap, err := h.heatmapBuilder.BuildHeatmap(r.Context(), dimensionType, 5*time.Minute, 24*time.Hour, 20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data": heatmap,
	})
}

// getRegionHeatmap handles GET /admin/latency/heatmap/regions
func (h *Handler) getRegionHeatmap(w http.ResponseWriter, r *http.Request) {
	heatmap, err := h.heatmapBuilder.BuildRegionHeatmap(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data": heatmap,
	})
}

// getTenantHeatmap handles GET /admin/latency/heatmap/tenants
func (h *Handler) getTenantHeatmap(w http.ResponseWriter, r *http.Request) {
	heatmap, err := h.heatmapBuilder.BuildTenantHeatmap(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data": heatmap,
	})
}

// getEndpointHeatmap handles GET /admin/latency/heatmap/endpoints
func (h *Handler) getEndpointHeatmap(w http.ResponseWriter, r *http.Request) {
	heatmap, err := h.heatmapBuilder.BuildEndpointHeatmap(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data": heatmap,
	})
}

// ========== Error Fingerprints ==========

// listFingerprints handles GET /admin/errors/fingerprints
func (h *Handler) listFingerprints(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 500 {
			limit = l
		}
	}

	fingerprints, err := h.fingerprinter.ListFingerprints(r.Context(), limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data": fingerprints,
	})
}

// getFingerprintHistory handles GET /admin/errors/fingerprints/{fingerprintID}
func (h *Handler) getFingerprintHistory(w http.ResponseWriter, r *http.Request) {
	fingerprintID := chi.URLParam(r, "fingerprintID")
	id, err := uuid.Parse(fingerprintID)
	if err != nil {
		http.Error(w, "invalid fingerprint id", http.StatusBadRequest)
		return
	}

	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	events, err := h.fingerprinter.GetFingerprintHistory(r.Context(), id, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data": events,
	})
}

// ========== Audit Log (Phase 2.4c) ==========

// GetIncidentAuditLog handles GET /admin/ops/incidents/:incidentID/audit
func (h *Handler) GetIncidentAuditLog(w http.ResponseWriter, r *http.Request) {
	incidentID := chi.URLParam(r, "incidentID")
	if incidentID == "" {
		http.Error(w, "incident_id is required", http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(incidentID)
	if err != nil {
		http.Error(w, "invalid incident_id", http.StatusBadRequest)
		return
	}

	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	auditLogs, err := h.store.ListIncidentAuditLogs(r.Context(), id, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"incident_id": incidentID,
		"audit_logs":  auditLogs,
		"count":       len(auditLogs),
	})
}

// ListAuditLogs handles GET /admin/ops/audit with optional filters
func (h *Handler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	offset := 0
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Build filters
	filters := AuditLogFilters{}

	if userID := r.URL.Query().Get("user_id"); userID != "" {
		filters.UserID = &userID
	}

	if actionType := r.URL.Query().Get("action_type"); actionType != "" {
		filters.ActionType = &actionType
	}

	if status := r.URL.Query().Get("status"); status != "" {
		filters.Status = &status
	}

	if incidentIDStr := r.URL.Query().Get("incident_id"); incidentIDStr != "" {
		if id, err := uuid.Parse(incidentIDStr); err == nil {
			filters.IncidentID = &id
		}
	}

	if startTimeStr := r.URL.Query().Get("start_time"); startTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			filters.StartTime = &t
		}
	}

	if endTimeStr := r.URL.Query().Get("end_time"); endTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			filters.EndTime = &t
		}
	}

	auditLogs, err := h.store.ListAuditLogs(r.Context(), filters, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"audit_logs": auditLogs,
		"count":      len(auditLogs),
		"limit":      limit,
		"offset":     offset,
	})
}

// GetAuditLog handles GET /admin/ops/audit/:auditID
func (h *Handler) GetAuditLog(w http.ResponseWriter, r *http.Request) {
	auditID := chi.URLParam(r, "auditID")
	if auditID == "" {
		http.Error(w, "audit_id is required", http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(auditID)
	if err != nil {
		http.Error(w, "invalid audit_id", http.StatusBadRequest)
		return
	}

	auditLog, err := h.store.GetAuditLog(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if auditLog == nil {
		http.Error(w, "audit log not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, auditLog)
}

// ========== Helpers ==========

// respondJSON writes a JSON response
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
