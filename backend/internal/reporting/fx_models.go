package reporting

import (
	"time"

	"github.com/google/uuid"
)

type CurrencyCode string

type CurrencyTier string

const (
	TierLocal        CurrencyTier = "LCY" // Instrument native currency
	TierBase         CurrencyTier = "BCY" // Portfolio accounting currency
	TierPresentation CurrencyTier = "PCY" // Client statement reporting currency
)

type FXRatePoint struct {
	FromCurrency CurrencyCode `json:"fromCurrency"`
	ToCurrency   CurrencyCode `json:"toCurrency"`
	Rate         float64      `json:"rate"`
	AsOfDate     time.Time    `json:"asOfDate"`
}

type PositionHolding struct {
	PositionID     string       `json:"positionId"`
	SecurityName   string       `json:"securityName"`
	LocalCurrency  CurrencyCode `json:"localCurrency"`
	Quantity       float64      `json:"quantity"`
	StartPriceLCY  float64      `json:"startPriceLcy"`
	EndPriceLCY    float64      `json:"endPriceLcy"`
	StartValLCY    float64      `json:"startValLcy"`
	EndValLCY      float64      `json:"endValLcy"`
}

type CurrencyAttributionBreakdown struct {
	PositionID        string       `json:"positionId"`
	SecurityName      string       `json:"securityName"`
	LocalCurrency     CurrencyCode `json:"localCurrency"`
	BaseCurrency      CurrencyCode `json:"baseCurrency"`
	ReportCurrency    CurrencyCode `json:"reportCurrency"`
	
	// Value Projections
	StartValReport    float64      `json:"startValReport"`
	EndValReport      float64      `json:"endValReport"`
	NetGainReport     float64      `json:"netGainReport"`
	
	// Returns & 3-Factor Attribution
	TotalReturnPct    float64      `json:"totalReturnPct"`    // RT
	AssetReturnPct    float64      `json:"assetReturnPct"`    // RL
	CurrencyReturnPct float64      `json:"currencyReturnPct"` // RC
	InteractionPct    float64      `json:"interactionPct"`    // RL * RC
	
	// FX Rate Markers
	StartFXRate       float64      `json:"startFxRate"` // LCY -> PCY at t0
	EndFXRate         float64      `json:"endFxRate"`   // LCY -> PCY at t1
}

type PortfolioFXAttributionSummary struct {
	TenantID             uuid.UUID                      `json:"tenantId"`
	BaseCurrency         CurrencyCode                   `json:"baseCurrency"`
	ReportCurrency       CurrencyCode                   `json:"reportCurrency"`
	StartDate            time.Time                      `json:"startDate"`
	EndDate              time.Time                      `json:"endDate"`
	TotalStartValReport  float64                        `json:"totalStartValReport"`
	TotalEndValReport    float64                        `json:"totalEndValReport"`
	PortfolioTotalReturn float64                        `json:"portfolioTotalReturn"`
	PortfolioAssetReturn float64                        `json:"portfolioAssetReturn"`
	PortfolioFXReturn    float64                        `json:"portfolioFxReturn"`
	PortfolioInteraction float64                        `json:"portfolioInteraction"`
	Holdings             []CurrencyAttributionBreakdown `json:"holdings"`
}
