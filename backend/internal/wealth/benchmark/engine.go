package benchmark

import (
"context"
"database/sql"
"fmt"
"math"
"sort"
"time"
)

// BenchmarkEngine provides benchmark creation and comparison capabilities
type BenchmarkEngine struct {
	db *sql.DB
}

// NewBenchmarkEngine creates a new benchmark engine
func NewBenchmarkEngine(db *sql.DB) *BenchmarkEngine {
	return &BenchmarkEngine{db: db}
}

// CreateBenchmark creates a new benchmark definition
func (e *BenchmarkEngine) CreateBenchmark(ctx context.Context, benchmark Benchmark) (*Benchmark, error) {
	// Validate benchmark
	if err := e.validateBenchmark(benchmark); err != nil {
		return nil, err
	}

	// Insert benchmark
	query := `
		INSERT INTO benchmarks (
id, name, description, benchmark_type, currency,
inception_date, is_active, rebalance_frequency,
target_return, created_at, updated_at, created_by
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`

	now := time.Now()
	benchmark.CreatedAt = now
	benchmark.UpdatedAt = now

	err := e.db.QueryRowContext(ctx, query,
benchmark.ID, benchmark.Name, benchmark.Description,
benchmark.Type, benchmark.Currency, benchmark.InceptionDate,
benchmark.IsActive, benchmark.RebalanceFreq, benchmark.TargetReturn,
benchmark.CreatedAt, benchmark.UpdatedAt, benchmark.CreatedBy,
).Scan(&benchmark.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to create benchmark: %w", err)
	}

	// Insert components for blended benchmarks
	if len(benchmark.Components) > 0 {
		if err := e.insertBenchmarkComponents(ctx, benchmark.ID, benchmark.Components); err != nil {
			return nil, err
		}
	}

	return &benchmark, nil
}

// CreateBlendedBenchmark creates a blended benchmark from multiple components
func (e *BenchmarkEngine) CreateBlendedBenchmark(ctx context.Context, name string, components []BenchmarkComponent, rebalanceFreq RebalanceFrequency) (*Benchmark, error) {
	// Validate weights sum to 1
	totalWeight := 0.0
	for _, c := range components {
		totalWeight += c.Weight
	}
	if math.Abs(totalWeight-1.0) > 0.001 {
		return nil, fmt.Errorf("component weights must sum to 1.0, got %f", totalWeight)
	}

	benchmark := Benchmark{
		Name:          name,
		Type:          BlendedBenchmark,
		Currency:      "USD",
		InceptionDate: time.Now(),
		IsActive:      true,
		Components:    components,
		RebalanceFreq: rebalanceFreq,
	}

	return e.CreateBenchmark(ctx, benchmark)
}

// CreateCustomBenchmark creates a custom rules-based benchmark
func (e *BenchmarkEngine) CreateCustomBenchmark(ctx context.Context, name string, rules CustomBenchmarkRules) (*Benchmark, error) {
	benchmark := Benchmark{
		Name:          name,
		Type:          CustomBenchmark,
		Currency:      "USD",
		InceptionDate: time.Now(),
		IsActive:      true,
		CustomRules:   &rules,
	}

	return e.CreateBenchmark(ctx, benchmark)
}

// GetBenchmark retrieves a benchmark by ID
func (e *BenchmarkEngine) GetBenchmark(ctx context.Context, benchmarkID string) (*Benchmark, error) {
	query := `
		SELECT 
			id, name, description, benchmark_type, currency,
			inception_date, is_active, rebalance_frequency,
			target_return, created_at, updated_at, created_by
		FROM benchmarks
		WHERE id = $1
	`

	var b Benchmark
	err := e.db.QueryRowContext(ctx, query, benchmarkID).Scan(
&b.ID, &b.Name, &b.Description, &b.Type, &b.Currency,
		&b.InceptionDate, &b.IsActive, &b.RebalanceFreq,
		&b.TargetReturn, &b.CreatedAt, &b.UpdatedAt, &b.CreatedBy,
	)
	if err != nil {
		return nil, err
	}

	// Load components
	b.Components, _ = e.getBenchmarkComponents(ctx, benchmarkID)

	return &b, nil
}

