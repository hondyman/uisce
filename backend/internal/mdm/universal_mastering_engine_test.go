package mdm

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestUniversalMasteringEngine(t *testing.T) {
	engine := NewUniversalMasteringEngine(nil)
	ctx := context.Background()
	tenantID := uuid.New()

	// 1. Test Checksum validation during identifier graph resolution
	validISIN := "US0378331005"   // AAPL ISIN
	invalidISIN := "US0378331009" // Invalid check digit

	_, err := engine.ResolveIdentifiersGraph(ctx, tenantID, map[string]string{"ISIN": validISIN})
	if err != nil {
		t.Fatalf("unexpected error resolving valid ISIN: %v", err)
	}

	_, err = engine.ResolveIdentifiersGraph(ctx, tenantID, map[string]string{"ISIN": invalidISIN})
	if err == nil {
		t.Fatalf("expected error on invalid ISIN checksum, got nil")
	}

	// 2. Test Neural Survivorship with Time-Decay
	now := time.Now().UTC()
	feeds := []VendorFeedPayload{
		{
			VendorName:    "REFINITIV",
			EffectiveDate: now.Add(-30 * time.Minute),
			Attributes:    map[string]interface{}{"price": 100.0},
		},
		{
			VendorName:    "BLOOMBERG",
			EffectiveDate: now.Add(-5 * time.Second),
			Attributes:    map[string]interface{}{"price": 102.5},
		},
	}

	winner, val, conf, err := engine.EvaluateNeuralSurvivorship(ctx, tenantID, "PRICING", "EQUITY", feeds)
	if err != nil {
		t.Fatalf("unexpected error evaluating survivorship: %v", err)
	}

	if winner != "BLOOMBERG" {
		t.Errorf("expected BLOOMBERG winner due to freshness, got %s", winner)
	}

	if conf < 0.85 {
		t.Errorf("expected high confidence score, got %f", conf)
	}

	if val == nil {
		t.Errorf("expected non-nil winning value")
	}

	// 3. Test Mastering and SEC Rule 17a-4 Merkle Receipt
	feedRecord := VendorFeedRecord{
		TenantID:      tenantID,
		DomainKey:     "PRICING",
		VendorSource:  "BLOOMBERG",
		EffectiveTime: now,
		Identifiers:   map[string]string{"ISIN": validISIN, "TICKER": "AAPL"},
		Attributes:    map[string]interface{}{"market_price": 224.50, "currency": "USD"},
	}

	res, err := engine.MasterAndSealRecord(ctx, tenantID, feedRecord)
	if err != nil {
		t.Fatalf("unexpected error mastering record: %v", err)
	}

	if len(res.MerkleAuditSeal) != 64 {
		t.Errorf("expected 64-char SHA256 merkle seal, got %d", len(res.MerkleAuditSeal))
	}

	if res.MasterEntitySID != validISIN {
		t.Errorf("expected SID %s, got %s", validISIN, res.MasterEntitySID)
	}
}

