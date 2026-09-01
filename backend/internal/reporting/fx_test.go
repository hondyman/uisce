package reporting_test

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/reporting"
)

func TestFXTriangulationEngine_KarnoskySingerAttribution(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	engine := reporting.NewFXTriangulationEngine("USD")

	startDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)

	// Rates per 1 USD at t0 and t1
	startRates := map[reporting.CurrencyCode]float64{
		"USD": 1.0,
		"JPY": 150.0, // 1 USD = 150 JPY
		"EUR": 0.90,  // 1 USD = 0.90 EUR
	}

	endRates := map[reporting.CurrencyCode]float64{
		"USD": 1.0,
		"JPY": 140.0, // JPY strengthened vs USD
		"EUR": 0.95,  // EUR weakened vs USD
	}

	// 1 Position: Tokyo Equity holding (Native JPY, Reporting in EUR)
	// Start: 1,000 shares @ 15,000 JPY = 15,000,000 JPY
	// End:   1,000 shares @ 18,000 JPY = 18,000,000 JPY
	holdings := []reporting.PositionHolding{
		{
			PositionID:    "pos_tyo_01",
			SecurityName:  "Tokyo Robotics Corp",
			LocalCurrency: "JPY",
			Quantity:      1000,
			StartPriceLCY: 15000.0,
			EndPriceLCY:   18000.0,
			StartValLCY:   15000000.0,
			EndValLCY:     18000000.0,
		},
	}

	summary, err := engine.ComputePortfolioAttribution(
		ctx,
		tenantID,
		"USD",
		"EUR",
		startDate,
		endDate,
		holdings,
		startRates,
		endRates,
	)
	if err != nil {
		t.Fatalf("attribution calculation failed: %v", err)
	}

	if len(summary.Holdings) != 1 {
		t.Fatalf("expected 1 holding, got %d", len(summary.Holdings))
	}

	h := summary.Holdings[0]

	// 1. Verify Local Asset Return: (18M - 15M) / 15M = +20.0%
	expectedAssetReturn := 0.20
	if math.Abs(h.AssetReturnPct-expectedAssetReturn) > 1e-6 {
		t.Errorf("expected RL = %f, got %f", expectedAssetReturn, h.AssetReturnPct)
	}

	// 2. Verify Triangulated Rates:
	// Start FX(JPY -> EUR) = 0.90 / 150.0 = 0.0060 EUR/JPY
	// End FX(JPY -> EUR)   = 0.95 / 140.0 = 0.006785714 EUR/JPY
	// FX Return (RC) = (0.006785714 - 0.0060) / 0.0060 = +13.095238%
	expectedStartFX := 0.90 / 150.0
	expectedEndFX := 0.95 / 140.0
	expectedFXReturn := (expectedEndFX - expectedStartFX) / expectedStartFX

	if math.Abs(h.StartFXRate-expectedStartFX) > 1e-6 {
		t.Errorf("expected start FX = %f, got %f", expectedStartFX, h.StartFXRate)
	}
	if math.Abs(h.CurrencyReturnPct-expectedFXReturn) > 1e-6 {
		t.Errorf("expected RC = %f, got %f", expectedFXReturn, h.CurrencyReturnPct)
	}

	// 3. Verify Exact Mathematical Identity: RT = RL + RC + (RL * RC)
	expectedInteraction := expectedAssetReturn * expectedFXReturn
	expectedTotalReturn := expectedAssetReturn + expectedFXReturn + expectedInteraction

	if math.Abs(h.InteractionPct-expectedInteraction) > 1e-6 {
		t.Errorf("expected interaction = %f, got %f", expectedInteraction, h.InteractionPct)
	}
	if math.Abs(h.TotalReturnPct-expectedTotalReturn) > 1e-6 {
		t.Errorf("expected RT = %f, got %f", expectedTotalReturn, h.TotalReturnPct)
	}
}
