package query

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestPreflightFinOpsService(t *testing.T) {
	svc := NewPreflightFinOpsService(nil)
	ctx := context.Background()
	tenantID := uuid.New()

	estimate, err := svc.EvaluatePreflightCost(ctx, tenantID, "trade_order", 3, 2, 2, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !estimate.PassesBreaker {
		t.Errorf("expected passes breaker, got false")
	}

	if estimate.HotPercentage != 85 {
		t.Errorf("expected 85%% hot percentage, got %d", estimate.HotPercentage)
	}

	if len(estimate.ExplainDAGSteps) != 3 {
		t.Errorf("expected 3 DAG steps, got %d", len(estimate.ExplainDAGSteps))
	}
}
