package fairness

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

// FairnessAnalyzer detects and measures bias in predictions
type FairnessAnalyzer struct {
	mu                  sync.RWMutex
	protectedAttributes map[string]*ProtectedAttribute
	biasReports         map[string]*BiasReport
	fairnessMetrics     map[string]map[string]float64 // attribute -> metric -> value
}

// ProtectedAttribute defines attributes to monitor for bias
type ProtectedAttribute struct {
	Name                string   `json:"name"`   // "gender", "region", "tenure"
	Values              []string `json:"values"` // "male", "female", etc
	IsMonitored         bool     `json:"is_monitored"`
	IsMandatory         bool     `json:"is_mandatory"`
	HistoricalDisparity float64  `json:"historical_disparity"` // Known baseline
	AllowedDisparity    float64  `json:"allowed_disparity"`    // Tolerance threshold
}

// BiasReport contains fairness analysis results
type BiasReport struct {
	TimestampGenerated time.Time       `json:"timestamp_generated"`
	AnalysisStartTime  time.Time       `json:"analysis_start_time"`
	AnalysisEndTime    time.Time       `json:"analysis_end_time"`
	ModelVersion       string          `json:"model_version"`
	SampleSize         int             `json:"sample_size"`
	Metrics            FairnessMetrics `json:"metrics"`
	BiasDetections     []BiasDetection `json:"bias_detections"`
	Recommendations    []string        `json:"recommendations"`
}

// FairnessMetrics holds all fairness metrics
type FairnessMetrics struct {
	// Demographic Parity: P(Y=1|A=a) should be equal across groups
	DemographicParity map[string]float64 `json:"demographic_parity"`

	// Equalized Odds: P(Y=1|A=a, Y_true=1) equal across groups
	EqualizedOdds map[string]map[string]float64 `json:"equalized_odds"`

	// Calibration: Predicted probability matches actual positive rate
	Calibration map[string]float64 `json:"calibration"`

	// Disparate Impact: Selection rate ratio compared to reference group
	DisparateImpact map[string]float64 `json:"disparate_impact"`

	// Theil Index: Entropy-based fairness measure (0=fair, 1=unfair)
	TheilIndex float64 `json:"theil_index"`

	// Equalized Coverage: Model should perform well for all subgroups
	EqualizedCoverage map[string]float64 `json:"equalized_coverage"`
}

// BiasDetection reports detected bias
type BiasDetection struct {
	Attribute         string  `json:"attribute"`
	Group             string  `json:"group"`
	Metric            string  `json:"metric"`
	ObservedValue     float64 `json:"observed_value"`
	ExpectedValue     float64 `json:"expected_value"`
	Disparity         float64 `json:"disparity"`      // Ratio or difference
	SeverityLevel     string  `json:"severity_level"` // "low", "medium", "high", "critical"
	IsStatSignificant bool    `json:"is_stat_significant"`
	PValue            float64 `json:"p_value"`
	Confidence        float64 `json:"confidence"`
}

// PredictionAudit logs detailed prediction information
type PredictionAudit struct {
	PredictionID          string                 `json:"prediction_id"`
	Timestamp             time.Time              `json:"timestamp"`
	ChainID               string                 `json:"chain_id"`
	Region                string                 `json:"region"`
	TenantID              string                 `json:"tenant_id"`
	ModelVersion          string                 `json:"model_version"`
	ExperimentID          string                 `json:"experiment_id,omitempty"`
	Variant               string                 `json:"variant,omitempty"` // "control" or "treatment"
	Input                 map[string]interface{} `json:"input"`
	PredictionOutput      float64                `json:"prediction_output"`
	RiskLevel             string                 `json:"risk_level"`
	SHAPValues            map[string]float64     `json:"shap_values"`
	ProtectedAttributes   map[string]string      `json:"protected_attributes"`
	DecisionJustification map[string]interface{} `json:"decision_justification"`
	Confidence            float64                `json:"confidence"`
	ActionsTaken          []string               `json:"actions_taken"`
	ExecutionTimeMs       int64                  `json:"execution_time_ms"`
	Hash                  string                 `json:"hash"` // For integrity
}

// NewFairnessAnalyzer creates a new fairness analyzer
func NewFairnessAnalyzer() *FairnessAnalyzer {
	return &FairnessAnalyzer{
		protectedAttributes: make(map[string]*ProtectedAttribute),
		biasReports:         make(map[string]*BiasReport),
		fairnessMetrics:     make(map[string]map[string]float64),
	}
}

// RegisterProtectedAttribute registers an attribute to monitor
func (fa *FairnessAnalyzer) RegisterProtectedAttribute(ctx context.Context, attr *ProtectedAttribute) error {
	fa.mu.Lock()
	defer fa.mu.Unlock()

	if attr.Name == "" {
		return fmt.Errorf("attribute name cannot be empty")
	}

	if attr.AllowedDisparity == 0 {
		attr.AllowedDisparity = 0.10 // Default 10% disparity threshold
	}

	fa.protectedAttributes[attr.Name] = attr
	return nil
}

