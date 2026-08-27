package compliance

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestCCDSARService(t *testing.T) {
	svc := NewCCDSARService(nil)
	ctx := context.Background()
	tenantID := uuid.New()
	portfolioID := uuid.New()
	ruleID := uuid.New()

	// 1. Stable portfolio - no basket
	basketID, err := svc.EvaluatePortfolioDriftAndStageBasket(ctx, tenantID, portfolioID, ruleID, 60.0, 72.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if basketID != nil {
		t.Errorf("expected no basket for stable utilization")
	}

	// 2. Drifting portfolio - stages basket
	basketID, err = svc.EvaluatePortfolioDriftAndStageBasket(ctx, tenantID, portfolioID, ruleID, 88.0, 96.5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if basketID == nil {
		t.Errorf("expected staged basket ID for critical drift")
	}
}
