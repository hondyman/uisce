package discovery

import (
	"fmt"
	"log"
	"math"
	"sort"

	"github.com/hondyman/semlayer/backend/internal/models"
)

// CandidateRanker scores and ranks discovered feature candidates
type CandidateRanker struct {
	logger *log.Logger
}

// ScoringWeights defines how to weight different scoring dimensions
type ScoringWeights struct {
	Completeness  float64 // How many non-null values (0-1)
	Cardinality   float64 // Diversity of values (0-1)
	Uniqueness    float64 // How many distinct values (0-1)
	Relevance     float64 // Business relevance (0-1)
	Correlation   float64 // Correlation with other features (0-1)
	TimelinessFit float64 // Fits temporal patterns well (0-1)
}

// DefaultWeights returns reasonable default weights
func DefaultWeights() ScoringWeights {
	return ScoringWeights{
		Completeness:  0.20, // 20%
		Cardinality:   0.15, // 15%
		Uniqueness:    0.15, // 15%
		Relevance:     0.25, // 25% (most important)
		Correlation:   0.15, // 15%
		TimelinessFit: 0.10, // 10%
	}
}

// toMap converts ScoringWeights to a map for API usage
func (sw ScoringWeights) toMap() map[string]float64 {
	return map[string]float64{
		"completeness":   sw.Completeness,
		"cardinality":    sw.Cardinality,
		"uniqueness":     sw.Uniqueness,
		"relevance":      sw.Relevance,
		"correlation":    sw.Correlation,
		"timeliness_fit": sw.TimelinessFit,
	}
}

// NewCandidateRanker creates a new candidate ranker
func NewCandidateRanker(logger *log.Logger) *CandidateRanker {
	return &CandidateRanker{
		logger: logger,
	}
}

// RankCandidates scores all candidates and returns them ranked
func (cr *CandidateRanker) RankCandidates(candidates []models.FeatureCandidate, weights ScoringWeights) []models.FeatureCandidate {
	// Score each candidate
	for i := range candidates {
		candidates[i].BusinessValue = cr.scoreCandidate(&candidates[i], weights)
	}

	// Sort by score (highest first)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].BusinessValue > candidates[j].BusinessValue
	})

	return candidates
}

// scoreCandidate calculates a composite score for a single candidate
func (cr *CandidateRanker) scoreCandidate(candidate *models.FeatureCandidate, weights ScoringWeights) float64 {
	var score float64

	// Completeness score (0-1 normalized from completeness %)
	completenessScore := candidate.Completeness * weights.Completeness
	score += completenessScore

	// Cardinality score (sweet spot: 10-1000 distinct values)
	cardinalityScore := cr.scoreCardinality(candidate.Cardinality) * weights.Cardinality
	score += cardinalityScore

	// Uniqueness (infer from cardinality / total rows, estimate 1M rows)
	uniquenessScore := math.Min(1.0, float64(candidate.Cardinality)/1000000.0) * weights.Uniqueness
	score += uniquenessScore

	// Relevance score (infer from source and name)
	relevanceScore := cr.scoreRelevance(candidate) * weights.Relevance
	score += relevanceScore

	// Correlation score (from technical score)
	correlationScore := candidate.TechnicalScore * weights.Correlation
	score += correlationScore

	// Timeliness fit (metrics are good for time-series, events are good too)
	timelinessScore := cr.scoreTimelinessFit(candidate) * weights.TimelinessFit
	score += timelinessScore

	return math.Min(1.0, score)
}

// scoreCardinality returns a score for cardinality (sweet spot 10-1000)
func (cr *CandidateRanker) scoreCardinality(cardinality int64) float64 {
	if cardinality < 0 {
		return 0.5 // Unknown cardinality, neutral score
	}
	if cardinality < 2 {
		return 0.0 // Binary field, not useful as feature
	}
	if cardinality < 10 {
		return 0.3 // Too low cardinality
	}
	if cardinality <= 100 {
		return 0.9 // Sweet spot
	}
	if cardinality <= 1000 {
		return 0.85
	}
	if cardinality <= 10000 {
		return 0.7 // Getting high
	}
	if cardinality <= 100000 {
		return 0.5 // Very high
	}
	// Over 100k is usually too high
	return 0.2
}

// scoreRelevance scores based on business relevance (inferred from naming)
func (cr *CandidateRanker) scoreRelevance(candidate *models.FeatureCandidate) float64 {
	score := 0.5 // Base score

	// Factor 1: Source importance
	if candidate.SourceDatabase == "incidents" || candidate.SourceDatabase == "events" {
		score += 0.3 // High relevance - directly from incidents
	} else if candidate.SourceDatabase == "logs" {
		score += 0.2 // Medium relevance
	} else if candidate.SourceDatabase == "prometheus" {
		score += 0.15 // Medium-low relevance
	}

	// Factor 2: Field name patterns (business terms)
	businessTerms := []string{"error", "latency", "throughput", "cpu", "memory", "failure", "incident", "alert", "status", "count", "duration", "time", "response"}
	for _, term := range businessTerms {
		if contains(candidate.Name, term) {
			score += 0.15
			break
		}
	}

	// Factor 3: Avoid technical noise
	noiseTerms := []string{"version", "hash", "uuid", "random", "debug", "internal", "_", "meta", "system"}
	for _, term := range noiseTerms {
		if contains(candidate.Name, term) {
			score -= 0.2
		}
	}

	return math.Max(0.0, math.Min(1.0, score))
}

