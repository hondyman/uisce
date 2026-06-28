package main

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"go.temporal.io/sdk/activity"
)

// ============================================================================
// YALE MODEL DATA STRUCTURES
// ============================================================================

type YaleModelParams struct {
	CallRate         float64 // RC: quarterly call rate (0.20 = 20%)
	GrowthRate       float64 // G: quarterly NAV growth (0.03 = 3%)
	YieldRate        float64 // Y: minimum quarterly distribution rate
	BowFactor        float64 // B: distribution timing curve (0.8-2.5)
	TerminationYears int     // L: fund lifetime
	TargetIRR        float64 // Target IRR for calibration
	TargetTVPI       float64 // Target TVPI for calibration
}

type FundPosition struct {
	PICC               float64 // Paid-in capital cumulative
	DCC                float64 // Distributed capital cumulative
	NAV                float64 // Net asset value
	RecallableCapital  float64
	UnfundedCommitment float64
}

type QuarterlyProjection struct {
	Quarter          int       // Quarter number from start
	QuarterDate      time.Time // Date of quarter end
	ProjectedCalls   float64   // Capital calls for this quarter
	ProjectedDist    float64   // Distributions for this quarter
	ProjectedNAV     float64   // NAV at end of quarter
	ProjectedPICC    float64   // Cumulative PICC
	ProjectedDCC     float64   // Cumulative DCC
	ProjectedTVPI    float64   // TVPI at end of quarter
	ProjectedDPI     float64   // DPI at end of quarter
	ProjectedIRR     float64   // IRR to this point
	DistributionRate float64   // Distribution rate used this quarter
}

type ForecastResult struct {
	CommitmentID     string
	Scenario         string
	Projections      []QuarterlyProjection
	P5Percentile     float64 // 5th percentile net cashflow
	P25Percentile    float64 // 25th percentile
	P75Percentile    float64 // 75th percentile
	P95Percentile    float64 // 95th percentile (MPC - Maximum Probable Call)
	ConfidenceScore  float64 // 0.0-1.0
	CalibratedParams *YaleModelParams
	Warnings         []string
}

// ============================================================================
// 1. YALE MODEL: PARAMETER CALIBRATION
// ============================================================================

// CalibrateYaleModel calibrates the Yale model growth rate to match target IRR
// Uses Newton-Raphson iteration to find the correct growth rate
func (a *RebalanceActivities) CalibrateYaleModel(
	ctx context.Context,
	commitment float64,
	picc float64,
	dcc float64,
	nav float64,
	params YaleModelParams,
) (*YaleModelParams, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Calibrating Yale model",
		"commitment", commitment,
		"targetIRR", params.TargetIRR,
		"targetTVPI", params.TargetTVPI)

	// For mature funds, don't recalibrate if performance is tracking well
	if picc > 0 {
		currentTVPI := (nav + dcc) / picc
		if math.Abs(currentTVPI-params.TargetTVPI) < params.TargetTVPI*0.1 {
			logger.Info("Fund tracking to target, using current params")
			return &params, nil
		}
	}

	// Iterative calibration: Newton-Raphson method
	growthRate := params.GrowthRate
	maxIterations := 50
	tolerance := 0.0001

	for i := 0; i < maxIterations; i++ {
		params.GrowthRate = growthRate

		// Project forward with current growth rate
		projections := a.projectQuarterly(commitment, picc, dcc, nav, params)
		if len(projections) == 0 {
			return nil, fmt.Errorf("projection failed")
		}

		projectedIRR := calculateIRR(projections)

		// Calculate residual (difference from target)
		targetToUse := params.TargetIRR
		if params.TargetTVPI > 0 {
			targetToUse = (params.TargetTVPI - 1.0) * 100 // Convert TVPI to IRR-like %
		}

		residual := (projectedIRR - targetToUse) / 100.0

		if math.Abs(residual) < tolerance {
			logger.Info("Calibration converged",
				"iterations", i,
				"finalGrowthRate", growthRate,
				"achievedIRR", projectedIRR,
				"targetIRR", targetToUse)
			params.GrowthRate = growthRate
			return &params, nil
		}

		// Newton-Raphson: adjust growth rate
		// Derivative: higher growth rate → higher IRR
		// Use finite difference approximation
		delta := 0.0001
		params.GrowthRate = growthRate + delta
		projections2 := a.projectQuarterly(commitment, picc, dcc, nav, params)
		irr2 := calculateIRR(projections2)

		derivative := (irr2 - projectedIRR) / (delta * 100)
		if math.Abs(derivative) < 0.001 {
			derivative = 0.001 // Avoid division by zero
		}

		// Update: new_x = old_x - f(x) / f'(x)
		growthRate = growthRate - (residual / derivative)

		// Bounds check
		if growthRate < -0.05 {
			growthRate = -0.05
		}
		if growthRate > 0.15 {
			growthRate = 0.15
		}
	}

	logger.Warn("Calibration did not converge after max iterations", "maxIterations", maxIterations)
	params.GrowthRate = growthRate
	return &params, nil
}

