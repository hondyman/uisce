package cbo

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
)

// SemanticRepository resolves BO and pre-agg SQL
type SemanticRepository interface {
	ResolveBaseSQL(ctx context.Context, pc PlanContext) (string, error)
	ResolvePreAggSQL(ctx context.Context, pc PlanContext, preAgg PreAggDescriptor) (string, error)
}

// PreAggRepository provides access to pre-aggregation metadata
type PreAggRepository interface {
	ListForBO(ctx context.Context, env string, tenantID *uuid.UUID, boName string, region string) ([]PreAggDescriptor, error)
}

// TelemetryRepository provides access to telemetry-derived features
type TelemetryRepository interface {
	GetBOFeatures(ctx context.Context, env string, tenantID *uuid.UUID, boName string, window string) (*BOFeatures, error)
	GetPreAggFeatures(ctx context.Context, env string, tenantID *uuid.UUID, preAggName string, window string) (*PreAggFeatures, error)
	RecordPlannerTelemetry(ctx context.Context, record *PlannerTelemetryRecord) error
	RecordSemanticEvent(ctx context.Context, record *SemanticEventRecord) error
}

// EntitlementRepository provides entitlement policy information
type EntitlementRepository interface {
	GetPoliciesForBO(ctx context.Context, tenantID *uuid.UUID, boName string) ([]EntitlementPolicy, error)
}

// EntitlementPolicy represents an entitlement policy
type EntitlementPolicy struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	BOName       string    `json:"bo_name"`
	Strategy     string    `json:"strategy"` // "join", "prefilter", "inline"
	OverheadMs   float64   `json:"overhead_ms"`
	FilterColumn string    `json:"filter_column,omitempty"`
}

// SLOProvider provides SLO constraints for planning
type SLOProvider interface {
	ForBO(ctx context.Context, env string, tenantID *uuid.UUID, boName string) *QuerySLO
	ForPage(ctx context.Context, env string, tenantID *uuid.UUID, pageSlug string) *QuerySLO
}

// QueryPlanner is the interface for the cost-based query planner
type QueryPlanner interface {
	Plan(ctx context.Context, pc PlanContext) (PlannedQuery, error)
}

// Planner is the implementation of the cost-based query planner
type Planner struct {
	semanticRepo    SemanticRepository
	preAggRepo      PreAggRepository
	entitlementRepo EntitlementRepository
	telemetryRepo   TelemetryRepository
	sloProvider     SLOProvider
}

// NewPlanner creates a new adaptive cost-based query planner
func NewPlanner(
	semanticRepo SemanticRepository,
	preAggRepo PreAggRepository,
	entitlementRepo EntitlementRepository,
	telemetryRepo TelemetryRepository,
	sloProvider SLOProvider,
) *Planner {
	return &Planner{
		semanticRepo:    semanticRepo,
		preAggRepo:      preAggRepo,
		entitlementRepo: entitlementRepo,
		telemetryRepo:   telemetryRepo,
		sloProvider:     sloProvider,
	}
}

