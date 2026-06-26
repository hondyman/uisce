package calcengine

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hondyman/semlayer/backend/internal/pricing"
	"github.com/hondyman/semlayer/backend/internal/wealth/risk"
	"github.com/jmoiron/sqlx"
)

// CalcEngine executes calculations using catalog DAG and external data
type CalcEngine interface {
	Run(ctx context.Context, metric string, inputs map[string]interface{}) (*CalcResult, error)
}

// CalcResult represents the output of a calculation
type CalcResult struct {
	Metric    string                   `json:"metric"`
	Value     float64                  `json:"value"`
	Sources   []string                 `json:"sources"`
	Breakdown []map[string]interface{} `json:"breakdown,omitempty"`
}

// PostgresCalcEngine uses Postgres catalog DAG for calculations
type PostgresCalcEngine struct {
	db              *sqlx.DB
	rawDB           *sql.DB
	pricingProvider pricing.PricingProvider
	riskEngine      *risk.RiskAnalyticsEngine
}

// NewPostgresCalcEngine creates a new Postgres-backed calculation engine
func NewPostgresCalcEngine(db *sqlx.DB, pricingProvider pricing.PricingProvider) *PostgresCalcEngine {
	// Get raw *sql.DB from sqlx for risk engine
	rawDB := db.DB

	return &PostgresCalcEngine{
		db:              db,
		rawDB:           rawDB,
		pricingProvider: pricingProvider,
		riskEngine:      risk.NewRiskAnalyticsEngine(rawDB),
	}
}

// Run executes a calculation
func (e *PostgresCalcEngine) Run(ctx context.Context, metric string, inputs map[string]interface{}) (*CalcResult, error) {
	switch metric {
	case "NAV":
		return e.calculateNAV(ctx, inputs)
	case "VaR":
		return e.calculateVaR(ctx, inputs)
	default:
		return nil, fmt.Errorf("unsupported metric: %s", metric)
	}
}

