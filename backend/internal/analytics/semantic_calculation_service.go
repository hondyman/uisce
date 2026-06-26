package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/models"
	"github.com/hondyman/semlayer/calc-engine/exec"
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

// CreateCalculation creates a new calculation definition in the database
func (s *SemanticCalculationService) CreateCalculation(calc *models.Calculation) error {
	calc.ID = uuid.New()
	calc.CreatedAt = time.Now()
	calc.UpdatedAt = time.Now()

	query := `
		INSERT INTO calculations (
			id, node_id, name, title, description, formula, engine_type, return_type, arguments, category, subcategory, domain_id, execution_type, engine, is_materialized, created_at, updated_at
		) VALUES (
			:id, :node_id, :name, :title, :description, :formula, :engine_type, :return_type, :arguments, :category, :subcategory, :domain_id, :execution_type, :engine, :is_materialized, :created_at, :updated_at
		)
	`
	_, err := s.db.NamedExec(query, calc)
	return err
}

// UpdateCalculation updates an existing calculation definition
func (s *SemanticCalculationService) UpdateCalculation(calc *models.Calculation) error {
	calc.UpdatedAt = time.Now()
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

// ExecuteVectorizedExcelCalc executes Excel formulas across multiple entities in batch
func ExecuteVectorizedExcelCalc(metrics []string, entities []string, db *sqlx.DB) (map[string]map[string]interface{}, error) {
	service := &SemanticCalculationService{db: db}
	return service.ExecuteVectorizedExcelCalculation(metrics, entities)
}

// ExecuteVectorizedExcelCalculation handles batch Excel formula execution across multiple metrics and entities
func (s *SemanticCalculationService) ExecuteVectorizedExcelCalculation(metrics []string, entities []string) (map[string]map[string]interface{}, error) {
	results := make(map[string]map[string]interface{})

	// For each metric, execute across all entities
	for _, metricID := range metrics {
		metricResults := make(map[string]interface{})

		// Get metric definition (this would come from your registry)
		metricDef, err := s.getMetricDefinition(metricID)
		if err != nil {
			return nil, fmt.Errorf("failed to get metric definition for %s: %w", metricID, err)
		}

		// Check if this is an Excel-based metric
		if metricDef.FinancialCalc == nil || metricDef.FinancialCalc.Type != "excel_formula" {
			continue // Skip non-Excel metrics
		}

		// Build vectorized arguments for all entities
		vectorizedArgs := make([]map[string]interface{}, len(entities))

		for i, entityID := range entities {
			// Get entity data (this would come from your data layer)
			entityData, err := s.getEntityData(entityID)
			if err != nil {
				metricResults[entityID] = map[string]interface{}{
					"error": fmt.Sprintf("failed to get entity data: %v", err),
				}
				continue
			}

			// Resolve arguments for this entity
			resolvedArgs, err := s.resolveArgumentsForEntity(metricDef.FinancialCalc, entityData)
			if err != nil {
				metricResults[entityID] = map[string]interface{}{
					"error": fmt.Sprintf("failed to resolve arguments: %v", err),
				}
				continue
			}

			vectorizedArgs[i] = resolvedArgs
		}

		// Execute vectorized calculation
		batchResults, err := s.executeVectorizedExcelFormula(metricDef.FinancialCalc.Formula, vectorizedArgs)
		if err != nil {
			return nil, fmt.Errorf("failed to execute vectorized Excel formula for metric %s: %w", metricID, err)
		}

		// Map results back to entity IDs
		for i, entityID := range entities {
			if i < len(batchResults) {
				metricResults[entityID] = batchResults[i]
			} else {
				metricResults[entityID] = map[string]interface{}{
					"error": "result index out of bounds",
				}
			}
		}

		results[metricID] = metricResults
	}

	return results, nil
}

// ExecuteCalculation interprets the business intent and executes the appropriate calculation
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
		return s.executeExcelFormula(calc)
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

// executeExcelFormula handles Excel formula evaluation with vectorized support
func (s *SemanticCalculationService) executeExcelFormula(calc FinancialCalculation) (interface{}, error) {
	// Get formula and arguments from the calculation
	formula := calc.GetFormula()
	arguments := calc.GetArguments()

	if formula == "" {
		return nil, fmt.Errorf("excel formula is required")
	}

	// Check if this is a vectorized request (contains arrays of argument sets)
	if vectorizedArgs, isVectorized := s.detectVectorizedArguments(arguments); isVectorized {
		// Execute formula for each argument set in the batch
		results, err := s.executeVectorizedExcelFormula(formula, vectorizedArgs)
		if err != nil {
			return nil, fmt.Errorf("failed to execute vectorized Excel formula: %w", err)
		}

		// Return batch results with metadata
		return map[string]interface{}{
			"results":          results,
			"formula":          formula,
			"arguments":        arguments,
			"calculation_type": "excel_formula_vectorized",
			"batch_size":       len(results),
			"business_context": "Vectorized Excel-based financial calculations",
		}, nil
	}

	// Single formula evaluation (existing logic)
	result, err := s.evaluateSimpleExcelFormula(formula, arguments)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate Excel formula: %w", err)
	}

	// Return result with metadata
	return map[string]interface{}{
		"result":           result,
		"formula":          formula,
		"arguments":        arguments,
		"calculation_type": "excel_formula",
		"business_context": "Excel-based financial calculation",
	}, nil
}

