package ops

import (
	"math"
	"time"
)

// CorrelationScore represents how likely event A caused event B
type CorrelationScore struct {
	FromEventID   string             `json:"from_event_id"`
	ToEventID     string             `json:"to_event_id"`
	Score         float64            `json:"score"` // 0.0 to 1.0
	TimeGapMs     int64              `json:"time_gap_ms"`
	ReasonScores  map[string]float64 `json:"reason_scores"`
	PrimaryReason string             `json:"primary_reason"` // e.g., "temporal_proximity", "event_relationship", "severity_match"
}

// RCAResult contains intelligent root cause analysis with scoring
type RCAResult struct {
	SuspectedRootCause    *ScoredEvent            `json:"suspected_root_cause"`
	CausalityChain        []ScoredEvent           `json:"causality_chain"`   // Ordered events from root to effect
	AffectedServices      []string                `json:"affected_services"` // Distinct endpoints/tenants
	SuggestedRemediations []RemediationSuggestion `json:"suggested_remediations"`
	ConfidenceScore       float64                 `json:"confidence_score"` // 0.0 to 1.0
}

// ScoredEvent is an event with causality scoring context
type ScoredEvent struct {
	Event            Event              `json:"event"`
	CausalityScore   float64            `json:"causality_score"` // Likelihood this caused others
	ImpactScore      float64            `json:"impact_score"`    // Severity of impact
	CorrelatedEvents []CorrelationScore `json:"correlated_events,omitempty"`
}

// RemediationSuggestion is an action suggested to prevent recurrence
type RemediationSuggestion struct {
	ActionType      string  `json:"action_type"`      // "restart_worker", "throttle_tenant", etc
	Priority        string  `json:"priority"`         // "high", "medium", "low"
	Confidence      float64 `json:"confidence"`       // How confident we are this will help
	Reason          string  `json:"reason"`           // Why we suggest this
	RecurrenceCount int     `json:"recurrence_count"` // How many similar incidents had this fix
}

// CorrelationEngine analyzes events to determine root cause
type CorrelationEngine struct {
	store          Store
	regionRegistry *RegionRegistry // Phase 3.2: Region-aware RCA scoring
}

// NewCorrelationEngine creates a new correlation engine with optional region awareness
func NewCorrelationEngine(store Store) *CorrelationEngine {
	return &CorrelationEngine{
		store:          store,
		regionRegistry: NewRegionRegistry(store), // Phase 3.2: Enable region-aware scoring
	}
}

// ComputeRCA performs intelligent root cause analysis on incident events
func (e *CorrelationEngine) ComputeRCA(events []Event) *RCAResult {
	if len(events) == 0 {
		return &RCAResult{
			ConfidenceScore: 0,
		}
	}

	// Score causality between all events
	correlations := e.scoreCorrelations(events)

	// Find root cause using causality scores
	rootCause := e.findRootCause(events, correlations)

	// Build causality chain from root to effects
	chain := e.buildCausalityChain(rootCause, events, correlations)

	// Extract affected services
	services := e.extractAffectedServices(events)

	// Generate remediation suggestions
	remediations := e.suggestRemediations(rootCause, events)

	// Calculate overall confidence
	confidence := e.calculateConfidence(events, rootCause, chain)

	return &RCAResult{
		SuspectedRootCause:    rootCause,
		CausalityChain:        chain,
		AffectedServices:      services,
		SuggestedRemediations: remediations,
		ConfidenceScore:       confidence,
	}
}

// scoreCorrelations computes causality scores between all event pairs
func (e *CorrelationEngine) scoreCorrelations(events []Event) map[string][]CorrelationScore {
	correlations := make(map[string][]CorrelationScore)

	for i := 0; i < len(events)-1; i++ {
		for j := i + 1; j < len(events); j++ {
			from := events[i]
			to := events[j]

			// Only compute correlation if 'to' happens after 'from'
			if to.OccurredAt.Before(from.OccurredAt) {
				from, to = to, from
			}

			score := e.computeCorrelationScore(from, to)
			if score.Score > 0.1 { // Only store significant correlations
				correlations[from.ID.String()] = append(correlations[from.ID.String()], score)
			}
		}
	}

	return correlations
}

