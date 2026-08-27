package mdm

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestCorporateActionService(t *testing.T) {
	svc := NewCorporateActionService(nil)
	ctx := context.Background()
	tenantID := uuid.New()

	payload := CorporateActionPayload{
		TenantID:             tenantID,
		SourceIdentifierType: "ISIN",
		SourceIdentifierVal:  "US67066G1040",
		ActionType:           "SPLIT",
		EffectiveDate:        "2026-09-01",
		AnnouncementSource:   "DTCC",
		Terms: map[string]interface{}{
			"ratio":       10.0,
			"description": "10-for-1 Stock Split",
		},
	}

	actionID, err := svc.PropagateCorporateAction(ctx, payload)
	if err != nil {
		t.Fatalf("unexpected error propagating corporate action: %v", err)
	}

	if actionID == uuid.Nil {
		t.Errorf("expected non-nil action ID")
	}
}