// evaluateSimpleExcelFormula provides basic Excel function evaluation
func (s *SemanticCalculationService) evaluateSimpleExcelFormula(formula string, arguments map[string]interface{}) (float64, error) {
	// Remove the = prefix if present
	if len(formula) > 0 && formula[0] == '=' {
		formula = formula[1:]
	}

	// Parse function name and arguments
	if len(formula) < 4 {
		return 0, fmt.Errorf("invalid formula format")
	}

	// Simple implementations for common Excel functions
	switch {
	case strings.HasPrefix(formula, "XIRR("):
		return s.evaluateXIRR(arguments)
	case strings.HasPrefix(formula, "NPV("):
		return s.evaluateNPV(arguments)
	case strings.HasPrefix(formula, "IRR("):
		return s.evaluateIRR(arguments)
	case strings.HasPrefix(formula, "PV("):
		return s.evaluatePV(arguments)
	case strings.HasPrefix(formula, "FV("):
		return s.evaluateFV(arguments)
	case strings.HasPrefix(formula, "PMT("):
		return s.evaluatePMT(arguments)
	case strings.HasPrefix(formula, "MIRR("):
		return s.evaluateMIRR(arguments)
	default:
		return 0, fmt.Errorf("unsupported Excel function: %s", formula)
	}
}

// evaluateXIRR implements Excel's XIRR function
func (s *SemanticCalculationService) evaluateXIRR(arguments map[string]interface{}) (float64, error) {
	cashFlows, ok := arguments["cash_flows"].([]interface{})
	if !ok {
		return 0, fmt.Errorf("cash_flows argument required for XIRR")
	}

	dates, ok := arguments["dates"].([]interface{})
	if !ok {
		return 0, fmt.Errorf("dates argument required for XIRR")
	}

	if len(cashFlows) != len(dates) {
		return 0, fmt.Errorf("cash flows and dates must have the same length")
	}

	// Convert to float64 arrays
	cf := make([]float64, len(cashFlows))
	for i, v := range cashFlows {
		if f, ok := v.(float64); ok {
			cf[i] = f
		} else {
			return 0, fmt.Errorf("invalid cash flow value")
		}
	}

	// For simplicity, use the existing IRR calculation
	// In production, you'd use proper XIRR with date weighting
	return s.calculateIRR(cf, 0.1), nil
}

// evaluateNPV implements Excel's NPV function
func (s *SemanticCalculationService) evaluateNPV(arguments map[string]interface{}) (float64, error) {
	rate, ok := arguments["rate"].(float64)
	if !ok {
		return 0, fmt.Errorf("rate argument required for NPV")
	}

	cashFlows, ok := arguments["cash_flows"].([]interface{})
	if !ok {
		return 0, fmt.Errorf("cash_flows argument required for NPV")
	}

	npv := 0.0
	for i, v := range cashFlows {
		if f, ok := v.(float64); ok {
			if i == 0 {
				npv += f
			} else {
				npv += f / math.Pow(1+rate, float64(i))
			}
		} else {
			return 0, fmt.Errorf("invalid cash flow value")
		}
	}

	return npv, nil
}

// evaluateIRR implements Excel's IRR function
func (s *SemanticCalculationService) evaluateIRR(arguments map[string]interface{}) (float64, error) {
	cashFlows, ok := arguments["cash_flows"].([]interface{})
	if !ok {
		return 0, fmt.Errorf("cash_flows argument required for IRR")
	}

	cf := make([]float64, len(cashFlows))
	for i, v := range cashFlows {
		if f, ok := v.(float64); ok {
			cf[i] = f
		} else {
			return 0, fmt.Errorf("invalid cash flow value")
		}
	}

	return s.calculateIRR(cf, 0.1), nil
}

