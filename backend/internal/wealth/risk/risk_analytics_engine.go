package risk

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

// RiskAnalyticsEngine provides institutional-grade risk analytics
// including VaR, CVaR, stress testing, and factor risk decomposition
type RiskAnalyticsEngine struct {
	db *sql.DB
}

// NewRiskAnalyticsEngine creates a new risk analytics engine
func NewRiskAnalyticsEngine(db *sql.DB) *RiskAnalyticsEngine {
	return &RiskAnalyticsEngine{db: db}
}

// RiskConfig configures the risk calculation
type RiskConfig struct {
	PortfolioID      string
	AsOfDate         time.Time
	ConfidenceLevels []float64 // e.g., [0.95, 0.99]
	Horizon          int       // days
	Method           VaRMethod
	SimulationCount  int // for Monte Carlo
	HistoricalPeriod int // lookback days
	Currency         string
}

// VaRMethod defines the VaR calculation methodology
type VaRMethod string

const (
	HistoricalVaR    VaRMethod = "historical"
	ParametricVaR    VaRMethod = "parametric"
	MonteCarloVaR    VaRMethod = "monte_carlo"
	CornishFisherVaR VaRMethod = "cornish_fisher"
)

// RiskResult contains comprehensive risk analysis
type RiskResult struct {
	Config             RiskConfig
	PortfolioValue     float64
	VaRResults         map[float64]*VaRResult  // by confidence level
	CVaRResults        map[float64]*CVaRResult // by confidence level
	StressTestResults  []StressTestResult
	FactorRiskAnalysis *FactorRiskResult
	ConcentrationRisk  *ConcentrationRiskResult
	LiquidityRisk      *LiquidityRiskResult
	TailRiskMetrics    *TailRiskMetrics
	RiskContributions  []PositionRiskContribution
	CorrelationMatrix  *CorrelationMatrix
	CalculatedAt       time.Time
}

// VaRResult contains Value at Risk calculation result
type VaRResult struct {
	Confidence      float64
	Method          VaRMethod
	VaRAbsolute     float64 // dollar amount at risk
	VaRRelative     float64 // percentage of portfolio
	Horizon         int     // days
	BreachCount     int     // historical breaches
	BreachFrequency float64 // breach rate
}

// CVaRResult contains Conditional VaR (Expected Shortfall)
type CVaRResult struct {
	Confidence      float64
	CVaRAbsolute    float64 // expected loss given VaR breach
	CVaRRelative    float64 // percentage
	AverageTailLoss float64 // average loss in tail
}

// StressTestResult contains stress scenario analysis
type StressTestResult struct {
	ScenarioName    string
	ScenarioType    string // "historical", "hypothetical", "reverse"
	Description     string
	ShockParameters map[string]float64
	PortfolioImpact float64 // absolute P&L
	ImpactPercent   float64
	PositionImpacts []PositionImpact
	RecoveryDays    int // estimated recovery time
}

// PositionImpact represents impact on a single position
type PositionImpact struct {
	SecurityID    string
	SecurityName  string
	CurrentValue  float64
	StressedValue float64
	Impact        float64
	ImpactPercent float64
}

// FactorRiskResult contains factor-based risk decomposition
type FactorRiskResult struct {
	TotalVolatility     float64
	SystematicRisk      float64
	IdiosyncraticRisk   float64
	FactorContributions map[string]*FactorContribution
	MarginalVaRByFactor map[string]float64
}

// FactorContribution represents risk from a single factor
type FactorContribution struct {
	FactorName        string
	Exposure          float64 // beta
	FactorVolatility  float64
	ContributionToVar float64
	ContributionPct   float64
}

// ConcentrationRiskResult contains concentration risk metrics
type ConcentrationRiskResult struct {
	HerfindahlIndex       float64 // HHI
	EffectivePositions    float64 // 1/HHI
	TopHoldingsPct        float64 // top 10
	SectorConcentration   map[string]float64
	CountryConcentration  map[string]float64
	CurrencyConcentration map[string]float64
	SingleNameRisk        []SingleNameRisk
}

// SingleNameRisk represents risk from single issuer
type SingleNameRisk struct {
	SecurityID   string
	SecurityName string
	Weight       float64
	VaRContrib   float64
}

// LiquidityRiskResult contains liquidity risk metrics
type LiquidityRiskResult struct {
	DaysToLiquidate      float64 // weighted average
	LiquidityBuckets     map[string]float64
	LiquidityCostBps     float64
	LiquidityAdjustedVaR float64
	IlliquidHoldings     []IlliquidHolding
}

// IlliquidHolding represents a potentially illiquid position
type IlliquidHolding struct {
	SecurityID      string
	SecurityName    string
	Weight          float64
	AvgDailyVolume  float64
	DaysToLiquidate float64
	LiquidityCost   float64
}

