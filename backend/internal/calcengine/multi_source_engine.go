package calcengine

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/hondyman/uisce/backend/internal/pricing"
	"github.com/hondyman/uisce/backend/internal/wealth/risk"
	"github.com/jmoiron/sqlx"
)

// DataTier indicates which data tier to query
type DataTier string

const (
	// HotTier uses StarRocks for real-time analytics (< 90 days default)
	HotTier DataTier = "hot"
	// ColdTier is deprecated - Trino/Iceberg removed
	ColdTier DataTier = "cold"
	// AutoTier automatically routes based on date range
	AutoTier DataTier = "auto"
)

// MultiSourceConfig configures the multi-source calc engine
type MultiSourceConfig struct {
	// StarRocks configuration for hot tier
	StarRocks *StarRocksConfig `yaml:"starrocks" json:"starrocks"`

	// PostgreSQL for metadata/catalog (existing)
	PostgresDSN string `yaml:"postgres_dsn" json:"postgres_dsn"`

	// Hot/Cold boundary in days (default: 90)
	HotColdBoundaryDays int `yaml:"hot_cold_boundary_days" json:"hot_cold_boundary_days"`
}

// MultiSourceCalcEngine routes queries to StarRocks (hot) or cold tier
// Deprecated: Cold tier (Trino/Iceberg) has been removed
type MultiSourceCalcEngine struct {
	// Hot tier: StarRocks for real-time analytics
	starrocks *StarRocksClient

	// Metadata tier: PostgreSQL for catalog, DAG, config
	postgres        *sqlx.DB
	postgresRaw     *sql.DB
	pricingProvider pricing.PricingProvider
	riskEngine      *risk.RiskAnalyticsEngine

	// Configuration
	config *MultiSourceConfig
}

// NewMultiSourceCalcEngine creates a new multi-source calculation engine
func NewMultiSourceCalcEngine(cfg *MultiSourceConfig, pricingProvider pricing.PricingProvider) (*MultiSourceCalcEngine, error) {
	engine := &MultiSourceCalcEngine{
		config:          cfg,
		pricingProvider: pricingProvider,
	}

	// Initialize hot tier (StarRocks)
	if cfg.StarRocks != nil {
		sr, err := NewStarRocksClient(cfg.StarRocks)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize StarRocks: %w", err)
		}
		engine.starrocks = sr
	}

	// Cold tier (Trino) has been removed

	// Initialize metadata tier (PostgreSQL)
	if cfg.PostgresDSN != "" {
		db, err := sqlx.Connect("postgres", cfg.PostgresDSN)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize PostgreSQL: %w", err)
		}
		engine.postgres = db
		engine.postgresRaw = db.DB
		engine.riskEngine = risk.NewRiskAnalyticsEngine(db.DB)
	}

	// Set defaults
	if cfg.HotColdBoundaryDays == 0 {
		cfg.HotColdBoundaryDays = 90
	}

	return engine, nil
}

// Run executes a calculation, routing to appropriate tier
func (e *MultiSourceCalcEngine) Run(ctx context.Context, metric string, inputs map[string]interface{}) (*CalcResult, error) {
	// Determine data tier
	tier := e.determineTier(inputs)

	switch metric {
	case "NAV":
		return e.calculateNAV(ctx, inputs, tier)
	case "VaR":
		return e.calculateVaR(ctx, inputs)
	case "PoP":
		return e.calculatePoP(ctx, inputs, tier)
	case "Anomaly":
		return e.detectAnomalies(ctx, inputs, tier)
	case "TimeSeries":
		return e.queryTimeSeries(ctx, inputs, tier)
	default:
		return nil, fmt.Errorf("unsupported metric: %s", metric)
	}
}

// RunFast executes a query directly on StarRocks for maximum speed
func (e *MultiSourceCalcEngine) RunFast(ctx context.Context, metric string, inputs map[string]interface{}) (*CalcResult, error) {
	if e.starrocks == nil {
		return nil, fmt.Errorf("StarRocks not configured for fast path")
	}

	// Force hot tier for speed
	return e.Run(ctx, metric, withTier(inputs, HotTier))
}

