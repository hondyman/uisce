package querybuilder

import (
	"testing"

	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMetricEquivalence_Matrix covers the 4-case canonical equivalence matrix:
// 1. Simple aggregation (SUM(price))
// 2. Formula kind with NULLIF division and bound variable
// 3. Derived metric composed of two base metrics
// 4. Adversarial variable injection check (ensures SQL syntax lands in Args, never in SQL string)
func TestMetricEquivalence_Matrix(t *testing.T) {
	compiler := NewMetricCompiler(boresolver.PostgresDialect{})

	// Case 1: Simple Aggregation
	t.Run("Case 1: Simple Aggregation Equivalence", func(t *testing.T) {
		metric := MetricDefinition{
			ID:   "m_notional",
			Name: "Total Notional",
			BOID: "bo_trade",
			Expression: MetricExpression{
				Kind:       "aggregation",
				Fn:         "sum",
				TermNodeID: "notional",
			},
			GrainAllowlist: []string{"desk", "currency", "trade_date"},
			Decomposable:   true,
		}

		res, err := compiler.CompileMetric(metric, nil, nil)
		require.NoError(t, err)

		// Hand-authored equivalent expression
		handAuthoredSQL := "SUM(t0.notional)"
		assert.Equal(t, handAuthoredSQL, res.SQLExpr)
		assert.Empty(t, res.Args)

		// Hash stability check
		h1 := ComputeMetricContentHash(metric)
		h2 := ComputeMetricContentHash(metric)
		assert.Equal(t, h1, h2)
		assert.NotEmpty(t, h1)
	})

	// Case 2: Formula with NULLIF Division and Bound Variable
	t.Run("Case 2: Formula with NULLIF and Bound Variable", func(t *testing.T) {
		metric := MetricDefinition{
			ID:   "m_adj_margin",
			Name: "Adjusted Margin Rate",
			BOID: "bo_trade",
			Expression: MetricExpression{
				Kind:    "formula",
				Formula: "SUM(t0.profit) / NULLIF(SUM(t0.revenue), 0) * @fx_rate",
			},
			GrainAllowlist: []string{"desk", "region"},
			Variables: []MetricVariable{
				{
					Name:         "fx_rate",
					Type:         "number",
					DefaultValue: 1.15,
					Required:     true,
				},
			},
			Decomposable: false, // Formula contains division
		}

		varBindings := map[string]interface{}{
			"fx_rate": 1.25,
		}

		res, err := compiler.CompileMetric(metric, varBindings, nil)
		require.NoError(t, err)

		// Expected compiled SQL with $1 parameter placeholder
		assert.Equal(t, "SUM(t0.profit) / NULLIF(SUM(t0.revenue), 0) * $1", res.SQLExpr)
		require.Len(t, res.Args, 1)
		assert.Equal(t, 1.25, res.Args[0])

		// Decomposable flag verification
		assert.False(t, DeriveDecomposable(metric.Expression))
	})

	// Case 3: Derived Metric (Composed of Two Base Metrics)
	t.Run("Case 3: Derived Metric Deterministic Composition", func(t *testing.T) {
		baseA := MetricDefinition{
			ID:   "m_base_rev",
			Name: "Total Revenue",
			BOID: "bo_sales",
			Expression: MetricExpression{
				Kind:       "aggregation",
				Fn:         "sum",
				TermNodeID: "revenue",
			},
			GrainAllowlist: []string{"rep", "region"},
		}
		baseB := MetricDefinition{
			ID:   "m_base_cost",
			Name: "Total Cost",
			BOID: "bo_sales",
			Expression: MetricExpression{
				Kind:       "aggregation",
				Fn:         "sum",
				TermNodeID: "cost",
			},
			GrainAllowlist: []string{"rep", "region"},
		}

		derivedMetric := MetricDefinition{
			ID:   "m_margin_ratio",
			Name: "Margin Ratio",
			BOID: "bo_sales",
			Expression: MetricExpression{
				Kind:          "derived",
				BaseMetricIDs: []string{"m_base_rev", "m_base_cost"},
				NumeratorID:   "m_base_rev",
				DenominatorID: "m_base_cost",
			},
			GrainAllowlist: []string{"rep", "region"},
		}

		lookup := map[string]MetricDefinition{
			"m_base_rev":  baseA,
			"m_base_cost": baseB,
		}

		res, err := compiler.CompileMetric(derivedMetric, nil, lookup)
		require.NoError(t, err)

		// The ratio names its operands, so its direction is data rather than a
		// convention. This case used to assert the opposite - "(SUM(t0.cost)) /
		// NULLIF((SUM(t0.revenue)), 0)" with a comment reading "Sorted base
		// IDs" - which made a metric named "Margin Ratio" compile to
		// cost/revenue. See ADR-025 and ADR-026.
		expectedSQL := "(SUM(t0.revenue)) / NULLIF((SUM(t0.cost)), 0)"
		assert.Equal(t, expectedSQL, res.SQLExpr)

		// Naming the operands the other way round must produce the reciprocal,
		// not the same number: that is what "explicit" has to mean.
		inverted := MetricDefinition{
			ID:   "m_margin_ratio",
			Name: "Margin Ratio",
			BOID: "bo_sales",
			Expression: MetricExpression{
				Kind:          "derived",
				BaseMetricIDs: []string{"m_base_rev", "m_base_cost"},
				NumeratorID:   "m_base_cost",
				DenominatorID: "m_base_rev",
			},
			GrainAllowlist: []string{"rep", "region"},
		}
		invertedSQL, err := compiler.CompileMetric(inverted, nil, lookup)
		require.NoError(t, err)
		assert.Equal(t, "(SUM(t0.cost)) / NULLIF((SUM(t0.revenue)), 0)", invertedSQL.SQLExpr)

		// And the ordered form is refused outright, so the direction can never
		// be inferred again (ADR-026).
		ordered := MetricDefinition{
			ID:   "m_margin_ratio",
			Name: "Margin Ratio",
			BOID: "bo_sales",
			Expression: MetricExpression{
				Kind:          "derived",
				BaseMetricIDs: []string{"m_base_rev", "m_base_cost"},
			},
			GrainAllowlist: []string{"rep", "region"},
		}
		_, err = compiler.CompileMetric(ordered, nil, lookup)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "requires numeratorId and denominatorId")

		// Opposite operand order must not collapse to one content hash: that
		// hash is the cube deploy key and part of the query cache key.
		assert.NotEqual(t, ComputeMetricContentHash(derivedMetric), ComputeMetricContentHash(inverted),
			"opposite operand order must not share a content hash")

		// Hand-authored equivalent hash check
		derivedHash := ComputeMetricContentHash(derivedMetric)
		assert.NotEmpty(t, derivedHash)
	})

	// Case 4: Adversarial Variable Injection Check
	t.Run("Case 4: Adversarial Variable Injection Proof", func(t *testing.T) {
		metric := MetricDefinition{
			ID:   "m_adv_var",
			Name: "Adversarial Variable Metric",
			BOID: "bo_trade",
			Expression: MetricExpression{
				Kind:    "formula",
				Formula: "SUM(t0.amount) * @tax_multiplier",
			},
			Variables: []MetricVariable{
				{Name: "tax_multiplier", Type: "number", Required: true},
			},
		}

		// Malicious input attempted inside variable value
		maliciousPayload := "1.0; DROP TABLE public.schedules; --"
		varBindings := map[string]interface{}{
			"tax_multiplier": maliciousPayload,
		}

		res, err := compiler.CompileMetric(metric, varBindings, nil)
		require.NoError(t, err)

		// SQL string must contain parameter placeholder $1 only, NEVER the injection payload
		assert.Equal(t, "SUM(t0.amount) * $1", res.SQLExpr)
		assert.NotContains(t, res.SQLExpr, "DROP TABLE")
		require.Len(t, res.Args, 1)
		assert.Equal(t, maliciousPayload, res.Args[0])
	})
}

