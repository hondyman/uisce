package attribution

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sort"
	"time"
)

// AttributionEngine provides institutional-grade performance attribution
// supporting Brinson-Fachler, factor-based, and multi-period analysis
type AttributionEngine struct {
	db *sql.DB
}

// NewAttributionEngine creates a new attribution engine instance
func NewAttributionEngine(db *sql.DB) *AttributionEngine {
	return &AttributionEngine{db: db}
}

// AttributionConfig configures the attribution calculation
type AttributionConfig struct {
	PortfolioID     string
	BenchmarkID     string
	StartDate       time.Time
	EndDate         time.Time
	Method          AttributionMethod
	HierarchyLevels []string // e.g., ["AssetClass", "Sector", "Security"]
	FactorModel     *FactorModelConfig
	Currency        string
	Frequency       string // "daily", "weekly", "monthly"
}

// AttributionMethod defines the attribution methodology
type AttributionMethod string

const (
	BrinsonFachler AttributionMethod = "brinson_fachler"
	BrinsonHood    AttributionMethod = "brinson_hood"
	FactorBased    AttributionMethod = "factor_based"
	RiskAdjusted   AttributionMethod = "risk_adjusted"
)

// FactorModelConfig configures factor-based attribution
type FactorModelConfig struct {
	ModelType     string   // "fama_french_3", "fama_french_5", "barra", "custom"
	Factors       []string // factor names
	RiskFreeRate  float64
	FactorReturns map[string][]float64 // historical factor returns
}

// AttributionResult contains comprehensive attribution analysis
type AttributionResult struct {
	Config          AttributionConfig
	TotalReturn     float64
	BenchmarkReturn float64
	ActiveReturn    float64

	// Brinson-Fachler decomposition
	AllocationEffect  float64
	SelectionEffect   float64
	InteractionEffect float64

	// Multi-level hierarchy breakdown
	HierarchyBreakdown []HierarchyLevel

	// Security-level attribution
	SecurityAttribution []SecurityContribution

	// Factor-based attribution (if applicable)
	FactorAttribution *FactorAttributionResult

	// Time series for charting
	PeriodReturns []PeriodAttribution

	// Risk-adjusted metrics
	RiskMetrics *AttributionRiskMetrics

	// Metadata
	CalculatedAt time.Time
	Warnings     []string
}

// HierarchyLevel represents attribution at a classification level
type HierarchyLevel struct {
	Level      string
	Categories []CategoryAttribution
}

// CategoryAttribution represents attribution for a single category
type CategoryAttribution struct {
	Name              string
	PortfolioWeight   float64
	BenchmarkWeight   float64
	ActiveWeight      float64
	PortfolioReturn   float64
	BenchmarkReturn   float64
	ActiveReturn      float64
	AllocationEffect  float64
	SelectionEffect   float64
	InteractionEffect float64
	TotalEffect       float64

	// Nested breakdown for drill-down
	SubCategories []CategoryAttribution
}

// SecurityContribution represents individual security contribution
type SecurityContribution struct {
	SecurityID           string
	SecurityName         string
	AssetClass           string
	Sector               string
	PortfolioWeight      float64
	BenchmarkWeight      float64
	ActiveWeight         float64
	SecurityReturn       float64
	ContributionToReturn float64
	ContributionToActive float64
}

// FactorAttributionResult contains factor-based attribution
type FactorAttributionResult struct {
	ModelType        string
	FactorExposures  map[string]float64 // beta to each factor
	FactorReturns    map[string]float64 // return attributed to each factor
	Alpha            float64            // unexplained return
	RSquared         float64            // model fit
	TrackingError    float64
	InformationRatio float64

	// Time series of factor exposures
	RollingExposures []RollingFactorExposure
}

// RollingFactorExposure tracks factor exposures over time
type RollingFactorExposure struct {
	Date      time.Time
	Exposures map[string]float64
}

// PeriodAttribution represents attribution for a single period
type PeriodAttribution struct {
	PeriodStart       time.Time
	PeriodEnd         time.Time
	PortfolioReturn   float64
	BenchmarkReturn   float64
	ActiveReturn      float64
	AllocationEffect  float64
	SelectionEffect   float64
	InteractionEffect float64
	CumulativeActive  float64
}

