package ai

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
)

// SLOForecaster predicts SLO breaches before they occur
type SLOForecaster struct {
	logger *slog.Logger
}

// NewSLOForecaster creates a new SLO forecaster
func NewSLOForecaster(logger *slog.Logger) *SLOForecaster {
	return &SLOForecaster{
		logger: logger,
	}
}

// SLODefinition defines an SLO target
type SLODefinition struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	JobID         uuid.UUID `json:"job_id"`
	MetricType    string    `json:"metric_type"` // latency, success_rate, throughput
	Target        float64   `json:"target"`      // e.g., 99.9 for 99.9%
	Threshold     float64   `json:"threshold"`   // Warning threshold
	WindowMinutes int       `json:"window_minutes"`
	Critical      bool      `json:"critical"`
}

// SLODataPoint represents a measurement
type SLODataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	JobID     uuid.UUID `json:"job_id"`
}

// SLOForecast contains prediction results
type SLOForecast struct {
	SLOID               uuid.UUID  `json:"slo_id"`
	SLOName             string     `json:"slo_name"`
	JobID               uuid.UUID  `json:"job_id"`
	CurrentValue        float64    `json:"current_value"`
	Target              float64    `json:"target"`
	PredictedValue      float64    `json:"predicted_value"`
	PredictionTime      time.Time  `json:"prediction_time"`
	BreachProbability   float64    `json:"breach_probability"` // 0-1
	BreachETA           *time.Time `json:"breach_eta,omitempty"`
	TrendDirection      string     `json:"trend_direction"` // improving, stable, degrading
	TrendVelocity       float64    `json:"trend_velocity"`  // rate of change per hour
	RiskLevel           string     `json:"risk_level"`      // low, medium, high, critical
	Confidence          float64    `json:"confidence"`
	ContributingFactors []Factor   `json:"contributing_factors"`
	Recommendations     []string   `json:"recommendations"`
}

// Factor describes something contributing to SLO risk
type Factor struct {
	Name        string  `json:"name"`
	Impact      float64 `json:"impact"` // -1 to 1, negative = bad
	Description string  `json:"description"`
}

// SLODashboard provides real-time SLO status
type SLODashboard struct {
	GeneratedAt   time.Time       `json:"generated_at"`
	OverallHealth string          `json:"overall_health"` // healthy, at_risk, degraded
	HealthScore   float64         `json:"health_score"`   // 0-100
	TotalSLOs     int             `json:"total_slos"`
	HealthySLOs   int             `json:"healthy_slos"`
	AtRiskSLOs    int             `json:"at_risk_slos"`
	BreachedSLOs  int             `json:"breached_slos"`
	Forecasts     []SLOForecast   `json:"forecasts"`
	ErrorBudget   ErrorBudgetInfo `json:"error_budget"`
	TrendingSLOs  []TrendingSLO   `json:"trending_slos"`
}

// ErrorBudgetInfo tracks error budget consumption
type ErrorBudgetInfo struct {
	TotalBudgetMinutes  float64    `json:"total_budget_minutes"`
	ConsumedMinutes     float64    `json:"consumed_minutes"`
	RemainingMinutes    float64    `json:"remaining_minutes"`
	ConsumptionRate     float64    `json:"consumption_rate_per_hour"`
	ProjectedExhaustion *time.Time `json:"projected_exhaustion,omitempty"`
	BurnRate            float64    `json:"burn_rate"` // 1.0 = normal, >1 = burning fast
	WindowStart         time.Time  `json:"window_start"`
	WindowEnd           time.Time  `json:"window_end"`
}

// TrendingSLO shows SLOs changing significantly
type TrendingSLO struct {
	SLOID      uuid.UUID `json:"slo_id"`
	SLOName    string    `json:"slo_name"`
	Direction  string    `json:"direction"` // up, down
	ChangeRate float64   `json:"change_rate_percent"`
	Reason     string    `json:"reason"`
}

