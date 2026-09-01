package regulatory

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestForm13FSynthesizer_Generate(t *testing.T) {
	synthesizer := NewForm13FSynthesizer(nil)

	tenantID := uuid.New()
	templateID := uuid.New()
	quarterEnd := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	knowledgeCutoff := time.Now().UTC()

	runID, xmlPayload, passport, err := synthesizer.GenerateForm13F(
		context.Background(),
		tenantID,
		templateID,
		quarterEnd,
		knowledgeCutoff,
		"compliance_officer_test",
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if runID == uuid.Nil {
		t.Errorf("expected valid runID")
	}

	if len(passport) != 64 {
		t.Errorf("expected 64-char SHA256 passport, got %s", passport)
	}

	if len(xmlPayload) == 0 {
		t.Errorf("expected non-empty XML payload")
	}
}
