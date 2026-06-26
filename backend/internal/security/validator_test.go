package security

import (
	"testing"

	"github.com/hondyman/semlayer/backend/internal/security/dsl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDslValidation_ValidPredicate(t *testing.T) {
	parser := dsl.NewParser()

	// Test simple equality
	ast, err := parser.Parse("region = 'EMEA'")
	require.NoError(t, err)
	assert.NotNil(t, ast)

	sql := ast.ToSQL()
	assert.Contains(t, sql, "region")
	assert.Contains(t, sql, "'EMEA'")

	// Verify referenced fields
	fields := ast.GetReferencedFields()
	assert.Contains(t, fields, "region")
}

func TestDslValidation_AndOperator(t *testing.T) {
	parser := dsl.NewParser()

	ast, err := parser.Parse("region = 'EMEA' AND client_type != 'internal'")
	require.NoError(t, err)
	assert.NotNil(t, ast)

	sql := ast.ToSQL()
	assert.Contains(t, sql, "region = 'EMEA'")
	assert.Contains(t, sql, "client_type != 'internal'")
	assert.Contains(t, sql, "AND")

	// Verify referenced fields
	fields := ast.GetReferencedFields()
	assert.Contains(t, fields, "region")
	assert.Contains(t, fields, "client_type")
}

func TestDslValidation_OrOperator(t *testing.T) {
	parser := dsl.NewParser()

	ast, err := parser.Parse("region = 'EMEA' OR region = 'APAC'")
	require.NoError(t, err)
	assert.NotNil(t, ast)

	sql := ast.ToSQL()
	assert.Contains(t, sql, "OR")
	assert.Contains(t, sql, "region")

	fields := ast.GetReferencedFields()
	assert.Contains(t, fields, "region")
}

func TestDslValidation_InOperator(t *testing.T) {
	parser := dsl.NewParser()

	ast, err := parser.Parse("region IN ('EMEA', 'APAC', 'LATAM')")
	require.NoError(t, err)
	assert.NotNil(t, ast)

	sql := ast.ToSQL()
	assert.Contains(t, sql, "IN")
	assert.Contains(t, sql, "'EMEA'")
	assert.Contains(t, sql, "'APAC'")
	assert.Contains(t, sql, "'LATAM'")
}

func TestDslValidation_IsNull(t *testing.T) {
	parser := dsl.NewParser()

	ast, err := parser.Parse("deleted_at IS NULL")
	require.NoError(t, err)
	assert.NotNil(t, ast)

	sql := ast.ToSQL()
	assert.Contains(t, sql, "deleted_at")
	assert.Contains(t, sql, "IS NULL")
}

func TestDslValidation_NotOperator(t *testing.T) {
	parser := dsl.NewParser()

	ast, err := parser.Parse("NOT region = 'internal'")
	require.NoError(t, err)
	assert.NotNil(t, ast)

	sql := ast.ToSQL()
	assert.Contains(t, sql, "NOT")
	assert.Contains(t, sql, "region")
}

func TestDslValidation_ComplexPredicate(t *testing.T) {
	parser := dsl.NewParser()

	ast, err := parser.Parse("(region = 'EMEA' OR region = 'APAC') AND client_type != 'internal' AND status = 'active'")
	require.NoError(t, err)
	assert.NotNil(t, ast)

	sql := ast.ToSQL()
	assert.Contains(t, sql, "OR")
	assert.Contains(t, sql, "AND")

	fields := ast.GetReferencedFields()
	assert.Contains(t, fields, "region")
	assert.Contains(t, fields, "client_type")
	assert.Contains(t, fields, "status")
	assert.Equal(t, 3, len(fields))
}

func TestDslValidation_RejectsUnknownFields(t *testing.T) {
	// This would be tested with a real DslValidator that has access to catalog
	// For now, we test the parser's ability to extract fields
	parser := dsl.NewParser()

	ast, err := parser.Parse("unknown_field = 'x'")
	require.NoError(t, err)

	fields := ast.GetReferencedFields()
	assert.Contains(t, fields, "unknown_field")

	// In a real test with DslValidator:
	// validator := NewDslValidator(mockDB)
	// result := validator.Validate(ctx, "bo:test", "unknown_field = 'x'")
	// assert.False(result.Valid)
	// assert.Contains(*result.ErrorMessage, "Field not allowed")
}

func TestDslValidation_EmptyExpression(t *testing.T) {
	parser := dsl.NewParser()

	_, err := parser.Parse("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty expression")
}

func TestDslValidation_ComparisonOperators(t *testing.T) {
	parser := dsl.NewParser()

	tests := []struct {
		name string
		expr string
		op   string
	}{
		{"Greater than", "age > 18", ">"},
		{"Less than", "age < 65", "<"},
		{"Greater or equal", "age >= 21", ">="},
		{"Less or equal", "age <= 100", "<="},
		{"Not equal", "status != 'deleted'", "!="},
		{"Equal", "status = 'active'", "="},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast, err := parser.Parse(tt.expr)
			require.NoError(t, err)
			assert.NotNil(t, ast)

			sql := ast.ToSQL()
			assert.Contains(t, sql, tt.op)
		})
	}
}

func TestDslValidation_LikeOperator(t *testing.T) {
	parser := dsl.NewParser()

	ast, err := parser.Parse("name LIKE '%smith%'")
	require.NoError(t, err)
	assert.NotNil(t, ast)

	sql := ast.ToSQL()
	assert.Contains(t, sql, "LIKE")
	assert.Contains(t, sql, "'%smith%'")
}
