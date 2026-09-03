package finops

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ForecastOutcome classifies the accuracy of a demand forecast after the fact.
type ForecastOutcome string

const (
	// OutcomeAccurate — spike happened and cost was within ±25% of the projection.
	OutcomeAccurate ForecastOutcome = "ACCURATE"
	// OutcomeFalsePositive — high peak_probability but no spike materialised.
	OutcomeFalsePositive ForecastOutcome = "FALSE_POSITIVE"
	// OutcomeMissedSpike — low peak_probability but a severe spike occurred.
	OutcomeMissedSpike ForecastOutcome = "MISSED_SPIKE"
	// OutcomePartialSpike — spike occurred but significantly below projection.
	OutcomePartialSpike ForecastOutcome = "PARTIAL_SPIKE"
)

// ForecastFeedback is a single feedback record for a completed forecast window.
type ForecastFeedback struct {
	FeedbackID          uuid.UUID       `db:"feedback_id"           json:"feedbackId"`
	TenantID            uuid.UUID       `db:"tenant_id"             json:"tenantId"`
	ForecastID          uuid.UUID       `db:"forecast_id"           json:"forecastId"`
	Outcome             ForecastOutcome `db:"outcome"               json:"outcome"`
	ActualScannedBytes  *int64          `db:"actual_scanned_bytes"  json:"actualScannedBytes,omitempty"`
	ActualCPUDurationMs *int64          `db:"actual_cpu_duration_ms" json:"actualCpuDurationMs,omitempty"`
	ActualCostUSD       *float64        `db:"actual_cost_usd"       json:"actualCostUsd,omitempty"`
	AccuracyRatio       *float64        `db:"accuracy_ratio"        json:"accuracyRatio,omitempty"`
	Notes               *string         `db:"notes"                 json:"notes,omitempty"`
	RecordedBy          *uuid.UUID      `db:"recorded_by"           json:"recordedBy,omitempty"`
	RecordedAt          time.Time       `db:"recorded_at"           json:"recordedAt"`
}

// SubmitFeedbackRequest carries the operator-supplied outcome for a forecast.
type SubmitFeedbackRequest struct {
	ForecastID          uuid.UUID       `json:"forecastId"`
	Outcome             ForecastOutcome `json:"outcome"`
	ActualScannedBytes  *int64          `json:"actualScannedBytes,omitempty"`
	ActualCPUDurationMs *int64          `json:"actualCpuDurationMs,omitempty"`
	ActualCostUSD       *float64        `json:"actualCostUsd,omitempty"`
	Notes               *string         `json:"notes,omitempty"`
	RecordedBy          *uuid.UUID      `json:"recordedBy,omitempty"`
}

// CalibrationState holds the rolling correction factor for a tenant.
type CalibrationState struct {
	TenantID          uuid.UUID `db:"tenant_id"          json:"tenantId"`
	CalibrationFactor float64   `db:"calibration_factor" json:"calibrationFactor"`
	SampleCount       int       `db:"sample_count"       json:"sampleCount"`
	LastComputedAt    time.Time `db:"last_computed_at"   json:"lastComputedAt"`
}

// calibrationWindowSize is the number of recent feedback records used to
// compute the rolling calibration factor.
const calibrationWindowSize = 30

// ForecastFeedbackService manages the submission of forecast outcomes and
// derives per-tenant calibration factors that self-correct future predictions.
type ForecastFeedbackService struct {
	db *sqlx.DB
}

// NewForecastFeedbackService constructs a ForecastFeedbackService.
func NewForecastFeedbackService(db *sqlx.DB) *ForecastFeedbackService {
	return &ForecastFeedbackService{db: db}
}

