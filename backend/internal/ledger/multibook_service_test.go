package ledger

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMultiBookSynchronizer(t *testing.T) {
	svc := NewMultiBookSynchronizerService(nil)

	tenantID := uuid.New()
	portfolioID := uuid.New()
	securityID := uuid.New()
	accountID := uuid.New()

	err := svc.ProcessTradeFill(context.Background(), &TradeFillEvent{
		TenantID:        tenantID,
		PortfolioNodeID: portfolioID,
		SecurityNodeID:  securityID,
		AccountNodeID:   accountID,
		Side:            "BUY",
		Shares:          1000,
		Price:           145.50,
		GrossAmount:     145500.00,
		Commission:      25.00,
		TradeDate:       time.Now().UTC(),
		SettleDate:      time.Now().UTC().AddDate(0, 0, 2),
	})

	if err != nil {
		t.Fatalf("unexpected error processing trade fill: %v", err)
	}

	err = svc.ReconcileCustodianPosition(context.Background(), tenantID, &CustodianPositionRecord{
		PortfolioNodeID: portfolioID,
		SecurityNodeID:  securityID,
		Shares:          25000,
		SourceFile:      "BNY_MELLON_MT535_20260825",
	}, 0.001)

	if err != nil {
		t.Fatalf("unexpected error reconciling position: %v", err)
	}
}