// ============================================================================
// 2. YALE MODEL: QUARTERLY PROJECTION LOOP
// ============================================================================

func (a *RebalanceActivities) projectQuarterly(
	commitment float64,
	initialPICC float64,
	initialDCC float64,
	initialNAV float64,
	params YaleModelParams,
) []QuarterlyProjection {
	var projections []QuarterlyProjection

	totalLifeQuarters := params.TerminationYears * 4
	currentAge := 0 // Assume starting from beginning; adjust if needed
	if initialPICC > 0 {
		currentAge = int(initialPICC / commitment * float64(totalLifeQuarters))
	}

	picc := initialPICC
	dcc := initialDCC
	nav := initialNAV
	currentDate := time.Now()

	for quarter := 1; quarter <= totalLifeQuarters; quarter++ {
		quarterDate := currentDate.AddDate(0, quarter*3, 0)
		currentAge++
		fractionOfLife := float64(currentAge) / float64(totalLifeQuarters)

		// Step 1: Capital Calls
		unfunded := commitment - picc
		calls := 0.0
		if unfunded > 0 {
			calls = params.CallRate * unfunded
			// Cap calls to not exceed remaining commitment
			if picc+calls > commitment {
				calls = commitment - picc
			}
		}

		// Step 2: Distribution Rate (bow factor curve)
		distRateExYield := math.Pow(fractionOfLife, params.BowFactor)
		distributionRate := math.Max(params.YieldRate, distRateExYield)

		// Step 3: Distributions
		openNAV := nav * (1 + params.GrowthRate)
		distributions := openNAV * distributionRate

		// Step 4: NAV Update
		nav = openNAV + calls - distributions

		// Bounds check
		if nav < 0 {
			nav = 0
		}

		// Step 5: Update cumulative metrics
		picc += calls
		dcc += distributions

		// Calculate metrics
		tvpi := 0.0
		dpi := 0.0
		if picc > 0 {
			tvpi = (nav + dcc) / picc
			dpi = dcc / picc
		}

		proj := QuarterlyProjection{
			Quarter:          quarter,
			QuarterDate:      quarterDate,
			ProjectedCalls:   calls,
			ProjectedDist:    distributions,
			ProjectedNAV:     nav,
			ProjectedPICC:    picc,
			ProjectedDCC:     dcc,
			ProjectedTVPI:    tvpi,
			ProjectedDPI:     dpi,
			DistributionRate: distributionRate,
		}

		projections = append(projections, proj)

		// Stop if fully invested
		if picc >= commitment && nav <= 0 {
			break
		}
	}

	return projections
}

// ============================================================================
// 3. BENCHMARK-DRIVEN FORECASTING
// ============================================================================

type BenchmarkTemplate struct {
	StrategyType    string
	AgeQuarters     int
	CallRatePattern float64 // Cumulative % of commitment called
	DistRatePattern float64 // Cumulative % of commitment distributed
	PICC_Pct        float64 // Average PICC as % of commitment
	DPI             float64 // Average DPI
}

// ApplyBenchmarkRefinement adjusts forecast based on fund pacing vs. benchmarks
func (a *RebalanceActivities) ApplyBenchmarkRefinement(
	ctx context.Context,
	commitment float64,
	picc float64,
	projections []QuarterlyProjection,
	benchmarks []BenchmarkTemplate,
) []QuarterlyProjection {
	logger := activity.GetLogger(ctx)

	// Calculate pace factor: how far ahead/behind is fund vs. benchmark?
	benchmarkPICC := commitment * (benchmarks[0].PICC_Pct / 100.0)
	paceFactor := 1.0
	if benchmarkPICC > 0 {
		paceFactor = picc / benchmarkPICC
	}

	logger.Info("Applying benchmark refinement",
		"paceFactor", paceFactor,
		"benchmarkPICC", benchmarkPICC,
		"actualPICC", picc)

	// Adjust projected calls based on pace
	refined := make([]QuarterlyProjection, len(projections))
	for i, proj := range projections {
		refined[i] = proj

		// If ahead of schedule (paceFactor > 1), slow down calls
		// If behind schedule (paceFactor < 1), speed up calls
		adjustedCalls := proj.ProjectedCalls / paceFactor
		refined[i].ProjectedCalls = adjustedCalls
	}

	return refined
}

