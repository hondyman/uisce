package finops

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ForecastResult holds the projected compute demand for a specific 24-hour window.
type ForecastResult struct {
	TenantID            uuid.UUID `json:"tenantId"`
	WindowStart         time.Time `json:"windowStart"`
	WindowEnd           time.Time `json:"windowEnd"`
	ProjectedBytes      int64     `json:"projectedBytes"`
	ProjectedCPUms      int64     `json:"projectedCpuMs"`
	ProjectedCostUSD    float64   `json:"projectedCostUsd"`
	ConfidenceScore     float64   `json:"confidenceScore"`
	PeakProbability     float64   `json:"peakProbability"`
	ContributingFactors []string  `json:"contributingFactors"`
	// CalibrationFactor is the rolling correction multiplier derived from past
	// feedback. 1.0 = neutral; >1.0 = model was under-predicting; <1.0 = over-predicting.
	CalibrationFactor float64 `json:"calibrationFactor"`
	// CalibrationSamples is the number of feedback records used to compute the factor.
	CalibrationSamples int `json:"calibrationSamples"`
	// ProjectedMultiplier is the composite calendar & burst multiplier applied to baseline.
	ProjectedMultiplier float64 `json:"projectedMultiplier"`
}

// DemandForecaster synthesises historical telemetry and exchange-calendar signals
// to predict compute demand for a given tenant and target date.
// When a FeedbackService is wired in, it applies the rolling calibration factor
// derived from past forecast outcomes to self-correct future projections.
type DemandForecaster struct {
	db              *sqlx.DB
	feedbackService *ForecastFeedbackService
	clock           Clock
}

// NewDemandForecaster constructs a DemandForecaster with the production RealClock.
// feedbackSvc may be nil — in that case no calibration is applied (factor = 1.0).
// db may be nil in unit-test contexts; queries are skipped and defaults are used.
func NewDemandForecaster(db *sqlx.DB) *DemandForecaster {
	return NewDemandForecasterWithClock(db, RealClock{})
}

// NewDemandForecasterWithClock constructs a DemandForecaster using the supplied
// clock. Used by tests to pin "today" to a deterministic date, and by callers
// that need to share a single clock across multiple components.
func NewDemandForecasterWithClock(db *sqlx.DB, clock Clock) *DemandForecaster {
	if clock == nil {
		clock = RealClock{}
	}
	return &DemandForecaster{
		db:              db,
		feedbackService: NewForecastFeedbackService(db),
		clock:           clock,
	}
}

// GenerateTenantDemandForecast predicts compute requirements for targetDate (24-hour window).
//
// Algorithm:
//  1. Pull 60-day historical DOW-matched baseline from audit.analytical_query_execution_logs.
//  2. Apply calendar multipliers (month-end × 2.8, quarter-end × 4.5).
//  3. Add burst-report schedule contribution (×0.4 per active schedule on month-end).
//  4. Derive peak probability = min(1.0, (multiplier-1) / 3.5).
func (f *DemandForecaster) GenerateTenantDemandForecast(
	ctx context.Context,
	tenantID uuid.UUID,
	targetDate time.Time,
) (*ForecastResult, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	// Midnight UTC boundaries eliminate DST / time-of-day boundary drift.
	utc := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, time.UTC)
	windowStart := utc
	windowEnd := utc.Add(24 * time.Hour)

	// ── Step 1: Historical DOW-matched baseline ──────────────────────────────
	var baseline struct {
		AvgBytes    int64   `db:"avg_bytes"`
		AvgCPUms    int64   `db:"avg_cpums"`
		AvgCost     float64 `db:"avg_cost"`
		SampleCount int64   `db:"sample_count"`
	}

	if f.db != nil {
		baselineQuery := `
			SELECT
				COALESCE(AVG(scanned_bytes)::BIGINT,  0) AS avg_bytes,
				COALESCE(AVG(cpu_duration_ms)::BIGINT, 0) AS avg_cpums,
				COALESCE(AVG(attributed_cost_usd),    0.0) AS avg_cost,
				COUNT(1)                                   AS sample_count
			FROM audit.analytical_query_execution_logs
			WHERE tenant_id = $1
			  AND created_at >= NOW() - INTERVAL '60 days'
			  AND EXTRACT(DOW FROM created_at) = EXTRACT(DOW FROM $2::TIMESTAMPTZ);
		`
		if err := f.db.GetContext(ctx, &baseline, baselineQuery, tenantID, targetDate); err != nil {
			return nil, fmt.Errorf("failed fetching baseline telemetry: %w", err)
		}
	} else {
		// Safe defaults for test/sandbox contexts with no DB.
		baseline.AvgBytes = 500_000_000  // 500 MB
		baseline.AvgCPUms = 30_000       // 30 s
		baseline.AvgCost = 0.85
		baseline.SampleCount = 42
	}

	// ── Step 2: Calendar multipliers ────────────────────────────────────────
	var calendarEvents []string
	multiplier := 1.0

	switch {
	case isQuarterEndWindow(targetDate):
		multiplier *= 4.5
		calendarEvents = append(calendarEvents, "CALENDAR_QUARTER_END")
	case isMonthEndWindow(targetDate):
		multiplier *= 2.8
		calendarEvents = append(calendarEvents, "CALENDAR_MONTH_END")
	}

	// ── Step 3: Active burst-report schedule contribution ────────────────────
	if f.db != nil && isMonthEndWindow(targetDate) {
		var burstCount int
		countQuery := `
			SELECT COUNT(1)
			FROM report_schedules
			WHERE tenant_id = $1 AND is_active = TRUE;
		`
		// Non-fatal: missing table / no rows means no burst contribution.
		if err := f.db.GetContext(ctx, &burstCount, countQuery, tenantID); err == nil && burstCount > 0 {
			multiplier += float64(burstCount) * 0.4
			calendarEvents = append(calendarEvents, fmt.Sprintf("BATCH_REPORT_BURST(%d)", burstCount))
		}
	}

	// ── Step 4: Apply rolling calibration factor (feedback loop) ─────────────
	// The calibration factor is the average of (actual_cost / projected_cost) over the
	// last N ACCURATE/PARTIAL_SPIKE outcomes for this tenant. It self-corrects the
	// model when it has been consistently over- or under-predicting.
	calibrationFactor := 1.0
	calibrationSamples := 0
	if f.feedbackService != nil {
		calibrationFactor, calibrationSamples, _ = f.feedbackService.GetCalibrationFactor(ctx, tenantID)
	}

	// ── Step 5: Derive calibrated projections ─────────────────────────────────
	projectedBytes := int64(float64(baseline.AvgBytes) * multiplier * calibrationFactor)
	projectedCPUms := int64(float64(baseline.AvgCPUms) * multiplier * calibrationFactor)
	projectedCost := baseline.AvgCost * multiplier * calibrationFactor

	// Peak probability saturates at 1.0; a 3.5× baseline day = certainty.
	peakProb := math.Min(1.0, (multiplier-1.0)/3.5)

	// Confidence dynamically scales with sample count and calendar signal.
	// In the absence of query history (e.g. nil DB / cold tenant), defaults are 0.65 baseline, 0.88 with calendar signal.
	baseConf := 0.65
	if f.db != nil {
		baseConf = 0.40 + math.Min(0.35, float64(baseline.SampleCount)/100.0*0.35)
	}
	if len(calendarEvents) > 0 {
		baseConf = math.Max(baseConf, 0.88)
	}
	// Calibration data boosts confidence: each sample adds a small increment (max +0.10).
	if calibrationSamples > 0 {
		calibBoost := math.Min(0.10, float64(calibrationSamples)/float64(calibrationWindowSize)*0.10)
		baseConf = math.Min(0.99, baseConf+calibBoost)
	}

	return &ForecastResult{
		TenantID:            tenantID,
		WindowStart:         windowStart,
		WindowEnd:           windowEnd,
		ProjectedBytes:      projectedBytes,
		ProjectedCPUms:      projectedCPUms,
		ProjectedCostUSD:    projectedCost,
		ConfidenceScore:     baseConf,
		PeakProbability:     peakProb,
		CalibrationFactor:   calibrationFactor,
		CalibrationSamples:  calibrationSamples,
		ProjectedMultiplier: multiplier,
		ContributingFactors: calendarEvents,
	}, nil
}

