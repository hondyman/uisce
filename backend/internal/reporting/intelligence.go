package reporting

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// ============================================================================
// AI-POWERED REPORT INTELLIGENCE
// ============================================================================

// ReportIntelligence provides AI-powered insights for reports
type ReportIntelligence struct {
	anomalyDetector  *AnomalyDetector
	trendAnalyzer    *TrendAnalyzer
	insightGenerator *InsightGenerator
	querySuggester   *QuerySuggester
}

// NewReportIntelligence creates a new report intelligence service
func NewReportIntelligence() *ReportIntelligence {
	return &ReportIntelligence{
		anomalyDetector:  NewAnomalyDetector(),
		trendAnalyzer:    NewTrendAnalyzer(),
		insightGenerator: NewInsightGenerator(),
		querySuggester:   NewQuerySuggester(),
	}
}

// ============================================================================
// ANOMALY DETECTION
// ============================================================================

// AnomalyDetector detects anomalies in report data
type AnomalyDetector struct {
	threshold float64 // Z-score threshold for anomaly detection
}

// NewAnomalyDetector creates an anomaly detector
func NewAnomalyDetector() *AnomalyDetector {
	return &AnomalyDetector{
		threshold: 2.5, // Values beyond 2.5 standard deviations are anomalies
	}
}

// Anomaly represents a detected anomaly
type Anomaly struct {
	Field         string     `json:"field"`
	Value         float64    `json:"value"`
	ExpectedRange [2]float64 `json:"expected_range"`
	ZScore        float64    `json:"z_score"`
	Severity      string     `json:"severity"` // low, medium, high, critical
	Description   string     `json:"description"`
	Timestamp     time.Time  `json:"timestamp,omitempty"`
	Context       string     `json:"context,omitempty"`
}

// AnomalyReport contains all detected anomalies
type AnomalyReport struct {
	Anomalies    []Anomaly `json:"anomalies"`
	TotalChecks  int       `json:"total_checks"`
	AnomalyCount int       `json:"anomaly_count"`
	AnomalyRate  float64   `json:"anomaly_rate"`
	AnalyzedAt   time.Time `json:"analyzed_at"`
}

// DetectAnomalies analyzes report data for anomalies
func (ad *AnomalyDetector) DetectAnomalies(data []map[string]interface{}, numericFields []string) *AnomalyReport {
	report := &AnomalyReport{
		Anomalies:  make([]Anomaly, 0),
		AnalyzedAt: time.Now(),
	}

	for _, field := range numericFields {
		values := extractNumericValues(data, field)
		if len(values) < 3 {
			continue
		}

		report.TotalChecks++

		// Calculate statistics
		mean := calculateMean(values)
		stdDev := calculateStdDev(values, mean)

		if stdDev == 0 {
			continue
		}

		// Check each value for anomalies
		for i, val := range values {
			zScore := (val - mean) / stdDev

			if math.Abs(zScore) > ad.threshold {
				severity := ad.classifySeverity(zScore)

				anomaly := Anomaly{
					Field:         field,
					Value:         val,
					ExpectedRange: [2]float64{mean - 2*stdDev, mean + 2*stdDev},
					ZScore:        zScore,
					Severity:      severity,
					Description:   ad.generateDescription(field, val, mean, zScore),
				}

				// Try to get context (e.g., date from same row)
				if i < len(data) {
					if ts, ok := data[i]["date"].(string); ok {
						if t, err := time.Parse(time.RFC3339, ts); err == nil {
							anomaly.Timestamp = t
						}
					}
					anomaly.Context = fmt.Sprintf("Row %d", i+1)
				}

				report.Anomalies = append(report.Anomalies, anomaly)
				report.AnomalyCount++
			}
		}
	}

	if report.TotalChecks > 0 {
		report.AnomalyRate = float64(report.AnomalyCount) / float64(report.TotalChecks*len(data))
	}

	return report
}

func (ad *AnomalyDetector) classifySeverity(zScore float64) string {
	absZ := math.Abs(zScore)
	switch {
	case absZ > 4:
		return "critical"
	case absZ > 3.5:
		return "high"
	case absZ > 3:
		return "medium"
	default:
		return "low"
	}
}

