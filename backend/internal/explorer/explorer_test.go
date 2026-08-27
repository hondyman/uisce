package explorer

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestSearchIQParser(t *testing.T) {
	parser := NewSearchIQParser(nil)
	ctx := context.Background()
	tenantID := uuid.New()
	boID := uuid.New()

	ast, err := parser.ParseSearchQuery(ctx, tenantID, boID, "top sector by market_value monthly")
	if err != nil {
		t.Fatalf("unexpected error parsing query: %v", err)
	}

	if len(ast.Tokens) == 0 {
		t.Errorf("expected parsed tokens, got 0")
	}

	hasMeasure := false
	hasDimension := false
	for _, tk := range ast.Tokens {
		if tk.TokenType == TokenMeasure {
			hasMeasure = true
		}
		if tk.TokenType == TokenDimension {
			hasDimension = true
		}
	}

	if !hasMeasure || !hasDimension {
		t.Errorf("expected both measure and dimension tokens parsed")
	}
}

func TestSpotIQEngine(t *testing.T) {
	engine := NewSpotIQEngine(nil)
	ctx := context.Background()
	tenantID := uuid.New()
	boID := uuid.New()

	baseline := map[string]float64{
		"Tech": 100.0,
		"Energy": 50.0,
	}
	compare := map[string]float64{
		"Tech": 80.0,
		"Energy": 55.0,
	}

	resp, err := engine.DecomposeMetricVariance(ctx, tenantID, boID, "revenue", "sector", baseline, compare)
	if err != nil {
		t.Fatalf("unexpected error in SpotIQ: %v", err)
	}

	if len(resp.TopContributors) != 2 {
		t.Errorf("expected 2 contributors, got %d", len(resp.TopContributors))
	}

	if resp.TotalVariancePct >= 0 {
		t.Errorf("expected negative variance, got %f", resp.TotalVariancePct)
	}
}
