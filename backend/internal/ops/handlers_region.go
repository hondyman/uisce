package ops

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ========== Region Management Handlers ==========

// ListRegions handles GET /admin/ops/regions
func (h *Handler) ListRegions(w http.ResponseWriter, r *http.Request) {
	activeOnly := r.URL.Query().Get("active") == "true"

	regions, err := h.store.ListRegionConfigs(r.Context(), activeOnly)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"regions": regions,
		"count":   len(regions),
	})
}

// GetRegion handles GET /admin/ops/regions/{regionCode}
func (h *Handler) GetRegion(w http.ResponseWriter, r *http.Request) {
	regionCode := chi.URLParam(r, "regionCode")
	if regionCode == "" {
		http.Error(w, "region_code is required", http.StatusBadRequest)
		return
	}

	if !IsValidRegion(regionCode) {
		http.Error(w, "invalid region format", http.StatusBadRequest)
		return
	}

	region, err := h.store.GetRegionConfig(r.Context(), regionCode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if region == nil {
		http.Error(w, "region not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, region)
}

// CreateRegionRoutingRequest is the request body for creating region routing
type CreateRegionRoutingRequest struct {
	Region            string `json:"region"`
	StarRocksCluster  string `json:"starrocks_cluster,omitempty"`
	RedpandaBroker    string `json:"redpanda_broker,omitempty"`
	TemporalNamespace string `json:"temporal_namespace,omitempty"`
	OpsWorkerPool     string `json:"ops_worker_pool,omitempty"`
}

// ConfigureTenantRegion handles POST /admin/ops/tenants/{tenantID}/regions
func (h *Handler) ConfigureTenantRegion(w http.ResponseWriter, r *http.Request) {
	tenantIDStr := chi.URLParam(r, "tenantID")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		http.Error(w, "invalid tenant_id", http.StatusBadRequest)
		return
	}

	var req CreateRegionRoutingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate region
	if !IsValidRegion(req.Region) {
		http.Error(w, "invalid region format", http.StatusBadRequest)
		return
	}

	// Verify region exists
	regionConfig, err := h.store.GetRegionConfig(r.Context(), req.Region)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if regionConfig == nil {
		http.Error(w, "region not found", http.StatusNotFound)
		return
	}

	// Create or update routing
	routing := &RegionRouting{
		ID:       uuid.New(),
		TenantID: tenantID,
		Region:   req.Region,
	}

	if req.StarRocksCluster != "" {
		routing.StarRocksCluster = &req.StarRocksCluster
	}
	if req.RedpandaBroker != "" {
		routing.RedpandaBroker = &req.RedpandaBroker
	}
	if req.TemporalNamespace != "" {
		routing.TemporalNamespace = &req.TemporalNamespace
	}
	if req.OpsWorkerPool != "" {
		routing.OpsWorkerPool = &req.OpsWorkerPool
	}

	if err := h.store.InsertRegionRouting(r.Context(), routing); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusCreated, routing)
}

// GetTenantRegions handles GET /admin/ops/tenants/{tenantID}/regions
func (h *Handler) GetTenantRegions(w http.ResponseWriter, r *http.Request) {
	tenantIDStr := chi.URLParam(r, "tenantID")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		http.Error(w, "invalid tenant_id", http.StatusBadRequest)
		return
	}

	routings, err := h.store.ListRegionRoutings(r.Context(), tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"tenant_id": tenantID.String(),
		"regions":   routings,
		"count":     len(routings),
	})
}

