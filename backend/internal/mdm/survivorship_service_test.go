package mdm

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestEnterpriseMDMService(t *testing.T) {
	svc := NewEnterpriseMDMService(nil)

	res, err := svc.IngestAndMasterRecord(context.Background(), &VendorFeedRecord{
		TenantID:     uuid.New(),
		DomainKey:    "SECURITY_MASTER",
		VendorSource: "BLOOMBERG",
		VendorID:     "AAPL_US",
		Identifiers: map[string]string{
			"ISIN":  "US0378331005",
			"CUSIP": "037833100",
		},
		Attributes: map[string]interface{}{
			"market_price": 226.40,
			"issuer":       "Apple Inc.",
		},
		EffectiveTime: time.Now().UTC(),
	})

	if err != nil {
		t.Fatalf("unexpected error mastering record: %v", err)
	}

	if len(res.MerkleVersionSeal) == 0 {
		t.Errorf("expected Merkle version seal to be populated")
	}

	if res.GoldenID == uuid.Nil {
		t.Errorf("expected golden ID to be assigned")
	}
}