// computeCorrelationScore computes likelihood that fromEvent caused toEvent
// Phase 3.8b: Now includes region affinity signals
func (e *CorrelationEngine) computeCorrelationScore(from, to Event) CorrelationScore {
	reasonScores := make(map[string]float64)

	// 1. Temporal proximity (closer events are more likely related)
	timeGap := to.OccurredAt.Sub(from.OccurredAt)
	timeProximityScore := e.scoreTemporalProximity(timeGap)
	reasonScores["temporal_proximity"] = timeProximityScore

	// 2. Event type relationships (some event types cause others)
	typeScore := e.scoreEventTypeRelationship(from.EventType, to.EventType)
	reasonScores["event_relationship"] = typeScore

	// 3. Scope matching (same tenant/endpoint = higher likelihood)
	scopeScore := e.scoreScopeMatching(from, to)
	reasonScores["scope_match"] = scopeScore

	// 4. Severity escalation (error follows warning = causality)
	severityScore := e.scoreSeverityEscalation(from.Severity, to.Severity)
	reasonScores["severity_escalation"] = severityScore

	// Phase 3.8b: 5. Region affinity (same region = higher correlation, cross-region = lower)
	regionAffinityScore := e.scoreRegionAffinity(from, to)
	reasonScores["region_affinity"] = regionAffinityScore

	// Weighted combination with Phase 3.8b region signals
	totalScore := 0.0
	totalScore += timeProximityScore * 0.35  // Temporal proximity still strong
	totalScore += typeScore * 0.30           // Event relationship
	totalScore += scopeScore * 0.12          // Scope match (reduced for region)
	totalScore += severityScore * 0.08       // Severity escalation
	totalScore += regionAffinityScore * 0.15 // Phase 3.8b: Region affinity (15% weight)

	// Clamp to 0-1 range
	totalScore = math.Min(1.0, math.Max(0.0, totalScore))

	// Determine primary reason based on highest individual score
	primaryReason := "temporal_proximity"
	maxScore := timeProximityScore
	if typeScore > maxScore {
		primaryReason = "event_relationship"
		maxScore = typeScore
	}
	if scopeScore > maxScore {
		primaryReason = "scope_match"
		maxScore = scopeScore
	}
	if severityScore > maxScore {
		primaryReason = "severity_escalation"
		maxScore = severityScore
	}
	if regionAffinityScore > maxScore {
		primaryReason = "region_affinity"
	}

	return CorrelationScore{
		FromEventID:   from.ID.String(),
		ToEventID:     to.ID.String(),
		Score:         totalScore,
		TimeGapMs:     timeGap.Milliseconds(),
		ReasonScores:  reasonScores,
		PrimaryReason: primaryReason,
	}
}

// scoreTemporalProximity scores how close two events are in time
// Events within 30 seconds: high score. Within 5 minutes: moderate. Beyond: low.
func (e *CorrelationEngine) scoreTemporalProximity(gap time.Duration) float64 {
	if gap < 30*time.Second {
		return 1.0
	}
	if gap < 2*time.Minute {
		return 0.8
	}
	if gap < 5*time.Minute {
		return 0.5
	}
	if gap < 15*time.Minute {
		return 0.2
	}
	return 0.0
}

// scoreEventTypeRelationship scores how likely one event type causes another
func (e *CorrelationEngine) scoreEventTypeRelationship(from, to EventType) float64 {
	// Define causal relationships
	relationships := map[EventType]map[EventType]float64{
		// Latency anomaly → errors → health degradation
		EventLatencyAnomaly: {
			EventAlert:          0.9,
			EventFingerprint:    0.7,
			EventEndpointHealth: 0.6,
		},
		// Errors → fingerprints, health issues
		EventFingerprint: {
			EventEndpointHealth: 0.8,
			EventAlert:          0.6,
			EventTenantHealth:   0.5,
		},
		// Tenant health → endpoint health
		EventTenantHealth: {
			EventEndpointHealth: 0.7,
			EventAlert:          0.4,
		},
		// Alerts often precede incidents
		EventAlert: {
			EventIncidentOpened: 0.9,
			EventEndpointHealth: 0.5,
		},
	}

	if rels, ok := relationships[from]; ok {
		if score, ok := rels[to]; ok {
			return score
		}
	}

	// Default: generic relationship based on severity progression
	return 0.1
}

