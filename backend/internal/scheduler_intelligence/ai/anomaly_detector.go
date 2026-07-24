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

// AnomalyDetector identifies unusual job behavior patterns
type AnomalyDetector struct {
	logger *slog.Logger
}

// NewAnomalyDetector creates a new anomaly detector
func NewAnomalyDetector(logger *slog.Logger) *AnomalyDetector {
	return &AnomalyDetector{
		logger: logger,
	}
}

// JobExecutionMetric represents metrics from a job execution
type JobExecutionMetric struct {
	JobID        uuid.UUID `json:"job_id"`
	JobName      string    `json:"job_name"`
	ExecutionID  uuid.UUID `json:"execution_id"`
	Timestamp    time.Time `json:"timestamp"`
	DurationMS   int64     `json:"duration_ms"`
	RowsAffected int64     `json:"rows_affected,omitempty"`
	MemoryUsedMB float64   `json:"memory_used_mb,omitempty"`
	CPUPercent   float64   `json:"cpu_percent,omitempty"`
	NetworkIOMB  float64   `json:"network_io_mb,omitempty"`
	RetryCount   int       `json:"retry_count"`
	Success      bool      `json:"success"`
}

// Anomaly represents a detected anomaly
type Anomaly struct {
	ID             string                 `json:"id"`
	JobID          uuid.UUID              `json:"job_id"`
	JobName        string                 `json:"job_name"`
	ExecutionID    uuid.UUID              `json:"execution_id"`
	DetectedAt     time.Time              `json:"detected_at"`
	AnomalyType    string                 `json:"anomaly_type"` // duration, resource, pattern, schedule
	Severity       string                 `json:"severity"`     // low, medium, high, critical
	Score          float64                `json:"score"`        // 0-1, higher = more anomalous
	Metric         string                 `json:"metric"`
	ExpectedValue  float64                `json:"expected_value"`
	ActualValue    float64                `json:"actual_value"`
	Deviation      float64                `json:"deviation"` // in standard deviations
	Description    string                 `json:"description"`
	RootCause      string                 `json:"root_cause,omitempty"`
	Recommendation string                 `json:"recommendation,omitempty"`
	Context        map[string]interface{} `json:"context,omitempty"`
}

// AnomalyReport summarizes detected anomalies
type AnomalyReport struct {
	GeneratedAt       time.Time            `json:"generated_at"`
	TimeRange         TimeRange            `json:"time_range"`
	TotalExecutions   int                  `json:"total_executions"`
	AnomaliesDetected int                  `json:"anomalies_detected"`
	AnomalyRate       float64              `json:"anomaly_rate"`
	Anomalies         []Anomaly            `json:"anomalies"`
	JobHealthSummary  map[string]JobHealth `json:"job_health_summary"`
	Patterns          []AnomalyPattern     `json:"patterns"`
}

// JobHealth shows health status for a job
type JobHealth struct {
	JobID          uuid.UUID `json:"job_id"`
	JobName        string    `json:"job_name"`
	HealthScore    float64   `json:"health_score"` // 0-100
	AnomalyCount   int       `json:"anomaly_count"`
	TrendDirection string    `json:"trend_direction"`
}

// AnomalyPattern describes a recurring anomaly pattern
type AnomalyPattern struct {
	Description  string   `json:"description"`
	Frequency    string   `json:"frequency"`
	AffectedJobs []string `json:"affected_jobs"`
	Correlation  string   `json:"correlation,omitempty"` // What seems to trigger it
}

// BaselineStats holds baseline statistics for a metric
type BaselineStats struct {
	Mean   float64
	StdDev float64
	Min    float64
	Max    float64
	P95    float64
	Count  int
}