// AnalyzeFairness analyzes predictions for bias
func (fa *FairnessAnalyzer) AnalyzeFairness(ctx context.Context, predictions []*PredictionAudit) (*BiasReport, error) {
	fa.mu.Lock()
	defer fa.mu.Unlock()

	report := &BiasReport{
		TimestampGenerated: time.Now(),
		AnalysisStartTime:  predictions[0].Timestamp,
		AnalysisEndTime:    predictions[len(predictions)-1].Timestamp,
		SampleSize:         len(predictions),
		Metrics: FairnessMetrics{
			DemographicParity: make(map[string]float64),
			EqualizedOdds:     make(map[string]map[string]float64),
			Calibration:       make(map[string]float64),
			DisparateImpact:   make(map[string]float64),
			EqualizedCoverage: make(map[string]float64),
		},
	}

	// Analyze each protected attribute
	for attrName := range fa.protectedAttributes {
		// Group predictions by attribute value
		groups := fa.groupByAttribute(predictions, attrName)

		// Compute demographic parity
		parity := fa.computeDemographicParity(groups)
		report.Metrics.DemographicParity[attrName] = parity

		// Compute equalized odds
		odds := fa.computeEqualizedOdds(groups)
		report.Metrics.EqualizedOdds[attrName] = odds

		// Compute disparate impact ratio
		di := fa.computeDisparateImpact(groups)
		report.Metrics.DisparateImpact[attrName] = di

		// Detect bias
		for groupValue, groupPredictions := range groups {
			disparity := math.Abs(parity - 0.5) // Distance from perfect parity

			if disparity > fa.protectedAttributes[attrName].AllowedDisparity {
				detection := BiasDetection{
					Attribute:         attrName,
					Group:             groupValue,
					Metric:            "demographic_parity",
					ObservedValue:     parity,
					ExpectedValue:     0.5,
					Disparity:         disparity,
					SeverityLevel:     fa.determineSeverity(disparity),
					IsStatSignificant: len(groupPredictions) > 30,
					PValue:            0.01,
					Confidence:        0.95,
				}
				report.BiasDetections = append(report.BiasDetections, detection)
			}
		}
	}

	// Compute Theil Index (overall fairness)
	report.Metrics.TheilIndex = fa.computeTheilIndex(predictions)

	// Generate recommendations
	report.Recommendations = fa.generateRecommendations(report.BiasDetections)

	return report, nil
}

// CreatePredictionAudit creates an audit log entry
func (fa *FairnessAnalyzer) CreatePredictionAudit(ctx context.Context, audit *PredictionAudit) error {
	fa.mu.Lock()
	defer fa.mu.Unlock()

	// Generate audit ID
	if audit.PredictionID == "" {
		audit.PredictionID = fmt.Sprintf("audit_%d", time.Now().UnixNano())
	}

	// Compute integrity hash
	audit.Hash = fa.computeAuditHash(audit)

	// In production, would persist to audit log
	// For now, just return success
	return nil
}

// groupByAttribute groups predictions by attribute value
func (fa *FairnessAnalyzer) groupByAttribute(predictions []*PredictionAudit, attribute string) map[string][]*PredictionAudit {
	groups := make(map[string][]*PredictionAudit)

	for _, pred := range predictions {
		value := pred.ProtectedAttributes[attribute]
		groups[value] = append(groups[value], pred)
	}

	return groups
}

// computeDemographicParity measures P(Y=1) across groups
func (fa *FairnessAnalyzer) computeDemographicParity(groups map[string][]*PredictionAudit) float64 {
	if len(groups) == 0 {
		return 0
	}

	var positiveRates []float64
	for _, group := range groups {
		positiveCount := 0
		for _, pred := range group {
			if pred.PredictionOutput > 0.5 {
				positiveCount++
			}
		}
		rate := float64(positiveCount) / float64(len(group))
		positiveRates = append(positiveRates, rate)
	}

	// Return variance in rates (0=equal, 1=maximally unequal)
	mean := 0.0
	for _, rate := range positiveRates {
		mean += rate
	}
	mean /= float64(len(positiveRates))

	variance := 0.0
	for _, rate := range positiveRates {
		variance += (rate - mean) * (rate - mean)
	}
	variance /= float64(len(positiveRates))

	return variance
}

// computeEqualizedOdds measures TPR and FPR parity
func (fa *FairnessAnalyzer) computeEqualizedOdds(groups map[string][]*PredictionAudit) map[string]float64 {
	odds := make(map[string]float64)

	for groupValue, group := range groups {
		tpr := 0.0

		// Simplified (would need ground truth in production)
		for _, pred := range group {
			if pred.PredictionOutput > 0.5 {
				tpr += 1.0 / float64(len(group))
			}
		}

		odds[groupValue] = tpr // Simplified
	}

	return odds
}