// AttributionRiskMetrics contains risk-adjusted attribution
type AttributionRiskMetrics struct {
	TrackingError    float64
	InformationRatio float64
	ActiveRisk       float64
	ActiveSharePct   float64
	BetaToBenchmark  float64

	// Risk decomposition
	SystematicRisk    float64
	IdiosyncraticRisk float64

	// Contribution to tracking error by category
	TrackingErrorContributions map[string]float64
}

// PortfolioHolding represents a position at a point in time
type PortfolioHolding struct {
	SecurityID   string
	SecurityName string
	AssetClass   string
	Sector       string
	Industry     string
	Country      string
	Currency     string
	MarketValue  float64
	Weight       float64
	BeginPrice   float64
	EndPrice     float64
	PeriodReturn float64
}

// Calculate performs comprehensive performance attribution
func (e *AttributionEngine) Calculate(ctx context.Context, config AttributionConfig) (*AttributionResult, error) {
	// Fetch portfolio and benchmark holdings
	portfolioHoldings, err := e.getPortfolioHoldings(ctx, config.PortfolioID, config.StartDate, config.EndDate)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch portfolio holdings: %w", err)
	}

	benchmarkHoldings, err := e.getBenchmarkHoldings(ctx, config.BenchmarkID, config.StartDate, config.EndDate)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch benchmark holdings: %w", err)
	}

	result := &AttributionResult{
		Config:       config,
		CalculatedAt: time.Now(),
		Warnings:     []string{},
	}

	// Calculate total returns
	result.TotalReturn = e.calculatePortfolioReturn(portfolioHoldings)
	result.BenchmarkReturn = e.calculatePortfolioReturn(benchmarkHoldings)

	// Calculate relative return
	// benchmarkTotalReturn := totalBenchmarkReturn // Unused
	// _ = benchmarkTotalReturn // Acknowledged unused for future enhancement

	result.ActiveReturn = result.TotalReturn - result.BenchmarkReturn

	// Perform attribution based on method
	switch config.Method {
	case BrinsonFachler:
		e.calculateBrinsonFachler(portfolioHoldings, benchmarkHoldings, result)
	case BrinsonHood:
		e.calculateBrinsonHood(portfolioHoldings, benchmarkHoldings, result)
	case FactorBased:
		if config.FactorModel != nil {
			factorResult, err := e.calculateFactorAttribution(ctx, portfolioHoldings, config.FactorModel)
			if err != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Factor attribution failed: %v", err))
			} else {
				result.FactorAttribution = factorResult
			}
		}
	case RiskAdjusted:
		e.calculateBrinsonFachler(portfolioHoldings, benchmarkHoldings, result)
	}

	// Build hierarchy breakdown
	result.HierarchyBreakdown = e.buildHierarchyBreakdown(portfolioHoldings, benchmarkHoldings, config.HierarchyLevels)

	// Calculate security-level attribution
	result.SecurityAttribution = e.calculateSecurityAttribution(portfolioHoldings, benchmarkHoldings, result.BenchmarkReturn)

	// Calculate period returns for time series
	result.PeriodReturns, err = e.calculatePeriodAttribution(ctx, config)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Period attribution warning: %v", err))
	}

	// Calculate risk metrics
	result.RiskMetrics = e.calculateRiskMetrics(portfolioHoldings, benchmarkHoldings, result.PeriodReturns)

	return result, nil
}

// calculateBrinsonFachler implements the Brinson-Fachler attribution model
func (e *AttributionEngine) calculateBrinsonFachler(portfolio, benchmark []PortfolioHolding, result *AttributionResult) {
	// Group by asset class for top-level attribution
	portfolioByClass := groupByAssetClass(portfolio)
	benchmarkByClass := groupByAssetClass(benchmark)

	benchmarkTotalReturn := e.calculatePortfolioReturn(benchmark)

	var allocationEffect, selectionEffect, interactionEffect float64

	// Get all unique asset classes
	allClasses := make(map[string]bool)
	for class := range portfolioByClass {
		allClasses[class] = true
	}
	for class := range benchmarkByClass {
		allClasses[class] = true
	}

	for class := range allClasses {
		pHoldings := portfolioByClass[class]
		bHoldings := benchmarkByClass[class]

		// Calculate weights
		pWeight := sumWeights(pHoldings)
		bWeight := sumWeights(bHoldings)

		// Calculate returns within each class
		pReturn := calculateGroupReturn(pHoldings)
		bReturn := calculateGroupReturn(bHoldings)

		// Brinson-Fachler formulas:
		// Allocation Effect = (Wp - Wb) * (Rb - RT)
		// Selection Effect = Wb * (Rp - Rb)
		// Interaction Effect = (Wp - Wb) * (Rp - Rb)

		allocation := (pWeight - bWeight) * (bReturn - benchmarkTotalReturn)
		selection := bWeight * (pReturn - bReturn)
		interaction := (pWeight - bWeight) * (pReturn - bReturn)

		allocationEffect += allocation
		selectionEffect += selection
		interactionEffect += interaction
	}

	result.AllocationEffect = allocationEffect
	result.SelectionEffect = selectionEffect
	result.InteractionEffect = interactionEffect
}

