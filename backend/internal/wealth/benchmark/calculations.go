package benchmark

import (
"context"
"math"
"time"
)

// Statistical calculations

// calculateTotalReturn calculates cumulative return from daily returns
func (e *BenchmarkEngine) calculateTotalReturn(returns []float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	cumulative := 1.0
	for _, r := range returns {
		cumulative *= (1 + r)
	}
	return cumulative - 1
}

// calculateVolatility calculates standard deviation of returns
func (e *BenchmarkEngine) calculateVolatility(returns []float64) float64 {
	if len(returns) < 2 {
		return 0
	}

	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))

	variance := 0.0
	for _, r := range returns {
		diff := r - mean
		variance += diff * diff
	}
	variance /= float64(len(returns) - 1)

	return math.Sqrt(variance)
}

// calculateTrackingError calculates tracking error between portfolio and benchmark
func (e *BenchmarkEngine) calculateTrackingError(portfolioReturns, benchmarkReturns []float64) float64 {
	n := len(portfolioReturns)
	if n != len(benchmarkReturns) {
		n = min(len(portfolioReturns), len(benchmarkReturns))
	}
	if n < 2 {
		return 0
	}

	activeReturns := make([]float64, n)
	for i := 0; i < n; i++ {
		activeReturns[i] = portfolioReturns[i] - benchmarkReturns[i]
	}

	return e.calculateVolatility(activeReturns)
}

// calculateBeta calculates portfolio beta relative to benchmark
func (e *BenchmarkEngine) calculateBeta(portfolioReturns, benchmarkReturns []float64) float64 {
	n := min(len(portfolioReturns), len(benchmarkReturns))
	if n < 2 {
		return 1.0
	}

	// Calculate means
	meanP := 0.0
	meanB := 0.0
	for i := 0; i < n; i++ {
		meanP += portfolioReturns[i]
		meanB += benchmarkReturns[i]
	}
	meanP /= float64(n)
	meanB /= float64(n)

	// Calculate covariance and variance
	covariance := 0.0
	varianceB := 0.0
	for i := 0; i < n; i++ {
		diffP := portfolioReturns[i] - meanP
		diffB := benchmarkReturns[i] - meanB
		covariance += diffP * diffB
		varianceB += diffB * diffB
	}

	if varianceB == 0 {
		return 1.0
	}

	return covariance / varianceB
}

// calculateAlpha calculates Jensen's alpha
func (e *BenchmarkEngine) calculateAlpha(portfolioReturns, benchmarkReturns []float64, riskFreeRate, beta float64) float64 {
n := min(len(portfolioReturns), len(benchmarkReturns))
if n == 0 {
return 0
}

// Average returns
avgPortfolio := 0.0
avgBenchmark := 0.0
for i := 0; i < n; i++ {
avgPortfolio += portfolioReturns[i]
avgBenchmark += benchmarkReturns[i]
}
avgPortfolio /= float64(n)
avgBenchmark /= float64(n)

// Alpha = Rp - [Rf + Beta * (Rb - Rf)]
return avgPortfolio - (riskFreeRate + beta*(avgBenchmark-riskFreeRate))
}

// calculateRSquared calculates R-squared between portfolio and benchmark
func (e *BenchmarkEngine) calculateRSquared(portfolioReturns, benchmarkReturns []float64) float64 {
n := min(len(portfolioReturns), len(benchmarkReturns))
if n < 2 {
return 0
}

// Calculate correlation
meanP := 0.0
meanB := 0.0
for i := 0; i < n; i++ {
meanP += portfolioReturns[i]
meanB += benchmarkReturns[i]
}
meanP /= float64(n)
meanB /= float64(n)

var sumPB, sumP2, sumB2 float64
for i := 0; i < n; i++ {
diffP := portfolioReturns[i] - meanP
diffB := benchmarkReturns[i] - meanB
sumPB += diffP * diffB
sumP2 += diffP * diffP
sumB2 += diffB * diffB
}

denominator := math.Sqrt(sumP2 * sumB2)
if denominator == 0 {
return 0
}

correlation := sumPB / denominator
return correlation * correlation
}

// calculateMaxDrawdown calculates maximum drawdown from returns
func (e *BenchmarkEngine) calculateMaxDrawdown(returns []float64) float64 {
if len(returns) == 0 {
return 0
}

peak := 1.0
maxDD := 0.0
cumulative := 1.0

for _, r := range returns {
cumulative *= (1 + r)
if cumulative > peak {
peak = cumulative
}
drawdown := (peak - cumulative) / peak
if drawdown > maxDD {
maxDD = drawdown
}
}

return maxDD
}

// calculatePeriodReturn calculates return for last n periods
func (e *BenchmarkEngine) calculatePeriodReturn(returns []float64, periods int) float64 {
if len(returns) < periods {
periods = len(returns)
}
if periods == 0 {
return 0
}

subset := returns[len(returns)-periods:]
return e.calculateTotalReturn(subset)
}

// calculateMTDReturn calculates month-to-date return
func (e *BenchmarkEngine) calculateMTDReturn(returns []float64, asOfDate time.Time) float64 {
// Calculate trading days in month so far
monthStart := time.Date(asOfDate.Year(), asOfDate.Month(), 1, 0, 0, 0, 0, time.UTC)
daysInMonth := int(asOfDate.Sub(monthStart).Hours() / 24)
tradingDays := daysInMonth * 5 / 7 // Approximate

return e.calculatePeriodReturn(returns, tradingDays)
}

