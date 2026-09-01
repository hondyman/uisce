package cel

import (
	"testing"

	"github.com/google/cel-go/cel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCELEvaluator_EvaluateRule(t *testing.T) {
	evaluator, err := NewCELEvaluator(
		cel.Variable("custom_leverage_limit", cel.DoubleType),
	)
	require.NoError(t, err)

	// Pre-trade compliance rule
	rule := `order_amount <= 1000000.0 && !restriction_flag && (account_subtype == "institutional" || is_qualified_investor)`

	validInput := map[string]interface{}{
		"order_amount":          500000.0,
		"restriction_flag":      false,
		"account_subtype":       "institutional",
		"is_qualified_investor": true,
	}

	pass, err := evaluator.EvaluateRule(rule, validInput)
	require.NoError(t, err)
	assert.True(t, pass)

	invalidInput := map[string]interface{}{
		"order_amount":          1500000.0,
		"restriction_flag":      false,
		"account_subtype":       "institutional",
		"is_qualified_investor": true,
	}

	pass, err = evaluator.EvaluateRule(rule, invalidInput)
	require.NoError(t, err)
	assert.False(t, pass)
}

func TestCELEvaluator_EvaluateBatch(t *testing.T) {
	evaluator, err := NewCELEvaluator()
	require.NoError(t, err)

	rule := `nav >= 100000.0 && liquidity_buffer_pct >= 0.05`

	batch := []map[string]interface{}{
		{"nav": 200000.0, "liquidity_buffer_pct": 0.10},
		{"nav": 50000.0, "liquidity_buffer_pct": 0.10},
		{"nav": 500000.0, "liquidity_buffer_pct": 0.02},
		{"nav": 1000000.0, "liquidity_buffer_pct": 0.08},
	}

	results, err := evaluator.EvaluateBatch(rule, batch)
	require.NoError(t, err)
	assert.Equal(t, []bool{true, false, false, true}, results)
}
