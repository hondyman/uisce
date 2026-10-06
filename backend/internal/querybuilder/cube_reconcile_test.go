package querybuilder

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

type memProbe struct {
	mu       sync.Mutex
	hot      map[string]bool // db\x1fname
	cold     map[string]bool
	dropped  []string
	quar     []string
	listDBs  map[string][]string
}

func newMemProbe() *memProbe {
	return &memProbe{
		hot:     map[string]bool{},
		cold:    map[string]bool{},
		listDBs: map[string][]string{},
	}
}

func (p *memProbe) key(db, name string) string { return db + "\x1f" + name }

func (p *memProbe) HotExists(_ context.Context, database, name string) (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.hot[p.key(database, name)], nil
}

func (p *memProbe) ColdExists(_ context.Context, icebergTable string) (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.cold[icebergTable], nil
}

func (p *memProbe) DropHot(_ context.Context, database, name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	k := p.key(database, name)
	delete(p.hot, k)
	p.dropped = append(p.dropped, "hot:"+k)
	return nil
}

func (p *memProbe) DropCold(_ context.Context, icebergTable string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.cold, icebergTable)
	p.dropped = append(p.dropped, "cold:"+icebergTable)
	return nil
}

func (p *memProbe) QuarantineHot(_ context.Context, database, name string) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	k := p.key(database, name)
	delete(p.hot, k)
	q := "cube_quarantine_" + name
	p.hot[p.key(database, q)] = true
	p.quar = append(p.quar, k+"->"+q)
	return q, nil
}

func (p *memProbe) ListHotNames(_ context.Context, database string) ([]string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if names, ok := p.listDBs[database]; ok {
		return append([]string(nil), names...), nil
	}
	var out []string
	prefix := database + "\x1f"
	for k, ok := range p.hot {
		if ok && len(k) > len(prefix) && k[:len(prefix)] == prefix {
			out = append(out, k[len(prefix):])
		}
	}
	return out, nil
}

func TestCubeReconciler_ActiveMissingPhysicalMarksFailed(t *testing.T) {
	db, mockSQL, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	sqlxDB := sqlx.NewDb(db, "postgres")

	nodeID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	props := models.PreAggProperties{
		TenantID:            "t1",
		TargetDatabase:      "tenant_t1",
		LifecycleStatus:     models.LifecycleActive,
		CubeID:              "cube-1",
		AttemptID:           "att-1",
		IcebergTable:        "iceberg_catalog.cubes.cube_t1_x",
		DualCommitWatermark: ptrTime(time.Now().UTC().Add(-time.Hour)),
	}
	propsJSON, err := json.Marshal(props)
	require.NoError(t, err)
	cfg := models.PreAggConfig{Materialization: models.MaterializationConfig{TargetName: "cube_t1_x"}}
	cfgJSON, err := json.Marshal(cfg)
	require.NoError(t, err)

	mockSQL.ExpectQuery(`SELECT cn.id, cn.node_name, cn.properties, cn.config`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "node_name", "properties", "config"}).
			AddRow(nodeID, "cube_t1_x", propsJSON, cfgJSON))

	// MarkFailedAttempt: SELECT properties then UPDATE
	mockSQL.ExpectQuery(`SELECT properties FROM catalog_node WHERE id = \$1`).
		WithArgs(nodeID).
		WillReturnRows(sqlmock.NewRows([]string{"properties"}).AddRow(propsJSON))
	mockSQL.ExpectExec(`UPDATE catalog_node SET properties`).
		WithArgs(failedLifecycleMatcher{}, nodeID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	probe := newMemProbe()
	// hot missing, cold present → active_missing_physical
	probe.cold[props.IcebergTable] = true

	r := NewCubeReconciler(sqlxDB, nil)
	r.SetProbe(probe)
	receipt, err := r.Reconcile(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, receipt.Examined)
	require.Len(t, receipt.Actions, 1)
	require.Equal(t, ReconcileActionActiveMissingPhysical, receipt.Actions[0].Kind)
	require.Contains(t, receipt.Actions[0].Detail, "missing=hot")
	require.NoError(t, mockSQL.ExpectationsWereMet())
}

