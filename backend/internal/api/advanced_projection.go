package api

import (
	"fmt"
	"math"
	"time"
)

type AdvancedProjectionService struct{}

func NewAdvancedProjectionService() *AdvancedProjectionService {
	return &AdvancedProjectionService{}
}

// ProjectedRow includes full attribution metrics for enterprise transparency
type ProjectedRow struct {
	DimensionKey       string  `json:"dimensionKey"`
	TimeVal            string  `json:"timeVal"`
	MeasureVal         float64 `json:"measureVal"`
	IsForecast         bool    `json:"isForecast"`
	// --- Attribution & Risk Metrics ---
	TrendComponent     float64 `json:"trendComponent"`     // Baseline linear regression value
	SeasonalMultiplier float64 `json:"seasonalMultiplier"` // Seasonal adjustment factor
	UpperBound         float64 `json:"upperBound"`         // 95% Confidence Upper Limit
	LowerBound         float64 `json:"lowerBound"`         // 95% Confidence Lower Limit
}

// ForecastByDimension computes trend, seasonality, confidence intervals, and attribution
func (s *AdvancedProjectionService) ForecastByDimension(
	rows []map[string]interface{},
	dimCol string,
	timeCol string,
	measureCol string,
	periodsAhead int,
) ([]ProjectedRow, error) {
	if len(rows) == 0 {
		return nil, fmt.Errorf("no data provided for forecasting")
	}

	groups := make(map[string][]map[string]interface{})
	for _, row := range rows {
		dimVal, ok := row[dimCol].(string)
		if !ok || dimVal == "" {
			dimVal = "Total"
		}
		groups[dimVal] = append(groups[dimVal], row)
	}

	var allResults []ProjectedRow

	for dimKey, groupRows := range groups {
		n := float64(len(groupRows))
		if n < 3 {
			for _, r := range groupRows {
				tStr, _ := r[timeCol].(string)
				val, _ := convertToFloat(r[measureCol])
				allResults = append(allResults, ProjectedRow{
					DimensionKey: dimKey, TimeVal: tStr, MeasureVal: val, IsForecast: false,
					TrendComponent: val, SeasonalMultiplier: 1.0, UpperBound: val, LowerBound: val,
				})
			}
			continue
		}

		// 1. Linear Trend & Residual Standard Error Calculation
		var sumX, sumY, sumXY, sumX2 float64
		yVals := make([]float64, int(n))

		for i, r := range groupRows {
			x := float64(i)
			val, err := convertToFloat(r[measureCol])
			if err != nil {
				val = 0.0
			}
			yVals[i] = val
			sumX += x
			sumY += val
			sumXY += x * val
			sumX2 += x * x
		}

		denominator := n*sumX2 - sumX*sumX
		var slope, intercept float64
		if denominator == 0 {
			slope = 0
			intercept = sumY / n
		} else {
			slope = (n*sumXY - sumX*sumY) / denominator
			intercept = (sumY - slope*sumX) / n
		}

		// Calculate Residual Sum of Squares (RSS) for confidence bounds
		var rss float64
		for i := 0; i < int(n); i++ {
			predicted := (slope * float64(i)) + intercept
			residual := yVals[i] - predicted
			rss += residual * residual
		}
		standardError := math.Sqrt(rss / math.Max(1, n-2))
		confidenceMargin := 1.96 * standardError

		// 2. Seasonality Indices (12-period cycle)
		cycleLength := 12
		seasonalFactors := make([]float64, cycleLength)
		seasonalCounts := make([]int, cycleLength)

		for i := 0; i < int(n); i++ {
			expectedTrend := (slope * float64(i)) + intercept
			cycleIndex := i % cycleLength
			if expectedTrend > 0 {
				seasonalFactors[cycleIndex] += yVals[i] / expectedTrend
				seasonalCounts[cycleIndex]++
			}
		}

		for i := 0; i < cycleLength; i++ {
			if seasonalCounts[i] > 0 {
				seasonalFactors[i] /= float64(seasonalCounts[i])
			} else {
				seasonalFactors[i] = 1.0
			}
		}

		// 3. Push Historical Actuals with attribution
		for i, r := range groupRows {
			tStr, _ := r[timeCol].(string)
			trend := (slope * float64(i)) + intercept
			cycleIndex := i % cycleLength
			mult := seasonalFactors[cycleIndex]

			allResults = append(allResults, ProjectedRow{
				DimensionKey:       dimKey,
				TimeVal:            tStr,
				MeasureVal:         yVals[i],
				IsForecast:         false,
				TrendComponent:     math.Round(trend*100) / 100,
				SeasonalMultiplier: math.Round(mult*1000) / 1000,
				UpperBound:         yVals[i],
				LowerBound:         yVals[i],
			})
		}

		// 4. Project Future Periods with Attribution & Widening Confidence Bounds
		lastTimeStr, _ := groupRows[len(groupRows)-1][timeCol].(string)
		lastTime, err := time.Parse(time.RFC3339, lastTimeStr)
		if err != nil {
			lastTime, err = time.Parse("2006-01-02", lastTimeStr)
			if err != nil {
				lastTime = time.Now()
			}
		}

		for step := 1; step <= periodsAhead; step++ {
			futureIdx := int(n) + step - 1
			futureX := float64(futureIdx)

			trendVal := (slope * futureX) + intercept
			cycleIndex := futureIdx % cycleLength
			mult := seasonalFactors[cycleIndex]
			finalVal := math.Max(0, trendVal*mult)

			// Uncertainty grows slightly further out into the future
			widenedMargin := confidenceMargin * math.Sqrt(1.0+(float64(step)/n))

			nextTime := lastTime.AddDate(0, step, 0)

			allResults = append(allResults, ProjectedRow{
				DimensionKey:       dimKey,
				TimeVal:            nextTime.Format("2006-01-02T15:04:05Z07:00"),
				MeasureVal:         math.Round(finalVal*100) / 100,
				IsForecast:         true,
				TrendComponent:     math.Round(trendVal*100) / 100,
				SeasonalMultiplier: math.Round(mult*1000) / 1000,
				UpperBound:         math.Round(math.Max(0, finalVal+widenedMargin)*100) / 100,
				LowerBound:         math.Round(math.Max(0, finalVal-widenedMargin)*100) / 100,
			})
		}
	}

	return allResults, nil
}

func convertToFloat(val interface{}) (float64, error) {
	if val == nil {
		return 0, nil
	}
	switch v := val.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case string:
		var f float64
		_, err := fmt.Sscanf(v, "%f", &f)
		return f, err
	default:
		return 0, fmt.Errorf("unknown numeric type")
	}
}