// scoreTimelinessFit scores how well the feature fits temporal analysis
func (cr *CandidateRanker) scoreTimelinessFit(candidate *models.FeatureCandidate) float64 {
	score := 0.5 // Base score

	// Metrics are great for time-series
	if candidate.SourceDatabase == "prometheus" {
		score += 0.4
	}

	// Logs with timestamps are good
	if candidate.SourceDatabase == "logs" && contains(candidate.Name, "time") {
		score += 0.3
	}

	// Numeric types are better for forecasting
	if candidate.DataType == "float" || candidate.DataType == "number" {
		score += 0.2
	}

	// Categorical can work with encoding
	if candidate.DataType == "string" || candidate.DataType == "categorical" {
		score += 0.05
	}

	return math.Min(1.0, score)
}

// FilterByQualityThreshold returns only candidates above a certain quality threshold
func (cr *CandidateRanker) FilterByQualityThreshold(candidates []models.FeatureCandidate, threshold float64) []models.FeatureCandidate {
	var filtered []models.FeatureCandidate
	for _, c := range candidates {
		if c.BusinessValue >= threshold {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// DiversifySelection ensures selected features are from diverse sources
func (cr *CandidateRanker) DiversifySelection(candidates []models.FeatureCandidate, topN int) []models.FeatureCandidate {
	if len(candidates) <= topN {
		return candidates
	}

	// Group by source
	sourceGroups := make(map[string][]models.FeatureCandidate)
	for _, c := range candidates {
		sourceGroups[c.SourceDatabase] = append(sourceGroups[c.SourceDatabase], c)
	}

	// Distribute selections across sources
	selected := make([]models.FeatureCandidate, 0, topN)
	perSource := topN / len(sourceGroups)
	if perSource < 1 {
		perSource = 1
	}

	for _, group := range sourceGroups {
		for i := 0; i < perSource && i < len(group); i++ {
			selected = append(selected, group[i])
		}
		if len(selected) >= topN {
			break
		}
	}

	return selected[:topN]
}

// ExplainScore provides human-readable explanation of why feature was ranked
func (cr *CandidateRanker) ExplainScore(candidate *models.FeatureCandidate, weights ScoringWeights) string {
	explanation := fmt.Sprintf("Feature '%s' scored %.2f:\n", candidate.Name, candidate.BusinessValue)

	// Breakdown each component
	completenessScore := candidate.Completeness * weights.Completeness
	explanation += fmt.Sprintf("  - Completeness (%.0f%% complete): %.3f\n", candidate.Completeness*100, completenessScore)

	cardinalityScore := calculateCardinalityScore(candidate.Cardinality) * weights.Cardinality
	explanation += fmt.Sprintf("  - Cardinality (%d distinct): %.3f\n", candidate.Cardinality, cardinalityScore)

	relevanceScore := calculateRelevance(candidate) * weights.Relevance
	explanation += fmt.Sprintf("  - Business Relevance (%s from %s): %.3f\n", candidate.Name, candidate.SourceDatabase, relevanceScore)

	correlationScore := candidate.TechnicalScore * weights.Correlation
	explanation += fmt.Sprintf("  - Technical Quality: %.3f\n", correlationScore)

	return explanation
}

// Helper functions
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0
}

// calculateCardinalityScore is a helper that delegates to CandidateRanker
func calculateCardinalityScore(card int64) float64 {
	cr := &CandidateRanker{}
	return cr.scoreCardinality(card)
}

func calculateRelevance(c *models.FeatureCandidate) float64 {
	cr := &CandidateRanker{logger: nil}
	return cr.scoreRelevance(c)
}

// GetFeatureSuggestions provides top N recommendations based on current data
func (cr *CandidateRanker) GetFeatureSuggestions(candidates []models.FeatureCandidate, topN int) []models.FeatureCandidate {
	if len(candidates) <= topN {
		return candidates
	}

	// Sort by business value
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].BusinessValue > candidates[j].BusinessValue
	})

	// Apply diversity filter
	diverse := cr.DiversifySelection(candidates, topN)

	// Re-sort by rank
	sort.Slice(diverse, func(i, j int) bool {
		return diverse[i].BusinessValue > diverse[j].BusinessValue
	})

	return diverse
}

// CalculateCandidateStats provides aggregate statistics
func (cr *CandidateRanker) CalculateCandidateStats(candidates []models.FeatureCandidate) map[string]interface{} {
	if len(candidates) == 0 {
		return map[string]interface{}{}
	}

	stats := make(map[string]interface{})

	// Score distribution
	var scores []float64
	minScore := 1.0
	maxScore := 0.0
	var sumScore float64

	sourceCount := make(map[string]int)
	typeCount := make(map[string]int)

	for _, c := range candidates {
		scores = append(scores, c.BusinessValue)
		sumScore += c.BusinessValue
		minScore = math.Min(minScore, c.BusinessValue)
		maxScore = math.Max(maxScore, c.BusinessValue)
		sourceCount[c.SourceDatabase]++
		typeCount[c.DataType]++
	}

	stats["total_candidates"] = len(candidates)
	stats["avg_score"] = sumScore / float64(len(candidates))
	stats["min_score"] = minScore
	stats["max_score"] = maxScore
	stats["source_distribution"] = sourceCount
	stats["type_distribution"] = typeCount

	// Calculate median
	sort.Float64s(scores)
	median := 0.0
	if len(scores)%2 == 0 {
		median = (scores[len(scores)/2-1] + scores[len(scores)/2]) / 2
	} else {
		median = scores[len(scores)/2]
	}
	stats["median_score"] = median

	return stats
}
