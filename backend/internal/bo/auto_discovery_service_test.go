package bo

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestAutoDiscoveryService(t *testing.T) {
	svc := NewAutoDiscoveryService(nil)

	res, err := svc.InspectDrivingTable(context.Background(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error inspecting driving table: %v", err)
	}

	if len(res.EligibleTerms) == 0 {
		t.Errorf("expected discovered eligible terms")
	}

	if res.PKColumnName == "" {
		t.Errorf("expected discovered PK column name")
	}
}
