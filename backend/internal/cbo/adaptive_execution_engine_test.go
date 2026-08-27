package cbo

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestAdaptiveExecutionEngine_BroadcastJoinConversion(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	aqe := NewAdaptiveExecutionEngine(100000)

	// Filtered rows = 4,200 <= 100,000 threshold -> should convert to BROADCAST_HASH
	plan, err := aqe.AdaptPlanAtRuntime(ctx, tenantID, 5000000, 4200, 100, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.AdaptedStrategy != JoinStrategyBroadcastHash {
		t.Errorf("expected AdaptedStrategy BROADCAST_HASH, got %s", plan.AdaptedStrategy)
	}
	if !plan.DynamicPruningActive {
		t.Errorf("expected DynamicPruningActive = true")
	}
	if plan.PrunedS3Splits <= 0 {
		t.Errorf("expected PrunedS3Splits > 0, got %d", plan.PrunedS3Splits)
	}
}

func TestAdaptiveExecutionEngine_LargeBatchRetainsDistributedHash(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	aqe := NewAdaptiveExecutionEngine(100000)

	// Filtered rows = 500,000 > 100,000 threshold -> should retain DISTRIBUTED_HASH
	plan, err := aqe.AdaptPlanAtRuntime(ctx, tenantID, 5000000, 500000, 100, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.AdaptedStrategy != JoinStrategyDistributedHash {
		t.Errorf("expected AdaptedStrategy DISTRIBUTED_HASH, got %s", plan.AdaptedStrategy)
	}
}
