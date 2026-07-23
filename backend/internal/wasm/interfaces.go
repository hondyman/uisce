// wasm/interfaces.go
package wasm

import (
	"context"
)

// Engine defines the WASM execution interface for all four engines
type Engine interface {
	// Compliance: Evaluate a single rule against a portfolio context
	EvaluateComplianceRule(ctx context.Context, rule RuleConfig, portfolioCtx ComplianceContext) (*ComplianceEvaluationResult, error)

	// Factor Model: Compute portfolio factor exposures and volatility
	ComputeFactorModel(ctx context.Context, factorCtx FactorModelContext) (*FactorModelResult, error)

	// VaR: Compute Value-at-Risk (historical or parametric)
	ComputeVaR(ctx context.Context, varCtx VaRContext) (*VaRResult, error)

	// Stress Testing: Apply scenario shocks and compute P&L
	EvaluateScenario(ctx context.Context, scenarioCtx ScenarioContext) (*ScenarioResult, error)

	// Close releases WASM runtime resources
	Close(ctx context.Context) error
}

// RuleConfig matches the ComplianceRule.expression DSL compiled to JSON
type RuleConfig struct {
	RuleID     string  `json:"rule_id"`
	RuleCode   string  `json:"rule_code"`
	Severity   string  `json:"severity"`    // HARD, SOFT, WARNING, ALERT
	MetricType string  `json:"metric_type"` // ISSUER_WEIGHT, CASH_RATIO, SECTOR_WEIGHT, etc.
	Target     string  `json:"target"`      // issuer_id, sector, country, etc.
	Operator   string  `json:"operator"`    // <=, >=, ==, <, >
	Threshold  float64 `json:"threshold"`
}

// ComplianceContext: Input schema for compliance evaluation
// Aligns with Whitepaper §7: Rules reference semantic terms, not columns
type ComplianceContext struct {
	Portfolio struct {
		ID          string  `json:"id" jsonschema:"required,format=uuid"`
		AUM         float64 `json:"aum" jsonschema:"required,minimum=0"`
		Strategy    string  `json:"strategy"`
		BenchmarkID string  `json:"benchmark_id"`
	} `json:"portfolio" jsonschema:"required"`
	Positions []struct {
		SecurityID  string  `json:"security_id" jsonschema:"required,format=uuid"`
		Quantity    float64 `json:"quantity"`
		MarketValue float64 `json:"market_value" jsonschema:"required,minimum=0"`
		IssuerID    string  `json:"issuer_id"`
		Sector      string  `json:"sector"`
		Country     string  `json:"country"`
		Rating      string  `json:"rating"`
		ESGScore    float64 `json:"esg_score" jsonschema:"minimum=0,maximum=10"`
	} `json:"positions" jsonschema:"required"`
	Cash struct {
		ClosingBalance float64 `json:"closing_balance" jsonschema:"required"`
		Currency       string  `json:"currency" jsonschema:"required,pattern=^[A-Z]{3}$"`
	} `json:"cash" jsonschema:"required"`
	Benchmark struct {
		SectorWeights map[string]float64 `json:"sector_weights"`
		IssuerWeights map[string]float64 `json:"issuer_weights"`
	} `json:"benchmark"`
	Risk struct {
		TrackingError float64 `json:"tracking_error" jsonschema:"minimum=0"`
		Var95         float64 `json:"var_95" jsonschema:"minimum=0"`
	} `json:"risk"`
	TenantID string `json:"tenant_id" jsonschema:"required,format=uuid"` // Usice Architecture §6.2
}

// ComplianceEvaluationResult: Output schema
type ComplianceEvaluationResult struct {
	RuleID         string      `json:"rule_id" jsonschema:"required,format=uuid"`
	Status         string      `json:"status" jsonschema:"required,enum=PASS,FAIL,WARNING"`
	MetricValue    float64     `json:"metric_value"`
	ThresholdValue float64     `json:"threshold_value"`
	Details        interface{} `json:"details"`
	Lineage        struct {
		SemanticTerms   []string `json:"semantic_terms"` // Whitepaper §9: Lineage
		ExecutionTimeMs int      `json:"execution_time_ms"`
	} `json:"lineage"`
}