func (ad *AnomalyDetector) generateDescription(field string, value, mean, zScore float64) string {
	direction := "above"
	if zScore < 0 {
		direction = "below"
	}
	percentDiff := ((value - mean) / mean) * 100
	return fmt.Sprintf("%s value of %.2f is %.1f%% %s the average (%.2f)",
		field, value, math.Abs(percentDiff), direction, mean)
}

// ============================================================================
// TREND ANALYSIS
// ============================================================================

// TrendAnalyzer analyzes trends in time-series data
type TrendAnalyzer struct{}

// NewTrendAnalyzer creates a trend analyzer
func NewTrendAnalyzer() *TrendAnalyzer {
	return &TrendAnalyzer{}
}

// Trend represents a detected trend
type Trend struct {
	Field      string    `json:"field"`
	Direction  string    `json:"direction"`   // increasing, decreasing, stable, volatile
	Strength   float64   `json:"strength"`    // 0-1 correlation coefficient
	ChangeRate float64   `json:"change_rate"` // Percentage change per period
	Confidence float64   `json:"confidence"`
	StartValue float64   `json:"start_value"`
	EndValue   float64   `json:"end_value"`
	Periods    int       `json:"periods"`
	Forecast   []float64 `json:"forecast,omitempty"` // Next N predicted values
}

// TrendReport contains trend analysis results
type TrendReport struct {
	Trends     []Trend   `json:"trends"`
	Summary    string    `json:"summary"`
	AnalyzedAt time.Time `json:"analyzed_at"`
}

// AnalyzeTrends analyzes trends in time-series data
func (ta *TrendAnalyzer) AnalyzeTrends(data []map[string]interface{}, valueField string, timeField string) *TrendReport {
	report := &TrendReport{
		Trends:     make([]Trend, 0),
		AnalyzedAt: time.Now(),
	}

	// Sort data by time
	sortByTime(data, timeField)

	values := extractNumericValues(data, valueField)
	if len(values) < 3 {
		report.Summary = "Insufficient data for trend analysis"
		return report
	}

	// Calculate linear regression
	slope, intercept, r := linearRegression(values)

	trend := Trend{
		Field:      valueField,
		Periods:    len(values),
		StartValue: values[0],
		EndValue:   values[len(values)-1],
	}

	// Determine direction
	if math.Abs(slope) < 0.01 {
		trend.Direction = "stable"
	} else if slope > 0 {
		trend.Direction = "increasing"
	} else {
		trend.Direction = "decreasing"
	}

	// Calculate strength (R-squared)
	trend.Strength = r * r
	trend.Confidence = math.Abs(r)

	// Calculate change rate
	if values[0] != 0 {
		trend.ChangeRate = ((values[len(values)-1] - values[0]) / values[0]) * 100
	}

	// Generate forecast (next 3 periods)
	for i := 1; i <= 3; i++ {
		forecastValue := intercept + slope*float64(len(values)+i-1)
		trend.Forecast = append(trend.Forecast, forecastValue)
	}

	report.Trends = append(report.Trends, trend)
	report.Summary = ta.generateTrendSummary(trend)

	return report
}

func (ta *TrendAnalyzer) generateTrendSummary(trend Trend) string {
	var summary strings.Builder

	summary.WriteString(fmt.Sprintf("%s is %s", trend.Field, trend.Direction))

	if trend.Direction != "stable" {
		summary.WriteString(fmt.Sprintf(" with a %.1f%% change over %d periods",
			math.Abs(trend.ChangeRate), trend.Periods))
	}

	if trend.Confidence > 0.8 {
		summary.WriteString(" (high confidence)")
	} else if trend.Confidence > 0.5 {
		summary.WriteString(" (moderate confidence)")
	} else {
		summary.WriteString(" (low confidence)")
	}

	return summary.String()
}

// ============================================================================
// INSIGHT GENERATION
// ============================================================================

// InsightGenerator generates natural language insights from data
type InsightGenerator struct{}

// NewInsightGenerator creates an insight generator
func NewInsightGenerator() *InsightGenerator {
	return &InsightGenerator{}
}

// Insight represents a generated insight
type Insight struct {
	Type        string   `json:"type"` // comparison, pattern, anomaly, trend, summary
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Importance  int      `json:"importance"` // 1-10
	Fields      []string `json:"fields"`
	Data        any      `json:"data,omitempty"`
	ActionItems []string `json:"action_items,omitempty"`
}

