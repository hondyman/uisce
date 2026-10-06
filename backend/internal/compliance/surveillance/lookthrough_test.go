package surveillance

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestMultiAssetLookthroughAggregator_DecomposesFundsAndDerivatives(t *testing.T) {
	appleIssuerID := uuid.New()
	etfSecurityID := uuid.New()
	accountID := uuid.New()
	accountNAV := decimal.NewFromInt(1000000) // $1,000,000 NAV

	constituents := []FundConstituent{
		{
			FundSecurityID: etfSecurityID,
			ConstituentID:  uuid.New(),
			IssuerID:       appleIssuerID,
			Weight:         decimal.RequireFromString("0.075"), // 7.5% weight in ETF
		},
	}

	aggregator := NewMultiAssetLookthroughAggregator(constituents)

	positions := []AssetPosition{
		// 1. Direct Equity Holding: $50,000
		{
			PositionID:  uuid.New(),
			AccountID:   accountID,
			SecurityID:  uuid.New(),
			IssuerID:    appleIssuerID,
			AssetClass:  "EQUITY",
			MarketValue: decimal.NewFromInt(50000),
		},
		// 2. ETF Holding: $200,000 -> 7.5% in Apple = $15,000 lookthrough value
		{
			PositionID:  uuid.New(),
			AccountID:   accountID,
			SecurityID:  etfSecurityID,
			IssuerID:    uuid.New(), // ETF Issuer
			AssetClass:  "ETF",
			MarketValue: decimal.NewFromInt(200000),
		},
		// 3. Option Position: $40,000 notional with 0.50 delta -> $20,000 effective exposure
		{
			PositionID:  uuid.New(),
			AccountID:   accountID,
			SecurityID:  uuid.New(),
			IssuerID:    appleIssuerID,
			AssetClass:  "OPTION",
			MarketValue: decimal.NewFromInt(40000),
			Delta:       decimal.RequireFromString("0.50"),
		},
	}

	summaries, err := aggregator.AggregateIssuerExposures(context.Background(), accountNAV, positions)
	if err != nil {
		t.Fatalf("AggregateIssuerExposures failed: %v", err)
	}

	appleSummary, ok := summaries[appleIssuerID]
	if !ok {
		t.Fatalf("Expected Apple issuer summary")
	}

	// Total Expected = $50,000 (direct) + $15,000 (lookthrough) + $20,000 (options) = $85,000
	expectedTotal := decimal.NewFromInt(85000)
	if !appleSummary.TotalExposure.Equal(expectedTotal) {
		t.Errorf("Expected total exposure %s, got %s", expectedTotal, appleSummary.TotalExposure)
	}

	// % of NAV = 85,000 / 1,000,000 * 100 = 8.5%
	expectedNAV := decimal.RequireFromString("8.5")
	if !appleSummary.PercentOfNAV.Equal(expectedNAV) {
		t.Errorf("Expected percent of NAV %s%%, got %s%%", expectedNAV, appleSummary.PercentOfNAV)
	}
}