// scoreScopeMatching scores if both events affect the same tenant/endpoint/region
func (e *CorrelationEngine) scoreScopeMatching(from, to Event) float64 {
	// Same tenant = strong signal
	if from.TenantID != nil && to.TenantID != nil && from.TenantID.String() == to.TenantID.String() {
		return 0.8
	}

	// Same endpoint = strong signal
	if from.EndpointPath != nil && to.EndpointPath != nil && *from.EndpointPath == *to.EndpointPath {
		return 0.8
	}

	// Phase 3.2: Region affinity scoring (replaces simple region check)
	if from.Region != nil && to.Region != nil {
		affinityScore := GetRegionAffinityScore(from.Region, to.Region)
		// Scale affinity score from [0.3-1.0] to [0.2-0.5] for scope matching
		// Same region (1.0) → 0.5, Adjacent (0.7) → 0.4, Cross (0.3) → 0.2
		return 0.2 + (affinityScore * 0.3)
	}

	// Different scope = weak signal (could be global issue)
	return 0.2
}

// scoreSeverityEscalation scores if severity increased (natural causal progression)
func (e *CorrelationEngine) scoreSeverityEscalation(from, to Severity) float64 {
	severityRank := map[Severity]int{
		SeverityInfo:     0,
		SeverityWarning:  1,
		SeverityError:    2,
		SeverityCritical: 3,
	}

	fromRank := severityRank[from]
	toRank := severityRank[to]

	// Escalation (from < to) is a good signal
	if toRank > fromRank {
		return 0.6
	}

	// Same severity = moderate signal
	if toRank == fromRank {
		return 0.3
	}

	// De-escalation = weak signal
	return 0.0
}

// Phase 3.8b: scoreRegionAffinity scores region-based correlation
// Same region: +0.2 boost (1.0)
// Cross-region: -0.3 penalty (0.4)
// Returns score in [0.3, 1.0] range
func (e *CorrelationEngine) scoreRegionAffinity(from, to Event) float64 {
	// Both events have no region = neutral
	if from.Region == nil && to.Region == nil {
		return 0.5
	}

	// One event missing region = slight penalty
	if (from.Region == nil) != (to.Region == nil) {
		return 0.4
	}

	// Both have regions - compute affinity
	if from.Region != nil && to.Region != nil {
		affinityScore := GetRegionAffinityScore(from.Region, to.Region)
		// Scale [0.3-1.0] affinity to correlation [0.4-1.0]
		// Same region (1.0) → 1.0 (strong causal link)
		// Adjacent (0.7) → 0.75 (good correlation)
		// Cross (0.3) → 0.4 (weak correlation)
		return 0.3 + (affinityScore * 0.7)
	}

	return 0.5
}

// findRootCause identifies the event most likely to be the root cause
func (e *CorrelationEngine) findRootCause(events []Event, correlations map[string][]CorrelationScore) *ScoredEvent {
	// Root cause is the event that:
	// 1. Has highest number of outgoing (caused) relationships
	// 2. Occurs earliest among critical/error events
	// 3. Is critical or error severity

	type eventScore struct {
		event         Event
		outgoingScore float64
		totalImpact   float64
	}

	scored := make([]eventScore, 0)

	for _, event := range events {
		// Score based on outgoing causality
		outgoing := correlations[event.ID.String()]
		var outgoingScore float64
		for _, corr := range outgoing {
			outgoingScore += corr.Score
		}

		// Normalize by count
		if len(outgoing) > 0 {
			outgoingScore /= float64(len(outgoing))
		}

		// Weight by severity
		severityWeight := map[Severity]float64{
			SeverityInfo:     0.1,
			SeverityWarning:  0.4,
			SeverityError:    0.8,
			SeverityCritical: 1.0,
		}[event.Severity]

		totalScore := outgoingScore * severityWeight

		scored = append(scored, eventScore{
			event:         event,
			outgoingScore: outgoingScore,
			totalImpact:   totalScore,
		})
	}

	// Find highest scoring event
	if len(scored) == 0 {
		return nil
	}

	best := scored[0]
	for _, s := range scored[1:] {
		if s.totalImpact > best.totalImpact {
			best = s
		}
	}

	return &ScoredEvent{
		Event:          best.event,
		CausalityScore: best.outgoingScore,
		ImpactScore:    best.totalImpact,
	}
}

