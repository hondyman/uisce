package analytics

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests cover the #394 removals. Each one asserts the behaviour the
// defect report claimed, by traversing the same predicate production uses.
//
// The trap these are written against: a test that asserts a *count* of
// rejections, or that greps the error string for a word it happened to put
// there, passes for the wrong reason and keeps passing after the behaviour
// regresses. So each assertion below is a property the operator depends on,
// checked against the real return value.

// preAggConfigJSON builds a catalog_node.config payload.
func preAggConfigJSON(t *testing.T, c models.PreAggConfig) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(c)
	require.NoError(t, err)
	return b
}

// preAggPropsJSON builds a catalog_node.properties payload.
func preAggPropsJSON(t *testing.T, p models.PreAggProperties) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(p)
	require.NoError(t, err)
	return b
}

// --- Item 1: a `table` target must fail refresh with an actionable error ---

// TestRefresh_TableTargetExplainsItselfRatherThanClaimingReapplyWorks is the
// regression test for the silent staleness defect.
//
// Before this change Refresh returned "refresh is only supported for
// materialized_view targets, got \"table\"", while the comment directly above it
// told the reader that re-running ApplyMaterialization would pick up new data.
// It would not: the generated DDL is CREATE TABLE IF NOT EXISTS, so re-applying
// it succeeds and replaces nothing. The test asserts the three things an
// operator needs in order to act, because a correct error that names none of
// them is not an improvement.
func TestRefresh_TableTargetExplainsItselfRatherThanClaimingReapplyWorks(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	sqlxDB := sqlx.NewDb(db, "postgres")

	svc := NewPreAggregationService(sqlxDB, nil, nil)
	// A non-nil StarRocks handle is required to get past the connection guard
	// and reach the strategy check, which is the code under test.
	svc.starrocksDB = db

	id := uuid.New()
	mock.ExpectQuery(`(?s)FROM catalog_node WHERE id = \$1`).
		WillReturnRows(sqlmock.NewRows([]string{"properties", "config"}).
			AddRow(
				preAggPropsJSON(t, models.PreAggProperties{TargetDatabase: "tenant_a"}),
				preAggConfigJSON(t, models.PreAggConfig{
					Materialization: models.MaterializationConfig{Type: "table", TargetName: "sales_daily"},
				}),
			))

	err = svc.Refresh(context.Background(), id)

	require.Error(t, err, "a table target has no refresh path and must not report success")

	// 1. Names the strategy, so the operator knows which setting is responsible.
	assert.Contains(t, err.Error(), `"table"`,
		"the error must name the materialization strategy that cannot refresh")
	// 2. Names the target, so it is identifiable when several pre-aggs exist.
	assert.Contains(t, err.Error(), "sales_daily",
		"the error must name the target that cannot be refreshed")
	// 3. States the consequence: re-applying is a silent no-op. This is the
	//    specific false claim the old comment made, and its absence would let
	//    the old lie back in through a shorter message.
	assert.Contains(t, err.Error(), "IF NOT EXISTS",
		"the error must say why re-applying the DDL does not refresh anything")
	// 4. Offers a remedy the operator can actually execute.
	assert.Contains(t, err.Error(), "materialized_view",
		"the error must name the strategy that does work")
	assert.Contains(t, err.Error(), "drop",
		"the error must offer the manual alternative for operators who keep table targets")
}

// TestRefresh_MaterializedViewIsUnaffected guards against item 1 over-reaching.
// The fix must not have made the supported strategy fail: a materialized view
// must still reach REFRESH MATERIALIZED VIEW.
func TestRefresh_MaterializedViewIsUnaffected(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	sqlxDB := sqlx.NewDb(db, "postgres")

	svc := NewPreAggregationService(sqlxDB, nil, nil)
	svc.starrocksDB = db

	id := uuid.New()
	mock.ExpectQuery(`(?s)FROM catalog_node WHERE id = \$1`).
		WillReturnRows(sqlmock.NewRows([]string{"properties", "config"}).
			AddRow(
				preAggPropsJSON(t, models.PreAggProperties{TargetDatabase: "tenant_a"}),
				preAggConfigJSON(t, models.PreAggConfig{
					Materialization: models.MaterializationConfig{Type: "materialized_view", TargetName: "sales_mv"},
				}),
			))
	mock.ExpectExec("REFRESH MATERIALIZED VIEW").
		WillReturnResult(sqlmock.NewResult(0, 0))

	require.NoError(t, svc.Refresh(context.Background(), id),
		"materialized_view is the one strategy that does refresh and must keep working")
	require.NoError(t, mock.ExpectationsWereMet(),
		"the materialized view path must actually have issued REFRESH MATERIALIZED VIEW")
}