// ============================================================================
// 4. MONTE CARLO STOCHASTIC FORECASTING
// ============================================================================

type PerformanceDistribution struct {
	Downside    float64 // 20% probability, lower IRR
	Base        float64 // 50% probability, target IRR
	Upside      float64 // 25% probability, higher IRR
	Exceptional float64 // 5% probability, exceptional IRR
}

// RunMonteCarloSimulation generates probabilistic cash flow forecasts
func (a *RebalanceActivities) RunMonteCarloSimulation(
	ctx context.Context,
	commitment float64,
	picc float64,
	dcc float64,
	nav float64,
	params YaleModelParams,
	distributions []PerformanceDistribution,
	numSimulations int,
) ForecastResult {
	logger := activity.GetLogger(ctx)
	logger.Info("Running Monte Carlo simulation",
		"numSimulations", numSimulations,
		"commitment", commitment)

	var allSimulationResults [][]QuarterlyProjection

	// Run simulations
	for sim := 0; sim < numSimulations; sim++ {
		// Randomly select outcome based on probabilities
		randVal := a.randomFloat(0, 1)
		var targetIRR float64

		if randVal < 0.20 {
			targetIRR = distributions[0].Downside
		} else if randVal < 0.70 {
			targetIRR = distributions[0].Base
		} else if randVal < 0.95 {
			targetIRR = distributions[0].Upside
		} else {
			targetIRR = distributions[0].Exceptional
		}

		// Run Yale model with this IRR
		params.TargetIRR = targetIRR
		simParams, _ := a.CalibrateYaleModel(ctx, commitment, picc, dcc, nav, params)

		simProj := a.projectQuarterly(commitment, picc, dcc, nav, *simParams)
		allSimulationResults = append(allSimulationResults, simProj)
	}

	// Extract base case (50th percentile) as primary forecast
	baseProj := a.projectQuarterly(commitment, picc, dcc, nav, params)

	// Calculate percentiles for net cashflow
	netFlows := make([]float64, len(allSimulationResults))
	for i, simProj := range allSimulationResults {
		netFlow := 0.0
		for _, q := range simProj {
			netFlow += q.ProjectedCalls - q.ProjectedDist
		}
		netFlows[i] = netFlow
	}

	sort.Float64s(netFlows)
	p5 := netFlows[int(float64(numSimulations)*0.05)]
	p25 := netFlows[int(float64(numSimulations)*0.25)]
	p75 := netFlows[int(float64(numSimulations)*0.75)]
	p95 := netFlows[int(float64(numSimulations)*0.95)]

	result := ForecastResult{
		Projections:      baseProj,
		P5Percentile:     p5,
		P25Percentile:    p25,
		P75Percentile:    p75,
		P95Percentile:    p95,
		ConfidenceScore:  0.95,
		CalibratedParams: &params,
	}

	logger.Info("Monte Carlo complete",
		"p5", p5,
		"p50", 0,
		"p95", p95,
		"mpc", p95)

	return result
}

// ============================================================================
// 5. J-CURVE MODELING (Deal-Level Bottom-Up)
// ============================================================================

type DealProjection struct {
	InvestmentAmount float64
	HoldingYears     int
	ExitMultiple     float64
	NAVCurve         []float64 // NAV multiplier by year (0.95, 0.90, 0.95, 1.10, 1.40, ...)
	YearlyNAV        []float64
	ExitValue        float64
}

// ProjectDealJCurve generates J-curve valuation trajectory for a single investment
func (a *RebalanceActivities) ProjectDealJCurve(
	investmentAmount float64,
	holdingYears int,
	exitMultiple float64,
) DealProjection {
	// Standard J-curve: initial decline due to fees, then growth, exit growth
	navCurve := []float64{0.95, 0.90, 0.95, 1.10, 1.40, 1.80, 2.20, 2.50}

	yearlyNAV := make([]float64, holdingYears+1)
	yearlyNAV[0] = investmentAmount

	for year := 1; year <= holdingYears && year < len(navCurve); year++ {
		yearlyNAV[year] = investmentAmount * navCurve[year-1]
	}

	exitValue := investmentAmount * exitMultiple

	return DealProjection{
		InvestmentAmount: investmentAmount,
		HoldingYears:     holdingYears,
		ExitMultiple:     exitMultiple,
		NAVCurve:         navCurve,
		YearlyNAV:        yearlyNAV,
		ExitValue:        exitValue,
	}
}