// SubmitFeedback records an operator outcome for a previously generated forecast.
// If actual metrics are provided, the computed accuracy_ratio is stored and the
// tenant's calibration state is refreshed immediately.
func (s *ForecastFeedbackService) SubmitFeedback(
	ctx context.Context,
	tenantID uuid.UUID,
	req SubmitFeedbackRequest,
) (*ForecastFeedback, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}
	if req.ForecastID == uuid.Nil {
		return nil, fmt.Errorf("forecast_id is required")
	}
	if !isValidOutcome(req.Outcome) {
		return nil, fmt.Errorf("invalid outcome %q: must be one of ACCURATE, FALSE_POSITIVE, MISSED_SPIKE, PARTIAL_SPIKE", req.Outcome)
	}

	if s.db == nil {
		// Sandbox/test mode — return a synthetic record.
		return &ForecastFeedback{
			FeedbackID: uuid.New(),
			TenantID:   tenantID,
			ForecastID: req.ForecastID,
			Outcome:    req.Outcome,
			RecordedAt: time.Now(),
		}, nil
	}

	var accuracyRatio *float64
	if req.ActualCostUSD != nil && *req.ActualCostUSD > 0 {
		var projectedCost float64
		err := s.db.GetContext(ctx, &projectedCost, `
			SELECT projected_cost_usd FROM finops.compute_demand_forecasts
			WHERE forecast_id = $1 AND tenant_id = $2
		`, req.ForecastID, tenantID)
		if err == nil && projectedCost > 0 {
			ratio := *req.ActualCostUSD / projectedCost
			accuracyRatio = &ratio
		}
	}

	var fb ForecastFeedback
	err := s.db.GetContext(ctx, &fb, `
		INSERT INTO finops.forecast_feedback (
			tenant_id, forecast_id, outcome,
			actual_scanned_bytes, actual_cpu_duration_ms, actual_cost_usd,
			accuracy_ratio, notes, recorded_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (forecast_id)
		DO UPDATE SET
			outcome               = EXCLUDED.outcome,
			actual_scanned_bytes  = EXCLUDED.actual_scanned_bytes,
			actual_cpu_duration_ms = EXCLUDED.actual_cpu_duration_ms,
			actual_cost_usd       = EXCLUDED.actual_cost_usd,
			accuracy_ratio        = EXCLUDED.accuracy_ratio,
			notes                 = EXCLUDED.notes,
			recorded_by           = EXCLUDED.recorded_by,
			recorded_at           = NOW()
		RETURNING
			feedback_id, tenant_id, forecast_id, outcome,
			actual_scanned_bytes, actual_cpu_duration_ms, actual_cost_usd,
			accuracy_ratio, notes, recorded_by, recorded_at;
	`,
		tenantID,
		req.ForecastID,
		req.Outcome,
		req.ActualScannedBytes,
		req.ActualCPUDurationMs,
		req.ActualCostUSD,
		accuracyRatio,
		req.Notes,
		req.RecordedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("failed recording forecast feedback: %w", err)
	}

	// Immediately refresh the calibration state so the next forecast benefits.
	if err := s.refreshCalibration(ctx, tenantID); err != nil {
		// Non-fatal: calibration refresh failure does not invalidate the feedback record.
		_ = err
	}

	return &fb, nil
}

// GetFeedback retrieves the feedback record for a specific forecast, if any.
func (s *ForecastFeedbackService) GetFeedback(
	ctx context.Context,
	tenantID uuid.UUID,
	forecastID uuid.UUID,
) (*ForecastFeedback, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}
	if s.db == nil {
		return nil, nil
	}

	var fb ForecastFeedback
	err := s.db.GetContext(ctx, &fb, `
		SELECT feedback_id, tenant_id, forecast_id, outcome,
		       actual_scanned_bytes, actual_cpu_duration_ms, actual_cost_usd,
		       accuracy_ratio, notes, recorded_by, recorded_at
		FROM   finops.forecast_feedback
		WHERE  tenant_id  = $1
		  AND  forecast_id = $2;
	`, tenantID, forecastID)
	if err != nil {
		return nil, nil // no feedback yet
	}
	return &fb, nil
}

// GetCalibrationFactor returns the current rolling calibration factor for a tenant.
// Returns 1.0 (neutral — no adjustment) when insufficient feedback exists.
func (s *ForecastFeedbackService) GetCalibrationFactor(
	ctx context.Context,
	tenantID uuid.UUID,
) (float64, int, error) {
	if tenantID == uuid.Nil {
		return 1.0, 0, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}
	if s.db == nil {
		return 1.0, 0, nil
	}

	var state CalibrationState
	err := s.db.GetContext(ctx, &state, `
		SELECT tenant_id, calibration_factor, sample_count, last_computed_at
		FROM   finops.forecast_calibration_state
		WHERE  tenant_id = $1;
	`, tenantID)
	if err != nil {
		// No calibration record yet → neutral factor.
		return 1.0, 0, nil
	}
	return state.CalibrationFactor, state.SampleCount, nil
}

