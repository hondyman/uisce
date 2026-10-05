package querybuilder

import (
	"context"
	"strings"
	"testing"

	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// positionAccountResolver is the CUBE-2.2 proof fixture: Position ⋈ Account
// on the shared semantic term account_number, each bound to its driving table.
func positionAccountResolver() MapFederationBindingResolver {
	return MapFederationBindingResolver{
		Tables: map[string]string{
			"oms.position": "oms.position",
			"oms.account":  "oms.account",
		},
		Columns: map[string]string{
			"oms.position|account_number": "account_number",
			"oms.account|account_number":  "account_number",
			"oms.position|as_of_date":     "as_of_date",
			"oms.position|market_value":   "market_value",
		},
	}
}

func TestCompileFederationJoinSQL_PositionAccountCommonKey(t *testing.T) {
	cube := sampleFederatedCube()
	compiled, err := CompileFederationJoinSQL(
		context.Background(),
		"99e99e99-99e9-49e9-89e9-99e99e99e999",
		cube.Federation,
		positionAccountResolver(),
	)
	require.NoError(t, err)
	require.NotNil(t, compiled)

	assert.Contains(t, compiled.FromSQL, "oms.position AS pos")
	assert.Contains(t, compiled.FromSQL, "INNER JOIN oms.account AS acct ON")
	assert.Contains(t, compiled.FromSQL, "pos.account_number = acct.account_number")
	require.Len(t, compiled.Joins, 1)
	assert.Equal(t, "common", compiled.Joins[0].KeyKind)
	assert.Equal(t, []string{"pos.account_number = acct.account_number"}, compiled.Joins[0].OnPredicates)

	assert.Equal(t, "pos.account_number", compiled.DimExpr("account_number", "oms.position"))
	assert.Equal(t, "acct.account_number", compiled.DimExpr("account_number", "oms.account"))

	require.NoError(t, compiled.EnrichGrainTerms(
		context.Background(), "t", "oms.position",
		[]string{"account_number", "as_of_date"},
		positionAccountResolver(),
	))
	dims := compiled.DimExprsForGrain([]string{"account_number", "as_of_date"}, "oms.position")
	assert.Equal(t, "pos.account_number", dims["account_number"])
	assert.Equal(t, "pos.as_of_date", dims["as_of_date"])
}

func TestCompileFederationJoinSQL_TransformKey(t *testing.T) {
	f := sampleFederatedCube().Federation
	f.Joins[0].KeyKind = "transform"
	f.Joins[0].TransformTermID = "account_number"
	// ValidateCubeFederation still requires parallel term id lists on transform edges.

	compiled, err := CompileFederationJoinSQL(context.Background(), "t", f, positionAccountResolver())
	require.NoError(t, err)
	assert.Contains(t, compiled.FromSQL, "pos.account_number = acct.account_number")
	assert.Equal(t, "transform", compiled.Joins[0].KeyKind)
}

func TestCompileFederationJoinSQL_UnboundTerm(t *testing.T) {
	f := sampleFederatedCube().Federation
	f.Joins[0].LeftTermIDs = []string{"missing_term"}
	f.Joins[0].RightTermIDs = []string{"account_number"}
	_, err := CompileFederationJoinSQL(context.Background(), "t", f, positionAccountResolver())
	require.ErrorIs(t, err, ErrCubeFederationCompile)
	assert.Contains(t, err.Error(), "missing_term")
}

func TestCompileFederationJoinSQL_EmptyRejected(t *testing.T) {
	_, err := CompileFederationJoinSQL(context.Background(), "t", CubeFederation{}, positionAccountResolver())
	require.ErrorIs(t, err, ErrCubeFederationCompile)
}

func TestGenerateFederatedCubeMaterializationDDL_PositionAccount(t *testing.T) {
	cube := sampleFederatedCube()
	cube.Name = "position_valuation"
	cube.MetricIDs = []string{"m_units"}
	cube.Grains = [][]string{{"account_number", "as_of_date"}}
	cube.Dimensions = []CubeDimension{
		{TermNodeID: "account_number"},
		{TermNodeID: "as_of_date"},
	}
	cube.TimeDimension = &CubeTimeDimension{TermNodeID: "as_of_date", DefaultGrain: "day"}

	resolver := positionAccountResolver()
	compiled, err := CompileFederationJoinSQL(context.Background(), "tenant_a", cube.Federation, resolver)
	require.NoError(t, err)
	grain := []string{"account_number", "as_of_date"}
	require.NoError(t, compiled.EnrichGrainTerms(context.Background(), "tenant_a", cube.BOID, grain, resolver))

	// Gate over terms the units metric + grain dims need.
	bo := &boresolver.BODefinition{ID: "oms.position", DrivingTable: "oms.position"}
	for _, term := range []string{"units_sold", "account_number", "as_of_date"} {
		bo.Fields = append(bo.Fields, boresolver.BOField{
			ID: "f_" + term, Name: term, PhysicalColumn: "oms.position." + term,
		})
	}
	gen := NewCubeDDLGenerator("starrocks")
	gen.SetTermGate(NewSensitivityTermGate(bo, "admin", "", nil))

	metric := unitsMetric()
	metric.GrainAllowlist = []string{"account_number", "as_of_date"}
	lookup := map[string]MetricDefinition{"m_units": metric}

	got, err := gen.GenerateFederatedCubeMaterializationDDL(
		"tenant_a", false, cube,
		grain,
		compiled.FromSQL,
		compiled.DimExprsForGrain(grain, cube.BOID),
		lookup, nil,
	)
	require.NoError(t, err)

	assert.Contains(t, got.DDL, "FROM oms.position AS pos")
	assert.Contains(t, got.DDL, "INNER JOIN oms.account AS acct ON")
	assert.Contains(t, got.DDL, "pos.account_number = acct.account_number")
	assert.Contains(t, got.DDL, "pos.account_number AS account_number")
	assert.Contains(t, got.DDL, "pos.as_of_date AS as_of_date")
	assert.Contains(t, got.DDL, "GROUP BY pos.account_number, pos.as_of_date")
	assert.Contains(t, strings.ToLower(got.DDL), "units_sold")
	assert.NotContains(t, got.DDL, "CUBE-2.2")
}
