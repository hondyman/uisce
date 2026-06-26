package benchmark

import (
"context"
"fmt"
"time"
)

// Database operations

// validateBenchmark validates a benchmark definition
func (e *BenchmarkEngine) validateBenchmark(b Benchmark) error {
	if b.Name == "" {
		return fmt.Errorf("benchmark name is required")
	}
	if b.Type == "" {
		return fmt.Errorf("benchmark type is required")
	}
	if b.Type == BlendedBenchmark && len(b.Components) == 0 {
		return fmt.Errorf("blended benchmark requires at least one component")
	}
	return nil
}

// insertBenchmarkComponents inserts components for a blended benchmark
func (e *BenchmarkEngine) insertBenchmarkComponents(ctx context.Context, benchmarkID string, components []BenchmarkComponent) error {
	query := `
		INSERT INTO benchmark_components (
benchmark_id, component_benchmark_id, weight,
asset_class, sector, region,
effective_date, expiration_date
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	for _, c := range components {
		_, err := e.db.ExecContext(ctx, query,
benchmarkID, c.BenchmarkID, c.Weight,
c.AssetClass, c.Sector, c.Region,
c.EffectiveDate, c.ExpirationDate,
)
		if err != nil {
			return fmt.Errorf("failed to insert component: %w", err)
		}
	}

	return nil
}

// getBenchmarkComponents retrieves components for a blended benchmark
func (e *BenchmarkEngine) getBenchmarkComponents(ctx context.Context, benchmarkID string) ([]BenchmarkComponent, error) {
	query := `
		SELECT 
			bc.component_benchmark_id,
			b.name as benchmark_name,
			bc.weight,
			bc.asset_class,
			bc.sector,
			bc.region,
			bc.effective_date,
			bc.expiration_date
		FROM benchmark_components bc
		JOIN benchmarks b ON bc.component_benchmark_id = b.id
		WHERE bc.benchmark_id = $1
		AND (bc.expiration_date IS NULL OR bc.expiration_date > CURRENT_DATE)
		ORDER BY bc.weight DESC
	`

	rows, err := e.db.QueryContext(ctx, query, benchmarkID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	components := []BenchmarkComponent{}
	for rows.Next() {
		var c BenchmarkComponent
		err := rows.Scan(
&c.BenchmarkID, &c.BenchmarkName, &c.Weight,
			&c.AssetClass, &c.Sector, &c.Region,
			&c.EffectiveDate, &c.ExpirationDate,
		)
		if err != nil {
			return nil, err
		}
		components = append(components, c)
	}

	return components, nil
}

// getIndexReturn retrieves return for an index benchmark
func (e *BenchmarkEngine) getIndexReturn(ctx context.Context, benchmarkID string, startDate, endDate time.Time) (float64, error) {
	query := `
		SELECT COALESCE(
EXP(SUM(LN(1 + return_value))) - 1,
0
) as total_return
		FROM benchmark_returns
		WHERE benchmark_id = $1
		AND return_date BETWEEN $2 AND $3
	`

	var totalReturn float64
	err := e.db.QueryRowContext(ctx, query, benchmarkID, startDate, endDate).Scan(&totalReturn)
	if err != nil {
		return 0, err
	}

	return totalReturn, nil
}

// getBenchmarkReturns retrieves daily returns for a benchmark
func (e *BenchmarkEngine) getBenchmarkReturns(ctx context.Context, benchmarkID string, startDate, endDate time.Time) ([]float64, error) {
	query := `
		SELECT return_value
		FROM benchmark_returns
		WHERE benchmark_id = $1
		AND return_date BETWEEN $2 AND $3
		ORDER BY return_date ASC
	`

	rows, err := e.db.QueryContext(ctx, query, benchmarkID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	returns := []float64{}
	for rows.Next() {
		var r float64
		if err := rows.Scan(&r); err != nil {
			return nil, err
		}
		returns = append(returns, r)
	}

	return returns, nil
}

// getPortfolioReturns retrieves daily returns for a portfolio
func (e *BenchmarkEngine) getPortfolioReturns(ctx context.Context, portfolioID string, startDate, endDate time.Time) ([]float64, error) {
	query := `
		SELECT return_value
		FROM portfolio_returns
		WHERE portfolio_id = $1
		AND return_date BETWEEN $2 AND $3
		ORDER BY return_date ASC
	`

	rows, err := e.db.QueryContext(ctx, query, portfolioID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	returns := []float64{}
	for rows.Next() {
		var r float64
		if err := rows.Scan(&r); err != nil {
			return nil, err
		}
		returns = append(returns, r)
	}

	return returns, nil
}

// getBenchmarkHoldings retrieves holdings for a benchmark
func (e *BenchmarkEngine) getBenchmarkHoldings(ctx context.Context, benchmarkID string, asOfDate time.Time) ([]BenchmarkHolding, error) {
	query := `
		SELECT 
			benchmark_id,
			security_id,
			security_name,
			COALESCE(asset_class, 'Other') as asset_class,
			COALESCE(sector, 'Other') as sector,
			COALESCE(country, 'US') as country,
			COALESCE(currency, 'USD') as currency,
			weight,
			COALESCE(market_cap, 0) as market_cap,
			effective_date
		FROM benchmark_holdings
		WHERE benchmark_id = $1
		AND effective_date <= $2
		AND (expiration_date IS NULL OR expiration_date > $2)
		ORDER BY weight DESC
	`

	rows, err := e.db.QueryContext(ctx, query, benchmarkID, asOfDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	holdings := []BenchmarkHolding{}
	for rows.Next() {
		var h BenchmarkHolding
		err := rows.Scan(
&h.BenchmarkID, &h.SecurityID, &h.SecurityName,
			&h.AssetClass, &h.Sector, &h.Country, &h.Currency,
			&h.Weight, &h.MarketCap, &h.EffectiveDate,
		)
		if err != nil {
			return nil, err
		}
		holdings = append(holdings, h)
	}

	return holdings, nil
}

// getPortfolioHoldings retrieves holdings for a portfolio
func (e *BenchmarkEngine) getPortfolioHoldings(ctx context.Context, portfolioID string, asOfDate time.Time) ([]BenchmarkHolding, error) {
	query := `
		SELECT 
			h.portfolio_id,
			h.security_id,
			s.security_name,
			COALESCE(s.asset_class, 'Other') as asset_class,
			COALESCE(s.sector, 'Other') as sector,
			COALESCE(s.country, 'US') as country,
			COALESCE(s.currency, 'USD') as currency,
			h.weight,
			COALESCE(s.market_cap, 0) as market_cap,
			h.holding_date
		FROM portfolio_holdings h
		JOIN securities s ON h.security_id = s.security_id
		WHERE h.portfolio_id = $1
		AND h.holding_date = $2
		ORDER BY h.weight DESC
	`

	rows, err := e.db.QueryContext(ctx, query, portfolioID, asOfDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	holdings := []BenchmarkHolding{}
	for rows.Next() {
		var h BenchmarkHolding
		err := rows.Scan(
&h.BenchmarkID, &h.SecurityID, &h.SecurityName,
			&h.AssetClass, &h.Sector, &h.Country, &h.Currency,
			&h.Weight, &h.MarketCap, &h.EffectiveDate,
		)
		if err != nil {
			return nil, err
		}
		holdings = append(holdings, h)
	}

	return holdings, nil
}

// UpdateBenchmark updates an existing benchmark
func (e *BenchmarkEngine) UpdateBenchmark(ctx context.Context, benchmark Benchmark) error {
	query := `
		UPDATE benchmarks SET
			name = $2,
			description = $3,
			is_active = $4,
			rebalance_frequency = $5,
			target_return = $6,
			updated_at = $7
		WHERE id = $1
	`

	_, err := e.db.ExecContext(ctx, query,
benchmark.ID, benchmark.Name, benchmark.Description,
benchmark.IsActive, benchmark.RebalanceFreq, benchmark.TargetReturn,
time.Now(),
	)

	return err
}

// DeleteBenchmark soft-deletes a benchmark
func (e *BenchmarkEngine) DeleteBenchmark(ctx context.Context, benchmarkID string) error {
	query := `UPDATE benchmarks SET is_active = false, updated_at = $2 WHERE id = $1`
	_, err := e.db.ExecContext(ctx, query, benchmarkID, time.Now())
	return err
}

// StoreBenchmarkReturn stores a daily return for a benchmark
func (e *BenchmarkEngine) StoreBenchmarkReturn(ctx context.Context, benchmarkID string, date time.Time, returnValue float64, returnType ReturnType) error {
	query := `
		INSERT INTO benchmark_returns (benchmark_id, return_date, return_value, return_type)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (benchmark_id, return_date, return_type)
		DO UPDATE SET return_value = EXCLUDED.return_value
	`

	_, err := e.db.ExecContext(ctx, query, benchmarkID, date, returnValue, returnType)
	return err
}

// StoreBenchmarkHolding stores a holding for a benchmark
func (e *BenchmarkEngine) StoreBenchmarkHolding(ctx context.Context, holding BenchmarkHolding) error {
	query := `
		INSERT INTO benchmark_holdings (
benchmark_id, security_id, security_name,
asset_class, sector, country, currency,
weight, market_cap, effective_date
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (benchmark_id, security_id, effective_date)
		DO UPDATE SET weight = EXCLUDED.weight, market_cap = EXCLUDED.market_cap
	`

	_, err := e.db.ExecContext(ctx, query,
holding.BenchmarkID, holding.SecurityID, holding.SecurityName,
holding.AssetClass, holding.Sector, holding.Country, holding.Currency,
holding.Weight, holding.MarketCap, holding.EffectiveDate,
)

	return err
}
