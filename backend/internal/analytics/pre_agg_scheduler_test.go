package analytics

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// nodeRow is one catalog_node row as PreAggScheduler.Tick scans it.
type nodeRow struct {
	ID         uuid.UUID       `db:"id"`
	Properties json.RawMessage `db:"properties"`
}

func newSchedulerFixture(t *testing.T) (*PreAggScheduler, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	sqlxDB := sqlx.NewDb(db, "postgres")
	s := NewPreAggScheduler(sqlxDB, NewPreAggLifecycleService(sqlxDB), nil)
	return s, mock
}

func propsJSON(t *testing.T, p models.PreAggProperties) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(p)
	require.NoError(t, err)
	return b
}

// TestScheduler_ShadowModeIsInert is acceptance test #14: with the scheduler in
// its default (shadow) mode, a due node produces a report entry and mutates
// nothing. No lifecycle write and no refresh may be attempted.
func TestScheduler_ShadowModeIsInert(t *testing.T) {
	s, mock := newSchedulerFixture(t)
	nodeID := uuid.New()

	props := propsJSON(t, models.PreAggProperties{
		BOName:                 "sales",
		TenantID:               "tenant_a",
		Dialect:                "starrocks",
		RefreshStrategy:        "interval",
		RefreshIntervalMinutes: 15,
		LifecycleStatus:        models.LifecycleActive,
	})

	mock.ExpectQuery(`(?s)FROM catalog_node n\s+JOIN catalog_node_type nt`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "properties"}).
			AddRow(nodeID, props))

	// Deliberately expect NO further queries. Any MarkRefreshing/Refresh/
	// MarkActive would show up as an unexpected-call error.
	report, err := s.Tick(context.Background())
	require.NoError(t, err)

	assert.Equal(t, SchedulerModeShadow, report.Mode)
	assert.False(t, s.Enabled(), "shadow is the default mode")
	assert.Equal(t, 1, report.Scanned)
	assert.Equal(t, []uuid.UUID{nodeID}, report.WouldRefresh,
		"shadow mode reports the action it would take")
	assert.Empty(t, report.Refreshed, "shadow mode must not refresh")
	assert.Empty(t, report.Failed)
	assert.NoError(t, mock.ExpectationsWereMet(),
		"shadow mode must issue no writes")
}

// TestScheduler_ShadowIsDefaultAndEnvFlagGatesEnable covers the global kill
// switch: an unset or non-"true" value must never enable refreshes.
func TestScheduler_ShadowIsDefaultAndEnvFlagGatesEnable(t *testing.T) {
	s, _ := newSchedulerFixture(t)
	assert.Equal(t, SchedulerModeShadow, s.Mode(),
		"NewPreAggScheduler must default to shadow")

	// Only the exact string "true" enables, matching the CBO_ENABLED precedent.
	for _, v := range []string{"", "false", "1", "TRUE", "yes", "enabled"} {
		s.SetMode(SchedulerModeFromEnv(v))
		assert.Equal(t, SchedulerModeShadow, s.Mode(),
			"env %q must not enable the scheduler", v)
	}

	s.SetMode(SchedulerModeFromEnv("true"))
	assert.Equal(t, SchedulerModeEnabled, s.Mode())
	assert.True(t, s.Enabled())
}

// TestScheduler_PerNodeManualOptOut is acceptance test #15: refresh_strategy
// "manual" is the per-node kill switch and works regardless of global mode.
//
// Shadow mode is used for the mode-independent assertion: the manual node must
// never appear in WouldRefresh. The global-flag interaction is covered
// separately by TestScheduler_ManualOptOutHoldsInEnabledMode, which supplies
// the DB expectations the refresh path needs.
func TestScheduler_PerNodeManualOptOut(t *testing.T) {
	s, mock := newSchedulerFixture(t)

	manual := uuid.New()
	interval := uuid.New()

	manualProps := propsJSON(t, models.PreAggProperties{
		BOName: "sales", TenantID: "t", Dialect: "starrocks",
		RefreshStrategy: "manual", LifecycleStatus: models.LifecycleActive,
	})
	intervalProps := propsJSON(t, models.PreAggProperties{
		BOName: "orders", TenantID: "t", Dialect: "starrocks",
		RefreshStrategy: "interval", RefreshIntervalMinutes: 5,
		LifecycleStatus: models.LifecycleActive,
	})

	mock.ExpectQuery(`(?s)FROM catalog_node n\s+JOIN catalog_node_type nt`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "properties"}).
			AddRow(manual, manualProps).
			AddRow(interval, intervalProps))

	report, err := s.Tick(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 2, report.Scanned)
	assert.Equal(t, 1, report.SkippedManual)
	assert.NotContains(t, report.WouldRefresh, manual,
		"manual node must be skipped even when due")
	assert.Contains(t, report.WouldRefresh, interval)
}

