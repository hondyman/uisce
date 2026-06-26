package risk

import (
	"context"

	"github.com/google/uuid"
)

type RiskDimension string

const (
	DimensionOperational RiskDimension = "operational"
	DimensionPerformance RiskDimension = "performance"
	DimensionCompliance  RiskDimension = "compliance"
	DimensionSemantic    RiskDimension = "semantic"
)

type DimensionScore struct {
	Dimension RiskDimension `json:"dimension"`
	Score     int           `json:"score"`      // 0-100 (higher is better)
	RiskLevel string        `json:"risk_level"` // low, medium, high, critical
	Details   string        `json:"details"`
}

type RiskScore struct {
	ChangeSetID                 uuid.UUID        `json:"changeset_id"`
	OverallScore                int              `json:"overall_score"` // 0-100
	OverallRiskLevel            string           `json:"overall_risk_level"`
	DimensionScores             []DimensionScore `json:"dimension_scores"`
	Mitigations                 []string         `json:"mitigations"`
	RequiresAdditionalReviewers bool             `json:"requires_additional_reviewers"`
}

type RiskScorer struct{}

func NewRiskScorer() *RiskScorer {
	return &RiskScorer{}
}

func (s *RiskScorer) Score(ctx context.Context, changesetID uuid.UUID) (*RiskScore, error) {
	// Mock: Generate risk score
	// Real: Analyze changeset across all dimensions

	dimensions := []DimensionScore{
		{
			Dimension: DimensionCompliance,
			Score:     20,
			RiskLevel: "high",
			Details:   "PII exposure risk detected in new field 'customer_ssn'",
		},
		{
			Dimension: DimensionPerformance,
			Score:     85,
			RiskLevel: "low",
			Details:   "Minor SLO impact (+50ms p95 render time)",
		},
		{
			Dimension: DimensionOperational,
			Score:     60,
			RiskLevel: "medium",
			Details:   "BO volatility: Position BO has 3 changes in last 7 days",
		},
		{
			Dimension: DimensionSemantic,
			Score:     70,
			RiskLevel: "medium",
			Details:   "Semantic quality score: 70/100. Missing field descriptions.",
		},
	}

	// Calculate overall score as weighted average
	totalScore := 0
	for _, dim := range dimensions {
		totalScore += dim.Score
	}
	overallScore := totalScore / len(dimensions)

	riskLevel := "low"
	if overallScore < 40 {
		riskLevel = "critical"
	} else if overallScore < 60 {
		riskLevel = "high"
	} else if overallScore < 80 {
		riskLevel = "medium"
	}

	score := &RiskScore{
		ChangeSetID:      changesetID,
		OverallScore:     overallScore,
		OverallRiskLevel: riskLevel,
		DimensionScores:  dimensions,
		Mitigations: []string{
			"Add masking rule for 'customer_ssn' field",
			"Run semantic regression tests",
			"Add field descriptions to improve semantic quality",
			"Monitor SLO metrics post-deployment",
		},
		RequiresAdditionalReviewers: overallScore < 60,
	}

	return score, nil
}