// TailRiskMetrics contains tail risk analysis
type TailRiskMetrics struct {
	Skewness        float64
	ExcessKurtosis  float64
	TailIndex       float64 // for EVT
	MaxDrawdown     float64
	MaxDrawdownDate time.Time
	RecoveryTime    int // days
	WorstDay        float64
	WorstWeek       float64
	WorstMonth      float64
}

// PositionRiskContribution represents risk from individual position
type PositionRiskContribution struct {
	SecurityID      string
	SecurityName    string
	Weight          float64
	Volatility      float64
	BetaToPortfolio float64
	MarginalVaR     float64 // change in VaR from adding position
	ComponentVaR    float64 // contribution to total VaR
	ContributionPct float64
	IncrementalVaR  float64
}

// CorrelationMatrix contains asset correlation analysis
type CorrelationMatrix struct {
	Assets       []string
	Correlations [][]float64
	Eigenvalues  []float64
	PCARiskPct   []float64 // % of risk explained by each PC
}

// PortfolioPosition represents a position for risk calculation
type PortfolioPosition struct {
	SecurityID     string
	SecurityName   string
	AssetClass     string
	Sector         string
	Country        string
	Currency       string
	MarketValue    float64
	Weight         float64
	Volatility     float64
	Returns        []float64 // historical returns
	AvgDailyVolume float64
}

// Calculate performs comprehensive risk analysis
func (e *RiskAnalyticsEngine) Calculate(ctx context.Context, config RiskConfig) (*RiskResult, error) {
	// Fetch portfolio positions with historical returns
	positions, err := e.getPortfolioPositions(ctx, config.PortfolioID, config.AsOfDate, config.HistoricalPeriod)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch positions: %w", err)
	}

	if len(positions) == 0 {
		return nil, fmt.Errorf("no positions found for portfolio")
	}

	// Calculate portfolio value
	portfolioValue := 0.0
	for _, p := range positions {
		portfolioValue += p.MarketValue
	}

	// Calculate portfolio returns
	portfolioReturns := e.calculatePortfolioReturns(positions)

	result := &RiskResult{
		Config:         config,
		PortfolioValue: portfolioValue,
		VaRResults:     make(map[float64]*VaRResult),
		CVaRResults:    make(map[float64]*CVaRResult),
		CalculatedAt:   time.Now(),
	}

	// Calculate VaR and CVaR for each confidence level
	for _, confidence := range config.ConfidenceLevels {
		var varResult *VaRResult

		switch config.Method {
		case HistoricalVaR:
			varResult = e.calculateHistoricalVaR(portfolioReturns, portfolioValue, confidence, config.Horizon)
		case ParametricVaR:
			varResult = e.calculateParametricVaR(portfolioReturns, portfolioValue, confidence, config.Horizon)
		case MonteCarloVaR:
			varResult = e.calculateMonteCarloVaR(positions, portfolioValue, confidence, config.Horizon, config.SimulationCount)
		case CornishFisherVaR:
			varResult = e.calculateCornishFisherVaR(portfolioReturns, portfolioValue, confidence, config.Horizon)
		default:
			varResult = e.calculateHistoricalVaR(portfolioReturns, portfolioValue, confidence, config.Horizon)
		}

		result.VaRResults[confidence] = varResult

		// Calculate CVaR (Expected Shortfall)
		cvarResult := e.calculateCVaR(portfolioReturns, portfolioValue, confidence)
		result.CVaRResults[confidence] = cvarResult
	}

	// Perform stress tests
	result.StressTestResults = e.runStressTests(ctx, positions, portfolioValue)

	// Factor risk decomposition
	result.FactorRiskAnalysis = e.calculateFactorRisk(ctx, positions)

	// Concentration risk
	result.ConcentrationRisk = e.calculateConcentrationRisk(positions)

	// Liquidity risk
	result.LiquidityRisk = e.calculateLiquidityRisk(positions, portfolioValue)

	// Tail risk metrics
	result.TailRiskMetrics = e.calculateTailRisk(portfolioReturns)

	// Position risk contributions
	result.RiskContributions = e.calculateRiskContributions(positions, portfolioReturns, result.VaRResults[0.95])

	// Correlation matrix
	result.CorrelationMatrix = e.calculateCorrelationMatrix(positions)

	return result, nil
}

