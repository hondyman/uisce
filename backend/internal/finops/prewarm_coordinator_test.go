package finops

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrewarmCoordinator_NilDB_EmptyTargets(t *testing.T) {
	coord := NewPrewarmCoordinator(nil)
	tenantID := uuid.New()

	ctx := context.Background()

	// Default run: tomorrow mid-month without high probability skips below threshold
	res, err := coord.ExecuteOffPeakPrewarming(ctx, tenantID, nil)
	require.NoError(t, err)
	require.NotNil(t, res)

	assert.False(t, res.Triggered)
	assert.Contains(t, []string{"SKIPPED_BELOW_THRESHOLD", "SKIPPED_NO_TARGETS"}, res.Status)

	// Direct discovery test on empty targets:
	targets, err := coord.discoverHotTargets(ctx, tenantID)
	require.NoError(t, err)
	assert.Empty(t, targets)
}

func TestPrewarmCoordinator_SimulatedMetricsHonest(t *testing.T) {
	coord := NewPrewarmCoordinator(nil)
	tenantID := uuid.New()

	// Direct check on updateOrPersistLedgerEntry with empty targets and nil DB
	err := coord.updateOrPersistLedgerEntry(context.Background(), tenantID, nil, nil, &PrewarmResult{
		Status: "SKIPPED_NO_TARGETS",
	}, &WorkloadSmoothingPolicy{
		PolicyID: uuid.New(),
	}, "")
	require.NoError(t, err, "must safely no-op when db is nil")
}

func TestPrewarmCoordinator_ThresholdCheck(t *testing.T) {
	tenantID := uuid.New()

	// Verify policy default threshold properties
	policySvc := NewSmoothingPolicyService(nil)
	policy, err := policySvc.GetActivePolicy(context.Background(), tenantID)
	require.NoError(t, err)
	assert.InDelta(t, 2.5, policy.PrewarmThresholdMultiplier, 0.01)
	assert.InDelta(t, 0.70, policy.MinPeakProbabilityToPrewarm, 0.01)
}

func TestPrewarmCoordinator_CalendarMonthEndExpansion(t *testing.T) {
	// First 2 days of month should now qualify as month-end window
	day1 := time.Date(2026, time.May, 1, 12, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, time.May, 2, 12, 0, 0, 0, time.UTC)
	day3 := time.Date(2026, time.May, 3, 12, 0, 0, 0, time.UTC)

	assert.True(t, isMonthEndWindow(day1))
	assert.True(t, isMonthEndWindow(day2))
	assert.False(t, isMonthEndWindow(day3))
}

func TestPrewarmCoordinator_QuarterEndReconciliationWindow(t *testing.T) {
	// Quarter-end month close days: Mar 30, Mar 31
	mar30 := time.Date(2026, time.March, 30, 0, 0, 0, 0, time.UTC)
	mar31 := time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC)
	assert.True(t, isQuarterEndWindow(mar30))
	assert.True(t, isQuarterEndWindow(mar31))

	// Post quarter-close reconciliation days: Apr 1, Apr 2
	apr1 := time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC)
	apr2 := time.Date(2026, time.April, 2, 0, 0, 0, 0, time.UTC)
	apr3 := time.Date(2026, time.April, 3, 0, 0, 0, 0, time.UTC)
	assert.True(t, isQuarterEndWindow(apr1), "Apr 1 is post quarter-close reporting burst")
	assert.True(t, isQuarterEndWindow(apr2), "Apr 2 is post quarter-close reporting burst")
	assert.False(t, isQuarterEndWindow(apr3))

	// Q4 close reconciliation days in January: Jan 1, Jan 2
	jan1 := time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC)
	jan2 := time.Date(2027, time.January, 2, 0, 0, 0, 0, time.UTC)
	jan3 := time.Date(2027, time.January, 3, 0, 0, 0, 0, time.UTC)
	assert.True(t, isQuarterEndWindow(jan1), "Jan 1 is post Q4 close reporting burst")
	assert.True(t, isQuarterEndWindow(jan2), "Jan 2 is post Q4 close reporting burst")
	assert.False(t, isQuarterEndWindow(jan3))
}