func TestCubeReconciler_StaleAttemptCleanupAfterMidSaga(t *testing.T) {
	db, mockSQL, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	sqlxDB := sqlx.NewDb(db, "postgres")

	nodeID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	started := time.Now().UTC().Add(-3 * time.Hour)
	props := models.PreAggProperties{
		TenantID:           "t1",
		TargetDatabase:     "tenant_t1",
		LifecycleStatus:    models.LifecycleMaterializing,
		CubeID:             "cube-1",
		AttemptID:          "att-mid-saga",
		LastMaterializedAt: &started,
		IcebergTable:       "iceberg_catalog.cubes.cube_t1_mid",
	}
	propsJSON, err := json.Marshal(props)
	require.NoError(t, err)
	cfg := models.PreAggConfig{Materialization: models.MaterializationConfig{TargetName: "cube_t1_mid"}}
	cfgJSON, err := json.Marshal(cfg)
	require.NoError(t, err)

	mockSQL.ExpectQuery(`SELECT cn.id, cn.node_name, cn.properties, cn.config`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "node_name", "properties", "config"}).
			AddRow(nodeID, "cube_t1_mid", propsJSON, cfgJSON))
	mockSQL.ExpectQuery(`SELECT properties FROM catalog_node WHERE id = \$1`).
		WithArgs(nodeID).
		WillReturnRows(sqlmock.NewRows([]string{"properties"}).AddRow(propsJSON))
	mockSQL.ExpectExec(`UPDATE catalog_node SET properties`).
		WithArgs(failedLifecycleMatcher{}, nodeID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	probe := newMemProbe()
	probe.hot[probe.key("tenant_t1", "cube_t1_mid")] = true
	probe.cold[props.IcebergTable] = true

	r := NewCubeReconciler(sqlxDB, nil)
	r.SetProbe(probe)
	r.SetStaleAttemptTTL(time.Hour)

	receipt, err := r.Reconcile(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, receipt.Examined)
	require.Len(t, receipt.Actions, 1)
	require.Equal(t, ReconcileActionStaleAttemptCleanup, receipt.Actions[0].Kind)
	require.Contains(t, receipt.Actions[0].Detail, "dropped_hot")
	require.Contains(t, receipt.Actions[0].Detail, "dropped_cold")
	require.False(t, probe.hot[probe.key("tenant_t1", "cube_t1_mid")])
	require.False(t, probe.cold[props.IcebergTable])
	require.NoError(t, mockSQL.ExpectationsWereMet())
}

func TestCubeReconciler_FailedLeftoverAfterMidSagaCompensate(t *testing.T) {
	db, mockSQL, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	sqlxDB := sqlx.NewDb(db, "postgres")

	nodeID := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	props := models.PreAggProperties{
		TenantID:        "t1",
		TargetDatabase:  "tenant_t1",
		LifecycleStatus: models.LifecycleFailed,
		CubeID:          "cube-1",
		AttemptID:       "att-cold-fail",
		IcebergTable:    "iceberg_catalog.cubes.cube_t1_left",
		LastRefreshError: "cold sink failed",
	}
	propsJSON, err := json.Marshal(props)
	require.NoError(t, err)
	cfg := models.PreAggConfig{Materialization: models.MaterializationConfig{TargetName: "cube_t1_left"}}
	cfgJSON, err := json.Marshal(cfg)
	require.NoError(t, err)

	mockSQL.ExpectQuery(`SELECT cn.id, cn.node_name, cn.properties, cn.config`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "node_name", "properties", "config"}).
			AddRow(nodeID, "cube_t1_left", propsJSON, cfgJSON))

	probe := newMemProbe()
	// CompensateHot failed: hot still present under Failed lifecycle.
	probe.hot[probe.key("tenant_t1", "cube_t1_left")] = true
	probe.listDBs["tenant_t1"] = []string{"cube_t1_left"}

	r := NewCubeReconciler(sqlxDB, nil)
	r.SetProbe(probe)
	receipt, err := r.Reconcile(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, receipt.Examined)
	require.Len(t, receipt.Actions, 1)
	require.Equal(t, ReconcileActionFailedLeftoverCleanup, receipt.Actions[0].Kind)
	require.Contains(t, receipt.Actions[0].Detail, "quarantined_hot=")
	require.False(t, probe.hot[probe.key("tenant_t1", "cube_t1_left")])
	require.NoError(t, mockSQL.ExpectationsWereMet())
}

