package reporting

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type BrinsonAttributionEngine struct{}

func NewBrinsonAttributionEngine() *BrinsonAttributionEngine {
	return &BrinsonAttributionEngine{}
}

// ComputeAttribution evaluates multi-sector Brinson-Fachler decomposition and drawdown series
func (e *BrinsonAttributionEngine) ComputeAttribution(
	ctx context.Context,
	tenantID uuid.UUID,
	portfolioID string,
	benchmarkID string,
	startDate time.Time,
	endDate time.Time,
	sectors []SectorAttribution,
	navSeries []DrawdownPoint,
) (*PerformanceAttributionReport, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	report := &PerformanceAttributionReport{
		TenantID:       tenantID,
		PortfolioID:    portfolioID,
		BenchmarkID:    benchmarkID,
		StartDate:      startDate,
		EndDate:        endDate,
		Sectors:        make([]SectorAttribution, len(sectors)),
		DrawdownSeries: make([]DrawdownPoint, len(navSeries)),
	}

	// 1. Calculate Base Total Benchmark Return (R^B) and Total Portfolio Return (R^P)
	var totalPortReturn, totalBenchReturn float64
	for _, s := range sectors {
		totalPortReturn += s.PortfolioWeight * s.PortfolioReturn
		totalBenchReturn += s.BenchmarkWeight * s.BenchmarkReturn
	}

	report.TotalPortfolioReturn = totalPortReturn
	report.TotalBenchmarkReturn = totalBenchReturn
	report.ExcessReturn = totalPortReturn - totalBenchReturn

	// 2. Compute Brinson-Fachler Attribution Components per Sector
	var totalAlloc, totalSelect, totalInteract float64
	for i, s := range sectors {
		alloc := (s.PortfolioWeight - s.BenchmarkWeight) * (s.BenchmarkReturn - totalBenchReturn)
		selectEff := s.BenchmarkWeight * (s.PortfolioReturn - s.BenchmarkReturn)
		interact := (s.PortfolioWeight - s.BenchmarkWeight) * (s.PortfolioReturn - s.BenchmarkReturn)
		totalContrib := alloc + selectEff + interact

		report.Sectors[i] = SectorAttribution{
			SectorKey:         s.SectorKey,
			SectorName:        s.SectorName,
			PortfolioWeight:   s.PortfolioWeight,
			BenchmarkWeight:   s.BenchmarkWeight,
			PortfolioReturn:   s.PortfolioReturn,
			BenchmarkReturn:   s.BenchmarkReturn,
			AllocationEffect:  alloc,
			SelectionEffect:   selectEff,
			InteractionEffect: interact,
			TotalContribution: totalContrib,
		}

		totalAlloc += alloc
		totalSelect += selectEff
		totalInteract += interact
	}

	report.TotalAllocation = totalAlloc
	report.TotalSelection = totalSelect
	report.TotalInteraction = totalInteract

	// 3. Compute High-Water Mark (HWM) and Drawdown Time-Series
	var portHWM, benchHWM float64
	maxPortDD := 0.0
	maxBenchDD := 0.0

	for i, pt := range navSeries {
		if i == 0 || pt.PortfolioNAV > portHWM {
			portHWM = pt.PortfolioNAV
		}
		if i == 0 || pt.BenchmarkNAV > benchHWM {
			benchHWM = pt.BenchmarkNAV
		}

		portDD := 0.0
		if portHWM > 0 {
			portDD = (pt.PortfolioNAV - portHWM) / portHWM
		}

		benchDD := 0.0
		if benchHWM > 0 {
			benchDD = (pt.BenchmarkNAV - benchHWM) / benchHWM
		}

		if portDD < maxPortDD {
			maxPortDD = portDD
		}
		if benchDD < maxBenchDD {
			maxBenchDD = benchDD
		}

		report.DrawdownSeries[i] = DrawdownPoint{
			AsOfDate:          pt.AsOfDate,
			PortfolioNAV:      pt.PortfolioNAV,
			BenchmarkNAV:      pt.BenchmarkNAV,
			PortfolioHWM:      portHWM,
			PortfolioDrawdown: portDD,
			BenchmarkDrawdown: benchDD,
		}
	}

	report.MaxDrawdownPortfolio = maxPortDD
	report.MaxDrawdownBenchmark = maxBenchDD

	return report, nil
}
