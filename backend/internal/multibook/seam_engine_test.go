package multibook

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestMultiBookSeamEngine(t *testing.T) {
	svc := NewMultiBookSeamService(nil)

	tenantID := uuid.New()
	accountID := uuid.New()
	secID := uuid.New()

	// 1. Intraday Fill Projection (IBOR)
	entry, err := svc.RecordIntradayFill(
		context.Background(),
		tenantID,
		accountID,
		secID,
		1000,
		150.25,
		"BUY",
	)

	if err != nil {
		t.Fatalf("unexpected error recording intraday fill: %v", err)
	}

	if entry.BookType != BookIBOR {
		t.Errorf("expected BookIBOR, got %s", entry.BookType)
	}

	if entry.CashDelta != -150250.00 {
		t.Errorf("expected cash delta -150250, got %.2f", entry.CashDelta)
	}

	if len(entry.MerkleEntryHash) == 0 {
		t.Errorf("expected Merkle entry hash to be generated")
	}

	// 2. Tax-Alpha In-Memory Rebalance
	rebal, err := svc.ExecuteTaxAlphaHarvesting(
		context.Background(),
		tenantID,
		uuid.New(),
		5000.0,
	)

	if err != nil {
		t.Fatalf("unexpected error executing tax alpha: %v", err)
	}

	if rebal.GrossLossHarvested <= 0 {
		t.Errorf("expected gross loss harvested > 0")
	}

	if rebal.WashSalePrevented <= 0 {
		t.Errorf("expected wash sale conflicts prevented")
	}
}
