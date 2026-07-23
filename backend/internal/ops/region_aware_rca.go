package ops

import (
	"fmt"
	"math"
)

// ============================================================================
// Phase 3.2: Region-Aware RCA Scoring
// Intelligent root cause analysis that factors region topology and propagation
// ============================================================================

// Helper to safely extract region from pointer
func getRegion(regionPtr *string) string {
	if regionPtr == nil {
		return "unknown"
	}
	return *regionPtr
}

// RegionAwareRCAScorer adds multi-region context to root cause analysis
type RegionAwareRCAScorer struct {
	regionRegistry *RegionRegistry
	baseScorer     *CorrelationEngine
}

// NewRegionAwareRCAScorer creates a region-aware RCA scorer
func NewRegionAwareRCAScorer(regionRegistry *RegionRegistry, baseScorer *CorrelationEngine) *RegionAwareRCAScorer {
	return &RegionAwareRCAScorer{
		regionRegistry: regionRegistry,
		baseScorer:     baseScorer,
	}
}

// RegionScoringContext provides region topology information for RCA
type RegionScoringContext struct {
	Regions            map[string]*RegionMetadata
	RegionAdjacency    map[string][]string // region -> neighboring regions
	CrossRegionLatency map[string]int64    // "region1->region2" -> latency_ms
}

// RegionMetadata contains scoring information about a region
type RegionMetadata struct {
	RegionCode   string
	RegionName   string
	Tier         string  // "us-east-1" is "tier-1", "eu-west-1" is "tier-1", "ap-south-1" is "tier-2"
	NumInstances int     // Number of instances/hosts in region
	IsHealthy    bool    // Overall health status
	HealthScore  float64 // 0.0-1.0
	AvgLatencyMS float64 // Average latency within region
}

// RegionAwareCorrelationScore extends CorrelationScore with region context
type RegionAwareCorrelationScore struct {
	BaseScore                    CorrelationScore
	FromRegion                   string
	ToRegion                     string
	SameRegion                   bool
	CrossRegionProximityBonus    float64 // Bonus if regions are adjacent
	CrossRegionLatencyPenalty    float64 // Penalty if latency is high
	RegionHealthContextScore     float64 // Factor in region health
	RegionAwareScore             float64 // Final region-adjusted score
	PropagationLikelihood        float64 // How likely is cross-region propagation
	IsLikelyPropagationPath      bool    // True if this could be a cross-region propagation
	EstimatedCrossRegionTravelMs int64   // Estimated time for issue to cross regions
}

// ScoringWeights control how much each factor contributes to region-aware scoring
type ScoringWeights struct {
	SameRegionWeight         float64
	AdjacentRegionWeight     float64
	DistalRegionWeight       float64
	HealthScoreWeight        float64
	LatencyImpactWeight      float64
	PropagationMinimumScore  float64 // Minimum score to consider cross-region propagation
	SeverityPropagationBoost float64 // Boost if severity is high (0.1-0.3)
}

// DefaultScoringWeights returns reasonable defaults for region-aware RCA scoring
func DefaultScoringWeights() ScoringWeights {
	return ScoringWeights{
		SameRegionWeight:         1.0,  // Same region: full weight
		AdjacentRegionWeight:     0.7,  // Adjacent: 70% of score
		DistalRegionWeight:       0.4,  // Distant regions: 40% of score
		HealthScoreWeight:        0.15, // Health factor: 15% impact
		LatencyImpactWeight:      0.1,  // Latency: 10% impact
		PropagationMinimumScore:  0.65, // Need 65% base correlation to propagate
		SeverityPropagationBoost: 0.15, // +15% boost if high severity
	}
}