// ListBenchmarks returns all benchmarks
func (e *BenchmarkEngine) ListBenchmarks(ctx context.Context, activeOnly bool) ([]Benchmark, error) {
	query := `
		SELECT 
			id, name, description, benchmark_type, currency,
			inception_date, is_active, rebalance_frequency,
			target_return, created_at, updated_at, created_by
		FROM benchmarks
	`
	if activeOnly {
		query += " WHERE is_active = true"
	}
	query += " ORDER BY name"

	rows, err := e.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	benchmarks := []Benchmark{}
	for rows.Next() {
		var b Benchmark
		err := rows.Scan(
&b.ID, &b.Name, &b.Description, &b.Type, &b.Currency,
			&b.InceptionDate, &b.IsActive, &b.RebalanceFreq,
			&b.TargetReturn, &b.CreatedAt, &b.UpdatedAt, &b.CreatedBy,
		)
		if err != nil {
			return nil, err
		}
		benchmarks = append(benchmarks, b)
	}

	return benchmarks, nil
}

// CalculateBenchmarkReturn calculates benchmark return for a period
func (e *BenchmarkEngine) CalculateBenchmarkReturn(ctx context.Context, benchmarkID string, startDate, endDate time.Time) (*BenchmarkReturn, error) {
	benchmark, err := e.GetBenchmark(ctx, benchmarkID)
	if err != nil {
		return nil, err
	}

	var totalReturn float64

	switch benchmark.Type {
	case BlendedBenchmark:
		totalReturn, err = e.calculateBlendedReturn(ctx, benchmark, startDate, endDate)
	case AbsoluteReturnBenchmark:
		days := endDate.Sub(startDate).Hours() / 24
		totalReturn = benchmark.TargetReturn * days / 365
	default:
		totalReturn, err = e.getIndexReturn(ctx, benchmarkID, startDate, endDate)
	}

	if err != nil {
		return nil, err
	}

	// Calculate annualized return
	years := endDate.Sub(startDate).Hours() / 24 / 365
	annualizedReturn := 0.0
	if years > 0 {
		annualizedReturn = math.Pow(1+totalReturn, 1/years) - 1
	}

	return &BenchmarkReturn{
		BenchmarkID:      benchmarkID,
		Date:             endDate,
		ReturnValue:      totalReturn,
		ReturnType:       TotalReturn,
		AnnualizedReturn: annualizedReturn,
	}, nil
}

// calculateBlendedReturn calculates return for a blended benchmark
func (e *BenchmarkEngine) calculateBlendedReturn(ctx context.Context, benchmark *Benchmark, startDate, endDate time.Time) (float64, error) {
	totalReturn := 0.0

	for _, component := range benchmark.Components {
		// Get component return
		componentReturn, err := e.getIndexReturn(ctx, component.BenchmarkID, startDate, endDate)
		if err != nil {
			continue // Skip unavailable components
		}

		// Weight the return
		totalReturn += component.Weight * componentReturn
	}

	return totalReturn, nil
}

