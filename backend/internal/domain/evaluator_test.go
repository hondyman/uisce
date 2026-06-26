package domain

import (
	"context"
	"testing"
)

type fakeClaims struct{ claims map[string][]EffectiveClaim }

func (f *fakeClaims) EffectiveClaims(ctx context.Context, userID, tenantID, assetID string) ([]EffectiveClaim, error) {
	key := tenantID + ":" + userID
	if claims, ok := f.claims[key]; ok {
		// Filter claims by assetID
		var filtered []EffectiveClaim
		for _, claim := range claims {
			if claim.AssetID == assetID {
				filtered = append(filtered, claim)
			}
		}
		return filtered, nil
	}
	return []EffectiveClaim{}, nil
}

func TestEvaluator_TableDriven(t *testing.T) {
	f := &fakeClaims{claims: map[string][]EffectiveClaim{
		"acme:patrick": {
			{AssetID: "asset-orders", Permission: PermRead, Scope: []string{"metrics"}, Source: "manual"},
			{AssetID: "asset-margin", Permission: PermRead, Scope: []string{"metrics"}, Source: "bundle"},
		},
	}}
	ev := &SimpleEvaluator{service: f}

	tests := []struct {
		name         string
		req          EvaluationRequest
		wantDecision bool
	}{
		{"allow_read_orders", EvaluationRequest{UserID: "patrick", TenantID: "acme", AssetID: "asset-orders", Action: PermRead}, true},
		{"deny_update_orders", EvaluationRequest{UserID: "patrick", TenantID: "acme", AssetID: "asset-orders", Action: PermUpdate}, false},
		{"deny_unknown_asset", EvaluationRequest{UserID: "patrick", TenantID: "acme", AssetID: "asset-unknown", Action: PermRead}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, _, err := ev.Evaluate(context.Background(), tt.req)
			if err != nil {
				t.Fatalf("Evaluate error: %v", err)
			}
			if got != tt.wantDecision {
				t.Fatalf("decision=%v want=%v", got, tt.wantDecision)
			}
		})
	}
}