// --- Item 3: partition_by must be refused, never accepted and discarded ---

// TestUpsert_PartitionByIsRefusedNotSilentlyDropped covers the half of item 3
// that can be built without a StarRocks instance.
//
// Emitting a correct PARTITION BY has to be validated against real StarRocks
// first. Until then the field is refused at the API boundary, because the
// defect was never "the DDL lacked PARTITION BY" - it was that a user could set
// it, watch it persist, and receive unpartitioned DDL with no warning anywhere.
func TestUpsert_PartitionByIsRefusedNotSilentlyDropped(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	sqlxDB := sqlx.NewDb(db, "postgres")
	svc := NewPreAggregationService(sqlxDB, nil, nil)

	_, err = svc.UpsertPreAggregation(context.Background(), models.UpsertPreAggRequest{
		TenantID:        "tenant-a",
		BOName:          "sales",
		Name:            "sales_daily",
		Materialization: models.MaterializationConfig{Type: "materialized_view", TargetName: "t", PartitionBy: "trade_date"},
	})

	require.ErrorIs(t, err, ErrPreAggNotSupported,
		"partition_by must be refused as unsupported, not persisted and ignored")

	// The message has to say the request is refused *because it would be
	// ignored*. A bare "unsupported" would let the same silent-acceptance
	// defect back in through a different wording.
	assert.Contains(t, err.Error(), "partition_by")
	assert.Contains(t, err.Error(), "silently ignored",
		"the refusal must name the silent-ignore behaviour it prevents")

	// Nothing may be written: a refused request must not leave a row behind.
	require.NoError(t, mock.ExpectationsWereMet(),
		"a refused upsert must not have written anything to catalog_node")
}

// TestUpdate_PartitionByIsRefusedToo pins the same rule to the second write
// path. Update builds its config identically to Upsert, so validating in only
// one of them would leave PUT /preaggregations/{id} accepting exactly the
// configuration the upsert refuses.
func TestUpdate_PartitionByIsRefusedToo(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	sqlxDB := sqlx.NewDb(db, "postgres")
	svc := NewPreAggregationService(sqlxDB, nil, nil)

	_, err = svc.Update(context.Background(), uuid.New(), models.UpsertPreAggRequest{
		TenantID:        "tenant-a",
		BOName:          "sales",
		Name:            "sales_daily",
		Materialization: models.MaterializationConfig{Type: "materialized_view", TargetName: "t", PartitionBy: "trade_date"},
	})

	require.ErrorIs(t, err, ErrPreAggNotSupported,
		"the update path must refuse partition_by exactly as the upsert path does")
	require.NoError(t, mock.ExpectationsWereMet(),
		"a refused update must not have written anything")
}