// calculateHistoricalVaR uses historical simulation
func (e *RiskAnalyticsEngine) calculateHistoricalVaR(returns []float64, portfolioValue float64, confidence float64, horizon int) *VaRResult {
	if len(returns) == 0 {
		return &VaRResult{Confidence: confidence, Method: HistoricalVaR}
	}

	// Sort returns ascending
	sortedReturns := make([]float64, len(returns))
	copy(sortedReturns, returns)
	sort.Float64s(sortedReturns)

	// Find VaR percentile
	percentile := 1 - confidence
	index := int(float64(len(sortedReturns)) * percentile)
	if index >= len(sortedReturns) {
		index = len(sortedReturns) - 1
	}

	varReturn := sortedReturns[index]

	// Scale to horizon (square root of time)
	varReturn = varReturn * math.Sqrt(float64(horizon))

	varAbsolute := math.Abs(varReturn * portfolioValue)

	// Count historical breaches
	breachCount := 0
	for _, r := range returns {
		if r < varReturn {
			breachCount++
		}
	}

	return &VaRResult{
		Confidence:      confidence,
		Method:          HistoricalVaR,
		VaRAbsolute:     varAbsolute,
		VaRRelative:     math.Abs(varReturn) * 100,
		Horizon:         horizon,
		BreachCount:     breachCount,
		BreachFrequency: float64(breachCount) / float64(len(returns)),
	}
}

// calculateParametricVaR uses normal distribution assumption
func (e *RiskAnalyticsEngine) calculateParametricVaR(returns []float64, portfolioValue float64, confidence float64, horizon int) *VaRResult {
	if len(returns) < 2 {
		return &VaRResult{Confidence: confidence, Method: ParametricVaR}
	}

	mean := calculateMean(returns)
	stdDev := calculateStdDev(returns)

	// Z-score for confidence level
	zScore := normalInverse(1 - confidence)

	// VaR = -mean + z * stdDev (scaled to horizon)
	varReturn := -mean + zScore*stdDev*math.Sqrt(float64(horizon))
	varAbsolute := math.Abs(varReturn * portfolioValue)

	return &VaRResult{
		Confidence:  confidence,
		Method:      ParametricVaR,
		VaRAbsolute: varAbsolute,
		VaRRelative: math.Abs(varReturn) * 100,
		Horizon:     horizon,
	}
}

// calculateMonteCarloVaR uses Monte Carlo simulation
func (e *RiskAnalyticsEngine) calculateMonteCarloVaR(positions []PortfolioPosition, portfolioValue float64, confidence float64, horizon int, simulations int) *VaRResult {
	if simulations <= 0 {
		simulations = 10000
	}

	// Calculate covariance matrix
	returns := make([][]float64, len(positions))
	for i, p := range positions {
		returns[i] = p.Returns
	}

	// Get portfolio weights
	weights := make([]float64, len(positions))
	for i, p := range positions {
		weights[i] = p.Weight
	}

	// Run simulations
	simulatedReturns := make([]float64, simulations)

	// Simplified: use position-level simulation
	var wg sync.WaitGroup
	chunkSize := simulations / 4

	for chunk := 0; chunk < 4; chunk++ {
		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			for i := start; i < end; i++ {
				portfolioReturn := 0.0
				for j, p := range positions {
					// Random return based on historical distribution
					if len(p.Returns) > 0 {
						randIdx := i % len(p.Returns)
						portfolioReturn += weights[j] * p.Returns[randIdx]
					}
				}
				simulatedReturns[i] = portfolioReturn
			}
		}(chunk*chunkSize, min((chunk+1)*chunkSize, simulations))
	}
	wg.Wait()

	// Sort and find percentile
	sort.Float64s(simulatedReturns)
	percentile := 1 - confidence
	index := int(float64(simulations) * percentile)

	varReturn := simulatedReturns[index] * math.Sqrt(float64(horizon))
	varAbsolute := math.Abs(varReturn * portfolioValue)

	return &VaRResult{
		Confidence:  confidence,
		Method:      MonteCarloVaR,
		VaRAbsolute: varAbsolute,
		VaRRelative: math.Abs(varReturn) * 100,
		Horizon:     horizon,
	}
}

// calculateCornishFisherVaR adjusts for skewness and kurtosis
func (e *RiskAnalyticsEngine) calculateCornishFisherVaR(returns []float64, portfolioValue float64, confidence float64, horizon int) *VaRResult {
	if len(returns) < 4 {
		return e.calculateParametricVaR(returns, portfolioValue, confidence, horizon)
	}

	mean := calculateMean(returns)
	stdDev := calculateStdDev(returns)
	skew := calculateSkewness(returns)
	kurt := calculateKurtosis(returns) - 3 // excess kurtosis

	// Standard normal quantile
	z := normalInverse(1 - confidence)

	// Cornish-Fisher expansion
	zCF := z + (z*z-1)*skew/6 + (z*z*z-3*z)*kurt/24 - (2*z*z*z-5*z)*skew*skew/36

	varReturn := -mean + zCF*stdDev*math.Sqrt(float64(horizon))
	varAbsolute := math.Abs(varReturn * portfolioValue)

	return &VaRResult{
		Confidence:  confidence,
		Method:      CornishFisherVaR,
		VaRAbsolute: varAbsolute,
		VaRRelative: math.Abs(varReturn) * 100,
		Horizon:     horizon,
	}
}

