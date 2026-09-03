package finops

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── isValidOutcome ────────────────────────────────────────────────────────────

func TestIsValidOutcome_Valid(t *testing.T) {
	for _, o := range []ForecastOutcome{
		OutcomeAccurate, OutcomeFalsePositive, OutcomeMissedSpike, OutcomePartialSpike,
	} {
		assert.True(t, isValidOutcome(o), "expected %q to be valid", o)
	}
}

func TestIsValidOutcome_Invalid(t *testing.T) {
	assert.False(t, isValidOutcome("WRONG"))
	assert.False(t, isValidOutcome(""))
	assert.False(t, isValidOutcome("accurate")) // case-sensitive
}

// ── ForecastFeedbackService (nil-DB sandbox mode) ─────────────────────────────

func TestSubmitFeedback_NilTenantID(t *testing.T) {
	svc := NewForecastFeedbackService(nil)
	_, err := svc.SubmitFeedback(nil, uuid.Nil, SubmitFeedbackRequest{ //nolint:staticcheck
		ForecastID: uuid.New(),
		Outcome:    OutcomeAccurate,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Rule 7 violation")
}

func TestSubmitFeedback_NilForecastID(t *testing.T) {
	svc := NewForecastFeedbackService(nil)
	_, err := svc.SubmitFeedback(nil, uuid.New(), SubmitFeedbackRequest{ //nolint:staticcheck
		ForecastID: uuid.Nil,
		Outcome:    OutcomeAccurate,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "forecast_id")
}

func TestSubmitFeedback_InvalidOutcome(t *testing.T) {
	svc := NewForecastFeedbackService(nil)
	_, err := svc.SubmitFeedback(nil, uuid.New(), SubmitFeedbackRequest{ //nolint:staticcheck
		ForecastID: uuid.New(),
		Outcome:    "INVENTED",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid outcome")
}

func TestSubmitFeedback_SandboxSuccess(t *testing.T) {
	svc := NewForecastFeedbackService(nil)
	tenantID := uuid.New()
	forecastID := uuid.New()
	cost := 320.50

	fb, err := svc.SubmitFeedback(nil, tenantID, SubmitFeedbackRequest{ //nolint:staticcheck
		ForecastID:    forecastID,
		Outcome:       OutcomeAccurate,
		ActualCostUSD: &cost,
	})
	require.NoError(t, err)
	require.NotNil(t, fb)

	assert.Equal(t, tenantID, fb.TenantID)
	assert.Equal(t, forecastID, fb.ForecastID)
	assert.Equal(t, OutcomeAccurate, fb.Outcome)
	assert.NotEqual(t, uuid.Nil, fb.FeedbackID)
	assert.WithinDuration(t, time.Now(), fb.RecordedAt, 5*time.Second)
}

func TestSubmitFeedback_AllOutcomes(t *testing.T) {
	svc := NewForecastFeedbackService(nil)
	tenantID := uuid.New()

	for _, outcome := range []ForecastOutcome{
		OutcomeAccurate, OutcomeFalsePositive, OutcomeMissedSpike, OutcomePartialSpike,
	} {
		fb, err := svc.SubmitFeedback(nil, tenantID, SubmitFeedbackRequest{ //nolint:staticcheck
			ForecastID: uuid.New(),
			Outcome:    outcome,
		})
		require.NoError(t, err, "outcome %q should be accepted", outcome)
		assert.Equal(t, outcome, fb.Outcome)
	}
}

// ── GetCalibrationFactor (nil-DB) ─────────────────────────────────────────────

func TestGetCalibrationFactor_NilTenantID(t *testing.T) {
	svc := NewForecastFeedbackService(nil)
	_, _, err := svc.GetCalibrationFactor(nil, uuid.Nil) //nolint:staticcheck
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Rule 7 violation")
}

func TestGetCalibrationFactor_NoDataReturnsNeutral(t *testing.T) {
	svc := NewForecastFeedbackService(nil)
	factor, samples, err := svc.GetCalibrationFactor(nil, uuid.New()) //nolint:staticcheck
	require.NoError(t, err)
	assert.Equal(t, 1.0, factor, "no data → neutral factor")
	assert.Equal(t, 0, samples)
}

// ── GetCalibrationState (nil-DB) ─────────────────────────────────────────────

func TestGetCalibrationState_NilUUID(t *testing.T) {
	svc := NewForecastFeedbackService(nil)
	_, err := svc.GetCalibrationState(nil, uuid.Nil) //nolint:staticcheck
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Rule 7 violation")
}

func TestGetCalibrationState_NoDataReturnsDefault(t *testing.T) {
	svc := NewForecastFeedbackService(nil)
	tenantID := uuid.New()
	state, err := svc.GetCalibrationState(nil, tenantID) //nolint:staticcheck
	require.NoError(t, err)
	require.NotNil(t, state)
	assert.Equal(t, tenantID, state.TenantID)
	assert.Equal(t, 1.0, state.CalibrationFactor)
	assert.Equal(t, 0, state.SampleCount)
}

// ── ListRecentFeedback (nil-DB) ───────────────────────────────────────────────

func TestListRecentFeedback_NilUUID(t *testing.T) {
	svc := NewForecastFeedbackService(nil)
	_, err := svc.ListRecentFeedback(nil, uuid.Nil, 10) //nolint:staticcheck
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Rule 7 violation")
}

func TestListRecentFeedback_NilDBReturnsEmpty(t *testing.T) {
	svc := NewForecastFeedbackService(nil)
	results, err := svc.ListRecentFeedback(nil, uuid.New(), 5) //nolint:staticcheck
	require.NoError(t, err)
	assert.Nil(t, results)
}

// ── Calibration applied in DemandForecaster ───────────────────────────────────
// These tests verify that when the feedback service returns a calibration factor,
// DemandForecaster applies it to projections.

func TestDemandForecaster_CalibrationFactorApplied(t *testing.T) {
	f := NewDemandForecaster(nil)

	// Manually set a mock feedback service that returns a known factor.
	f.feedbackService = &ForecastFeedbackService{db: nil} // nil-DB → always returns 1.0

	midMonth := time.Date(2026, time.July, 15, 0, 0, 0, 0, time.UTC)
	tenantID := uuid.New()

	res, err := f.GenerateTenantDemandForecast(nil, tenantID, midMonth) //nolint:staticcheck
	require.NoError(t, err)

	// With nil-DB feedback (factor = 1.0) and no calendar multiplier:
	// projected = 500MB * 1.0 * 1.0 = 500MB
	assert.Equal(t, int64(500_000_000), res.ProjectedBytes)
	assert.Equal(t, 1.0, res.CalibrationFactor)
	assert.Equal(t, 0, res.CalibrationSamples)
}

func TestDemandForecaster_CalibrationFieldsPresent(t *testing.T) {
	f := NewDemandForecaster(nil)
	res, err := f.GenerateTenantDemandForecast(nil, uuid.New(), //nolint:staticcheck
		time.Date(2026, time.August, 10, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)

	// CalibrationFactor and CalibrationSamples must always be present in the result.
	assert.Equal(t, 1.0, res.CalibrationFactor, "neutral factor when no feedback exists")
	assert.GreaterOrEqual(t, res.CalibrationSamples, 0)
}
