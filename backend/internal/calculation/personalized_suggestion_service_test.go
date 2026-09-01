package calculation

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestPersonalizedSuggestionServiceWithFeedback(t *testing.T) {
	svc := NewPersonalizedSuggestionService(nil)
	ctx := context.Background()
	tenantID := uuid.New()
	userID := "user-risk-analyst-01"

	suggestions, err := svc.GetSuggestionsForUser(ctx, tenantID, userID, "trade_order")
	if err != nil {
		t.Fatalf("unexpected error fetching suggestions: %v", err)
	}

	if len(suggestions) == 0 {
		t.Fatalf("expected suggestions, got 0")
	}

	// 1. Test Bayesian Posterior Score Calibration
	prior := 0.90
	weightBoost := 1.10
	posterior := svc.CalculatePosteriorConfidence(prior, weightBoost)
	if posterior <= prior {
		t.Errorf("expected boosted posterior confidence, got %f", posterior)
	}

	// 2. Test Dismissal & Negative Reinforcement
	err = svc.DismissSuggestion(ctx, tenantID, userID, suggestions[0].SuggestedCalcKey, "trade_order")
	if err != nil {
		t.Fatalf("unexpected error dismissing suggestion: %v", err)
	}

	// 3. Test Acceptance & Positive Reinforcement
	err = svc.AcceptAndApplySuggestion(ctx, tenantID, userID, suggestions[1], true)
	if err != nil {
		t.Fatalf("unexpected error accepting suggestion: %v", err)
	}
}