// calculateCVaR calculates Conditional VaR (Expected Shortfall)
func (e *RiskAnalyticsEngine) calculateCVaR(returns []float64, portfolioValue float64, confidence float64) *CVaRResult {
	if len(returns) == 0 {
		return &CVaRResult{Confidence: confidence}
	}

	sortedReturns := make([]float64, len(returns))
	copy(sortedReturns, returns)
	sort.Float64s(sortedReturns)

	// Find returns below VaR threshold
	percentile := 1 - confidence
	cutoff := int(float64(len(sortedReturns)) * percentile)
	if cutoff == 0 {
		cutoff = 1
	}

	// Average of tail losses
	tailSum := 0.0
	for i := 0; i < cutoff; i++ {
		tailSum += sortedReturns[i]
	}
	avgTailLoss := tailSum / float64(cutoff)

	cvarAbsolute := math.Abs(avgTailLoss * portfolioValue)

	return &CVaRResult{
		Confidence:      confidence,
		CVaRAbsolute:    cvarAbsolute,
		CVaRRelative:    math.Abs(avgTailLoss) * 100,
		AverageTailLoss: avgTailLoss,
	}
}

// runStressTests executes predefined stress scenarios
func (e *RiskAnalyticsEngine) runStressTests(ctx context.Context, positions []PortfolioPosition, portfolioValue float64) []StressTestResult {
	scenarios := []struct {
		Name        string
		Type        string
		Description string
		Shocks      map[string]float64
	}{
		{
			Name:        "2008 Financial Crisis",
			Type:        "historical",
			Description: "Recreates 2008 market conditions with equity crash and credit spread widening",
			Shocks: map[string]float64{
				"equity":        -0.50,
				"fixed_income":  0.05,
				"credit_spread": 0.10,
				"volatility":    2.5,
				"commodities":   -0.35,
			},
		},
		{
			Name:        "COVID-19 March 2020",
			Type:        "historical",
			Description: "Recreates March 2020 pandemic crash with rapid drawdown",
			Shocks: map[string]float64{
				"equity":        -0.34,
				"fixed_income":  0.02,
				"credit_spread": 0.04,
				"volatility":    3.0,
				"commodities":   -0.25,
			},
		},
		{
			Name:        "Interest Rate Shock +200bp",
			Type:        "hypothetical",
			Description: "Parallel shift of yield curve by 200 basis points",
			Shocks: map[string]float64{
				"equity":       -0.10,
				"fixed_income": -0.15,
				"rates":        0.02,
				"commodities":  -0.05,
			},
		},
		{
			Name:        "Tech Sector Correction",
			Type:        "hypothetical",
			Description: "30% decline in technology stocks with rotation to value",
			Shocks: map[string]float64{
				"technology": -0.30,
				"equity":     -0.15,
				"growth":     -0.25,
				"value":      0.05,
			},
		},
		{
			Name:        "Stagflation Scenario",
			Type:        "hypothetical",
			Description: "High inflation with slowing growth",
			Shocks: map[string]float64{
				"equity":       -0.20,
				"fixed_income": -0.10,
				"commodities":  0.20,
				"real_estate":  -0.15,
				"tips":         0.05,
			},
		},
		{
			Name:        "Dollar Collapse",
			Type:        "hypothetical",
			Description: "20% depreciation of USD",
			Shocks: map[string]float64{
				"usd":           -0.20,
				"international": 0.15,
				"emerging":      0.10,
				"commodities":   0.15,
				"gold":          0.25,
			},
		},
	}

	results := make([]StressTestResult, 0, len(scenarios))

	for _, scenario := range scenarios {
		positionImpacts := make([]PositionImpact, 0, len(positions))
		totalImpact := 0.0

		for _, p := range positions {
			shock := e.getApplicableShock(p, scenario.Shocks)
			impact := p.MarketValue * shock

			positionImpacts = append(positionImpacts, PositionImpact{
				SecurityID:    p.SecurityID,
				SecurityName:  p.SecurityName,
				CurrentValue:  p.MarketValue,
				StressedValue: p.MarketValue + impact,
				Impact:        impact,
				ImpactPercent: shock * 100,
			})

			totalImpact += impact
		}

		// Sort by impact
		sort.Slice(positionImpacts, func(i, j int) bool {
			return positionImpacts[i].Impact < positionImpacts[j].Impact
		})

		// Estimate recovery (simplified)
		recoveryDays := int(math.Abs(totalImpact/portfolioValue) * 365)

		results = append(results, StressTestResult{
			ScenarioName:    scenario.Name,
			ScenarioType:    scenario.Type,
			Description:     scenario.Description,
			ShockParameters: scenario.Shocks,
			PortfolioImpact: totalImpact,
			ImpactPercent:   (totalImpact / portfolioValue) * 100,
			PositionImpacts: positionImpacts,
			RecoveryDays:    recoveryDays,
		})
	}

	return results
}

