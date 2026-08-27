package compliance_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/compliance"
)

func TestPreTradeComplianceVM_ConcentrationLimitBreach(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	service := compliance.NewExceptionReopenRadarService(nil)

	// Proposed portfolio: $10M total AUM, AAPL = $3M (30%), Threshold = 20%
	holdings := []compliance.PortfolioHoldingSnapshot{
		{SecurityID: "AAPL", Quantity: 15000, MarketValue: 3000000.0},
		{SecurityID: "MSFT", Quantity: 10000, MarketValue: 4000000.0},
		{SecurityID: "CASH", Quantity: 3000000, MarketValue: 3000000.0},
	}

	// 1. What-If Simulation on POSORD
	isBreach, conc, err := service.EvaluateWhatIfCompliance(
		ctx,
		tenantID,
		"PORT-ALPHA-01",
		compliance.ScopeWhatIfProposed,
		holdings,
		25.0, // 25% max limit
	)
	if err != nil {
		t.Fatalf("what-if evaluation failed: %v", err)
	}

	// MSFT is 40% of 10M -> Breaches 25% limit
	if !isBreach {
		t.Errorf("expected concentration breach for MSFT at 40%%")
	}
	if conc != 40.0 {
		t.Errorf("expected max concentration 40.0%%, got %f%%", conc)
	}

	// 2. Exception Re-open Radar: CLOSED_CORRECTED but condition persisted
	shouldReopen, reason := service.EvaluateReopenState(
		"CLOSED_CORRECTED",
		10000000.0,
		10000000.0,
		"fp_orig",
		"fp_orig",
		10.0,
		true, // still violating
	)
	if !shouldReopen {
		t.Errorf("expected re-open on uncorrected persistent breach")
	}
	if reason == "" {
		t.Errorf("expected descriptive re-open reason")
	}

	// 3. Exception Re-open Radar: CLOSED_NO_ACTION with 15% Market Value Drift (> 10% tolerance)
	shouldReopenMV, _ := service.EvaluateReopenState(
		"CLOSED_NO_ACTION",
		10000000.0,
		11500000.0, // +15% MV
		"fp_orig",
		"fp_orig",
		10.0,
		true,
	)
	if !shouldReopenMV {
		t.Errorf("expected re-open on MV drift > tolerance")
	}
}