// evaluatePV implements Excel's PV function
func (s *SemanticCalculationService) evaluatePV(arguments map[string]interface{}) (float64, error) {
	rate, ok := arguments["rate"].(float64)
	if !ok {
		return 0, fmt.Errorf("rate argument required for PV")
	}

	nper, ok := arguments["nper"].(float64)
	if !ok {
		return 0, fmt.Errorf("nper argument required for PV")
	}

	pmt, ok := arguments["pmt"].(float64)
	if !ok {
		return 0, fmt.Errorf("pmt argument required for PV")
	}

	fv, ok := arguments["fv"].(float64)
	if !ok {
		fv = 0
	}

	return -s.calculatePV(rate, int(nper), pmt, fv), nil
}

// evaluateFV implements Excel's FV function
func (s *SemanticCalculationService) evaluateFV(arguments map[string]interface{}) (float64, error) {
	rate, ok := arguments["rate"].(float64)
	if !ok {
		return 0, fmt.Errorf("rate argument required for FV")
	}

	nper, ok := arguments["nper"].(float64)
	if !ok {
		return 0, fmt.Errorf("nper argument required for FV")
	}

	pmt, ok := arguments["pmt"].(float64)
	if !ok {
		return 0, fmt.Errorf("pmt argument required for FV")
	}

	pv, ok := arguments["pv"].(float64)
	if !ok {
		pv = 0
	}

	return s.calculateFV(rate, int(nper), pmt, pv), nil
}

// evaluatePMT implements Excel's PMT function
func (s *SemanticCalculationService) evaluatePMT(arguments map[string]interface{}) (float64, error) {
	rate, ok := arguments["rate"].(float64)
	if !ok {
		return 0, fmt.Errorf("rate argument required for PMT")
	}

	nper, ok := arguments["nper"].(float64)
	if !ok {
		return 0, fmt.Errorf("nper argument required for PMT")
	}

	pv, ok := arguments["pv"].(float64)
	if !ok {
		return 0, fmt.Errorf("pv argument required for PMT")
	}

	fv, ok := arguments["fv"].(float64)
	if !ok {
		fv = 0
	}

	return s.calculatePMT(rate, int(nper), pv, fv), nil
}

// evaluateMIRR implements Excel's MIRR function
func (s *SemanticCalculationService) evaluateMIRR(arguments map[string]interface{}) (float64, error) {
	cashFlows, ok := arguments["cash_flows"].([]interface{})
	if !ok {
		return 0, fmt.Errorf("cash_flows argument required for MIRR")
	}

	financeRate, ok := arguments["finance_rate"].(float64)
	if !ok {
		return 0, fmt.Errorf("finance_rate argument required for MIRR")
	}

	reinvestRate, ok := arguments["reinvest_rate"].(float64)
	if !ok {
		return 0, fmt.Errorf("reinvest_rate argument required for MIRR")
	}

	cf := make([]float64, len(cashFlows))
	for i, v := range cashFlows {
		if f, ok := v.(float64); ok {
			cf[i] = f
		} else {
			return 0, fmt.Errorf("invalid cash flow value")
		}
	}

	return s.calculateMIRR(cf, financeRate, reinvestRate), nil
}

// Helper methods for financial calculations
func (s *SemanticCalculationService) calculatePV(rate float64, nper int, pmt, fv float64) float64 {
	if rate == 0 {
		return -fv - pmt*float64(nper)
	}

	return (fv + pmt*(1-math.Pow(1+rate, float64(-nper)))/rate) / math.Pow(1+rate, float64(nper))
}

func (s *SemanticCalculationService) calculateFV(rate float64, nper int, pmt, pv float64) float64 {
	if rate == 0 {
		return -pv - pmt*float64(nper)
	}

	return pv*math.Pow(1+rate, float64(nper)) + pmt*(math.Pow(1+rate, float64(nper))-1)/rate
}

func (s *SemanticCalculationService) calculatePMT(rate float64, nper int, pv, fv float64) float64 {
	if rate == 0 {
		return (-pv - fv) / float64(nper)
	}

	return (pv + fv*math.Pow(1+rate, float64(-nper))) * rate / (1 - math.Pow(1+rate, float64(-nper)))
}

func (s *SemanticCalculationService) calculateMIRR(cashFlows []float64, financeRate, reinvestRate float64) float64 {
	if len(cashFlows) == 0 {
		return 0
	}

	positiveFlows := 0.0
	negativeFlows := 0.0

	for i, cf := range cashFlows {
		if cf > 0 {
			positiveFlows += cf / math.Pow(1+reinvestRate, float64(i))
		} else {
			negativeFlows += cf / math.Pow(1+financeRate, float64(i))
		}
	}

	if negativeFlows == 0 {
		return 0
	}

	return math.Pow(positiveFlows/math.Abs(negativeFlows), 1.0/float64(len(cashFlows)-1)) - 1
}

