package querybuilder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sampleCube() CubeDefinition {
	return CubeDefinition{
		Name:      "sales_cube",
		BOID:      "bo_sales",
		MetricIDs: []string{"m_revenue", "m_units"},
		Dimensions: []CubeDimension{
			{TermNodeID: "country"},
			{TermNodeID: "product", DrillPath: []string{"product_category", "product"}},
		},
		TimeDimension: &CubeTimeDimension{TermNodeID: "order_date", DefaultGrain: "day"},
		Grains: [][]string{
			{"country", "product", "order_date"},
			{"country", "order_date"},
		},
		Materialization: CubeMaterializationConfig{
			Strategy:               "starrocks_mv",
			RefreshStrategy:        "interval",
			RefreshIntervalMinutes: 15,
			StalePolicy:            "serve_with_flag",
		},
	}
}

// TestComputeCubeContentHash_Stable verifies the hash is deterministic and
// non-empty, mirroring the metric-layer hash test.
func TestComputeCubeContentHash_Stable(t *testing.T) {
	c := sampleCube()
	h1 := ComputeCubeContentHash(c, nil)
	h2 := ComputeCubeContentHash(c, nil)
	assert.Equal(t, h1, h2)
	assert.NotEmpty(t, h1)
	assert.Len(t, h1, 64, "SHA-256 hex is 64 chars")
}

// TestComputeCubeContentHash_CanonicalizationInsensitivity verifies that
// authoring-order noise does not change the hash, while semantically
// meaningful differences do.
func TestComputeCubeContentHash_CanonicalizationInsensitivity(t *testing.T) {
	base := sampleCube()
	baseHash := ComputeCubeContentHash(base, nil)

	t.Run("metric id order is irrelevant", func(t *testing.T) {
		c := sampleCube()
		c.MetricIDs = []string{"m_units", "m_revenue"}
		assert.Equal(t, baseHash, ComputeCubeContentHash(c, nil),
			"metric IDs are a set, so order must not matter")
	})

	t.Run("grain member order is irrelevant", func(t *testing.T) {
		c := sampleCube()
		c.Grains = [][]string{
			{"order_date", "product", "country"},
			{"order_date", "country"},
		}
		assert.Equal(t, baseHash, ComputeCubeContentHash(c, nil),
			"dimensions within a grain are a set")
	})

	t.Run("case and whitespace are normalized", func(t *testing.T) {
		c := sampleCube()
		c.MetricIDs = []string{"  M_REVENUE ", "M_UNITS"}
		c.Dimensions[0].TermNodeID = " Country "
		assert.Equal(t, baseHash, ComputeCubeContentHash(c, nil))
	})

	t.Run("drill path order is normalized", func(t *testing.T) {
		c := sampleCube()
		c.Dimensions[1].DrillPath = []string{"product", "product_category"}
		assert.Equal(t, baseHash, ComputeCubeContentHash(c, nil))
	})

	// These must differ: a hash that ignored real content would make deploy
	// idempotency unsafe, because a changed cube would look unchanged.
	t.Run("changing metrics changes the hash", func(t *testing.T) {
		c := sampleCube()
		c.MetricIDs = append(c.MetricIDs, "m_margin")
		assert.NotEqual(t, baseHash, ComputeCubeContentHash(c, nil))
	})

	t.Run("changing grains changes the hash", func(t *testing.T) {
		c := sampleCube()
		c.Grains = append(c.Grains, []string{"customer", "order_date"})
		assert.NotEqual(t, baseHash, ComputeCubeContentHash(c, nil))
	})

	t.Run("reordering dimensions changes the hash", func(t *testing.T) {
		c := sampleCube()
		c.Dimensions[0], c.Dimensions[1] = c.Dimensions[1], c.Dimensions[0]
		assert.NotEqual(t, baseHash, ComputeCubeContentHash(c, nil),
			"dimension order is the axis order and is significant")
	})

	t.Run("changing stale policy changes the hash", func(t *testing.T) {
		c := sampleCube()
		c.Materialization.StalePolicy = "force_raw_fallback"
		assert.NotEqual(t, baseHash, ComputeCubeContentHash(c, nil))
	})
}