// buildCausalityChain orders events from root cause to effects
func (e *CorrelationEngine) buildCausalityChain(root *ScoredEvent, events []Event, correlations map[string][]CorrelationScore) []ScoredEvent {
	if root == nil {
		return []ScoredEvent{}
	}

	chain := []ScoredEvent{*root}
	visited := make(map[string]bool)
	visited[root.Event.ID.String()] = true

	// BFS to build chain
	for len(chain) > 0 {
		current := chain[len(chain)-1]
		found := false

		// Find next event with highest correlation
		var nextEvent Event
		var maxScore float64

		for _, corr := range correlations[current.Event.ID.String()] {
			if visited[corr.ToEventID] {
				continue
			}

			if corr.Score > maxScore {
				maxScore = corr.Score
				// Find the actual event
				for _, e := range events {
					if e.ID.String() == corr.ToEventID {
						nextEvent = e
						break
					}
				}
			}
		}

		if maxScore > 0.1 {
			visited[nextEvent.ID.String()] = true
			chain = append(chain, ScoredEvent{
				Event:          nextEvent,
				CausalityScore: maxScore,
				ImpactScore:    maxScore,
			})
			found = true
		}

		if !found {
			break
		}
	}

	return chain
}

// extractAffectedServices extracts unique tenants and endpoints
func (e *CorrelationEngine) extractAffectedServices(events []Event) []string {
	services := make(map[string]bool)

	for _, event := range events {
		if event.TenantID != nil {
			services["tenant:"+event.TenantID.String()] = true
		}
		if event.EndpointPath != nil {
			services["endpoint:"+*event.EndpointPath] = true
		}
	}

	result := make([]string, 0, len(services))
	for s := range services {
		result = append(result, s)
	}
	return result
}

// suggestRemediations suggests actions to prevent recurrence
func (e *CorrelationEngine) suggestRemediations(root *ScoredEvent, events []Event) []RemediationSuggestion {
	if root == nil {
		return []RemediationSuggestion{}
	}

	suggestions := []RemediationSuggestion{}

	// Based on root cause event type, suggest remediations
	switch root.Event.EventType {
	case EventLatencyAnomaly:
		suggestions = append(suggestions, RemediationSuggestion{
			ActionType:      "restart_worker",
			Priority:        "high",
			Confidence:      0.8,
			Reason:          "Latency anomalies often caused by stuck workers or resource exhaustion",
			RecurrenceCount: 0,
		})

	case EventFingerprint:
		suggestions = append(suggestions, RemediationSuggestion{
			ActionType:      "throttle_tenant",
			Priority:        "high",
			Confidence:      0.7,
			Reason:          "High error fingerprints from tenant; throttle to prevent cascade",
			RecurrenceCount: 0,
		})

	case EventTenantHealth:
		suggestions = append(suggestions, RemediationSuggestion{
			ActionType:      "throttle_tenant",
			Priority:        "medium",
			Confidence:      0.6,
			Reason:          "Tenant health degradation; throttle to allow recovery",
			RecurrenceCount: 0,
		})
	}

	return suggestions
}

// calculateConfidence returns overall confidence in RCA (0.0 to 1.0)
func (e *CorrelationEngine) calculateConfidence(events []Event, root *ScoredEvent, chain []ScoredEvent) float64 {
	if root == nil || len(chain) == 0 {
		return 0.0
	}

	// Confidence factors:
	// 1. Root cause causality score (0-1)
	// 2. Length of causality chain (longer = more confident)
	// 3. Event severity (critical/error = more confident)

	score := 0.0

	// Factor 1: Root causality
	score += root.CausalityScore * 0.50

	// Factor 2: Chain length (normalize by event count)
	chainRatio := float64(len(chain)) / float64(len(events))
	if chainRatio > 0.5 {
		chainRatio = 0.5
	}
	score += chainRatio * 0.30

	// Factor 3: Root severity (critical = 1.0, error = 0.8, warning = 0.5)
	severityConfidence := map[Severity]float64{
		SeverityInfo:     0.0,
		SeverityWarning:  0.2,
		SeverityError:    0.6,
		SeverityCritical: 1.0,
	}[root.Event.Severity]
	score += severityConfidence * 0.20

	return math.Min(1.0, score)
}
