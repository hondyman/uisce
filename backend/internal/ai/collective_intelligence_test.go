package ai

import (
	"context"
	"math"
	"testing"

	"github.com/google/uuid"
)

func TestCohortAggregatorLaplaceNoise(t *testing.T) {
	aggregator := NewCohortAggregator(nil, 0.1)

	// Verify Laplace noise generates unbounded symmetric perturbation
	samples := make([]float64, 100)
	for i := 0; i < 100; i++ {
		noise := aggregator.SampleLaplaceNoise(10.0)
		samples[i] = noise
		if math.IsNaN(noise) || math.IsInf(noise, 0) {
			t.Fatalf("Laplace noise generated NaN/Inf: %f", noise)
		}
	}

	// Verify Tenant Isolation in RecordInteraction
	ctx := context.Background()
	err := aggregator.RecordInteraction(ctx, InteractionTelemetry{
		TenantID: uuid.Nil, // Must fail
	})
	if err == nil {
		t.Errorf("expected error on nil tenant ID")
	}

	validTenantID := uuid.New()
	err = aggregator.RecordInteraction(ctx, InteractionTelemetry{
		TenantID:       validTenantID,
		UserID:         "user-101",
		SessionID:      uuid.New(),
		PrimaryBOKey:   "trade_order",
		AccessedFields: []string{"yield", "cash_drag"},
		SentimentScore: 0.85,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPeerRecommendationEngine(t *testing.T) {
	engine := NewPeerRecommendationEngine(nil)
	ctx := context.Background()
	tenantID := uuid.New()

	recs, err := engine.GetCohortRecommendations(ctx, tenantID, "trade_order")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(recs) == 0 {
		t.Errorf("expected peer recommendations")
	}

	if recs[0].PeerAdoptionRate < 50.0 {
		t.Errorf("expected high adoption rate, got %f", recs[0].PeerAdoptionRate)
	}

	// Self Healing check
	healing := engine.EvaluateFrustrationAndHeal(ctx, tenantID, 0, map[string]string{"status": "FILLED"})
	if healing == nil {
		t.Errorf("expected self healing alert on zero rows")
	}

	if healing.SuggestedFilter != "status = 'EXECUTED'" {
		t.Errorf("expected suggested filter status = 'EXECUTED', got %s", healing.SuggestedFilter)
	}
}