// calculateBrinsonHood implements the Brinson-Hood-Beebower attribution model
func (e *AttributionEngine) calculateBrinsonHood(portfolio, benchmark []PortfolioHolding, result *AttributionResult) {
	// Similar to Brinson-Fachler but with different interaction handling
	portfolioByClass := groupByAssetClass(portfolio)
	benchmarkByClass := groupByAssetClass(benchmark)

	// benchmarkTotalReturn := e.calculatePortfolioReturn(benchmark) // Unused in BHB

	var allocationEffect, selectionEffect float64

	allClasses := make(map[string]bool)
	for class := range portfolioByClass {
		allClasses[class] = true
	}
	for class := range benchmarkByClass {
		allClasses[class] = true
	}

	for class := range allClasses {
		pHoldings := portfolioByClass[class]
		bHoldings := benchmarkByClass[class]

		pWeight := sumWeights(pHoldings)
		bWeight := sumWeights(bHoldings)
		pReturn := calculateGroupReturn(pHoldings)
		bReturn := calculateGroupReturn(bHoldings)

		// BHB formulas:
		// Allocation = (Wp - Wb) * Rb
		// Selection = Wp * (Rp - Rb)
		allocation := (pWeight - bWeight) * bReturn
		selection := pWeight * (pReturn - bReturn)

		allocationEffect += allocation
		selectionEffect += selection
	}

	result.AllocationEffect = allocationEffect
	result.SelectionEffect = selectionEffect
	result.InteractionEffect = result.ActiveReturn - allocationEffect - selectionEffect
}

// calculateFactorAttribution performs factor-based attribution analysis
func (e *AttributionEngine) calculateFactorAttribution(ctx context.Context, holdings []PortfolioHolding, config *FactorModelConfig) (*FactorAttributionResult, error) {
	result := &FactorAttributionResult{
		ModelType:       config.ModelType,
		FactorExposures: make(map[string]float64),
		FactorReturns:   make(map[string]float64),
	}

	// Fetch factor exposures for each holding
	exposures, err := e.getSecurityFactorExposures(ctx, holdings, config.Factors)
	if err != nil {
		return nil, err
	}

	// Calculate portfolio-level factor exposures (weighted average)
	for _, factor := range config.Factors {
		portfolioExposure := 0.0
		for _, h := range holdings {
			if exp, ok := exposures[h.SecurityID]; ok {
				if factorExp, ok := exp[factor]; ok {
					portfolioExposure += h.Weight * factorExp
				}
			}
		}
		result.FactorExposures[factor] = portfolioExposure
	}

	// Calculate return attributed to each factor
	totalExplained := 0.0
	if len(config.FactorReturns) > 0 {
		for _, factor := range config.Factors {
			if returns, ok := config.FactorReturns[factor]; ok && len(returns) > 0 {
				// Use the most recent factor return
				factorReturn := returns[len(returns)-1]
				attributedReturn := result.FactorExposures[factor] * factorReturn
				result.FactorReturns[factor] = attributedReturn
				totalExplained += attributedReturn
			}
		}
	}

	// Calculate alpha (unexplained return)
	portfolioReturn := e.calculatePortfolioReturn(holdings)
	result.Alpha = portfolioReturn - totalExplained - config.RiskFreeRate

	// Calculate R-squared
	result.RSquared = e.calculateRSquared(holdings, config.FactorReturns, result.FactorExposures)

	return result, nil
}