// getApplicableShock determines the shock to apply to a position
func (e *RiskAnalyticsEngine) getApplicableShock(p PortfolioPosition, shocks map[string]float64) float64 {
	// Check for specific sector/asset class shocks first
	assetClassLower := strings.ToLower(p.AssetClass)
	sectorLower := strings.ToLower(p.Sector)

	// Check sector-specific shock
	if shock, ok := shocks[sectorLower]; ok {
		return shock
	}

	// Check asset class shock
	switch assetClassLower {
	case "equity", "stock", "stocks":
		if shock, ok := shocks["equity"]; ok {
			return shock
		}
	case "fixed income", "bond", "bonds":
		if shock, ok := shocks["fixed_income"]; ok {
			return shock
		}
	case "commodity", "commodities":
		if shock, ok := shocks["commodities"]; ok {
			return shock
		}
	case "real estate", "reit":
		if shock, ok := shocks["real_estate"]; ok {
			return shock
		}
	}

	// Default to equity shock
	if shock, ok := shocks["equity"]; ok {
		return shock * 0.5 // Apply half the equity shock as default
	}

	return -0.10 // Default 10% loss
}

// calculateFactorRisk performs factor-based risk decomposition
func (e *RiskAnalyticsEngine) calculateFactorRisk(ctx context.Context, positions []PortfolioPosition) *FactorRiskResult {
	result := &FactorRiskResult{
		FactorContributions: make(map[string]*FactorContribution),
		MarginalVaRByFactor: make(map[string]float64),
	}

	// Calculate total portfolio volatility
	portfolioReturns := e.calculatePortfolioReturns(positions)
	totalVol := calculateStdDev(portfolioReturns) * math.Sqrt(252) // Annualized
	result.TotalVolatility = totalVol

	// Standard factors
	factors := []string{"Market", "Size", "Value", "Momentum", "Quality", "Volatility"}

	// Simplified factor decomposition
	systematicRisk := 0.0
	for _, factor := range factors {
		// In production, these would come from a factor model database
		exposure := 0.0
		factorVol := 0.15 // Assume 15% factor volatility

		switch factor {
		case "Market":
			exposure = 1.0
			factorVol = 0.18
		case "Size":
			exposure = 0.2
			factorVol = 0.08
		case "Value":
			exposure = -0.1
			factorVol = 0.07
		case "Momentum":
			exposure = 0.15
			factorVol = 0.12
		}

		contribution := exposure * exposure * factorVol * factorVol
		systematicRisk += contribution

		result.FactorContributions[factor] = &FactorContribution{
			FactorName:        factor,
			Exposure:          exposure,
			FactorVolatility:  factorVol,
			ContributionToVar: math.Sqrt(contribution),
			ContributionPct:   contribution / (totalVol * totalVol) * 100,
		}
	}

	result.SystematicRisk = math.Sqrt(systematicRisk)
	result.IdiosyncraticRisk = math.Sqrt(math.Max(0, totalVol*totalVol-systematicRisk))

	return result
}

// calculateConcentrationRisk measures portfolio concentration
func (e *RiskAnalyticsEngine) calculateConcentrationRisk(positions []PortfolioPosition) *ConcentrationRiskResult {
	result := &ConcentrationRiskResult{
		SectorConcentration:   make(map[string]float64),
		CountryConcentration:  make(map[string]float64),
		CurrencyConcentration: make(map[string]float64),
		SingleNameRisk:        make([]SingleNameRisk, 0),
	}

	// Calculate HHI
	hhi := 0.0
	for _, p := range positions {
		hhi += p.Weight * p.Weight
	}
	result.HerfindahlIndex = hhi
	result.EffectivePositions = 1.0 / hhi

	// Sort by weight for top holdings
	sortedPositions := make([]PortfolioPosition, len(positions))
	copy(sortedPositions, positions)
	sort.Slice(sortedPositions, func(i, j int) bool {
		return sortedPositions[i].Weight > sortedPositions[j].Weight
	})

	// Top 10 holdings percentage
	top10 := 0.0
	for i := 0; i < min(10, len(sortedPositions)); i++ {
		top10 += sortedPositions[i].Weight
	}
	result.TopHoldingsPct = top10 * 100

	// Sector concentration
	for _, p := range positions {
		sector := p.Sector
		if sector == "" {
			sector = "Other"
		}
		result.SectorConcentration[sector] += p.Weight
	}

	// Country concentration
	for _, p := range positions {
		country := p.Country
		if country == "" {
			country = "Other"
		}
		result.CountryConcentration[country] += p.Weight
	}

	// Currency concentration
	for _, p := range positions {
		currency := p.Currency
		if currency == "" {
			currency = "USD"
		}
		result.CurrencyConcentration[currency] += p.Weight
	}

	// Single name risk (positions > 5%)
	for _, p := range positions {
		if p.Weight > 0.05 {
			result.SingleNameRisk = append(result.SingleNameRisk, SingleNameRisk{
				SecurityID:   p.SecurityID,
				SecurityName: p.SecurityName,
				Weight:       p.Weight,
			})
		}
	}

	return result
}

