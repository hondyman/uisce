package surveillance

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// TradeRecord represents a historical trade execution for wash-sale surveillance
type TradeRecord struct {
	ExecutionID       uuid.UUID
	TenantID          uuid.UUID
	AccountID         uuid.UUID
	BeneficialOwnerID uuid.UUID
	SecurityID        uuid.UUID
	Side              string // "BUY" or "SELL"
	Quantity          decimal.Decimal
	Price             decimal.Decimal
	CostBasis         decimal.Decimal
	RealizedGainLoss  decimal.Decimal
	ExecutedAt        time.Time
}

// WashSaleViolation records an identified wash-sale event under IRS 1091 rules (30-day window)
type WashSaleViolation struct {
	TenantID              uuid.UUID
	BeneficialOwnerID     uuid.UUID
	LossExecutionID       uuid.UUID
	LossAccountID         uuid.UUID
	LossExecutedAt        time.Time
	LossAmount            decimal.Decimal
	ReplacementExecID     uuid.UUID
	ReplacementAcctID     uuid.UUID
	ReplacementExecutedAt time.Time
	SecurityID            uuid.UUID
	WindowDays            int
	DetectedAt            time.Time
}

// WashSaleDetector scans trade history for loss-harvesting accompanied by replacement purchases within ±30 days
type WashSaleDetector struct {
	windowDays int
}

func NewWashSaleDetector(windowDays int) *WashSaleDetector {
	if windowDays <= 0 {
		windowDays = 30
	}
	return &WashSaleDetector{
		windowDays: windowDays,
	}
}

// DetectWashSales evaluates a slice of executions for a beneficial owner and identifies wash-sale violations
func (d *WashSaleDetector) DetectWashSales(ctx context.Context, trades []TradeRecord) ([]WashSaleViolation, error) {
	var violations []WashSaleViolation
	windowDuration := time.Duration(d.windowDays*24) * time.Hour

	// Group trades by (BeneficialOwnerID, SecurityID)
	type key struct {
		ownerID    uuid.UUID
		securityID uuid.UUID
	}
	grouped := make(map[key][]TradeRecord)
	for _, t := range trades {
		k := key{ownerID: t.BeneficialOwnerID, securityID: t.SecurityID}
		grouped[k] = append(grouped[k] , t)
	}

	for k, group := range grouped {
		// Identify loss sales
		for _, sale := range group {
			if sale.Side != "SELL" || sale.RealizedGainLoss.GreaterThanOrEqual(decimal.Zero) {
				continue // Only sales with realized loss trigger wash-sale review
			}

			// Scan for replacement purchases in the window [sale.ExecutedAt - 30d, sale.ExecutedAt + 30d]
			for _, buy := range group {
				if buy.Side != "BUY" || buy.ExecutionID == sale.ExecutionID {
					continue
				}

				// Post-sale repurchase within 30 days
				if buy.ExecutedAt.After(sale.ExecutedAt) && buy.ExecutedAt.Sub(sale.ExecutedAt) <= windowDuration {
					violations = append(violations, WashSaleViolation{
						TenantID:              sale.TenantID,
						BeneficialOwnerID:     k.ownerID,
						LossExecutionID:       sale.ExecutionID,
						LossAccountID:         sale.AccountID,
						LossExecutedAt:        sale.ExecutedAt,
						LossAmount:            sale.RealizedGainLoss.Abs(),
						ReplacementExecID:     buy.ExecutionID,
						ReplacementAcctID:     buy.AccountID,
						ReplacementExecutedAt: buy.ExecutedAt,
						SecurityID:            k.securityID,
						WindowDays:            d.windowDays,
						DetectedAt:            time.Now().UTC(),
					})
					break
				}
			}
		}
	}

	return violations, nil
}