// buildHierarchyBreakdown creates multi-level attribution breakdown
func (e *AttributionEngine) buildHierarchyBreakdown(portfolio, benchmark []PortfolioHolding, levels []string) []HierarchyLevel {
	result := make([]HierarchyLevel, 0, len(levels))

	benchmarkReturn := e.calculatePortfolioReturn(benchmark)

	for _, level := range levels {
		hierarchyLevel := HierarchyLevel{
			Level:      level,
			Categories: make([]CategoryAttribution, 0),
		}

		// Group by the current level
		portfolioGroups := e.groupByLevel(portfolio, level)
		benchmarkGroups := e.groupByLevel(benchmark, level)

		// Get all unique categories
		allCategories := make(map[string]bool)
		for cat := range portfolioGroups {
			allCategories[cat] = true
		}
		for cat := range benchmarkGroups {
			allCategories[cat] = true
		}

		for category := range allCategories {
			pHoldings := portfolioGroups[category]
			bHoldings := benchmarkGroups[category]

			pWeight := sumWeights(pHoldings)
			bWeight := sumWeights(bHoldings)
			pReturn := calculateGroupReturn(pHoldings)
			bReturn := calculateGroupReturn(bHoldings)

			// Brinson-Fachler at this level
			allocation := (pWeight - bWeight) * (bReturn - benchmarkReturn)
			selection := bWeight * (pReturn - bReturn)
			interaction := (pWeight - bWeight) * (pReturn - bReturn)

			catAttr := CategoryAttribution{
				Name:              category,
				PortfolioWeight:   pWeight,
				BenchmarkWeight:   bWeight,
				ActiveWeight:      pWeight - bWeight,
				PortfolioReturn:   pReturn,
				BenchmarkReturn:   bReturn,
				ActiveReturn:      pReturn - bReturn,
				AllocationEffect:  allocation,
				SelectionEffect:   selection,
				InteractionEffect: interaction,
				TotalEffect:       allocation + selection + interaction,
			}

			hierarchyLevel.Categories = append(hierarchyLevel.Categories, catAttr)
		}

		// Sort by absolute total effect descending
		sort.Slice(hierarchyLevel.Categories, func(i, j int) bool {
			return math.Abs(hierarchyLevel.Categories[i].TotalEffect) > math.Abs(hierarchyLevel.Categories[j].TotalEffect)
		})

		result = append(result, hierarchyLevel)
	}

	return result
}

// calculateSecurityAttribution computes security-level attribution
func (e *AttributionEngine) calculateSecurityAttribution(portfolio, benchmark []PortfolioHolding, benchmarkReturn float64) []SecurityContribution {
	// Create benchmark lookup
	benchmarkMap := make(map[string]PortfolioHolding)
	for _, h := range benchmark {
		benchmarkMap[h.SecurityID] = h
	}

	result := make([]SecurityContribution, 0, len(portfolio))

	for _, p := range portfolio {
		bHolding, inBenchmark := benchmarkMap[p.SecurityID]

		bWeight := 0.0
		if inBenchmark {
			bWeight = bHolding.Weight
		}

		// Contribution to return = weight * return
		contributionToReturn := p.Weight * p.PeriodReturn

		// Contribution to active = (Wp - Wb) * Rp + Wb * (Rp - Rb)
		activeContribution := (p.Weight - bWeight) * p.PeriodReturn
		if inBenchmark {
			activeContribution += bWeight * (p.PeriodReturn - bHolding.PeriodReturn)
		}

		result = append(result, SecurityContribution{
			SecurityID:           p.SecurityID,
			SecurityName:         p.SecurityName,
			AssetClass:           p.AssetClass,
			Sector:               p.Sector,
			PortfolioWeight:      p.Weight,
			BenchmarkWeight:      bWeight,
			ActiveWeight:         p.Weight - bWeight,
			SecurityReturn:       p.PeriodReturn,
			ContributionToReturn: contributionToReturn,
			ContributionToActive: activeContribution,
		})
	}

	// Sort by absolute contribution to active
	sort.Slice(result, func(i, j int) bool {
		return math.Abs(result[i].ContributionToActive) > math.Abs(result[j].ContributionToActive)
	})

	return result
}