// GetCalibrationState returns the full calibration state for a tenant.
func (s *ForecastFeedbackService) GetCalibrationState(
	ctx context.Context,
	tenantID uuid.UUID,
) (*CalibrationState, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}
	if s.db == nil {
		return &CalibrationState{
			TenantID:          tenantID,
			CalibrationFactor: 1.0,
			SampleCount:       0,
			LastComputedAt:    time.Now(),
		}, nil
	}

	var state CalibrationState
	err := s.db.GetContext(ctx, &state, `
		SELECT tenant_id, calibration_factor, sample_count, last_computed_at
		FROM   finops.forecast_calibration_state
		WHERE  tenant_id = $1;
	`, tenantID)
	if err != nil {
		return &CalibrationState{
			TenantID:          tenantID,
			CalibrationFactor: 1.0,
			SampleCount:       0,
			LastComputedAt:    time.Now(),
		}, nil
	}
	return &state, nil
}

// ListRecentFeedback returns the most recent N feedback records for a tenant.
func (s *ForecastFeedbackService) ListRecentFeedback(
	ctx context.Context,
	tenantID uuid.UUID,
	limit int,
) ([]ForecastFeedback, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}
	if s.db == nil {
		return nil, nil
	}
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	var records []ForecastFeedback
	err := s.db.SelectContext(ctx, &records, `
		SELECT feedback_id, tenant_id, forecast_id, outcome,
		       actual_scanned_bytes, actual_cpu_duration_ms, actual_cost_usd,
		       accuracy_ratio, notes, recorded_by, recorded_at
		FROM   finops.forecast_feedback
		WHERE  tenant_id = $1
		ORDER  BY recorded_at DESC
		LIMIT  $2;
	`, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed listing recent feedback: %w", err)
	}
	return records, nil
}

// ── Internal calibration computation ─────────────────────────────────────────

// refreshCalibration recomputes the rolling calibration factor for a tenant and
// upserts the result into finops.forecast_calibration_state.
//
// Algorithm:
//   Only ACCURATE and PARTIAL_SPIKE outcomes carry an accuracy_ratio signal.
//   We take the average of the last N ratios (exponential towards more recent data
//   via recency ordering) and store it as the correction multiplier.
//
//   factor > 1.0  → we have been under-predicting; future projections scaled up
//   factor < 1.0  → we have been over-predicting; future projections scaled down
//   factor = 1.0  → neutral (insufficient data or perfect historical accuracy)
func (s *ForecastFeedbackService) refreshCalibration(
	ctx context.Context,
	tenantID uuid.UUID,
) error {
	if s.db == nil {
		return nil
	}

	var result struct {
		AvgRatio *float64 `db:"avg_ratio"`
		Count    int      `db:"cnt"`
	}

	// Only outcomes that carry real actuals contribute to calibration.
	err := s.db.GetContext(ctx, &result, `
		SELECT
			AVG(accuracy_ratio)   AS avg_ratio,
			COUNT(accuracy_ratio) AS cnt
		FROM (
			SELECT accuracy_ratio
			FROM   finops.forecast_feedback
			WHERE  tenant_id = $1
			  AND  outcome   IN ('ACCURATE', 'PARTIAL_SPIKE')
			  AND  accuracy_ratio IS NOT NULL
			ORDER  BY recorded_at DESC
			LIMIT  $2
		) sub;
	`, tenantID, calibrationWindowSize)
	if err != nil || result.Count == 0 || result.AvgRatio == nil {
		// Not enough signal — keep/set neutral factor.
		_, _ = s.db.ExecContext(ctx, `
			INSERT INTO finops.forecast_calibration_state
				(tenant_id, calibration_factor, sample_count, last_computed_at)
			VALUES ($1, 1.0000, 0, NOW())
			ON CONFLICT (tenant_id)
			DO UPDATE SET
				calibration_factor = 1.0000,
				sample_count       = 0,
				last_computed_at   = NOW();
		`, tenantID)
		return nil
	}

	factor := *result.AvgRatio

	// Clamp to a sensible range — never more than 3× correction in either direction.
	if factor < 0.333 {
		factor = 0.333
	}
	if factor > 3.0 {
		factor = 3.0
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO finops.forecast_calibration_state
			(tenant_id, calibration_factor, sample_count, last_computed_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (tenant_id)
		DO UPDATE SET
			calibration_factor = EXCLUDED.calibration_factor,
			sample_count       = EXCLUDED.sample_count,
			last_computed_at   = NOW();
	`, tenantID, factor, result.Count)
	if err != nil {
		return fmt.Errorf("failed upserting calibration state: %w", err)
	}
	return nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func isValidOutcome(o ForecastOutcome) bool {
	switch o {
	case OutcomeAccurate, OutcomeFalsePositive, OutcomeMissedSpike, OutcomePartialSpike:
		return true
	}
	return false
}
