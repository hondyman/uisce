package planner

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
)

// PlannerStore is the minimal store interface used by the Planner
type PlannerStore interface {
	SaveDecision(ctx context.Context, req *QueryRequest, plan *QueryPlan, regionHealth interface{}) error
	GetFeaturePlannerConfig(ctx context.Context, featureID string) (*FeaturePlannerConfig, error)
	UpdateDecisionExecution(ctx context.Context, planID string, actualLatencyMS float64, actualCost float64, status string, errMsg string) error
	GetDecision(ctx context.Context, planID string) (*PlannerDecision, error)
	GetSLOCompliance(ctx context.Context, queryType string, hoursBack int) (*SLOCompliance, error)
	GetDecisionsForTarget(ctx context.Context, semanticTarget string, limit int) ([]PlannerDecision, error)
}

// RegionManagerAPI is the minimal interface used by Planner for region data
type RegionManagerAPI interface {
	GetAllRegionHealth(ctx context.Context) (map[string]*RegionPerformance, error)
	GetRegionHealth(ctx context.Context, region string) (*RegionPerformance, error)
	InvalidateCache()
}

// Planner is the main query planning engine
type Planner struct {
	store     PlannerStore
	regionMgr RegionManagerAPI
	costModel *CostModel
}

// NewPlanner creates a new planner instance
func NewPlanner(store PlannerStore, regionMgr RegionManagerAPI) *Planner {
	return &Planner{
		store:     store,
		regionMgr: regionMgr,
		costModel: NewCostModel(),
	}
}

// Plan generates a query execution plan
func (p *Planner) Plan(ctx context.Context, req *QueryRequest) (*QueryPlan, error) {
	// 1. Get region health
	regionHealth, err := p.regionMgr.GetAllRegionHealth(ctx)
	if err != nil {
		return nil, fmt.Errorf("get region health: %w", err)
	}

	// 2. Get feature planner config (if applicable)
	var featureConfig *FeaturePlannerConfig
	if req.SemanticTarget != "" {
		featureConfig, err = p.store.GetFeaturePlannerConfig(ctx, req.SemanticTarget)
		if err != nil {
			// Log but don't fail
			fmt.Printf("failed to get feature config: %v\n", err)
		}
	}

	// 3. Apply region selection logic
	selectedRegions := p.selectRegions(req, featureConfig, regionHealth)
	if len(selectedRegions) == 0 {
		return nil, fmt.Errorf("no regions available for query")
	}

	// 4. Determine plan type based on query type and regions
	planType := p.determinePlanType(req, selectedRegions)

	// 5. Build engine routes
	engineRoutes := p.buildEngineRoutes(req, selectedRegions, planType)

	// 6. Estimate cost and latency
	estimatedCost, estimatedLatency := p.costModel.EstimateCostAndLatency(
		req.QueryType,
		planType,
		len(selectedRegions),
		req.Priority,
		regionHealth,
		selectedRegions,
	)

	// 7. Choose degradation strategy
	degradationStrategy := p.chooseDegradationStrategy(req, selectedRegions, regionHealth)

	// 8. Generate explain text
	explain := p.generateExplain(req, selectedRegions, planType, featureConfig)

	// 9. Create plan
	plan := &QueryPlan{
		PlanID:              uuid.New().String(),
		PlanType:            planType,
		SelectedRegions:     selectedRegions,
		EngineRoutes:        engineRoutes,
		EstimatedCost:       estimatedCost,
		EstimatedLatencyMS:  estimatedLatency,
		DegradationStrategy: degradationStrategy,
		Explain:             explain,
	}

	// 10. Persist decision
	err = p.store.SaveDecision(ctx, req, plan, regionHealth)
	if err != nil {
		// Log but don't fail - planner still works
		fmt.Printf("failed to save planner decision: %v\n", err)
	}

	return plan, nil
}