// ForecastSLOBreach predicts if/when an SLO will breach
func (f *SLOForecaster) ForecastSLOBreach(
	ctx context.Context,
	slo SLODefinition,
	dataPoints []SLODataPoint,
	forecastHours int,
) (*SLOForecast, error) {
	f.logger.Info("Forecasting SLO breach",
		"slo_id", slo.ID,
		"slo_name", slo.Name,
		"data_points", len(dataPoints),
	)

	if len(dataPoints) < 10 {
		return nil, fmt.Errorf("insufficient data points for forecasting (min 10, got %d)", len(dataPoints))
	}

	// Sort by timestamp
	sort.Slice(dataPoints, func(i, j int) bool {
		return dataPoints[i].Timestamp.Before(dataPoints[j].Timestamp)
	})

	// Calculate current value (last N points average)
	current := f.calculateCurrentValue(dataPoints)

	// Calculate trend using linear regression
	slope, intercept := f.linearRegression(dataPoints)

	// Forecast future value
	forecastTime := time.Now().Add(time.Duration(forecastHours) * time.Hour)
	predictedValue := f.predict(forecastTime, slope, intercept, dataPoints[0].Timestamp)

	// Calculate breach probability
	breachProb := f.calculateBreachProbability(slo, current, predictedValue, dataPoints)

	// Estimate time to breach
	var breachETA *time.Time
	if breachProb > 0.5 && slope != 0 {
		eta := f.estimateBreachTime(slo, slope, intercept, dataPoints[0].Timestamp, current)
		if eta != nil && eta.After(time.Now()) {
			breachETA = eta
		}
	}

	// Determine trend
	trend := f.determineTrend(slope, slo.MetricType)

	// Calculate confidence based on data quality
	confidence := f.calculateConfidence(dataPoints, slope)

	forecast := &SLOForecast{
		SLOID:               slo.ID,
		SLOName:             slo.Name,
		JobID:               slo.JobID,
		CurrentValue:        current,
		Target:              slo.Target,
		PredictedValue:      predictedValue,
		PredictionTime:      forecastTime,
		BreachProbability:   breachProb,
		BreachETA:           breachETA,
		TrendDirection:      trend.direction,
		TrendVelocity:       slope * 3600, // per hour
		RiskLevel:           f.assessRisk(breachProb, breachETA, slo.Critical),
		Confidence:          confidence,
		ContributingFactors: f.identifyFactors(dataPoints, slope),
		Recommendations:     f.generateRecommendations(breachProb, trend.direction, slo.MetricType),
	}

	return forecast, nil
}

// calculateCurrentValue computes the current SLO value
func (f *SLOForecaster) calculateCurrentValue(dataPoints []SLODataPoint) float64 {
	// Use last 5 points average
	window := 5
	if len(dataPoints) < window {
		window = len(dataPoints)
	}

	var sum float64
	for i := len(dataPoints) - window; i < len(dataPoints); i++ {
		sum += dataPoints[i].Value
	}
	return sum / float64(window)
}

