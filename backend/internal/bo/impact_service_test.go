package bo

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestImpactSimulatorService(t *testing.T) {
	svc := NewImpactSimulatorService(nil)

	res, err := svc.SimulateMutationBlastRadius(
		context.Background(),
		uuid.New(),
		uuid.New(),
		"FORMULA_UPDATE",
		"order_total * 0.85",
	)

	if err != nil {
		t.Fatalf("unexpected error simulating blast radius: %v", err)
	}

	if res.TotalImpacted == 0 {
		t.Errorf("expected downstream impacted nodes")
	}

	if res.OverallSeverity != "RED" {
		t.Errorf("expected RED overall severity due to compliance rule dependency, got %s", res.OverallSeverity)
	}

	if res.AutoDraftShim == "" {
		t.Errorf("expected auto draft shim to be populated")
	}
}
