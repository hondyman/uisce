package position

import (
	"testing"

	"github.com/google/uuid"
)

func TestPositionRecord_Validate_ValidSubtypes(t *testing.T) {
	validSubtypes := []string{"settled_long", "derivative_exposure", "pledged_collateral", "unsettled_pipeline"}
	for _, s := range validSubtypes {
		rec := PositionRecord{SubtypeCode: s}
		if err := rec.Validate(); err != nil {
			t.Errorf("expected subtype %q to be valid, got %v", s, err)
		}
	}
}

func TestPositionRecord_Validate_InvalidSubtype(t *testing.T) {
	rec := PositionRecord{SubtypeCode: "invalid_type"}
	if err := rec.Validate(); err != ErrInvalidSubtype {
		t.Errorf("expected ErrInvalidSubtype, got %v", err)
	}
}

func TestPositionRecord_Validate_ShortBorrowedRequiresPrimeBroker(t *testing.T) {
	rec := PositionRecord{SubtypeCode: "short_borrowed"}
	if err := rec.Validate(); err != ErrRequiresPrimeBroker {
		t.Errorf("expected ErrRequiresPrimeBroker, got %v", err)
	}
}

func TestPositionRecord_Validate_ShortBorrowedWithPrimeBroker(t *testing.T) {
	pbID := uuid.New()
	rec := PositionRecord{SubtypeCode: "short_borrowed", PrimeBrokerID: &pbID}
	if err := rec.Validate(); err != nil {
		t.Errorf("expected valid, got %v", err)
	}
}

func TestPositionRecord_Validate_SettledLong(t *testing.T) {
	rec := PositionRecord{SubtypeCode: "settled_long"}
	if err := rec.Validate(); err != nil {
		t.Errorf("expected valid, got %v", err)
	}
}