// calculatePeriodAttribution calculates attribution over multiple periods
func (e *AttributionEngine) calculatePeriodAttribution(ctx context.Context, config AttributionConfig) ([]PeriodAttribution, error) {
	periods := generatePeriods(config.StartDate, config.EndDate, config.Frequency)
	results := make([]PeriodAttribution, 0, len(periods))

	cumulativeActive := 0.0

	for _, period := range periods {
		// Fetch holdings for this period
		portfolioHoldings, err := e.getPortfolioHoldings(ctx, config.PortfolioID, period.Start, period.End)
		if err != nil {
			continue
		}

		benchmarkHoldings, err := e.getBenchmarkHoldings(ctx, config.BenchmarkID, period.Start, period.End)
		if err != nil {
			continue
		}

		pReturn := e.calculatePortfolioReturn(portfolioHoldings)
		bReturn := e.calculatePortfolioReturn(benchmarkHoldings)

		// Calculate Brinson-Fachler for this period
		tempResult := &AttributionResult{}
		e.calculateBrinsonFachler(portfolioHoldings, benchmarkHoldings, tempResult)

		// Compound cumulative active return
		cumulativeActive = (1+cumulativeActive)*(1+(pReturn-bReturn)) - 1

		results = append(results, PeriodAttribution{
			PeriodStart:       period.Start,
			PeriodEnd:         period.End,
			PortfolioReturn:   pReturn,
			BenchmarkReturn:   bReturn,
			ActiveReturn:      pReturn - bReturn,
			AllocationEffect:  tempResult.AllocationEffect,
			SelectionEffect:   tempResult.SelectionEffect,
			InteractionEffect: tempResult.InteractionEffect,
			CumulativeActive:  cumulativeActive,
		})
	}

	return results, nil
}

// calculateRiskMetrics computes risk-adjusted attribution metrics
func (e *AttributionEngine) calculateRiskMetrics(portfolio, benchmark []PortfolioHolding, periods []PeriodAttribution) *AttributionRiskMetrics {
	if len(periods) < 2 {
		return &AttributionRiskMetrics{}
	}

	// Extract active returns
	activeReturns := make([]float64, len(periods))
	portfolioReturns := make([]float64, len(periods))
	benchmarkReturns := make([]float64, len(periods))

	for i, p := range periods {
		activeReturns[i] = p.ActiveReturn
		portfolioReturns[i] = p.PortfolioReturn
		benchmarkReturns[i] = p.BenchmarkReturn
	}

	// Calculate tracking error (std dev of active returns)
	trackingError := calculateStdDev(activeReturns) * math.Sqrt(12) // Annualized

	// Calculate mean active return
	meanActive := calculateMean(activeReturns) * 12 // Annualized

	// Information ratio
	informationRatio := 0.0
	if trackingError > 0 {
		informationRatio = meanActive / trackingError
	}

	// Active share (sum of absolute active weights / 2)
	activeShare := 0.0
	benchmarkMap := make(map[string]float64)
	for _, b := range benchmark {
		benchmarkMap[b.SecurityID] = b.Weight
	}
	for _, p := range portfolio {
		bWeight := benchmarkMap[p.SecurityID]
		activeShare += math.Abs(p.Weight - bWeight)
	}
	// Add benchmark securities not in portfolio
	portfolioMap := make(map[string]bool)
	for _, p := range portfolio {
		portfolioMap[p.SecurityID] = true
	}
	for _, b := range benchmark {
		if !portfolioMap[b.SecurityID] {
			activeShare += b.Weight
		}
	}
	activeShare = activeShare / 2 * 100 // Convert to percentage

	// Beta to benchmark
	beta := calculateBeta(portfolioReturns, benchmarkReturns)

	// Systematic and idiosyncratic risk
	portfolioVol := calculateStdDev(portfolioReturns) * math.Sqrt(12)
	benchmarkVol := calculateStdDev(benchmarkReturns) * math.Sqrt(12)

	systematicRisk := beta * benchmarkVol
	idiosyncraticRisk := math.Sqrt(math.Max(0, portfolioVol*portfolioVol-systematicRisk*systematicRisk))

	return &AttributionRiskMetrics{
		TrackingError:     trackingError,
		InformationRatio:  informationRatio,
		ActiveRisk:        portfolioVol,
		ActiveSharePct:    activeShare,
		BetaToBenchmark:   beta,
		SystematicRisk:    systematicRisk,
		IdiosyncraticRisk: idiosyncraticRisk,
	}
}

