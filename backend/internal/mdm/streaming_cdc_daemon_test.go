package mdm

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestStreamingCDCDaemon(t *testing.T) {
	fanoutCalled := false
	daemon := NewStreamingCDCDaemon(nil, func(ctx context.Context, res *ProcessedTickResult) error {
		fanoutCalled = true
		return nil
	})

	ctx := context.Background()
	tenantID := uuid.New().String()

	payload := map[string]interface{}{
		"tenantId":        tenantID,
		"domainKey":       "PRICING",
		"vendorSource":    "BLOOMBERG",
		"masterEntitySid": "US0378331005",
		"effectiveTime":   time.Now().UTC().Format(time.RFC3339),
		"identifiers": map[string]string{
			"ISIN":   "US0378331005",
			"TICKER": "AAPL",
		},
		"attributes": map[string]interface{}{
			"market_price": 226.75,
			"currency":     "USD",
		},
	}

	rawJSON, _ := json.Marshal(payload)

	res, err := daemon.ProcessTick(ctx, rawJSON)
	if err != nil {
		t.Fatalf("unexpected error processing CDC tick: %v", err)
	}

	if res.MasterEntitySID != "US0378331005" {
		t.Errorf("expected SID US0378331005, got %s", res.MasterEntitySID)
	}

	if !fanoutCalled {
		t.Errorf("expected downstream fanout handler to be called")
	}

	if len(res.MerkleAuditSeal) != 64 {
		t.Errorf("expected 64-character SHA256 merkle seal, got %d", len(res.MerkleAuditSeal))
	}
}
