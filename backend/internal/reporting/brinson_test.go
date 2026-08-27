package reporting_test

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/reporting"
)

func TestBrinsonAttributionEngine_DecompositionAndDrawdown(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	engine := reporting.NewBrinsonAttributionEngine()

	startDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)

	// 2 Sectors: Technology & Energy
	// Port: 60% Tech (+15%), 40% Energy (+5%) -> Total Return = 0.60*0.15 + 0.40*0.05 = 0.09 + 0.02 = 11.0%
	// Bench: 40% Tech (+10%), 60% Energy (+2%) -> Total Return = 0.40*0.10 + 0.60*0.02 = 0.04 + 0.012 = 5.2%
	// Excess Return = 11.0% - 5.2% = +5.8%
	sectors := []reporting.SectorAttribution{
		{
			SectorKey:       "tech",
			SectorName:      "Technology",
			PortfolioWeight: 0.60,
			BenchmarkWeight: 0.40,
			PortfolioReturn: 0.15,
			BenchmarkReturn: 0.10,
		},
		{
			SectorKey:       "energy",
			SectorName:      "Energy",
			PortfolioWeight: 0.40,
			BenchmarkWeight: 0.60,
			PortfolioReturn: 0.05,
			BenchmarkReturn: 0.02,
		},
	}

	// NAV Series with Drawdown (Peak 1,000,000 -> Trough 850,000 -> Recovery 950,000)
	navSeries := []reporting.DrawdownPoint{
		{AsOfDate: startDate, PortfolioNAV: 1000000.0, BenchmarkNAV: 1000000.0},
		{AsOfDate: startDate.AddDate(0, 1, 0), PortfolioNAV: 850000.0, BenchmarkNAV: 900000.0},
		{AsOfDate: endDate, PortfolioNAV: 950000.0, BenchmarkNAV: 920000.0},
	}

	report, err := engine.ComputeAttribution(
		ctx,
		tenantID,
		"port_global_01",
		"bench_msci_01",
		startDate,
		endDate,
		sectors,
		navSeries,
	)
	if err != nil {
		t.Fatalf("attribution computation failed: %v", err)
	}

	// 1. Verify Return Totals
	if math.Abs(report.TotalPortfolioReturn-0.11) > 1e-6 {
		t.Errorf("expected R^P = 0.11, got %f", report.TotalPortfolioReturn)
	}
	if math.Abs(report.TotalBenchmarkReturn-0.052) > 1e-6 {
		t.Errorf("expected R^B = 0.052, got %f", report.TotalBenchmarkReturn)
	}
	if math.Abs(report.ExcessReturn-0.058) > 1e-6 {
		t.Errorf("expected Excess = 0.058, got %f", report.ExcessReturn)
	}

	// 2. Verify Exact Mathematical Sum: ExcessReturn == Allocation + Selection + Interaction
	totalAttributionSum := report.TotalAllocation + report.TotalSelection + report.TotalInteraction
	if math.Abs(report.ExcessReturn-totalAttributionSum) > 1e-6 {
		t.Errorf("mathematical identity violation: Excess (%f) != Sum of effects (%f)", report.ExcessReturn, totalAttributionSum)
	}

	// 3. Verify Drawdown Calculation: Max DD = (850k - 1M)/1M = -15.0%
	expectedMaxDD := -0.15
	if math.Abs(report.MaxDrawdownPortfolio-expectedMaxDD) > 1e-6 {
		t.Errorf("expected Max Drawdown = %f, got %f", expectedMaxDD, report.MaxDrawdownPortfolio)
	}
}
