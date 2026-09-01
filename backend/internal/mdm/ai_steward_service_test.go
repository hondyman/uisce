package mdm

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAIMDMStewardService(t *testing.T) {
	svc := NewAIMDMStewardService(nil)
	ctx := context.Background()
	tenantID := uuid.New()

	now := time.Now().UTC()
	feeds := []VendorFeedPayload{
		{
			VendorName:    "REFINITIV",
			EffectiveDate: now.Add(-4 * time.Minute),
			Attributes:    map[string]interface{}{"price": 92.70},
		},
		{
			VendorName:    "BLOOMBERG",
			EffectiveDate: now.Add(-12 * time.Second),
			Attributes:    map[string]interface{}{"price": 98.42},
		},
		{
			VendorName:    "IDC",
			EffectiveDate: now.Add(-1 * time.Minute),
			Attributes:    map[string]interface{}{"price": 98.40},
		},
	}

	winner, val, conf, err := svc.EvaluateNeuralSurvivorship(ctx, tenantID, "PRICING", "FIXED_INCOME", feeds)
	if err != nil {
		t.Fatalf("unexpected error evaluating survivorship: %v", err)
	}

	if winner != "BLOOMBERG" {
		t.Errorf("expected BLOOMBERG winner due to freshness and high accuracy, got %s", winner)
	}

	if conf < 0.80 {
		t.Errorf("expected high confidence score, got %f", conf)
	}

	if val == nil {
		t.Errorf("expected non-nil winning value")
	}

	triage, err := svc.GenerateAgenticBreakTriage(
		ctx,
		tenantID,
		uuid.New(),
		"PRICING",
		"SEC_US912810TL44",
		"market_price",
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error generating break triage: %v", err)
	}

	if triage.WinningVendor != "BLOOMBERG" {
		t.Errorf("expected BLOOMBERG triage winner, got %s", triage.WinningVendor)
	}

	if len(triage.MerkleReceipt) != 64 {
		t.Errorf("expected 64-char SHA256 merkle seal, got %d", len(triage.MerkleReceipt))
	}
}