// RunHistorical executes a query against cold tier for historical analysis
// Deprecated: Trino/Iceberg has been removed
func (e *MultiSourceCalcEngine) RunHistorical(ctx context.Context, metric string, inputs map[string]interface{}) (*CalcResult, error) {
	return nil, fmt.Errorf("historical queries disabled: Trino/Iceberg audit chain removed")
}

// determineTier decides which data tier to query based on inputs
func (e *MultiSourceCalcEngine) determineTier(inputs map[string]interface{}) DataTier {
	// Check explicit tier preference
	if tier, ok := inputs["data_tier"].(DataTier); ok {
		return tier
	}
	if tierStr, ok := inputs["data_tier"].(string); ok {
		switch tierStr {
		case "hot":
			return HotTier
		case "cold":
			return ColdTier
		}
	}

	// Auto-detect based on date range
	boundary := time.Now().AddDate(0, 0, -e.config.HotColdBoundaryDays)

	// Check as_of_date
	if asOf, ok := inputs["as_of_date"].(time.Time); ok {
		if asOf.After(boundary) {
			return HotTier
		}
		return ColdTier
	}

	// Check date range
	if startDate, ok := inputs["start_date"].(time.Time); ok {
		if startDate.After(boundary) {
			return HotTier
		}
		return ColdTier
	}

	// Default to hot tier for real-time
	return HotTier
}

// calculateNAV computes NAV using the appropriate tier
func (e *MultiSourceCalcEngine) calculateNAV(ctx context.Context, inputs map[string]interface{}, tier DataTier) (*CalcResult, error) {
	tenantID, ok := inputs["tenant_id"].(string)
	if !ok {
		return nil, fmt.Errorf("tenant_id required")
	}

	datasourceID, _ := inputs["datasource_id"].(string)
	if datasourceID == "" {
		datasourceID = "default"
	}

	portfolioID, ok := inputs["portfolio_id"].(string)
	if !ok {
		return nil, fmt.Errorf("portfolio_id required")
	}

	asOfDate := time.Now()
	if d, ok := inputs["as_of_date"].(time.Time); ok {
		asOfDate = d
	}

	var holdings []HoldingData
	var err error

	switch tier {
	case HotTier:
		holdings, err = e.fetchHoldingsFromStarRocks(ctx, tenantID, datasourceID, portfolioID, asOfDate)
	case ColdTier:
		return nil, fmt.Errorf("cold tier disabled: Trino/Iceberg removed")
	default:
		// Auto: use StarRocks only
		holdings, err = e.fetchHoldingsFromStarRocks(ctx, tenantID, datasourceID, portfolioID, asOfDate)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to fetch holdings: %w", err)
	}

	// Calculate NAV
	return e.computeNAVFromHoldings(ctx, holdings, inputs)
}

// HoldingData represents a portfolio holding
type HoldingData struct {
	HoldingID string    `json:"holding_id"`
	Ticker    string    `json:"ticker"`
	Quantity  float64   `json:"quantity"`
	Currency  string    `json:"currency"`
	CostBasis float64   `json:"cost_basis"`
	AsOfDate  time.Time `json:"as_of_date"`
}

