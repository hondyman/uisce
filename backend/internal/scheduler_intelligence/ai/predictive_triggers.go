package ai

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"
)

// PredictiveTriggerEngine predicts optimal job execution times based on patterns
type PredictiveTriggerEngine struct {
	logger *slog.Logger
}

// NewPredictiveTriggerEngine creates a new predictive trigger engine
func NewPredictiveTriggerEngine(logger *slog.Logger) *PredictiveTriggerEngine {
	return &PredictiveTriggerEngine{
		logger: logger,
	}
}

// PredictionRequest contains data for trigger prediction
type PredictionRequest struct {
	TenantID       uuid.UUID       `json:"tenant_id"`
	JobID          uuid.UUID       `json:"job_id"`
	JobName        string          `json:"job_name"`
	HistoricalRuns []HistoricalRun `json:"historical_runs"`
	SLOTargetMS    int64           `json:"slo_target_ms,omitempty"`
	SLOCritical    bool            `json:"slo_critical"`
	DataSources    []DataSource    `json:"data_sources,omitempty"`
	LookaheadHours int             `json:"lookahead_hours"`
}

// HistoricalRun represents past execution data
type HistoricalRun struct {
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	DurationMS  int64     `json:"duration_ms"`
	Success     bool      `json:"success"`
	SLOBreached bool      `json:"slo_breached"`
	DataArrival time.Time `json:"data_arrival,omitempty"`
}

// DataSource represents an upstream data dependency
type DataSource struct {
	Name               string    `json:"name"`
	AvgArrivalTime     time.Time `json:"avg_arrival_time"`
	ArrivalVarianceMin int       `json:"arrival_variance_minutes"`
}

// TriggerPrediction contains the predicted optimal trigger time
type TriggerPrediction struct {
	RecommendedTime   time.Time  `json:"recommended_time"`
	EarliestSafeTime  time.Time  `json:"earliest_safe_time"`
	LatestSafeTime    time.Time  `json:"latest_safe_time"`
	Confidence        float64    `json:"confidence"`
	SLORiskLevel      string     `json:"slo_risk_level"` // low, medium, high
	Reasoning         string     `json:"reasoning"`
	TriggerType       string     `json:"trigger_type"` // time, data_arrival, capacity
	DataReadyEstimate *time.Time `json:"data_ready_estimate,omitempty"`
}

// PredictOptimalTrigger determines the best time to trigger a job
func (p *PredictiveTriggerEngine) PredictOptimalTrigger(ctx context.Context, req PredictionRequest) (*TriggerPrediction, error) {
	p.logger.Info("Predicting optimal trigger time",
		"job_id", req.JobID,
		"job_name", req.JobName,
		"historical_runs", len(req.HistoricalRuns),
	)

	// Analyze historical patterns
	stats := p.analyzeHistoricalRuns(req.HistoricalRuns)

	// Calculate data arrival prediction if applicable
	var dataReady *time.Time
	if len(req.DataSources) > 0 {
		dr := p.predictDataArrival(req.DataSources)
		dataReady = &dr
	}

	// Calculate SLO-aware timing
	now := time.Now()
	recommendation := p.calculateOptimalTime(now, stats, dataReady, req)

	prediction := &TriggerPrediction{
		RecommendedTime:   recommendation.optimal,
		EarliestSafeTime:  recommendation.earliest,
		LatestSafeTime:    recommendation.latest,
		Confidence:        recommendation.confidence,
		SLORiskLevel:      p.assessSLORisk(recommendation, req),
		Reasoning:         recommendation.reasoning,
		TriggerType:       recommendation.triggerType,
		DataReadyEstimate: dataReady,
	}

	p.logger.Info("Prediction complete",
		"recommended_time", prediction.RecommendedTime,
		"confidence", prediction.Confidence,
		"slo_risk", prediction.SLORiskLevel,
	)

	return prediction, nil
}

// RunStats contains statistics from historical runs
type RunStats struct {
	AvgDurationMS    float64
	StdDevDurationMS float64
	SuccessRate      float64
	SLOBreachRate    float64
	TypicalStartHour int
	RunCount         int
}

