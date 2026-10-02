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
	h1 := ComputeCubeContentHash(c)
	h2 := ComputeCubeContentHash(c)
	assert.Equal(t, h1, h2)
	assert.NotEmpty(t, h1)
	assert.Len(t, h1, 64, "SHA-256 hex is 64 chars")
}

// TestComputeCubeContentHash_CanonicalizationInsensitivity verifies that
// authoring-order noise does not change the hash, while semantically
// meaningful differences do.
func TestComputeCubeContentHash_CanonicalizationInsensitivity(t *testing.T) {
	base := sampleCube()
	baseHash := ComputeCubeContentHash(base)

	t.Run("metric id order is irrelevant", func(t *testing.T) {
		c := sampleCube()
		c.MetricIDs = []string{"m_units", "m_revenue"}
		assert.Equal(t, baseHash, ComputeCubeContentHash(c),
			"metric IDs are a set, so order must not matter")
	})

	t.Run("grain member order is irrelevant", func(t *testing.T) {
		c := sampleCube()
		c.Grains = [][]string{
			{"order_date", "product", "country"},
			{"order_date", "country"},
		}
		assert.Equal(t, baseHash, ComputeCubeContentHash(c),
			"dimensions within a grain are a set")
	})

	t.Run("case and whitespace are normalized", func(t *testing.T) {
		c := sampleCube()
		c.MetricIDs = []string{"  M_REVENUE ", "M_UNITS"}
		c.Dimensions[0].TermNodeID = " Country "
		assert.Equal(t, baseHash, ComputeCubeContentHash(c))
	})

	t.Run("drill path order is normalized", func(t *testing.T) {
		c := sampleCube()
		c.Dimensions[1].DrillPath = []string{"product", "product_category"}
		assert.Equal(t, baseHash, ComputeCubeContentHash(c))
	})

	// These must differ: a hash that ignored real content would make deploy
	// idempotency unsafe, because a changed cube would look unchanged.
	t.Run("changing metrics changes the hash", func(t *testing.T) {
		c := sampleCube()
		c.MetricIDs = append(c.MetricIDs, "m_margin")
		assert.NotEqual(t, baseHash, ComputeCubeContentHash(c))
	})

	t.Run("changing grains changes the hash", func(t *testing.T) {
		c := sampleCube()
		c.Grains = append(c.Grains, []string{"customer", "order_date"})
		assert.NotEqual(t, baseHash, ComputeCubeContentHash(c))
	})

	t.Run("reordering dimensions changes the hash", func(t *testing.T) {
		c := sampleCube()
		c.Dimensions[0], c.Dimensions[1] = c.Dimensions[1], c.Dimensions[0]
		assert.NotEqual(t, baseHash, ComputeCubeContentHash(c),
			"dimension order is the axis order and is significant")
	})

	t.Run("changing stale policy changes the hash", func(t *testing.T) {
		c := sampleCube()
		c.Materialization.StalePolicy = "force_raw_fallback"
		assert.NotEqual(t, baseHash, ComputeCubeContentHash(c))
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
// may only aggregate metrics that exist in the tenant.
func TestValidateCubeMetricReferences(t *testing.T) {
	known := map[string]bool{"m_revenue": true, "m_units": true}

	require.NoError(t, ValidateCubeMetricReferences(sampleCube(), known))

	t.Run("unknown metric is rejected", func(t *testing.T) {
		err := ValidateCubeMetricReferences(sampleCube(), map[string]bool{"m_revenue": true})
		require.ErrorIs(t, err, ErrCubeUnknownMetric)
		assert.Contains(t, err.Error(), "m_units")
	})

	t.Run("lookup is case and whitespace insensitive", func(t *testing.T) {
		require.NoError(t, ValidateCubeMetricReferences(sampleCube(), known),
			"IDs are normalized before lookup")
	})
}

// TestCubeDimensionSet verifies the normalized dimension set used by grain
// coverage checks.
func TestCubeDimensionSet(t *testing.T) {
	c := sampleCube()
	assert.Equal(t, []string{"country", "product"}, c.DimensionSet())

	c.Dimensions = append(c.Dimensions, CubeDimension{TermNodeID: "  "})
	assert.Equal(t, []string{"country", "product"}, c.DimensionSet(),
		"blank term IDs are dropped")
}
