package querybuilder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func agg(id, term string) MetricDefinition {
	return MetricDefinition{
		ID:         id,
		Name:       id,
		BOID:       "bo_sales",
		Expression: MetricExpression{Kind: "aggregation", Fn: "sum", TermNodeID: term},
	}
}

func TestImportMetricBundle_HappyPath(t *testing.T) {
	rev := agg("m_rev", "revenue")
	cost := agg("m_cost", "cost")
	margin := MetricDefinition{
		ID:   "m_margin",
		Name: "Margin Ratio",
		BOID: "bo_sales",
		Expression: MetricExpression{
			Kind:          "derived",
			BaseMetricIDs: []string{"m_rev", "m_cost"},
			NumeratorID:   "m_rev",
			DenominatorID: "m_cost",
		},
	}

	bundle, err := ExportMetricBundle([]MetricDefinition{rev, cost, margin})
	require.NoError(t, err)

	got, err := ImportMetricBundle(bundle, nil)
	require.NoError(t, err)
	require.Len(t, got, 3)

	// Every imported metric carries a content hash, so downstream cache keys
	// and the cube deploy identity have something to work with.
	for _, m := range got {
		assert.NotEmpty(t, m.ContentHash, "metric %s", m.ID)
	}
	assert.Equal(t, ComputeMetricContentHash(rev), got[0].ContentHash)
}

// TestImportMetricBundle_RejectsAmbiguousRatioAtIngestion is the reason this
// function exists. A bundle can arrive by migration, seed or corpus and never
// touch CompileMetric, so if the ratio rule only lived in the compiler an
// ambiguous metric would be stored happily and rejected much later, at query
// or deploy time, far from whoever authored it.
func TestImportMetricBundle_RejectsAmbiguousRatioAtIngestion(t *testing.T) {
	rev := agg("m_rev", "revenue")
	cost := agg("m_cost", "cost")
	ordered := MetricDefinition{
		ID:   "m_margin",
		Name: "Margin Ratio",
		BOID: "bo_sales",
		Expression: MetricExpression{
			Kind:          "derived",
			BaseMetricIDs: []string{"m_rev", "m_cost"},
		},
	}

	bundle, err := ExportMetricBundle([]MetricDefinition{rev, cost, ordered})
	require.NoError(t, err, "the exporter still emits it; the importer is the gate")

	_, err = ImportMetricBundle(bundle, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidMetricBundle)
	assert.Contains(t, err.Error(), "requires numeratorId and denominatorId")
	assert.Contains(t, err.Error(), `metric "m_margin"`,
		"the error must name the offending metric, not just the bundle")
}

