package calculation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompileExpression(t *testing.T) {
	svc := NewService()

	formula := "IF([Account.market_value] > 1000000, [Account.market_value] * 0.01, 0)"
	res, err := svc.CompileExpression(context.Background(), ExpressionCompileRequest{
		Formula: formula,
		BOName:  "Account",
	})

	assert.NoError(t, err)
	assert.True(t, res.Valid)
	assert.Contains(t, res.ExtractedFields, "Account.market_value")
	assert.Contains(t, res.CompiledSQL, "market_value * 0.01")
}
