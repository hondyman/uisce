package drift_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/drift"
)

func TestRemediationMatcher_SynonymMatching(t *testing.T) {
	matcher := drift.NewDriftRemediationMatcher(nil)
	tenantID := uuid.New()
	tableNodeID := uuid.New()

	candidates, err := matcher.FindCandidateMatches(context.Background(), tenantID, tableNodeID, "px_last")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(candidates) != 0 {
		t.Logf("Found %d candidate matches in nil db context", len(candidates))
	}

	_, nilTenantErr := matcher.FindCandidateMatches(context.Background(), uuid.Nil, tableNodeID, "px_last")
	if nilTenantErr == nil {
		t.Fatalf("expected Rule 7 violation error on nil tenant_id")
	}
}