// FactorModelContext: Input for factor model computation
type FactorModelContext struct {
	Portfolio struct {
		ID  string  `json:"id" jsonschema:"required,format=uuid"`
		AUM float64 `json:"aum" jsonschema:"required,minimum=0"`
	} `json:"portfolio" jsonschema:"required"`
	Positions []struct {
		SecurityID  string  `json:"security_id" jsonschema:"required,format=uuid"`
		MarketValue float64 `json:"market_value" jsonschema:"required,minimum=0"`
	} `json:"positions" jsonschema:"required"`
	FactorExposures []struct {
		SecurityID string  `json:"security_id" jsonschema:"required,format=uuid"`
		FactorID   string  `json:"factor_id" jsonschema:"required"`
		Exposure   float64 `json:"exposure" jsonschema:"required"`
	} `json:"factor_exposures" jsonschema:"required"`
	FactorCovariance map[string]map[string]float64 `json:"factor_covariance" jsonschema:"required"`
	TenantID         string                        `json:"tenant_id" jsonschema:"required,format=uuid"`
}

// FactorModelResult: Output with factor contributions
type FactorModelResult struct {
	PortfolioFactorExposures map[string]float64 `json:"portfolio_factor_exposures"`
	TotalVolatility          float64            `json:"total_volatility" jsonschema:"minimum=0"`
	FactorContributions      []struct {
		FactorID     string  `json:"factor_id"`
		Contribution float64 `json:"contribution"`
	} `json:"factor_contributions"`
	Lineage struct {
		Method          string   `json:"method"` // parametric, historical
		SemanticTerms   []string `json:"semantic_terms"`
		ExecutionTimeMs int      `json:"execution_time_ms"`
	} `json:"lineage"`
}

// VaRContext: Input for VaR computation (supports both methods)
type VaRContext struct {
	Portfolio struct {
		ID  string  `json:"id" jsonschema:"required,format=uuid"`
		AUM float64 `json:"aum" jsonschema:"required,minimum=0"`
	} `json:"portfolio" jsonschema:"required"`
	Method           string                        `json:"method" jsonschema:"required,enum=historical,parametric"`
	Returns          []float64                     `json:"returns,omitempty"` // for historical
	ConfidenceLevels []float64                     `json:"confidence_levels" jsonschema:"required,minItems=1"`
	FactorExposures  map[string]float64            `json:"factor_exposures,omitempty"`  // for parametric
	FactorCovariance map[string]map[string]float64 `json:"factor_covariance,omitempty"` // for parametric
	TenantID         string                        `json:"tenant_id" jsonschema:"required,format=uuid"`
}

// VaRResult: Output with VaR and Expected Shortfall
type VaRResult struct {
	Method            string             `json:"method"`
	VaR               map[string]float64 `json:"var" jsonschema:"required"`
	ExpectedShortfall map[string]float64 `json:"expected_shortfall"`
	Lineage           struct {
		SemanticTerms   []string `json:"semantic_terms"`
		ExecutionTimeMs int      `json:"execution_time_ms"`
	} `json:"lineage"`
}

// ScenarioContext: Input for stress testing
type ScenarioContext struct {
	Portfolio struct {
		ID  string  `json:"id" jsonschema:"required,format=uuid"`
		AUM float64 `json:"aum" jsonschema:"required,minimum=0"`
	} `json:"portfolio" jsonschema:"required"`
	Positions []struct {
		SecurityID      string             `json:"security_id" jsonschema:"required,format=uuid"`
		MarketValue     float64            `json:"market_value" jsonschema:"required,minimum=0"`
		FactorExposures map[string]float64 `json:"factor_exposures" jsonschema:"required"`
	} `json:"positions" jsonschema:"required"`
	Scenario struct {
		ScenarioID string `json:"scenario_id" jsonschema:"required,format=uuid"`
		Name       string `json:"name" jsonschema:"required"`
		Shocks     struct {
			Factors []struct {
				FactorID string  `json:"factor_id" jsonschema:"required"`
				Shock    float64 `json:"shock" jsonschema:"required"`
			} `json:"factors"`
		} `json:"shocks" jsonschema:"required"`
	} `json:"scenario" jsonschema:"required"`
	TenantID string `json:"tenant_id" jsonschema:"required,format=uuid"`
}

// ScenarioResult: Output with P&L breakdown
type ScenarioResult struct {
	ScenarioID   string           `json:"scenario_id" jsonschema:"required,format=uuid"`
	PortfolioPnL float64          `json:"portfolio_pnl"`
	ByFactor     []map[string]any `json:"by_factor"`
	BySecurity   []map[string]any `json:"by_security"`
	BySector     []map[string]any `json:"by_sector"`
	Lineage      struct {
		SemanticTerms   []string `json:"semantic_terms"`
		ExecutionTimeMs int      `json:"execution_time_ms"`
	} `json:"lineage"`
}