func TestCubeReconciler_OrphanQuarantined(t *testing.T) {
	db, mockSQL, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	sqlxDB := sqlx.NewDb(db, "postgres")

	nodeID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	wm := time.Now().UTC()
	props := models.PreAggProperties{
		TenantID:            "t1",
		TargetDatabase:      "tenant_t1",
		LifecycleStatus:     models.LifecycleActive,
		CubeID:              "cube-1",
		AttemptID:           "att-ok",
		IcebergTable:        "iceberg_catalog.cubes.cube_t1_ok",
		DualCommitWatermark: &wm,
		HotCommittedAt:      &wm,
		ColdCommittedAt:     &wm,
	}
	propsJSON, err := json.Marshal(props)
	require.NoError(t, err)
	cfg := models.PreAggConfig{Materialization: models.MaterializationConfig{TargetName: "cube_t1_ok"}}
	cfgJSON, err := json.Marshal(cfg)
	require.NoError(t, err)

	mockSQL.ExpectQuery(`SELECT cn.id, cn.node_name, cn.properties, cn.config`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "node_name", "properties", "config"}).
			AddRow(nodeID, "cube_t1_ok", propsJSON, cfgJSON))

	probe := newMemProbe()
	probe.hot[probe.key("tenant_t1", "cube_t1_ok")] = true
	probe.cold[props.IcebergTable] = true
	probe.hot[probe.key("tenant_t1", "cube_t1_orphan")] = true
	probe.listDBs["tenant_t1"] = []string{"cube_t1_ok", "cube_t1_orphan"}

	r := NewCubeReconciler(sqlxDB, nil)
	r.SetProbe(probe)
	receipt, err := r.Reconcile(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, receipt.Examined)
	require.Len(t, receipt.Actions, 1)
	require.Equal(t, ReconcileActionOrphanQuarantined, receipt.Actions[0].Kind)
	require.Equal(t, "cube_t1_orphan", receipt.Actions[0].NodeName)
	require.False(t, probe.hot[probe.key("tenant_t1", "cube_t1_orphan")])
	require.True(t, probe.hot[probe.key("tenant_t1", "cube_quarantine_cube_t1_orphan")])
	require.NoError(t, mockSQL.ExpectationsWereMet())
}

func TestCubeReconcileWorkflow_ReturnsReceipt(t *testing.T) {
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	receipt := &CubeReconcileReceipt{
		StartedAt:  time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC),
		FinishedAt: time.Date(2026, 10, 5, 12, 0, 1, 0, time.UTC),
		Examined:   2,
		Actions: []CubeReconcileAction{{
			Kind:     ReconcileActionStaleAttemptCleanup,
			NodeName: "cube_t1_mid",
			Detail:   "dropped_hot dropped_cold",
		}},
	}

	acts := &CubeReconcileActivities{}
	env.RegisterActivityWithOptions(acts.CubeReconcile, activity.RegisterOptions{Name: ActCubeReconcile})
	env.OnActivity(ActCubeReconcile, mock.Anything).Return(receipt, nil)

	env.ExecuteWorkflow(CubeReconcileWorkflow)
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var got CubeReconcileReceipt
	require.NoError(t, env.GetWorkflowResult(&got))
	require.Equal(t, 2, got.Examined)
	require.Len(t, got.Actions, 1)
	require.Equal(t, ReconcileActionStaleAttemptCleanup, got.Actions[0].Kind)
}

func ptrTime(t time.Time) *time.Time { return &t }

type failedLifecycleMatcher struct{}

func (failedLifecycleMatcher) Match(v driver.Value) bool {
	var b []byte
	switch x := v.(type) {
	case []byte:
		b = x
	case string:
		b = []byte(x)
	default:
		return false
	}
	var p models.PreAggProperties
	if err := json.Unmarshal(b, &p); err != nil {
		return false
	}
	return p.LifecycleStatus == models.LifecycleFailed && p.LastRefreshStatus == "failed"
}

func (failedLifecycleMatcher) String() string {
	return "lifecycle_status=failed"
}
