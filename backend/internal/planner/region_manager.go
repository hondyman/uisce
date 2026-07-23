package planner

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RegionManager handles region health and availability information
type RegionManager struct {
	store *Store
	cache map[string]*RegionPerformance
	mu    sync.RWMutex
	ttl   time.Duration
}

// NewRegionManager creates a region manager
func NewRegionManager(store *Store, cacheTTL time.Duration) *RegionManager {
	return &RegionManager{
		store: store,
		cache: make(map[string]*RegionPerformance),
		ttl:   cacheTTL,
	}
}

// GetAllRegionHealth returns current health status for all regions
func (rm *RegionManager) GetAllRegionHealth(ctx context.Context) (map[string]*RegionPerformance, error) {
	// Try cache first
	rm.mu.RLock()
	if len(rm.cache) > 0 {
		// Check if cache is fresh
		if time.Since(rm.cache["_last_update"].LastUpdated) < rm.ttl {
			defer rm.mu.RUnlock()
			result := make(map[string]*RegionPerformance)
			for k, v := range rm.cache {
				if k != "_last_update" {
					result[k] = v
				}
			}
			return result, nil
		}
	}
	rm.mu.RUnlock()

	// Load from database
	regions, err := rm.store.ListAllRegionPerformance(ctx)
	if err != nil {
		return nil, err
	}

	// Update cache
	rm.mu.Lock()
	rm.cache = regions
	// Store last update time (hack: use a sentinel entry)
	if len(regions) > 0 {
		rm.cache["_last_update"] = &RegionPerformance{
			LastUpdated: time.Now(),
		}
	}
	rm.mu.Unlock()

	return regions, nil
}

// GetRegionHealth returns health for a specific region
func (rm *RegionManager) GetRegionHealth(ctx context.Context, region string) (*RegionPerformance, error) {
	perf, err := rm.store.GetRegionPerformance(ctx, region)
	if err != nil {
		return nil, err
	}
	return perf, nil
}

// InvalidateCache clears the region health cache (e.g., after admin change)
func (rm *RegionManager) InvalidateCache() {
	rm.mu.Lock()
	rm.cache = make(map[string]*RegionPerformance)
	rm.mu.Unlock()
}

// CostModel estimates cost and latency for a query plan
type CostModel struct {
	// Base costs per engine type (abstract units)
	baseCosts map[string]float64

	// Latency overheads per engine type (ms)
	latencyOverheads map[string]float64
}

// NewCostModel creates a cost model
func NewCostModel() *CostModel {
	return &CostModel{
		baseCosts: map[string]float64{
			"trino":             1.0,
			"ts_service":        1.5,
			"drift_service":     2.0,
			"discovery_service": 2.5,
		},
		latencyOverheads: map[string]float64{
			"trino":             100.0,
			"ts_service":        150.0,
			"drift_service":     200.0,
			"discovery_service": 250.0,
		},
	}
}

// EstimateCostAndLatency calculates cost and latency for a query plan
func (cm *CostModel) EstimateCostAndLatency(
	queryType string,
	planType string,
	regionCount int,
	priority string,
	regionHealth map[string]*RegionPerformance,
	selectedRegions []string,
) (cost float64, latency float64) {
	// Base cost depends on query type
	baseCost := cm.baseCosts["trino"]
	switch queryType {
	case "ts":
		baseCost = cm.baseCosts["ts_service"]
	case "drift":
		baseCost = cm.baseCosts["drift_service"]
	case "importance", "discovery":
		baseCost = cm.baseCosts["trino"]
	}

	// Multiply by region count
	cost = baseCost * float64(regionCount)

	// Adjust cost by priority
	switch priority {
	case "batch":
		cost *= 0.8 // Batch can be cheaper
	case "background":
		cost *= 0.6
	case "interactive":
		cost *= 1.0 // No change
	}

	// Estimate latency based on region RTT
	latency = cm.estimateLatencyForRegions(regionHealth, selectedRegions)

	// Add engine overhead
	engineOverhead := cm.latencyOverheads["trino"]
	switch queryType {
	case "ts":
		engineOverhead = cm.latencyOverheads["ts_service"]
	case "drift":
		engineOverhead = cm.latencyOverheads["drift_service"]
	}
	latency += engineOverhead

	// For multi-region fan-out, latency is max of all regions (parallel)
	// For single region, it's just that region's latency
	if planType == "multi_region_fanout" {
		// Already computed as max
	}

	// Adjust for priority (interactive gets optimized routing)
	if priority == "interactive" {
		latency *= 0.9
	}

	return cost, latency
}

func (cm *CostModel) estimateLatencyForRegions(
	regionHealth map[string]*RegionPerformance,
	regions []string,
) float64 {
	if len(regions) == 0 {
		return 1000.0 // Default high latency
	}

	var maxLatency float64
	for _, region := range regions {
		perf, ok := regionHealth[region]
		if !ok || perf.LatencyP99MS == nil {
			maxLatency = 1000.0 // Default
			continue
		}

		if *perf.LatencyP99MS > maxLatency {
			maxLatency = *perf.LatencyP99MS
		}
	}

	if maxLatency == 0 {
		return 500.0 // Default fallback
	}

	return maxLatency
}

// QueryPlanValidator checks if a plan is feasible
type QueryPlanValidator struct {
	maxLatencyMS int     // max latency tolerance (ms)
	maxCost      float64 // max cost tolerance
}

// NewQueryPlanValidator creates a validator
func NewQueryPlanValidator(maxLatencyMS int, maxCost float64) *QueryPlanValidator {
	return &QueryPlanValidator{
		maxLatencyMS: maxLatencyMS,
		maxCost:      maxCost,
	}
}

// Validate checks if a plan meets constraints
func (v *QueryPlanValidator) Validate(plan *QueryPlan) error {
	if plan.EstimatedLatencyMS > float64(v.maxLatencyMS) {
		return fmt.Errorf(
			"plan exceeds max latency: estimated %.0fms > max %dms",
			plan.EstimatedLatencyMS,
			v.maxLatencyMS,
		)
	}

	if plan.EstimatedCost > v.maxCost {
		return fmt.Errorf(
			"plan exceeds max cost: estimated %.2f > max %.2f",
			plan.EstimatedCost,
			v.maxCost,
		)
	}

	if len(plan.SelectedRegions) == 0 {
		return fmt.Errorf("plan has no selected regions")
	}

	if len(plan.EngineRoutes) == 0 {
		return fmt.Errorf("plan has no engine routes")
	}

	return nil
}
