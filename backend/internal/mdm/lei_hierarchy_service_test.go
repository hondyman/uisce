package mdm

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestLEIHierarchyService(t *testing.T) {
	svc := NewLEIHierarchyService(nil)
	ctx := context.Background()
	tenantID := uuid.New()
	startNodeID := uuid.New()

	nodes, err := svc.TraverseUltimateParent(ctx, tenantID, startNodeID)
	if err != nil {
		t.Fatalf("unexpected error traversing LEI hierarchy: %v", err)
	}

	if len(nodes) != 2 {
		t.Fatalf("expected 2 hierarchy nodes, got %d", len(nodes))
	}

	if nodes[0].Depth != 0 || nodes[1].Depth != 1 {
		t.Errorf("expected depths 0 and 1, got %d and %d", nodes[0].Depth, nodes[1].Depth)
	}
}
