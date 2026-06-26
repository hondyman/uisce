package rules

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// CandidateValue represents a value from a specific source
type CandidateValue struct {
	SourceID         uuid.UUID
	SourceSystem     string
	SourcePriority   int
	SourceConfidence int
	Value            bool // IsBusinessDay
	HolidayName      *string
}

// SurvivingValue represents the result of the survivorship process
type SurvivingValue struct {
	SourceID        uuid.UUID
	SourceSystem    string
	WinningValue    bool
	ConfidenceScore int
	HolidayName     *string
	RuleApplied     string
	CompetingValues []CandidateValue
}

// ConflictAnalysis represents the results of the conflict detection
type ConflictAnalysis struct {
	HasConflict       bool
	ConflictType      string // SOURCE_DISAGREEMENT, LOW_CONFIDENCE, MISSING_DATA
	Message           string
	CompetingSources  []string
	RecommendedAction string
	Severity          string // CRITICAL, HIGH, MEDIUM, LOW
}

// RulesEngine implements the survivorship hierarchy
type RulesEngine struct {
	logger *logrus.Entry
}

// NewRulesEngine creates a new rules engine
func NewRulesEngine(logger *logrus.Entry) *RulesEngine {
	if logger == nil {
		logger = logrus.NewEntry(logrus.New())
	}
	return &RulesEngine{logger: logger}
}

// ExecuteSurvivorship determines the winning value based on source priority
func (e *RulesEngine) ExecuteSurvivorship(ctx context.Context, date string, region string, candidates []CandidateValue) (*SurvivingValue, error) {
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no candidate values for survivorship")
	}

	e.logger.WithFields(logrus.Fields{
		"date":            date,
		"region":          region,
		"candidate_count": len(candidates),
	}).Debug("Executing survivorship rules")

	// Sort candidates by priority (lower priority_score = higher trust)
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].SourcePriority != candidates[j].SourcePriority {
			return candidates[i].SourcePriority < candidates[j].SourcePriority
		}
		// Tiebreaker: higher confidence
		return candidates[i].SourceConfidence > candidates[j].SourceConfidence
	})

	winner := candidates[0]
	conflicts := e.detectConflicts(candidates, winner)
	confidence := e.calculateConfidence(candidates, winner, conflicts)

	result := &SurvivingValue{
		SourceID:        winner.SourceID,
		SourceSystem:    winner.SourceSystem,
		WinningValue:    winner.Value,
		ConfidenceScore: confidence,
		HolidayName:     winner.HolidayName,
		RuleApplied:     fmt.Sprintf("CalendarSurvivorship_Priority_%d", winner.SourcePriority),
		CompetingValues: candidates,
	}

	e.logger.WithFields(logrus.Fields{
		"date":           date,
		"winning_source": winner.SourceSystem,
		"confidence":     confidence,
		"conflicts":      len(conflicts),
	}).Debug("Survivorship execution completed")

	return result, nil
}

// detectConflicts identifies if high-priority sources disagree
func (e *RulesEngine) detectConflicts(candidates []CandidateValue, winner CandidateValue) []CandidateValue {
	var conflicts []CandidateValue
	for i := 1; i < len(candidates); i++ {
		candidate := candidates[i]
		// If candidate has high confidence (>85) and disagrees with winner, it's a conflict
		if candidate.SourceConfidence > 85 && candidate.Value != winner.Value {
			conflicts = append(conflicts, candidate)
		}
	}
	return conflicts
}

// calculateConfidence computes a 0-100 confidence score
func (e *RulesEngine) calculateConfidence(candidates []CandidateValue, winner CandidateValue, conflicts []CandidateValue) int {
	confidence := winner.SourceConfidence

	// Reduce confidence if we have conflicts
	if len(conflicts) > 0 {
		confidence = confidence - (len(conflicts) * 10)
	}

	// Bonus if all top candidates agree
	if len(candidates) > 1 && candidates[0].Value == candidates[1].Value {
		confidence = confidence + 5
	}

	// Ensure bounds
	if confidence > 100 {
		confidence = 100
	}
	if confidence < 0 {
		confidence = 0
	}

	return confidence
}

// AnalyzeConflict checks for potential issues in the result
func (e *RulesEngine) AnalyzeConflict(date string, region string, candidates []CandidateValue, result *SurvivingValue) *ConflictAnalysis {
	analysis := &ConflictAnalysis{
		HasConflict: false,
	}

	// Check 1: High-priority sources disagree
	if len(candidates) > 1 && candidates[0].Value != candidates[1].Value && candidates[1].SourceConfidence > 80 {
		analysis.HasConflict = true
		analysis.ConflictType = "SOURCE_DISAGREEMENT"
		analysis.Message = fmt.Sprintf("Top sources disagree: %s (%.0f%%) vs %s (%.0f%%)",
			candidates[0].SourceSystem, float64(candidates[0].SourceConfidence),
			candidates[1].SourceSystem, float64(candidates[1].SourceConfidence))
		analysis.CompetingSources = []string{candidates[0].SourceSystem, candidates[1].SourceSystem}
		analysis.RecommendedAction = "MANUAL_REVIEW"
		analysis.Severity = "CRITICAL"
	}

	// Check 2: Winning source has low confidence
	if result.ConfidenceScore < 70 {
		analysis.HasConflict = true
		analysis.ConflictType = "LOW_CONFIDENCE"
		analysis.Message = fmt.Sprintf("Winning source has low confidence: %.0f%%", float64(result.ConfidenceScore))
		analysis.RecommendedAction = "ESCALATE_TO_STEWARDSHIP"
		analysis.Severity = "HIGH"
	}

	return analysis
}