// DetectAnomalies analyzes executions for anomalous behavior
func (d *AnomalyDetector) DetectAnomalies(
	ctx context.Context,
	metrics []JobExecutionMetric,
	sensitivityLevel float64, // 1.0 = normal, higher = more sensitive
) (*AnomalyReport, error) {
	d.logger.Info("Detecting anomalies",
		"execution_count", len(metrics),
		"sensitivity", sensitivityLevel,
	)

	if len(metrics) < 20 {
		return nil, fmt.Errorf("insufficient data for anomaly detection (min 20, got %d)", len(metrics))
	}

	// Group metrics by job
	byJob := make(map[uuid.UUID][]JobExecutionMetric)
	for _, m := range metrics {
		byJob[m.JobID] = append(byJob[m.JobID], m)
	}

	var allAnomalies []Anomaly
	jobHealth := make(map[string]JobHealth)

	// Analyze each job
	for jobID, jobMetrics := range byJob {
		if len(jobMetrics) < 10 {
			continue // Not enough data for this job
		}

		// Sort by timestamp
		sort.Slice(jobMetrics, func(i, j int) bool {
			return jobMetrics[i].Timestamp.Before(jobMetrics[j].Timestamp)
		})

		// Calculate baselines
		durationBaseline := d.calculateBaseline(extractDurations(jobMetrics))
		// memoryBaseline := d.calculateBaseline(extractMemory(jobMetrics))

		// Detect anomalies for recent executions
		recentWindow := len(jobMetrics) / 4
		if recentWindow < 5 {
			recentWindow = 5
		}
		recent := jobMetrics[len(jobMetrics)-recentWindow:]

		anomalyCount := 0
		for _, m := range recent {
			// Duration anomaly
			if anomaly := d.checkDurationAnomaly(m, durationBaseline, sensitivityLevel); anomaly != nil {
				allAnomalies = append(allAnomalies, *anomaly)
				anomalyCount++
			}

			// Pattern anomaly (sudden change from recent behavior)
			if anomaly := d.checkPatternAnomaly(m, jobMetrics, sensitivityLevel); anomaly != nil {
				allAnomalies = append(allAnomalies, *anomaly)
				anomalyCount++
			}

			// Schedule anomaly (unusual execution time)
			if anomaly := d.checkScheduleAnomaly(m, jobMetrics); anomaly != nil {
				allAnomalies = append(allAnomalies, *anomaly)
				anomalyCount++
			}
		}

		// Calculate job health
		healthScore := 100.0 - (float64(anomalyCount) / float64(len(recent)) * 50)
		if healthScore < 0 {
			healthScore = 0
		}

		jobHealth[jobID.String()] = JobHealth{
			JobID:          jobID,
			JobName:        jobMetrics[0].JobName,
			HealthScore:    healthScore,
			AnomalyCount:   anomalyCount,
			TrendDirection: d.calculateTrendDirection(jobMetrics),
		}
	}

	// Detect cross-job patterns
	patterns := d.detectPatterns(allAnomalies)

	// Sort anomalies by severity
	sort.Slice(allAnomalies, func(i, j int) bool {
		return anomalySeverityRank(allAnomalies[i].Severity) > anomalySeverityRank(allAnomalies[j].Severity)
	})

	report := &AnomalyReport{
		GeneratedAt:       time.Now(),
		TotalExecutions:   len(metrics),
		AnomaliesDetected: len(allAnomalies),
		AnomalyRate:       float64(len(allAnomalies)) / float64(len(metrics)),
		Anomalies:         allAnomalies,
		JobHealthSummary:  jobHealth,
		Patterns:          patterns,
		TimeRange: TimeRange{
			From: d.findEarliestTime(metrics),
			To:   d.findLatestTime(metrics),
		},
	}

	d.logger.Info("Anomaly detection complete",
		"anomalies", len(allAnomalies),
		"anomaly_rate", fmt.Sprintf("%.2f%%", report.AnomalyRate*100),
	)

	return report, nil
}

// calculateBaseline computes statistical baseline
func (d *AnomalyDetector) calculateBaseline(values []float64) BaselineStats {
	if len(values) == 0 {
		return BaselineStats{}
	}

	// Calculate mean
	var sum float64
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))

	// Calculate standard deviation
	var sumSq float64
	for _, v := range values {
		diff := v - mean
		sumSq += diff * diff
	}
	stdDev := math.Sqrt(sumSq / float64(len(values)))

	// Find min/max and P95
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	p95Index := int(float64(len(sorted)) * 0.95)
	if p95Index >= len(sorted) {
		p95Index = len(sorted) - 1
	}

	return BaselineStats{
		Mean:   mean,
		StdDev: stdDev,
		Min:    sorted[0],
		Max:    sorted[len(sorted)-1],
		P95:    sorted[p95Index],
		Count:  len(values),
	}
}