// ============================================================================
// 6. FORECAST GENERATION ACTIVITY (Temporal)
// ============================================================================

// GenerateCashFlowForecast is the main Temporal activity to generate forecasts
func (a *RebalanceActivities) GenerateCashFlowForecast(
	ctx context.Context,
	commitmentID string,
	commitment float64,
	picc float64,
	dcc float64,
	nav float64,
	params YaleModelParams,
) (map[string]interface{}, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Generating cash flow forecast",
		"commitmentID", commitmentID,
		"commitment", commitment,
		"targetIRR", params.TargetIRR)

	// Step 1: Calibrate parameters
	calibrated, err := a.CalibrateYaleModel(ctx, commitment, picc, dcc, nav, params)
	if err != nil {
		logger.Error("Calibration failed", "error", err)
		return nil, err
	}

	// Step 2: Generate base case projection
	baseProjections := a.projectQuarterly(commitment, picc, dcc, nav, *calibrated)

	// Step 3: Generate stochastic scenarios (Monte Carlo)
	distrib := []PerformanceDistribution{
		{
			Downside:    params.TargetIRR - 8.0,
			Base:        params.TargetIRR,
			Upside:      params.TargetIRR + 7.0,
			Exceptional: params.TargetIRR + 15.0,
		},
	}
	stochasticResult := a.RunMonteCarloSimulation(
		ctx, commitment, picc, dcc, nav, *calibrated, distrib, 1000)

	// Step 4: Package results
	result := map[string]interface{}{
		"commitmentID":     commitmentID,
		"baseProjections":  baseProjections,
		"stochasticResult": stochasticResult,
		"calibratedParams": calibrated,
		"p95Percentile":    stochasticResult.P95Percentile,
		"mpc":              stochasticResult.P95Percentile, // Maximum Probable Call
	}

	logger.Info("Forecast generated successfully",
		"projectionQuarters", len(baseProjections),
		"mpc", stochasticResult.P95Percentile)

	return result, nil
}

// ============================================================================
// 7. HELPER FUNCTIONS
// ============================================================================

func calculateIRR(projections []QuarterlyProjection) float64 {
	// Simplified IRR calculation using final TVPI as proxy
	// In production, use proper cash flow IRR calculation
	if len(projections) == 0 {
		return 0
	}
	final := projections[len(projections)-1]
	quarters := len(projections)
	years := float64(quarters) / 4.0

	if final.ProjectedPICC <= 0 {
		return 0
	}

	tvpi := (final.ProjectedNAV + final.ProjectedDCC) / final.ProjectedPICC

	// TVPI to annual IRR (simplified): (TVPI)^(1/years) - 1
	if years > 0 && tvpi > 0 {
		return (math.Pow(tvpi, 1/years) - 1) * 100
	}
	return 0
}

func (a *RebalanceActivities) randomFloat(min, max float64) float64 {
	// Simple pseudo-random; use rand/math.Rand for production
	return min + (max-min)*0.5
}

// ReconcileCapitalActivity: Three-way matching of capital events
func (a *RebalanceActivities) ReconcileCapitalActivity(
	ctx context.Context,
	commitmentID string,
	eventDate time.Time,
	fundAmount float64,
	bankAmount float64,
	internalAmount float64,
	tolerance float64,
) (map[string]interface{}, error) {
	logger := activity.GetLogger(ctx)

	logger.Info("Reconciling capital activity",
		"commitmentID", commitmentID,
		"fundAmount", fundAmount,
		"bankAmount", bankAmount,
		"internalAmount", internalAmount)

	// Three-way match
	status := "exception"
	variance := math.Abs(fundAmount - bankAmount)

	if variance <= tolerance {
		if math.Abs(bankAmount-internalAmount) <= tolerance {
			status = "reconciled"
		} else {
			status = "partial_match"
		}
	}

	result := map[string]interface{}{
		"commitmentID": commitmentID,
		"status":       status,
		"variance":     variance,
		"variancePct":  (variance / fundAmount) * 100,
		"matchType":    "fund_bank_internal",
	}

	logger.Info("Reconciliation complete", "status", status)
	return result, nil
}