// GetTenantRegionRouting handles GET /admin/ops/tenants/{tenantID}/regions/{region}
func (h *Handler) GetTenantRegionRouting(w http.ResponseWriter, r *http.Request) {
	tenantIDStr := chi.URLParam(r, "tenantID")
	region := chi.URLParam(r, "region")

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		http.Error(w, "invalid tenant_id", http.StatusBadRequest)
		return
	}

	if !IsValidRegion(region) {
		http.Error(w, "invalid region format", http.StatusBadRequest)
		return
	}

	routing, err := h.store.GetRegionRouting(r.Context(), tenantID, region)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if routing == nil {
		http.Error(w, "region routing not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, routing)
}

// ========== Timeline with Region Support ==========

// GetTimelineFilters defines timeline filtering options
type GetTimelineFilters struct {
	Since  string
	Limit  int
	Region *string
	Tenant *string
}

// GetTimelineWithRegion handles GET /admin/ops/timeline?region=...&tenant=...
// Extends GetTimeline with region awareness
func (h *Handler) GetTimelineWithRegion(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	sinceStr := q.Get("since")
	limitStr := q.Get("limit")
	regionStr := q.Get("region")

	// Parse since duration (e.g., "1h", "24h", "7d")
	since := parseSinceDuration(sinceStr)

	limit := 200
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 && v <= 1000 {
			limit = v
		}
	}

	// Get events (filter by region client-side for now, will be DB-level in Phase 3.2)
	events, err := h.store.ListEvents(r.Context(), since, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Filter by region if provided
	if regionStr != "" {
		if !IsValidRegion(regionStr) {
			http.Error(w, "invalid region format", http.StatusBadRequest)
			return
		}

		events = filterEventsByRegion(events, regionStr)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(TimelineResponse{
		Events: events,
		Total:  len(events),
	})
}

// ====== Helper Functions ======

func parseSinceDuration(sinceStr string) time.Time {
	since := time.Now().Add(-1 * time.Hour)
	if sinceStr != "" {
		if d, err := time.ParseDuration(sinceStr); err == nil {
			since = time.Now().Add(-d)
		}
	}
	return since
}

func filterEventsByRegion(events []Event, region string) []Event {
	var filtered []Event
	for _, e := range events {
		if e.Region != nil && *e.Region == region {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// ========== Phase 3.9: Region-Aware Dashboard API Handlers ==========

// ListRegionsSummary handles GET /admin/ops/regions/summary
// Returns aggregated health, SLA, metrics, and incident counts for all regions
func (h *Handler) ListRegionsSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get metrics, health, SLA, and incident counts
	metrics, err := h.store.ListRegionalMetrics(ctx, 100)
	if err != nil {
		http.Error(w, "failed to list regional metrics", http.StatusInternalServerError)
		return
	}

	health, err := h.store.ListRegionalHealth(ctx, 100)
	if err != nil {
		http.Error(w, "failed to list regional health", http.StatusInternalServerError)
		return
	}

	latestSLA, err := h.store.ListLatestRegionalSLAStatuses(ctx)
	if err != nil {
		http.Error(w, "failed to list regional sla status", http.StatusInternalServerError)
		return
	}

	incidentCounts, err := h.store.ListRegionalIncidentCounts(ctx, time.Now().Add(-24*time.Hour), time.Now())
	if err != nil {
		http.Error(w, "failed to list regional incident counts", http.StatusInternalServerError)
		return
	}

	// Index by region for efficient lookup
	healthByRegion := make(map[string]*RegionalHealth)
	for i := range health {
		healthByRegion[health[i].Region] = &health[i]
	}

	slaByRegion := make(map[string]*RegionalSLAStatus)
	for i := range latestSLA {
		slaByRegion[latestSLA[i].Region] = &latestSLA[i]
	}

	incidentsByRegion := make(map[string]int)
	for _, c := range incidentCounts {
		incidentsByRegion[c.Region] = c.Count
	}

	// Build region summaries
	summaries := make([]RegionSummary, 0, len(metrics))
	for _, m := range metrics {
		h := healthByRegion[m.Region]
		s := slaByRegion[m.Region]
		count := incidentsByRegion[m.Region]

		healthScore := 100
		healthStatus := "healthy"
		if h != nil {
			healthScore = h.Score
			healthStatus = h.Status
		}

		slaCompliance := 100.0
		if s != nil {
			slaCompliance = s.CompliancePct
		}

		summaries = append(summaries, RegionSummary{
			Region:           m.Region,
			HealthScore:      healthScore,
			HealthStatus:     healthStatus,
			SLACompliance:    slaCompliance,
			ErrorRate:        m.ErrorRate,
			LatencyP95Ms:     float64(m.P95Latency),
			Availability:     m.Availability,
			IncidentCount24h: count,
			UpdatedAt:        m.UpdatedAt,
		})
	}

	respondJSON(w, http.StatusOK, summaries)
}

// GetRegionDetail handles GET /admin/ops/regions/{region}
// Returns comprehensive drill-down data for a specific region
func (h *Handler) GetRegionDetail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	region := chi.URLParam(r, "region")

	if region == "" {
		http.Error(w, "region is required", http.StatusBadRequest)
		return
	}

	if !IsValidRegion(region) {
		http.Error(w, "invalid region format", http.StatusBadRequest)
		return
	}

	// Fetch all regional data
	metrics, err := h.store.GetRegionalMetrics(ctx, region)
	if err != nil {
		http.Error(w, "failed to get regional metrics", http.StatusInternalServerError)
		return
	}

	health, err := h.store.GetRegionalHealth(ctx, region)
	if err != nil {
		http.Error(w, "failed to get regional health", http.StatusInternalServerError)
		return
	}

	sla, err := h.store.GetRegionalSLA(ctx, region)
	if err != nil {
		http.Error(w, "failed to get regional sla", http.StatusInternalServerError)
		return
	}

	slaHistory, err := h.store.ListRegionalSLAStatuses(ctx, region, 50)
	if err != nil {
		http.Error(w, "failed to list sla history", http.StatusInternalServerError)
		return
	}

	incidents, err := h.store.ListIncidentsByRegion(ctx, region, 50)
	if err != nil {
		http.Error(w, "failed to list incidents", http.StatusInternalServerError)
		return
	}

	events, err := h.store.ListOpsEventsByRegion(ctx, region, 100)
	if err != nil {
		http.Error(w, "failed to list ops events", http.StatusInternalServerError)
		return
	}

	actions, err := h.store.ListAuditLogsByRegion(ctx, region, 100)
	if err != nil {
		http.Error(w, "failed to list audit logs", http.StatusInternalServerError)
		return
	}

	detail := RegionDetail{
		Region:           region,
		Metrics:          metrics,
		Health:           health,
		SLA:              sla,
		SLAStatusHistory: slaHistory,
		RecentIncidents:  incidents,
		RecentOpsEvents:  events,
		RecentActions:    actions,
		RecentRCASummary: []interface{}{}, // TODO: wire in RCA summaries once available
	}

	respondJSON(w, http.StatusOK, detail)
}

// ListIncidentsWithRegion handles GET /admin/ops/incidents?region=us-east-1
// Returns incidents, optionally filtered by region
func (h *Handler) ListIncidentsWithRegion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	region := r.URL.Query().Get("region")
	limitStr := r.URL.Query().Get("limit")

	limit := 100
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 && v <= 1000 {
			limit = v
		}
	}

	var incidents []Incident
	var err error

	if region != "" {
		if !IsValidRegion(region) {
			http.Error(w, "invalid region format", http.StatusBadRequest)
			return
		}
		incidents, err = h.store.ListIncidentsByRegion(ctx, region, limit)
	} else {
		incidents, err = h.store.ListIncidents(ctx, limit)
	}

	if err != nil {
		http.Error(w, "failed to list incidents", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"incidents": incidents,
		"count":     len(incidents),
		"region":    region,
	})
}

// ========== Phase 3.10: Failover Policy Handlers ==========

// CreateFailoverPolicy handles POST /admin/ops/failover-policies
func (h *Handler) CreateFailoverPolicy(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name               string   `json:"name"`
		SourceRegion       string   `json:"source_region"`
		TargetRegions      []string `json:"target_regions"`
		TriggerHealthScore *int     `json:"trigger_health_score"`
		TriggerErrorRate   *float64 `json:"trigger_error_rate"`
		TriggerLatency     *int     `json:"trigger_latency_ms"`
		IsAutomatic        bool     `json:"is_automatic"`
		CooldownMinutes    int      `json:"cooldown_minutes"`
		Priority           int      `json:"priority"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Parse authenticated user context (from audit logger)
	tenantID := uuid.Nil // In production, get from auth context
	if req.CooldownMinutes == 0 {
		req.CooldownMinutes = 30 // Default 30 minute cooldown
	}

	// Marshal target regions as JSON
	targetRegionsJSON, _ := json.Marshal(req.TargetRegions)

	policy := &FailoverPolicy{
		TenantID:           tenantID,
		Name:               req.Name,
		SourceRegion:       req.SourceRegion,
		TargetRegions:      string(targetRegionsJSON),
		TriggerHealthScore: req.TriggerHealthScore,
		TriggerErrorRate:   req.TriggerErrorRate,
		TriggerLatency:     req.TriggerLatency,
		IsAutomatic:        req.IsAutomatic,
		CooldownMinutes:    req.CooldownMinutes,
		Priority:           req.Priority,
		IsEnabled:          true,
	}

	if err := h.store.InsertFailoverPolicy(r.Context(), policy); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"policy_id": policy.ID.String(),
		"policy":    policy,
	})
}

// ListFailoverPolicies handles GET /admin/ops/failover-policies
func (h *Handler) ListFailoverPolicies(w http.ResponseWriter, r *http.Request) {
	tenantID := uuid.Nil // In production, get from auth context

	policies, err := h.store.ListFailoverPolicies(r.Context(), tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Parse target regions for each policy
	var policiesWithTargets []map[string]interface{}
	for _, policy := range policies {
		var targets []string
		_ = json.Unmarshal([]byte(policy.TargetRegions), &targets)

		policiesWithTargets = append(policiesWithTargets, map[string]interface{}{
			"id":                   policy.ID.String(),
			"name":                 policy.Name,
			"source_region":        policy.SourceRegion,
			"target_regions":       targets,
			"trigger_health_score": policy.TriggerHealthScore,
			"trigger_error_rate":   policy.TriggerErrorRate,
			"trigger_latency_ms":   policy.TriggerLatency,
			"is_automatic":         policy.IsAutomatic,
			"cooldown_minutes":     policy.CooldownMinutes,
			"priority":             policy.Priority,
			"is_enabled":           policy.IsEnabled,
			"created_at":           policy.CreatedAt,
			"updated_at":           policy.UpdatedAt,
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"policies": policiesWithTargets,
		"count":    len(policiesWithTargets),
	})
}

// GetFailoverPolicy handles GET /admin/ops/failover-policies/{id}
func (h *Handler) GetFailoverPolicy(w http.ResponseWriter, r *http.Request) {
	policyID := chi.URLParam(r, "id")
	if policyID == "" {
		http.Error(w, "policy id is required", http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(policyID)
	if err != nil {
		http.Error(w, "invalid policy id", http.StatusBadRequest)
		return
	}

	policy, err := h.store.GetFailoverPolicy(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if policy == nil {
		http.Error(w, "policy not found", http.StatusNotFound)
		return
	}

	var targets []string
	_ = json.Unmarshal([]byte(policy.TargetRegions), &targets)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":                   policy.ID.String(),
		"name":                 policy.Name,
		"source_region":        policy.SourceRegion,
		"target_regions":       targets,
		"trigger_health_score": policy.TriggerHealthScore,
		"trigger_error_rate":   policy.TriggerErrorRate,
		"trigger_latency_ms":   policy.TriggerLatency,
		"is_automatic":         policy.IsAutomatic,
		"cooldown_minutes":     policy.CooldownMinutes,
		"priority":             policy.Priority,
		"is_enabled":           policy.IsEnabled,
		"created_at":           policy.CreatedAt,
		"updated_at":           policy.UpdatedAt,
	})
}

// UpdateFailoverPolicy handles PUT /admin/ops/failover-policies/{id}
func (h *Handler) UpdateFailoverPolicy(w http.ResponseWriter, r *http.Request) {
	policyID := chi.URLParam(r, "id")
	if policyID == "" {
		http.Error(w, "policy id is required", http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(policyID)
	if err != nil {
		http.Error(w, "invalid policy id", http.StatusBadRequest)
		return
	}

	var req struct {
		Name               string   `json:"name"`
		TargetRegions      []string `json:"target_regions"`
		TriggerHealthScore *int     `json:"trigger_health_score"`
		TriggerErrorRate   *float64 `json:"trigger_error_rate"`
		TriggerLatency     *int     `json:"trigger_latency_ms"`
		IsAutomatic        bool     `json:"is_automatic"`
		CooldownMinutes    int      `json:"cooldown_minutes"`
		Priority           int      `json:"priority"`
		IsEnabled          bool     `json:"is_enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Get existing policy
	policy, err := h.store.GetFailoverPolicy(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if policy == nil {
		http.Error(w, "policy not found", http.StatusNotFound)
		return
	}

	// Update fields
	policy.Name = req.Name
	if req.CooldownMinutes > 0 {
		policy.CooldownMinutes = req.CooldownMinutes
	}
	policy.Priority = req.Priority
	policy.IsAutomatic = req.IsAutomatic
	policy.IsEnabled = req.IsEnabled
	policy.TriggerHealthScore = req.TriggerHealthScore
	policy.TriggerErrorRate = req.TriggerErrorRate
	policy.TriggerLatency = req.TriggerLatency

	// Update target regions
	targetRegionsJSON, _ := json.Marshal(req.TargetRegions)
	policy.TargetRegions = string(targetRegionsJSON)

	if err := h.store.UpdateFailoverPolicy(r.Context(), id, policy); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"policy_id":  policy.ID.String(),
		"message":    "policy updated successfully",
		"updated_at": time.Now().UTC(),
	})
}

