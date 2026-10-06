package surveillance

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// BlockOrderAllocation represents an allocation to an individual account within a block order
type BlockOrderAllocation struct {
	AccountID         uuid.UUID
	RequestedQuantity decimal.Decimal
	AllocatedQuantity decimal.Decimal
	AllocatedPrice    decimal.Decimal
}

// BlockOrderExecution represents the complete block order and its member allocations
type BlockOrderExecution struct {
	BlockOrderID   uuid.UUID
	TenantID       uuid.UUID
	SecurityID     uuid.UUID
	TotalExecuted  decimal.Decimal
	AveragePrice   decimal.Decimal
	ExecutedAt     time.Time
	Allocations    []BlockOrderAllocation
}

// AllocationFairnessViolation records unfair allocation or price dispersion breaches
type AllocationFairnessViolation struct {
	TenantID          uuid.UUID
	BlockOrderID      uuid.UUID
	AccountID         uuid.UUID
	ViolationType     string // "PRO_RATA_DEVIATION", "PRICE_DISPERSION"
	ExpectedRatio     decimal.Decimal
	ActualRatio       decimal.Decimal
	DeviationPercent  decimal.Decimal
	PriceVariance     decimal.Decimal
	Details           string
	DetectedAt        time.Time
}

// AllocationFairnessDetector evaluates block executions against pro-rata fairness tolerances
type AllocationFairnessDetector struct {
	maxDeviationPct decimal.Decimal // Max permissible ratio deviation (e.g. 0.02 = 2%)
	maxPriceVariance decimal.Decimal // Max permissible price deviation from block average
}

func NewAllocationFairnessDetector(maxDeviationPct, maxPriceVariance decimal.Decimal) *AllocationFairnessDetector {
	return &AllocationFairnessDetector{
		maxDeviationPct:  maxDeviationPct,
		maxPriceVariance: maxPriceVariance,
	}
}

func (d *AllocationFairnessDetector) EvaluateBlockFairness(ctx context.Context, block BlockOrderExecution) ([]AllocationFairnessViolation, error) {
	var violations []AllocationFairnessViolation

	totalRequested := decimal.Zero
	totalAllocated := decimal.Zero
	for _, a := range block.Allocations {
		totalRequested = totalRequested.Add(a.RequestedQuantity)
		totalAllocated = totalAllocated.Add(a.AllocatedQuantity)
	}

	if totalRequested.IsZero() || totalAllocated.IsZero() {
		return nil, fmt.Errorf("block order has zero requested or allocated quantity")
	}

	for _, a := range block.Allocations {
		expectedRatio := a.RequestedQuantity.Div(totalRequested)
		actualRatio := a.AllocatedQuantity.Div(totalAllocated)
		deviation := actualRatio.Sub(expectedRatio).Abs()

		// 1. Pro-Rata Allocation Fairness Check
		if deviation.GreaterThan(d.maxDeviationPct) {
			violations = append(violations, AllocationFairnessViolation{
				TenantID:         block.TenantID,
				BlockOrderID:     block.BlockOrderID,
				AccountID:        a.AccountID,
				ViolationType:    "PRO_RATA_DEVIATION",
				ExpectedRatio:    expectedRatio,
				ActualRatio:      actualRatio,
				DeviationPercent: deviation.Mul(decimal.NewFromInt(100)),
				Details:          fmt.Sprintf("Account received %s ratio vs expected %s (deviation %s > limit %s)", actualRatio.StringFixed(4), expectedRatio.StringFixed(4), deviation.StringFixed(4), d.maxDeviationPct.StringFixed(4)),
				DetectedAt:       time.Now().UTC(),
			})
		}

		// 2. Price Dispersion Fairness Check (preferential pricing)
		if !d.maxPriceVariance.IsZero() && !a.AllocatedPrice.IsZero() {
			priceDiff := a.AllocatedPrice.Sub(block.AveragePrice).Abs()
			if priceDiff.GreaterThan(d.maxPriceVariance) {
				violations = append(violations, AllocationFairnessViolation{
					TenantID:         block.TenantID,
					BlockOrderID:     block.BlockOrderID,
					AccountID:        a.AccountID,
					ViolationType:    "PRICE_DISPERSION",
					PriceVariance:    priceDiff,
					Details:          fmt.Sprintf("Account allocated price %s deviates from block average %s by %s > %s", a.AllocatedPrice.StringFixed(4), block.AveragePrice.StringFixed(4), priceDiff.StringFixed(4), d.maxPriceVariance.StringFixed(4)),
					DetectedAt:       time.Now().UTC(),
				})
			}
		}
	}

	return violations, nil
}