// calculateLiquidityRisk assesses portfolio liquidity
func (e *RiskAnalyticsEngine) calculateLiquidityRisk(positions []PortfolioPosition, portfolioValue float64) *LiquidityRiskResult {
	result := &LiquidityRiskResult{
		LiquidityBuckets: make(map[string]float64),
		IlliquidHoldings: make([]IlliquidHolding, 0),
	}

	totalDaysWeighted := 0.0
	totalLiquidityCost := 0.0

	for _, p := range positions {
		// Estimate days to liquidate (position value / daily volume)
		daysToLiquidate := 1.0
		if p.AvgDailyVolume > 0 {
			daysToLiquidate = p.MarketValue / p.AvgDailyVolume
		}

		// Estimate liquidity cost (market impact)
		liquidityCost := math.Min(0.05, 0.001*math.Sqrt(daysToLiquidate))

		totalDaysWeighted += p.Weight * daysToLiquidate
		totalLiquidityCost += p.Weight * liquidityCost

		// Categorize into buckets
		var bucket string
		switch {
		case daysToLiquidate <= 1:
			bucket = "1 day"
		case daysToLiquidate <= 5:
			bucket = "1 week"
		case daysToLiquidate <= 20:
			bucket = "1 month"
		default:
			bucket = ">1 month"
		}
		result.LiquidityBuckets[bucket] += p.Weight

		// Flag illiquid holdings (>5 days to liquidate)
		if daysToLiquidate > 5 {
			result.IlliquidHoldings = append(result.IlliquidHoldings, IlliquidHolding{
				SecurityID:      p.SecurityID,
				SecurityName:    p.SecurityName,
				Weight:          p.Weight,
				AvgDailyVolume:  p.AvgDailyVolume,
				DaysToLiquidate: daysToLiquidate,
				LiquidityCost:   liquidityCost,
			})
		}
	}

	result.DaysToLiquidate = totalDaysWeighted
	result.LiquidityCostBps = totalLiquidityCost * 10000

	return result
}

// calculateTailRisk analyzes distribution tail characteristics
func (e *RiskAnalyticsEngine) calculateTailRisk(returns []float64) *TailRiskMetrics {
	if len(returns) < 20 {
		return &TailRiskMetrics{}
	}

	result := &TailRiskMetrics{
		Skewness:       calculateSkewness(returns),
		ExcessKurtosis: calculateKurtosis(returns) - 3,
	}

	// Calculate max drawdown
	peak := 0.0
	maxDD := 0.0
	cumReturn := 1.0

	for i, r := range returns {
		cumReturn *= (1 + r)
		if cumReturn > peak {
			peak = cumReturn
		}
		drawdown := (peak - cumReturn) / peak
		if drawdown > maxDD {
			maxDD = drawdown
			// Note: in production, track the actual date
			_ = i
		}
	}
	result.MaxDrawdown = maxDD

	// Worst periods
	sortedReturns := make([]float64, len(returns))
	copy(sortedReturns, returns)
	sort.Float64s(sortedReturns)

	result.WorstDay = sortedReturns[0]

	// Weekly returns (simplified: sum of 5 days)
	if len(returns) >= 5 {
		worstWeek := 0.0
		for i := 0; i <= len(returns)-5; i++ {
			weekReturn := 0.0
			for j := 0; j < 5; j++ {
				weekReturn += returns[i+j]
			}
			if weekReturn < worstWeek {
				worstWeek = weekReturn
			}
		}
		result.WorstWeek = worstWeek
	}

	// Monthly returns (simplified: sum of 21 days)
	if len(returns) >= 21 {
		worstMonth := 0.0
		for i := 0; i <= len(returns)-21; i++ {
			monthReturn := 0.0
			for j := 0; j < 21; j++ {
				monthReturn += returns[i+j]
			}
			if monthReturn < worstMonth {
				worstMonth = monthReturn
			}
		}
		result.WorstMonth = worstMonth
	}

	return result
}