// TestGrainValidation_Actionable422 ensures unauthorized grains return HTTP 422
// with the complete allowed grains list in the error description.
func TestGrainValidation_Actionable422(t *testing.T) {
	allowlist := []string{"desk", "currency", "region"}

	// Allowed grains
	err := ValidateGrains([]string{"desk", "region"}, allowlist)
	assert.NoError(t, err)

	// Unauthorized grain
	err = ValidateGrains([]string{"desk", "trader_ssn"}, allowlist)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrGrainNotAllowed)
	assert.Contains(t, err.Error(), "'trader_ssn' is not allowed")
	assert.Contains(t, err.Error(), "allowed grains: [desk currency region]")
}

// TestMetricCacheKey_InvalidationRule verifies that editing a metric definition's
// formula/grain changes its content hash and invalidates composite query cache keys.
func TestMetricCacheKey_InvalidationRule(t *testing.T) {
	tenantID := "t-acme"
	queryHash := "q_hash_abc123"
	paramsHash := "p_hash_def456"
	abacHash := "abac_hash_ghi789"

	metricV1 := MetricDefinition{
		ID:   "m_kpi",
		Name: "Revenue KPI",
		BOID: "bo_sales",
		Expression: MetricExpression{
			Kind:       "aggregation",
			Fn:         "sum",
			TermNodeID: "revenue",
		},
		GrainAllowlist: []string{"region"},
	}

	keyV1 := ComputeQueryAndMetricsCacheKey(tenantID, queryHash, []MetricDefinition{metricV1}, paramsHash, abacHash, "hot", "v1")

	// Modify metric definition (e.g. change grain or formula)
	metricV2 := metricV1
	metricV2.GrainAllowlist = []string{"region", "country"}

	keyV2 := ComputeQueryAndMetricsCacheKey(tenantID, queryHash, []MetricDefinition{metricV2}, paramsHash, abacHash, "hot", "v1")

	// Cache key MUST change upon metric modification
	assert.NotEqual(t, keyV1, keyV2, "modifying metric definition must invalidate query cache key")
}
