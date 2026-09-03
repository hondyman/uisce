package analytics

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/hondyman/uisce/backend/models"
	"github.com/hondyman/uisce/calc-engine/exec"
	"github.com/jmoiron/sqlx"
	"gonum.org/v1/gonum/mat"
)

// Local FinancialCalc type to avoid import cycles
type FinancialCalc struct {
	Type      string                 `json:"type"`
	Formula   string                 `json:"formula,omitempty"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// CashFlow represents a single cash flow
type CashFlow struct {
	Amount float64 `json:"amount"`
	Period int     `json:"period"`
}

// FinancialCalculation represents the interface for financial calculations
type FinancialCalculation interface {
	GetType() string
	GetMu() []float64
	GetCovariance() [][]float64
	GetLongOnly() bool
	GetRiskFreeRate() float64
	GetWeights() []float64
	GetBenchmarkWeights() []float64
	GetReturns() []float64
	GetConfidenceLevel() float64
	GetCashFlows() []CashFlow
	GetGuess() float64
	GetS0() []float64
	GetStrikePrice() float64
	GetTimeHorizon() float64
	GetNumSimulations() int
	GetStartValue() float64
	GetYieldToMaturity() float64
	GetFrequency() int
	GetPoints() int
	GetFormula() string
	GetArguments() map[string]interface{}
	GetEngine() string
	GetExecutionType() string
}

// FinancialCalcAdapter adapts the existing FinancialCalc struct to the FinancialCalculation interface
type FinancialCalcAdapter struct {
	calc interface{} // This will hold the actual FinancialCalc from the api package
}

// NewFinancialCalcAdapter creates a new adapter
func NewFinancialCalcAdapter(calc interface{}) *FinancialCalcAdapter {
	// Normalize incoming calc to a map[string]interface{} when possible so
	// adapter methods can uniformly access fields regardless of whether the
	// caller passed a map (from dynamic JSON) or a concrete struct (from
	// the httpapi package). We avoid importing httpapi here to prevent
	// import cycles and instead use JSON round-trip conversion.
	if _, ok := calc.(map[string]interface{}); !ok {
		if b, err := json.Marshal(calc); err == nil {
			var m map[string]interface{}
			if err2 := json.Unmarshal(b, &m); err2 == nil {
				return &FinancialCalcAdapter{calc: m}
			}
		}
	}
	return &FinancialCalcAdapter{calc: calc}
}

// GetType returns the calculation type
func (f *FinancialCalcAdapter) GetType() string {
	// Try to access the Type field using reflection or type assertion
	if fc, ok := f.calc.(map[string]interface{}); ok {
		if t, ok := fc["type"].(string); ok {
			return t
		}
	}
	// If it's the actual FinancialCalc struct, we need to handle it differently
	// For now, return empty string and let the caller handle it
	return ""
}

// GetMu returns the expected returns vector
func (f *FinancialCalcAdapter) GetMu() []float64 {
	if calc, ok := f.calc.(map[string]interface{}); ok {
		if mu, ok := calc["mu"].([]interface{}); ok {
			var result []float64
			for _, v := range mu {
				if f, ok := v.(float64); ok {
					result = append(result, f)
				}
			}
			return result
		}
		if returns, ok := calc["returns"].([]interface{}); ok {
			var result []float64
			for _, v := range returns {
				if f, ok := v.(float64); ok {
					result = append(result, f)
				}
			}
			return result
		}
	}
	return nil
}

// GetCovariance returns the covariance matrix
func (f *FinancialCalcAdapter) GetCovariance() [][]float64 {
	if calc, ok := f.calc.(map[string]interface{}); ok {
		if cov, ok := calc["covariance"].([]interface{}); ok {
			var result [][]float64
			for _, row := range cov {
				if rowSlice, ok := row.([]interface{}); ok {
					var rowResult []float64
					for _, v := range rowSlice {
						if f, ok := v.(float64); ok {
							rowResult = append(rowResult, f)
						}
					}
					result = append(result, rowResult)
				}
			}
			return result
		}
	}
	return nil
}

// GetLongOnly returns whether long-only constraint is applied
func (f *FinancialCalcAdapter) GetLongOnly() bool {
	if calc, ok := f.calc.(map[string]interface{}); ok {
		if lo, ok := calc["long_only"].(bool); ok {
			return lo
		}
	}
	return false
}

// GetRiskFreeRate returns the risk-free rate
func (f *FinancialCalcAdapter) GetRiskFreeRate() float64 {
	if calc, ok := f.calc.(map[string]interface{}); ok {
		if rfr, ok := calc["risk_free_rate"].(float64); ok {
			return rfr
		}
		if r, ok := calc["r"].(float64); ok {
			return r
		}
	}
	return 0.0
}

// GetWeights returns portfolio weights
func (f *FinancialCalcAdapter) GetWeights() []float64 {
	if calc, ok := f.calc.(map[string]interface{}); ok {
		if weights, ok := calc["weights"].([]interface{}); ok {
			var result []float64
			for _, v := range weights {
				if f, ok := v.(float64); ok {
					result = append(result, f)
				}
			}
			return result
		}
	}
	return nil
}

// GetBenchmarkWeights returns benchmark weights
func (f *FinancialCalcAdapter) GetBenchmarkWeights() []float64 {
	if calc, ok := f.calc.(map[string]interface{}); ok {
		if weights, ok := calc["benchmark_weights"].([]interface{}); ok {
			var result []float64
			for _, v := range weights {
				if f, ok := v.(float64); ok {
					result = append(result, f)
				}
			}
			return result
		}
	}
	return nil
}

// GetReturns returns historical returns
func (f *FinancialCalcAdapter) GetReturns() []float64 {
	if calc, ok := f.calc.(map[string]interface{}); ok {
		if returns, ok := calc["returns"].([]interface{}); ok {
			var result []float64
			for _, v := range returns {
				if f, ok := v.(float64); ok {
					result = append(result, f)
				}
			}
			return result
		}
		if assetReturns, ok := calc["asset_returns"].([]interface{}); ok {
			var result []float64
			for _, v := range assetReturns {
				if f, ok := v.(float64); ok {
					result = append(result, f)
				}
			}
			return result
		}
	}
	return nil
}

// GetConfidenceLevel returns confidence level for VaR
func (f *FinancialCalcAdapter) GetConfidenceLevel() float64 {
	if calc, ok := f.calc.(map[string]interface{}); ok {
		if cl, ok := calc["confidence_level"].(float64); ok {
			return cl
		}
	}
	return 0.95
}

// GetCashFlows returns cash flows
func (f *FinancialCalcAdapter) GetCashFlows() []CashFlow {
	if calc, ok := f.calc.(map[string]interface{}); ok {
		if cfs, ok := calc["cash_flows"].([]interface{}); ok {
			var result []CashFlow
			for _, cf := range cfs {
				if cfMap, ok := cf.(map[string]interface{}); ok {
					amount := 0.0
					period := 0
					if a, ok := cfMap["amount"].(float64); ok {
						amount = a
					}
					if p, ok := cfMap["period"].(float64); ok {
						period = int(p)
					}
					result = append(result, CashFlow{Amount: amount, Period: period})
				}
			}
			return result
		}
	}
	return nil
}

// GetGuess returns initial guess for IRR
func (f *FinancialCalcAdapter) GetGuess() float64 {
	if calc, ok := f.calc.(map[string]interface{}); ok {
		if guess, ok := calc["guess"].(float64); ok {
			return guess
		}
	}
	return 0.1
}

// GetS0 returns initial stock prices
func (f *FinancialCalcAdapter) GetS0() []float64 {
	if calc, ok := f.calc.(map[string]interface{}); ok {
		if s0, ok := calc["S0"].([]interface{}); ok {
			var result []float64
			for _, v := range s0 {
				if f, ok := v.(float64); ok {
					result = append(result, f)
				}
			}
			return result
		}
		if initialValues, ok := calc["initial_values"].([]interface{}); ok {
			var result []float64
			for _, v := range initialValues {
				if f, ok := v.(float64); ok {
					result = append(result, f)
				}
			}
			return result
		}
	}
	return nil
}

// GetStrikePrice returns strike price
func (f *FinancialCalcAdapter) GetStrikePrice() float64 {
	if calc, ok := f.calc.(map[string]interface{}); ok {
		if sp, ok := calc["strike_price"].(float64); ok {
			return sp
		}
		if strike, ok := calc["strike"].(float64); ok {
			return strike
		}
	}
	return 0.0
}

// GetTimeHorizon returns time horizon
func (f *FinancialCalcAdapter) GetTimeHorizon() float64 {
	if calc, ok := f.calc.(map[string]interface{}); ok {
		if th, ok := calc["time_horizon"].(float64); ok {
			return th
		}
		if t, ok := calc["T"].(float64); ok {
			return t
		}
	}
	return 0.0
}

// GetNumSimulations returns number of simulations
func (f *FinancialCalcAdapter) GetNumSimulations() int {
	if calc, ok := f.calc.(map[string]interface{}); ok {
		if ns, ok := calc["num_simulations"].(float64); ok {
			return int(ns)
		}
		if sims, ok := calc["sims"].(float64); ok {
			return int(sims)
		}
	}
	return 1000
}

// GetStartValue returns start value
func (f *FinancialCalcAdapter) GetStartValue() float64 {
	if calc, ok := f.calc.(map[string]interface{}); ok {
		if sv, ok := calc["start_value"].(float64); ok {
			return sv
		}
	}
	return 0.0
}

// GetYieldToMaturity returns yield to maturity
func (f *FinancialCalcAdapter) GetYieldToMaturity() float64 {
	if calc, ok := f.calc.(map[string]interface{}); ok {
		if ytm, ok := calc["yield_to_maturity"].(float64); ok {
			return ytm
		}
	}
	return 0.0
}

// GetFrequency returns payment frequency
func (f *FinancialCalcAdapter) GetFrequency() int {
	if calc, ok := f.calc.(map[string]interface{}); ok {
		if freq, ok := calc["frequency"].(float64); ok {
			return int(freq)
		}
	}
	return 1
}

// GetPoints returns number of points for efficient frontier
func (f *FinancialCalcAdapter) GetPoints() int {
	if calc, ok := f.calc.(map[string]interface{}); ok {
		if points, ok := calc["points"].(float64); ok {
			return int(points)
		}
	}
	return 50
}

// GetFormula returns the Excel formula
func (f *FinancialCalcAdapter) GetFormula() string {
	if calc, ok := f.calc.(map[string]interface{}); ok {
		if formula, ok := calc["formula"].(string); ok {
			return formula
		}
	}
	return ""
}

// GetArguments returns the formula arguments
func (f *FinancialCalcAdapter) GetArguments() map[string]interface{} {
	if calc, ok := f.calc.(map[string]interface{}); ok {
		if args, ok := calc["arguments"].(map[string]interface{}); ok {
			return args
		}
	}
	return nil
}

// GetEngine returns the execution engine
func (f *FinancialCalcAdapter) GetEngine() string {
	if calc, ok := f.calc.(map[string]interface{}); ok {
		if engine, ok := calc["engine"].(string); ok {
			return engine
		}
	}
	return "internal"
}

// GetExecutionType returns the execution type (realtime/batch)
func (f *FinancialCalcAdapter) GetExecutionType() string {
	if calc, ok := f.calc.(map[string]interface{}); ok {
		if execType, ok := calc["execution_type"].(string); ok {
			return execType
		}
	}
	return "realtime"
}

// SemanticCalculationService provides semantic interpretation and execution of financial calculations
type SemanticCalculationService struct {
	db      *sqlx.DB
	monitor *ExecutionMonitorService
}

// NewSemanticCalculationService creates a new semantic calculation service
func NewSemanticCalculationService(db *sqlx.DB) *SemanticCalculationService {
	return &SemanticCalculationService{
		db:      db,
		monitor: NewExecutionMonitorService(db),
	}
}

// GetDB returns the underlying database connection
func (s *SemanticCalculationService) GetDB() *sqlx.DB {
	return s.db
}

// GetCalculationByName retrieves a calculation definition from the database
func (s *SemanticCalculationService) GetCalculationByName(name string) (*models.Calculation, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database connection not initialized")
	}
	var calc models.Calculation
	err := s.db.Get(&calc, "SELECT * FROM calculations WHERE name = $1", name)
	if err != nil {
		return nil, err
	}
	return &calc, nil
}

// resolveTier parses calc.Formula and resolves its execution tier via the
// centralized calc engine (boresolver.ResolveTier), storing the result on
// calc.Tier. This is what makes Tier a persisted single source of truth
// rather than something every caller re-derives: Create/Update compute it
// once at save time, and the execute endpoint just reads it back.
//
// calc.ExecutionPreference ("auto"/"pushdown"/"host_runtime") defaults to
// "auto" when unset. A parse failure or an explicit "pushdown" preference
// that the formula can't satisfy is returned as an error — a calc must not
// be saved with an unresolvable tier.
func resolveTier(calc *models.Calculation) error {
	if calc.Formula == "" {
		calc.Tier = boresolver.TierPushdown.String()
		return nil
	}

	expr, err := boresolver.ParseCalcFormula(calc.Formula)
	if err != nil {
		return fmt.Errorf("invalid formula: %w", err)
	}

	var pref boresolver.ExecutionPreference
	switch calc.ExecutionPreference {
	case "", "auto":
		calc.ExecutionPreference = "auto"
		pref = boresolver.PreferAuto
	case "pushdown":
		pref = boresolver.PreferPushdown
	case "host_runtime":
		pref = boresolver.PreferHostRuntime
	default:
		return fmt.Errorf("invalid execution_preference %q (must be auto, pushdown, or host_runtime)", calc.ExecutionPreference)
	}

	tier, err := boresolver.ResolveTier(expr, boresolver.PostgresDialect{}, pref)
	if err != nil {
		return err
	}
	calc.Tier = tier.String()
	return nil
}

// CreateCalculation creates a new calculation definition in the database
func (s *SemanticCalculationService) CreateCalculation(calc *models.Calculation) error {
	calc.ID = uuid.New()
	calc.CreatedAt = time.Now()
	calc.UpdatedAt = time.Now()

	if err := resolveTier(calc); err != nil {
		return err
	}

	query := `
		INSERT INTO calculations (
			id, node_id, name, title, description, formula, engine_type, return_type, arguments, category, subcategory, domain_id, execution_type, engine, is_materialized, tier, execution_preference, created_at, updated_at
		) VALUES (
			:id, :node_id, :name, :title, :description, :formula, :engine_type, :return_type, :arguments, :category, :subcategory, :domain_id, :execution_type, :engine, :is_materialized, :tier, :execution_preference, :created_at, :updated_at
		)
	`
	_, err := s.db.NamedExec(query, calc)
	return err
}

// UpdateCalculation updates an existing calculation definition
func (s *SemanticCalculationService) UpdateCalculation(calc *models.Calculation) error {
	calc.UpdatedAt = time.Now()

	if err := resolveTier(calc); err != nil {
		return err
	}

	query := `
		UPDATE calculations SET
			name = :name,
			title = :title,
			description = :description,
			formula = :formula,
			engine_type = :engine_type,
			return_type = :return_type,
			arguments = :arguments,
			category = :category,
			subcategory = :subcategory,
			domain_id = :domain_id,
			execution_type = :execution_type,
			engine = :engine,
			is_materialized = :is_materialized,
			tier = :tier,
			execution_preference = :execution_preference,
			updated_at = :updated_at
		WHERE id = :id
	`
	_, err := s.db.NamedExec(query, calc)
	return err
}

// ListCalculations retrieves all calculation definitions from the database
func (s *SemanticCalculationService) ListCalculations() ([]models.Calculation, error) {
	var calcs []models.Calculation
	query := `SELECT * FROM calculations ORDER BY name`
	err := s.db.Select(&calcs, query)
	if err != nil {
		return nil, err
	}
	return calcs, nil
}

// BuildCalcGraph recursively resolves a calc's dependency chain from
// STORED calculations into a boresolver.CalcGraph: any term the formula
// references that matches another calculation's Name is pulled in as a
// nested calc (its own formula included, recursively); anything else is
// treated as a base field (IsBaseField=true, no formula — resolved by
// whatever runs the graph, e.g. SQLRowSource for host-runtime nodes or the
// base query layer for pushdown nodes).
//
// This is what makes "calc in a calc" work at the persistence layer, not
// just when a caller hand-builds a CalcGraph in Go — the centralized calc
// engine's execute path (CalculationHandler.Execute) calls this to compile
// and run any calc's full dependency chain in one pass.
func (s *SemanticCalculationService) BuildCalcGraph(root *models.Calculation) (*boresolver.CalcGraph, error) {
	graph := boresolver.NewCalcGraph()
	visited := make(map[string]bool)

	var visit func(calc *models.Calculation) error
	visit = func(calc *models.Calculation) error {
		if visited[calc.Name] {
			return nil
		}
		visited[calc.Name] = true

		expr, err := boresolver.ParseCalcFormula(calc.Formula)
		if err != nil {
			return fmt.Errorf("calc %q: %w", calc.Name, err)
		}

		var deps []string
		for _, ref := range boresolver.CollectTermRefs(expr) {
			deps = append(deps, ref)
			if visited[ref] {
				continue
			}
			dep, err := s.GetCalculationByName(ref)
			if err != nil {
				if !errors.Is(err, sql.ErrNoRows) {
					return fmt.Errorf("failed to resolve dependency %q of calc %q: %w", ref, calc.Name, err)
				}
				// No calc with this name -> a base field, resolved elsewhere.
				graph.AddNode(&boresolver.CalcNode{TermKey: ref, IsBaseField: true})
				visited[ref] = true
				continue
			}
			if err := visit(dep); err != nil {
				return err
			}
		}

		var pref boresolver.ExecutionPreference
		switch calc.ExecutionPreference {
		case "pushdown":
			pref = boresolver.PreferPushdown
		case "host_runtime":
			pref = boresolver.PreferHostRuntime
		default:
			pref = boresolver.PreferAuto
		}

		graph.AddNode(&boresolver.CalcNode{
			TermKey:      calc.Name,
			Formula:      calc.Formula,
			Dependencies: deps,
			Preference:   pref,
		})
		return nil
	}

	if err := visit(root); err != nil {
		return nil, err
	}
	return graph, nil
}

// ExecuteFinancialCalc executes a financial calculation using semantic interpretation
// This function can be called from the dispatch function to route through the semantic layer
func ExecuteFinancialCalc(calc interface{}, db *sqlx.DB) (interface{}, error) {
	service := &SemanticCalculationService{db: db}

	// If calc is a string, treat it as a calculation name and look it up
	if name, ok := calc.(string); ok {
		dbCalc, err := service.GetCalculationByName(name)
		if err != nil {
			return nil, fmt.Errorf("failed to find calculation '%s': %w", name, err)
		}
		// Convert DB model to map for adapter
		// This assumes the 'Arguments' JSONB matches what the adapter expects
		calcMap := map[string]interface{}{
			"type":      dbCalc.Formula, // Or EngineType? Need to align.
			"arguments": dbCalc.Arguments,
		}
		// For now, let's assume 'Formula' holds the type if it's a standard financial calc,
		// or we need a mapping. The migration said 'formula' is the actual formula.
		// But FinancialCalcAdapter expects 'type'.
		// Let's use the 'Category' or a new field 'Algorithm' if needed.
		// For this iteration, let's assume the input 'calc' is the full definition if not a string.
		return service.ExecuteCalculation(NewFinancialCalcAdapter(calcMap))
	}

	adapter := NewFinancialCalcAdapter(calc)
	return service.ExecuteCalculation(adapter)
}

// ExecuteVectorizedExcelCalc and ExecuteVectorizedExcelCalculation were
// removed (2026-09) — they were dead code (zero real callers; confirmed via
// impact analysis and a repo-wide text search) built on placeholder data
// sources (getMetricDefinition/getEntityData explicitly returned canned
// sample data, never real entity data). Running a formula across many
// entities in one batch is now a real, tested capability:
// boresolver.HostRuntimeExecutor.Execute already evaluates a calc across
// every entity a RowSource returns in one batched query (see
// boresolver.SQLRowSource), and datapipeline.HostRuntimeCalcTransformer
// wraps the same executor for scheduled/precalc batch runs.

// ExecuteCalculation dispatches a FIXED quant model (portfolio
// optimization, Black-Scholes, Monte Carlo, ...) by its FinancialCalculation
// interface — typed vector/matrix inputs, not a stored formula. This is
// deliberately a separate system from ExecuteFormulaCalculation
// (execute_calculation.go), which runs formula-driven models.Calculation
// rows through the tiered boresolver calc engine: quant models here aren't
// user-authorable strings and don't belong in the AST/tier-resolution
// path.
func (s *SemanticCalculationService) ExecuteCalculation(calc FinancialCalculation) (interface{}, error) {
	return s.ExecuteCalculationWithContext(calc, nil)
}

// ExecuteCalculationWithContext executes a calculation with additional context (e.g. argument mapping)
func (s *SemanticCalculationService) ExecuteCalculationWithContext(calc FinancialCalculation, mapping map[string]string) (interface{}, error) {
	// Log start of execution
	var logID uuid.UUID
	if s.monitor != nil {
		payload, _ := json.Marshal(calc)
		log := MonitorExecutionLog{
			EventType: "calculation",
			Engine:    calc.GetEngine(),
			Payload:   payload,
		}
		if log.Engine == "" {
			log.Engine = "internal"
		}
		logID, _ = s.monitor.LogStart(context.Background(), log)
	}

	var result interface{}
	var err error

	defer func() {
		if s.monitor != nil && logID != uuid.Nil {
			if err != nil {
				s.monitor.LogFailure(context.Background(), logID, err.Error())
			} else {
				resJSON, _ := json.Marshal(result)
				s.monitor.LogCompletion(context.Background(), logID, resJSON)
			}
		}
	}()
	// Routing logic based on engine
	engine := calc.GetEngine()
	if engine == "cube" {
		return s.executeCubeCalculation(calc, mapping)
	} else if engine == "spark" {
		return s.executeSparkCalculation(calc)
	}

	// Default to internal semantic interpretation layer
	switch calc.GetType() {
	case "markowitz":
		return s.executePortfolioOptimization(calc)
	case "efficient_frontier":
		return s.executeEfficientFrontierAnalysis(calc)
	case "tangency":
		return s.executeTangencyPortfolio(calc)
	case "tracking_error":
		return s.executeTrackingErrorAnalysis(calc)
	case "var_historical":
		return s.executeRiskAnalytics(calc)
	case "black_scholes":
		return s.executeDerivativePricing(calc)
	case "gbm":
		return s.executeAssetSimulation(calc)
	case "monte_carlo":
		return s.executeProbabilisticAnalysis(calc)
	case "duration_convexity":
		return s.executeFixedIncomeAnalytics(calc)
	case "irr":
		return s.executeCashFlowAnalysis(calc)
	case "excel_formula":
		return s.executeFormulaViaCalcEngine(calc)
	default:
		return nil, fmt.Errorf("unsupported calculation type: %s", calc.GetType())
	}
}

// executePortfolioOptimization handles portfolio optimization with semantic understanding
func (s *SemanticCalculationService) executePortfolioOptimization(calc FinancialCalculation) (interface{}, error) {
	// Semantic validation
	if len(calc.GetMu()) == 0 {
		return nil, fmt.Errorf("portfolio optimization requires expected returns (mu) - this represents the anticipated performance of each asset")
	}

	if len(calc.GetCovariance()) == 0 {
		return nil, fmt.Errorf("portfolio optimization requires covariance matrix - this represents how assets move together")
	}

	// Convert to calculation engine format
	n := len(calc.GetMu())
	Sigma := mat.NewDense(n, n, nil)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			Sigma.Set(i, j, calc.GetCovariance()[i][j])
		}
	}

	// Execute optimization
	opts := exec.MarkowitzOpts{
		MaxWeight: 1.0,
		MinWeight: 0.0,
	}
	if calc.GetLongOnly() {
		opts.MinWeight = 0.0
	} else {
		opts.MinWeight = -1.0
	}

	weights, err := exec.MarkowitzOptimize(calc.GetMu(), Sigma, opts, calc.GetRiskFreeRate())
	if err != nil {
		return nil, fmt.Errorf("portfolio optimization failed: %w", err)
	}

	// Semantic enrichment - add business context
	result := map[string]interface{}{
		"weights":              weights,
		"expected_return":      s.calculatePortfolioReturn(calc.GetMu(), weights),
		"portfolio_volatility": s.calculatePortfolioVolatility(Sigma, weights),
		"sharpe_ratio":         s.calculateSharpeRatio(calc.GetMu(), Sigma, weights, calc.GetRiskFreeRate()),
		"business_context":     "Optimal portfolio allocation maximizes risk-adjusted returns",
	}

	return result, nil
}

// executeEfficientFrontierAnalysis handles efficient frontier analysis
func (s *SemanticCalculationService) executeEfficientFrontierAnalysis(calc FinancialCalculation) (interface{}, error) {
	if len(calc.GetMu()) == 0 {
		return nil, fmt.Errorf("efficient frontier analysis requires expected returns")
	}

	n := len(calc.GetMu())
	Sigma := mat.NewDense(n, n, nil)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			Sigma.Set(i, j, calc.GetCovariance()[i][j])
		}
	}

	points := calc.GetPoints()
	if points == 0 {
		points = 50
	}

	sols, err := exec.EfficientFrontier(calc.GetMu(), Sigma, calc.GetRiskFreeRate(), points, calc.GetLongOnly())
	if err != nil {
		return nil, fmt.Errorf("efficient frontier analysis failed: %w", err)
	}

	return map[string]interface{}{
		"frontier_points":  sols,
		"business_context": "Efficient frontier shows optimal risk-return combinations",
		"interpretation":   "Each point represents a portfolio with maximum return for given risk level",
	}, nil
}

// executeTangencyPortfolio handles tangency portfolio calculation
func (s *SemanticCalculationService) executeTangencyPortfolio(calc FinancialCalculation) (interface{}, error) {
	if len(calc.GetMu()) == 0 {
		return nil, fmt.Errorf("tangency portfolio requires expected returns")
	}

	n := len(calc.GetMu())
	Sigma := mat.NewDense(n, n, nil)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			Sigma.Set(i, j, calc.GetCovariance()[i][j])
		}
	}

	weights, err := exec.TangencyPortfolio(calc.GetMu(), Sigma, calc.GetRiskFreeRate(), calc.GetLongOnly())
	if err != nil {
		return nil, fmt.Errorf("tangency portfolio calculation failed: %w", err)
	}

	return map[string]interface{}{
		"weights":          weights,
		"business_context": "Tangency portfolio offers highest Sharpe ratio",
		"interpretation":   "This portfolio provides optimal risk-adjusted returns given the risk-free rate",
	}, nil
}

// executeTrackingErrorAnalysis handles tracking error analysis
func (s *SemanticCalculationService) executeTrackingErrorAnalysis(calc FinancialCalculation) (interface{}, error) {
	if len(calc.GetMu()) == 0 || len(calc.GetBenchmarkWeights()) == 0 {
		return nil, fmt.Errorf("tracking error analysis requires asset returns and benchmark weights")
	}

	n := len(calc.GetMu())
	Sigma := mat.NewDense(n, n, nil)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			Sigma.Set(i, j, calc.GetCovariance()[i][j])
		}
	}

	w := calc.GetWeights()
	if len(w) == 0 {
		w = make([]float64, len(calc.GetMu()))
		for i := range w {
			w[i] = 1.0 / float64(len(calc.GetMu()))
		}
	}

	trackingError := s.calculateTrackingError(Sigma, w, calc.GetBenchmarkWeights())

	return map[string]interface{}{
		"tracking_error":   trackingError,
		"business_context": "Tracking error measures portfolio deviation from benchmark",
		"interpretation":   "Lower tracking error indicates closer benchmark replication",
	}, nil
}

// executeRiskAnalytics handles risk analytics with semantic interpretation
func (s *SemanticCalculationService) executeRiskAnalytics(calc FinancialCalculation) (interface{}, error) {
	if len(calc.GetReturns()) == 0 {
		return nil, fmt.Errorf("risk analytics requires historical returns data")
	}

	confidence := calc.GetConfidenceLevel()
	if confidence == 0 {
		confidence = 0.95 // default 95% confidence
	}

	// Calculate VaR using historical simulation
	returns := append([]float64{}, calc.GetReturns()...)

	var99 := s.calculateHistoricalVaR(returns, 0.99)
	var95 := s.calculateHistoricalVaR(returns, 0.95)

	return map[string]interface{}{
		"var_99":           var99,
		"var_95":           var95,
		"confidence_level": confidence,
		"business_context": "Value at Risk quantifies potential portfolio losses",
		"interpretation":   "99% VaR means 99% confidence that losses won't exceed this amount",
	}, nil
}

// executeDerivativePricing handles derivative pricing
func (s *SemanticCalculationService) executeDerivativePricing(calc FinancialCalculation) (interface{}, error) {
	if len(calc.GetS0()) == 0 {
		return nil, fmt.Errorf("derivative pricing requires underlying asset prices")
	}

	// This would integrate with Black-Scholes calculation engine
	// For now, return semantic context
	return map[string]interface{}{
		"business_context": "Derivative pricing for hedging and investment strategies",
		"interpretation":   "Black-Scholes model values options based on underlying asset dynamics",
		"parameters": map[string]interface{}{
			"underlying_price": calc.GetS0(),
			"strike_price":     calc.GetStrikePrice(),
			"time_to_maturity": calc.GetTimeHorizon(),
			"risk_free_rate":   calc.GetRiskFreeRate(),
			"volatility":       calc.GetMu(), // Using Mu as volatility for now
		},
	}, nil
}

// executeAssetSimulation handles asset price simulation
func (s *SemanticCalculationService) executeAssetSimulation(calc FinancialCalculation) (interface{}, error) {
	if len(calc.GetS0()) == 0 {
		return nil, fmt.Errorf("asset simulation requires initial prices")
	}

	return map[string]interface{}{
		"business_context": "Stochastic simulation for scenario analysis and risk assessment",
		"interpretation":   "Geometric Brownian Motion models realistic asset price movements",
		"simulation_parameters": map[string]interface{}{
			"initial_price": calc.GetS0(),
			"drift":         calc.GetMu(),
			"volatility":    calc.GetMu(), // Using Mu as volatility for now
			"time_horizon":  calc.GetTimeHorizon(),
			"steps":         calc.GetPoints(), // Using Points as steps for now
		},
	}, nil
}

// executeProbabilisticAnalysis handles Monte Carlo analysis
func (s *SemanticCalculationService) executeProbabilisticAnalysis(calc FinancialCalculation) (interface{}, error) {
	return map[string]interface{}{
		"business_context": "Probabilistic analysis for complex financial instruments",
		"interpretation":   "Monte Carlo simulation provides distribution of possible outcomes",
		"analysis_parameters": map[string]interface{}{
			"simulations":    calc.GetNumSimulations(),
			"start_value":    calc.GetStartValue(),
			"strike_price":   calc.GetStrikePrice(),
			"risk_free_rate": calc.GetRiskFreeRate(),
			"volatility":     calc.GetMu(), // Using Mu as volatility for now
			"time_horizon":   calc.GetTimeHorizon(),
		},
	}, nil
}

// executeFixedIncomeAnalytics handles fixed income analysis
func (s *SemanticCalculationService) executeFixedIncomeAnalytics(calc FinancialCalculation) (interface{}, error) {
	if len(calc.GetCashFlows()) == 0 {
		return nil, fmt.Errorf("fixed income analysis requires cash flow schedule")
	}

	return map[string]interface{}{
		"business_context": "Fixed income risk management and yield optimization",
		"interpretation":   "Duration and convexity measure interest rate risk",
		"analysis_parameters": map[string]interface{}{
			"cash_flows":        calc.GetCashFlows(),
			"yield_to_maturity": calc.GetYieldToMaturity(),
			"frequency":         calc.GetFrequency(),
		},
	}, nil
}

// executeCashFlowAnalysis handles IRR and cash flow analysis
func (s *SemanticCalculationService) executeCashFlowAnalysis(calc FinancialCalculation) (interface{}, error) {
	if len(calc.GetCashFlows()) == 0 {
		return nil, fmt.Errorf("cash flow analysis requires cash flow data")
	}

	flows := make([]float64, len(calc.GetCashFlows()))
	for i, cf := range calc.GetCashFlows() {
		flows[i] = cf.Amount
	}

	irr := s.calculateIRR(flows, calc.GetGuess())

	return map[string]interface{}{
		"irr":              irr,
		"business_context": "Internal Rate of Return measures investment profitability",
		"interpretation":   "IRR is the discount rate that makes NPV zero",
		"cash_flows":       calc.GetCashFlows(),
	}, nil
}

// Helper methods for semantic calculations

func (s *SemanticCalculationService) calculatePortfolioReturn(mu []float64, weights []float64) float64 {
	var expectedReturn float64
	for i, weight := range weights {
		expectedReturn += weight * mu[i]
	}
	return expectedReturn
}

func (s *SemanticCalculationService) calculatePortfolioVolatility(sigma *mat.Dense, weights []float64) float64 {
	var variance float64
	for i, wi := range weights {
		for j, wj := range weights {
			variance += wi * wj * sigma.At(i, j)
		}
	}
	return math.Sqrt(variance)
}

func (s *SemanticCalculationService) calculateSharpeRatio(mu []float64, sigma *mat.Dense, weights []float64, riskFreeRate float64) float64 {
	expectedReturn := s.calculatePortfolioReturn(mu, weights)
	volatility := s.calculatePortfolioVolatility(sigma, weights)
	if volatility == 0 {
		return 0
	}
	return (expectedReturn - riskFreeRate) / volatility
}

func (s *SemanticCalculationService) calculateTrackingError(sigma *mat.Dense, portfolioWeights []float64, benchmarkWeights []float64) float64 {
	var variance float64
	for i, pi := range portfolioWeights {
		for j, pj := range portfolioWeights {
			bi := benchmarkWeights[i]
			bj := benchmarkWeights[j]
			variance += (pi - bi) * (pj - bj) * sigma.At(i, j)
		}
	}
	return math.Sqrt(variance)
}

func (s *SemanticCalculationService) calculateHistoricalVaR(returns []float64, confidence float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	// Sort returns in ascending order (worst to best)
	sortedReturns := make([]float64, len(returns))
	copy(sortedReturns, returns)

	for i := 0; i < len(sortedReturns); i++ {
		for j := i + 1; j < len(sortedReturns); j++ {
			if sortedReturns[i] > sortedReturns[j] {
				sortedReturns[i], sortedReturns[j] = sortedReturns[j], sortedReturns[i]
			}
		}
	}

	// Find the VaR at the specified confidence level
	index := int(float64(len(sortedReturns)) * (1 - confidence))
	if index >= len(sortedReturns) {
		index = len(sortedReturns) - 1
	}

	return -sortedReturns[index] // Negative because we want the loss amount
}

func (s *SemanticCalculationService) calculateIRR(cashFlows []float64, guess float64) float64 {
	if len(cashFlows) == 0 {
		return 0
	}

	if guess == 0 {
		guess = 0.1 // 10% initial guess
	}

	// Simple IRR calculation using Newton-Raphson method
	rate := guess
	maxIterations := 100
	tolerance := 1e-6

	for i := 0; i < maxIterations; i++ {
		npv := 0.0
		dnpv := 0.0

		for t, cf := range cashFlows {
			npv += cf / math.Pow(1+rate, float64(t))
			if t > 0 {
				dnpv -= float64(t) * cf / math.Pow(1+rate, float64(t+1))
			}
		}

		if math.Abs(npv) < tolerance {
			return rate
		}

		if dnpv != 0 {
			rate = rate - npv/dnpv
		} else {
			break
		}
	}

	return rate
}


// executeFormulaViaCalcEngine evaluates an "excel_formula" calculation
// through the real boresolver calc engine (finlib-backed for xirr/irr,
// dialect-aware for everything else) instead of the hand-rolled
// evaluateXIRR/evaluateNPV/... stubs this replaced (2026-09) — those were
// dead code (zero real callers) with a strictly worse XIRR ("for
// simplicity, use the existing IRR calculation" — i.e. treated irregular
// dates as equally-spaced periods) than finlib.XIRR's actual Actual/365
// date-weighted solve.
//
// Arguments are expected as parallel arrays keyed by term name (e.g.
// {"cash_flows": [...], "dates": [...]}) — one row per index, scalar
// arguments broadcast to every row — matching boresolver.CalcRow, so the
// same functions registered for the tenant-facing formula engine (see
// calc_functions.go) work here too.
func (s *SemanticCalculationService) executeFormulaViaCalcEngine(calc FinancialCalculation) (interface{}, error) {
	formula := calc.GetFormula()
	if formula == "" {
		return nil, fmt.Errorf("excel formula is required")
	}

	expr, err := boresolver.ParseCalcFormula(formula)
	if err != nil {
		return nil, fmt.Errorf("failed to parse formula: %w", err)
	}

	arguments := calc.GetArguments()
	rows, err := argumentsToCalcRows(arguments)
	if err != nil {
		return nil, fmt.Errorf("failed to interpret arguments: %w", err)
	}

	value, err := boresolver.EvalHostExpr(expr, rows, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate formula: %w", err)
	}

	return map[string]interface{}{
		"result":           value,
		"formula":          formula,
		"arguments":        arguments,
		"calculation_type": "excel_formula",
	}, nil
}

// argumentsToCalcRows converts a flat arguments map (as produced by
// FinancialCalculation.GetArguments) into boresolver.CalcRow series:
// array-valued arguments become one value per row (all arrays must share
// the same length), scalar-valued arguments are broadcast to every row.
func argumentsToCalcRows(args map[string]interface{}) ([]boresolver.CalcRow, error) {
	seriesLen := -1
	series := make(map[string][]interface{})
	scalars := make(map[string]interface{})

	for k, v := range args {
		if arr, ok := v.([]interface{}); ok {
			if seriesLen == -1 {
				seriesLen = len(arr)
			} else if len(arr) != seriesLen {
				return nil, fmt.Errorf("argument %q has length %d, expected %d (all array arguments must be the same length)", k, len(arr), seriesLen)
			}
			series[k] = arr
			continue
		}
		scalars[k] = v
	}
	if seriesLen == -1 {
		seriesLen = 1 // no array arguments at all -- a single scalar "row"
	}

	rows := make([]boresolver.CalcRow, seriesLen)
	for i := 0; i < seriesLen; i++ {
		row := make(boresolver.CalcRow, len(args))
		for k, v := range scalars {
			row[k] = v
		}
		for k, arr := range series {
			row[k] = arr[i]
		}
		rows[i] = row
	}
	return rows, nil
}
