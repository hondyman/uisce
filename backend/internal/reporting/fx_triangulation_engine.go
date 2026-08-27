package reporting

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type FXTriangulationEngine struct {
	anchorCurrency CurrencyCode
}

func NewFXTriangulationEngine(anchor CurrencyCode) *FXTriangulationEngine {
	if anchor == "" {
		anchor = "USD"
	}
	return &FXTriangulationEngine{anchorCurrency: anchor}
}

// ResolveCrossRate computes the FX conversion rate from LCY to PCY via anchor triangulation
func (e *FXTriangulationEngine) ResolveCrossRate(
	from CurrencyCode,
	to CurrencyCode,
	ratesMap map[CurrencyCode]float64,
) (float64, error) {
	if from == to {
		return 1.0, nil
	}

	fromRate, fromExists := ratesMap[from]
	toRate, toExists := ratesMap[to]

	if !fromExists {
		return 0, fmt.Errorf("missing FX rate quote for currency: %s", from)
	}
	if !toExists {
		return 0, fmt.Errorf("missing FX rate quote for currency: %s", to)
	}

	if fromRate <= 0 {
		return 0, fmt.Errorf("invalid non-positive FX rate for currency: %s", from)
	}

	// Triangulation: FX(From -> To) = Rate(Anchor -> To) / Rate(Anchor -> From)
	return toRate / fromRate, nil
}

// ComputePortfolioAttribution decomposes portfolio returns across Local, Base, and Presentation currencies
func (e *FXTriangulationEngine) ComputePortfolioAttribution(
	ctx context.Context,
	tenantID uuid.UUID,
	baseCCY CurrencyCode,
	presentationCCY CurrencyCode,
	startDate time.Time,
	endDate time.Time,
	holdings []PositionHolding,
	startAnchorRates map[CurrencyCode]float64,
	endAnchorRates map[CurrencyCode]float64,
) (*PortfolioFXAttributionSummary, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	summary := &PortfolioFXAttributionSummary{
		TenantID:       tenantID,
		BaseCurrency:   baseCCY,
		ReportCurrency: presentationCCY,
		StartDate:      startDate,
		EndDate:        endDate,
		Holdings:       make([]CurrencyAttributionBreakdown, len(holdings)),
	}

	var totalStartVal, totalEndVal float64
	var weightedAssetReturn, weightedCurrencyReturn, weightedInteraction float64

	for i, h := range holdings {
		// 1. Resolve Triangulated Cross-Rates at t0 and t1
		startRate, err := e.ResolveCrossRate(h.LocalCurrency, presentationCCY, startAnchorRates)
		if err != nil {
			return nil, fmt.Errorf("failed resolving t0 FX rate for position %s: %w", h.PositionID, err)
		}

		endRate, err := e.ResolveCrossRate(h.LocalCurrency, presentationCCY, endAnchorRates)
		if err != nil {
			return nil, fmt.Errorf("failed resolving t1 FX rate for position %s: %w", h.PositionID, err)
		}

		// 2. Compute Values in Presentation Currency
		startValReport := h.StartValLCY * startRate
		endValReport := h.EndValLCY * endRate
		netGainReport := endValReport - startValReport

		// 3. Compute 3-Factor Attribution Returns
		assetReturn := 0.0
		if h.StartValLCY > 0 {
			assetReturn = (h.EndValLCY - h.StartValLCY) / h.StartValLCY
		}

		fxReturn := (endRate - startRate) / startRate
		interaction := assetReturn * fxReturn
		totalReturn := assetReturn + fxReturn + interaction

		summary.Holdings[i] = CurrencyAttributionBreakdown{
			PositionID:        h.PositionID,
			SecurityName:      h.SecurityName,
			LocalCurrency:     h.LocalCurrency,
			BaseCurrency:      baseCCY,
			ReportCurrency:    presentationCCY,
			StartValReport:    startValReport,
			EndValReport:      endValReport,
			NetGainReport:     netGainReport,
			TotalReturnPct:    totalReturn,
			AssetReturnPct:    assetReturn,
			CurrencyReturnPct: fxReturn,
			InteractionPct:    interaction,
			StartFXRate:       startRate,
			EndFXRate:         endRate,
		}

		totalStartVal += startValReport
		totalEndVal += endValReport
	}

	// 4. Calculate Portfolio-Level Weighted Contributions
	summary.TotalStartValReport = totalStartVal
	summary.TotalEndValReport = totalEndVal

	if totalStartVal > 0 {
		summary.PortfolioTotalReturn = (totalEndVal - totalStartVal) / totalStartVal

		for _, item := range summary.Holdings {
			weight := item.StartValReport / totalStartVal
			weightedAssetReturn += item.AssetReturnPct * weight
			weightedCurrencyReturn += item.CurrencyReturnPct * weight
			weightedInteraction += item.InteractionPct * weight
		}
	}

	summary.PortfolioAssetReturn = weightedAssetReturn
	summary.PortfolioFXReturn = weightedCurrencyReturn
	summary.PortfolioInteraction = weightedInteraction

	return summary, nil
}
