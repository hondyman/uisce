package dq

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestDataQualityScorecardService(t *testing.T) {
	svc := NewDataQualityScorecardService(nil)
	ctx := context.Background()
	tenantID := uuid.New()

	snapshots, err := svc.ComputeDomainHealthSnapshots(ctx, tenantID)
	if err != nil {
		t.Fatalf("unexpected error computing DQ snapshots: %v", err)
	}

	if len(snapshots) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(snapshots))
	}

	if snapshots[0].CompositeHealthScore != 98.4 {
		t.Errorf("expected score 98.4, got %f", snapshots[0].CompositeHealthScore)
	}
}
