package wealth

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
)

func TestCalculatePEMetrics(t *testing.T) {
	service := &AlternativeInvestmentService{}
	ctx := context.Background()

	t.Run("TypicalPEInvestment", func(t *testing.T) {
		// capitalCalls total: 3M
		totalCalled := decimal.NewFromInt(3000000)
		// distributions total: 1.5M
		totalDistributed := decimal.NewFromInt(1500000)

		result, err := service.CalculatePEMetrics(
			ctx,
			"investment-123",
			decimal.NewFromInt(5000000), // $5M commitment
			totalCalled,
			totalDistributed,
			decimal.NewFromInt(4000000), // $4M current NAV
		)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify IRR (not calculated in simplified service, so we skip check)
		// Verify fields
		if !result.CapitalCalled.Equal(totalCalled) {
			t.Errorf("Expected called %s, got %s", totalCalled, result.CapitalCalled)
		}

		if !result.Distributions.Equal(totalDistributed) {
			t.Errorf("Expected distributions %s, got %s", totalDistributed, result.Distributions)
		}

		// Verify MOIC (Total Value / Capital Called) = (1.5M + 4M) / 3M = 1.8333
		expectedTotalValue := decimal.NewFromInt(5500000)
		if !result.TotalValue.Equal(expectedTotalValue) {
			t.Errorf("Expected total value %s, got %s", expectedTotalValue, result.TotalValue)
		}

		// 5.5 / 3.0 = 1.8333
		if result.MOIC.LessThan(decimal.NewFromFloat(1.8)) {
			t.Errorf("Expected MOIC around 1.83, got %s", result.MOIC)
		}

		// Verify DPI (Distributions to Paid-In) = 1.5 / 3.0 = 0.5
		if !result.DPI.Equal(decimal.NewFromFloat(0.5)) {
			t.Errorf("Expected DPI 0.5, got %s", result.DPI.String())
		}
	})

	t.Run("NoDistributions", func(t *testing.T) {
		result, err := service.CalculatePEMetrics(
			ctx,
			"investment-456",
			decimal.NewFromInt(5000000),
			decimal.NewFromInt(2000000),
			decimal.Zero, // No distributions
			decimal.NewFromInt(2500000),
		)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// DPI should be 0
		if !result.DPI.Equal(decimal.Zero) {
			t.Errorf("Expected DPI 0, got %s", result.DPI.String())
		}
	})
}

func TestModelVCExitScenarios(t *testing.T) {
	service := &AlternativeInvestmentService{}
	ctx := context.Background()

	t.Run("StandardVCInvestment", func(t *testing.T) {
		scenarios := service.ModelVCExitScenarios(
			ctx,
			decimal.NewFromInt(1000000), // $1M investment
			decimal.NewFromFloat(0.10),  // 10% ownership
		)

		// Should have 4 scenarios
		if len(scenarios) != 4 {
			t.Errorf("Expected 4 exit scenarios, got %d", len(scenarios))
		}

		// Verify scenario types
		expectedScenarios := map[string]bool{
			"IPO":                false,
			"ACQUISITION_STRONG": false,
			"ACQUISITION_MODEST": false,
			"WRITE_OFF":          false,
		}

		for _, s := range scenarios {
			expectedScenarios[s.Scenario] = true
		}

		for k, found := range expectedScenarios {
			if !found {
				t.Errorf("Missing scenario type: %s", k)
			}
		}

		// Check IPO scenario logic
		var ipoScenario *VCExitScenario
		for i := range scenarios {
			if scenarios[i].Scenario == "IPO" {
				ipoScenario = &scenarios[i]
				break
			}
		}

		if ipoScenario == nil {
			t.Fatal("IPO scenario not found")
		}

		// IPO valuation logic in service: Investment / Ownership * 25
		// 1M / 0.10 * 25 = 10M * 25 = 250M
		expectedVal := decimal.NewFromInt(250000000)
		if !ipoScenario.ExitValuation.Equal(expectedVal) {
			t.Errorf("Expected IPO valuation %s, got %s", expectedVal, ipoScenario.ExitValuation)
		}
	})
}

func TestCalculate1031ExchangeOpportunity(t *testing.T) {
	service := &AlternativeInvestmentService{}
	ctx := context.Background()

	t.Run("ProfitableProperty", func(t *testing.T) {
		result := service.Calculate1031ExchangeOpportunity(
			ctx,
			decimal.NewFromInt(8000000), // Property Value
			decimal.NewFromInt(5000000), // Cost Basis
			decimal.NewFromInt(8000000), // Expected Sale Price
		)

		// Capital gain = 8M - 5M = 3M
		expectedGain := decimal.NewFromInt(3000000)
		if !result.CapitalGain.Equal(expectedGain) {
			t.Errorf("Expected gain %s, got %s", expectedGain, result.CapitalGain)
		}

		// Tax deferred should be positive
		if result.TaxDeferred1031.LessThanOrEqual(decimal.Zero) {
			t.Error("Expected positive tax deferred amount")
		}

		// Net Proceeds 1031 should match sale price (full reinvestment)
		if !result.NetProceeds1031.Equal(decimal.NewFromInt(8000000)) {
			t.Errorf("Expected net proceeds 1031 to be 8M, got %s", result.NetProceeds1031)
		}
	})
}

func TestTrackArtAppreciation(t *testing.T) {
	service := &AlternativeInvestmentService{}
	ctx := context.Background()

	t.Run("AppreciatingArt", func(t *testing.T) {
		// 10M -> 15M over 5 years
		cagr := service.TrackArtAppreciation(
			ctx,
			decimal.NewFromInt(10000000),
			decimal.NewFromInt(15000000),
			5,
		)

		// 1.5 ^ (1/5) - 1 = 1.0844 - 1 = 8.44%
		if cagr.LessThan(decimal.NewFromFloat(8.0)) || cagr.GreaterThan(decimal.NewFromFloat(9.0)) {
			t.Errorf("Expected CAGR around 8.44%%, got %s", cagr)
		}
	})

	t.Run("DepreciatingArt", func(t *testing.T) {
		// 500k -> 400k over 3 years
		cagr := service.TrackArtAppreciation(
			ctx,
			decimal.NewFromInt(500000),
			decimal.NewFromInt(400000),
			3,
		)

		// Should be negative
		if cagr.GreaterThanOrEqual(decimal.Zero) {
			t.Errorf("Expected negative CAGR for depreciation, got %s", cagr)
		}
	})

	t.Run("ZeroHoldingPeriod", func(t *testing.T) {
		cagr := service.TrackArtAppreciation(
			ctx,
			decimal.NewFromInt(1000000),
			decimal.NewFromInt(1000000),
			0,
		)
		if !cagr.Equal(decimal.Zero) {
			t.Errorf("Expected 0 CAGR for 0 years, got %s", cagr)
		}
	})
}
