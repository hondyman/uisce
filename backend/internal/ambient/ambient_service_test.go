package ambient

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestAmbientServiceSanityCheck(t *testing.T) {
	svc := NewAmbientService(nil, nil)

	tenantID := uuid.New()
	proposalID, err := svc.IngestRawMessage(
		context.Background(),
		tenantID,
		"SLACK",
		"lead_data_architect",
		"For CRM data, use Affinity for USCAN deals from 2025, else Salesforce",
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if proposalID == uuid.Nil {
		t.Fatalf("expected non-nil proposalID")
	}

	report := svc.runSanityCheck(
		context.Background(),
		tenantID,
		[]string{"region", "created_at"},
		"CASE WHEN region = 'USCAN' THEN 'affinity' ELSE 'salesforce' END",
	)

	if !report.SQLSyntaxValid {
		t.Errorf("expected SQL syntax to be valid")
	}
	if !report.GraphResolved {
		t.Errorf("expected GraphResolved to be true")
	}
}