// Plan generates the optimal query plan for the given context
func (p *Planner) Plan(ctx context.Context, pc PlanContext) (PlannedQuery, error) {
	startTime := time.Now()

	// Get SLO if not provided
	if pc.SLO == nil && p.sloProvider != nil {
		pc.SLO = p.sloProvider.ForBO(ctx, pc.Env, pc.TenantID, pc.BOName)
	}

	// Merge with Page SLO if provided
	if pc.PageSlug != "" && p.sloProvider != nil {
		pageSlo := p.sloProvider.ForPage(ctx, pc.Env, pc.TenantID, pc.PageSlug)
		if pageSlo != nil {
			pc.SLO = p.mergeSLOs(pc.SLO, pageSlo)
		}
	}

	// Merge with API SLO if provided
	if pc.ApiID != "" && p.sloProvider != nil {
		apiSlo := p.sloProvider.ForPage(ctx, pc.Env, pc.TenantID, pc.ApiID) // Reusing ForPage for now
		if apiSlo != nil {
			pc.SLO = p.mergeSLOs(pc.SLO, apiSlo)
		}
	}

	// Generate candidate plans
	candidates, err := p.generateCandidates(ctx, pc)
	if err != nil {
		log.Printf("[planner] Error generating candidates: %v", err)
		// Fall back to base plan
		return p.fallbackPlan(ctx, pc, startTime)
	}

	// Estimate costs for each candidate
	var scoredCandidates []PlannedQuery
	for _, candidate := range candidates {
		cost := p.estimateCost(ctx, candidate, pc)
		planned := PlannedQuery{
			SQL:                 candidate.SQL,
			PlanType:            candidate.PlanType,
			EntitlementStrategy: candidate.EntitlementStrategy,
			Cost:                cost,
		}
		if candidate.PreAgg != nil {
			planned.PreAggName = &candidate.PreAgg.Name
			planned.PreAggID = &candidate.PreAgg.ID
		}
		scoredCandidates = append(scoredCandidates, planned)
	}

	// Select best plan that satisfies SLO
	best := p.selectBestPlan(scoredCandidates, pc.SLO)
	if best == nil {
		return p.fallbackPlan(ctx, pc, startTime)
	}

	best.CandidatesEvaluated = len(candidates)
	best.SLOSatisfied = p.satisfiesSLO(pc.SLO, best.Cost)
	best.PlanningTimeMs = float64(time.Since(startTime).Microseconds()) / 1000.0

	// Record telemetry
	p.logTelemetry(ctx, pc, best, startTime)

	log.Printf("[planner] Selected plan: type=%s, latency=%.2fms, candidates=%d, slo_satisfied=%v",
		best.PlanType, best.Cost.EstimatedLatencyMs, best.CandidatesEvaluated, best.SLOSatisfied)

	return *best, nil
}

// logTelemetry logs the planning result and semantic event
func (p *Planner) logTelemetry(ctx context.Context, pc PlanContext, best *PlannedQuery, startTime time.Time) {
	if p.telemetryRepo == nil {
		return
	}

	// Log Planner Telemetry
	telemetry := &PlannerTelemetryRecord{
		Env:                 pc.Env,
		TenantID:            pc.TenantID,
		BOName:              pc.BOName,
		PlanType:            string(best.PlanType),
		PreAggName:          best.PreAggName,
		PreAggID:            best.PreAggID,
		EntitlementStrategy: string(best.EntitlementStrategy),
		EstimatedLatencyMs:  best.Cost.EstimatedLatencyMs,
		ActualLatencyMs:     0, // Planned only
		EstimatedScanBytes:  best.Cost.EstimatedScanBytes,
		ActualScanBytes:     0,
		SLOSatisfied:        best.SLOSatisfied,
		CandidatesEvaluated: best.CandidatesEvaluated,
		PlanningTimeMs:      best.PlanningTimeMs,
		Success:             true,
		UserID:              pc.CurrentUserID,
	}

	// Fire and forget
	go func() {
		// Create a detached context for logging
		logCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := p.telemetryRepo.RecordPlannerTelemetry(logCtx, telemetry); err != nil {
			log.Printf("[planner] Failed to record planner telemetry: %v", err)
		}

		// Log Semantic Event (for Suggestion Engine)
		groupByJSON, _ := json.Marshal(pc.GroupBy)
		measureJSON, _ := json.Marshal(pc.Measures)

		var filterKeys []string
		if pc.Filters != nil {
			for k := range pc.Filters {
				filterKeys = append(filterKeys, k)
			}
		}
		filterKeysJSON, _ := json.Marshal(filterKeys)

		event := &SemanticEventRecord{
			Datasource:     pc.BOName, // Using BO Name as datasource
			SQLFingerprint: best.SQL,  // Using generated SQL as fingerprint
			SQLLatencyMs:   best.Cost.EstimatedLatencyMs,
			SQLRows:        int(best.Cost.EstimatedScanBytes / 100), // Crude estimate
			GroupByFields:  string(groupByJSON),
			FilterFields:   string(filterKeysJSON), // Suggestion engine expects field names
			MeasureFields:  string(measureJSON),
			PreAggID:       best.PreAggID,
			PreAggHit:      best.UsePreAgg(),
		}

		if pc.TenantID != nil {
			event.TenantID = *pc.TenantID
		}

		if err := p.telemetryRepo.RecordSemanticEvent(logCtx, event); err != nil {
			log.Printf("[planner] Failed to record semantic event: %v", err)
		}
	}()
}

