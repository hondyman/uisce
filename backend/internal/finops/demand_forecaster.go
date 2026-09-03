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
	CalibrationFactor  float64 `json:"calibrationFactor"`
	// CalibrationSamples is the number of feedback records used to compute the factor.
	CalibrationSamples int     `json:"calibrationSamples"`
}

// DemandForecaster synthesises historical telemetry and exchange-calendar signals
// to predict compute demand for a given tenant and target date.
// When a FeedbackService is wired in, it applies the rolling calibration factor
// derived from past forecast outcomes to self-correct future projections.
type DemandForecaster struct {
	db              *sqlx.DB
	feedbackService *ForecastFeedbackService
}

// NewDemandForecaster constructs a DemandForecaster.
// feedbackSvc may be nil — in that case no calibration is applied (factor = 1.0).
// db may be nil in unit-test contexts; queries are skipped and defaults are used.
func NewDemandForecaster(db *sqlx.DB) *DemandForecaster {
	return &DemandForecaster{
		db:              db,
		feedbackService: NewForecastFeedbackService(db),
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

	loc := targetDate.Location()
	windowStart := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, loc)
	windowEnd := windowStart.Add(24 * time.Hour)

	// ── Step 1: Historical DOW-matched baseline ──────────────────────────────
	var baseline struct {
		AvgBytes int64   `db:"avg_bytes"`
		AvgCPUms int64   `db:"avg_cpums"`
		AvgCost  float64 `db:"avg_cost"`
	}

	if f.db != nil {
		baselineQuery := `
			SELECT
				COALESCE(AVG(scanned_bytes)::BIGINT,  0) AS avg_bytes,
				COALESCE(AVG(cpu_duration_ms)::BIGINT, 0) AS avg_cpums,
				COALESCE(AVG(attributed_cost_usd),    0.0) AS avg_cost
			FROM audit.analytical_query_execution_logs
			WHERE tenant_id = $1
			  AND executed_at >= NOW() - INTERVAL '60 days'
			  AND EXTRACT(DOW FROM executed_at) = EXTRACT(DOW FROM $2::TIMESTAMPTZ);
		`
		if err := f.db.GetContext(ctx, &baseline, baselineQuery, tenantID, targetDate); err != nil {
			return nil, fmt.Errorf("failed fetching baseline telemetry: %w", err)
		}
	} else {
		// Safe defaults for test/sandbox contexts with no DB.
		baseline.AvgBytes = 500_000_000  // 500 MB
		baseline.AvgCPUms = 30_000       // 30 s
		baseline.AvgCost = 0.85
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

	// Confidence grows when we have calendar signal AND calibration samples.
	confidence := 0.65
	if len(calendarEvents) > 0 {
		confidence = 0.88
	}
	// Calibration data boosts confidence: each sample adds a small increment (max +0.10).
	if calibrationSamples > 0 {
		calibBoost := math.Min(0.10, float64(calibrationSamples)/float64(calibrationWindowSize)*0.10)
		confidence = math.Min(0.99, confidence+calibBoost)
	}

	return &ForecastResult{
		TenantID:            tenantID,
		WindowStart:         windowStart,
		WindowEnd:           windowEnd,
		ProjectedBytes:      projectedBytes,
		ProjectedCPUms:      projectedCPUms,
		ProjectedCostUSD:    projectedCost,
		ConfidenceScore:     confidence,
		PeakProbability:     peakProb,
		CalibrationFactor:   calibrationFactor,
		CalibrationSamples:  calibrationSamples,
		ContributingFactors: calendarEvents,
	}, nil
}

// PersistForecast upserts the forecast result into finops.compute_demand_forecasts.
// Existing rows for the same (tenant_id, forecast_window_start) are overwritten.
func (f *DemandForecaster) PersistForecast(ctx context.Context, r *ForecastResult) error {
	if f.db == nil {
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

// isMonthEndWindow returns true when targetDate falls within 2 calendar days of the month boundary.
func isMonthEndWindow(t time.Time) bool {
	nextDay := t.Add(48 * time.Hour)
	return nextDay.Month() != t.Month()
}

// isQuarterEndWindow returns true when targetDate is a month-end of a fiscal quarter close
// (March, June, September, or December).
func isQuarterEndWindow(t time.Time) bool {
	if !isMonthEndWindow(t) {
		return false
	}
	m := t.Month()
	return m == time.March || m == time.June || m == time.September || m == time.December
}
