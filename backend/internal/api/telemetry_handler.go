package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/rules"
)

type RuleTelemetryHandler struct {
	engine   *rules.RuleEngine
	profiler *rules.LatencyProfiler
}

func NewRuleTelemetryHandler(e *rules.RuleEngine, p *rules.LatencyProfiler) *RuleTelemetryHandler {
	return &RuleTelemetryHandler{engine: e, profiler: p}
}

func (h *RuleTelemetryHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/compliance", func(r chi.Router) {
		r.Get("/telemetry", h.HandleGetRuleTelemetry)
	})
}

type TelemetryResponse struct {
	Metrics   rules.EngineMetricsSnapshot `json:"metrics"`
	Profiler  rules.LatencyReport       `json:"profiler"`
	TenantCount int                      `json:"tenant_count"`
	CacheSize   int                      `json:"cache_size"`
}

func (h *RuleTelemetryHandler) HandleGetRuleTelemetry(w http.ResponseWriter, r *http.Request) {
	snapshot := h.engine.MetricsSnapshot()
	dist := h.profiler.GetDistribution()
	tenantCount := h.engine.TenantCount()
	cacheSize := h.engine.CurrentCacheSize()

	response := TelemetryResponse{
		Metrics:     snapshot,
		Profiler:    dist,
		TenantCount: tenantCount,
		CacheSize:   cacheSize,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
