package ops

import (
	"crypto/md5"
	"fmt"
	"math"
	"sort"
	"time"
)

// IncidentPattern represents a recurring incident pattern
type IncidentPattern struct {
	ID               string    `json:"id"`                // MD5 hash of event sequence
	EventSignature   []string  `json:"event_signature"`   // Ordered list of event types
	Severity         Severity  `json:"severity"`          // Most common severity
	TimelineMinutes  int       `json:"timeline_minutes"`  // How long the pattern lasts
	AffectedServices []string  `json:"affected_services"` // Services impacted
	RecurrenceCount  int       `json:"recurrence_count"`  // Times this pattern occurred
	FirstSeen        time.Time `json:"first_seen"`
	LastSeen         time.Time `json:"last_seen"`
	SuccessfulFixes  []string  `json:"successful_fixes"` // Actions that resolved it
	AverageDuration  int       `json:"average_duration"` // Minutes to resolve
	Confidence       float64   `json:"confidence"`       // 0-1 pattern confidence
}

// IncidentSimilarity measures how similar two incidents are
type IncidentSimilarity struct {
	IncidentID1     string  `json:"incident_1_id"`
	IncidentID2     string  `json:"incident_2_id"`
	SimilarityScore float64 `json:"similarity_score"` // 0.0 to 1.0
	MatchedEvents   int     `json:"matched_events"`
	PatternID       string  `json:"pattern_id,omitempty"` // If matches known pattern
}

// PatternMatcher identifies recurring incident patterns
type PatternMatcher struct {
	store Store
}

// NewPatternMatcher creates a new pattern matcher
func NewPatternMatcher(store Store) *PatternMatcher {
	return &PatternMatcher{store: store}
}

// CreateIncidentPattern analyzes events to create a pattern fingerprint
func (m *PatternMatcher) CreateIncidentPattern(events []Event) *IncidentPattern {
	if len(events) == 0 {
		return nil
	}

	// Sort events by time
	sortedEvents := make([]Event, len(events))
	copy(sortedEvents, events)
	sort.Slice(sortedEvents, func(i, j int) bool {
		return sortedEvents[i].OccurredAt.Before(sortedEvents[j].OccurredAt)
	})

	// Build event signature (sequence of event types)
	signature := m.buildEventSignature(sortedEvents)

	// Extract pattern metadata
	severity := m.extractMaxSeverity(sortedEvents)
	services := m.extractServices(sortedEvents)
	duration := m.calculateDuration(sortedEvents)

	// Generate pattern ID as MD5 of signature
	patternID := m.hashSignature(signature)

	return &IncidentPattern{
		ID:               patternID,
		EventSignature:   signature,
		Severity:         severity,
		TimelineMinutes:  duration,
		AffectedServices: services,
		RecurrenceCount:  1,
		FirstSeen:        time.Now(),
		LastSeen:         time.Now(),
		SuccessfulFixes:  []string{},
		Confidence:       0.5, // Start with moderate confidence
	}
}

// buildEventSignature creates ordered event type sequence (compressed for matching)
func (m *PatternMatcher) buildEventSignature(events []Event) []string {
	var signature []string
	var lastType EventType

	for _, event := range events {
		// Skip duplicate consecutive event types
		if event.EventType == lastType && len(signature) > 0 {
			continue
		}
		signature = append(signature, string(event.EventType))
		lastType = event.EventType
	}

	return signature
}

// extractMaxSeverity finds highest severity in event set
func (m *PatternMatcher) extractMaxSeverity(events []Event) Severity {
	max := SeverityInfo
	for _, event := range events {
		if event.Severity == SeverityCritical {
			return SeverityCritical
		}
		if event.Severity == SeverityError && max != SeverityCritical {
			max = SeverityError
		}
		if event.Severity == SeverityWarning && max == SeverityInfo {
			max = SeverityWarning
		}
	}
	return max
}

// extractServices extracts unique tenant/endpoint combinations
func (m *PatternMatcher) extractServices(events []Event) []string {
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
	sort.Strings(result)
	return result
}

// calculateDuration measures time from first to last event
func (m *PatternMatcher) calculateDuration(events []Event) int {
	if len(events) < 2 {
		return 0
	}

	first := events[0].OccurredAt
	last := events[len(events)-1].OccurredAt

	return int(last.Sub(first).Minutes())
}

// hashSignature creates MD5 hash of signature for pattern ID
func (m *PatternMatcher) hashSignature(signature []string) string {
	signatureStr := ""
	for _, s := range signature {
		signatureStr += s + "|"
	}

	hash := md5.Sum([]byte(signatureStr))
	return fmt.Sprintf("%x", hash)
}

