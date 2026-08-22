package trade_order

import (
	"testing"
)

func TestTradeOrderRecord_Validate_ValidSubtypes(t *testing.T) {
	validSubtypes := []string{"block_parent", "dma_execution", "otc_bilateral", "fx_spot_forward", "primary_auction"}
	for _, s := range validSubtypes {
		rec := TradeOrderRecord{SubtypeCode: s}
		if err := rec.Validate(); err != nil {
			t.Errorf("expected subtype %q to be valid, got %v", s, err)
		}
	}
}

func TestTradeOrderRecord_Validate_InvalidSubtype(t *testing.T) {
	rec := TradeOrderRecord{SubtypeCode: "invalid_type"}
	if err := rec.Validate(); err != ErrInvalidSubtype {
		t.Errorf("expected ErrInvalidSubtype, got %v", err)
	}
}
