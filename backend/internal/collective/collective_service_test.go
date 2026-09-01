package collective

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCollectiveServiceShadowReplay(t *testing.T) {
	svc := NewCollectiveService(nil)

	res, err := svc.RunShadowReplay(
		context.Background(),
		uuid.New(),
		uuid.New(),
		"rule.routing.crm_affinity_vs_salesforce",
		90,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.TransactionsEvaluated <= 0 {
		t.Errorf("expected evaluated transactions > 0")
	}

	if !res.SMTInvariantPassed {
		t.Errorf("expected SMT invariant proof to pass")
	}
}

func TestStalenessDecay(t *testing.T) {
	svc := NewCollectiveService(nil)

	// Fresh rule
	scoreFresh, statusFresh := svc.CalculateStalenessDecay(time.Now(), 500)
	if scoreFresh < 90.0 || statusFresh != "ACTIVE" {
		t.Errorf("expected fresh rule to be ACTIVE with high score, got %v (%s)", scoreFresh, statusFresh)
	}

	// 200-day stale rule
	oldTime := time.Now().Add(-200 * 24 * time.Hour)
	scoreStale, statusStale := svc.CalculateStalenessDecay(oldTime, 0)
	if scoreStale > 30.0 || statusStale != "PENDING_DEPRECATION" {
		t.Errorf("expected old rule to be PENDING_DEPRECATION, got %v (%s)", scoreStale, statusStale)
	}
}
