package explorer

import (
	"context"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type DriverAttributionItem struct {
	DimensionValue string  `json:"dimensionValue"`
	BaselineValue  float64 `json:"baselineValue"`
	CompareValue   float64 `json:"compareValue"`
	ImpactDelta    float64 `json:"impactDelta"`
	ImpactPct      float64 `json:"impactPct"`
	ZScore         float64 `json:"zScore"`
	IsAnomaly      bool    `json:"isAnomaly"`
}

type SpotIQDecompositionResponse struct {
	MetricKey        string                  `json:"metricKey"`
	DimensionKey     string                  `json:"dimensionKey"`
	TotalVariancePct float64                 `json:"totalVariancePct"`
	TopContributors  []DriverAttributionItem `json:"topContributors"`
	Narrative        string                  `json:"narrative"`
}

type SpotIQEngine struct {
	db *sqlx.DB
}

func NewSpotIQEngine(db *sqlx.DB) *SpotIQEngine {
	return &SpotIQEngine{db: db}
}

// DecomposeMetricVariance calculates positive and negative dimension drivers behind a metric movement
func (e *SpotIQEngine) DecomposeMetricVariance(
	ctx context.Context,
	tenantID, boID uuid.UUID,
	metricKey, dimensionKey string,
	baselineRows, compareRows map[string]float64,
) (*SpotIQDecompositionResponse, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	var baseTotal, compTotal float64
	for _, v := range baselineRows {
		baseTotal += v
	}
	for _, v := range compareRows {
		compTotal += v
	}

	totalDelta := compTotal - baseTotal
	var totalVariancePct float64
	if baseTotal != 0 {
		totalVariancePct = (totalDelta / baseTotal) * 100.0
	}

	contributors := make([]DriverAttributionItem, 0)
	var sumSquares float64
	count := 0

	for dimVal, cVal := range compareRows {
		bVal := baselineRows[dimVal]
		delta := cVal - bVal
		var impactPct float64
		if baseTotal != 0 {
			impactPct = (delta / baseTotal) * 100.0
		}

		sumSquares += delta * delta
		count++

		contributors = append(contributors, DriverAttributionItem{
			DimensionValue: dimVal,
			BaselineValue:  bVal,
			CompareValue:   cVal,
			ImpactDelta:    delta,
			ImpactPct:      impactPct,
		})
	}

	// Calculate Standard Deviation & Z-Scores for Outlier Radar
	stdDev := 0.0
	if count > 1 {
		stdDev = math.Sqrt(sumSquares / float64(count))
	}

	topDriver := ""
	maxImpact := -1.0

	for i := range contributors {
		if stdDev > 0 {
			contributors[i].ZScore = math.Abs(contributors[i].ImpactDelta) / stdDev
			if contributors[i].ZScore >= 2.5 {
				contributors[i].IsAnomaly = true
			}
		}
		if math.Abs(contributors[i].ImpactPct) > maxImpact {
			maxImpact = math.Abs(contributors[i].ImpactPct)
			topDriver = contributors[i].DimensionValue
		}
	}

	narrative := fmt.Sprintf(
		"Metric [%s] shifted by %.2f%%. The primary driver was [%s] contributing %.2f%% of total variance.",
		metricKey, totalVariancePct, topDriver, maxImpact,
	)

	return &SpotIQDecompositionResponse{
		MetricKey:        metricKey,
		DimensionKey:     dimensionKey,
		TotalVariancePct: totalVariancePct,
		TopContributors:  contributors,
		Narrative:        narrative,
	}, nil
}
