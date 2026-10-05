package surveillance

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestWashSaleDetector_DetectsLossAndRepurchaseAcrossAccounts(t *testing.T) {
	detector := NewWashSaleDetector(30)
	tenantID := uuid.New()
	ownerID := uuid.New()
	securityID := uuid.New()
	acct1 := uuid.New()
	acct2 := uuid.New()

	baseTime := time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)

	trades := []TradeRecord{
		// 1. Initial Buy in Account 1 at $100
		{
			ExecutionID:       uuid.New(),
			TenantID:          tenantID,
			AccountID:         acct1,
			BeneficialOwnerID: ownerID,
			SecurityID:        securityID,
			Side:              "BUY",
			Quantity:          decimal.NewFromInt(100),
			Price:             decimal.RequireFromString("100.00"),
			ExecutedAt:        baseTime,
		},
		// 2. Sell in Account 1 at $80 (Loss of $2,000) on Day 10
		{
			ExecutionID:       uuid.New(),
			TenantID:          tenantID,
			AccountID:         acct1,
			BeneficialOwnerID: ownerID,
			SecurityID:        securityID,
			Side:              "SELL",
			Quantity:          decimal.NewFromInt(100),
			Price:             decimal.RequireFromString("80.00"),
			CostBasis:         decimal.RequireFromString("100.00"),
			RealizedGainLoss:  decimal.RequireFromString("-2000.00"),
			ExecutedAt:        baseTime.Add(10 * 24 * time.Hour),
		},
		// 3. Repurchase in Account 2 (Same Beneficial Owner) on Day 25 (Within 30-day window)
		{
			ExecutionID:       uuid.New(),
			TenantID:          tenantID,
			AccountID:         acct2,
			BeneficialOwnerID: ownerID,
			SecurityID:        securityID,
			Side:              "BUY",
			Quantity:          decimal.NewFromInt(100),
			Price:             decimal.RequireFromString("82.00"),
			ExecutedAt:        baseTime.Add(25 * 24 * time.Hour),
		},
	}

	violations, err := detector.DetectWashSales(context.Background(), trades)
	if err != nil {
		t.Fatalf("DetectWashSales failed: %v", err)
	}

	if len(violations) != 1 {
		t.Fatalf("Expected 1 wash-sale violation, got %d", len(violations))
	}

	v := violations[0]
	if v.BeneficialOwnerID != ownerID {
		t.Errorf("Expected owner %s, got %s", ownerID, v.BeneficialOwnerID)
	}
	if !v.LossAmount.Equal(decimal.RequireFromString("2000.00")) {
		t.Errorf("Expected loss amount 2000.00, got %s", v.LossAmount)
	}
	if v.ReplacementAcctID != acct2 {
		t.Errorf("Expected replacement account %s, got %s", acct2, v.ReplacementAcctID)
	}
}