func TestImportMetricBundle_Rejections(t *testing.T) {
	t.Run("nil bundle", func(t *testing.T) {
		_, err := ImportMetricBundle(nil, nil)
		require.ErrorIs(t, err, ErrInvalidMetricBundle)
	})

	t.Run("unknown schema version", func(t *testing.T) {
		bundle, err := ExportMetricBundle([]MetricDefinition{agg("m_rev", "revenue")})
		require.NoError(t, err)
		bundle.SchemaVersion = "uisce.metric-bundle/99"
		_, err = ImportMetricBundle(bundle, nil)
		require.ErrorIs(t, err, ErrInvalidMetricBundle)
		assert.Contains(t, err.Error(), "uisce.metric-bundle/99")
	})

	t.Run("metric with no id", func(t *testing.T) {
		bundle, err := ExportMetricBundle([]MetricDefinition{agg("", "revenue")})
		require.NoError(t, err)
		_, err = ImportMetricBundle(bundle, nil)
		require.ErrorIs(t, err, ErrInvalidMetricBundle)
		assert.Contains(t, err.Error(), "no id")
	})

	t.Run("duplicate id against the existing set", func(t *testing.T) {
		bundle, err := ExportMetricBundle([]MetricDefinition{agg("m_rev", "revenue")})
		require.NoError(t, err)
		_, err = ImportMetricBundle(bundle, map[string]MetricDefinition{"M_REV ": agg("m_rev", "revenue")})
		require.ErrorIs(t, err, ErrInvalidMetricBundle)
		assert.Contains(t, err.Error(), "twice",
			"two definitions for one ID is the same ambiguity class as a bare ratio")
	})

	t.Run("dangling base metric", func(t *testing.T) {
		orphan := MetricDefinition{
			ID:   "m_margin",
			Name: "Margin Ratio",
			BOID: "bo_sales",
			Expression: MetricExpression{
				Kind:          "derived",
				BaseMetricIDs: []string{"m_rev", "m_cost"},
				NumeratorID:   "m_rev",
				DenominatorID: "m_cost",
			},
		}
		bundle, err := ExportMetricBundle([]MetricDefinition{orphan})
		require.NoError(t, err)
		_, err = ImportMetricBundle(bundle, nil)
		require.ErrorIs(t, err, ErrInvalidMetricBundle)
		assert.Contains(t, err.Error(), "neither in the bundle nor already present")
	})

	t.Run("base metric resolved from the existing set", func(t *testing.T) {
		margin := MetricDefinition{
			ID:   "m_margin",
			Name: "Margin Ratio",
			BOID: "bo_sales",
			Expression: MetricExpression{
				Kind:          "derived",
				BaseMetricIDs: []string{"m_rev", "m_cost"},
				NumeratorID:   "m_rev",
				DenominatorID: "m_cost",
			},
		}
		bundle, err := ExportMetricBundle([]MetricDefinition{margin})
		require.NoError(t, err)
		existing := map[string]MetricDefinition{
			"m_rev":  agg("m_rev", "revenue"),
			"M_COST": agg("m_cost", "cost"),
		}
		got, err := ImportMetricBundle(bundle, existing)
		require.NoError(t, err, "an existing base metric is a legitimate reference")
		require.Len(t, got, 1)
	})

	t.Run("dependency cycle is named, not stack-overflowed", func(t *testing.T) {
		a := MetricDefinition{
			ID:   "m_a",
			Name: "a",
			BOID: "bo_sales",
			Expression: MetricExpression{
				Kind: "derived", BaseMetricIDs: []string{"m_b"},
				NumeratorID: "m_b", DenominatorID: "m_b",
			},
		}
		b := MetricDefinition{
			ID:   "m_b",
			Name: "b",
			BOID: "bo_sales",
			Expression: MetricExpression{
				Kind: "derived", BaseMetricIDs: []string{"m_a"},
				NumeratorID: "m_a", DenominatorID: "m_a",
			},
		}
		// Numerator == denominator is rejected on its own, so make each a
		// legitimate two-operand ratio that points at the other plus a real
		// base, which is what actually produces a cycle.
		base := agg("m_base", "revenue")
		a.Expression.NumeratorID, a.Expression.DenominatorID = "m_b", "m_base"
		a.Expression.BaseMetricIDs = []string{"m_b", "m_base"}
		b.Expression.NumeratorID, b.Expression.DenominatorID = "m_a", "m_base"
		b.Expression.BaseMetricIDs = []string{"m_a", "m_base"}

		bundle, err := ExportMetricBundle([]MetricDefinition{base, a, b})
		require.NoError(t, err)
		_, err = ImportMetricBundle(bundle, nil)
		require.ErrorIs(t, err, ErrInvalidMetricBundle)
		assert.Contains(t, err.Error(), "cycle")
	})

	t.Run("hand-edited bundle is caught by the content hash", func(t *testing.T) {
		bundle, err := ExportMetricBundle([]MetricDefinition{agg("m_rev", "revenue")})
		require.NoError(t, err)

		// The shape of a "golden" fixture someone tuned by hand: the content
		// moved but the exported hash did not. Without this check the corpus
		// could drift away from the semantics it claims to protect.
		bundle.Metrics[0].Expression.TermNodeID = "revenue_typo"
		_, err = ImportMetricBundle(bundle, nil)
		require.ErrorIs(t, err, ErrInvalidMetricBundle)
		assert.Contains(t, err.Error(), "edited after export")
	})

	t.Run("a bundle with no exported hash is filled in", func(t *testing.T) {
		raw := &MetricBundle{
			SchemaVersion: MetricBundleSchemaVersion,
			Metrics:       []MetricDefinition{agg("m_rev", "revenue")},
		}
		got, err := ImportMetricBundle(raw, nil)
		require.NoError(t, err)
		assert.Equal(t, ComputeMetricContentHash(raw.Metrics[0]), got[0].ContentHash)
	})
}