// PersistForecast upserts the forecast result into finops.compute_demand_forecasts.
// Existing rows for the same (tenant_id, forecast_window_start) are overwritten.
func (f *DemandForecaster) PersistForecast(ctx context.Context, r *ForecastResult) error {
	if f.db == nil || r == nil {
		return nil
	}
	factorsJSON, err := json.Marshal(r.ContributingFactors)
	if err != nil {
		return fmt.Errorf("failed marshalling contributing_factors: %w", err)
	}

	_, err = f.db.ExecContext(ctx, `
		INSERT INTO finops.compute_demand_forecasts (
			tenant_id, forecast_window_start, forecast_window_end,
			projected_scanned_bytes, projected_cpu_duration_ms, projected_cost_usd,
			confidence_score, peak_probability, contributing_factors, generated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		ON CONFLICT (tenant_id, forecast_window_start)
		DO UPDATE SET
			forecast_window_end       = EXCLUDED.forecast_window_end,
			projected_scanned_bytes   = EXCLUDED.projected_scanned_bytes,
			projected_cpu_duration_ms = EXCLUDED.projected_cpu_duration_ms,
			projected_cost_usd        = EXCLUDED.projected_cost_usd,
			confidence_score          = EXCLUDED.confidence_score,
			peak_probability          = EXCLUDED.peak_probability,
			contributing_factors      = EXCLUDED.contributing_factors,
			generated_at              = NOW();
	`,
		r.TenantID,
		r.WindowStart,
		r.WindowEnd,
		r.ProjectedBytes,
		r.ProjectedCPUms,
		r.ProjectedCostUSD,
		r.ConfidenceScore,
		r.PeakProbability,
		factorsJSON,
	)
	if err != nil {
		return fmt.Errorf("failed persisting demand forecast: %w", err)
	}
	return nil
}

// ── Calendar helpers ─────────────────────────────────────────────────────────

// isMonthEndWindow returns true when targetDate falls within the last 2 days of the month
// or the first 2 days of the subsequent month (covering month-end close and post-close reconciliations).
func isMonthEndWindow(t time.Time) bool {
	utc := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	// First 2 days of month (day 1, 2)
	if utc.Day() <= 2 {
		return true
	}
	// Last 2 days of month: 2 days ahead crosses to next month
	return utc.AddDate(0, 0, 2).Month() != utc.Month()
}

// isQuarterEndWindow returns true when targetDate is a month-end window of a fiscal quarter close
// (March, June, September, or December, or the first 2 days of April, July, October, or January for quarter-close reconciliation).
func isQuarterEndWindow(t time.Time) bool {
	if !isMonthEndWindow(t) {
		return false
	}
	utc := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	m := utc.Month()
	// Quarter-end close days in the quarter itself:
	if m == time.March || m == time.June || m == time.September || m == time.December {
		return true
	}
	// First 2 days of the subsequent month (quarter-close reconciliation and NAV finalize):
	if utc.Day() <= 2 && (m == time.April || m == time.July || m == time.October || m == time.January) {
		return true
	}
	return false
}