// calculateRiskContributions computes position-level risk contributions
func (e *RiskAnalyticsEngine) calculateRiskContributions(positions []PortfolioPosition, portfolioReturns []float64, varResult *VaRResult) []PositionRiskContribution {
	contributions := make([]PositionRiskContribution, 0, len(positions))

	portfolioVol := calculateStdDev(portfolioReturns)
	if portfolioVol == 0 {
		portfolioVol = 0.01
	}

	for _, p := range positions {
		posVol := calculateStdDev(p.Returns)

		// Calculate beta to portfolio
		beta := e.calculateBeta(p.Returns, portfolioReturns)

		// Marginal VaR
		marginalVaR := beta * varResult.VaRRelative / 100

		// Component VaR
		componentVaR := p.Weight * marginalVaR * varResult.VaRAbsolute

		contributions = append(contributions, PositionRiskContribution{
			SecurityID:      p.SecurityID,
			SecurityName:    p.SecurityName,
			Weight:          p.Weight,
			Volatility:      posVol * math.Sqrt(252),
			BetaToPortfolio: beta,
			MarginalVaR:     marginalVaR,
			ComponentVaR:    componentVaR,
			ContributionPct: p.Weight * beta / portfolioVol * 100,
		})
	}

	// Sort by contribution
	sort.Slice(contributions, func(i, j int) bool {
		return contributions[i].ComponentVaR > contributions[j].ComponentVaR
	})

	return contributions
}

// calculateCorrelationMatrix computes asset correlations
func (e *RiskAnalyticsEngine) calculateCorrelationMatrix(positions []PortfolioPosition) *CorrelationMatrix {
	n := len(positions)
	if n == 0 {
		return nil
	}

	assets := make([]string, n)
	correlations := make([][]float64, n)

	for i, p := range positions {
		assets[i] = p.SecurityName
		correlations[i] = make([]float64, n)

		for j, q := range positions {
			if i == j {
				correlations[i][j] = 1.0
			} else if i < j {
				corr := e.calculateCorrelation(p.Returns, q.Returns)
				correlations[i][j] = corr
				correlations[j][i] = corr
			}
		}
	}

	return &CorrelationMatrix{
		Assets:       assets,
		Correlations: correlations,
	}
}

// Database methods
func (e *RiskAnalyticsEngine) getPortfolioPositions(ctx context.Context, portfolioID string, asOfDate time.Time, lookbackDays int) ([]PortfolioPosition, error) {
	query := `
		WITH position_data AS (
SELECT 
h.security_id,
s.security_name,
COALESCE(s.asset_class, 'Other') as asset_class,
COALESCE(s.sector, 'Other') as sector,
COALESCE(s.country, 'US') as country,
COALESCE(s.currency, 'USD') as currency,
h.market_value,
h.weight,
COALESCE(s.avg_daily_volume, 1000000) as avg_daily_volume
FROM portfolio_holdings h
JOIN securities s ON h.security_id = s.security_id
WHERE h.portfolio_id = $1
AND h.holding_date = $2
)
		SELECT * FROM position_data ORDER BY weight DESC
	`

	rows, err := e.db.QueryContext(ctx, query, portfolioID, asOfDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	positions := []PortfolioPosition{}
	securityIDs := []string{}

	for rows.Next() {
		var p PortfolioPosition
		err := rows.Scan(
			&p.SecurityID, &p.SecurityName, &p.AssetClass, &p.Sector,
			&p.Country, &p.Currency, &p.MarketValue, &p.Weight, &p.AvgDailyVolume,
		)
		if err != nil {
			return nil, err
		}
		positions = append(positions, p)
		securityIDs = append(securityIDs, p.SecurityID)
	}

	// Fetch historical returns
	if len(securityIDs) > 0 {
		startDate := asOfDate.AddDate(0, 0, -lookbackDays)
		returnsMap, err := e.getSecurityReturns(ctx, securityIDs, startDate, asOfDate)
		if err == nil {
			for i := range positions {
				if returns, ok := returnsMap[positions[i].SecurityID]; ok {
					positions[i].Returns = returns
					positions[i].Volatility = calculateStdDev(returns)
				}
			}
		}
	}

	return positions, nil
}

func (e *RiskAnalyticsEngine) getSecurityReturns(ctx context.Context, securityIDs []string, startDate, endDate time.Time) (map[string][]float64, error) {
	query := `
		SELECT security_id, return_value
		FROM security_returns
		WHERE security_id = ANY($1)
		AND return_date BETWEEN $2 AND $3
		ORDER BY security_id, return_date
	`

	rows, err := e.db.QueryContext(ctx, query, securityIDs, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]float64)
	for rows.Next() {
		var securityID string
		var returnValue float64
		if err := rows.Scan(&securityID, &returnValue); err != nil {
			return nil, err
		}
		result[securityID] = append(result[securityID], returnValue)
	}

	return result, nil
}