// TestScheduler_ManualOptOutHoldsInEnabledMode proves the per-node opt-out is
// independent of the global flag: with the scheduler enabled, a manual node is
// still skipped and no refresh is attempted for it.
func TestScheduler_ManualOptOutHoldsInEnabledMode(t *testing.T) {
	s, mock := newSchedulerFixture(t)
	s.SetMode(SchedulerModeEnabled)

	manual := uuid.New()
	manualProps := propsJSON(t, models.PreAggProperties{
		BOName: "sales", TenantID: "t", Dialect: "starrocks",
		RefreshStrategy: "manual", LifecycleStatus: models.LifecycleActive,
	})
	mock.ExpectQuery(`(?s)FROM catalog_node n\s+JOIN catalog_node_type nt`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "properties"}).
			AddRow(manual, manualProps))

	report, err := s.Tick(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 1, report.SkippedManual)
	assert.NotContains(t, report.WouldRefresh, manual)
	assert.Empty(t, report.Refreshed)
	// No further DB calls: a manual node must not be MarkRefreshing'd.
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestScheduler_NotDueIsSkipped verifies due-logic: a node scheduled in the
// future is not refreshed.
func TestScheduler_NotDueIsSkipped(t *testing.T) {
	s, mock := newSchedulerFixture(t)
	future := time.Now().UTC().Add(time.Hour)

	props := propsJSON(t, models.PreAggProperties{
		BOName: "sales", TenantID: "t", Dialect: "starrocks",
		RefreshStrategy:        "interval",
		RefreshIntervalMinutes: 15,
		LifecycleStatus:        models.LifecycleActive,
		NextScheduledRefresh:   &future,
	})
	mock.ExpectQuery(`(?s)FROM catalog_node n\s+JOIN catalog_node_type nt`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "properties"}).
			AddRow(uuid.New(), props))

	report, err := s.Tick(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, report.NotDue)
	assert.Empty(t, report.WouldRefresh)
}

// TestScheduler_StaleAlwaysDue verifies a stale node is refreshed regardless of
// its schedule, so a failed materialization retries promptly.
func TestScheduler_StaleAlwaysDue(t *testing.T) {
	s, mock := newSchedulerFixture(t)
	future := time.Now().UTC().Add(time.Hour)

	props := propsJSON(t, models.PreAggProperties{
		BOName: "sales", TenantID: "t", Dialect: "starrocks",
		RefreshStrategy: "interval", LifecycleStatus: models.LifecycleStale,
		NextScheduledRefresh: &future,
	})
	mock.ExpectQuery(`(?s)FROM catalog_node n\s+JOIN catalog_node_type nt`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "properties"}).
			AddRow(uuid.New(), props))

	report, err := s.Tick(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, len(report.WouldRefresh),
		"stale nodes are always due regardless of next_scheduled_refresh")
}

// TestScheduler_MalformedPropertiesAreCountedNotFatal verifies one bad node
// does not abort the cycle.
func TestScheduler_MalformedPropertiesAreCountedNotFatal(t *testing.T) {
	s, mock := newSchedulerFixture(t)

	good := propsJSON(t, models.PreAggProperties{
		BOName: "sales", TenantID: "t", Dialect: "starrocks",
		RefreshStrategy: "interval", LifecycleStatus: models.LifecycleActive,
	})
	goodID := uuid.New()

	mock.ExpectQuery(`(?s)FROM catalog_node n\s+JOIN catalog_node_type nt`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "properties"}).
			AddRow(uuid.New(), []byte(`{not valid json`)).
			AddRow(goodID, good))

	report, err := s.Tick(context.Background())
	require.NoError(t, err, "a malformed node must not fail the tick")
	assert.Equal(t, 1, report.ParseErrors)
	assert.Equal(t, []uuid.UUID{goodID}, report.WouldRefresh,
		"a malformed node must not block valid ones")
}

// TestJitteredInterval_Bounds verifies ±20% jitter stays within range and
// handles degenerate inputs without panicking or returning non-positive values.
func TestJitteredInterval_Bounds(t *testing.T) {
	base := 60 * time.Second

	// Extreme sources: rand in [0,1).
	low := jitteredInterval(base, 0.20, func() float64 { return 0 })
	high := jitteredInterval(base, 0.20, func() float64 { return 0.999999 })

	assert.GreaterOrEqual(t, low, time.Duration(float64(base)*0.8),
		"jitter must not fall below -20%")
	assert.LessOrEqual(t, high, time.Duration(float64(base)*1.2),
		"jitter must not exceed +20%")
	assert.Equal(t, base, jitteredInterval(base, 0, nil),
		"zero jitter is a no-op")
	assert.Equal(t, time.Duration(0), jitteredInterval(0, 0.2, nil),
		"non-positive base is returned unchanged (not the positive base)")
	assert.Greater(t, jitteredInterval(base, 0.20, nil), time.Duration(0),
		"jittered interval must remain positive")
}

// TestScheduler_DefaultIntervalIsSixtySeconds pins the documented cadence and
// records that sub-60s refresh is intentionally not expressible.
func TestScheduler_DefaultIntervalIsSixtySeconds(t *testing.T) {
	assert.Equal(t, 60*time.Second, DefaultPreAggTickInterval)
	assert.InDelta(t, 0.20, DefaultPreAggTickJitterPercent, 0.0001)
}