// ComputeRegionAwareCorrelation scores correlation between events in different regions
func (s *RegionAwareRCAScorer) ComputeRegionAwareCorrelation(
	baseScore *CorrelationScore,
	fromEvent, toEvent *Event,
	context *RegionScoringContext,
	weights ScoringWeights,
) *RegionAwareCorrelationScore {

	result := &RegionAwareCorrelationScore{
		BaseScore:  *baseScore,
		FromRegion: getRegion(fromEvent.Region),
		ToRegion:   getRegion(toEvent.Region),
		SameRegion: getRegion(fromEvent.Region) == getRegion(toEvent.Region),
	}

	// Step 1: Determine region relationship
	var multiplier float64
	if result.SameRegion {
		multiplier = weights.SameRegionWeight
		result.PropagationLikelihood = 0.95 // High likelihood within same region
	} else {
		// Check region adjacency
		isAdjacent := s.isAdjacentRegion(getRegion(fromEvent.Region), getRegion(toEvent.Region), context)
		if isAdjacent {
			multiplier = weights.AdjacentRegionWeight
			result.CrossRegionProximityBonus = 0.15
			result.PropagationLikelihood = 0.65
		} else {
			multiplier = weights.DistalRegionWeight
			result.PropagationLikelihood = 0.35
		}

		result.IsLikelyPropagationPath = baseScore.Score >= weights.PropagationMinimumScore
	}

	// Step 2: Apply latency penalty for cross-region
	if !result.SameRegion {
		latency := s.getCrossRegionLatency(getRegion(fromEvent.Region), getRegion(toEvent.Region), context)
		result.EstimatedCrossRegionTravelMs = latency

		// High latency reduces likelihood of causal relationship
		latencyPenalty := math.Min(float64(latency)/1000.0, 0.5) * weights.LatencyImpactWeight
		result.CrossRegionLatencyPenalty = latencyPenalty
	}

	// Step 3: Apply health score factor
	fromHealth := s.getRegionHealthScore(getRegion(fromEvent.Region), context)
	toHealth := s.getRegionHealthScore(getRegion(toEvent.Region), context)
	healthFactor := (fromHealth + toHealth) / 2.0
	result.RegionHealthContextScore = healthFactor * weights.HealthScoreWeight

	// Step 4: Apply severity propagation boost
	severityBoost := 0.0
	if baseScore.Score >= 0.8 {
		severityBoost = weights.SeverityPropagationBoost
	}

	// Step 5: Compute final region-aware score
	baseScoreAdjusted := baseScore.Score * multiplier
	penalty := result.CrossRegionLatencyPenalty
	bonus := result.CrossRegionProximityBonus + result.RegionHealthContextScore + severityBoost

	result.RegionAwareScore = math.Max(0, math.Min(1.0, baseScoreAdjusted-penalty+bonus))

	return result
}

// ScoreRCAWithRegionContext adds region awareness to RCA results
func (s *RegionAwareRCAScorer) ScoreRCAWithRegionContext(
	baseRCA *RCAResult,
	context *RegionScoringContext,
	weights ScoringWeights,
) (*RCAResultWithRegionContext, error) {

	result := &RCAResultWithRegionContext{
		BaseRCA:                     *baseRCA,
		ScoredCorrelations:          make([]*RegionAwareCorrelationScore, 0),
		RegionCausalityChain:        make([]*RegionCausalityStep, 0),
		CrossRegionPropagationPaths: make([]*PropagationPath, 0),
	}

	// Step 1: Score root cause region
	if baseRCA.SuspectedRootCause != nil {
		result.RootCauseRegion = getRegion(baseRCA.SuspectedRootCause.Event.Region)
		result.RootCauseRegionHealth = s.getRegionHealthScore(result.RootCauseRegion, context)
	}

	// Step 2: Analyze causality chain with region context
	for i := 0; i < len(baseRCA.CausalityChain)-1; i++ {
		current := &baseRCA.CausalityChain[i]
		next := &baseRCA.CausalityChain[i+1]

		step := &RegionCausalityStep{
			FromEvent:          current.Event,
			ToEvent:            next.Event,
			CausalityScore:     current.CausalityScore,
			SameRegion:         getRegion(current.Event.Region) == getRegion(next.Event.Region),
			CrossRegionLatency: s.getCrossRegionLatency(getRegion(current.Event.Region), getRegion(next.Event.Region), context),
		}

		result.RegionCausalityChain = append(result.RegionCausalityChain, step)

		// Identify probable cross-region propagation paths
		if !step.SameRegion && current.CausalityScore >= 0.6 {
			path := &PropagationPath{
				FromRegion:   getRegion(current.Event.Region),
				ToRegion:     getRegion(next.Event.Region),
				HopCount:     1,
				EstimatedMs:  step.CrossRegionLatency,
				IsLikely:     current.CausalityScore >= 0.75,
				ReasonScores: current.CorrelatedEvents,
			}
			result.CrossRegionPropagationPaths = append(result.CrossRegionPropagationPaths, path)
		}
	}

	// Step 3: Determine region-scoped remediation strategy
	result.RemediationStrategy = s.determineRemediationStrategy(baseRCA, result, context)

	return result, nil
}