// analyzeHistoricalRuns computes statistics from past runs
func (p *PredictiveTriggerEngine) analyzeHistoricalRuns(runs []HistoricalRun) *RunStats {
	if len(runs) == 0 {
		return &RunStats{
			AvgDurationMS:    300000, // Default 5 min
			SuccessRate:      0.95,
			TypicalStartHour: 2, // Default 2 AM
		}
	}

	stats := &RunStats{RunCount: len(runs)}

	var totalDuration float64
	var successCount, sloBreachCount int
	hourCounts := make(map[int]int)

	for _, run := range runs {
		totalDuration += float64(run.DurationMS)
		if run.Success {
			successCount++
		}
		if run.SLOBreached {
			sloBreachCount++
		}
		hourCounts[run.StartTime.Hour()]++
	}

	stats.AvgDurationMS = totalDuration / float64(len(runs))
	stats.SuccessRate = float64(successCount) / float64(len(runs))
	stats.SLOBreachRate = float64(sloBreachCount) / float64(len(runs))

	// Find most common start hour
	maxCount := 0
	for hour, count := range hourCounts {
		if count > maxCount {
			maxCount = count
			stats.TypicalStartHour = hour
		}
	}

	// Calculate standard deviation
	var sumSquares float64
	for _, run := range runs {
		diff := float64(run.DurationMS) - stats.AvgDurationMS
		sumSquares += diff * diff
	}
	stats.StdDevDurationMS = math.Sqrt(sumSquares / float64(len(runs)))

	return stats
}

// predictDataArrival estimates when data will be ready
func (p *PredictiveTriggerEngine) predictDataArrival(sources []DataSource) time.Time {
	if len(sources) == 0 {
		return time.Now()
	}

	// Take the latest source arrival time + variance
	var latest time.Time
	for _, src := range sources {
		arrival := src.AvgArrivalTime.Add(time.Duration(src.ArrivalVarianceMin) * time.Minute)
		if arrival.After(latest) {
			latest = arrival
		}
	}
	return latest
}

// TimeRecommendation contains timing calculations
type TimeRecommendation struct {
	optimal     time.Time
	earliest    time.Time
	latest      time.Time
	confidence  float64
	reasoning   string
	triggerType string
}

// calculateOptimalTime determines the best execution time
func (p *PredictiveTriggerEngine) calculateOptimalTime(
	now time.Time,
	stats *RunStats,
	dataReady *time.Time,
	req PredictionRequest,
) TimeRecommendation {
	rec := TimeRecommendation{
		confidence:  0.7,
		triggerType: "time",
	}

	// Base time on historical pattern
	baseTime := time.Date(now.Year(), now.Month(), now.Day()+1,
		stats.TypicalStartHour, 0, 0, 0, now.Location())

	// Adjust for data readiness
	if dataReady != nil && dataReady.After(baseTime) {
		baseTime = dataReady.Add(5 * time.Minute) // 5 min buffer
		rec.triggerType = "data_arrival"
		rec.reasoning = "Trigger after data arrival with safety buffer"
	}

	// For SLO-critical jobs, add buffer
	if req.SLOCritical {
		bufferMS := stats.StdDevDurationMS * 2
		baseTime = baseTime.Add(-time.Duration(bufferMS) * time.Millisecond)
		rec.reasoning = "Moved earlier for SLO buffer"
		rec.confidence += 0.1
	}

	rec.optimal = baseTime
	rec.earliest = baseTime.Add(-30 * time.Minute)
	rec.latest = baseTime.Add(30 * time.Minute)

	if rec.reasoning == "" {
		rec.reasoning = fmt.Sprintf("Based on historical pattern (typical start: %d:00)", stats.TypicalStartHour)
	}

	// Adjust confidence based on data quality
	if stats.RunCount > 50 {
		rec.confidence = 0.9
	} else if stats.RunCount < 10 {
		rec.confidence = 0.6
	}

	return rec
}

// assessSLORisk evaluates the SLO risk of the predicted time
func (p *PredictiveTriggerEngine) assessSLORisk(rec TimeRecommendation, req PredictionRequest) string {
	if !req.SLOCritical {
		return "low"
	}

	// Check historical SLO breach rate
	sloBreaches := 0
	for _, run := range req.HistoricalRuns {
		if run.SLOBreached {
			sloBreaches++
		}
	}

	breachRate := float64(sloBreaches) / float64(len(req.HistoricalRuns)+1)

	if breachRate > 0.1 {
		return "high"
	}
	if breachRate > 0.05 {
		return "medium"
	}
	return "low"
}

// PredictCapacityWindow finds optimal execution windows based on capacity
func (p *PredictiveTriggerEngine) PredictCapacityWindow(
	ctx context.Context,
	tenantID uuid.UUID,
	jobDurationMS int64,
	lookaheadHours int,
) ([]TimeWindow, error) {
	// In real implementation, would query resource metrics
	// For now, return optimal off-peak windows

	now := time.Now()
	var windows []TimeWindow

	for h := 0; h < lookaheadHours; h++ {
		t := now.Add(time.Duration(h) * time.Hour)
		hour := t.Hour()

		// Off-peak hours (1-5 AM) are best
		if hour >= 1 && hour <= 5 {
			windows = append(windows, TimeWindow{
				Start:  t.Truncate(time.Hour),
				End:    t.Truncate(time.Hour).Add(time.Hour),
				Reason: "Off-peak capacity window",
			})
		}
	}

	return windows, nil
}
