package finops

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── isMonthEndWindow ──────────────────────────────────────────────────────────

func TestIsMonthEndWindow_LastDayOfMonth(t *testing.T) {
	// Jan 31 → Feb 1 is a different month: should be true.
	jan31 := time.Date(2026, time.January, 31, 12, 0, 0, 0, time.UTC)
	assert.True(t, isMonthEndWindow(jan31), "Jan 31 should be in the month-end window")
}

func TestIsMonthEndWindow_TwoDaysBeforeEnd(t *testing.T) {
	// Jan 29: adding 48h = Jan 31, still January → false.
	jan29 := time.Date(2026, time.January, 29, 0, 0, 0, 0, time.UTC)
	assert.False(t, isMonthEndWindow(jan29), "Jan 29 is not in the month-end window")
}

func TestIsMonthEndWindow_LastDayFeb(t *testing.T) {
	// Feb 28 (non-leap year) is month-end.
	feb28 := time.Date(2026, time.February, 28, 0, 0, 0, 0, time.UTC)
	assert.True(t, isMonthEndWindow(feb28))
}

func TestIsMonthEndWindow_LeapYearFeb28(t *testing.T) {
	// In a leap year, Feb 28 + 48h = Mar 1 → different month → true.
	feb28Leap := time.Date(2024, time.February, 28, 0, 0, 0, 0, time.UTC)
	assert.True(t, isMonthEndWindow(feb28Leap), "Feb 28 in leap year crosses month boundary at +48h")
}

// ── isQuarterEndWindow ────────────────────────────────────────────────────────

func TestIsQuarterEndWindow_MarEnd(t *testing.T) {
	mar30 := time.Date(2026, time.March, 30, 0, 0, 0, 0, time.UTC)
	assert.True(t, isQuarterEndWindow(mar30), "March 30 should be quarter-end window")
}

func TestIsQuarterEndWindow_JunEnd(t *testing.T) {
	jun29 := time.Date(2026, time.June, 29, 0, 0, 0, 0, time.UTC)
	assert.True(t, isQuarterEndWindow(jun29))
}

func TestIsQuarterEndWindow_SepEnd(t *testing.T) {
	sep29 := time.Date(2026, time.September, 29, 0, 0, 0, 0, time.UTC)
	assert.True(t, isQuarterEndWindow(sep29))
}

func TestIsQuarterEndWindow_DecEnd(t *testing.T) {
	dec30 := time.Date(2026, time.December, 30, 0, 0, 0, 0, time.UTC)
	assert.True(t, isQuarterEndWindow(dec30))
}

func TestIsQuarterEndWindow_NonQuarterMonthEnd(t *testing.T) {
	// April is a month-end but NOT a quarter-end.
	apr29 := time.Date(2026, time.April, 29, 0, 0, 0, 0, time.UTC)
	assert.False(t, isQuarterEndWindow(apr29), "April is not a quarter-end month")
}

func TestIsQuarterEndWindow_MidMonth(t *testing.T) {
	mar15 := time.Date(2026, time.March, 15, 0, 0, 0, 0, time.UTC)
	assert.False(t, isQuarterEndWindow(mar15), "Mid-March is not quarter-end window")
}

// ── DemandForecaster (nil-DB sandbox mode) ────────────────────────────────────

func TestGenerateTenantDemandForecast_NilUUID(t *testing.T) {
	f := NewDemandForecaster(nil)
	_, err := f.GenerateTenantDemandForecast(nil, uuid.Nil, time.Now()) //nolint:staticcheck
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Rule 7 violation")
}

func TestGenerateTenantDemandForecast_MidMonthBaseline(t *testing.T) {
	f := NewDemandForecaster(nil) // nil DB → uses sandbox defaults

	// Pick a date safely in the middle of a month (no calendar multiplier).
	midMonth := time.Date(2026, time.July, 15, 0, 0, 0, 0, time.UTC)
	tenantID := uuid.New()

	res, err := f.GenerateTenantDemandForecast(nil, tenantID, midMonth) //nolint:staticcheck
	require.NoError(t, err)

	assert.Equal(t, tenantID, res.TenantID)
	assert.Empty(t, res.ContributingFactors, "mid-month should have no calendar factors")
	// Multiplier is 1.0, so projected = defaults × 1.0
	assert.Equal(t, int64(500_000_000), res.ProjectedBytes)
	assert.InDelta(t, 0.0, res.PeakProbability, 0.001)
	assert.InDelta(t, 0.65, res.ConfidenceScore, 0.001)
}

