package render

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"cube-gonja/internal/cube"

	"github.com/nikolalohinski/gonja/v2"
	"github.com/nikolalohinski/gonja/v2/exec"
)

type Dimension struct {
	Name       string      `json:"name" yaml:"name"`
	Sql        string      `json:"sql" yaml:"sql"`
	Type       string      `json:"type" yaml:"type"`
	Format     interface{} `json:"format,omitempty" yaml:"format,omitempty"`
	PrimaryKey bool        `json:"primary_key" yaml:"primary_key"`
	// Geo-specific fields
	Latitude  *GeoCoord `json:"latitude,omitempty" yaml:"latitude,omitempty"`
	Longitude *GeoCoord `json:"longitude,omitempty" yaml:"longitude,omitempty"`
	// AtScale: Time intelligence
	TimeIntelligence *TimeIntelligence `json:"time_intelligence,omitempty" yaml:"time_intelligence,omitempty"`
	// Looker: Custom filters and user attributes
	CustomFilter   string            `json:"custom_filter,omitempty" yaml:"custom_filter,omitempty"`
	UserAttributes map[string]string `json:"user_attributes,omitempty" yaml:"user_attributes,omitempty"`
	// Microsoft Fabric: Field parameters
	FieldParameters []FieldParameter `json:"field_parameters,omitempty" yaml:"field_parameters,omitempty"`
	// DBT: Data quality
	DataQualityTests []DataQualityTest `json:"data_quality_tests,omitempty" yaml:"data_quality_tests,omitempty"`
	// Advanced features
	Description  string      `json:"description,omitempty" yaml:"description,omitempty"`
	Tags         []string    `json:"tags,omitempty" yaml:"tags,omitempty"`
	Hidden       bool        `json:"hidden" yaml:"hidden"`
	Required     bool        `json:"required" yaml:"required"`
	DefaultValue interface{} `json:"default_value,omitempty" yaml:"default_value,omitempty"`
	// If true, this dimension is populated via a sub-query reference to another cube
	SubQuery bool `json:"sub_query,omitempty" yaml:"sub_query,omitempty"`
}

type GeoCoord struct {
	Sql string `json:"sql" yaml:"sql"`
}

type TimeIntelligence struct {
	Type        string `json:"type" yaml:"type"` // "period_over_period", "rolling_average", "year_to_date", etc.
	Period      string `json:"period,omitempty" yaml:"period,omitempty"`
	Granularity string `json:"granularity,omitempty" yaml:"granularity,omitempty"`
	Offset      int    `json:"offset,omitempty" yaml:"offset,omitempty"`
}

type FieldParameter struct {
	Name         string      `json:"name" yaml:"name"`
	DisplayName  string      `json:"display_name" yaml:"display_name"`
	Type         string      `json:"type" yaml:"type"`
	DefaultValue interface{} `json:"default_value,omitempty" yaml:"default_value,omitempty"`
	Values       []string    `json:"values,omitempty" yaml:"values,omitempty"`
}