// checkDurationAnomaly checks if execution duration is anomalous
func (d *AnomalyDetector) checkDurationAnomaly(m JobExecutionMetric, baseline BaselineStats, sensitivity float64) *Anomaly {
	if baseline.StdDev == 0 {
		return nil
	}

	deviation := (float64(m.DurationMS) - baseline.Mean) / baseline.StdDev
	threshold := 3.0 / sensitivity // 3 sigma by default

	if math.Abs(deviation) < threshold {
		return nil
	}

	severity := "low"
	if math.Abs(deviation) > threshold*1.5 {
		severity = "medium"
	}
	if math.Abs(deviation) > threshold*2 {
		severity = "high"
	}
	if math.Abs(deviation) > threshold*3 {
		severity = "critical"
	}

	direction := "slower"
	if deviation < 0 {
		direction = "faster"
	}

	return &Anomaly{
		ID:             uuid.NewString()[:8],
		JobID:          m.JobID,
		JobName:        m.JobName,
		ExecutionID:    m.ExecutionID,
		DetectedAt:     time.Now(),
		AnomalyType:    "duration",
		Severity:       severity,
		Score:          math.Min(math.Abs(deviation)/10, 1.0),
		Metric:         "duration_ms",
		ExpectedValue:  baseline.Mean,
		ActualValue:    float64(m.DurationMS),
		Deviation:      deviation,
		Description:    fmt.Sprintf("Execution was %.1fx %s than usual (%.1fσ)", math.Abs(deviation), direction, deviation),
		Recommendation: d.getDurationRecommendation(deviation),
	}
}

// checkPatternAnomaly looks for sudden changes in behavior
func (d *AnomalyDetector) checkPatternAnomaly(m JobExecutionMetric, history []JobExecutionMetric, sensitivity float64) *Anomaly {
	if len(history) < 10 {
		return nil
	}

	// Compare to last 5 executions
	recentWindow := history[len(history)-5:]
	var recentAvg float64
	for _, h := range recentWindow {
		recentAvg += float64(h.DurationMS)
	}
	recentAvg /= float64(len(recentWindow))

	// If current execution differs significantly from recent pattern
	changeRatio := float64(m.DurationMS) / recentAvg
	threshold := 2.0 / sensitivity

	if changeRatio < threshold && 1/changeRatio < threshold {
		return nil
	}

	return &Anomaly{
		ID:             uuid.NewString()[:8],
		JobID:          m.JobID,
		JobName:        m.JobName,
		ExecutionID:    m.ExecutionID,
		DetectedAt:     time.Now(),
		AnomalyType:    "pattern",
		Severity:       "medium",
		Score:          math.Min(math.Abs(changeRatio-1)/5, 1.0),
		Metric:         "pattern_change",
		ExpectedValue:  recentAvg,
		ActualValue:    float64(m.DurationMS),
		Deviation:      changeRatio,
		Description:    fmt.Sprintf("Sudden %.0f%% change from recent pattern", (changeRatio-1)*100),
		Recommendation: "Investigate recent changes to data volume or dependencies",
	}
}

// checkScheduleAnomaly looks for unusual execution times
func (d *AnomalyDetector) checkScheduleAnomaly(m JobExecutionMetric, history []JobExecutionMetric) *Anomaly {
	if len(history) < 10 {
		return nil
	}

	// Count typical execution hours
	hourCount := make(map[int]int)
	for _, h := range history {
		hourCount[h.Timestamp.Hour()]++
	}

	// Find if current hour is unusual
	currentHour := m.Timestamp.Hour()
	if hourCount[currentHour] > len(history)/10 {
		return nil // This hour is typical
	}

	// Check if ANY executions happened at this hour before
	if hourCount[currentHour] > 0 {
		return nil
	}

	return &Anomaly{
		ID:             uuid.NewString()[:8],
		JobID:          m.JobID,
		JobName:        m.JobName,
		ExecutionID:    m.ExecutionID,
		DetectedAt:     time.Now(),
		AnomalyType:    "schedule",
		Severity:       "low",
		Score:          0.3,
		Metric:         "execution_hour",
		ExpectedValue:  float64(d.findTypicalHour(hourCount)),
		ActualValue:    float64(currentHour),
		Deviation:      0,
		Description:    fmt.Sprintf("Unusual execution at %d:00 - not seen in history", currentHour),
		Recommendation: "Verify this was an intentional schedule change or manual trigger",
	}
}