// linearRegression calculates trend line
func (f *SLOForecaster) linearRegression(dataPoints []SLODataPoint) (slope, intercept float64) {
	n := float64(len(dataPoints))
	if n < 2 {
		return 0, 0
	}

	// Convert timestamps to seconds from start
	start := dataPoints[0].Timestamp
	var sumX, sumY, sumXY, sumX2 float64

	for _, dp := range dataPoints {
		x := dp.Timestamp.Sub(start).Seconds()
		y := dp.Value
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	denominator := n*sumX2 - sumX*sumX
	if denominator == 0 {
		return 0, sumY / n
	}

	slope = (n*sumXY - sumX*sumY) / denominator
	intercept = (sumY - slope*sumX) / n

	return slope, intercept
}

// predict calculates the forecasted value at a given time
func (f *SLOForecaster) predict(t time.Time, slope, intercept float64, startTime time.Time) float64 {
	x := t.Sub(startTime).Seconds()
	return slope*x + intercept
}

// calculateBreachProbability estimates likelihood of breach
func (f *SLOForecaster) calculateBreachProbability(slo SLODefinition, current, predicted float64, dataPoints []SLODataPoint) float64 {
	// Calculate standard deviation
	var sum, sumSq float64
	for _, dp := range dataPoints {
		sum += dp.Value
		sumSq += dp.Value * dp.Value
	}
	n := float64(len(dataPoints))
	mean := sum / n
	variance := (sumSq / n) - (mean * mean)
	stdDev := math.Sqrt(math.Abs(variance))

	// Distance from target in standard deviations
	var distanceFromTarget float64
	switch slo.MetricType {
	case "latency":
		// For latency, lower is better, breach if above target
		distanceFromTarget = (predicted - slo.Target) / (stdDev + 0.0001)
	case "success_rate", "throughput":
		// For success rate, higher is better, breach if below target
		distanceFromTarget = (slo.Target - predicted) / (stdDev + 0.0001)
	}

	// Convert to probability using sigmoid
	prob := 1.0 / (1.0 + math.Exp(-distanceFromTarget))

	// Adjust for trend
	if current < predicted && slo.MetricType == "latency" {
		prob *= 1.2 // Increasing latency = higher breach risk
	}

	// Clamp to 0-1
	if prob > 1 {
		prob = 1
	}
	if prob < 0 {
		prob = 0
	}

	return prob
}

// estimateBreachTime calculates when breach will occur
func (f *SLOForecaster) estimateBreachTime(slo SLODefinition, slope, intercept float64, startTime time.Time, current float64) *time.Time {
	if slope == 0 {
		return nil
	}

	// Solve for x when y = target
	var targetValue float64
	switch slo.MetricType {
	case "latency":
		targetValue = slo.Target
		if slope < 0 { // Improving, won't breach
			return nil
		}
	default:
		targetValue = slo.Target
		if slope > 0 { // Improving, won't breach
			return nil
		}
	}

	x := (targetValue - intercept) / slope
	if x < 0 {
		return nil
	}

	eta := startTime.Add(time.Duration(x) * time.Second)
	return &eta
}

// TrendInfo holds trend data
type TrendInfo struct {
	direction string
	velocity  float64
}

// determineTrend classifies the trend direction
func (f *SLOForecaster) determineTrend(slope float64, metricType string) TrendInfo {
	absSlope := math.Abs(slope)

	// Threshold for "significant" change
	threshold := 0.0001 // per second

	if absSlope < threshold {
		return TrendInfo{direction: "stable", velocity: slope}
	}

	switch metricType {
	case "latency":
		if slope > 0 {
			return TrendInfo{direction: "degrading", velocity: slope}
		}
		return TrendInfo{direction: "improving", velocity: slope}
	default: // success_rate, throughput
		if slope > 0 {
			return TrendInfo{direction: "improving", velocity: slope}
		}
		return TrendInfo{direction: "degrading", velocity: slope}
	}
}

// calculateConfidence estimates prediction confidence
func (f *SLOForecaster) calculateConfidence(dataPoints []SLODataPoint, slope float64) float64 {
	// More data = higher confidence
	dataConfidence := math.Min(float64(len(dataPoints))/100.0, 1.0)

	// Stable trends are more predictable
	trendConfidence := 1.0 - math.Min(math.Abs(slope)*1000, 0.5)

	return (dataConfidence + trendConfidence) / 2
}

// assessRisk determines the overall risk level
func (f *SLOForecaster) assessRisk(breachProb float64, breachETA *time.Time, critical bool) string {
	// Critical SLOs have elevated risk
	multiplier := 1.0
	if critical {
		multiplier = 1.5
	}

	adjustedProb := breachProb * multiplier

	// Check ETA urgency
	if breachETA != nil {
		hoursToBreak := time.Until(*breachETA).Hours()
		if hoursToBreak < 1 {
			return "critical"
		}
		if hoursToBreak < 6 {
			adjustedProb *= 1.5
		}
	}

	if adjustedProb > 0.8 {
		return "critical"
	}
	if adjustedProb > 0.5 {
		return "high"
	}
	if adjustedProb > 0.2 {
		return "medium"
	}
	return "low"
}

// identifyFactors finds contributing factors
func (f *SLOForecaster) identifyFactors(dataPoints []SLODataPoint, slope float64) []Factor {
	var factors []Factor

	// Check for time-based patterns
	hourCounts := make(map[int]float64)
	hourSums := make(map[int]float64)
	for _, dp := range dataPoints {
		h := dp.Timestamp.Hour()
		hourCounts[h]++
		hourSums[h] += dp.Value
	}

	// Find problematic hours
	for h, count := range hourCounts {
		if count > 0 {
			avg := hourSums[h] / count
			if avg > f.calculateCurrentValue(dataPoints)*1.2 {
				factors = append(factors, Factor{
					Name:        fmt.Sprintf("Peak hour: %d:00", h),
					Impact:      -0.3,
					Description: fmt.Sprintf("Higher latency observed around %d:00", h),
				})
			}
		}
	}

	// Check recent volatility
	if len(dataPoints) > 10 {
		recent := dataPoints[len(dataPoints)-10:]
		var variance float64
		mean := f.calculateCurrentValue(recent)
		for _, dp := range recent {
			diff := dp.Value - mean
			variance += diff * diff
		}
		variance /= float64(len(recent))
		if variance > mean*0.1 {
			factors = append(factors, Factor{
				Name:        "High volatility",
				Impact:      -0.2,
				Description: "Recent measurements show high variability",
			})
		}
	}

	return factors
}

// generateRecommendations provides actionable suggestions
func (f *SLOForecaster) generateRecommendations(breachProb float64, trend, metricType string) []string {
	var recs []string

	if breachProb > 0.7 {
		recs = append(recs, "⚠️ High breach risk - consider immediate intervention")
	}

	if trend == "degrading" {
		switch metricType {
		case "latency":
			recs = append(recs, "Investigate slow queries or external dependencies")
			recs = append(recs, "Consider scaling up resources")
			recs = append(recs, "Review recent code deployments")
		case "success_rate":
			recs = append(recs, "Check error logs for recurring issues")
			recs = append(recs, "Review retry policies")
			recs = append(recs, "Verify external service health")
		}
	}

	if trend == "stable" && breachProb > 0.3 {
		recs = append(recs, "Consider adjusting SLO target if consistently close to threshold")
	}

	return recs
}

// GenerateDashboard creates a real-time SLO dashboard
func (f *SLOForecaster) GenerateDashboard(
	ctx context.Context,
	slos []SLODefinition,
	dataByJob map[uuid.UUID][]SLODataPoint,
) (*SLODashboard, error) {
	dashboard := &SLODashboard{
		GeneratedAt: time.Now(),
		TotalSLOs:   len(slos),
	}

	for _, slo := range slos {
		data := dataByJob[slo.JobID]
		if len(data) < 10 {
			continue
		}

		forecast, err := f.ForecastSLOBreach(ctx, slo, data, 24)
		if err != nil {
			continue
		}

		dashboard.Forecasts = append(dashboard.Forecasts, *forecast)

		// Categorize
		switch forecast.RiskLevel {
		case "critical", "high":
			dashboard.BreachedSLOs++
		case "medium":
			dashboard.AtRiskSLOs++
		default:
			dashboard.HealthySLOs++
		}
	}

	// Calculate overall health
	if dashboard.BreachedSLOs > 0 {
		dashboard.OverallHealth = "degraded"
	} else if dashboard.AtRiskSLOs > 0 {
		dashboard.OverallHealth = "at_risk"
	} else {
		dashboard.OverallHealth = "healthy"
	}

	dashboard.HealthScore = float64(dashboard.HealthySLOs) / float64(max(dashboard.TotalSLOs, 1)) * 100

	return dashboard, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