// calculateNAV computes Net Asset Value using holdings from catalog + external prices
func (e *PostgresCalcEngine) calculateNAV(ctx context.Context, inputs map[string]interface{}) (*CalcResult, error) {
	tenantID, ok := inputs["tenant_id"].(string)
	if !ok {
		return nil, fmt.Errorf("tenant_id required")
	}

	portfolioID, ok := inputs["portfolio_id"].(string)
	if !ok {
		return nil, fmt.Errorf("portfolio_id required")
	}

	// 1. Get holdings from catalog
	type Holding struct {
		NodeName string `db:"node_name"`
		Ticker   string `db:"ticker"`
		Quantity string `db:"quantity"`
		Currency string `db:"currency"`
	}

	var holdings []Holding
	query := `
		SELECT 
			node_name,
			properties->>'ticker' as ticker,
			properties->>'quantity' as quantity,
			COALESCE(properties->>'currency', 'USD') as currency
		FROM catalog_node
		WHERE node_type_id = (SELECT id FROM catalog_node_type WHERE name = 'Holding')
		  AND tenant_id = $1
		  AND properties->>'portfolio_id' = $2
	`

	if err := e.db.SelectContext(ctx, &holdings, query, tenantID, portfolioID); err != nil {
		return nil, fmt.Errorf("failed to fetch holdings: %w", err)
	}

	// 2. Fetch prices and calculate NAV
	var navValue float64
	var breakdown []map[string]interface{}
	var sources []string

	for _, holding := range holdings {
		// Get price from provider or inputs
		var price float64
		var err error

		if priceVal, ok := inputs[holding.Ticker+"_price"]; ok {
			price, _ = priceVal.(float64)
		} else {
			// Fetch from pricing provider
			price, err = e.pricingProvider.GetPrice(ctx, holding.Ticker)
			if err != nil {
				return nil, fmt.Errorf("failed to get price for %s: %w", holding.Ticker, err)
			}
		}

		// Get FX rate if needed
		fxRate := 1.0
		if holding.Currency != "USD" {
			fxPair := holding.Currency + "USD"
			if fxVal, ok := inputs[fxPair]; ok {
				fxRate, _ = fxVal.(float64)
			} else {
				fxRate, err = e.pricingProvider.GetFXRate(ctx, fxPair)
				if err != nil {
					// Default to 1.0 if FX lookup fails
					fxRate = 1.0
				}
			}
		}

		// Parse quantity
		var quantity float64
		fmt.Sscanf(holding.Quantity, "%f", &quantity)

		// Calculate position value
		positionValue := quantity * price * fxRate
		navValue += positionValue

		breakdown = append(breakdown, map[string]interface{}{
			"holding":   holding.NodeName,
			"ticker":    holding.Ticker,
			"quantity":  quantity,
			"price":     price,
			"currency":  holding.Currency,
			"fx_rate":   fxRate,
			"value_usd": positionValue,
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

// calculateVaR computes Value at Risk using the Risk Analytics Engine
func (e *PostgresCalcEngine) calculateVaR(ctx context.Context, inputs map[string]interface{}) (*CalcResult, error) {
	portfolioID, ok := inputs["portfolio_id"].(string)
	if !ok {
		return nil, fmt.Errorf("portfolio_id required")
	}

	// Parse optional parameters with defaults
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

	simulationCount := 10000
	if sc, ok := inputs["simulation_count"].(int); ok {
		simulationCount = sc
	}

	historicalPeriod := 252 // 1 year default
	if hp, ok := inputs["historical_period"].(int); ok {
		historicalPeriod = hp
	}

	// Build risk config
	config := risk.RiskConfig{
		PortfolioID:      portfolioID,
		AsOfDate:         time.Now(),
		ConfidenceLevels: []float64{confidenceLevel},
		Horizon:          horizon,
		Method:           method,
		SimulationCount:  simulationCount,
		HistoricalPeriod: historicalPeriod,
		Currency:         "USD",
	}

	// Calculate risk using the Risk Analytics Engine
	result, err := e.riskEngine.Calculate(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("risk calculation failed: %w", err)
	}

	// Extract VaR result for requested confidence level
	varResult, ok := result.VaRResults[confidenceLevel]
	if !ok {
		return nil, fmt.Errorf("no VaR result for confidence level %.2f", confidenceLevel)
	}

	// Build breakdown with risk details
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

	// Add CVaR if available
	if cvarResult, ok := result.CVaRResults[confidenceLevel]; ok {
		breakdown = append(breakdown, map[string]interface{}{
			"metric":           "CVaR",
			"confidence_level": confidenceLevel,
			"cvar_absolute":    cvarResult.CVaRAbsolute,
			"cvar_relative":    cvarResult.CVaRRelative,
			"avg_tail_loss":    cvarResult.AverageTailLoss,
		})
	}

	// Add tail risk metrics if available
	if result.TailRiskMetrics != nil {
		breakdown = append(breakdown, map[string]interface{}{
			"metric":          "TailRisk",
			"skewness":        result.TailRiskMetrics.Skewness,
			"excess_kurtosis": result.TailRiskMetrics.ExcessKurtosis,
			"max_drawdown":    result.TailRiskMetrics.MaxDrawdown,
			"worst_day":       result.TailRiskMetrics.WorstDay,
		})
	}

	// Add top risk contributors
	for i, contrib := range result.RiskContributions {
		if i >= 5 { // Top 5 contributors
			break
		}
		breakdown = append(breakdown, map[string]interface{}{
			"metric":           "RiskContribution",
			"security_id":      contrib.SecurityID,
			"security_name":    contrib.SecurityName,
			"weight":           contrib.Weight,
			"volatility":       contrib.Volatility,
			"marginal_var":     contrib.MarginalVaR,
			"component_var":    contrib.ComponentVaR,
			"contribution_pct": contrib.ContributionPct,
		})
	}

	return &CalcResult{
		Metric:    "VaR",
		Value:     varResult.VaRAbsolute,
		Sources:   []string{"portfolio_positions", "market_data", "historical_returns"},
		Breakdown: breakdown,
	}, nil
}

// GetDAG retrieves the calculation DAG from Postgres
func (e *PostgresCalcEngine) GetDAG(ctx context.Context, metricPath string, tenantID string) (map[string]interface{}, error) {
	var dagJSON []byte
	query := `SELECT get_calc_dag_with_metadata($1, $2)`

	if err := e.db.GetContext(ctx, &dagJSON, query, metricPath, tenantID); err != nil {
		return nil, fmt.Errorf("failed to get DAG: %w", err)
	}

	var dag map[string]interface{}
	if err := json.Unmarshal(dagJSON, &dag); err != nil {
		return nil, fmt.Errorf("failed to parse DAG: %w", err)
	}

	return dag, nil
}