// selectRegions determines which region(s) to target
func (p *Planner) selectRegions(req *QueryRequest, featureConfig *FeaturePlannerConfig, regionHealth map[string]*RegionPerformance) []string {
	// Get available healthy regions
	healthyRegions := p.getHealthyRegions(regionHealth)
	if len(healthyRegions) == 0 {
		return []string{} // Will fail later
	}

	// Apply feature-specific disallowed regions
	if featureConfig != nil && len(featureConfig.DisallowedRegions) > 0 {
		disallowedSet := make(map[string]bool)
		for _, r := range featureConfig.DisallowedRegions {
			disallowedSet[r] = true
		}
		var filtered []string
		for _, r := range healthyRegions {
			if !disallowedSet[r] {
				filtered = append(filtered, r)
			}
		}
		healthyRegions = filtered
	}

	if len(healthyRegions) == 0 {
		return []string{} // All regions are disallowed
	}

	// Case 1: User provided region hint
	if req.RegionHint != "" {
		for _, r := range healthyRegions {
			if r == req.RegionHint {
				return []string{r}
			}
		}
		// Hint not available, fall through
	}

	// Case 2: Feature has preferred regions
	if featureConfig != nil && len(featureConfig.PreferredRegions) > 0 {
		var preferred []string
		for _, r := range featureConfig.PreferredRegions {
			for _, h := range healthyRegions {
				if r == h {
					preferred = append(preferred, r)
					break
				}
			}
		}
		if len(preferred) > 0 {
			// For regional queries, use first preferred region
			if req.QueryType == "feature" || req.QueryType == "metric" || req.QueryType == "ts" {
				return []string{preferred[0]}
			}
			// For global queries, use all preferred
			if req.QueryType == "drift" || req.QueryType == "importance" || req.QueryType == "discovery" {
				return preferred
			}
		}
	}

	// Case 3: Global queries → use all healthy regions
	if req.QueryType == "drift" || req.QueryType == "importance" || req.QueryType == "discovery" {
		return healthyRegions
	}

	// Case 4: Regional queries → pick lowest latency
	sort.Slice(healthyRegions, func(i, j int) bool {
		latencyI := 1000.0 // default if nil
		if perf, ok := regionHealth[healthyRegions[i]]; ok && perf.LatencyP50MS != nil {
			latencyI = *perf.LatencyP50MS
		}
		latencyJ := 1000.0
		if perf, ok := regionHealth[healthyRegions[j]]; ok && perf.LatencyP50MS != nil {
			latencyJ = *perf.LatencyP50MS
		}
		return latencyI < latencyJ
	})

	return []string{healthyRegions[0]}
}

// determinePlanType decides single vs multi vs federated
func (p *Planner) determinePlanType(req *QueryRequest, regions []string) string {
	if len(regions) == 0 {
		return "single_region"
	}

	if len(regions) == 1 {
		return "single_region"
	}

	// Multi-region: check if federated makes sense
	if req.QueryType == "drift" || req.QueryType == "importance" || req.QueryType == "discovery" {
		// Could use federated views or fan-out
		// For now, prefer fan-out (more parallelism)
		return "multi_region_fanout"
	}

	// Single region even if multiple available
	return "single_region"
}

