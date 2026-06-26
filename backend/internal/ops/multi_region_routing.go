package ops

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// Phase 3.2: Multi-Region Routing Layer
// Tenant → region → infrastructure service discovery & routing decisions
// ============================================================================

// MultiRegionRoutingEngine manages intelligent routing decisions across regions
type MultiRegionRoutingEngine struct {
	mu                sync.RWMutex
	regionRouter      RegionRouter
	tenantPreferences map[string]*TenantRegionPreference
	routingCache      map[string]*RoutingDecision
	cacheExpiry       time.Duration
	lastCacheUpdate   time.Time
}

// NewMultiRegionRoutingEngine creates a new routing engine
func NewMultiRegionRoutingEngine(regionRouter RegionRouter) *MultiRegionRoutingEngine {
	return &MultiRegionRoutingEngine{
		regionRouter:      regionRouter,
		tenantPreferences: make(map[string]*TenantRegionPreference),
		routingCache:      make(map[string]*RoutingDecision),
		cacheExpiry:       1 * time.Minute,
	}
}

// TenantRegionPreference captures how a tenant should be routed
type TenantRegionPreference struct {
	TenantID           string
	PreferredRegion    string
	AllowedRegions     []string
	FallbackOrder      []string // Order to try regions if primary fails
	LocalityPreference string   // "latency", "data_residency", "cost"
	DataResidencyRule  string   // "must_stay_region" or "can_cross"
	CostThreshold      float64  // Max acceptable cost multiplier vs cheapest
	LatencyThreshold   int64    // Max acceptable latency in ms
}

// RoutingDecision represents a decision to route to a specific region
type RoutingDecision struct {
	TenantID            string
	RequestType         string // "query", "materialization", "drift_detection"
	SelectedRegion      string
	Rationale           string
	AlternateRegions    []string
	EstimatedLatencyMs  int64
	EstimatedCostFactor float64
	HealthScoreOfRegion float64
	IsFallback          bool // True if primary region unavailable
	DecisionTime        time.Time
	ValidUntil          time.Time
}

// RoutingContext provides context for making routing decisions
type RoutingContext struct {
	RequestMetadata         map[string]string
	UserLocation            *string // IP/geo if available
	DataRequirements        []string
	PerformanceRequirements *PerformanceRequirements
	CostContext             *CostContext
}

// PerformanceRequirements specifies performance expectations
type PerformanceRequirements struct {
	MaxLatencyMs   int64
	MaxFailureRate float64
	RequiredUptime float64 // 0.999 = 99.9%
}

// CostContext specifies cost constraints
type CostContext struct {
	Budget          float64
	CostSensitivity string // "high", "medium", "low"
	PreferCheapest  bool
}

// GetRoutingDecision determines which region to route a request to
func (e *MultiRegionRoutingEngine) GetRoutingDecision(
	ctx context.Context,
	tenantID string,
	requestType string,
	routingContext *RoutingContext,
) (*RoutingDecision, error) {

	// Step 1: Get tenant preferences
	preference, err := e.getTenantPreferences(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant preferences: %w", err)
	}

	// Step 2: Evaluate candidate regions
	candidates := e.evaluateRegionCandidates(ctx, preference, requestType, routingContext)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no viable regions available for tenant %s", tenantID)
	}

	// Step 3: Select best region based on optimization criteria
	selectedDecision := e.selectBestRegion(candidates, preference, routingContext)

	e.mu.Lock()
	e.routingCache[fmt.Sprintf("%s:%s", tenantID, requestType)] = selectedDecision
	e.lastCacheUpdate = time.Now()
	e.mu.Unlock()

	return selectedDecision, nil
}

