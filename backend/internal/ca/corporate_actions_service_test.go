package ca

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestCorporateActionsService(t *testing.T) {
	svc := NewCorporateActionsService(nil)

	res, err := svc.ExecuteExDateProcessing(
		context.Background(),
		uuid.New(),
		uuid.New(),
		128.40,
	)

	if err != nil {
		t.Fatalf("unexpected error executing ex-date processing: %v", err)
	}

	if res.TotalGrossShares <= 0 {
		t.Errorf("expected post-split gross shares > 0, got %f", res.TotalGrossShares)
	}

	if len(res.MerkleExecutionSeal) == 0 {
		t.Errorf("expected Merkle execution seal to be populated")
	}
}