type DataQualityTest struct {
	Type        string                 `json:"type" yaml:"type"`         // "not_null", "unique", "accepted_values", etc.
	Severity    string                 `json:"severity" yaml:"severity"` // "error", "warning", "info"
	Parameters  map[string]interface{} `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Description string                 `json:"description,omitempty" yaml:"description,omitempty"`
}

type Measure struct {
	Name         string      `json:"name" yaml:"name"`
	Sql          string      `json:"sql,omitempty" yaml:"sql,omitempty"`
	Type         string      `json:"type" yaml:"type"`
	Format       interface{} `json:"format,omitempty" yaml:"format,omitempty"`
	DrillMembers []string    `json:"drill_members,omitempty" yaml:"drill_members,omitempty"`
	// AtScale: Advanced aggregations
	AggregationType string `json:"aggregation_type,omitempty" yaml:"aggregation_type,omitempty"`
	// Microsoft Fabric: Calculation groups
	CalculationGroup *CalculationGroup `json:"calculation_group,omitempty" yaml:"calculation_group,omitempty"`
	// Looker: Custom expressions
	CustomExpression string `json:"custom_expression,omitempty" yaml:"custom_expression,omitempty"`
	// DBT: Materialized views
	MaterializedView *MaterializedView `json:"materialized_view,omitempty" yaml:"materialized_view,omitempty"`
	// Advanced features
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Tags        []string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Hidden      bool     `json:"hidden" yaml:"hidden"`
	Required    bool     `json:"required" yaml:"required"`
	// Performance optimization
	PreAggregated   bool   `json:"pre_aggregated" yaml:"pre_aggregated"`
	RefreshSchedule string `json:"refresh_schedule,omitempty" yaml:"refresh_schedule,omitempty"`
	// Financial calculations
	FinancialCalc *FinancialCalculation `json:"financial_calc,omitempty" yaml:"financial_calc,omitempty"`
}

type CalculationGroup struct {
	Name       string            `json:"name" yaml:"name"`
	Expression string            `json:"expression" yaml:"expression"`
	Format     string            `json:"format,omitempty" yaml:"format,omitempty"`
	Priority   int               `json:"priority" yaml:"priority"`
	Parameters map[string]string `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

type FinancialCalculation struct {
	Type         string             `json:"type" yaml:"type"` // "irr", "xirr", "npv", "fv", etc.
	CashFlows    []CashFlow         `json:"cash_flows,omitempty" yaml:"cash_flows,omitempty"`
	Guess        float64            `json:"guess,omitempty" yaml:"guess,omitempty"`     // For IRR/XIRR (default 0.1)
	Rate         float64            `json:"rate,omitempty" yaml:"rate,omitempty"`       // For NPV calculations
	Periods      int                `json:"periods,omitempty" yaml:"periods,omitempty"` // For annuity calculations
	Payment      float64            `json:"payment,omitempty" yaml:"payment,omitempty"` // For loan/payment calculations
	PresentValue float64            `json:"present_value,omitempty" yaml:"present_value,omitempty"`
	FutureValue  float64            `json:"future_value,omitempty" yaml:"future_value,omitempty"`
	Parameters   map[string]float64 `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

type CashFlow struct {
	Amount      float64 `json:"amount" yaml:"amount"`
	Date        string  `json:"date,omitempty" yaml:"date,omitempty"`         // For XIRR
	Period      int     `json:"period,omitempty" yaml:"period,omitempty"`     // For IRR
	Category    string  `json:"category,omitempty" yaml:"category,omitempty"` // "investment", "return", "fee", etc.
	Description string  `json:"description,omitempty" yaml:"description,omitempty"`
}

type MaterializedView struct {
	Name            string `json:"name" yaml:"name"`
	RefreshType     string `json:"refresh_type" yaml:"refresh_type"` // "full", "incremental"
	RefreshSchedule string `json:"refresh_schedule,omitempty" yaml:"refresh_schedule,omitempty"`
	PartitionBy     string `json:"partition_by,omitempty" yaml:"partition_by,omitempty"`
	ClusterBy       string `json:"cluster_by,omitempty" yaml:"cluster_by,omitempty"`
}

type Hierarchy struct {
	Name   string   `json:"name" yaml:"name"`
	Title  string   `json:"title" yaml:"title"`
	Levels []string `json:"levels" yaml:"levels"`
	Public bool     `json:"public" yaml:"public"`
}

type Segment struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description" yaml:"description"`
	Sql         string `json:"sql" yaml:"sql"`
	Public      bool   `json:"public" yaml:"public"`
}

// AtScale: Perspectives for security and organization
type Perspective struct {
	Name        string   `json:"name" yaml:"name"`
	Description string   `json:"description" yaml:"description"`
	Dimensions  []string `json:"dimensions" yaml:"dimensions"`
	Measures    []string `json:"measures" yaml:"measures"`
	Users       []string `json:"users" yaml:"users"`
	Groups      []string `json:"groups" yaml:"groups"`
}

// Looker: Custom filters
type CustomFilter struct {
	Name         string      `json:"name" yaml:"name"`
	Type         string      `json:"type" yaml:"type"`
	Expression   string      `json:"expression" yaml:"expression"`
	DefaultValue interface{} `json:"default_value,omitempty" yaml:"default_value,omitempty"`
	Required     bool        `json:"required" yaml:"required"`
}

// Advanced data quality rules
type DataQualityRule struct {
	Name        string                 `json:"name" yaml:"name"`
	Type        string                 `json:"type" yaml:"type"` // "completeness", "accuracy", "consistency", etc.
	Severity    string                 `json:"severity" yaml:"severity"`
	Threshold   float64                `json:"threshold,omitempty" yaml:"threshold,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Description string                 `json:"description,omitempty" yaml:"description,omitempty"`
}

// Performance optimization hints
type PerformanceHint struct {
	Type        string                 `json:"type" yaml:"type"` // "index", "partition", "cache", etc.
	Table       string                 `json:"table" yaml:"table"`
	Columns     []string               `json:"columns,omitempty" yaml:"columns,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Description string                 `json:"description,omitempty" yaml:"description,omitempty"`
}

// Mutual Fund and Advanced Financial Calculations
type WeightedAverage struct {
	Name         string             `json:"name" yaml:"name"`
	Weights      []float64          `json:"weights" yaml:"weights"`
	Values       []float64          `json:"values" yaml:"values"`
	WeightColumn string             `json:"weight_column,omitempty" yaml:"weight_column,omitempty"`
	ValueColumn  string             `json:"value_column,omitempty" yaml:"value_column,omitempty"`
	Parameters   map[string]float64 `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

type GreeksCalculation struct {
	Type          string             `json:"type" yaml:"type"` // "delta", "gamma", "theta", "vega", "rho"
	AssetPrice    float64            `json:"asset_price" yaml:"asset_price"`
	StrikePrice   float64            `json:"strike_price" yaml:"strike_price"`
	TimeToExpiry  float64            `json:"time_to_expiry" yaml:"time_to_expiry"`
	Volatility    float64            `json:"volatility" yaml:"volatility"`
	RiskFreeRate  float64            `json:"risk_free_rate" yaml:"risk_free_rate"`
	DividendYield float64            `json:"dividend_yield,omitempty" yaml:"dividend_yield,omitempty"`
	Parameters    map[string]float64 `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

type MutualFundMetric struct {
	Name       string             `json:"name" yaml:"name"`
	Type       string             `json:"type" yaml:"type"` // "sharpe_ratio", "sortino_ratio", "alpha", "beta", "max_drawdown", "volatility", "tracking_error", etc.
	Parameters map[string]float64 `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Benchmark  string             `json:"benchmark,omitempty" yaml:"benchmark,omitempty"`
	TimePeriod string             `json:"time_period,omitempty" yaml:"time_period,omitempty"`
}

type TenantCalculationParams struct {
	TenantID            string                 `json:"tenant_id" yaml:"tenant_id"`
	DefaultRiskFreeRate float64                `json:"default_risk_free_rate" yaml:"default_risk_free_rate"`
	DefaultBenchmark    string                 `json:"default_benchmark" yaml:"default_benchmark"`
	CustomMetrics       map[string]interface{} `json:"custom_metrics,omitempty" yaml:"custom_metrics,omitempty"`
	DataQualityRules    []DataQualityRule      `json:"data_quality_rules,omitempty" yaml:"data_quality_rules,omitempty"`
	PerformanceHints    []PerformanceHint      `json:"performance_hints,omitempty" yaml:"performance_hints,omitempty"`
}

type ScalingConfig struct {
	MaterializedViews []MaterializedView `json:"materialized_views,omitempty" yaml:"materialized_views,omitempty"`
	Partitioning      []PartitionConfig  `json:"partitioning,omitempty" yaml:"partitioning,omitempty"`
	Caching           []CacheConfig      `json:"caching,omitempty" yaml:"caching,omitempty"`
}

type PartitionConfig struct {
	Table       string `json:"table" yaml:"table"`
	Column      string `json:"column" yaml:"column"`
	Type        string `json:"type" yaml:"type"` // "range", "hash", "list"
	Granularity string `json:"granularity,omitempty" yaml:"granularity,omitempty"`
}

type CacheConfig struct {
	Name            string `json:"name" yaml:"name"`
	Table           string `json:"table" yaml:"table"`
	TTL             int    `json:"ttl" yaml:"ttl"`                   // Time to live in seconds
	RefreshType     string `json:"refresh_type" yaml:"refresh_type"` // "manual", "auto", "scheduled"
	RefreshSchedule string `json:"refresh_schedule,omitempty" yaml:"refresh_schedule,omitempty"`
}

// Wealth Management Metrics
type WealthManagementMetric struct {
	Name       string             `json:"name" yaml:"name"`
	Type       string             `json:"type" yaml:"type"` // "sharpe_ratio", "sortino_ratio", "information_ratio", "upside_capture", "downside_capture", "var", "cvar", etc.
	Parameters map[string]float64 `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Benchmark  string             `json:"benchmark,omitempty" yaml:"benchmark,omitempty"`
	TimePeriod string             `json:"time_period,omitempty" yaml:"time_period,omitempty"`
	Confidence float64            `json:"confidence,omitempty" yaml:"confidence,omitempty"` // For VaR calculations (e.g., 0.95 for 95% confidence)
}

type RiskMetrics struct {
	Name             string             `json:"name" yaml:"name"`
	PortfolioReturns []float64          `json:"portfolio_returns" yaml:"portfolio_returns"`
	BenchmarkReturns []float64          `json:"benchmark_returns,omitempty" yaml:"benchmark_returns,omitempty"`
	RiskFreeRate     float64            `json:"risk_free_rate" yaml:"risk_free_rate"`
	TargetReturn     float64            `json:"target_return,omitempty" yaml:"target_return,omitempty"`
	Parameters       map[string]float64 `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

type BenchmarkingMetrics struct {
	Name             string             `json:"name" yaml:"name"`
	PortfolioReturns []float64          `json:"portfolio_returns" yaml:"portfolio_returns"`
	BenchmarkReturns []float64          `json:"benchmark_returns" yaml:"benchmark_returns"`
	RiskFreeRate     float64            `json:"risk_free_rate" yaml:"risk_free_rate"`
	Parameters       map[string]float64 `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

type PortfolioAnalytics struct {
	Name             string             `json:"name" yaml:"name"`
	PortfolioReturns []float64          `json:"portfolio_returns" yaml:"portfolio_returns"`
	BenchmarkReturns []float64          `json:"benchmark_returns,omitempty" yaml:"benchmark_returns,omitempty"`
	Weights          []float64          `json:"weights,omitempty" yaml:"weights,omitempty"`
	RiskFreeRate     float64            `json:"risk_free_rate" yaml:"risk_free_rate"`
	TimePeriod       string             `json:"time_period,omitempty" yaml:"time_period,omitempty"`
	Parameters       map[string]float64 `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

type PrivateEquityMetrics struct {
	Name             string             `json:"name" yaml:"name"`
	Type             string             `json:"type" yaml:"type"` // "moic", "tvpi", "dpi", "rvpi", "pme"
	Distributions    []float64          `json:"distributions,omitempty" yaml:"distributions,omitempty"`
	Contributions    []float64          `json:"contributions,omitempty" yaml:"contributions,omitempty"`
	ResidualValue    float64            `json:"residual_value,omitempty" yaml:"residual_value,omitempty"`
	PaidInCapital    float64            `json:"paid_in_capital,omitempty" yaml:"paid_in_capital,omitempty"`
	BenchmarkReturns []float64          `json:"benchmark_returns,omitempty" yaml:"benchmark_returns,omitempty"`
	Parameters       map[string]float64 `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

// Additional Wealth Management Metrics
type PortfolioEfficiencyMetric struct {
	Name                  string             `json:"name" yaml:"name"`
	Type                  string             `json:"type" yaml:"type"` // "expense_ratio", "turnover_ratio", "liquidity_ratio"
	TotalExpenses         float64            `json:"total_expenses,omitempty" yaml:"total_expenses,omitempty"`
	AverageAssets         float64            `json:"average_assets,omitempty" yaml:"average_assets,omitempty"`
	Purchases             float64            `json:"purchases,omitempty" yaml:"purchases,omitempty"`
	Sales                 float64            `json:"sales,omitempty" yaml:"sales,omitempty"`
	AveragePortfolioValue float64            `json:"average_portfolio_value,omitempty" yaml:"average_portfolio_value,omitempty"`
	LiquidAssets          float64            `json:"liquid_assets,omitempty" yaml:"liquid_assets,omitempty"`
	TotalPortfolioValue   float64            `json:"total_portfolio_value,omitempty" yaml:"total_portfolio_value,omitempty"`
	Parameters            map[string]float64 `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

type TaxAwareMetric struct {
	Name           string             `json:"name" yaml:"name"`
	Type           string             `json:"type" yaml:"type"` // "tax_drag", "effective_tax_rate"
	PreTaxReturn   float64            `json:"pre_tax_return,omitempty" yaml:"pre_tax_return,omitempty"`
	AfterTaxReturn float64            `json:"after_tax_return,omitempty" yaml:"after_tax_return,omitempty"`
	TaxesPaid      float64            `json:"taxes_paid,omitempty" yaml:"taxes_paid,omitempty"`
	RealizedGains  float64            `json:"realized_gains,omitempty" yaml:"realized_gains,omitempty"`
	Parameters     map[string]float64 `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

type GoalBasedMetric struct {
	Name                    string             `json:"name" yaml:"name"`
	Type                    string             `json:"type" yaml:"type"` // "probability_of_success", "funding_ratio"
	PresentValueAssets      float64            `json:"present_value_assets,omitempty" yaml:"present_value_assets,omitempty"`
	PresentValueLiabilities float64            `json:"present_value_liabilities,omitempty" yaml:"present_value_liabilities,omitempty"`
	GoalCashFlows           []float64          `json:"goal_cash_flows,omitempty" yaml:"goal_cash_flows,omitempty"`
	InflationAssumptions    []float64          `json:"inflation_assumptions,omitempty" yaml:"inflation_assumptions,omitempty"`
	AssetProjections        []float64          `json:"asset_projections,omitempty" yaml:"asset_projections,omitempty"`
	Simulations             int                `json:"simulations,omitempty" yaml:"simulations,omitempty"` // For Monte Carlo
	Parameters              map[string]float64 `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

type BehavioralMetric struct {
	Name            string    `json:"name" yaml:"name"`
	Type            string    `json:"type" yaml:"type"` // "behavior_gap", "diversification_score"
	PortfolioReturn float64   `json:"portfolio_return,omitempty" yaml:"portfolio_return,omitempty"`
	InvestorReturn  float64   `json:"investor_return,omitempty" yaml:"investor_return,omitempty"`
	CashFlows       []float64 `json:"cash_flows,omitempty" yaml:"cash_flows,omitempty"`
	TimestampFlows  []string  `json:"timestamp_flows,omitempty" yaml:"timestamp_flows,omitempty"`
	// For diversification score
	AssetClassWeights map[string]float64 `json:"asset_class_weights,omitempty" yaml:"asset_class_weights,omitempty"`
	GeographyWeights  map[string]float64 `json:"geography_weights,omitempty" yaml:"geography_weights,omitempty"`
	SectorWeights     map[string]float64 `json:"sector_weights,omitempty" yaml:"sector_weights,omitempty"`
	FactorWeights     map[string]float64 `json:"factor_weights,omitempty" yaml:"factor_weights,omitempty"`
	Parameters        map[string]float64 `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

// Catalog-driven Metric Definition
type MetricDefinition struct {
	Name        string                 `json:"name" yaml:"name"`
	Type        string                 `json:"type" yaml:"type"` // "financial", "wealth_management", "risk", "benchmarking", "portfolio", "private_equity", "efficiency", "tax_aware", "goal_based", "behavioral"
	Description string                 `json:"description,omitempty" yaml:"description,omitempty"`
	Tags        []string               `json:"tags,omitempty" yaml:"tags,omitempty"`
	Version     string                 `json:"version" yaml:"version"`
	SchemaDef   map[string]interface{} `json:"schema_def" yaml:"schema_def"` // JSON Schema for validation
	Parameters  map[string]interface{} `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	// Catalog metadata
	CatalogID  string   `json:"catalog_id" yaml:"catalog_id"`
	CreatedAt  string   `json:"created_at" yaml:"created_at"`
	UpdatedAt  string   `json:"updated_at" yaml:"updated_at"`
	CreatedBy  string   `json:"created_by" yaml:"created_by"`
	IsActive   bool     `json:"is_active" yaml:"is_active"`
	AuditTrail []string `json:"audit_trail,omitempty" yaml:"audit_trail,omitempty"`
	// Governance
	DataQualityRules []DataQualityRule `json:"data_quality_rules,omitempty" yaml:"data_quality_rules,omitempty"`
	PerformanceHints []PerformanceHint `json:"performance_hints,omitempty" yaml:"performance_hints,omitempty"`
	// Template integration
	TemplatePath string `json:"template_path,omitempty" yaml:"template_path,omitempty"`
}

// PreAggregation defines a materialized or rollup pre-aggregation
type PreAggregation struct {
	Name                   string                 `json:"name" yaml:"name"`
	Type                   string                 `json:"type" yaml:"type"` // rollup, rollup_with_time_dimension, original
	TimeDimension          string                 `json:"time_dimension,omitempty" yaml:"time_dimension,omitempty"`
	Granularity            string                 `json:"granularity,omitempty" yaml:"granularity,omitempty"`
	Measures               []string               `json:"measures" yaml:"measures"`
	Dimensions             []string               `json:"dimensions" yaml:"dimensions"`
	PartitionBy            string                 `json:"partition_by,omitempty" yaml:"partition_by,omitempty"`
	ClusterBy              string                 `json:"cluster_by,omitempty" yaml:"cluster_by,omitempty"`
	RefreshKey             interface{}            `json:"refresh_key,omitempty" yaml:"refresh_key,omitempty"`
	Scheduled              string                 `json:"scheduled,omitempty" yaml:"scheduled,omitempty"` // cron or interval
	Storage                string                 `json:"storage,omitempty" yaml:"storage,omitempty"`     // table|materialized_view
	SQL                    string                 `json:"sql,omitempty" yaml:"sql,omitempty"`             // optional override
	PreAggregatedTableName string                 `json:"pre_aggregated_table_name,omitempty" yaml:"pre_aggregated_table_name,omitempty"`
	Meta                   map[string]interface{} `json:"meta,omitempty" yaml:"meta,omitempty"`
}

type Context struct {
	// Map cube -> data_source (hard bind)
	DataSources map[string]string `json:"data_sources"`
	// Map cube -> dimensions
	Dimensions map[string][]Dimension `json:"dimensions"`
	// Map cube -> measures
	Measures map[string][]Measure `json:"measures"`
	// Map cube -> hierarchies
	Hierarchies map[string][]Hierarchy `json:"hierarchies"`
	// Map cube -> segments
	Segments map[string][]Segment `json:"segments"`
	// AtScale: Perspectives for security and organization
	Perspectives map[string][]Perspective `json:"perspectives,omitempty"`
	// Microsoft Fabric: Calculation groups
	CalculationGroups map[string][]CalculationGroup `json:"calculation_groups,omitempty"`
	// DBT: Materialized views
	MaterializedViews map[string][]MaterializedView `json:"materialized_views,omitempty"`
	// Looker: User attributes and custom filters
	UserAttributes map[string]map[string]string `json:"user_attributes,omitempty"`
	CustomFilters  map[string][]CustomFilter    `json:"custom_filters,omitempty"`
	// Advanced features
	DataQualityRules map[string][]DataQualityRule `json:"data_quality_rules,omitempty"`
	PerformanceHints map[string][]PerformanceHint `json:"performance_hints,omitempty"`
	// Mutual Fund and Advanced Financial Calculations
	WeightedAverages   map[string][]WeightedAverage       `json:"weighted_averages,omitempty"`
	GreeksCalculations map[string][]GreeksCalculation     `json:"greeks_calculations,omitempty"`
	MutualFundMetrics  map[string][]MutualFundMetric      `json:"mutual_fund_metrics,omitempty"`
	TenantParams       map[string]TenantCalculationParams `json:"tenant_params,omitempty"`
	ScalingConfig      ScalingConfig                      `json:"scaling_config,omitempty"`
	// Wealth Management Metrics
	WealthManagementMetrics map[string][]WealthManagementMetric `json:"wealth_management_metrics,omitempty"`
	RiskMetrics             map[string][]RiskMetrics            `json:"risk_metrics,omitempty"`
	BenchmarkingMetrics     map[string][]BenchmarkingMetrics    `json:"benchmarking_metrics,omitempty"`
	PortfolioAnalytics      map[string][]PortfolioAnalytics     `json:"portfolio_analytics,omitempty"`
	PrivateEquityMetrics    map[string][]PrivateEquityMetrics   `json:"private_equity_metrics,omitempty"`
	// Additional Wealth Management Metrics
	PortfolioEfficiencyMetrics map[string][]PortfolioEfficiencyMetric `json:"portfolio_efficiency_metrics,omitempty"`
	TaxAwareMetrics            map[string][]TaxAwareMetric            `json:"tax_aware_metrics,omitempty"`
	GoalBasedMetrics           map[string][]GoalBasedMetric           `json:"goal_based_metrics,omitempty"`
	BehavioralMetrics          map[string][]BehavioralMetric          `json:"behavioral_metrics,omitempty"`
	// Catalog Metrics
	Metrics map[string][]MetricDefinition `json:"metrics,omitempty"`
	// Pre-aggregations (modeled after Cube.js nested pre-aggregates)
	PreAggregations map[string][]PreAggregation `json:"pre_aggregations,omitempty" yaml:"pre_aggregations,omitempty"`
	// Arbitrary extra data for templates
	Extra map[string]any `json:"extra"`
}

type Service struct {
	tmplDir     string
	baseTmplDir string
	outDir      string
	mu          sync.RWMutex
	ctx         Context
	// Allowed data sources for enforcement
	allowedDS map[string]struct{}
}

func NewService(templateDir, baseTemplateDir, outputDir string, allowedDS map[string]struct{}) *Service {
	return &Service{
		tmplDir:     templateDir,
		baseTmplDir: baseTemplateDir,
		outDir:      outputDir,
		allowedDS:   allowedDS,
	}
}

func (s *Service) UpdateContext(ctx Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ctx = ctx
}

func (s *Service) contextFunctions(cubeName string) *exec.Context {
	s.mu.RLock()
	local := s.ctx
	s.mu.RUnlock()

	// Start with cube package's global template context
	ctx := cube.GetGlobalTemplateContext().ToGonjaContext()

	// Set the CUBE variable to the current cube name
	ctx.Set("CUBE", cubeName)

	// Set up FILTER_PARAMS for the current cube
	cubeFilterParams := cube.NewCubeFilterParams(cubeName)
	ctx.Set("FILTER_PARAMS", map[string]interface{}{
		cubeName: cubeFilterParams,
	})

	// Set up SQL_UTILS functions
	ctx.Set("convertTz", func(expr string) string {
		return fmt.Sprintf("CONVERT_TZ(%s, @@session.time_zone, '+00:00')", expr)
	})

	// Set up COMPILE_CONTEXT with some default values
	ctx.Set("COMPILE_CONTEXT", map[string]interface{}{
		"securityContext": map[string]interface{}{
			"tenant_id": "default",
			"user_id":   "test_user",
		},
		"extra": map[string]interface{}{
			"environment": "development",
		},
	})

	// Add our specific functions
	ctx.Set("get_dimensions", func(cube string) []Dimension {
		return local.Dimensions[cube]
	})
	ctx.Set("get_data_source", func(cube string) string {
		ds := local.DataSources[cube]
		return ds
	})
	ctx.Set("get_hierarchies", func(cube string) []Hierarchy {
		return local.Hierarchies[cube]
	})
	ctx.Set("get_segments", func(cube string) []Segment {
		return local.Segments[cube]
	})
	ctx.Set("get_measures", func(cube string) []Measure {
		return local.Measures[cube]
	})

	// AtScale: Perspectives
	ctx.Set("get_perspectives", func(cube string) []Perspective {
		return local.Perspectives[cube]
	})

	// Microsoft Fabric: Calculation groups
	ctx.Set("get_calculation_groups", func(cube string) []CalculationGroup {
		return local.CalculationGroups[cube]
	})

	// DBT: Materialized views
	ctx.Set("get_materialized_views", func(cube string) []MaterializedView {
		return local.MaterializedViews[cube]
	})

	// Looker: User attributes and custom filters
	ctx.Set("get_user_attributes", func(user string) map[string]string {
		if local.UserAttributes != nil {
			return local.UserAttributes[user]
		}
		return nil
	})
	ctx.Set("get_custom_filters", func(cube string) []CustomFilter {
		return local.CustomFilters[cube]
	})

	// Advanced features
	ctx.Set("get_data_quality_rules", func(cube string) []DataQualityRule {
		return local.DataQualityRules[cube]
	})
	ctx.Set("get_performance_hints", func(cube string) []PerformanceHint {
		return local.PerformanceHints[cube]
	})

	// Mutual Fund and Advanced Financial Calculations
	ctx.Set("get_weighted_averages", func(cube string) []WeightedAverage {
		return local.WeightedAverages[cube]
	})
	ctx.Set("get_greeks_calculations", func(cube string) []GreeksCalculation {
		return local.GreeksCalculations[cube]
	})
	ctx.Set("get_mutual_fund_metrics", func(cube string) []MutualFundMetric {
		return local.MutualFundMetrics[cube]
	})
	ctx.Set("get_tenant_params", func(tenantID string) TenantCalculationParams {
		return local.TenantParams[tenantID]
	})
	ctx.Set("get_scaling_config", func() ScalingConfig {
		return local.ScalingConfig
	})

	// Wealth Management Metrics
	ctx.Set("get_wealth_management_metrics", func(cube string) []WealthManagementMetric {
		return local.WealthManagementMetrics[cube]
	})
	ctx.Set("get_risk_metrics", func(cube string) []RiskMetrics {
		return local.RiskMetrics[cube]
	})
	ctx.Set("get_benchmarking_metrics", func(cube string) []BenchmarkingMetrics {
		return local.BenchmarkingMetrics[cube]
	})
	ctx.Set("get_portfolio_analytics", func(cube string) []PortfolioAnalytics {
		return local.PortfolioAnalytics[cube]
	})
	ctx.Set("get_private_equity_metrics", func(cube string) []PrivateEquityMetrics {
		return local.PrivateEquityMetrics[cube]
	})

	// Additional Wealth Management Metrics
	ctx.Set("get_portfolio_efficiency_metrics", func(cube string) []PortfolioEfficiencyMetric {
		return local.PortfolioEfficiencyMetrics[cube]
	})
	ctx.Set("get_tax_aware_metrics", func(cube string) []TaxAwareMetric {
		return local.TaxAwareMetrics[cube]
	})
	ctx.Set("get_goal_based_metrics", func(cube string) []GoalBasedMetric {
		return local.GoalBasedMetrics[cube]
	})
	ctx.Set("get_behavioral_metrics", func(cube string) []BehavioralMetric {
		return local.BehavioralMetrics[cube]
	})

	// Time intelligence functions (AtScale)
	ctx.Set("period_over_period", func(expr string, period string, offset int) string {
		return fmt.Sprintf("PERIOD_OVER_PERIOD(%s, '%s', %d)", expr, period, offset)
	})
	ctx.Set("rolling_average", func(expr string, window int) string {
		return fmt.Sprintf("ROLLING_AVERAGE(%s, %d)", expr, window)
	})
	ctx.Set("year_to_date", func(expr string) string {
		return fmt.Sprintf("YEAR_TO_DATE(%s)", expr)
	})

	// Microsoft Fabric calculation functions
	ctx.Set("calculate", func(expr string, filter string) string {
		if filter != "" {
			return fmt.Sprintf("CALCULATE(%s, %s)", expr, filter)
		}
		return fmt.Sprintf("CALCULATE(%s)", expr)
	})

	// Financial calculation functions
	ctx.Set("irr", func(cashFlows []float64, guess float64) string {
		if guess == 0 {
			guess = 0.1 // Default 10%
		}
		return fmt.Sprintf("IRR(ARRAY[%s], %.6f)", joinFloats(cashFlows, ","), guess)
	})

	ctx.Set("xirr", func(cashFlows []float64, dates []string, guess float64) string {
		if guess == 0 {
			guess = 0.1 // Default 10%
		}
		return fmt.Sprintf("XIRR(ARRAY[%s], ARRAY[%s], %.6f)", joinFloats(cashFlows, ","), joinStrings(dates, ","), guess)
	})

	ctx.Set("npv", func(rate float64, cashFlows []float64) string {
		return fmt.Sprintf("NPV(%.6f, ARRAY[%s])", rate, joinFloats(cashFlows, ","))
	})

	ctx.Set("fv", func(rate float64, periods int, payment float64, presentValue float64) string {
		return fmt.Sprintf("FV(%.6f, %d, %.2f, %.2f)", rate, periods, payment, presentValue)
	})

	ctx.Set("pv", func(rate float64, periods int, payment float64, futureValue float64) string {
		return fmt.Sprintf("PV(%.6f, %d, %.2f, %.2f)", rate, periods, payment, futureValue)
	})

	ctx.Set("pmt", func(rate float64, periods int, presentValue float64, futureValue float64) string {
		return fmt.Sprintf("PMT(%.6f, %d, %.2f, %.2f)", rate, periods, presentValue, futureValue)
	})

	// Weighted average functions
	ctx.Set("weighted_average", func(weights []float64, values []float64) string {
		return fmt.Sprintf("WEIGHTED_AVERAGE(ARRAY[%s], ARRAY[%s])", joinFloats(weights, ","), joinFloats(values, ","))
	})

	ctx.Set("weighted_average_sql", func(weightCol string, valueCol string, table string) string {
		return fmt.Sprintf("SUM(%s * %s) / SUM(%s) FROM %s", weightCol, valueCol, weightCol, table)
	})

	// Greeks calculation functions
	ctx.Set("delta", func(assetPrice float64, strikePrice float64, timeToExpiry float64, volatility float64, riskFreeRate float64, dividendYield float64) string {
		return fmt.Sprintf("DELTA(%.2f, %.2f, %.4f, %.4f, %.6f, %.6f)", assetPrice, strikePrice, timeToExpiry, volatility, riskFreeRate, dividendYield)
	})

	ctx.Set("gamma", func(assetPrice float64, strikePrice float64, timeToExpiry float64, volatility float64, riskFreeRate float64, dividendYield float64) string {
		return fmt.Sprintf("GAMMA(%.2f, %.2f, %.4f, %.4f, %.6f, %.6f)", assetPrice, strikePrice, timeToExpiry, volatility, riskFreeRate, dividendYield)
	})

	ctx.Set("theta", func(assetPrice float64, strikePrice float64, timeToExpiry float64, volatility float64, riskFreeRate float64, dividendYield float64) string {
		return fmt.Sprintf("THETA(%.2f, %.2f, %.4f, %.4f, %.6f, %.6f)", assetPrice, strikePrice, timeToExpiry, volatility, riskFreeRate, dividendYield)
	})

	ctx.Set("vega", func(assetPrice float64, strikePrice float64, timeToExpiry float64, volatility float64, riskFreeRate float64, dividendYield float64) string {
		return fmt.Sprintf("VEGA(%.2f, %.2f, %.4f, %.4f, %.6f, %.6f)", assetPrice, strikePrice, timeToExpiry, volatility, riskFreeRate, dividendYield)
	})

	ctx.Set("rho", func(assetPrice float64, strikePrice float64, timeToExpiry float64, volatility float64, riskFreeRate float64, dividendYield float64) string {
		return fmt.Sprintf("RHO(%.2f, %.2f, %.4f, %.4f, %.6f, %.6f)", assetPrice, strikePrice, timeToExpiry, volatility, riskFreeRate, dividendYield)
	})

	// Mutual fund metrics
	ctx.Set("sharpe_ratio", func(returns []float64, riskFreeRate float64) string {
		return fmt.Sprintf("SHARPE_RATIO(ARRAY[%s], %.6f)", joinFloats(returns, ","), riskFreeRate)
	})

	ctx.Set("sortino_ratio", func(returns []float64, riskFreeRate float64, targetReturn float64) string {
		return fmt.Sprintf("SORTINO_RATIO(ARRAY[%s], %.6f, %.6f)", joinFloats(returns, ","), riskFreeRate, targetReturn)
	})

	ctx.Set("alpha", func(portfolioReturns []float64, benchmarkReturns []float64, riskFreeRate float64) string {
		return fmt.Sprintf("ALPHA(ARRAY[%s], ARRAY[%s], %.6f)", joinFloats(portfolioReturns, ","), joinFloats(benchmarkReturns, ","), riskFreeRate)
	})

	ctx.Set("beta", func(portfolioReturns []float64, benchmarkReturns []float64) string {
		return fmt.Sprintf("BETA(ARRAY[%s], ARRAY[%s])", joinFloats(portfolioReturns, ","), joinFloats(benchmarkReturns, ","))
	})

	ctx.Set("max_drawdown", func(values []float64) string {
		return fmt.Sprintf("MAX_DRAWDOWN(ARRAY[%s])", joinFloats(values, ","))
	})

	ctx.Set("volatility", func(returns []float64, annualize bool) string {
		annualizeStr := "FALSE"
		if annualize {
			annualizeStr = "TRUE"
		}
		return fmt.Sprintf("VOLATILITY(ARRAY[%s], %s)", joinFloats(returns, ","), annualizeStr)
	})

	ctx.Set("tracking_error", func(portfolioReturns []float64, benchmarkReturns []float64) string {
		return fmt.Sprintf("TRACKING_ERROR(ARRAY[%s], ARRAY[%s])", joinFloats(portfolioReturns, ","), joinFloats(benchmarkReturns, ","))
	})

	// Wealth Management - Risk-Adjusted Return Metrics
	ctx.Set("information_ratio", func(portfolioReturns []float64, benchmarkReturns []float64) string {
		return fmt.Sprintf("INFORMATION_RATIO(ARRAY[%s], ARRAY[%s])", joinFloats(portfolioReturns, ","), joinFloats(benchmarkReturns, ","))
	})

	ctx.Set("downside_deviation", func(returns []float64, targetReturn float64) string {
		return fmt.Sprintf("DOWNSIDE_DEVIATION(ARRAY[%s], %.6f)", joinFloats(returns, ","), targetReturn)
	})

	ctx.Set("upside_capture_ratio", func(portfolioReturns []float64, benchmarkReturns []float64) string {
		return fmt.Sprintf("UPSIDE_CAPTURE_RATIO(ARRAY[%s], ARRAY[%s])", joinFloats(portfolioReturns, ","), joinFloats(benchmarkReturns, ","))
	})

	ctx.Set("downside_capture_ratio", func(portfolioReturns []float64, benchmarkReturns []float64) string {
		return fmt.Sprintf("DOWNSIDE_CAPTURE_RATIO(ARRAY[%s], ARRAY[%s])", joinFloats(portfolioReturns, ","), joinFloats(benchmarkReturns, ","))
	})

	ctx.Set("value_at_risk", func(returns []float64, confidence float64) string {
		if confidence == 0 {
			confidence = 0.95 // Default 95% confidence
		}
		return fmt.Sprintf("VALUE_AT_RISK(ARRAY[%s], %.4f)", joinFloats(returns, ","), confidence)
	})

	ctx.Set("conditional_var", func(returns []float64, confidence float64) string {
		if confidence == 0 {
			confidence = 0.95 // Default 95% confidence
		}
		return fmt.Sprintf("CONDITIONAL_VAR(ARRAY[%s], %.4f)", joinFloats(returns, ","), confidence)
	})

	ctx.Set("r_squared", func(portfolioReturns []float64, benchmarkReturns []float64) string {
		return fmt.Sprintf("R_SQUARED(ARRAY[%s], ARRAY[%s])", joinFloats(portfolioReturns, ","), joinFloats(benchmarkReturns, ","))
	})

	ctx.Set("jensens_alpha", func(portfolioReturns []float64, benchmarkReturns []float64, riskFreeRate float64) string {
		return fmt.Sprintf("JENSENS_ALPHA(ARRAY[%s], ARRAY[%s], %.6f)", joinFloats(portfolioReturns, ","), joinFloats(benchmarkReturns, ","), riskFreeRate)
	})

	// Private Equity - Multiples and PME
	ctx.Set("moic", func(distributions []float64, contributions []float64, residualValue float64) string {
		return fmt.Sprintf("MOIC(ARRAY[%s], ARRAY[%s], %.2f)", joinFloats(distributions, ","), joinFloats(contributions, ","), residualValue)
	})

	ctx.Set("tvpi", func(distributions []float64, contributions []float64, residualValue float64) string {
		return fmt.Sprintf("TVPI(ARRAY[%s], ARRAY[%s], %.2f)", joinFloats(distributions, ","), joinFloats(contributions, ","), residualValue)
	})

	ctx.Set("dpi", func(distributions []float64, contributions []float64) string {
		return fmt.Sprintf("DPI(ARRAY[%s], ARRAY[%s])", joinFloats(distributions, ","), joinFloats(contributions, ","))
	})

	ctx.Set("rvpi", func(residualValue float64, contributions []float64) string {
		return fmt.Sprintf("RVPI(%.2f, ARRAY[%s])", residualValue, joinFloats(contributions, ","))
	})

	ctx.Set("pme", func(fundDistributions []float64, fundContributions []float64, benchmarkReturns []float64) string {
		return fmt.Sprintf("PME(ARRAY[%s], ARRAY[%s], ARRAY[%s])", joinFloats(fundDistributions, ","), joinFloats(fundContributions, ","), joinFloats(benchmarkReturns, ","))
	})

	// Additional Wealth Management Metrics
	ctx.Set("expense_ratio", func(totalExpenses float64, averageAssets float64) string {
		return fmt.Sprintf("EXPENSE_RATIO(%.2f, %.2f)", totalExpenses, averageAssets)
	})

	ctx.Set("turnover_ratio", func(purchases float64, sales float64, averagePortfolioValue float64) string {
		return fmt.Sprintf("TURNOVER_RATIO(%.2f, %.2f, %.2f)", purchases, sales, averagePortfolioValue)
	})

	ctx.Set("liquidity_ratio", func(liquidAssets float64, totalPortfolioValue float64) string {
		return fmt.Sprintf("LIQUIDITY_RATIO(%.2f, %.2f)", liquidAssets, totalPortfolioValue)
	})

	ctx.Set("tax_drag", func(preTaxReturn float64, afterTaxReturn float64) string {
		return fmt.Sprintf("TAX_DRAG(%.6f, %.6f)", preTaxReturn, afterTaxReturn)
	})

	ctx.Set("effective_tax_rate", func(taxesPaid float64, realizedGains float64) string {
		return fmt.Sprintf("EFFECTIVE_TAX_RATE(%.2f, %.2f)", taxesPaid, realizedGains)
	})

	ctx.Set("funding_ratio", func(presentValueAssets float64, presentValueLiabilities float64) string {
		return fmt.Sprintf("FUNDING_RATIO(%.2f, %.2f)", presentValueAssets, presentValueLiabilities)
	})

	ctx.Set("behavior_gap", func(portfolioReturn float64, investorReturn float64) string {
		return fmt.Sprintf("BEHAVIOR_GAP(%.6f, %.6f)", portfolioReturn, investorReturn)
	})

	ctx.Set("diversification_score", func(assetClassWeights map[string]float64, geographyWeights map[string]float64, sectorWeights map[string]float64, factorWeights map[string]float64) string {
		// Convert maps to string format
		assetStr := joinMap(assetClassWeights, ",")
		geoStr := joinMap(geographyWeights, ",")
		sectorStr := joinMap(sectorWeights, ",")
		factorStr := joinMap(factorWeights, ",")
		return fmt.Sprintf("DIVERSIFICATION_SCORE('%s', '%s', '%s', '%s')", assetStr, geoStr, sectorStr, factorStr)
	})

	ctx.Set("probability_of_success", func(goalCashFlows []float64, inflationAssumptions []float64, assetProjections []float64, simulations int) string {
		if simulations == 0 {
			simulations = 10000 // Default Monte Carlo simulations
		}
		return fmt.Sprintf("PROBABILITY_OF_SUCCESS(ARRAY[%s], ARRAY[%s], ARRAY[%s], %d)", joinFloats(goalCashFlows, ","), joinFloats(inflationAssumptions, ","), joinFloats(assetProjections, ","), simulations)
	})

	// Portfolio Analytics
	ctx.Set("portfolio_volatility", func(weights []float64, volatilities []float64, correlations [][]float64) string {
		// Convert correlation matrix to string format
		corrStr := ""
		for i, row := range correlations {
			if i > 0 {
				corrStr += ";"
			}
			corrStr += joinFloats(row, ",")
		}
		return fmt.Sprintf("PORTFOLIO_VOLATILITY(ARRAY[%s], ARRAY[%s], ARRAY[%s])", joinFloats(weights, ","), joinFloats(volatilities, ","), corrStr)
	})

	ctx.Set("portfolio_sharpe", func(weights []float64, expectedReturns []float64, volatilities []float64, correlations [][]float64, riskFreeRate float64) string {
		// Convert correlation matrix to string format
		corrStr := ""
		for i, row := range correlations {
			if i > 0 {
				corrStr += ";"
			}
			corrStr += joinFloats(row, ",")
		}
		return fmt.Sprintf("PORTFOLIO_SHARPE(ARRAY[%s], ARRAY[%s], ARRAY[%s], '%s', %.6f)", joinFloats(weights, ","), joinFloats(expectedReturns, ","), joinFloats(volatilities, ","), corrStr, riskFreeRate)
	})

	ctx.Set("ctx", local.Extra) // expose arbitrary data as ctx.something

	return ctx
}

func (s *Service) RenderOne(name string) (string, []byte, error) {
	tplPath := filepath.Join(s.tmplDir, fmt.Sprintf("%s.yml.gonja", name))
	tpl, err := gonja.FromFile(tplPath)
	if err != nil && s.baseTmplDir != "" {
		// Fall back to base template
		tplPath = filepath.Join(s.baseTmplDir, fmt.Sprintf("%s.yml.gonja", name))
		tpl, err = gonja.FromFile(tplPath)
	}
	if err != nil {
		return "", nil, err
	}
	out, err := tpl.ExecuteToString(s.contextFunctions(name))
	if err != nil {
		return "", nil, err
	}
	// Hard-binding enforcement: ensure output contains data_source and is allowed
	if err := s.enforceHardBinding(name, out); err != nil {
		return "", nil, err
	}
	outPath := filepath.Join(s.outDir, fmt.Sprintf("%s.yml", name))
	return outPath, []byte(out), nil
}

func (s *Service) RenderAll() (map[string][]byte, error) {
	// Get files from tenant dir
	files, err := filepath.Glob(filepath.Join(s.tmplDir, "*.yml.gonja"))
	if err != nil {
		return nil, err
	}

	// If base dir exists, get files from there too
	baseFiles := []string{}
	if s.baseTmplDir != "" {
		baseFiles, err = filepath.Glob(filepath.Join(s.baseTmplDir, "*.yml.gonja"))
		if err != nil {
			return nil, err
		}
	}

	// Combine, prioritizing tenant files
	allFiles := make(map[string]string) // name -> path
	for _, f := range baseFiles {
		base := filepath.Base(f)
		name := strings.TrimSuffix(strings.TrimSuffix(base, ".gonja"), ".yml")
		allFiles[name] = f
	}
	for _, f := range files {
		base := filepath.Base(f)
		name := strings.TrimSuffix(strings.TrimSuffix(base, ".gonja"), ".yml")
		allFiles[name] = f // Override with tenant file if exists
	}

	res := make(map[string][]byte, len(allFiles))
	for name, f := range allFiles {
		tpl, err := gonja.FromFile(f)
		if err != nil {
			return nil, err
		}
		out, err := tpl.ExecuteToString(s.contextFunctions(name))
		if err != nil {
			return nil, err
		}
		if err := s.enforceHardBinding(name, out); err != nil {
			return nil, err
		}
		res[filepath.Join(s.outDir, name+".yml")] = []byte(out)
	}
	return res, nil
}

// RenderString renders a template string using the service's context for the given cube
func (s *Service) RenderString(cubeName, tpl string) (string, error) {
	t, err := gonja.FromString(tpl)
	if err != nil {
		return "", err
	}
	out, err := t.ExecuteToString(s.contextFunctions(cubeName))
	if err != nil {
		return "", err
	}
	return out, nil
}

func (s *Service) enforceHardBinding(cube string, rendered string) error {
	// Minimal, robust check: ensure data_source is present and approved
	// The template itself must include: data_source: {{ get_data_source("cube") }}
	if !strings.Contains(rendered, "data_source:") {
		return fmt.Errorf("cube %q: missing data_source (hard binding required)", cube)
	}
	// Extract simple yaml key (best-effort). For strictness, also parse YAML in validate.go.
	line := ""
	for _, l := range strings.Split(rendered, "\n") {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "data_source:") {
			line = t
			break
		}
	}
	if line == "" {
		return fmt.Errorf("cube %q: data_source not found", cube)
	}
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("cube %q: malformed data_source line", cube)
	}
	ds := strings.TrimSpace(parts[1])
	if ds == "" {
		return fmt.Errorf("cube %q: empty data_source value", cube)
	}
	ds = strings.Trim(ds, `"'`) // handle quoted
	if _, ok := s.allowedDS[ds]; !ok {
		return fmt.Errorf("cube %q: data_source %q not in allowed set", cube, ds)
	}
	return nil
}

// Helper functions for financial calculations
func joinFloats(values []float64, sep string) string {
	if len(values) == 0 {
		return ""
	}
	result := make([]string, len(values))
	for i, v := range values {
		result[i] = fmt.Sprintf("%.2f", v)
	}
	return strings.Join(result, sep)
}

func joinStrings(values []string, sep string) string {
	return strings.Join(values, sep)
}

func joinMap(values map[string]float64, sep string) string {
	if len(values) == 0 {
		return ""
	}
	result := make([]string, 0, len(values))
	for k, v := range values {
		result = append(result, fmt.Sprintf("%s:%.2f", k, v))
	}
	return strings.Join(result, sep)
}
