package drift

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestAutoHealerService(t *testing.T) {
	svc := NewAutoHealerService(nil)

	res, err := svc.EvaluateAndProposeFix(
		context.Background(),
		uuid.New(),
		uuid.New(),
		uuid.New(),
		uuid.New(),
	)

	if err != nil {
		t.Fatalf("unexpected error evaluating auto-heal candidate: %v", err)
	}

	if res.CosineSimilarity < 0.88 {
		t.Errorf("expected high cosine similarity candidate, got %f", res.CosineSimilarity)
	}

	if !res.SyntheticQueryPass {
		t.Errorf("expected synthetic query AST verification to pass")
	}

	if len(res.RemediationSQL) == 0 {
		t.Errorf("expected remediation SQL to be generated")
	}
}