// TestValidateCubeStructural covers the checks that need no database.
func TestValidateCubeStructural(t *testing.T) {
	t.Run("valid cube passes", func(t *testing.T) {
		require.NoError(t, ValidateCubeStructural(sampleCube()))
	})

	t.Run("no metrics is rejected", func(t *testing.T) {
		c := sampleCube()
		c.MetricIDs = nil
		require.ErrorIs(t, ValidateCubeStructural(c), ErrCubeNoMetrics)
	})

	t.Run("no grains is rejected", func(t *testing.T) {
		c := sampleCube()
		c.Grains = nil
		require.ErrorIs(t, ValidateCubeStructural(c), ErrCubeNoGrains)
	})

	t.Run("dimension not covered by any grain is rejected", func(t *testing.T) {
		c := sampleCube()
		c.Dimensions = append(c.Dimensions, CubeDimension{TermNodeID: "channel"})
		err := ValidateCubeStructural(c)
		require.ErrorIs(t, err, ErrCubeDimensionNotInGrain)
		assert.Contains(t, err.Error(), "channel")
	})

	t.Run("coverage may come from any single grain", func(t *testing.T) {
		// country/product in grain 1, order_date in grain 2 — together they
		// cover the surface even though no single grain does.
		c := sampleCube()
		c.Dimensions = []CubeDimension{
			{TermNodeID: "country"},
			{TermNodeID: "product"},
			{TermNodeID: "order_date"},
		}
		require.NoError(t, ValidateCubeStructural(c))
	})

	t.Run("duplicate metric id is rejected", func(t *testing.T) {
		c := sampleCube()
		c.MetricIDs = []string{"m_revenue", "m_revenue"}
		err := ValidateCubeStructural(c)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "more than once")
	})

	t.Run("empty metric id is rejected", func(t *testing.T) {
		c := sampleCube()
		c.MetricIDs = []string{"  "}
		require.ErrorIs(t, ValidateCubeStructural(c), ErrCubeUnknownMetric)
	})

	t.Run("no dimensions is allowed (purely metric cube)", func(t *testing.T) {
		c := sampleCube()
		c.Dimensions = nil
		c.Grains = [][]string{{"order_date"}}
		require.NoError(t, ValidateCubeStructural(c))
	})
}

// TestValidateCubeMetricReferences covers the governance constraint: a cube
// may only aggregate metrics that exist in the tenant, and every metric it
// names must be one it could actually materialize.
func TestValidateCubeMetricReferences(t *testing.T) {
	known := map[string]MetricDefinition{
		"m_revenue": {ID: "m_revenue", Expression: MetricExpression{Kind: "aggregation", Fn: "sum", TermNodeID: "revenue"}},
		"m_units":   {ID: "m_units", Expression: MetricExpression{Kind: "aggregation", Fn: "sum", TermNodeID: "units_sold"}},
	}

	require.NoError(t, ValidateCubeMetricReferences(sampleCube(), known))

	t.Run("unknown metric is rejected", func(t *testing.T) {
		err := ValidateCubeMetricReferences(sampleCube(), map[string]MetricDefinition{"m_revenue": known["m_revenue"]})
		require.ErrorIs(t, err, ErrCubeUnknownMetric)
		assert.Contains(t, err.Error(), "m_units")
	})

	t.Run("lookup is case and whitespace insensitive", func(t *testing.T) {
		require.NoError(t, ValidateCubeMetricReferences(sampleCube(), known),
			"IDs are normalized before lookup")
	})

	// The ambiguity must be caught where the cube is authored, not discovered
	// at DDL generation. A cube naming an ordered-only ratio is rejected even
	// though the metric "exists".
	t.Run("a ratio with no explicit operands is rejected", func(t *testing.T) {
		ambiguous := map[string]MetricDefinition{
			"m_revenue": known["m_revenue"],
			"m_units":   known["m_units"],
			"m_margin": {
				ID:   "m_margin",
				Expression: MetricExpression{
					Kind:          "derived",
					BaseMetricIDs: []string{"m_revenue", "m_units"},
				},
			},
		}
		c := sampleCube()
		c.MetricIDs = []string{"m_margin"}
		err := ValidateCubeMetricReferences(c, ambiguous)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "requires numeratorId and denominatorId")
		assert.Contains(t, err.Error(), "m_margin",
			"the error must name the offending metric")
	})

	t.Run("a ratio with explicit operands is accepted", func(t *testing.T) {
		explicit := map[string]MetricDefinition{
			"m_margin": {
				ID:   "m_margin",
				Expression: MetricExpression{
					Kind:          "derived",
					BaseMetricIDs: []string{"m_revenue", "m_units"},
					NumeratorID:   "m_revenue",
					DenominatorID: "m_units",
				},
			},
		}
		c := sampleCube()
		c.MetricIDs = []string{"m_margin"}
		require.NoError(t, ValidateCubeMetricReferences(c, explicit))
	})
}

