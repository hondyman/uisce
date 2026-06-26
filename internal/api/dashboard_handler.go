package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/sirupsen/logrus"
)

// DashboardHandler serves real-time monitoring data
type DashboardHandler struct {
	logger       *logrus.Entry
	promRegistry prometheus.Gatherer
}

// NewDashboardHandler creates a new dashboard handler
func NewDashboardHandler(logger *logrus.Entry, registry prometheus.Gatherer) *DashboardHandler {
	return &DashboardHandler{
		logger:       logger.WithField("handler", "dashboard"),
		promRegistry: registry,
	}
}

// DashboardData aggregated data for frontend
type DashboardData struct {
	Timestamp           time.Time          `json:"timestamp"`
	CDCLag              map[string]int64   `json:"cdc_lag"`
	WorkflowCount       int                `json:"workflow_count"`
	RecentReschedules   []RescheduleEvent  `json:"recent_reschedules"`
	CacheHitRate        float64            `json:"cache_hit_rate"`
	AvailabilityLatency map[string]float64 `json:"availability_latency"`
}

type RescheduleEvent struct {
	JobID     string    `json:"job_id"`
	TenantID  string    `json:"tenant_id"`
	OldTime   time.Time `json:"old_time"`
	NewTime   time.Time `json:"new_time"`
	Reason    string    `json:"reason"`
	Timestamp time.Time `json:"timestamp"`
}

// GetDataHandler returns aggregated dashboard data
// @Summary Get dashboard metrics
// @Tags dashboard
// @Produce json
// @Param tenant_id query string false "Filter by tenant"
// @Success 200 {object} DashboardData
// @Router /api/v1/dashboard/data [get]
func (h *DashboardHandler) GetDataHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	data := DashboardData{
		Timestamp:           time.Now().UTC(),
		CDCLag:              make(map[string]int64),
		RecentReschedules:   []RescheduleEvent{},
		AvailabilityLatency: make(map[string]float64),
	}

	// Collect Prometheus metrics
	metricFamilies, err := h.promRegistry.Gather()
	if err != nil {
		h.logger.WithError(err).Warn("Failed to gather metrics")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Extract relevant metrics
	hitCounter := 0.0
	missCounter := 0.0
	for _, mf := range metricFamilies {
		switch mf.GetName() {
		case "calendar_cdc_lag_seconds":
			for _, m := range mf.GetMetric() {
				labels := labelToMap(m.GetLabel())
				if tenantID == "" || labels["tenant_id"] == tenantID {
					topic := labels["topic"]
					if g := m.GetGauge(); g != nil {
						data.CDCLag[topic] = int64(g.GetValue())
					}
				}
			}
		case "calendar_cache_hits_total":
			for _, m := range mf.GetMetric() {
				if c := m.GetCounter(); c != nil {
					hitCounter += c.GetValue()
				}
			}
		case "calendar_cache_misses_total":
			for _, m := range mf.GetMetric() {
				if c := m.GetCounter(); c != nil {
					missCounter += c.GetValue()
				}
			}
		case "availability_check_duration_seconds":
			for _, m := range mf.GetMetric() {
				labels := labelToMap(m.GetLabel())
				if tenantID == "" || labels["tenant_id"] == tenantID {
					profile := labels["profile_name"]
					if h := m.GetHistogram(); h != nil && h.GetSampleCount() > 0 {
						avg := h.GetSampleSum() / float64(h.GetSampleCount())
						data.AvailabilityLatency[profile] = avg * 1000 // to ms
					}
				}
			}
		}
	}

	// Calculate cache hit rate
	total := hitCounter + missCounter
	if total > 0 {
		data.CacheHitRate = hitCounter / total
	}

	// In production, would query Temporal for active workflows and recent reschedules
	data.WorkflowCount = 0 // placeholder

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// SSEHandler streams real-time metrics via Server-Sent Events
// @Summary Stream real-time dashboard metrics
// @Tags dashboard
// @Produce text/event-stream
// @Param tenant_id query string false "Filter by tenant"
// @Router /api/v1/dashboard/stream [get]
func (h *DashboardHandler) SSEHandler(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	tenantID := r.URL.Query().Get("tenant_id")
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Send initial data
	sendSSEEvent(w, flusher, "data", h.getDashboardData(tenantID))

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			sendSSEEvent(w, flusher, "data", h.getDashboardData(tenantID))
		}
	}
}

// getDashboardData is a helper for SSE
func (h *DashboardHandler) getDashboardData(tenantID string) DashboardData {
	// Simplified; in production would use GetDataHandler logic
	return DashboardData{
		Timestamp: time.Now().UTC(),
	}
}

func sendSSEEvent(w http.ResponseWriter, flusher http.Flusher, event string, data interface{}) {
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, jsonData)
	flusher.Flush()
}

// Helper: convert label pairs to map
func labelToMap(labels []*dto.LabelPair) map[string]string {
	m := make(map[string]string)
	for _, l := range labels {
		m[l.GetName()] = l.GetValue()
	}
	return m
}