// RCAResultWithRegionContext extends RCAResult with region-aware analysis
type RCAResultWithRegionContext struct {
	BaseRCA                     RCAResult
	RootCauseRegion             string
	RootCauseRegionHealth       float64
	ScoredCorrelations          []*RegionAwareCorrelationScore
	RegionCausalityChain        []*RegionCausalityStep
	CrossRegionPropagationPaths []*PropagationPath
	AffectedRegions             []string
	RemediationStrategy         *RegionRemediationStrategy
}

// RegionCausalityStep represents causality within or across regions
type RegionCausalityStep struct {
	FromEvent          Event
	ToEvent            Event
	CausalityScore     float64
	SameRegion         bool
	CrossRegionLatency int64
}

// PropagationPath represents how an issue propagates across regions
type PropagationPath struct {
	FromRegion   string
	ToRegion     string
	HopCount     int
	EstimatedMs  int64
	IsLikely     bool
	ReasonScores []CorrelationScore
}

// RegionRemediationStrategy provides region-scoped remediation suggestions
type RegionRemediationStrategy struct {
	PrimaryRegionActions   []RemediationAction
	SecondaryRegionActions map[string][]RemediationAction // region -> actions
	IsolationStrategy      string                         // "isolate_region", "failover", "throttle"
	PropagationPrevention  []string                       // Actions to block cross-region propagation
}

// RemediationAction is a region-specific remediation
type RemediationAction struct {
	ActionType   string
	TargetRegion string
	Priority     string
	Confidence   float64
	Reason       string
}

// Helper methods

func (s *RegionAwareRCAScorer) isAdjacentRegion(from, to string, context *RegionScoringContext) bool {
	if from == to {
		return true
	}
	for _, neighbor := range context.RegionAdjacency[from] {
		if neighbor == to {
			return true
		}
	}
	return false
}

func (s *RegionAwareRCAScorer) getCrossRegionLatency(from, to string, context *RegionScoringContext) int64 {
	key := from + "->" + to
	if latency, exists := context.CrossRegionLatency[key]; exists {
		return latency
	}
	// Default if not specified
	return 100
}

func (s *RegionAwareRCAScorer) getRegionHealthScore(region string, context *RegionScoringContext) float64 {
	if meta, exists := context.Regions[region]; exists {
		return meta.HealthScore
	}
	return 0.7 // Default moderate health
}

func (s *RegionAwareRCAScorer) determineRemediationStrategy(
	baseRCA *RCAResult,
	regionAware *RCAResultWithRegionContext,
	context *RegionScoringContext,
) *RegionRemediationStrategy {

	strategy := &RegionRemediationStrategy{
		PrimaryRegionActions:   make([]RemediationAction, 0),
		SecondaryRegionActions: make(map[string][]RemediationAction),
		PropagationPrevention:  make([]string, 0),
	}

	rootRegion := regionAware.RootCauseRegion
	seenRegions := make(map[string]bool)

	// Primary actions in root cause region
	for _, suggestion := range baseRCA.SuggestedRemediations {
		action := RemediationAction{
			ActionType:   suggestion.ActionType,
			TargetRegion: rootRegion,
			Priority:     suggestion.Priority,
			Confidence:   suggestion.Confidence,
			Reason:       suggestion.Reason,
		}
		strategy.PrimaryRegionActions = append(strategy.PrimaryRegionActions, action)
	}

	// Secondary actions in affected regions
	for _, path := range regionAware.CrossRegionPropagationPaths {
		if !seenRegions[path.ToRegion] && path.IsLikely {
			seenRegions[path.ToRegion] = true

			// Add throttling/isolation in secondary regions
			secondaryAction := RemediationAction{
				ActionType:   "throttle_region",
				TargetRegion: path.ToRegion,
				Priority:     "medium",
				Confidence:   0.8,
				Reason:       fmt.Sprintf("Prevent propagation from %s (estimated travel: %d ms)", path.FromRegion, path.EstimatedMs),
			}
			strategy.SecondaryRegionActions[path.ToRegion] = append(strategy.SecondaryRegionActions[path.ToRegion], secondaryAction)

			// Track propagation prevention actions
			strategy.PropagationPrevention = append(strategy.PropagationPrevention,
				fmt.Sprintf("throttle-%s-%s", path.FromRegion, path.ToRegion))
		}
	}

	// Determine isolation strategy
	if len(regionAware.CrossRegionPropagationPaths) > 0 {
		strategy.IsolationStrategy = "isolate_region"
	} else {
		strategy.IsolationStrategy = "local_failover"
	}

	return strategy
}