// Helper methods
func (e *RiskAnalyticsEngine) calculatePortfolioReturns(positions []PortfolioPosition) []float64 {
	if len(positions) == 0 {
		return []float64{}
	}

	// Find minimum return length
	minLen := len(positions[0].Returns)
	for _, p := range positions {
		if len(p.Returns) < minLen {
			minLen = len(p.Returns)
		}
	}

	if minLen == 0 {
		return []float64{}
	}

	portfolioReturns := make([]float64, minLen)
	for i := 0; i < minLen; i++ {
		for _, p := range positions {
			if i < len(p.Returns) {
				portfolioReturns[i] += p.Weight * p.Returns[i]
			}
		}
	}

	return portfolioReturns
}

func (e *RiskAnalyticsEngine) calculateBeta(assetReturns, portfolioReturns []float64) float64 {
	n := min(len(assetReturns), len(portfolioReturns))
	if n < 2 {
		return 1.0
	}

	meanA := calculateMean(assetReturns[:n])
	meanP := calculateMean(portfolioReturns[:n])

	covariance := 0.0
	varianceP := 0.0

	for i := 0; i < n; i++ {
		diffA := assetReturns[i] - meanA
		diffP := portfolioReturns[i] - meanP
		covariance += diffA * diffP
		varianceP += diffP * diffP
	}

	if varianceP == 0 {
		return 1.0
	}

	return covariance / varianceP
}

func (e *RiskAnalyticsEngine) calculateCorrelation(x, y []float64) float64 {
	n := min(len(x), len(y))
	if n < 2 {
		return 0
	}

	meanX := calculateMean(x[:n])
	meanY := calculateMean(y[:n])

	var sumXY, sumX2, sumY2 float64
	for i := 0; i < n; i++ {
		diffX := x[i] - meanX
		diffY := y[i] - meanY
		sumXY += diffX * diffY
		sumX2 += diffX * diffX
		sumY2 += diffY * diffY
	}

	denominator := math.Sqrt(sumX2 * sumY2)
	if denominator == 0 {
		return 0
	}

	return sumXY / denominator
}

// Statistical helper functions
func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateStdDev(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	mean := calculateMean(values)
	sumSquares := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}
	return math.Sqrt(sumSquares / float64(len(values)-1))
}

func calculateSkewness(values []float64) float64 {
	n := len(values)
	if n < 3 {
		return 0
	}

	mean := calculateMean(values)
	stdDev := calculateStdDev(values)
	if stdDev == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		z := (v - mean) / stdDev
		sum += z * z * z
	}

	return sum * float64(n) / float64((n-1)*(n-2))
}

func calculateKurtosis(values []float64) float64 {
	n := len(values)
	if n < 4 {
		return 3 // Normal distribution kurtosis
	}

	mean := calculateMean(values)
	stdDev := calculateStdDev(values)
	if stdDev == 0 {
		return 3
	}

	sum := 0.0
	for _, v := range values {
		z := (v - mean) / stdDev
		sum += z * z * z * z
	}

	return sum / float64(n)
}

// normalInverse approximates the inverse normal CDF
func normalInverse(p float64) float64 {
	// Rational approximation for lower tail
	a := []float64{-3.969683028665376e+01, 2.209460984245205e+02,
		-2.759285104469687e+02, 1.383577518672690e+02,
		-3.066479806614716e+01, 2.506628277459239e+00}
	b := []float64{-5.447609879822406e+01, 1.615858368580409e+02,
		-1.556989798598866e+02, 6.680131188771972e+01, -1.328068155288572e+01}
	c := []float64{-7.784894002430293e-03, -3.223964580411365e-01,
		-2.400758277161838e+00, -2.549732539343734e+00,
		4.374664141464968e+00, 2.938163982698783e+00}
	d := []float64{7.784695709041462e-03, 3.224671290700398e-01,
		2.445134137142996e+00, 3.754408661907416e+00}

	pLow := 0.02425
	pHigh := 1 - pLow

	var q, r float64
	if p < pLow {
		q = math.Sqrt(-2 * math.Log(p))
		return (((((c[0]*q+c[1])*q+c[2])*q+c[3])*q+c[4])*q + c[5]) /
			((((d[0]*q+d[1])*q+d[2])*q+d[3])*q + 1)
	} else if p <= pHigh {
		q = p - 0.5
		r = q * q
		return (((((a[0]*r+a[1])*r+a[2])*r+a[3])*r+a[4])*r + a[5]) * q /
			(((((b[0]*r+b[1])*r+b[2])*r+b[3])*r+b[4])*r + 1)
	} else {
		q = math.Sqrt(-2 * math.Log(1-p))
		return -(((((c[0]*q+c[1])*q+c[2])*q+c[3])*q+c[4])*q + c[5]) /
			((((d[0]*q+d[1])*q+d[2])*q+d[3])*q + 1)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