// InsightReport contains generated insights
type InsightReport struct {
	Insights    []Insight `json:"insights"`
	GeneratedAt time.Time `json:"generated_at"`
}

// GenerateInsights generates insights from report data
func (ig *InsightGenerator) GenerateInsights(
	data []map[string]interface{},
	definition *ReportDefinition,
	anomalies *AnomalyReport,
	trends *TrendReport,
) *InsightReport {

	report := &InsightReport{
		Insights:    make([]Insight, 0),
		GeneratedAt: time.Now(),
	}

	// Summary insight
	report.Insights = append(report.Insights, ig.generateSummaryInsight(data, definition))

	// Anomaly insights
	if anomalies != nil && len(anomalies.Anomalies) > 0 {
		report.Insights = append(report.Insights, ig.generateAnomalyInsight(anomalies))
	}

	// Trend insights
	if trends != nil && len(trends.Trends) > 0 {
		report.Insights = append(report.Insights, ig.generateTrendInsight(trends))
	}

	// Top/Bottom performers
	if len(data) > 5 {
		report.Insights = append(report.Insights, ig.generateTopPerformersInsight(data)...)
	}

	// Sort by importance
	sort.Slice(report.Insights, func(i, j int) bool {
		return report.Insights[i].Importance > report.Insights[j].Importance
	})

	return report
}

func (ig *InsightGenerator) generateSummaryInsight(data []map[string]interface{}, def *ReportDefinition) Insight {
	return Insight{
		Type:        "summary",
		Title:       "Report Overview",
		Description: fmt.Sprintf("This report contains %d records for %s", len(data), def.DisplayName),
		Importance:  5,
	}
}

func (ig *InsightGenerator) generateAnomalyInsight(anomalies *AnomalyReport) Insight {
	criticalCount := 0
	for _, a := range anomalies.Anomalies {
		if a.Severity == "critical" || a.Severity == "high" {
			criticalCount++
		}
	}

	description := fmt.Sprintf("%d anomalies detected in the data", len(anomalies.Anomalies))
	importance := 6
	var actions []string

	if criticalCount > 0 {
		description += fmt.Sprintf(", including %d critical/high severity", criticalCount)
		importance = 9
		actions = append(actions, "Review critical anomalies immediately")
		actions = append(actions, "Verify data sources for accuracy")
	}

	return Insight{
		Type:        "anomaly",
		Title:       "Anomaly Detection",
		Description: description,
		Importance:  importance,
		ActionItems: actions,
		Data:        anomalies.Anomalies[:min(5, len(anomalies.Anomalies))],
	}
}

func (ig *InsightGenerator) generateTrendInsight(trends *TrendReport) Insight {
	if len(trends.Trends) == 0 {
		return Insight{}
	}

	t := trends.Trends[0]
	importance := 7
	var actions []string

	if t.Direction == "decreasing" && t.ChangeRate < -10 {
		importance = 8
		actions = append(actions, "Investigate cause of decline")
	} else if t.Direction == "increasing" && t.ChangeRate > 20 {
		importance = 8
		actions = append(actions, "Analyze drivers of growth")
	}

	return Insight{
		Type:        "trend",
		Title:       "Trend Analysis",
		Description: trends.Summary,
		Importance:  importance,
		Fields:      []string{t.Field},
		ActionItems: actions,
		Data:        t,
	}
}

func (ig *InsightGenerator) generateTopPerformersInsight(data []map[string]interface{}) []Insight {
	insights := make([]Insight, 0)

	// Find numeric fields
	if len(data) == 0 {
		return insights
	}

	for field := range data[0] {
		values := extractNumericValues(data, field)
		if len(values) < 5 {
			continue
		}

		// Sort and get top/bottom
		sorted := make([]float64, len(values))
		copy(sorted, values)
		sort.Float64s(sorted)

		top3 := sorted[len(sorted)-3:]
		bottom3 := sorted[:3]

		mean := calculateMean(values)

		// Top performers insight
		if top3[0] > mean*1.5 {
			insights = append(insights, Insight{
				Type:        "pattern",
				Title:       fmt.Sprintf("Top %s Performers", field),
				Description: fmt.Sprintf("Top values significantly outperform average (%.2f vs %.2f)", top3[2], mean),
				Importance:  6,
				Fields:      []string{field},
			})
		}

		// Bottom performers insight
		if bottom3[2] < mean*0.5 && bottom3[2] > 0 {
			insights = append(insights, Insight{
				Type:        "pattern",
				Title:       fmt.Sprintf("Low %s Performers", field),
				Description: fmt.Sprintf("Bottom values significantly underperform average (%.2f vs %.2f)", bottom3[0], mean),
				Importance:  6,
				Fields:      []string{field},
				ActionItems: []string{"Investigate underperforming items"},
			})
		}

		break // Just analyze first numeric field for now
	}

	return insights
}