// generateCandidates generates all candidate execution plans
func (p *Planner) generateCandidates(ctx context.Context, pc PlanContext) ([]CandidatePlan, error) {
	var candidates []CandidatePlan

	// 1. Base table plan (always available)
	if p.semanticRepo != nil {
		baseSQL, err := p.semanticRepo.ResolveBaseSQL(ctx, pc)
		if err == nil && baseSQL != "" {
			entStrategy := p.chooseEntitlementStrategy(ctx, nil, pc)
			candidates = append(candidates, CandidatePlan{
				SQL:                 baseSQL,
				PlanType:            PlanTypeBase,
				EntitlementStrategy: entStrategy,
			})
		}
	}

	// 2. Pre-aggregation plans
	if p.preAggRepo != nil {
		preAggs, err := p.preAggRepo.ListForBO(ctx, pc.Env, pc.TenantID, pc.BOName, pc.Region)
		if err == nil {
			for _, pa := range preAggs {
				if !p.isCompatible(pa, pc) {
					continue
				}

				var preAggSQL string
				if p.semanticRepo != nil {
					preAggSQL, _ = p.semanticRepo.ResolvePreAggSQL(ctx, pc, pa)
				}
				if preAggSQL == "" {
					// Generate simple SELECT from pre-agg table
					preAggSQL = "SELECT * FROM " + pa.TargetTable
				}

				paCopy := pa
				entStrategy := p.chooseEntitlementStrategy(ctx, &paCopy, pc)
				candidates = append(candidates, CandidatePlan{
					SQL:                 preAggSQL,
					PlanType:            PlanTypePreAgg,
					PreAgg:              &paCopy,
					EntitlementStrategy: entStrategy,
				})
			}
		}
	}

	// Ensure at least one candidate exists
	if len(candidates) == 0 {
		candidates = append(candidates, CandidatePlan{
			SQL:                 p.buildDefaultSQL(pc),
			PlanType:            PlanTypeBase,
			EntitlementStrategy: EntitlementJoin,
		})
	}

	return candidates, nil
}

// isCompatible checks if a pre-aggregation is compatible with the query
func (p *Planner) isCompatible(pa PreAggDescriptor, pc PlanContext) bool {
	// Check if all requested dimensions are available in the pre-agg
	dimSet := make(map[string]bool)
	for _, d := range pa.Dimensions {
		dimSet[d] = true
	}
	for _, reqDim := range pc.GroupBy {
		if !dimSet[reqDim] {
			return false
		}
	}
	for _, reqDim := range pc.Dimensions {
		if !dimSet[reqDim] {
			return false
		}
	}

	// Check if all requested measures are available
	measureSet := make(map[string]bool)
	for _, m := range pa.Measures {
		measureSet[m] = true
	}
	for _, reqMeasure := range pc.Measures {
		if !measureSet[reqMeasure] {
			return false
		}
	}

	return true
}

