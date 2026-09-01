package compliance

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestBitemporalReplayService(t *testing.T) {
	svc := NewBitemporalReplayService(nil)
	ctx := context.Background()
	tenantID := uuid.New()

	req := BacktestRequest{
		RuleID:             uuid.New(),
		RuleName:           "At least 80% of Debt > BBB+",
		DynamicDenominator: "ST1 Debt Market Value",
		AsOfStartDate:      "2024-01-01",
		AsOfEndDate:        "2026-08-01",
	}

	report, err := svc.RunTimeTravelBacktest(ctx, tenantID, req)
	if err != nil {
		t.Fatalf("unexpected error running backtest: %v", err)
	}

	if report.DaysEvaluated != 640 {
		t.Errorf("expected 640 days, got %d", report.DaysEvaluated)
	}

	if len(report.MerkleRootHash) != 64 {
		t.Errorf("expected 64-char SHA256 merkle root hash, got %d", len(report.MerkleRootHash))
	}
}
