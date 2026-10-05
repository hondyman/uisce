package querybuilder

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const routerTenant = "11111111-1111-1111-1111-111111111111"

func routerFixture(t *testing.T) (*CubeRouter, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewCubeRouter(sqlx.NewDb(db, "postgres")), mock
}

func mustJSON(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

func nullJSON(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	b, _ := json.Marshal(v)
	return b
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// expectCubeQuery wires the cube_definition lookup.
func expectCubeQuery(t *testing.T, mock sqlmock.Sqlmock, cube *CubeDefinition) {
	t.Helper()
	rows := sqlmock.NewRows([]string{
		"id", "tenant_id", "name", "description", "bo_id", "dimensions",
		"time_dimension", "metric_ids", "grains", "materialization",
		"federation", "contract_version",
		"content_hash", "is_core", "status", "archived_at", "created_by",
		"created_at", "updated_at",
	})
	if cube != nil {
		ver := cube.ContractVersion
		if ver < 1 {
			ver = 1
		}
		rows.AddRow(
			cube.ID, routerTenant, cube.Name, cube.Description, cube.BOID,
			mustJSON(t, cube.Dimensions), nullJSON(cube.TimeDimension),
			mustJSON(t, cube.MetricIDs), mustJSON(t, cube.Grains),
			mustJSON(t, cube.Materialization), mustJSON(t, cube.Federation), ver,
			nullStr(cube.ContentHash),
			cube.IsCore, cube.Status, nil, nil,
			time.Now().UTC(), time.Now().UTC(),
		)
	}
	mock.ExpectQuery(`FROM data_explorer.cube_definition`).
		WithArgs(routerTenant, "bo_sales").
		WillReturnRows(rows)
}

// expectMetricQuery wires the metric_definition lookup.
func expectMetricQuery(t *testing.T, mock sqlmock.Sqlmock, metrics ...MetricDefinition) {
	t.Helper()
	rows := sqlmock.NewRows([]string{
		"id", "tenant_id", "name", "description", "bo_id", "catalog_term_id",
		"expression", "grain_allowlist", "format_config", "variables",
		"materialization_config", "decomposable", "content_hash", "tags",
		"is_core", "status", "archived_at", "created_by", "created_at", "updated_at",
	})
	for _, m := range metrics {
		rows.AddRow(
			uuid.New(), routerTenant, m.Name, m.Description, m.BOID,
			nil, mustJSON(t, m.Expression), mustJSON(t, m.GrainAllowlist),
			mustJSON(t, m.FormatConfig), mustJSON(t, m.Variables),
			mustJSON(t, m.MaterializationConfig), m.Decomposable,
			nullStr(m.ContentHash), pq.Array(m.Tags), m.IsCore,
			"active", nil, nil, time.Now().UTC(), time.Now().UTC(),
		)
	}
	mock.ExpectQuery(`FROM data_explorer.metric_definition`).
		WillReturnRows(rows)
}

// expectMaterialization wires the catalog_node lifecycle lookup.
func expectMaterialization(t *testing.T, mock sqlmock.Sqlmock, nodeName string, status string, refreshed *time.Time) {
	t.Helper()
	p := models.PreAggProperties{
		BOName: "sales", TenantID: routerTenant, Dialect: "starrocks",
		RefreshStrategy: "interval", LifecycleStatus: status,
		LastRefreshedAt: refreshed,
	}
	mock.ExpectQuery(`FROM catalog_node n`).
		WithArgs(nodeName).
		WillReturnRows(sqlmock.NewRows([]string{"properties"}).
			AddRow(mustJSON(t, p)))
}

// expectStaleMaterialization wires a catalog_node the scheduler has marked
// stale. Stale is servable-but-flagged, so the lifecycle lookup succeeds and
// the router proceeds to the stalePolicy decision.
func expectStaleMaterialization(t *testing.T, mock sqlmock.Sqlmock, nodeName string) {
	t.Helper()
	p := models.PreAggProperties{
		BOName: "sales", TenantID: routerTenant, Dialect: "starrocks",
		RefreshStrategy: "interval", LifecycleStatus: models.LifecycleStale,
	}
	mock.ExpectQuery(`FROM catalog_node n`).
		WithArgs(nodeName).
		WillReturnRows(sqlmock.NewRows([]string{"properties"}).
			AddRow(mustJSON(t, p)))
}

func routedCube() CubeDefinition {
	return CubeDefinition{
		ID:              uuid.New().String(),
		TenantID:        routerTenant,
		Name:            "sales_cube",
		BOID:            "bo_sales",
		MetricIDs:       []string{"m_revenue", "m_units"},
		Dimensions:      []CubeDimension{{TermNodeID: "country"}, {TermNodeID: "order_date"}},
		TimeDimension:   &CubeTimeDimension{TermNodeID: "order_date", DefaultGrain: "day"},
		Grains:          [][]string{{"country", "order_date"}},
		ContentHash:     "abc123",
		IsCore:          false,
		Status:          "active",
		Materialization: CubeMaterializationConfig{StalePolicy: "serve_with_flag"},
	}
}

// countryMonthQuery groups by the cube's time dimension at its own grain, so
// no time rollup occurs and a distributive metric is servable. Tests that want
// to exercise a rollup omit the time dimension deliberately.
func countryMonthQuery() *boresolver.QueryDef {
	return &boresolver.QueryDef{
		Context: boresolver.QueryContext{BOID: "bo_sales", TenantID: routerTenant},
		Query: boresolver.QueryRequest{
			Dimensions: []boresolver.DimensionDef{
				{TermNodeID: "country", Alias: "country"},
				{TermNodeID: "order_date", Alias: "order_date"},
			},
			Measures: []boresolver.MeasureDef{{TermNodeID: "m_revenue", Alias: "revenue", Aggregation: "sum"}},
		},
	}
}

func freshClock() *time.Time {
	t := time.Now().UTC()
	return &t
}

// TestCubeRouter_HitOnMatchingGrain is the happy path: a request whose
// dimensions are covered by a declared grain, with an active fresh
// materialization, routes.
func TestCubeRouter_HitOnMatchingGrain(t *testing.T) {
	r, mock := routerFixture(t)
	cube := routedCube()
	r.SetClock(func() time.Time { return time.Now().UTC() })

	expectCubeQuery(t, mock, &cube)
	expectMetricQuery(t, mock, revenueMetric(), unitsMetric())
	expectMaterialization(t, mock, CubeMaterializationName(routerTenant, false, cube, []string{"country", "order_date"}),
		models.LifecycleActive, freshClock())

	d := r.Route(context.Background(), routerTenant, countryMonthQuery(), nil)
	require.NotNil(t, d.Route, "expected a route, got miss=%s detail=%s", d.MissReason, d.Detail)
	assert.Equal(t, "sales_cube", d.Route.CubeName)
	assert.False(t, d.Route.Stale)
	assert.Contains(t, d.Route.Grain, "country")
}

// TestCubeRouter_ABACBelowGrainBlocks is acceptance test #2, the security
// gate: a caller restricted below the materialization grain must NOT be served
// by it, because the materialization has already aggregated that dimension
// away. This is asserted as a routing refusal, not as a boolean flag.
func TestCubeRouter_ABACBelowGrainBlocks(t *testing.T) {
	r, mock := routerFixture(t)
	cube := routedCube()
	r.SetClock(func() time.Time { return time.Now().UTC() })

	expectCubeQuery(t, mock, &cube)
	expectMetricQuery(t, mock, revenueMetric(), unitsMetric())
	// Deliberately expect NO catalog_node lookup: the ABAC check must happen
	// before any materialization is considered served.

	d := r.Route(context.Background(), routerTenant, countryMonthQuery(), []string{"account_id"})
	require.Nil(t, d.Route, "a sub-grain restriction must prevent serving")
	assert.Equal(t, CubeMissABACBelowGrain, d.MissReason)
	assert.Contains(t, d.Detail, "account_id")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCubeRouter_ABACAtOrAboveGrainAllows verifies the check is not blanket
// denial: a restriction at the materialization's own grain is safe.
func TestCubeRouter_ABACAtOrAboveGrainAllows(t *testing.T) {
	r, mock := routerFixture(t)
	cube := routedCube()
	r.SetClock(func() time.Time { return time.Now().UTC() })

	expectCubeQuery(t, mock, &cube)
	expectMetricQuery(t, mock, revenueMetric(), unitsMetric())
	expectMaterialization(t, mock, CubeMaterializationName(routerTenant, false, cube, []string{"country", "order_date"}),
		models.LifecycleActive, freshClock())

	d := r.Route(context.Background(), routerTenant, countryMonthQuery(), []string{"country"})
	require.NotNil(t, d.Route, "a restriction at the materialization grain is safe: %s", d.Detail)
}

// TestCubeRouter_StalePolicy covers acceptance test #4.
func TestCubeRouter_StalePolicy(t *testing.T) {
	t.Run("serve_with_flag serves stale data with a flag", func(t *testing.T) {
		r, mock := routerFixture(t)
		cube := routedCube() // serve_with_flag
		cube.Materialization.StalePolicy = "serve_with_flag"
		r.SetClock(func() time.Time { return time.Now().UTC() })

		expectCubeQuery(t, mock, &cube)
		expectMetricQuery(t, mock, revenueMetric(), unitsMetric())
		// Staleness is status-driven: the scheduler marks a materialization
		// stale, so an active-but-stale node is the case under test.
		expectStaleMaterialization(t, mock,
			CubeMaterializationName(routerTenant, false, cube, []string{"country", "order_date"}))

		d := r.Route(context.Background(), routerTenant, countryMonthQuery(), nil)
		require.NotNil(t, d.Route, "%s", d.Detail)
		assert.True(t, d.Route.Stale, "stale data must be flagged, not silently served")
	})

	t.Run("force_raw_fallback declines stale data", func(t *testing.T) {
		r, mock := routerFixture(t)
		cube := routedCube()
		cube.Materialization.StalePolicy = "force_raw_fallback"
		r.SetClock(func() time.Time { return time.Now().UTC() })

		expectCubeQuery(t, mock, &cube)
		expectMetricQuery(t, mock, revenueMetric(), unitsMetric())
		// The lifecycle lookup happens before the freshness decision, so a
		// stale-but-active materialization is still read and then declined.
		expectStaleMaterialization(t, mock,
			CubeMaterializationName(routerTenant, false, cube, []string{"country", "order_date"}))

		d := r.Route(context.Background(), routerTenant, countryMonthQuery(), nil)
		assert.Nil(t, d.Route)
		assert.Equal(t, CubeMissStale, d.MissReason)
	})

	t.Run("never-refreshed counts as stale", func(t *testing.T) {
		r, mock := routerFixture(t)
		cube := routedCube()
		r.SetClock(func() time.Time { return time.Now().UTC() })

		expectCubeQuery(t, mock, &cube)
		expectMetricQuery(t, mock, revenueMetric(), unitsMetric())
		expectMaterialization(t, mock, CubeMaterializationName(routerTenant, false, cube, []string{"country", "order_date"}),
			models.LifecycleActive, nil)

		d := r.Route(context.Background(), routerTenant, countryMonthQuery(), nil)
		require.NotNil(t, d.Route)
		assert.True(t, d.Route.Stale, "a zero refresh timestamp is stale by definition")
	})
}

// TestCubeRouter_NonDecomposableRollupRefused is acceptance test #3: AVG and
// derived metrics cannot be served by rolling up a finer materialization,
// because averaging averages is not a valid rollup.
func TestCubeRouter_NonDecomposableRollupRefused(t *testing.T) {
	r, mock := routerFixture(t)
	cube := routedCube()
	r.SetClock(func() time.Time { return time.Now().UTC() })

	avg := MetricDefinition{
		ID:             "m_revenue",
		Name:           "Avg Revenue",
		BOID:           "bo_sales",
		Expression:     MetricExpression{Kind: "aggregation", Fn: "avg", TermNodeID: "revenue"},
		GrainAllowlist: []string{"country", "order_date"},
		Decomposable:   false, // DeriveDecomposable: avg is not distributive
	}

	expectCubeQuery(t, mock, &cube)
	expectMetricQuery(t, mock, avg, unitsMetric())

	// Drop the time dimension: collapsing a day-grain materialization to a
	// total IS a time rollup, which is exactly what AVG cannot support.
	qd := countryMonthQuery()
	qd.Query.Dimensions = []boresolver.DimensionDef{{TermNodeID: "country", Alias: "country"}}

	d := r.Route(context.Background(), routerTenant, qd, nil)
	assert.Nil(t, d.Route, "a non-decomposable metric must not be rolled up")
	assert.Equal(t, CubeMissNotDecomposed, d.MissReason)
}

// TestCubeRouter_StaleMaterializationNotServed verifies an inactive
// materialization is not routed to, and a missing node is treated as inactive
// rather than an error.
func TestCubeRouter_StaleMaterializationNotServed(t *testing.T) {
	t.Run("non-active lifecycle is declined", func(t *testing.T) {
		r, mock := routerFixture(t)
		cube := routedCube()
		r.SetClock(func() time.Time { return time.Now().UTC() })

		expectCubeQuery(t, mock, &cube)
		expectMetricQuery(t, mock, revenueMetric(), unitsMetric())
		expectMaterialization(t, mock, CubeMaterializationName(routerTenant, false, cube, []string{"country", "order_date"}),
			models.LifecycleMaterializing, freshClock())

		d := r.Route(context.Background(), routerTenant, countryMonthQuery(), nil)
		assert.Nil(t, d.Route)
		assert.Equal(t, CubeMissNotActive, d.MissReason)
	})

	t.Run("missing catalog node is not an error", func(t *testing.T) {
		r, mock := routerFixture(t)
		cube := routedCube()
		r.SetClock(func() time.Time { return time.Now().UTC() })

		expectCubeQuery(t, mock, &cube)
		expectMetricQuery(t, mock, revenueMetric(), unitsMetric())
		mock.ExpectQuery(`FROM catalog_node n`).
			WithArgs(CubeMaterializationName(routerTenant, false, cube, []string{"country", "order_date"})).
			WillReturnRows(sqlmock.NewRows([]string{"properties"}))

		d := r.Route(context.Background(), routerTenant, countryMonthQuery(), nil)
		assert.Nil(t, d.Route)
		assert.Equal(t, CubeMissNotActive, d.MissReason,
			"an undeployed materialization is a miss, not a failure")
	})
}

// TestCubeRouter_NeverFailsTheQuery is acceptance test #6: every internal
// error degrades to "no route" so the base BO path still serves.
func TestCubeRouter_NeverFailsTheQuery(t *testing.T) {
	t.Run("cube lookup error", func(t *testing.T) {
		r, mock := routerFixture(t)
		mock.ExpectQuery(`FROM data_explorer.cube_definition`).
			WillReturnError(assert.AnError)

		d := r.Route(context.Background(), routerTenant, countryMonthQuery(), nil)
		assert.Nil(t, d.Route)
		assert.Equal(t, CubeMissError, d.MissReason)
	})

	t.Run("metric lookup error", func(t *testing.T) {
		r, mock := routerFixture(t)
		cube := routedCube()
		expectCubeQuery(t, mock, &cube)
		mock.ExpectQuery(`FROM data_explorer.metric_definition`).
			WillReturnError(assert.AnError)

		d := r.Route(context.Background(), routerTenant, countryMonthQuery(), nil)
		assert.Nil(t, d.Route)
		assert.Equal(t, CubeMissError, d.MissReason)
	})

	t.Run("materialization lookup error", func(t *testing.T) {
		r, mock := routerFixture(t)
		cube := routedCube()
		r.SetClock(func() time.Time { return time.Now().UTC() })
		expectCubeQuery(t, mock, &cube)
		expectMetricQuery(t, mock, revenueMetric(), unitsMetric())
		mock.ExpectQuery(`FROM catalog_node n`).WillReturnError(assert.AnError)

		d := r.Route(context.Background(), routerTenant, countryMonthQuery(), nil)
		assert.Nil(t, d.Route)
		assert.Equal(t, CubeMissError, d.MissReason)
	})

	t.Run("nil router and nil query are safe", func(t *testing.T) {
		var nilRouter *CubeRouter
		assert.Nil(t, nilRouter.Route(context.Background(), routerTenant, countryMonthQuery(), nil).Route)

		r, _ := routerFixture(t)
		assert.Nil(t, r.Route(context.Background(), routerTenant, nil, nil).Route)
	})
}

// TestCubeRouter_NoCubeDeployed covers acceptance test #5's premise: with no
// cube, the router simply declines and never queries anything else.
func TestCubeRouter_NoCubeDeployed(t *testing.T) {
	r, mock := routerFixture(t)
	expectCubeQuery(t, mock, nil)

	d := r.Route(context.Background(), routerTenant, countryMonthQuery(), nil)
	assert.Nil(t, d.Route)
	assert.Equal(t, CubeMissNoCube, d.MissReason)
	assert.NoError(t, mock.ExpectationsWereMet(),
		"a missing cube must not trigger metric or materialization lookups")
}

// TestCubeRouter_MetricNotInCube verifies a request for a measure the cube does
// not publish is declined.
func TestCubeRouter_MetricNotInCube(t *testing.T) {
	r, mock := routerFixture(t)
	cube := routedCube()
	expectCubeQuery(t, mock, &cube)
	expectMetricQuery(t, mock, revenueMetric(), unitsMetric())

	qd := countryMonthQuery()
	qd.Query.Measures = append(qd.Query.Measures,
		boresolver.MeasureDef{TermNodeID: "m_margin", Alias: "margin", Aggregation: "sum"})

	d := r.Route(context.Background(), routerTenant, qd, nil)
	assert.Nil(t, d.Route)
	assert.Equal(t, CubeMissMetricSubset, d.MissReason)
}

// TestCubeRouter_GrainTooCoarseIsDeclined documents the ADR-015 limitation: a
// request for a dimension no grain covers falls through to base tables.
func TestCubeRouter_GrainTooCoarseIsDeclined(t *testing.T) {
	r, mock := routerFixture(t)
	cube := routedCube()
	expectCubeQuery(t, mock, &cube)
	expectMetricQuery(t, mock, revenueMetric(), unitsMetric())

	qd := countryMonthQuery()
	qd.Query.Dimensions = append(qd.Query.Dimensions,
		boresolver.DimensionDef{TermNodeID: "product", Alias: "product"})

	d := r.Route(context.Background(), routerTenant, qd, nil)
	assert.Nil(t, d.Route)
	assert.Equal(t, CubeMissNoGrainMatch, d.MissReason,
		"a request finer than any declared grain must fall through")
}

// TestCubeRouter_PrefersTightestGrain verifies that when several grains cover
// the request, the tightest (smallest) is chosen, deterministically.
func TestCubeRouter_PrefersTightestGrain(t *testing.T) {
	r, mock := routerFixture(t)
	cube := routedCube()
	cube.Grains = [][]string{
		{"country", "product", "order_date"},
		{"country", "order_date"},
	}
	cube.Dimensions = []CubeDimension{{TermNodeID: "country"}, {TermNodeID: "order_date"}}
	r.SetClock(func() time.Time { return time.Now().UTC() })

	tight := []string{"country", "order_date"}
	expectCubeQuery(t, mock, &cube)
	expectMetricQuery(t, mock, revenueMetric(), unitsMetric())
	expectMaterialization(t, mock, CubeMaterializationName(routerTenant, false, cube, tight),
		models.LifecycleActive, freshClock())

	d := r.Route(context.Background(), routerTenant, countryMonthQuery(), nil)
	require.NotNil(t, d.Route, "%s", d.Detail)
	assert.Equal(t, CubeMaterializationName(routerTenant, false, cube, tight), d.Route.Materialization,
		"the tightest covering grain must win")
}