// chooseEntitlementStrategy selects the best entitlement strategy
func (p *Planner) chooseEntitlementStrategy(ctx context.Context, preAgg *PreAggDescriptor, pc PlanContext) EntitlementStrategy {
	if p.entitlementRepo == nil {
		return EntitlementNone
	}

	policies, err := p.entitlementRepo.GetPoliciesForBO(ctx, pc.TenantID, pc.BOName)
	if err != nil || len(policies) == 0 {
		return EntitlementNone
	}

	// Use the first policy's strategy for now
	// In production, we'd evaluate cost of each strategy
	for _, policy := range policies {
		switch policy.Strategy {
		case "prefilter":
			return EntitlementPrefilter
		case "inline":
			return EntitlementInline
		case "join":
			return EntitlementJoin
		}
	}

	return EntitlementJoin
}

// estimateCost estimates the cost of a candidate plan
func (p *Planner) estimateCost(ctx context.Context, candidate CandidatePlan, pc PlanContext) PlanCost {
	cost := PlanCost{
		EstimatedLatencyMs:   500, // default
		EstimatedScanBytes:   1_000_000,
		EstimatedComputeCost: 1.0,
		EntitlementCostMs:    10,
		FreshnessLagSec:      0,
	}

	// Use telemetry for base table plans
	if candidate.PlanType == PlanTypeBase && p.telemetryRepo != nil {
		bf, err := p.telemetryRepo.GetBOFeatures(ctx, pc.Env, pc.TenantID, pc.BOName, "7d")
		if err == nil && bf != nil {
			cost.EstimatedLatencyMs = bf.P95LatencyMs
			cost.EstimatedScanBytes = bf.AvgScanBytes
			cost.FreshnessLagSec = 0 // Base tables are always fresh
		}
	}

	// Use telemetry for pre-agg plans
	if candidate.PlanType == PlanTypePreAgg && candidate.PreAgg != nil {
		// Start with speedup from pre-agg descriptor
		if candidate.PreAgg.AvgSpeedup > 0 {
			// Estimate latency as base latency / speedup
			baseLatency := cost.EstimatedLatencyMs
			if p.telemetryRepo != nil {
				bf, _ := p.telemetryRepo.GetBOFeatures(ctx, pc.Env, pc.TenantID, pc.BOName, "7d")
				if bf != nil {
					baseLatency = bf.P95LatencyMs
				}
			}
			cost.EstimatedLatencyMs = baseLatency / candidate.PreAgg.AvgSpeedup
		} else {
			cost.EstimatedLatencyMs = 100 // Assume pre-aggs are fast
		}

		// Set freshness lag from refresh frequency
		cost.FreshnessLagSec = float64(candidate.PreAgg.RefreshFrequencySec)
		if candidate.PreAgg.LastRefreshAt != nil {
			actualLag := time.Since(*candidate.PreAgg.LastRefreshAt).Seconds()
			if actualLag > cost.FreshnessLagSec {
				cost.FreshnessLagSec = actualLag
			}
		}

		// Set storage from descriptor
		cost.EstimatedScanBytes = float64(candidate.PreAgg.StorageBytes)

		// Get more accurate features from telemetry
		if p.telemetryRepo != nil {
			pf, err := p.telemetryRepo.GetPreAggFeatures(ctx, pc.Env, pc.TenantID, candidate.PreAgg.Name, "7d")
			if err == nil && pf != nil {
				if pf.AvgSpeedup > 0 {
					baseLatency := cost.EstimatedLatencyMs * pf.AvgSpeedup
					cost.EstimatedLatencyMs = baseLatency / pf.AvgSpeedup
				}
				cost.FreshnessLagSec = pf.AvgFreshnessLagSec
			}
		}
	}

	// Estimate entitlement cost
	cost.EntitlementCostMs = p.estimateEntitlementCost(ctx, candidate.EntitlementStrategy, pc)

	// Compute cost is proportional to scan bytes
	cost.EstimatedComputeCost = cost.EstimatedScanBytes / 1_000_000_000.0

	return cost
}

