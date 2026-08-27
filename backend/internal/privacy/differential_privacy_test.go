package privacy

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestDifferentialPrivacyService_ApplyLaplaceNoise(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	service := NewDifferentialPrivacyService()

	rawFundReturn := 14.24 // 14.24%
	budget := PrivacyBudget{
		Epsilon: 0.5,
		Delta:   1e-5,
	}

	result, err := service.ApplyLaplaceNoise(ctx, tenantID, "peer_fund_yield", rawFundReturn, 0.05, budget)
	if err != nil {
		t.Fatalf("unexpected error applying Laplace noise: %v", err)
	}

	if result.EpsilonUsed != 0.5 {
		t.Errorf("expected EpsilonUsed 0.5, got %.2f", result.EpsilonUsed)
	}
	if result.ZKPassportHash == "" {
		t.Errorf("expected non-empty ZKPassportHash")
	}
	if result.NoisyValue == 0.0 {
		t.Errorf("expected non-zero noisy value")
	}
}

func TestDifferentialPrivacyService_Rule7TenantValidation(t *testing.T) {
	ctx := context.Background()
	service := NewDifferentialPrivacyService()

	budget := PrivacyBudget{Epsilon: 0.5}
	_, err := service.ApplyLaplaceNoise(ctx, uuid.Nil, "test_metric", 10.0, 1.0, budget)
	if err == nil {
		t.Fatalf("expected error on nil tenant_id, got nil")
	}
}
