package compliance

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestUnifiedComplianceEngine_EvaluateTradeTicket(t *testing.T) {
	engine := NewUnifiedComplianceEngine(nil)

	ticket := TradeTicket{
		TicketID:     "TICKET-ORD-9021",
		TenantID:     uuid.New(),
		AccountID:    uuid.New().String(),
		SecurityID:   uuid.New().String(),
		OrderAction:  "BUY",
		OrderShares:  5000,
		OrderPrice:   140.00,
		IndustryCode: "TECHNOLOGY",
	}

	results, canExecute, err := engine.EvaluateTradeTicket(context.Background(), ticket)
	if err != nil {
		t.Fatalf("unexpected error evaluating ticket: %v", err)
	}

	if len(results) == 0 {
		t.Fatalf("expected compliance evaluation results")
	}

	for _, res := range results {
		if res.ExecutionTimeNs <= 0 {
			t.Errorf("expected recorded nanosecond execution time, got %d", res.ExecutionTimeNs)
		}
		if res.RuleCode == "" {
			t.Errorf("expected rule code to be present")
		}
	}

	_ = canExecute
}