// ListFailoverEvents handles GET /admin/ops/failover-events?policy_id=...
func (h *Handler) ListFailoverEvents(w http.ResponseWriter, r *http.Request) {
	policyIDStr := r.URL.Query().Get("policy_id")
	if policyIDStr == "" {
		http.Error(w, "policy_id is required", http.StatusBadRequest)
		return
	}

	policyID, err := uuid.Parse(policyIDStr)
	if err != nil {
		http.Error(w, "invalid policy_id", http.StatusBadRequest)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		limit, _ = strconv.Atoi(limitStr)
	}

	events, err := h.store.ListFailoverEvents(r.Context(), policyID, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"events": events,
		"count":  len(events),
	})
}

// ========== Phase 3.11: Failover Chain Handlers ==========

// CreateFailoverChain handles POST /admin/ops/failover-chains
func (h *Handler) CreateFailoverChain(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name               string   `json:"name"`
		SourceRegion       string   `json:"source_region"`
		ChainTargets       []string `json:"chain_targets"`
		TriggerHealthScore *int     `json:"trigger_health_score"`
		TriggerErrorRate   *float64 `json:"trigger_error_rate"`
		TriggerLatency     *int     `json:"trigger_latency_ms"`
		MaxChainDepth      int      `json:"max_chain_depth"`
		CooldownMinutes    int      `json:"cooldown_minutes"`
		Priority           int      `json:"priority"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	tenantID := uuid.Nil // In production, get from auth context

	if req.MaxChainDepth == 0 {
		req.MaxChainDepth = 3
	}
	if req.CooldownMinutes == 0 {
		req.CooldownMinutes = 30
	}

	// Marshal chain targets as JSON
	chainTargetsJSON, _ := json.Marshal(req.ChainTargets)

	chain := &FailoverChain{
		TenantID:           tenantID,
		Name:               req.Name,
		SourceRegion:       req.SourceRegion,
		ChainTargets:       string(chainTargetsJSON),
		TriggerHealthScore: req.TriggerHealthScore,
		TriggerErrorRate:   req.TriggerErrorRate,
		TriggerLatency:     req.TriggerLatency,
		MaxChainDepth:      req.MaxChainDepth,
		CooldownMinutes:    req.CooldownMinutes,
		Priority:           req.Priority,
		IsEnabled:          true,
	}

	if err := h.store.InsertFailoverChain(r.Context(), chain); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"chain_id": chain.ID.String(),
		"chain":    chain,
	})
}

// ListFailoverChains handles GET /admin/ops/failover-chains
func (h *Handler) ListFailoverChains(w http.ResponseWriter, r *http.Request) {
	tenantID := uuid.Nil // In production, get from auth context

	chains, err := h.store.ListFailoverChains(r.Context(), tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Parse targets for each chain
	var chainsWithTargets []map[string]interface{}
	for _, chain := range chains {
		var targets []string
		_ = json.Unmarshal([]byte(chain.ChainTargets), &targets)

		chainsWithTargets = append(chainsWithTargets, map[string]interface{}{
			"id":                   chain.ID.String(),
			"name":                 chain.Name,
			"source_region":        chain.SourceRegion,
			"chain_targets":        targets,
			"trigger_health_score": chain.TriggerHealthScore,
			"trigger_error_rate":   chain.TriggerErrorRate,
			"trigger_latency_ms":   chain.TriggerLatency,
			"max_chain_depth":      chain.MaxChainDepth,
			"cooldown_minutes":     chain.CooldownMinutes,
			"priority":             chain.Priority,
			"is_enabled":           chain.IsEnabled,
			"created_at":           chain.CreatedAt,
			"updated_at":           chain.UpdatedAt,
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"chains": chainsWithTargets,
		"count":  len(chainsWithTargets),
	})
}

// GetFailoverChain handles GET /admin/ops/failover-chains/{id}
func (h *Handler) GetFailoverChain(w http.ResponseWriter, r *http.Request) {
	chainID := chi.URLParam(r, "id")
	if chainID == "" {
		http.Error(w, "chain id is required", http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(chainID)
	if err != nil {
		http.Error(w, "invalid chain id", http.StatusBadRequest)
		return
	}

	chain, err := h.store.GetFailoverChain(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if chain == nil {
		http.Error(w, "chain not found", http.StatusNotFound)
		return
	}

	var targets []string
	_ = json.Unmarshal([]byte(chain.ChainTargets), &targets)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":                   chain.ID.String(),
		"name":                 chain.Name,
		"source_region":        chain.SourceRegion,
		"chain_targets":        targets,
		"trigger_health_score": chain.TriggerHealthScore,
		"trigger_error_rate":   chain.TriggerErrorRate,
		"trigger_latency_ms":   chain.TriggerLatency,
		"max_chain_depth":      chain.MaxChainDepth,
		"cooldown_minutes":     chain.CooldownMinutes,
		"priority":             chain.Priority,
		"is_enabled":           chain.IsEnabled,
		"created_at":           chain.CreatedAt,
		"updated_at":           chain.UpdatedAt,
	})
}

// ListFailoverChainExecutions handles GET /admin/ops/failover-chain-executions?chain_id=...
func (h *Handler) ListFailoverChainExecutions(w http.ResponseWriter, r *http.Request) {
	chainIDStr := r.URL.Query().Get("chain_id")
	if chainIDStr == "" {
		http.Error(w, "chain_id is required", http.StatusBadRequest)
		return
	}

	chainID, err := uuid.Parse(chainIDStr)
	if err != nil {
		http.Error(w, "invalid chain_id", http.StatusBadRequest)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		limit, _ = strconv.Atoi(limitStr)
	}

	executions, err := h.store.ListFailoverChainExecutions(r.Context(), chainID, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"executions": executions,
		"count":      len(executions),
	})
}