// fetchHoldingsFromStarRocks queries hot tier for holdings
func (e *MultiSourceCalcEngine) fetchHoldingsFromStarRocks(ctx context.Context,
	tenantID, datasourceID, portfolioID string, asOfDate time.Time) ([]HoldingData, error) {

	if e.starrocks == nil {
		return nil, fmt.Errorf("StarRocks not configured")
	}

	// Set resource group for QoS
	resourceGroup := fmt.Sprintf("tenant_%s_%s", tenantID, datasourceID)
	_ = e.starrocks.SetResourceGroup(ctx, resourceGroup)

	query := `
		SELECT 
			holding_id,
			ticker,
			quantity,
			currency,
			cost_basis,
			as_of_date
		FROM holdings
		WHERE tenant_id = ?
		  AND datasource_id = ?
		  AND portfolio_id = ?
		  AND as_of_date <= ?
		ORDER BY as_of_date DESC
	`

	rows, err := e.starrocks.Query(ctx, query, tenantID, datasourceID, portfolioID, asOfDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var holdings []HoldingData
	for rows.Next() {
		var h HoldingData
		if err := rows.Scan(&h.HoldingID, &h.Ticker, &h.Quantity, &h.Currency, &h.CostBasis, &h.AsOfDate); err != nil {
			return nil, err
		}
		holdings = append(holdings, h)
	}

	return holdings, rows.Err()
}

// fetchHoldingsFromTrino queries cold tier for historical holdings
// Deprecated: Trino/Iceberg has been removed
func (e *MultiSourceCalcEngine) fetchHoldingsFromTrino(ctx context.Context,
	tenantID, datasourceID, portfolioID string, asOfDate time.Time) ([]HoldingData, error) {
	return nil, fmt.Errorf("Trino queries disabled: Trino/Iceberg audit chain removed")
}

// computeNAVFromHoldings calculates NAV from holding data
func (e *MultiSourceCalcEngine) computeNAVFromHoldings(ctx context.Context,
	holdings []HoldingData, inputs map[string]interface{}) (*CalcResult, error) {

	var navValue float64
	var breakdown []map[string]interface{}
	var sources []string

	for _, holding := range holdings {
		// Get price
		var price float64
		var err error

		if priceVal, ok := inputs[holding.Ticker+"_price"]; ok {
			price, _ = priceVal.(float64)
		} else if e.pricingProvider != nil {
			price, err = e.pricingProvider.GetPrice(ctx, holding.Ticker)
			if err != nil {
				continue // Skip if price unavailable
			}
		}

		// Get FX rate
		fxRate := 1.0
		if holding.Currency != "USD" {
			fxPair := holding.Currency + "USD"
			if fxVal, ok := inputs[fxPair]; ok {
				fxRate, _ = fxVal.(float64)
			} else if e.pricingProvider != nil {
				fxRate, _ = e.pricingProvider.GetFXRate(ctx, fxPair)
			}
		}

		// Calculate position value
		positionValue := holding.Quantity * price * fxRate
		navValue += positionValue

		breakdown = append(breakdown, map[string]interface{}{
			"holding":        holding.HoldingID,
			"ticker":         holding.Ticker,
			"quantity":       holding.Quantity,
			"price":          price,
			"currency":       holding.Currency,
			"fx_rate":        fxRate,
			"value_usd":      positionValue,
			"cost_basis":     holding.CostBasis,
			"unrealized_pnl": positionValue - holding.CostBasis,
		})

		sources = append(sources, holding.Ticker)
	}

	return &CalcResult{
		Metric:    "NAV",
		Value:     navValue,
		Sources:   sources,
		Breakdown: breakdown,
	}, nil
}

// calculateVaR uses the risk analytics engine (unchanged)
func (e *MultiSourceCalcEngine) calculateVaR(ctx context.Context, inputs map[string]interface{}) (*CalcResult, error) {
	if e.riskEngine == nil {
		return nil, fmt.Errorf("risk engine not configured")
	}

	portfolioID, ok := inputs["portfolio_id"].(string)
	if !ok {
		return nil, fmt.Errorf("portfolio_id required")
	}

	confidenceLevel := 0.95
	if cl, ok := inputs["confidence_level"].(float64); ok {
		confidenceLevel = cl
	}

	horizon := 1
	if h, ok := inputs["horizon"].(int); ok {
		horizon = h
	}

	method := risk.HistoricalVaR
	if m, ok := inputs["method"].(string); ok {
		switch m {
		case "parametric":
			method = risk.ParametricVaR
		case "monte_carlo":
			method = risk.MonteCarloVaR
		case "cornish_fisher":
			method = risk.CornishFisherVaR
		}
	}

	config := risk.RiskConfig{
		PortfolioID:      portfolioID,
		AsOfDate:         time.Now(),
		ConfidenceLevels: []float64{confidenceLevel},
		Horizon:          horizon,
		Method:           method,
		SimulationCount:  10000,
		HistoricalPeriod: 252,
		Currency:         "USD",
	}

	result, err := e.riskEngine.Calculate(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("risk calculation failed: %w", err)
	}

	varResult, ok := result.VaRResults[confidenceLevel]
	if !ok {
		return nil, fmt.Errorf("no VaR result for confidence level %.2f", confidenceLevel)
	}

	breakdown := []map[string]interface{}{
		{
			"metric":           "VaR",
			"confidence_level": confidenceLevel,
			"horizon_days":     horizon,
			"method":           string(method),
			"var_absolute":     varResult.VaRAbsolute,
			"var_relative":     varResult.VaRRelative,
			"portfolio_value":  result.PortfolioValue,
		},
	}

	return &CalcResult{
		Metric:    "VaR",
		Value:     varResult.VaRAbsolute,
		Sources:   []string{"portfolio_positions", "market_data", "historical_returns"},
		Breakdown: breakdown,
	}, nil
}

// calculatePoP computes period-over-period metrics
func (e *MultiSourceCalcEngine) calculatePoP(ctx context.Context, inputs map[string]interface{}, tier DataTier) (*CalcResult, error) {
	tenantID, _ := inputs["tenant_id"].(string)
	datasourceID, _ := inputs["datasource_id"].(string)
	metricID, _ := inputs["metric_id"].(string)
	periodLabel, _ := inputs["period_label"].(string)

	if tenantID == "" || metricID == "" || periodLabel == "" {
		return nil, fmt.Errorf("tenant_id, metric_id, and period_label required")
	}

	var query string
	var rows interface {
		Close() error
		Next() bool
		Scan(dest ...interface{}) error
		Err() error
	}
	var err error

	popQuery := `
		SELECT 
			metric_id,
			current_value,
			previous_value,
			delta,
			percent_change,
			period_label
		FROM metrics_pop
		WHERE tenant_id = ?
		  AND datasource_id = ?
		  AND metric_id = ?
		  AND period_label = ?
	`

	switch tier {
	case HotTier:
		if e.starrocks == nil {
			return nil, fmt.Errorf("StarRocks not configured")
		}
		query = popQuery
		rows, err = e.starrocks.Query(ctx, query, tenantID, datasourceID, metricID, periodLabel)
	case ColdTier:
		return nil, fmt.Errorf("cold tier disabled: Trino/Iceberg removed")
	default:
		return nil, fmt.Errorf("invalid tier")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query PoP: %w", err)
	}
	defer rows.Close()

	var result struct {
		MetricID      string
		CurrentValue  float64
		PreviousValue float64
		Delta         float64
		PercentChange float64
		PeriodLabel   string
	}

	if rows.Next() {
		if err := rows.Scan(&result.MetricID, &result.CurrentValue, &result.PreviousValue,
			&result.Delta, &result.PercentChange, &result.PeriodLabel); err != nil {
			return nil, err
		}
	}

	return &CalcResult{
		Metric:  "PoP",
		Value:   result.PercentChange,
		Sources: []string{metricID},
		Breakdown: []map[string]interface{}{
			{
				"metric_id":      result.MetricID,
				"current_value":  result.CurrentValue,
				"previous_value": result.PreviousValue,
				"delta":          result.Delta,
				"percent_change": result.PercentChange,
				"period_label":   result.PeriodLabel,
			},
		},
	}, nil
}

// detectAnomalies runs z-score anomaly detection
func (e *MultiSourceCalcEngine) detectAnomalies(ctx context.Context, inputs map[string]interface{}, tier DataTier) (*CalcResult, error) {
	tenantID, _ := inputs["tenant_id"].(string)
	datasourceID, _ := inputs["datasource_id"].(string)
	metricID, _ := inputs["metric_id"].(string)
	threshold := 3.0 // Default z-score threshold
	if t, ok := inputs["threshold"].(float64); ok {
		threshold = t
	}

	if tenantID == "" || metricID == "" {
		return nil, fmt.Errorf("tenant_id and metric_id required")
	}

	var query string
	if tier == HotTier && e.starrocks != nil {
		query = fmt.Sprintf(`
			SELECT 
				as_of_date,
				metric_value,
				z_score,
				is_anomaly
			FROM metrics_anomalies
			WHERE tenant_id = '%s'
			  AND datasource_id = '%s'
			  AND metric_id = '%s'
			  AND ABS(z_score) >= %f
			ORDER BY as_of_date DESC
			LIMIT 100
		`, tenantID, datasourceID, metricID, threshold)

		rows, err := e.starrocks.Query(ctx, query)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		return e.scanAnomalies(rows, metricID)
	}

	return nil, fmt.Errorf("no data tier available for anomaly detection")
}

func (e *MultiSourceCalcEngine) scanAnomalies(rows interface {
	Next() bool
	Scan(dest ...interface{}) error
	Err() error
}, metricID string) (*CalcResult, error) {
	var anomalies []map[string]interface{}
	var anomalyCount int

	for rows.Next() {
		var asOfDate time.Time
		var value, zScore float64
		var isAnomaly bool

		if err := rows.Scan(&asOfDate, &value, &zScore, &isAnomaly); err != nil {
			return nil, err
		}

		if isAnomaly {
			anomalyCount++
		}

		anomalies = append(anomalies, map[string]interface{}{
			"as_of_date": asOfDate,
			"value":      value,
			"z_score":    zScore,
			"is_anomaly": isAnomaly,
		})
	}

	return &CalcResult{
		Metric:    "Anomaly",
		Value:     float64(anomalyCount),
		Sources:   []string{metricID},
		Breakdown: anomalies,
	}, rows.Err()
}

// queryTimeSeries executes optimized time series queries
func (e *MultiSourceCalcEngine) queryTimeSeries(ctx context.Context, inputs map[string]interface{}, tier DataTier) (*CalcResult, error) {
	tenantID, _ := inputs["tenant_id"].(string)
	datasourceID, _ := inputs["datasource_id"].(string)
	table, _ := inputs["table"].(string)
	valueColumn, _ := inputs["value_column"].(string)
	timeColumn, _ := inputs["time_column"].(string)
	startDate, _ := inputs["start_date"].(time.Time)
	endDate, _ := inputs["end_date"].(time.Time)
	granularity, _ := inputs["granularity"].(string)

	if tenantID == "" || table == "" || valueColumn == "" {
		return nil, fmt.Errorf("tenant_id, table, and value_column required")
	}

	if timeColumn == "" {
		timeColumn = "as_of_date"
	}
	if granularity == "" {
		granularity = "DAY"
	}
	if startDate.IsZero() {
		startDate = time.Now().AddDate(0, -1, 0)
	}
	if endDate.IsZero() {
		endDate = time.Now()
	}

	if tier == HotTier && e.starrocks != nil {
		points, err := e.starrocks.QueryTimeSeries(ctx, tenantID, datasourceID,
			table, valueColumn, timeColumn, startDate, endDate, granularity)
		if err != nil {
			return nil, err
		}

		var breakdown []map[string]interface{}
		var totalValue float64
		for _, p := range points {
			totalValue += p.Value
			breakdown = append(breakdown, map[string]interface{}{
				"timestamp": p.Timestamp,
				"value":     p.Value,
				"count":     p.Count,
			})
		}

		return &CalcResult{
			Metric:    "TimeSeries",
			Value:     totalValue,
			Sources:   []string{table},
			Breakdown: breakdown,
		}, nil
	}

	return nil, fmt.Errorf("time series query requires StarRocks for optimal performance")
}

// Close closes all data tier connections
func (e *MultiSourceCalcEngine) Close() error {
	var errs []error

	if e.starrocks != nil {
		if err := e.starrocks.Close(); err != nil {
			errs = append(errs, fmt.Errorf("StarRocks close error: %w", err))
		}
	}

	if e.postgres != nil {
		if err := e.postgres.Close(); err != nil {
			errs = append(errs, fmt.Errorf("PostgreSQL close error: %w", err))
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// Helper function
func withTier(inputs map[string]interface{}, tier DataTier) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range inputs {
		result[k] = v
	}
	result["data_tier"] = tier
	return result
}