// TestPartitionByIsNotDroppedFromTheAPIType is a structural guard. The defect
// was a field that round-tripped through the API while nothing read it. If a
// future change re-adds a materialization knob that the service neither honours
// nor refuses, this test is the thing that should notice - so it asserts the
// *field* still exists and is still a refusal case, rather than asserting a
// count of fields.
func TestPartitionByIsNotDroppedFromTheAPIType(t *testing.T) {
	cfg := models.MaterializationConfig{Type: "table", TargetName: "t", PartitionBy: "trade_date"}

	// It must survive a JSON round trip, or a client could never send it and
	// the refusal would be unreachable.
	raw, err := json.Marshal(cfg)
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"partition_by":"trade_date"`)

	var back models.MaterializationConfig
	require.NoError(t, json.Unmarshal(raw, &back))
	assert.Equal(t, "trade_date", back.PartitionBy)
}

// --- Item 4: the incremental promise ---

// TestIncrementalRefreshStrategyIsRelabelledToInterval covers the migration
// path: configurations that stored "incremental" were already running the
// interval path, because the only branch anywhere is a check for "manual".
// Relabelling is therefore a semantic no-op that stops the stored value from
// continuing to claim something untrue.
func TestIncrementalRefreshStrategyIsRelabelledToInterval(t *testing.T) {
	req := models.UpsertPreAggRequest{
		RefreshStrategy:        "incremental",
		Materialization:        models.MaterializationConfig{Type: "materialized_view", TargetName: "t"},
		RefreshIntervalMinutes: 15,
	}
	require.NoError(t, normaliseMaterializationRequest(&req))
	assert.Equal(t, "interval", req.RefreshStrategy,
		"a legacy incremental strategy must be relabelled, not rejected and not preserved")
}

// TestRefreshStrategyNormalisationLeavesRealStrategiesAlone guards the fix
// against over-reach in the other direction: manual and interval are the two
// strategies that mean something, and neither may be altered.
func TestRefreshStrategyNormalisationLeavesRealStrategiesAlone(t *testing.T) {
	for _, strategy := range []string{"manual", "interval"} {
		req := models.UpsertPreAggRequest{
			RefreshStrategy: strategy,
			Materialization: models.MaterializationConfig{Type: "materialized_view", TargetName: "t"},
		}
		require.NoError(t, normaliseMaterializationRequest(&req))
		assert.Equal(t, strategy, req.RefreshStrategy,
			"strategy %q must pass through untouched", strategy)
	}
}

// TestIncrementalFieldsAreGoneFromTheAPIType is a structural guard against the
// wizard re-introducing a field the backend cannot honour.
//
// It asserts on the struct's declared fields via reflection, NOT on marshalled
// JSON. Marshalling an empty MaterializationConfig cannot see a field marked
// `omitempty`, so a JSON-based version of this assertion passes even when the
// fields are present - verified by mutation, which is how it was found. A
// guard that cannot fail when the defect returns is not a guard.
func TestIncrementalFieldsAreGoneFromTheAPIType(t *testing.T) {
	rt := reflect.TypeOf(models.MaterializationConfig{})

	for _, name := range []string{"IncrementalColumn", "IncrementalWindowDays"} {
		_, present := rt.FieldByName(name)
		assert.False(t, present,
			"MaterializationConfig.%s must not exist while no code honours it; "+
				"its presence is what let the wizard offer a control with no effect", name)
	}

	// partition_by must remain: it is refused by the service, not ignored, and
	// it becomes honoured once the DDL is validated against StarRocks. If this
	// field ever disappears the refusal in normaliseMaterializationRequest
	// becomes unreachable and the defect returns by a different route.
	_, present := rt.FieldByName("PartitionBy")
	assert.True(t, present,
		"MaterializationConfig.PartitionBy must stay so the service can refuse it explicitly")
}

// TestLegacyIncrementalConfigIsDroppedOnReParse checks the storage side. A row
// written before this change still carries incremental_column in its config
// JSON; parsing it must not resurrect the field as something the service will
// then act on.
func TestLegacyIncrementalConfigIsDroppedOnReParse(t *testing.T) {
	legacy := []byte(`{"terms":["a"],"materialization":{"type":"table","target_name":"t",` +
		`"incremental_column":"trade_date","incremental_window_days":2}}`)

	cfg, err := models.ParsePreAggConfig(legacy)
	require.NoError(t, err)
	assert.Equal(t, "table", cfg.Materialization.Type)

	// Re-marshalling must not carry the retired fields forward.
	raw, err := json.Marshal(cfg)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "incremental_column")
	assert.NotContains(t, string(raw), "incremental_window_days")
}

// --- The rule the whole change rests on ---

// TestNoAcceptedConfigurationIsSilentlyDiscarded is the invariant test. For
// every materialization configuration the API can express, the service must
// either honour it or refuse it - and "silently discard" is not available as a
// third option. This is written as a table over inputs rather than as a count,
// so adding a case makes the test stronger instead of making it stale.
func TestNoAcceptedConfigurationIsSilentlyDiscarded(t *testing.T) {
	cases := []struct {
		name        string
		mat         models.MaterializationConfig
		refuseAsErr bool
	}{
		{"plain materialized view", models.MaterializationConfig{Type: "materialized_view", TargetName: "t"}, false},
		{"plain table", models.MaterializationConfig{Type: "table", TargetName: "t"}, false},
		{"partitioned materialized view", models.MaterializationConfig{Type: "materialized_view", TargetName: "t", PartitionBy: "d"}, true},
		{"partitioned table", models.MaterializationConfig{Type: "table", TargetName: "t", PartitionBy: "d"}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := models.UpsertPreAggRequest{
				TenantID:        "tenant-a",
				Materialization: tc.mat,
			}
			err := normaliseMaterializationRequest(&req)

			if tc.refuseAsErr {
				require.ErrorIs(t, err, ErrPreAggNotSupported,
					"a configuration the service cannot honour must be refused outright")
				return
			}
			require.NoError(t, err, "a supported configuration must pass")
			// Passing validation must mean the configuration is carried
			// through, not quietly reduced on the way in.
			assert.Equal(t, tc.mat, req.Materialization,
				"an accepted configuration must survive normalisation unchanged")
		})
	}
}
