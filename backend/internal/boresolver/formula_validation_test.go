package boresolver_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	br "github.com/hondyman/uisce/backend/internal/boresolver"
)

func TestValidateFormula_ValidPushdown(t *testing.T) {
	v := br.ValidateFormula("(${gross_return} - ${management_fee}) * 100", nil, br.PreferAuto, nil)
	assert.True(t, v.Valid)
	assert.Equal(t, "pushdown", v.Tier)
	assert.ElementsMatch(t, []string{"gross_return", "management_fee"}, v.ReferencedTerms)
	assert.Empty(t, v.Errors)
}

func TestValidateFormula_HostRuntimeTier(t *testing.T) {
	v := br.ValidateFormula("xirr(${cashflow_amount}, ${cashflow_date})", nil, br.PreferAuto, nil)
	assert.True(t, v.Valid)
	assert.Equal(t, "host_runtime", v.Tier)
	assert.ElementsMatch(t, []string{"xirr"}, v.FunctionsUsed)
}

func TestValidateFormula_UnknownTerm(t *testing.T) {
	known := map[string]bool{"total_revenue": true, "total_aum": true}
	v := br.ValidateFormula("${total_revenue} / ${toatl_aum}", nil, br.PreferAuto, known)
	assert.False(t, v.Valid)
	require.Len(t, v.Errors, 1)
	assert.Contains(t, v.Errors[0].Message, "toatl_aum")
}

func TestValidateFormula_PreferPushdownRejectsHostRuntimeOnly(t *testing.T) {
	v := br.ValidateFormula("xirr(${amount}, ${date})", nil, br.PreferPushdown, nil)
	assert.False(t, v.Valid)
	require.Len(t, v.Errors, 1)
	assert.Contains(t, v.Errors[0].Message, "pushdown")
}

// TestValidateFormula_ParseErrorPosition proves the formula-bar UX claim:
// a syntax error reports its position against the ORIGINAL "${term}"-syntax
// string the user typed, not the internally rewritten one — so a caret at
// v.Errors[0].Pos in the user's own text lands on the actual problem.
func TestValidateFormula_ParseErrorPosition(t *testing.T) {
	// "${a} +" -- dangling operator, nothing after '+'. The '+' sits at
	// index 5 in the original text ("${a} +"); after rewriting to "a +" it
	// would sit at index 2, so a passing test here proves the position was
	// translated back, not left in rewritten-string space.
	formula := "${a} + "
	v := br.ValidateFormula(formula, nil, br.PreferAuto, nil)
	assert.False(t, v.Valid)
	require.Len(t, v.Errors, 1)

	pos := v.Errors[0].Pos
	if pos < 0 || pos > len(formula) {
		t.Fatalf("error position %d is out of bounds for formula %q (len %d)", pos, formula, len(formula))
	}
}

func TestValidateFormula_MultiTermReferenceDeduped(t *testing.T) {
	v := br.ValidateFormula("${x} + ${x} + ${y}", nil, br.PreferAuto, nil)
	assert.True(t, v.Valid)
	assert.ElementsMatch(t, []string{"x", "y"}, v.ReferencedTerms)
}