// buildEngineRoutes creates execution routes for the plan
func (p *Planner) buildEngineRoutes(req *QueryRequest, regions []string, planType string) []EngineRoute {
	var routes []EngineRoute

	for _, region := range regions {
		var engineType, endpoint, catalog, notes string

		switch req.QueryType {
		case "feature", "metric":
			engineType = "trino"
			endpoint = fmt.Sprintf("https://trino.%s.internal", region)
			catalog = fmt.Sprintf("iceberg_%s", region)
			notes = "Feature/metric query via Trino"

		case "ts":
			engineType = "ts_service"
			endpoint = fmt.Sprintf("https://ts-service.%s.internal", region)
			notes = "Time-series service (forecasting, decomposition, anomalies)"

		case "drift":
			engineType = "drift_service"
			endpoint = fmt.Sprintf("https://drift-service.%s.internal", region)
			notes = "Drift detection service"
		case "importance", "discovery":
			engineType = "trino"
			endpoint = fmt.Sprintf("https://trino.%s.internal", region)
			catalog = fmt.Sprintf("iceberg_%s", region)
			notes = fmt.Sprintf("Global %s view via Trino", req.QueryType)

		default:
			continue
		}

		route := EngineRoute{
			EngineType: engineType,
			Region:     region,
			Endpoint:   endpoint,
			Catalog:    catalog,
			Notes:      notes,
		}

		routes = append(routes, route)
	}

	return routes
}

// chooseDegradationStrategy defines failure handling
func (p *Planner) chooseDegradationStrategy(req *QueryRequest, regions []string, regionHealth map[string]*RegionPerformance) DegradationStrategy {
	// Global queries allow partial results
	if req.QueryType == "drift" || req.QueryType == "importance" || req.QueryType == "discovery" {
		fallback := []string{}
		for region := range regionHealth {
			if !contains(regions, region) && regionHealth[region].IsHealthy {
				fallback = append(fallback, region)
			}
		}

		maxStaleness := "30m"
		if req.FreshnessRequirement != "" {
			maxStaleness = req.FreshnessRequirement
		}

		return DegradationStrategy{
			Mode:            "partial_results",
			FallbackRegions: fallback,
			MaxStaleness:    maxStaleness,
		}
	}

	// Interactive regional queries should provide a fallback region even when healthy
	if req.Priority == "interactive" {
		fallback := []string{}
		// Collect other healthy regions not in the selected set
		for r, perf := range regionHealth {
			if perf != nil && perf.IsHealthy && !contains(regions, r) {
				fallback = append(fallback, r)
			}
		}

		return DegradationStrategy{
			Mode:            "fallback_region",
			FallbackRegions: fallback,
			MaxStaleness:    "5m",
		}
	}

	// Batch queries fail fast
	return DegradationStrategy{
		Mode:            "fail_fast",
		FallbackRegions: nil,
		MaxStaleness:    "",
	}
}

// generateExplain creates human-readable explanation
func (p *Planner) generateExplain(req *QueryRequest, regions []string, planType string, featureConfig *FeaturePlannerConfig) string {
	var sb strings.Builder
	sb.WriteString("Plan type: " + planType + ". ")
	sb.WriteString("Query type: " + req.QueryType + ". ")
	sb.WriteString("Selected regions: " + strings.Join(regions, ", ") + ". ")

	if req.RegionHint != "" {
		sb.WriteString("Region hint: " + req.RegionHint + ". ")
	}
	if req.TenantID != "" {
		sb.WriteString("Tenant: " + req.TenantID + ". ")
	}
	if req.Priority != "" {
		sb.WriteString("Priority: " + req.Priority + ". ")
	}
	if featureConfig != nil {
		sb.WriteString("Using feature-specific config. ")
	}

	return sb.String()
}