// Database query methods
func (e *AttributionEngine) getPortfolioHoldings(ctx context.Context, portfolioID string, startDate, endDate time.Time) ([]PortfolioHolding, error) {
	query := `
		SELECT 
			s.security_id,
			s.security_name,
			COALESCE(s.asset_class, 'Other') as asset_class,
			COALESCE(s.sector, 'Other') as sector,
			COALESCE(s.industry, 'Other') as industry,
			COALESCE(s.country, 'US') as country,
			COALESCE(s.currency, 'USD') as currency,
			h.market_value,
			h.weight,
			COALESCE(pb.price, 100) as begin_price,
			COALESCE(pe.price, pb.price, 100) as end_price
		FROM portfolio_holdings h
		JOIN securities s ON h.security_id = s.security_id
		LEFT JOIN security_prices pb ON s.security_id = pb.security_id 
			AND pb.price_date = $2
		LEFT JOIN security_prices pe ON s.security_id = pe.security_id 
			AND pe.price_date = $3
		WHERE h.portfolio_id = $1
		AND h.holding_date = $2
		ORDER BY h.weight DESC
	`

	rows, err := e.db.QueryContext(ctx, query, portfolioID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	holdings := []PortfolioHolding{}
	for rows.Next() {
		var h PortfolioHolding
		err := rows.Scan(
			&h.SecurityID, &h.SecurityName, &h.AssetClass, &h.Sector,
			&h.Industry, &h.Country, &h.Currency, &h.MarketValue,
			&h.Weight, &h.BeginPrice, &h.EndPrice,
		)
		if err != nil {
			return nil, err
		}

		// Calculate period return
		if h.BeginPrice > 0 {
			h.PeriodReturn = (h.EndPrice - h.BeginPrice) / h.BeginPrice
		}

		holdings = append(holdings, h)
	}

	return holdings, nil
}

func (e *AttributionEngine) getBenchmarkHoldings(ctx context.Context, benchmarkID string, startDate, endDate time.Time) ([]PortfolioHolding, error) {
	query := `
		SELECT 
			s.security_id,
			s.security_name,
			COALESCE(s.asset_class, 'Other') as asset_class,
			COALESCE(s.sector, 'Other') as sector,
			COALESCE(s.industry, 'Other') as industry,
			COALESCE(s.country, 'US') as country,
			COALESCE(s.currency, 'USD') as currency,
			b.market_value,
			b.weight,
			COALESCE(pb.price, 100) as begin_price,
			COALESCE(pe.price, pb.price, 100) as end_price
		FROM benchmark_constituents b
		JOIN securities s ON b.security_id = s.security_id
		LEFT JOIN security_prices pb ON s.security_id = pb.security_id 
			AND pb.price_date = $2
		LEFT JOIN security_prices pe ON s.security_id = pe.security_id 
			AND pe.price_date = $3
		WHERE b.benchmark_id = $1
		AND b.effective_date = $2
		ORDER BY b.weight DESC
	`

	rows, err := e.db.QueryContext(ctx, query, benchmarkID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	holdings := []PortfolioHolding{}
	for rows.Next() {
		var h PortfolioHolding
		err := rows.Scan(
			&h.SecurityID, &h.SecurityName, &h.AssetClass, &h.Sector,
			&h.Industry, &h.Country, &h.Currency, &h.MarketValue,
			&h.Weight, &h.BeginPrice, &h.EndPrice,
		)
		if err != nil {
			return nil, err
		}

		if h.BeginPrice > 0 {
			h.PeriodReturn = (h.EndPrice - h.BeginPrice) / h.BeginPrice
		}

		holdings = append(holdings, h)
	}

	return holdings, nil
}

func (e *AttributionEngine) getSecurityFactorExposures(ctx context.Context, holdings []PortfolioHolding, factors []string) (map[string]map[string]float64, error) {
	result := make(map[string]map[string]float64)

	// Query factor exposures for each security
	query := `
		SELECT security_id, factor_name, exposure_value
		FROM security_factor_exposures
		WHERE security_id = ANY($1)
		AND factor_name = ANY($2)
	`

	securityIDs := make([]string, len(holdings))
	for i, h := range holdings {
		securityIDs[i] = h.SecurityID
	}

	rows, err := e.db.QueryContext(ctx, query, securityIDs, factors)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var securityID, factorName string
		var exposure float64
		if err := rows.Scan(&securityID, &factorName, &exposure); err != nil {
			return nil, err
		}

		if _, ok := result[securityID]; !ok {
			result[securityID] = make(map[string]float64)
		}
		result[securityID][factorName] = exposure
	}

	return result, nil
}

func (e *AttributionEngine) calculateRSquared(holdings []PortfolioHolding, factorReturns map[string][]float64, exposures map[string]float64) float64 {
	// Simplified R-squared calculation
	// In practice, this would use regression analysis
	totalExplainedVar := 0.0
	for factor, exposure := range exposures {
		if returns, ok := factorReturns[factor]; ok && len(returns) > 1 {
			factorVar := calculateVariance(returns)
			totalExplainedVar += exposure * exposure * factorVar
		}
	}

	// Get portfolio returns variance
	portfolioReturns := make([]float64, len(holdings))
	for i, h := range holdings {
		portfolioReturns[i] = h.PeriodReturn
	}
	portfolioVar := calculateVariance(portfolioReturns)

	if portfolioVar > 0 {
		return math.Min(1.0, totalExplainedVar/portfolioVar)
	}
	return 0
}

// Helper functions
func (e *AttributionEngine) calculatePortfolioReturn(holdings []PortfolioHolding) float64 {
	totalReturn := 0.0
	totalWeight := 0.0

	for _, h := range holdings {
		totalReturn += h.Weight * h.PeriodReturn
		totalWeight += h.Weight
	}

	if totalWeight > 0 && totalWeight != 1.0 {
		totalReturn = totalReturn / totalWeight
	}

	return totalReturn
}

func (e *AttributionEngine) groupByLevel(holdings []PortfolioHolding, level string) map[string][]PortfolioHolding {
	result := make(map[string][]PortfolioHolding)

	for _, h := range holdings {
		var key string
		switch level {
		case "AssetClass":
			key = h.AssetClass
		case "Sector":
			key = h.Sector
		case "Industry":
			key = h.Industry
		case "Country":
			key = h.Country
		case "Currency":
			key = h.Currency
		default:
			key = "Other"
		}

		if key == "" {
			key = "Other"
		}

		result[key] = append(result[key], h)
	}

	return result
}

// Period represents a time period
type Period struct {
	Start time.Time
	End   time.Time
}

func generatePeriods(start, end time.Time, frequency string) []Period {
	periods := []Period{}
	current := start

	for current.Before(end) {
		var periodEnd time.Time

		switch frequency {
		case "daily":
			periodEnd = current.AddDate(0, 0, 1)
		case "weekly":
			periodEnd = current.AddDate(0, 0, 7)
		case "monthly":
			periodEnd = current.AddDate(0, 1, 0)
		case "quarterly":
			periodEnd = current.AddDate(0, 3, 0)
		default:
			periodEnd = current.AddDate(0, 1, 0) // Default to monthly
		}

		if periodEnd.After(end) {
			periodEnd = end
		}

		periods = append(periods, Period{Start: current, End: periodEnd})
		current = periodEnd
	}

	return periods
}

func groupByAssetClass(holdings []PortfolioHolding) map[string][]PortfolioHolding {
	result := make(map[string][]PortfolioHolding)
	for _, h := range holdings {
		class := h.AssetClass
		if class == "" {
			class = "Other"
		}
		result[class] = append(result[class], h)
	}
	return result
}

func sumWeights(holdings []PortfolioHolding) float64 {
	total := 0.0
	for _, h := range holdings {
		total += h.Weight
	}
	return total
}

func calculateGroupReturn(holdings []PortfolioHolding) float64 {
	if len(holdings) == 0 {
		return 0
	}

	totalWeightedReturn := 0.0
	totalWeight := sumWeights(holdings)

	if totalWeight == 0 {
		return 0
	}

	for _, h := range holdings {
		totalWeightedReturn += h.Weight * h.PeriodReturn
	}

	return totalWeightedReturn / totalWeight
}

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

func calculateVariance(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	mean := calculateMean(values)
	sumSquares := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}
	return sumSquares / float64(len(values)-1)
}

func calculateStdDev(values []float64) float64 {
	return math.Sqrt(calculateVariance(values))
}

func calculateBeta(portfolioReturns, benchmarkReturns []float64) float64 {
	if len(portfolioReturns) != len(benchmarkReturns) || len(portfolioReturns) < 2 {
		return 1.0
	}

	meanP := calculateMean(portfolioReturns)
	meanB := calculateMean(benchmarkReturns)

	covariance := 0.0
	varianceB := 0.0

	for i := range portfolioReturns {
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