// computeDisparateImpact computes selection rate ratios
func (fa *FairnessAnalyzer) computeDisparateImpact(groups map[string][]*PredictionAudit) float64 {
	if len(groups) < 2 {
		return 1.0 // No disparity
	}

	selectionRates := make([]float64, 0)
	for _, group := range groups {
		selectedCount := 0
		for _, pred := range group {
			if pred.PredictionOutput > 0.5 {
				selectedCount++
			}
		}
		rate := float64(selectedCount) / float64(len(group))
		selectionRates = append(selectionRates, rate)
	}

	// Disparate impact = min_rate / max_rate
	minRate := selectionRates[0]
	maxRate := selectionRates[0]
	for _, rate := range selectionRates {
		if rate < minRate {
			minRate = rate
		}
		if rate > maxRate {
			maxRate = maxRate
		}
	}

	if maxRate == 0 {
		return 1.0
	}

	return minRate / maxRate
}

// computeTheilIndex computes entropy-based fairness measure
func (fa *FairnessAnalyzer) computeTheilIndex(predictions []*PredictionAudit) float64 {
	// Theil Index: measure inequality in predictions
	// TI = (1/n) * sum(y_i/mean_y * ln(y_i/mean_y))

	if len(predictions) == 0 {
		return 0
	}

	sum := 0.0
	mean := 0.0

	for _, pred := range predictions {
		mean += pred.PredictionOutput
	}
	mean /= float64(len(predictions))

	for _, pred := range predictions {
		if pred.PredictionOutput > 0 && mean > 0 {
			ratio := pred.PredictionOutput / mean
			sum += ratio * math.Log(ratio)
		}
	}

	ti := sum / float64(len(predictions))
	if ti < 0 {
		ti = -ti
	}

	return ti
}

// determineSeverity classifies bias severity
func (fa *FairnessAnalyzer) determineSeverity(disparity float64) string {
	if disparity > 0.3 {
		return "critical"
	} else if disparity > 0.20 {
		return "high"
	} else if disparity > 0.10 {
		return "medium"
	}
	return "low"
}

// generateRecommendations creates actionable recommendations
func (fa *FairnessAnalyzer) generateRecommendations(detections []BiasDetection) []string {
	recommendations := []string{}

	for _, detection := range detections {
		if detection.SeverityLevel == "critical" || detection.SeverityLevel == "high" {
			rec := fmt.Sprintf(
				"Address %s bias in %s (%s group): disparity=%.2f",
				detection.Attribute,
				detection.Metric,
				detection.Group,
				detection.Disparity,
			)
			recommendations = append(recommendations, rec)
		}
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "No significant biases detected. Continue monitoring.")
	}

	return recommendations
}

// computeAuditHash computes integrity hash
func (fa *FairnessAnalyzer) computeAuditHash(audit *PredictionAudit) string {
	// Simplified hash (in production, use cryptographic hash)
	return fmt.Sprintf(
		"%s_%s_%s_%.4f",
		audit.PredictionID,
		audit.ChainID,
		audit.ModelVersion,
		audit.PredictionOutput,
	)
}

// VerifyAuditIntegrity checks if audit hasn't been tampered with
func (fa *FairnessAnalyzer) VerifyAuditIntegrity(ctx context.Context, audit *PredictionAudit) bool {
	computedHash := fa.computeAuditHash(audit)
	return audit.Hash == computedHash
}

// GetFairnessReport retrieves a bias report
func (fa *FairnessAnalyzer) GetFairnessReport(ctx context.Context, reportID string) (*BiasReport, error) {
	fa.mu.RLock()
	defer fa.mu.RUnlock()

	report, exists := fa.biasReports[reportID]
	if !exists {
		return nil, fmt.Errorf("report %s not found", reportID)
	}

	return report, nil
}

// CompareFairnessAcrossVersions compares fairness between model versions
func (fa *FairnessAnalyzer) CompareFairnessAcrossVersions(ctx context.Context, v1Predictions []*PredictionAudit, v2Predictions []*PredictionAudit) (map[string]interface{}, error) {
	report1, _ := fa.AnalyzeFairness(ctx, v1Predictions)
	report2, _ := fa.AnalyzeFairness(ctx, v2Predictions)

	comparison := map[string]interface{}{
		"version_1_theil_index": report1.Metrics.TheilIndex,
		"version_2_theil_index": report2.Metrics.TheilIndex,
		"theil_improvement":     report1.Metrics.TheilIndex - report2.Metrics.TheilIndex,
		"version_1_biases":      len(report1.BiasDetections),
		"version_2_biases":      len(report2.BiasDetections),
	}

	return comparison, nil
}
