package bo

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestBindingDiscoveryService(t *testing.T) {
	svc := NewBindingDiscoveryService(nil)
	ctx := context.Background()
	tenantID := uuid.New()
	drivingNodeID := uuid.New()

	terms, err := svc.DiscoverEligibleTermsForBinding(ctx, tenantID, drivingNodeID)
	if err != nil {
		t.Fatalf("unexpected error discovering terms: %v", err)
	}

	if len(terms) != 3 {
		t.Fatalf("expected 3 terms, got %d", len(terms))
	}

	if terms[0].TermKey != "customer_bk" {
		t.Errorf("expected termKey customer_bk, got %s", terms[0].TermKey)
	}

	if len(terms[0].Mappings) == 0 || terms[0].Mappings[0].ColumnName != "CustomerID" {
		t.Errorf("expected column mapping CustomerID")
	}
}