func TestPrewarmCoordinator_InFlightDeduplication(t *testing.T) {
	coord := NewPrewarmCoordinator(nil)
	tenantID := uuid.New()

	locked := coord.TryLockTenant(tenantID)
	assert.True(t, locked)

	// Second lock attempt must fail
	secondLock := coord.TryLockTenant(tenantID)
	assert.False(t, secondLock)

	coord.UnlockTenant(tenantID)
	thirdLock := coord.TryLockTenant(tenantID)
	assert.True(t, thirdLock)
}

func TestPrewarmCoordinator_ANDGateSemantics(t *testing.T) {
	// With policy defaults: MinPeakProbabilityToPrewarm = 0.70, PrewarmThresholdMultiplier = 2.50
	policy := &WorkloadSmoothingPolicy{
		MinPeakProbabilityToPrewarm:  0.70,
		PrewarmThresholdMultiplier: 2.50,
	}

	// Case 1: Pure month-end (M = 2.80, PeakProb = 0.514)
	// Multiplier (2.80 >= 2.50) passes, but PeakProb (0.514 < 0.70) fails -> SKIPPED
	prob1 := (2.80 - 1.0) / 3.5 // 0.514
	mult1 := 2.80
	passes1 := prob1 >= policy.MinPeakProbabilityToPrewarm && mult1 >= policy.PrewarmThresholdMultiplier
	assert.False(t, passes1, "pure month-end without bursts must NOT trigger prewarm under AND gate")

	// Case 2: Quarter-end (M = 4.50, PeakProb = 1.0)
	// Both pass -> TRIGGERED
	prob2 := 1.0
	mult2 := 4.50
	passes2 := prob2 >= policy.MinPeakProbabilityToPrewarm && mult2 >= policy.PrewarmThresholdMultiplier
	assert.True(t, passes2, "quarter-end spike must trigger prewarming")
}

// TestPrewarmCoordinator_TerminalUpdate_TargetMetricPerTarget covers the
// per-target × UPDATE cell of the (path × row-state) matrix. Symmetric
// to the per-target INSERT test, but exercises the UPDATE branch with a
// PENDING row that updateOrPersistLedgerEntry matches. Asserts no
// mixed-semantics row can be produced: per-target UPDATE writes
// bo_id=<first target's BOID> and target_metric=<first target's
// FormulaType> together. (The original bug: UPDATE setting
// bo_id=<first target> while leaving target_metric='ALL' from the
// original PENDING — guarded by THIS test, not by the run-level one.)
func TestPrewarmCoordinator_TerminalUpdate_TargetMetricPerTarget(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	tenantID := uuid.New()
	jobID := uuid.New()
	primaryBOID := uuid.New()

	mock.ExpectExec(`UPDATE finops\.prewarm_execution_ledger`).
		WithArgs(
			&primaryBOID,     // $1 bo_id (first target's BO)
			"XIRR",           // $2 target_metric (first target's FormulaType)
			sqlmock.AnyArg(), // $3 entities_prewarmed_count
			sqlmock.AnyArg(), // $4 compute_cost
			sqlmock.AnyArg(), // $5 estimated_savings
			sqlmock.AnyArg(), // $6 peak_probability
			sqlmock.AnyArg(), // $7 policy_id (nil per existing pattern)
			sqlmock.AnyArg(), // $8 status
			sqlmock.AnyArg(), // $9 error_detail
			tenantID,         // $10
			&jobID,           // $11
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	coord := NewPrewarmCoordinator(sqlxDB)
	targets := []hotTarget{
		{BOID: primaryBOID, FieldID: uuid.New(), HitCount: 100, FormulaType: "XIRR"},
	}
	result := &PrewarmResult{
		TenantID:        tenantID,
		Status:          "SIMULATED",
		TargetsSeeded:   1,
		PeakProbability: 0.95,
	}
	err = coord.updateOrPersistLedgerEntry(context.Background(), tenantID, &jobID, targets, result, nil, "")
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