// calculateQTDReturn calculates quarter-to-date return
func (e *BenchmarkEngine) calculateQTDReturn(returns []float64, asOfDate time.Time) float64 {
// Calculate trading days in quarter so far
quarter := (int(asOfDate.Month()) - 1) / 3
quarterStart := time.Date(asOfDate.Year(), time.Month(quarter*3+1), 1, 0, 0, 0, 0, time.UTC)
daysInQuarter := int(asOfDate.Sub(quarterStart).Hours() / 24)
tradingDays := daysInQuarter * 5 / 7

return e.calculatePeriodReturn(returns, tradingDays)
}

// calculateYTDReturn calculates year-to-date return
func (e *BenchmarkEngine) calculateYTDReturn(returns []float64, asOfDate time.Time) float64 {
// Calculate trading days in year so far
yearStart := time.Date(asOfDate.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
daysInYear := int(asOfDate.Sub(yearStart).Hours() / 24)
tradingDays := daysInYear * 5 / 7

return e.calculatePeriodReturn(returns, tradingDays)
}

// annualizeReturn converts cumulative return to annualized
func (e *BenchmarkEngine) annualizeReturn(totalReturn float64, years float64) float64 {
if years <= 0 {
return 0
}
return math.Pow(1+totalReturn, 1/years) - 1
}

// calculatePeriodComparisons generates comparison for standard periods
func (e *BenchmarkEngine) calculatePeriodComparisons(ctx context.Context, portfolioID, benchmarkID string, asOfDate time.Time) []PeriodComparison {
periods := []struct {
label string
start time.Time
}{
{"1D", asOfDate.AddDate(0, 0, -1)},
{"1W", asOfDate.AddDate(0, 0, -7)},
{"1M", asOfDate.AddDate(0, -1, 0)},
{"3M", asOfDate.AddDate(0, -3, 0)},
{"6M", asOfDate.AddDate(0, -6, 0)},
{"YTD", time.Date(asOfDate.Year(), 1, 1, 0, 0, 0, 0, time.UTC)},
{"1Y", asOfDate.AddDate(-1, 0, 0)},
{"3Y", asOfDate.AddDate(-3, 0, 0)},
{"5Y", asOfDate.AddDate(-5, 0, 0)},
}

comparisons := make([]PeriodComparison, 0, len(periods))

for _, p := range periods {
pReturns, _ := e.getPortfolioReturns(ctx, portfolioID, p.start, asOfDate)
bReturns, _ := e.getBenchmarkReturns(ctx, benchmarkID, p.start, asOfDate)

pReturn := e.calculateTotalReturn(pReturns)
bReturn := e.calculateTotalReturn(bReturns)
te := e.calculateTrackingError(pReturns, bReturns) * math.Sqrt(252)

comparisons = append(comparisons, PeriodComparison{
Period:          p.label,
PortfolioReturn: pReturn,
BenchmarkReturn: bReturn,
ActiveReturn:    pReturn - bReturn,
TrackingError:   te,
})
}

return comparisons
}

// calculateActiveWeights calculates over/underweight positions
func (e *BenchmarkEngine) calculateActiveWeights(ctx context.Context, portfolioID, benchmarkID string, asOfDate time.Time) []ActiveWeight {
// Get portfolio holdings
portfolioHoldings, _ := e.getPortfolioHoldings(ctx, portfolioID, asOfDate)

// Get benchmark holdings
benchmarkHoldings, _ := e.getBenchmarkHoldings(ctx, benchmarkID, asOfDate)

// Build benchmark weight map
benchmarkWeightMap := make(map[string]BenchmarkHolding)
for _, h := range benchmarkHoldings {
benchmarkWeightMap[h.SecurityID] = h
}

activeWeights := make([]ActiveWeight, 0)

// Calculate active weights for portfolio holdings
for _, ph := range portfolioHoldings {
bh := benchmarkWeightMap[ph.SecurityID]
activeWeight := ph.Weight - bh.Weight

activeWeights = append(activeWeights, ActiveWeight{
SecurityID:      ph.SecurityID,
SecurityName:    ph.SecurityName,
AssetClass:      ph.AssetClass,
Sector:          ph.Sector,
PortfolioWeight: ph.Weight,
BenchmarkWeight: bh.Weight,
ActiveWeight:    activeWeight,
})

delete(benchmarkWeightMap, ph.SecurityID)
}

// Add securities only in benchmark (underweight to zero)
for _, bh := range benchmarkWeightMap {
activeWeights = append(activeWeights, ActiveWeight{
SecurityID:      bh.SecurityID,
SecurityName:    bh.SecurityName,
AssetClass:      bh.AssetClass,
Sector:          bh.Sector,
PortfolioWeight: 0,
BenchmarkWeight: bh.Weight,
ActiveWeight:    -bh.Weight,
})
}

return activeWeights
}

// calculateSectorDeviations calculates sector-level deviations
func (e *BenchmarkEngine) calculateSectorDeviations(activeWeights []ActiveWeight) map[string]float64 {
deviations := make(map[string]float64)

for _, aw := range activeWeights {
sector := aw.Sector
if sector == "" {
sector = "Other"
}
deviations[sector] += aw.ActiveWeight
}

return deviations
}

// Helper function
func min(a, b int) int {
if a < b {
return a
}
return b
}
