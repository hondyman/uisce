package vendorcatalog

import (
	"context"
	"testing"
)

func TestVendorIngestionService(t *testing.T) {
	svc := NewVendorIngestionService(nil)

	records := []VendorFieldRecord{
		{
			Mnemonic:    "PX_LAST",
			FieldName:   "Last Price",
			Category:    "Pricing",
			FeedType:    "Data License",
			DataType:    "NUMERIC(18,6)",
			Description: "Official closing transaction price",
			Aliases:     []string{"last_price", "close_px", "closing_price"},
		},
		{
			Mnemonic:    "ID_ISIN",
			FieldName:   "ISIN Identifier",
			Category:    "Symbology",
			FeedType:    "Data License",
			DataType:    "VARCHAR(12)",
			Description: "International Securities Identification Number",
			Aliases:     []string{"isin", "isin_code"},
		},
	}

	count, err := svc.IngestVendorDictionary(context.Background(), "BLOOMBERG", records)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 records ingested, got %d", count)
	}

	candidates, err := svc.FindCandidateVendorFields(context.Background(), make([]float32, 1536), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(candidates) == 0 {
		t.Errorf("expected candidate records")
	}
}