// estimateEntitlementCost estimates the overhead of entitlement processing
func (p *Planner) estimateEntitlementCost(ctx context.Context, strategy EntitlementStrategy, pc PlanContext) float64 {
	switch strategy {
	case EntitlementNone:
		return 0
	case EntitlementPrefilter:
		return 5 // Fastest - uses cached entitlements
	case EntitlementInline:
		return 15 // Medium - inlines predicates
	case EntitlementJoin:
		return 30 // Slowest - requires join
	default:
		return 10
	}
}

// selectBestPlan selects the lowest cost plan that satisfies the SLO
func (p *Planner) selectBestPlan(candidates []PlannedQuery, slo *QuerySLO) *PlannedQuery {
	var best *PlannedQuery

	for i := range candidates {
		c := &candidates[i]

		// Skip if doesn't satisfy SLO
		if !p.satisfiesSLO(slo, c.Cost) {
			continue
		}

		// Select lowest cost
		if best == nil || c.Cost.TotalCost() < best.Cost.TotalCost() {
			best = c
		}
	}

	// If no plan satisfies SLO, return lowest latency plan
	if best == nil {
		for i := range candidates {
			c := &candidates[i]
			if best == nil || c.Cost.EstimatedLatencyMs < best.Cost.EstimatedLatencyMs {
				best = c
			}
		}
	}

	return best
}

// satisfiesSLO checks if a plan cost satisfies the SLO constraints
func (p *Planner) satisfiesSLO(slo *QuerySLO, cost PlanCost) bool {
	if slo == nil {
		return true
	}
	if slo.MaxP95LatencyMs != nil && cost.EstimatedLatencyMs > *slo.MaxP95LatencyMs {
		return false
	}
	if slo.MaxFreshnessLagSec != nil && cost.FreshnessLagSec > *slo.MaxFreshnessLagSec {
		return false
	}
	return true
}

// fallbackPlan returns a simple base plan when planning fails
func (p *Planner) fallbackPlan(ctx context.Context, pc PlanContext, startTime time.Time) (PlannedQuery, error) {
	sql := p.buildDefaultSQL(pc)

	return PlannedQuery{
		SQL:                 sql,
		PlanType:            PlanTypeBase,
		EntitlementStrategy: EntitlementJoin,
		Cost: PlanCost{
			EstimatedLatencyMs:   1000,
			EstimatedScanBytes:   10_000_000,
			EstimatedComputeCost: 10,
			EntitlementCostMs:    30,
		},
		CandidatesEvaluated: 0,
		SLOSatisfied:        false,
		PlanningTimeMs:      float64(time.Since(startTime).Microseconds()) / 1000.0,
	}, nil
}

// mergeSLOs combines two SLOs, prioritizing the more restrictive ones
func (p *Planner) mergeSLOs(base, override *QuerySLO) *QuerySLO {
	if base == nil {
		return override
	}
	if override == nil {
		return base
	}

	merged := *base
	if override.MaxP95LatencyMs != nil {
		if merged.MaxP95LatencyMs == nil || *override.MaxP95LatencyMs < *merged.MaxP95LatencyMs {
			merged.MaxP95LatencyMs = override.MaxP95LatencyMs
		}
	}
	if override.MaxFreshnessLagSec != nil {
		if merged.MaxFreshnessLagSec == nil || *override.MaxFreshnessLagSec < *merged.MaxFreshnessLagSec {
			merged.MaxFreshnessLagSec = override.MaxFreshnessLagSec
		}
	}
	if override.MaxErrorRate != nil {
		if merged.MaxErrorRate == nil || *override.MaxErrorRate < *merged.MaxErrorRate {
			merged.MaxErrorRate = override.MaxErrorRate
		}
	}

	return &merged
}

// buildDefaultSQL builds a simple SQL query
func (p *Planner) buildDefaultSQL(pc PlanContext) string {
	// This is a placeholder - real implementation would use the semantic repository
	return "SELECT * FROM " + pc.BOName
}