// evaluateRegionCandidates scores all viable regions for routing
func (e *MultiRegionRoutingEngine) evaluateRegionCandidates(
	ctx context.Context,
	preference *TenantRegionPreference,
	requestType string,
	routingContext *RoutingContext,
) []*RoutingDecision {

	candidates := make([]*RoutingDecision, 0)

	// Determine which regions are allowed
	allowedRegions := preference.AllowedRegions
	if len(allowedRegions) == 0 {
		// Get all active regions if no preference
		targets, err := e.regionRouter.ListRegionTargets(ctx)
		if err == nil {
			for region := range targets {
				allowedRegions = append(allowedRegions, region)
			}
		}
	}

	// Evaluate each region
	for _, region := range allowedRegions {
		target, err := e.regionRouter.GetRegionTarget(ctx, region)
		if err != nil {
			continue
		}

		if !target.IsActive {
			continue
		}

		decision := &RoutingDecision{
			TenantID:       preference.TenantID,
			RequestType:    requestType,
			SelectedRegion: region,
			DecisionTime:   time.Now(),
			ValidUntil:     time.Now().Add(e.cacheExpiry),
		}

		// Calculate metrics for this region
		decision.HealthScoreOfRegion = e.getRegionHealthScore(ctx, region)
		decision.EstimatedLatencyMs = e.getEstimatedLatency(ctx, region, routingContext)
		decision.EstimatedCostFactor = e.getEstimatedCost(ctx, region, preference, requestType)

		// Determine rationale
		decision.Rationale = fmt.Sprintf(
			"Region %s: Health=%.2f, Latency=%dms, Cost=%.2fx",
			region,
			decision.HealthScoreOfRegion,
			decision.EstimatedLatencyMs,
			decision.EstimatedCostFactor,
		)

		// Check if region meets constraints
		if routingContext.PerformanceRequirements != nil {
			if decision.EstimatedLatencyMs > routingContext.PerformanceRequirements.MaxLatencyMs {
				continue
			}
		}

		if routingContext.CostContext != nil {
			if decision.EstimatedCostFactor > routingContext.CostContext.Budget {
				continue
			}
		}

		candidates = append(candidates, decision)
	}

	return candidates
}

// selectBestRegion chooses the optimal region from candidates
func (e *MultiRegionRoutingEngine) selectBestRegion(
	candidates []*RoutingDecision,
	preference *TenantRegionPreference,
	routingContext *RoutingContext,
) *RoutingDecision {

	if len(candidates) == 0 {
		return nil
	}

	// Scoring function
	type scoredCandidate struct {
		decision *RoutingDecision
		score    float64
	}

	scored := make([]scoredCandidate, 0, len(candidates))

	for _, decision := range candidates {
		score := 0.0

		// Priority 1: Preferred region
		if decision.SelectedRegion == preference.PreferredRegion {
			score += 100.0
		}

		// Priority 2: Locality preference
		switch preference.LocalityPreference {
		case "latency":
			// Lower latency = higher score
			score += 50.0 - float64(decision.EstimatedLatencyMs)/10.0
		case "data_residency":
			// Stay in preferred region
			if decision.SelectedRegion == preference.PreferredRegion {
				score += 50.0
			}
		case "cost":
			// Lower cost = higher score
			score += (2.0 - decision.EstimatedCostFactor) * 25.0
		}

		// Priority 3: Health score
		score += decision.HealthScoreOfRegion * 20.0

		// Priority 4: Cost constraint (apply penalty if over budget)
		if routingContext.CostContext != nil && decision.EstimatedCostFactor > routingContext.CostContext.Budget {
			score -= 30.0
		}

		// Priority 5: Latency constraint (apply penalty if over threshold)
		if routingContext.PerformanceRequirements != nil &&
			decision.EstimatedLatencyMs > routingContext.PerformanceRequirements.MaxLatencyMs {
			score -= 30.0
		}

		scored = append(scored, scoredCandidate{decision, score})
	}

	// Find max score
	maxIdx := 0
	maxScore := scored[0].score
	for i, candidate := range scored {
		if candidate.score > maxScore {
			maxScore = candidate.score
			maxIdx = i
		}
	}

	selected := scored[maxIdx].decision
	selected.AlternateRegions = make([]string, 0)
	for i, candidate := range scored {
		if i != maxIdx {
			selected.AlternateRegions = append(selected.AlternateRegions, candidate.decision.SelectedRegion)
		}
	}

	return selected
}

// SetTenantRegionPreference updates routing preferences for a tenant
func (e *MultiRegionRoutingEngine) SetTenantRegionPreference(
	ctx context.Context,
	tenantID string,
	preference *TenantRegionPreference,
) error {

	e.mu.Lock()
	e.tenantPreferences[tenantID] = preference
	e.mu.Unlock()

	return nil
}