// getDurationRecommendation provides recommendations for duration anomalies
func (d *AnomalyDetector) getDurationRecommendation(deviation float64) string {
	if deviation > 0 {
		return "Investigate slow downstream dependencies, increased data volume, or resource contention"
	}
	return "Unusually fast execution may indicate missing data, early termination, or caching"
}

// detectPatterns finds recurring anomaly patterns
func (d *AnomalyDetector) detectPatterns(anomalies []Anomaly) []AnomalyPattern {
	var patterns []AnomalyPattern

	// Group by anomaly type
	byType := make(map[string][]Anomaly)
	for _, a := range anomalies {
		byType[a.AnomalyType] = append(byType[a.AnomalyType], a)
	}

	for aType, typeAnomalies := range byType {
		if len(typeAnomalies) < 3 {
			continue
		}

		// Find affected jobs
		jobSet := make(map[string]bool)
		for _, a := range typeAnomalies {
			jobSet[a.JobName] = true
		}
		var jobs []string
		for j := range jobSet {
			jobs = append(jobs, j)
		}

		patterns = append(patterns, AnomalyPattern{
			Description:  fmt.Sprintf("Recurring %s anomalies", aType),
			Frequency:    fmt.Sprintf("%d occurrences", len(typeAnomalies)),
			AffectedJobs: jobs,
		})
	}

	return patterns
}

// calculateTrendDirection determines if job health is improving or degrading
func (d *AnomalyDetector) calculateTrendDirection(metrics []JobExecutionMetric) string {
	if len(metrics) < 10 {
		return "stable"
	}

	// Compare first half success rate to second half
	mid := len(metrics) / 2
	firstHalf := metrics[:mid]
	secondHalf := metrics[mid:]

	firstSuccess := countSuccesses(firstHalf)
	secondSuccess := countSuccesses(secondHalf)

	firstRate := float64(firstSuccess) / float64(len(firstHalf))
	secondRate := float64(secondSuccess) / float64(len(secondHalf))

	if secondRate > firstRate+0.05 {
		return "improving"
	}
	if secondRate < firstRate-0.05 {
		return "degrading"
	}
	return "stable"
}

// Helper functions
func extractDurations(metrics []JobExecutionMetric) []float64 {
	result := make([]float64, len(metrics))
	for i, m := range metrics {
		result[i] = float64(m.DurationMS)
	}
	return result
}

func countSuccesses(metrics []JobExecutionMetric) int {
	count := 0
	for _, m := range metrics {
		if m.Success {
			count++
		}
	}
	return count
}

func (d *AnomalyDetector) findEarliestTime(metrics []JobExecutionMetric) time.Time {
	if len(metrics) == 0 {
		return time.Time{}
	}
	earliest := metrics[0].Timestamp
	for _, m := range metrics {
		if m.Timestamp.Before(earliest) {
			earliest = m.Timestamp
		}
	}
	return earliest
}

func (d *AnomalyDetector) findLatestTime(metrics []JobExecutionMetric) time.Time {
	if len(metrics) == 0 {
		return time.Time{}
	}
	latest := metrics[0].Timestamp
	for _, m := range metrics {
		if m.Timestamp.After(latest) {
			latest = m.Timestamp
		}
	}
	return latest
}

func (d *AnomalyDetector) findTypicalHour(hourCount map[int]int) int {
	maxHour := 0
	maxCount := 0
	for h, c := range hourCount {
		if c > maxCount {
			maxCount = c
			maxHour = h
		}
	}
	return maxHour
}

func anomalySeverityRank(severity string) int {
	switch severity {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}