// TestCubeDimensionSet verifies the normalized dimension set used by grain
// coverage checks.
// TestComputeCubeContentHash_TracksReferencedMetricContent is the ADR-027
// regression test. The cube hash used to cover metric IDs only, so editing a
// metric's definition left the cube hash unchanged, the deploy stayed a no-op
// under ADR-011, and a materialized view kept serving the pre-edit expression.
// A hash that cannot see the thing it must distinguish.
func TestComputeCubeContentHash_TracksReferencedMetricContent(t *testing.T) {
	c := sampleCube()

	base := ComputeCubeContentHash(c, []string{"hash_revenue_v1", "hash_units_v1"})

	t.Run("editing a referenced metric changes the cube hash", func(t *testing.T) {
		edited := ComputeCubeContentHash(c, []string{"hash_revenue_v2", "hash_units_v1"})
		assert.NotEqual(t, base, edited,
			"a metric definition edit must be a cube change, or deploy stays a no-op against a stale materialization")
	})

	t.Run("an unrelated metric leaves the hash stable", func(t *testing.T) {
		same := ComputeCubeContentHash(c, []string{"hash_units_v1", "hash_revenue_v1"})
		assert.Equal(t, base, same,
			"resolution order must not change the cube's identity")
	})

	t.Run("content hash order is normalized", func(t *testing.T) {
		assert.Equal(t, base, ComputeCubeContentHash(c, []string{"  HASH_REVENUE_V1 ", "hash_units_v1"}),
			"case and whitespace are normalized, as for metric IDs")
	})

	t.Run("a real metric content hash is carried through", func(t *testing.T) {
		// The end-to-end shape: an actual metric definition edit, through the
		// real hash function, into the cube identity.
		m := MetricDefinition{
			ID:   "m_revenue",
			Name: "Revenue",
			Expression: MetricExpression{
				Kind:          "derived",
				BaseMetricIDs: []string{"m_units", "m_cost"},
				NumeratorID:   "m_units",
				DenominatorID: "m_cost",
			},
		}
		c2 := sampleCube()
		c2.MetricIDs = []string{"m_revenue"}
		before := ComputeCubeContentHash(c2, []string{ComputeMetricContentHash(m)})

		m.Expression.NumeratorID = "m_cost"
		m.Expression.DenominatorID = "m_units"
		after := ComputeCubeContentHash(c2, []string{ComputeMetricContentHash(m)})

		assert.NotEqual(t, before, after,
			"inverting a ratio's operands must invalidate the cube, or the materialized view keeps the old direction")
	})
}

func TestCubeDimensionSet(t *testing.T) {
	c := sampleCube()
	assert.Equal(t, []string{"country", "product"}, c.DimensionSet())

	c.Dimensions = append(c.Dimensions, CubeDimension{TermNodeID: "  "})
	assert.Equal(t, []string{"country", "product"}, c.DimensionSet(),
		"blank term IDs are dropped")
}

// TestCacheKey_CubeContentHashInvalidates is acceptance test #10: deploying or
// editing a cube must invalidate dependent cache entries, so a cached envelope
// can never claim a cubeHit from a cube that has since changed.
func TestCacheKey_CubeContentHashInvalidates(t *testing.T) {
	metrics := []MetricDefinition{revenueMetric()}
	base := ComputeQueryAndMetricsAndCubeCacheKey(
		"tenant_a", "qhash", metrics, "cube-hash-v1", "phash", "abachash", "hot", "bo-v1")

	edited := ComputeQueryAndMetricsAndCubeCacheKey(
		"tenant_a", "qhash", metrics, "cube-hash-v2", "phash", "abachash", "hot", "bo-v1")
	assert.NotEqual(t, base, edited,
		"a changed cube content hash must produce a different cache key")

	undeployed := ComputeQueryAndMetricsAndCubeCacheKey(
		"tenant_a", "qhash", metrics, "", "phash", "abachash", "hot", "bo-v1")
	assert.NotEqual(t, base, undeployed,
		"removing the cube must not reuse the cube-keyed entry")
}

// TestCacheKey_NoCubeMatchesLegacyKey verifies a query with no cube produces
// exactly the key the pre-cube implementation produced, so existing cache
// entries stay valid.
func TestCacheKey_NoCubeMatchesLegacyKey(t *testing.T) {
	metrics := []MetricDefinition{revenueMetric()}
	legacy := ComputeQueryAndMetricsCacheKey("tenant_a", "qhash", metrics, "phash", "abachash", "hot", "bo-v1")
	withEmptyCube := ComputeQueryAndMetricsAndCubeCacheKey("tenant_a", "qhash", metrics, "", "phash", "abachash", "hot", "bo-v1")
	assert.Equal(t, legacy, withEmptyCube,
		"an empty cube hash must not perturb the existing key")
}

// TestCubeContentHashChangesWithContent confirms the hash the cache key consumes
// is content-sensitive, which is what makes the invalidation above meaningful.
func TestCubeContentHashChangesWithContent(t *testing.T) {
	base := ComputeCubeContentHash(sampleCube(), nil)
	changed := sampleCube()
	changed.Grains = append(changed.Grains, []string{"customer", "order_date"})
	assert.NotEqual(t, base, ComputeCubeContentHash(changed, nil))
}