// getTenantPreferences retrieves preferences for a tenant
func (e *MultiRegionRoutingEngine) getTenantPreferences(
	ctx context.Context,
	tenantID string,
) (*TenantRegionPreference, error) {

	e.mu.RLock()
	if pref, exists := e.tenantPreferences[tenantID]; exists {
		defer e.mu.RUnlock()
		return pref, nil
	}
	e.mu.RUnlock()

	// Default: use first available region
	preference := &TenantRegionPreference{
		TenantID:           tenantID,
		LocalityPreference: "cost",
		DataResidencyRule:  "can_cross",
		LatencyThreshold:   500,
	}

	// Try to load from region router
	// Parse tenant ID as UUID (with fallback if not a valid UUID)
	var tenantUUID uuid.UUID
	parsed, err := uuid.Parse(tenantID)
	if err == nil {
		tenantUUID = parsed
	} else {
		// Generate deterministic UUID from string for testing
		tenantUUID = uuid.NewSHA1(uuid.Nil, []byte(tenantID))
	}

	allowedRegions, err := e.regionRouter.GetTenantAllowedRegions(ctx, tenantUUID)
	if err == nil && len(allowedRegions) > 0 {
		preference.AllowedRegions = allowedRegions
		preference.PreferredRegion = allowedRegions[0]
		preference.FallbackOrder = allowedRegions[1:]
	}

	return preference, nil
}

// getRegionHealthScore gets the current health score for a region
func (e *MultiRegionRoutingEngine) getRegionHealthScore(ctx context.Context, region string) float64 {
	target, err := e.regionRouter.GetRegionTarget(ctx, region)
	if err != nil {
		return 0.5
	}

	// Score based on active status and time since last health check
	if !target.IsActive {
		return 0.2
	}

	timeSinceCheck := time.Since(target.LastHealthCheckAt)
	if timeSinceCheck < 1*time.Minute {
		return 0.95
	} else if timeSinceCheck < 5*time.Minute {
		return 0.80
	}
	return 0.60
}

// getEstimatedLatency estimates latency to a region
func (e *MultiRegionRoutingEngine) getEstimatedLatency(
	ctx context.Context,
	region string,
	routingContext *RoutingContext,
) int64 {

	target, err := e.regionRouter.GetRegionTarget(ctx, region)
	if err != nil {
		return 100
	}

	// Base latency depends on distance
	baseLatency := int64(50)
	if region == "us-east-1" {
		baseLatency = 10
	} else if region == "eu-west-1" || region == "us-west-2" {
		baseLatency = 30
	} else if region == "ap-south-1" {
		baseLatency = 80
	}

	// Add variance based on time since last check
	timeSinceCheck := time.Since(target.LastHealthCheckAt)
	variance := int64(timeSinceCheck.Milliseconds() / 1000)
	if variance > 20 {
		variance = 20
	}

	return baseLatency + variance
}

// getEstimatedCost estimates cost factor for a region
func (e *MultiRegionRoutingEngine) getEstimatedCost(
	ctx context.Context,
	region string,
	preference *TenantRegionPreference,
	requestType string,
) float64 {

	// Base cost factors (relative to us-east-1)
	costFactors := map[string]float64{
		"us-east-1":  1.0,
		"us-west-2":  1.1,
		"eu-west-1":  1.2,
		"ap-south-1": 1.5,
	}

	factor, exists := costFactors[region]
	if !exists {
		factor = 1.2 // Default
	}

	// Adjust based on request type
	switch requestType {
	case "drift_detection":
		factor *= 0.8 // Cheaper for background jobs
	case "materialization":
		factor *= 1.3 // More expensive for materialization
	}

	return factor
}

// ForceRegionFailover forces routing to move to an alternate region
func (e *MultiRegionRoutingEngine) ForceRegionFailover(
	ctx context.Context,
	tenantID string,
	fromRegion string,
	toRegion string,
) error {

	preference, err := e.getTenantPreferences(ctx, tenantID)
	if err != nil {
		return err
	}

	// Update preference to prioritize alternate region
	if preference.PreferredRegion == fromRegion {
		preference.PreferredRegion = toRegion
		preference.FallbackOrder = append([]string{fromRegion}, preference.FallbackOrder...)
	}

	return e.SetTenantRegionPreference(ctx, tenantID, preference)
}

// InvalidateRegionCache clears cached routing decisions
func (e *MultiRegionRoutingEngine) InvalidateRegionCache() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.routingCache = make(map[string]*RoutingDecision)
}