// ============================================================================
// QUERY SUGGESTIONS
// ============================================================================

// QuerySuggester suggests relevant queries based on context
type QuerySuggester struct{}

// NewQuerySuggester creates a query suggester
func NewQuerySuggester() *QuerySuggester {
	return &QuerySuggester{}
}

// QuerySuggestion represents a suggested query
type QuerySuggestion struct {
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Query       json.RawMessage `json:"query"`
	Category    string          `json:"category"` // drill-down, compare, time-shift, aggregate
	Relevance   float64         `json:"relevance"`
}

// SuggestQueries suggests related queries based on current report
func (qs *QuerySuggester) SuggestQueries(
	ctx context.Context,
	definition *ReportDefinition,
	currentParams map[string]interface{},
) []QuerySuggestion {

	suggestions := make([]QuerySuggestion, 0)

	// Time-based suggestions
	if hasDateParameter(definition) {
		suggestions = append(suggestions, QuerySuggestion{
			Title:       "Previous Period Comparison",
			Description: "Compare with the same metrics from the previous period",
			Category:    "time-shift",
			Relevance:   0.9,
		})

		suggestions = append(suggestions, QuerySuggestion{
			Title:       "Year-over-Year Analysis",
			Description: "Compare current data with same period last year",
			Category:    "time-shift",
			Relevance:   0.85,
		})
	}

	// Aggregation suggestions
	suggestions = append(suggestions, QuerySuggestion{
		Title:       "Summary by Category",
		Description: "View aggregated totals grouped by category",
		Category:    "aggregate",
		Relevance:   0.8,
	})

	// Drill-down suggestions based on context
	suggestions = append(suggestions, QuerySuggestion{
		Title:       "Detailed Breakdown",
		Description: "View detailed records for specific items",
		Category:    "drill-down",
		Relevance:   0.75,
	})

	// Sort by relevance
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Relevance > suggestions[j].Relevance
	})

	return suggestions
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func extractNumericValues(data []map[string]interface{}, field string) []float64 {
	values := make([]float64, 0, len(data))
	for _, row := range data {
		if v, ok := row[field]; ok {
			switch val := v.(type) {
			case float64:
				values = append(values, val)
			case int:
				values = append(values, float64(val))
			case int64:
				values = append(values, float64(val))
			}
		}
	}
	return values
}

func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateStdDev(values []float64, mean float64) float64 {
	if len(values) < 2 {
		return 0
	}
	sumSq := 0.0
	for _, v := range values {
		diff := v - mean
		sumSq += diff * diff
	}
	return math.Sqrt(sumSq / float64(len(values)-1))
}

func linearRegression(values []float64) (slope, intercept, r float64) {
	n := float64(len(values))
	if n < 2 {
		return 0, 0, 0
	}

	sumX, sumY, sumXY, sumX2, sumY2 := 0.0, 0.0, 0.0, 0.0, 0.0

	for i, y := range values {
		x := float64(i)
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
		sumY2 += y * y
	}

	denom := n*sumX2 - sumX*sumX
	if denom == 0 {
		return 0, sumY / n, 0
	}

	slope = (n*sumXY - sumX*sumY) / denom
	intercept = (sumY - slope*sumX) / n

	// Calculate correlation coefficient
	numerator := n*sumXY - sumX*sumY
	denominator := math.Sqrt((n*sumX2 - sumX*sumX) * (n*sumY2 - sumY*sumY))
	if denominator != 0 {
		r = numerator / denominator
	}

	return
}

func sortByTime(data []map[string]interface{}, timeField string) {
	sort.Slice(data, func(i, j int) bool {
		ti, _ := data[i][timeField].(string)
		tj, _ := data[j][timeField].(string)
		return ti < tj
	})
}

func hasDateParameter(def *ReportDefinition) bool {
	for _, param := range def.ParametersSchema {
		if param.Type == "date" || param.Type == "dateRange" {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
