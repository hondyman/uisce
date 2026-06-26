package quality

import (
	"context"

	"github.com/google/uuid"
)

type QualityDimension string

const (
	DimensionClarity      QualityDimension = "clarity"
	DimensionConsistency  QualityDimension = "consistency"
	DimensionCompleteness QualityDimension = "completeness"
	DimensionGovernance   QualityDimension = "governance"
	DimensionUsageHealth  QualityDimension = "usage_health"
)

type DimensionScore struct {
	Dimension QualityDimension `json:"dimension"`
	Score     int              `json:"score"` // 0-100
	Details   string           `json:"details"`
}

type QualityReport struct {
	BOID               uuid.UUID        `json:"bo_id"`
	BOName             string           `json:"bo_name"`
	OverallScore       int              `json:"overall_score"` // 0-100
	DimensionScores    []DimensionScore `json:"dimension_scores"`
	TopRecommendations []string         `json:"top_recommendations"`
}

type QualityScorer struct{}

func NewQualityScorer() *QualityScorer {
	return &QualityScorer{}
}

func (s *QualityScorer) ScoreBO(ctx context.Context, boID uuid.UUID) (*QualityReport, error) {
	// Mock: Generate quality report
	// Real: Analyze BO definition, relationships, usage, governance

	report := &QualityReport{
		BOID:   boID,
		BOName: "Position",
		DimensionScores: []DimensionScore{
			{
				Dimension: DimensionClarity,
				Score:     90,
				Details:   "All fields have descriptions; naming is clear and consistent.",
			},
			{
				Dimension: DimensionConsistency,
				Score:     70,
				Details:   "Some field names don't follow naming conventions (e.g., 'mv' vs 'market_value').",
			},
			{
				Dimension: DimensionCompleteness,
				Score:     85,
				Details:   "Most required fields defined; missing some optional metadata.",
			},
			{
				Dimension: DimensionGovernance,
				Score:     60,
				Details:   "PII classification incomplete; missing data residency policies.",
			},
			{
				Dimension: DimensionUsageHealth,
				Score:     75,
				Details:   "Moderate usage across 8 pages; not over-used.",
			},
		},
		TopRecommendations: []string{
			"Add PII classification for sensitive fields",
			"Standardize field naming (rename 'mv' to 'market_value')",
			"Define data residency policies for cross-border usage",
		},
	}

	// Calculate overall score as weighted average
	totalScore := 0
	for _, dim := range report.DimensionScores {
		totalScore += dim.Score
	}
	report.OverallScore = totalScore / len(report.DimensionScores)

	return report, nil
}