// FindSimilarIncidents compares current incident against historical incidents
func (m *PatternMatcher) FindSimilarIncidents(currentEvents []Event, historicalIncidents [][]Event) []IncidentSimilarity {
	if len(currentEvents) == 0 {
		return []IncidentSimilarity{}
	}

	currentPattern := m.CreateIncidentPattern(currentEvents)
	currentSignature := currentPattern.EventSignature

	var similarities []IncidentSimilarity

	for _, historicalEvents := range historicalIncidents {
		historicalPattern := m.CreateIncidentPattern(historicalEvents)
		historicalSignature := historicalPattern.EventSignature

		// Calculate similarity score
		score := m.calculateSignatureSimilarity(currentSignature, historicalSignature)

		if score > 0.5 { // Only include >50% matches
			similarities = append(similarities, IncidentSimilarity{
				IncidentID1:     "current",
				IncidentID2:     "", // Would be populated with actual incident ID
				SimilarityScore: score,
				MatchedEvents:   m.countMatchedEvents(currentSignature, historicalSignature),
				PatternID:       historicalPattern.ID,
			})
		}
	}

	// Sort by similarity score descending
	sort.Slice(similarities, func(i, j int) bool {
		return similarities[i].SimilarityScore > similarities[j].SimilarityScore
	})

	return similarities
}

// calculateSignatureSimilarity scores how similar two event signatures are
// Uses longest common subsequence (LCS) approach for partial matching
func (m *PatternMatcher) calculateSignatureSimilarity(sig1, sig2 []string) float64 {
	if len(sig1) == 0 && len(sig2) == 0 {
		return 1.0
	}
	if len(sig1) == 0 || len(sig2) == 0 {
		return 0.0
	}

	// Calculate longest common subsequence
	lcs := m.longestCommonSubsequence(sig1, sig2)

	// Similarity is LCS length weighted by signature lengths
	maxLen := float64(math.Max(float64(len(sig1)), float64(len(sig2))))
	lcsLen := float64(len(lcs))

	// Weighting: prefer exact matches but allow partial
	exactMatch := 1.0
	if len(sig1) != len(sig2) {
		exactMatch = 0.8 // Penalize length mismatch
	}

	return (lcsLen / maxLen) * exactMatch
}

// longestCommonSubsequence finds LCS of two string slices
// Using dynamic programming approach
func (m *PatternMatcher) longestCommonSubsequence(s1, s2 []string) []string {
	// Build DP table
	dp := make([][]int, len(s1)+1)
	for i := range dp {
		dp[i] = make([]int, len(s2)+1)
	}

	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			if s1[i-1] == s2[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				if dp[i-1][j] > dp[i][j-1] {
					dp[i][j] = dp[i-1][j]
				} else {
					dp[i][j] = dp[i][j-1]
				}
			}
		}
	}

	// Reconstruct LCS
	var lcs []string
	i, j := len(s1), len(s2)

	for i > 0 && j > 0 {
		if s1[i-1] == s2[j-1] {
			lcs = append([]string{s1[i-1]}, lcs...)
			i--
			j--
		} else if dp[i-1][j] > dp[i][j-1] {
			i--
		} else {
			j--
		}
	}

	return lcs
}

// countMatchedEvents returns number of events that match between signatures
func (m *PatternMatcher) countMatchedEvents(sig1, sig2 []string) int {
	lcs := m.longestCommonSubsequence(sig1, sig2)
	return len(lcs)
}

// RecalculateConfidence updates pattern confidence based on recurrence
func (p *IncidentPattern) RecalculateConfidence() {
	// Confidence increases with:
	// 1. Number of recurrences (more = higher confidence)
	// 2. Time between recurrences (consistent = higher)
	// 3. Successful fixes (remediation = higher confidence in pattern)

	baseConfidence := 0.5

	// Factor 1: Recurrence count
	recurrenceFactor := math.Min(1.0, float64(p.RecurrenceCount)/5.0) // Max at 5 recurrences
	baseConfidence += recurrenceFactor * 0.3

	// Factor 2: Number of successful fixes
	fixFactor := float64(len(p.SuccessfulFixes)) / 3.0 // Max 3 fixes
	baseConfidence += fixFactor * 0.2

	p.Confidence = math.Min(1.0, baseConfidence)
}

// ClusterIncidents groups incidents by pattern similarity
// Returns map of pattern ID -> list of similar incidents
func (m *PatternMatcher) ClusterIncidents(incidents []struct {
	ID     string
	Events []Event
}) map[string][]string {
	clusters := make(map[string][]string)

	for _, incident := range incidents {
		pattern := m.CreateIncidentPattern(incident.Events)
		if pattern != nil {
			clusters[pattern.ID] = append(clusters[pattern.ID], incident.ID)
		}
	}

	return clusters
}
