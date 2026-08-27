package cbo

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestCostGovernor_Rule8_CircuitBreaker(t *testing.T) {
	logger := zap.NewNop()
	governor := NewCostGovernor(85.0, 65.0, logger)
	tenantID := uuid.New()
	ctx := context.Background()

	t.Run("Simple Query Passes Circuit Breaker", func(t *testing.T) {
		res := governor.EvaluatePreFlightComplexity(ctx, tenantID, "SELECT account_bk, nav FROM oms.account WHERE tenant_id = '...' AND as_of_date = '2026-01-01'", 1, false, false)
		assert.False(t, res.Blocked)
		assert.Equal(t, "ALLOWED", res.Status)
		assert.Equal(t, 25.0, res.ComplexityScore) // 10 base + 15*1
	})

	t.Run("Warning Status for Moderately Complex Query", func(t *testing.T) {
		res := governor.EvaluatePreFlightComplexity(ctx, tenantID, "SELECT * FROM oms.position JOIN oms.account ...", 4, false, false)
		assert.False(t, res.Blocked)
		assert.Equal(t, "WARNING", res.Status)
		assert.Equal(t, 70.0, res.ComplexityScore) // 10 base + 15*4
	})

	t.Run("Rule 8 Circuit Breaker Blocks Runaway Cross-Tier Unpartitioned Query", func(t *testing.T) {
		res := governor.EvaluatePreFlightComplexity(ctx, tenantID, "SELECT * FROM large_lakehouse_table JOIN multi_tenant_mart ...", 3, true, true)
		assert.True(t, res.Blocked)
		assert.Equal(t, "FORBIDDEN", res.Status)
		// 10 base + 3*15 (45) + 30 (unpartitioned) + 40 (federation) = 125.0 >= 85.0
		assert.Equal(t, 125.0, res.ComplexityScore)
		assert.Contains(t, res.Reason, "Circuit breaker triggered")
	})
}
