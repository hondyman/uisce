package ai

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestJITContextPruner(t *testing.T) {
	pruner := NewJITContextPruner(nil)

	res, err := pruner.PruneContext(
		context.Background(),
		uuid.New(),
		"PORTFOLIO_MANAGER",
		[]float32{0.1, 0.2, 0.3},
		1200,
	)

	if err != nil {
		t.Fatalf("unexpected error pruning context: %v", err)
	}

	if res.TotalExtractedNodes <= 0 {
		t.Errorf("expected extracted nodes > 0, got %d", res.TotalExtractedNodes)
	}

	if len(res.ActiveDirectives) == 0 {
		t.Errorf("expected active directives to be populated")
	}

	if res.ExtractionLatencyMs <= 0 {
		t.Errorf("expected measured extraction latency")
	}
}
