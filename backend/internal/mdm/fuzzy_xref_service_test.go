package mdm

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestFuzzyXREFResolver(t *testing.T) {
	resolver := NewFuzzyXREFResolver(nil)
	ctx := context.Background()
	tenantID := uuid.New()

	// 1. Probabilistic Match
	inbound := VendorEntityAttributes{
		Name:        "Apple Operations Int.",
		Ticker:      "AAPL-INT",
		Country:     "IE",
		AssetClass:  "EQUITY",
		Description: "Irish holding subsidiary for international operations",
	}

	res, err := resolver.ResolveOrMatchEntity(ctx, tenantID, inbound)
	if err != nil {
		t.Fatalf("unexpected error resolving fuzzy entity: %v", err)
	}

	if res.ResolutionType != "PROBABILISTIC_VECTOR_MATCH" {
		t.Errorf("expected PROBABILISTIC_VECTOR_MATCH, got %s", res.ResolutionType)
	}

	if res.ConfidenceScore < 0.95 {
		t.Errorf("expected high confidence score >= 0.95, got %f", res.ConfidenceScore)
	}

	// 2. Unmatched Entity - Stages New Entity Provisioning
	unknownInbound := VendorEntityAttributes{
		Name:        "Totally Random Unregistered Corp",
		Ticker:      "TRUC-XYZ",
		Country:     "KY",
		AssetClass:  "EQUITY",
		Description: "Cayman special purpose vehicle",
	}

	res, err = resolver.ResolveOrMatchEntity(ctx, tenantID, unknownInbound)
	if err != nil {
		t.Fatalf("unexpected error resolving entity: %v", err)
	}

	if res.ResolutionType != "NEW_ENTITY_PROVISIONED" {
		t.Errorf("expected NEW_ENTITY_PROVISIONED, got %s", res.ResolutionType)
	}
}
