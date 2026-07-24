package domain

import (
	"context"
	"testing"
)

func TestSemanticPlannerAdapter_PlanQuery(t *testing.T) {
	tests := []struct {
		name          string
		req           EvaluationRequest
		mockClaims    []EffectiveClaim
		expectedHints int
		expectedAllow bool
	}{
		{
			name: "Generate hints for allowed request",
			req: EvaluationRequest{
				UserID:   "user1",
				TenantID: "tenant1",
				AssetID:  "asset1",
				Action:   PermRead,
			},
			mockClaims: []EffectiveClaim{
				{AssetID: "asset1", Permission: PermRead, Scope: []string{"tenant1"}, Source: "manual"},
			},
			expectedHints: 1, // Row pruning hint for tenant
			expectedAllow: true,
		},
		{
			name: "Block all rows for denied request",
			req: EvaluationRequest{
				UserID:   "user1",
				TenantID: "tenant1",
				AssetID:  "asset1",
				Action:   PermRead,
			},
			mockClaims:    []EffectiveClaim{},
			expectedHints: 1, // Block all rows hint
			expectedAllow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEvaluator := &mockEvaluator{claims: tt.mockClaims}
			mockChecker := &mockPolicyChecker{allow: tt.expectedAllow}
			adapter := &SemanticPlannerAdapter{
				Evaluator: mockEvaluator,
				Checker:   mockChecker,
			}

			hints, err := adapter.PlanQuery(context.Background(), tt.req, "SELECT * FROM table1")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(hints) != tt.expectedHints {
				t.Errorf("expected %d hints, got %d", tt.expectedHints, len(hints))
			}
		})
	}
}

// Mock implementations for semantic planner adapter test
type mockEvaluator struct {
	claims []EffectiveClaim
}

func (m *mockEvaluator) Evaluate(ctx context.Context, req EvaluationRequest) (bool, string, []EffectiveClaim, error) {
	if len(m.claims) == 0 {
		return false, "No claims", nil, nil
	}
	return true, "Allowed", m.claims, nil
}

type mockPolicyChecker struct {
	allow bool
}

func (m *mockPolicyChecker) Check(ctx context.Context, req EvaluationRequest, claims []EffectiveClaim) (bool, string, []map[string]any, []string, error) {
	if !m.allow {
		return false, "Denied", nil, nil, nil
	}
	return true, "Allowed", []map[string]any{{"policyId": "baseline_allow_read_or_action", "result": "pass"}}, []string{"tenant1"}, nil
}
