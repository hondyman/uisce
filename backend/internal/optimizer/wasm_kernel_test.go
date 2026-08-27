package optimizer

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestWASMOptimizerEngine(t *testing.T) {
	engine := NewWASMOptimizerEngine(nil)

	res, err := engine.ExecuteTaxLossHarvesting(
		context.Background(),
		uuid.New(),
		uuid.New(),
		uuid.New(),
		5000.0,
	)

	if err != nil {
		t.Fatalf("unexpected error executing tax loss harvesting: %v", err)
	}

	if res.HarvestedTaxUSD <= 0 {
		t.Errorf("expected harvested tax > 0, got %f", res.HarvestedTaxUSD)
	}

	if len(res.GeneratedTickets) == 0 {
		t.Errorf("expected generated tickets > 0")
	}

	if len(res.MerkleRoot) == 0 {
		t.Errorf("expected Merkle root hash")
	}

	if res.SolverLatencyMs <= 0 {
		t.Errorf("expected measured solver latency")
	}
}
