package surveillance

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestAllocationFairnessDetector_FlagsUnfairAllocationAndPriceDispersion(t *testing.T) {
	// Max allowed deviation 1% (0.01), max price variance $0.05
	detector := NewAllocationFairnessDetector(decimal.RequireFromString("0.01"), decimal.RequireFromString("0.05"))

	blockID := uuid.New()
	tenantID := uuid.New()
	securityID := uuid.New()
	acctA := uuid.New()
	acctB := uuid.New()

	block := BlockOrderExecution{
		BlockOrderID:  blockID,
		TenantID:      tenantID,
		SecurityID:    securityID,
		TotalExecuted: decimal.NewFromInt(1000),
		AveragePrice:  decimal.RequireFromString("50.00"),
		ExecutedAt:    time.Now().UTC(),
		Allocations: []BlockOrderAllocation{
			// Account A: requested 500 (50%), but allocated 700 (70%) at $49.90 (preferential price)
			{
				AccountID:         acctA,
				RequestedQuantity: decimal.NewFromInt(500),
				AllocatedQuantity: decimal.NewFromInt(700),
				AllocatedPrice:    decimal.RequireFromString("49.90"),
			},
			// Account B: requested 500 (50%), but allocated 300 (30%) at $50.23 (worse price)
			{
				AccountID:         acctB,
				RequestedQuantity: decimal.NewFromInt(500),
				AllocatedQuantity: decimal.NewFromInt(300),
				AllocatedPrice:    decimal.RequireFromString("50.23"),
			},
		},
	}

	violations, err := detector.EvaluateBlockFairness(context.Background(), block)
	if err != nil {
		t.Fatalf("EvaluateBlockFairness failed: %v", err)
	}

	if len(violations) < 2 {
		t.Fatalf("Expected at least 2 violations (deviation + price dispersion), got %d", len(violations))
	}

	var hasDeviation, hasPriceDispersion bool
	for _, v := range violations {
		if v.ViolationType == "PRO_RATA_DEVIATION" {
			hasDeviation = true
		}
		if v.ViolationType == "PRICE_DISPERSION" {
			hasPriceDispersion = true
		}
	}

	if !hasDeviation {
		t.Errorf("Expected PRO_RATA_DEVIATION violation")
	}
	if !hasPriceDispersion {
		t.Errorf("Expected PRICE_DISPERSION violation")
	}
}