// detectVectorizedArguments checks if arguments contain arrays indicating vectorized execution
func (s *SemanticCalculationService) detectVectorizedArguments(arguments map[string]interface{}) ([]map[string]interface{}, bool) {
	// Look for arguments that are arrays of values or arrays of objects
	var batchSize int = -1

	for _, value := range arguments {
		switch v := value.(type) {
		case []interface{}:
			// Check if this is an array of primitive values (single batch dimension)
			if len(v) > 0 {
				if _, ok := v[0].(map[string]interface{}); !ok {
					// This is an array of primitives - indicates vectorized execution
					if batchSize == -1 {
						batchSize = len(v)
					} else if batchSize != len(v) {
						// Inconsistent batch sizes
						return nil, false
					}
				}
			}
		case []map[string]interface{}:
			// This is explicitly an array of argument sets
			if batchSize == -1 {
				batchSize = len(v)
			} else if batchSize != len(v) {
				return nil, false
			}
		}
	}

	if batchSize <= 1 {
		return nil, false
	}

	// Build vectorized argument sets
	vectorizedArgs := make([]map[string]interface{}, batchSize)
	for i := 0; i < batchSize; i++ {
		argSet := make(map[string]interface{})
		for key, value := range arguments {
			switch v := value.(type) { //nolint:gocritic
			case []any:
				if len(v) > i {
					argSet[key] = v[i]
				}
			case []map[string]any:
				if len(v) > i {
					// Merge the argument set
					for k, val := range v[i] {
						argSet[k] = val
					}
				}
			default:
				// Scalar values are replicated across all batch items
				argSet[key] = value
			}
		}
		vectorizedArgs[i] = argSet
	}

	return vectorizedArgs, true
}

// executeVectorizedExcelFormula executes the same Excel formula across multiple argument sets
func (s *SemanticCalculationService) executeVectorizedExcelFormula(formula string, vectorizedArgs []map[string]interface{}) ([]interface{}, error) {
	results := make([]interface{}, len(vectorizedArgs))

	// Execute formula for each argument set
	for i, args := range vectorizedArgs {
		result, err := s.evaluateSimpleExcelFormula(formula, args)
		if err != nil {
			// For vectorized execution, we can choose to continue with partial results
			// or fail fast. Here we choose to continue but mark failed calculations
			results[i] = map[string]interface{}{
				"error": err.Error(),
				"index": i,
			}
			continue
		}
		results[i] = result
	}

	return results, nil
}

// getMetricDefinition retrieves metric definition from registry (placeholder implementation)
func (s *SemanticCalculationService) getMetricDefinition(metricID string) (*MetricDefinition, error) {
	// This would integrate with your metric registry
	// For now, return a placeholder
	return &MetricDefinition{
		ID: metricID,
		FinancialCalc: &FinancialCalc{
			Type:    "excel_formula",
			Formula: "=XIRR({cash_flows}, {dates})",
			Arguments: map[string]interface{}{
				"cash_flows": "ARRAY_AGG(net_cash_flow)",
				"dates":      "ARRAY_AGG(transaction_date)",
			},
		},
	}, nil
}

// getEntityData retrieves entity data for calculation (placeholder implementation)
func (s *SemanticCalculationService) getEntityData(entityID string) (map[string]interface{}, error) {
	// This would integrate with your data layer
	// For now, return sample data
	return map[string]interface{}{
		"entity_id":  entityID,
		"cash_flows": []interface{}{-1000.0, 200.0, 300.0, 400.0, 500.0},
		"dates":      []interface{}{1.0, 2.0, 3.0, 4.0, 5.0},
	}, nil
}

// resolveArgumentsForEntity resolves formula arguments for a specific entity
func (s *SemanticCalculationService) resolveArgumentsForEntity(calc *FinancialCalc, entityData map[string]interface{}) (map[string]interface{}, error) {
	resolved := make(map[string]interface{})

	for key, value := range calc.Arguments {
		switch v := value.(type) {
		case string:
			// Handle SQL-like expressions (simplified)
			if v == "ARRAY_AGG(net_cash_flow)" {
				if cashFlows, ok := entityData["cash_flows"].([]interface{}); ok {
					resolved[key] = cashFlows
				} else {
					resolved[key] = []interface{}{}
				}
			} else if v == "ARRAY_AGG(transaction_date)" {
				if dates, ok := entityData["dates"].([]interface{}); ok {
					resolved[key] = dates
				} else {
					resolved[key] = []interface{}{}
				}
			} else {
				resolved[key] = v
			}
		default:
			resolved[key] = value
		}
	}

	return resolved, nil
}

// MetricDefinition represents a metric from the registry
type MetricDefinition struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	FinancialCalc *FinancialCalc `json:"financial_calc,omitempty"`
}