// CompareToPortfolio compares a portfolio to a benchmark
func (e *BenchmarkEngine) CompareToPortfolio(ctx context.Context, portfolioID, benchmarkID string, startDate, endDate time.Time) (*BenchmarkComparison, error) {
	comparison := &BenchmarkComparison{
		PortfolioID:     portfolioID,
		BenchmarkID:     benchmarkID,
		AsOfDate:        endDate,
		SectorDeviation: make(map[string]float64),
	}

	// Get benchmark info
	benchmark, err := e.GetBenchmark(ctx, benchmarkID)
	if err != nil {
		return nil, err
	}
	comparison.BenchmarkName = benchmark.Name

	// Get returns
	portfolioReturns, err := e.getPortfolioReturns(ctx, portfolioID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	benchmarkReturns, err := e.getBenchmarkReturns(ctx, benchmarkID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Calculate period returns
	comparison.PortfolioReturn = e.calculateTotalReturn(portfolioReturns)
	comparison.BenchmarkReturn = e.calculateTotalReturn(benchmarkReturns)
	comparison.ActiveReturn = comparison.PortfolioReturn - comparison.BenchmarkReturn

	// Calculate risk metrics
	comparison.PortfolioVol = e.calculateVolatility(portfolioReturns) * math.Sqrt(252)
	comparison.BenchmarkVol = e.calculateVolatility(benchmarkReturns) * math.Sqrt(252)
	comparison.TrackingError = e.calculateTrackingError(portfolioReturns, benchmarkReturns) * math.Sqrt(252)

	// Information ratio
	if comparison.TrackingError > 0 {
		annualizedActive := comparison.ActiveReturn * 252 / float64(len(portfolioReturns))
		comparison.InformationRatio = annualizedActive / comparison.TrackingError
	}

	// Beta and Alpha
	comparison.Beta = e.calculateBeta(portfolioReturns, benchmarkReturns)
	riskFreeRate := 0.02 / 252 // Daily risk-free rate
	comparison.Alpha = e.calculateAlpha(portfolioReturns, benchmarkReturns, riskFreeRate, comparison.Beta)
	comparison.RSqaured = e.calculateRSquared(portfolioReturns, benchmarkReturns)

	// Period comparisons
	comparison.ReturnPeriods = e.calculatePeriodComparisons(ctx, portfolioID, benchmarkID, endDate)

	// Active weights
	comparison.ActiveWeights = e.calculateActiveWeights(ctx, portfolioID, benchmarkID, endDate)

	// Sector deviations
	comparison.SectorDeviation = e.calculateSectorDeviations(comparison.ActiveWeights)

	return comparison, nil
}

// GetBenchmarkStatistics calculates comprehensive benchmark statistics
func (e *BenchmarkEngine) GetBenchmarkStatistics(ctx context.Context, benchmarkID string, asOfDate time.Time) (*BenchmarkStatistics, error) {
	stats := &BenchmarkStatistics{
		BenchmarkID:    benchmarkID,
		AsOfDate:       asOfDate,
		SectorWeights:  make(map[string]float64),
		CountryWeights: make(map[string]float64),
		CurrencyWeights: make(map[string]float64),
	}

	// Get historical returns
	startDate := asOfDate.AddDate(-10, 0, 0)
	returns, err := e.getBenchmarkReturns(ctx, benchmarkID, startDate, asOfDate)
	if err != nil {
		return nil, err
	}

	// Calculate period returns
	stats.DailyReturn = e.calculatePeriodReturn(returns, 1)
	stats.MTDReturn = e.calculateMTDReturn(returns, asOfDate)
	stats.QTDReturn = e.calculateQTDReturn(returns, asOfDate)
	stats.YTDReturn = e.calculateYTDReturn(returns, asOfDate)
	stats.OneYearReturn = e.calculatePeriodReturn(returns, 252)
	stats.ThreeYearReturn = e.annualizeReturn(e.calculatePeriodReturn(returns, 756), 3)
	stats.FiveYearReturn = e.annualizeReturn(e.calculatePeriodReturn(returns, 1260), 5)
	stats.TenYearReturn = e.annualizeReturn(e.calculatePeriodReturn(returns, 2520), 10)

	// Risk metrics
	stats.Volatility = e.calculateVolatility(returns) * math.Sqrt(252)
	riskFreeRate := 0.02
	if stats.Volatility > 0 {
		stats.SharpeRatio = (stats.OneYearReturn - riskFreeRate) / stats.Volatility
	}
	stats.MaxDrawdown = e.calculateMaxDrawdown(returns)
	if stats.MaxDrawdown != 0 {
		stats.CalmarRatio = stats.OneYearReturn / math.Abs(stats.MaxDrawdown)
	}

	// Get holdings and characteristics
	holdings, err := e.getBenchmarkHoldings(ctx, benchmarkID, asOfDate)
	if err == nil {
		stats.NumberOfHoldings = len(holdings)

		// Calculate characteristics
		totalMarketCap := 0.0
		marketCaps := make([]float64, 0)

		for _, h := range holdings {
			stats.SectorWeights[h.Sector] += h.Weight
			stats.CountryWeights[h.Country] += h.Weight
			stats.CurrencyWeights[h.Currency] += h.Weight
			totalMarketCap += h.Weight * h.MarketCap
			marketCaps = append(marketCaps, h.MarketCap)
		}

		stats.AverageMarketCap = totalMarketCap
		if len(marketCaps) > 0 {
			sort.Float64s(marketCaps)
			stats.MedianMarketCap = marketCaps[len(marketCaps)/2]
		}
	}

	return stats, nil
}

// GetBenchmarkHoldings returns the holdings of a benchmark
func (e *BenchmarkEngine) GetBenchmarkHoldings(ctx context.Context, benchmarkID string, asOfDate time.Time) ([]BenchmarkHolding, error) {
	return e.getBenchmarkHoldings(ctx, benchmarkID, asOfDate)
}
