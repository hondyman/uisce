package reporting

import (
	"time"

	"github.com/google/uuid"
)

type SectorAttribution struct {
	SectorKey         string  `json:"sectorKey"`
	SectorName        string  `json:"sectorName"`
	PortfolioWeight   float64 `json:"portfolioWeight"`   // w_i^P
	BenchmarkWeight   float64 `json:"benchmarkWeight"`   // w_i^B
	PortfolioReturn   float64 `json:"portfolioReturn"`   // R_i^P
	BenchmarkReturn   float64 `json:"benchmarkReturn"`   // R_i^B
	AllocationEffect  float64 `json:"allocationEffect"`  // A_i
	SelectionEffect   float64 `json:"selectionEffect"`   // S_i
	InteractionEffect float64 `json:"interactionEffect"` // I_i
	TotalContribution float64 `json:"totalContribution"` // A_i + S_i + I_i
}

type DrawdownPoint struct {
	AsOfDate          time.Time `json:"asOfDate"`
	PortfolioNAV      float64   `json:"portfolioNav"`
	BenchmarkNAV      float64   `json:"benchmarkNav"`
	PortfolioHWM      float64   `json:"portfolioHwm"`
	PortfolioDrawdown float64   `json:"portfolioDrawdown"` // DD_t
	BenchmarkDrawdown float64   `json:"benchmarkDrawdown"`
}

type PerformanceAttributionReport struct {
	TenantID             uuid.UUID           `json:"tenantId"`
	PortfolioID          string              `json:"portfolioId"`
	BenchmarkID          string              `json:"benchmarkId"`
	StartDate            time.Time           `json:"startDate"`
	EndDate              time.Time           `json:"endDate"`
	TotalPortfolioReturn float64             `json:"totalPortfolioReturn"`
	TotalBenchmarkReturn float64             `json:"totalBenchmarkReturn"`
	ExcessReturn         float64             `json:"excessReturn"`
	TotalAllocation      float64             `json:"totalAllocation"`
	TotalSelection       float64             `json:"totalSelection"`
	TotalInteraction     float64             `json:"totalInteraction"`
	MaxDrawdownPortfolio float64             `json:"maxDrawdownPortfolio"`
	MaxDrawdownBenchmark float64             `json:"maxDrawdownBenchmark"`
	Sectors              []SectorAttribution `json:"sectors"`
	DrawdownSeries       []DrawdownPoint     `json:"drawdownSeries"`
}