// GetExplainPlan generates a detailed explanation for UI
func (p *Planner) GetExplainPlan(ctx context.Context, planID string) (*ExplainPlan, error) {
	decision, err := p.store.GetDecision(ctx, planID)
	if err != nil || decision == nil {
		return nil, fmt.Errorf("plan not found")
	}

	// Parse the raw request and plan
	var req QueryRequest
	var plan QueryPlan
	err = unmarshalJSON(decision.RawRequest, &req)
	if err != nil {
		return nil, err
	}
	err = unmarshalJSON(decision.RawPlan, &plan)
	if err != nil {
		return nil, err
	}

	// Build explanation
	explain := &ExplainPlan{
		PlanID: planID,
		Summary: ExplanationSummary{
			PlanType:  plan.PlanType,
			Regions:   plan.SelectedRegions,
			LatencyMS: plan.EstimatedLatencyMS,
			Cost:      plan.EstimatedCost,
			Degraded:  decision.ExecutionStatus == "partial_failure",
		},
		Routing: ExplanationRouting{
			SelectedRegions:   plan.SelectedRegions,
			FallbackRegions:   plan.DegradationStrategy.FallbackRegions,
			Consistency:       req.ConsistencyLevel,
			FreshnessRequired: req.FreshnessRequirement,
		},
		Engines: []ExplanationEngine{},
		Explain: ExplainDetails{
			DecisionText:              plan.Explain,
			RegionSelectionReason:     p.explainRegionSelection(&req, plan.SelectedRegions),
			EngineSelectionReason:     p.explainEngineSelection(req.QueryType),
			LatencyEstimateReason:     p.explainLatencyEstimate(&plan.EstimatedLatencyMS, decision.ActualLatencyMS),
			CostEstimateReason:        p.explainCostEstimate(req.QueryType, len(plan.SelectedRegions)),
			DegradationStrategyReason: p.explainDegradationStrategy(plan.DegradationStrategy),
		},
	}

	// Convert engine routes
	for _, route := range plan.EngineRoutes {
		explain.Engines = append(explain.Engines, ExplanationEngine{
			EngineType: route.EngineType,
			Region:     route.Region,
			Endpoint:   route.Endpoint,
			Catalog:    route.Catalog,
			Notes:      route.Notes,
		})
	}

	return explain, nil
}

// Helper functions

func (p *Planner) getHealthyRegions(regionHealth map[string]*RegionPerformance) []string {
	var healthy []string
	for region, perf := range regionHealth {
		if perf != nil && perf.IsHealthy {
			healthy = append(healthy, region)
		}
	}
	// Sort for determinism
	sort.Strings(healthy)
	return healthy
}

func (p *Planner) explainRegionSelection(req *QueryRequest, regions []string) string {
	if len(regions) == 0 {
		return "No healthy regions available."
	}
	if len(regions) == 1 {
		return fmt.Sprintf("Selected %s (single region). %s are regional queries.", regions[0], req.QueryType)
	}
	return fmt.Sprintf("Selected %s for global query type %s.", strings.Join(regions, ", "), req.QueryType)
}

func (p *Planner) explainEngineSelection(queryType string) string {
	switch queryType {
	case "feature", "metric":
		return "Metric/feature queries use Trino for historical feature tables (Iceberg)."
	case "ts":
		return "Time-series queries use TS service (forecasting, decomposition, anomalies)."
	case "drift":
		return "Drift detection uses dedicated Drift service for statistical tests."
	case "importance", "discovery":
		return fmt.Sprintf("%s queries use Trino for federated views across regions.", queryType)
	default:
		return "Default engine selection applied."
	}
}

func (p *Planner) explainLatencyEstimate(estimated *float64, actual *float64) string {
	if actual == nil {
		return fmt.Sprintf("Estimated latency: %.0fms.", *estimated)
	}
	accuracy := 100.0 - ((*actual - *estimated) / *estimated * 100.0)
	return fmt.Sprintf("Estimated: %.0fms, Actual: %.0fms (accuracy: %.1f%%).", *estimated, *actual, accuracy)
}

func (p *Planner) explainCostEstimate(queryType string, regionCount int) string {
	baseCost := 1.0
	totalCost := baseCost * float64(regionCount)
	return fmt.Sprintf("Cost estimate: %.2f (base %.2f × %d regions).", totalCost, baseCost, regionCount)
}

func (p *Planner) explainDegradationStrategy(strategy DegradationStrategy) string {
	return fmt.Sprintf("Degradation mode: %s, max staleness: %s.", strategy.Mode, strategy.MaxStaleness)
}

func unmarshalJSON(data []byte, v interface{}) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, v)
}

func contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}