func TestGenerateTenantDemandForecast_MonthEnd(t *testing.T) {
	f := NewDemandForecaster(nil)
	tenantID := uuid.New()

	// March 30 is month-end AND quarter-end → multiplier 4.5
	mar30 := time.Date(2026, time.March, 30, 0, 0, 0, 0, time.UTC)
	res, err := f.GenerateTenantDemandForecast(nil, tenantID, mar30) //nolint:staticcheck
	require.NoError(t, err)

	assert.Contains(t, res.ContributingFactors, "CALENDAR_QUARTER_END")
	assert.NotContains(t, res.ContributingFactors, "CALENDAR_MONTH_END",
		"quarter-end supersedes month-end factor label")

	// projected_bytes = 500MB × 4.5 = 2,250,000,000
	assert.Equal(t, int64(2_250_000_000), res.ProjectedBytes)

	// peak_prob = min(1.0, (4.5-1.0)/3.5) = 1.0
	assert.InDelta(t, 1.0, res.PeakProbability, 0.001)
	assert.InDelta(t, 0.88, res.ConfidenceScore, 0.001)
}

func TestGenerateTenantDemandForecast_PureMonthEnd(t *testing.T) {
	f := NewDemandForecaster(nil)
	tenantID := uuid.New()

	// April 29: month-end but not quarter-end → multiplier 2.8
	apr29 := time.Date(2026, time.April, 29, 0, 0, 0, 0, time.UTC)
	res, err := f.GenerateTenantDemandForecast(nil, tenantID, apr29) //nolint:staticcheck
	require.NoError(t, err)

	assert.Contains(t, res.ContributingFactors, "CALENDAR_MONTH_END")
	assert.NotContains(t, res.ContributingFactors, "CALENDAR_QUARTER_END")

	// projected_bytes = 500MB × 2.8 = 1,400,000,000
	assert.Equal(t, int64(1_400_000_000), res.ProjectedBytes)

	// peak_prob = min(1.0, (2.8-1.0)/3.5) ≈ 0.514
	assert.InDelta(t, (2.8-1.0)/3.5, res.PeakProbability, 0.001)
}

func TestGenerateTenantDemandForecast_WindowBoundaries(t *testing.T) {
	f := NewDemandForecaster(nil)
	tenantID := uuid.New()
	targetDate := time.Date(2026, time.August, 10, 15, 30, 0, 0, time.UTC)

	res, err := f.GenerateTenantDemandForecast(nil, tenantID, targetDate) //nolint:staticcheck
	require.NoError(t, err)

	// Window should be midnight-to-midnight on the target date.
	expectedStart := time.Date(2026, time.August, 10, 0, 0, 0, 0, time.UTC)
	expectedEnd := expectedStart.Add(24 * time.Hour)
	assert.Equal(t, expectedStart, res.WindowStart)
	assert.Equal(t, expectedEnd, res.WindowEnd)
}

// ── SmoothingPolicyService (nil-DB mode) ─────────────────────────────────────

func TestGetActivePolicy_NilUUID(t *testing.T) {
	svc := NewSmoothingPolicyService(nil)
	_, err := svc.GetActivePolicy(nil, uuid.Nil) //nolint:staticcheck
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Rule 7 violation")
}

func TestGetActivePolicy_ReturnsDefault(t *testing.T) {
	svc := NewSmoothingPolicyService(nil)
	tenantID := uuid.New()
	policy, err := svc.GetActivePolicy(nil, tenantID) //nolint:staticcheck
	require.NoError(t, err)
	require.NotNil(t, policy)
	assert.Equal(t, tenantID, policy.TenantID)
	assert.Equal(t, "0 2 * * *", policy.OffPeakCron)
	assert.Equal(t, 0.700, policy.MinPeakProbabilityToPrewarm)
	assert.True(t, policy.EnableBurstDeferral)
}

func TestUpsertPolicy_Validation(t *testing.T) {
	svc := NewSmoothingPolicyService(nil)
	tenantID := uuid.New()

	// Invalid probability range.
	_, err := svc.UpsertPolicy(nil, &WorkloadSmoothingPolicy{ //nolint:staticcheck
		TenantID:                    tenantID,
		PolicyName:                  "BAD",
		MinPeakProbabilityToPrewarm: 1.5,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "min_peak_probability_to_prewarm")

	// Missing policy name.
	_, err = svc.UpsertPolicy(nil, &WorkloadSmoothingPolicy{ //nolint:staticcheck
		TenantID: tenantID,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "policy_name")
}
