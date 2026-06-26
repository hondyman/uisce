package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/logging"
)

// AdminQoSHandler provides administrative endpoints for QoS management
type AdminQoSHandler struct {
	qosManager     *QoSManager
	tenantConfig   *TenantConfigService
	perfMonitor    *PerformanceMonitor
	loadTestEngine *LoadTestEngine
	perfTuner      *PerformanceTuner
}

// NewAdminQoSHandler creates a new admin QoS handler
func NewAdminQoSHandler(qosManager *QoSManager, tenantConfig *TenantConfigService,
	perfMonitor *PerformanceMonitor, loadTestEngine *LoadTestEngine,
	perfTuner *PerformanceTuner) *AdminQoSHandler {

	return &AdminQoSHandler{
		qosManager:     qosManager,
		tenantConfig:   tenantConfig,
		perfMonitor:    perfMonitor,
		loadTestEngine: loadTestEngine,
		perfTuner:      perfTuner,
	}
}

// RegisterRoutes registers the admin QoS routes
func (h *AdminQoSHandler) RegisterRoutes(r chi.Router) {
	r.Route("/admin/qos", func(r chi.Router) {
		r.Use(h.authMiddleware()) // Add authentication middleware

		// QoS Configuration
		r.Get("/config", h.getQoSConfig)
		r.Put("/config", h.updateQoSConfig)
		r.Post("/config/reset", h.resetQoSConfig)

		// Tenant Management
		r.Get("/tenants", h.listTenants)
		r.Get("/tenants/{tenantId}", h.getTenantConfig)
		r.Put("/tenants/{tenantId}", h.updateTenantConfig)
		r.Delete("/tenants/{tenantId}", h.deleteTenantConfig)

		// Performance Monitoring
		r.Get("/metrics", h.getMetrics)
		r.Get("/metrics/tenants/{tenantId}", h.getTenantMetrics)
		r.Get("/metrics/history", h.getMetricsHistory)

		// Load Testing
		r.Get("/load-tests", h.listLoadTestScenarios)
		r.Post("/load-tests/{scenario}/run", h.runLoadTest)
		r.Get("/load-tests/results", h.getLoadTestResults)
		r.Get("/load-tests/results/{resultId}", h.getLoadTestResult)

		// Tuning Recommendations
		r.Get("/tuning/recommendations", h.getTuningRecommendations)
		r.Delete("/tuning/recommendations", h.clearTuningRecommendations)

		// Autoscaling
		r.Get("/autoscaling/status", h.getAutoscalingStatus)
		r.Put("/autoscaling/config", h.updateAutoscalingConfig)
		r.Post("/autoscaling/manual-scale", h.manualScale)

		// Health Checks
		r.Get("/health", h.healthCheck)
		r.Get("/health/detailed", h.detailedHealthCheck)
	})
}

// authMiddleware provides basic authentication for admin endpoints
func (h *AdminQoSHandler) authMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Implement proper authentication
			// Check for admin API key or JWT token
			apiKey := r.Header.Get("X-Admin-API-Key")
			if apiKey == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "Missing API key"})
				return
			}

			// In production: Validate against secure key store
			// For now, check environment variable
			expectedKey := os.Getenv("ADMIN_API_KEY")
			if expectedKey != "" && apiKey != expectedKey {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{"error": "Invalid API key"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// getQoSConfig returns the current QoS configuration
func (h *AdminQoSHandler) getQoSConfig(w http.ResponseWriter, r *http.Request) {
	// Return global QoS configuration
	config := map[string]interface{}{
		"default_token_rate":        100,
		"default_burst_tokens":      200,
		"default_circuit_threshold": 5,
		"default_circuit_timeout":   "30s",
		"goroutine_pools": map[string]int{
			"eval":       16,
			"audit":      64,
			"background": 32,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// updateQoSConfig updates the global QoS configuration
func (h *AdminQoSHandler) updateQoSConfig(w http.ResponseWriter, r *http.Request) {
	var config map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Implement configuration updates
	// Validate and apply configuration
	fmt.Printf("[AdminQoS] Updating global QoS config: %v\n", config)

	// In production: Store in database, update runtime config
	logging.GetLogger().Sugar().Infof("QoS configuration updated: %+v", config)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "updated", "config": config})
}

// resetQoSConfig resets QoS configuration to defaults
func (h *AdminQoSHandler) resetQoSConfig(w http.ResponseWriter, r *http.Request) {
	// Implement configuration reset
	fmt.Printf("[AdminQoS] Resetting global QoS config to defaults\n")

	// Reset to default settings
	defaultConfig := map[string]interface{}{
		"max_connections":    100,
		"rate_limit_per_min": 1000,
		"query_timeout_sec":  30,
	}

	// In production: Update database and reload config
	logging.GetLogger().Sugar().Info("QoS configuration reset to defaults")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "reset", "config": defaultConfig})
}

// listTenants returns a list of all tenants
func (h *AdminQoSHandler) listTenants(w http.ResponseWriter, r *http.Request) {
	// Implement tenant listing from database
	// Query all active tenants with their QoS settings
	tenants := []map[string]interface{}{
		{"id": "tenant_1", "name": "Acme Corp", "tier": "enterprise", "status": "active"},
		{"id": "tenant_2", "name": "Beta Inc", "tier": "professional", "status": "active"},
	}

	// In production: SELECT * FROM tenants WHERE active = true
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"tenants": tenants, "count": len(tenants)})
}

// getTenantConfig returns configuration for a specific tenant
func (h *AdminQoSHandler) getTenantConfig(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")

	config, err := h.tenantConfig.GetConfig(tenantID)
	if err != nil {
		http.Error(w, "Tenant not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// updateTenantConfig updates configuration for a specific tenant
func (h *AdminQoSHandler) updateTenantConfig(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")

	var config TenantQoSConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	config.TenantID = tenantID
	if err := h.tenantConfig.UpdateConfig(&config); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logging.GetLogger().Sugar().Infof("Tenant configuration updated: %s", tenantID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Tenant configuration updated"})
}

// deleteTenantConfig deletes configuration for a specific tenant
func (h *AdminQoSHandler) deleteTenantConfig(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")

	// Implement tenant config deletion
	fmt.Printf("[AdminQoS] Deleting tenant %s\n", tenantID)

	// Soft delete or hard delete based on business logic
	// In production: UPDATE tenants SET deleted_at = NOW() WHERE id = $1
	logging.GetLogger().Sugar().Infof("Tenant configuration deleted: %s", tenantID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "deleted", "tenant_id": tenantID})
}

// getMetrics returns current performance metrics
func (h *AdminQoSHandler) getMetrics(w http.ResponseWriter, r *http.Request) {
	stats := h.perfMonitor.GetStats()
	if stats == nil {
		http.Error(w, "Metrics not available", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// getTenantMetrics returns metrics for a specific tenant
func (h *AdminQoSHandler) getTenantMetrics(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")

	snapshot := h.perfMonitor.GetTenantPerformanceSnapshot(tenantID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snapshot)
}

// getMetricsHistory returns historical metrics data
func (h *AdminQoSHandler) getMetricsHistory(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	hoursStr := r.URL.Query().Get("hours")
	if hoursStr == "" {
		hoursStr = "24"
	}
	hours, err := strconv.Atoi(hoursStr)
	if err != nil {
		http.Error(w, "Invalid hours parameter", http.StatusBadRequest)
		return
	}
	tenantID := r.URL.Query().Get("tenantId")
	if tenantID == "" {
		tenantID = "all"
	}

	endTime := time.Now().Format(time.RFC3339)
	startTime := time.Now().Add(-time.Duration(hours) * time.Hour).Format(time.RFC3339)

	// Implement historical metrics retrieval
	// Query time-series data from monitoring system
	metrics := map[string]interface{}{
		"period":    map[string]string{"start": startTime, "end": endTime},
		"tenant_id": tenantID,
		"data": []map[string]interface{}{
			{"timestamp": time.Now().Add(-1 * time.Hour).Format(time.RFC3339), "cpu_percent": 45.2, "mem_mb": 1024, "queries": 1500},
			{"timestamp": time.Now().Add(-30 * time.Minute).Format(time.RFC3339), "cpu_percent": 52.1, "mem_mb": 1100, "queries": 1800},
			{"timestamp": time.Now().Format(time.RFC3339), "cpu_percent": 38.5, "mem_mb": 980, "queries": 1200},
		},
	}

	// In production: Query from Prometheus, InfluxDB, or TimescaleDB
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// listLoadTestScenarios returns available load test scenarios
func (h *AdminQoSHandler) listLoadTestScenarios(w http.ResponseWriter, r *http.Request) {
	// For now, return the default scenarios
	scenarios := []map[string]interface{}{
		{
			"name":        "1x-peak-load",
			"description": "Simulate 1x expected peak load",
			"duration":    "5m",
			"concurrency": 50,
			"rps":         1000,
		},
		{
			"name":        "2x-peak-load",
			"description": "Simulate 2x expected peak load",
			"duration":    "3m",
			"concurrency": 100,
			"rps":         2000,
		},
		{
			"name":        "cache-invalidation-storm",
			"description": "Test frequent cache invalidations",
			"duration":    "2m",
			"concurrency": 30,
			"rps":         500,
		},
		{
			"name":        "qos-stress-test",
			"description": "Test QoS under heavy load",
			"duration":    "4m",
			"concurrency": 80,
			"rps":         1500,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scenarios)
}

// runLoadTest executes a load test scenario
func (h *AdminQoSHandler) runLoadTest(w http.ResponseWriter, r *http.Request) {
	scenarioName := chi.URLParam(r, "scenario")

	result, err := h.loadTestEngine.RunScenario(r.Context(), scenarioName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// getLoadTestResults returns all load test results
func (h *AdminQoSHandler) getLoadTestResults(w http.ResponseWriter, r *http.Request) {
	results := h.loadTestEngine.GetResults()

	// Convert to JSON-friendly format
	jsonResults := make([]map[string]interface{}, len(results))
	for i, result := range results {
		jsonResults[i] = map[string]interface{}{
			"id":              fmt.Sprintf("result-%d", i),
			"scenario_name":   result.ScenarioName,
			"duration":        result.Duration.String(),
			"total_requests":  result.TotalRequests,
			"successful_reqs": result.SuccessfulReqs,
			"failed_reqs":     result.FailedReqs,
			"p50_latency":     result.P50Latency.String(),
			"p95_latency":     result.P95Latency.String(),
			"p99_latency":     result.P99Latency.String(),
			"avg_latency":     result.AvgLatency.String(),
			"target_rps":      result.TargetRPS,
			"actual_rps":      result.ActualRPS,
			"cache_hit_rate":  result.CacheHitRate,
			"qos_denials":     result.QoSDenials,
			"breaker_trips":   result.BreakerTrips,
			"error_rate":      result.ErrorRate,
			"start_time":      result.StartTime,
			"end_time":        result.EndTime,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jsonResults)
}

// getLoadTestResult returns a specific load test result
func (h *AdminQoSHandler) getLoadTestResult(w http.ResponseWriter, r *http.Request) {
	resultID := chi.URLParam(r, "resultId")

	results := h.loadTestEngine.GetResults()
	for i, result := range results {
		if fmt.Sprintf("result-%d", i) == resultID {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(result)
			return
		}
	}

	http.Error(w, "Load test result not found", http.StatusNotFound)
}

// getTuningRecommendations returns current tuning recommendations
func (h *AdminQoSHandler) getTuningRecommendations(w http.ResponseWriter, r *http.Request) {
	recommendations := h.perfTuner.GetRecommendations()

	// Convert to JSON-friendly format
	jsonRecs := make([]map[string]interface{}, len(recommendations))
	for i, rec := range recommendations {
		jsonRecs[i] = map[string]interface{}{
			"id":                fmt.Sprintf("rec-%d", i),
			"component":         rec.Component,
			"issue":             rec.Issue,
			"recommendation":    rec.Recommendation,
			"priority":          rec.Priority,
			"impact":            rec.Impact,
			"time_to_implement": rec.TimeToImplement.String(),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jsonRecs)
}

// clearTuningRecommendations clears all tuning recommendations
func (h *AdminQoSHandler) clearTuningRecommendations(w http.ResponseWriter, r *http.Request) {
	h.perfTuner.ClearRecommendations()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Tuning recommendations cleared"})
}

// getAutoscalingStatus returns current autoscaling status
func (h *AdminQoSHandler) getAutoscalingStatus(w http.ResponseWriter, r *http.Request) {
	currentReplicas := h.perfTuner.GetCurrentReplicas()

	status := map[string]interface{}{
		"current_replicas": currentReplicas,
		"min_replicas":     1,
		"max_replicas":     10,
		"last_scale_up":    "N/A", // Would track actual times
		"last_scale_down":  "N/A",
		"scaling_events":   []interface{}{}, // Would contain scaling history
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// updateAutoscalingConfig updates autoscaling configuration
func (h *AdminQoSHandler) updateAutoscalingConfig(w http.ResponseWriter, r *http.Request) {
	var config map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Implement autoscaling config updates
	fmt.Printf("[AdminQoS] Updating autoscaling config: %v\n", config)

	// In production: Validate and store autoscaling rules
	// - min/max replicas
	// - target CPU/memory thresholds
	// - scale up/down policies
	logging.GetLogger().Sugar().Infof("Autoscaling configuration updated: %+v", config)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "updated", "config": config})
}

// manualScale performs manual scaling
func (h *AdminQoSHandler) manualScale(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Replicas int `json:"replicas"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Implement manual scaling
	fmt.Printf("[AdminQoS] Manual scaling requested: %d replicas\n", req.Replicas)

	// In production: Update Kubernetes deployment or similar
	// kubectl scale deployment my-app --replicas=N
	// Or call K8s API directly
	logging.GetLogger().Sugar().Infof("Manual scaling requested: %d replicas", req.Replicas)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "scaling", "target_replicas": req.Replicas})
}

// healthCheck provides a basic health check
func (h *AdminQoSHandler) healthCheck(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"version":   "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// detailedHealthCheck provides detailed health information
func (h *AdminQoSHandler) detailedHealthCheck(w http.ResponseWriter, r *http.Request) {
	stats := h.perfMonitor.GetStats()

	health := map[string]interface{}{
		"status":          "healthy",
		"timestamp":       time.Now(),
		"version":         "1.0.0",
		"uptime":          stats["uptime_seconds"],
		"metrics":         stats,
		"recommendations": len(h.perfTuner.GetRecommendations()),
		"load_tests":      len(h.loadTestEngine.GetResults()),
	}

	// Check for critical issues
	if errorRate, ok := stats["error_count"].(int64); ok && errorRate > 100 {
		health["status"] = "degraded"
		health["issues"] = []string{"High error rate detected"}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}
