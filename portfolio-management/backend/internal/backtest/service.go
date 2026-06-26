package backtest

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ============================================================================
// Backtest Service - Core Business Logic
// ============================================================================

// HasuraClient interface for GraphQL operations
type HasuraClient interface {
	Query(query string, variables map[string]interface{}) (map[string]interface{}, error)
	Mutate(mutation string, variables map[string]interface{}) (map[string]interface{}, error)
}

type Service struct {
	db     *sqlx.DB
	hasura HasuraClient
}

func NewService(db *sqlx.DB) *Service {
	return &Service{db: db}
}

func NewServiceWithHasura(db *sqlx.DB, hasura HasuraClient) *Service {
	return &Service{db: db, hasura: hasura}
}

// ============================================================================
// Portfolio Operations
// ============================================================================

// GetPortfolio retrieves a portfolio with all holdings
func (s *Service) GetPortfolio(ctx context.Context, portfolioID string) (*Portfolio, error) {
	// Use Hasura if available, otherwise fallback to direct DB
	if s.hasura != nil {
		return s.getPortfolioWithHasura(ctx, portfolioID)
	}

	// TODO: Hasura-first pattern already implemented via getPortfolioWithHasura()
	// Primary implementation uses Hasura GraphQL, this SQL is fallback only
	// See getPortfolioWithHasura() for the Hasura query: query GetPortfolio($id: uuid!)
	portfolio := &Portfolio{}
	query := `
		SELECT id, tenant_id, client_id, type, benchmark, asset_allocation_targets, 
		       performance_metrics, advisor_discretion, client_approval_required, 
		       template_id, custom_fields, created_at, updated_at
		FROM portfolios
		WHERE id = $1
	`
	err := s.db.GetContext(ctx, portfolio, query, portfolioID)
	if err != nil {
		return nil, err
	}

	// Fetch holdings
	holdings, err := s.GetHoldings(ctx, portfolioID)
	if err != nil {
		return nil, err
	}
	portfolio.Holdings = holdings

	return portfolio, nil
}

func (s *Service) getPortfolioWithHasura(ctx context.Context, portfolioID string) (*Portfolio, error) {
	query := `
		query GetPortfolio($id: uuid!) {
			portfolios_by_pk(id: $id) {
				id
				tenant_id
				client_id
				type
				benchmark
				asset_allocation_targets
				performance_metrics
				advisor_discretion
				client_approval_required
				template_id
				custom_fields
				created_at
				updated_at
			}
		}
	`

	result, err := s.hasura.Query(query, map[string]interface{}{"id": portfolioID})
	if err != nil {
		return nil, err
	}

	portfolioData, ok := result["portfolios_by_pk"].(map[string]interface{})
	if !ok || portfolioData == nil {
		return nil, fmt.Errorf("portfolio not found")
	}

	portfolio := &Portfolio{}
	if id, ok := portfolioData["id"].(string); ok {
		portfolio.ID = uuid.MustParse(id)
	}
	if tenantID, ok := portfolioData["tenant_id"].(string); ok {
		portfolio.TenantID = uuid.MustParse(tenantID)
	}
	if clientID, ok := portfolioData["client_id"].(string); ok {
		portfolio.ClientID = uuid.MustParse(clientID)
	}
	if typ, ok := portfolioData["type"].(string); ok {
		portfolio.Type = typ
	}
	if benchmark, ok := portfolioData["benchmark"].(string); ok {
		portfolio.Benchmark = benchmark
	}
	if targets, ok := portfolioData["asset_allocation_targets"].(string); ok {
		portfolio.AssetAllocationTargets = json.RawMessage(targets)
	}
	if metrics, ok := portfolioData["performance_metrics"].(string); ok {
		portfolio.PerformanceMetrics = json.RawMessage(metrics)
	}
	if discretion, ok := portfolioData["advisor_discretion"].(bool); ok {
		portfolio.AdvisorDiscretion = discretion
	}
	if approval, ok := portfolioData["client_approval_required"].(bool); ok {
		portfolio.ClientApprovalRequired = approval
	}
	if templateID, ok := portfolioData["template_id"].(string); ok && templateID != "" {
		tid := uuid.MustParse(templateID)
		portfolio.TemplateID = &tid
	}
	if customFields, ok := portfolioData["custom_fields"].(string); ok {
		portfolio.CustomFields = json.RawMessage(customFields)
	}

	return portfolio, nil
} // GetHoldings retrieves all holdings for a portfolio
func (s *Service) GetHoldings(ctx context.Context, portfolioID string) ([]Holding, error) {
	// TODO: Refactor to Hasura GraphQL
	// query { holdings(where: {portfolio_id: {_eq: $portfolio_id}}, order_by: {current_value: desc}) {
	//   id portfolio_id symbol name asset_class quantity average_cost current_price current_value sector geography
	// }}
	var holdings []Holding
	query := `
		SELECT id, portfolio_id, symbol, name, asset_class, quantity, average_cost, 
		       current_price, current_value, sector, geography, created_at, updated_at
		FROM holdings
		WHERE portfolio_id = $1
		ORDER BY current_value DESC
	`
	err := s.db.SelectContext(ctx, &holdings, query, portfolioID)
	return holdings, err
}

// CreatePortfolio creates a new portfolio
func (s *Service) CreatePortfolio(ctx context.Context, req CreatePortfolioRequest, tenantID, clientID string) (*Portfolio, error) {
	// Use Hasura if available, otherwise fallback to direct DB
	if s.hasura != nil {
		return s.createPortfolioWithHasura(ctx, req, tenantID, clientID)
	}

	// TODO: Hasura-first pattern already implemented via createPortfolioWithHasura()
	// Primary implementation uses Hasura GraphQL, this SQL is fallback only
	// See createPortfolioWithHasura() for the Hasura mutation: mutation InsertPortfolio($object: portfolios_insert_input!)
	portfolioID := uuid.New()

	query := `
		INSERT INTO portfolios (id, tenant_id, client_id, type, benchmark, asset_allocation_targets, 
		                        performance_metrics, advisor_discretion, client_approval_required, 
		                        custom_fields, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
		RETURNING id, tenant_id, client_id, type, benchmark, asset_allocation_targets, 
		          performance_metrics, advisor_discretion, client_approval_required, 
		          template_id, custom_fields, created_at, updated_at
	`

	portfolio := &Portfolio{}
	err := s.db.GetContext(ctx, portfolio, query, portfolioID, tenantID, clientID,
		req.Type, req.Benchmark, req.AssetAllocationTargets, req.PerformanceMetrics,
		req.AdvisorDiscretion, req.ClientApprovalRequired, req.CustomFields)
	if err != nil {
		return nil, err
	}

	// Create holdings if provided
	for _, h := range req.Holdings {
		holding := Holding{
			ID:           uuid.New(),
			PortfolioID:  portfolioID,
			Symbol:       h.Symbol,
			Name:         h.Name,
			AssetClass:   h.AssetClass,
			Quantity:     h.Quantity,
			AverageCost:  h.AverageCost,
			CurrentPrice: h.AverageCost, // Start with average cost
			CurrentValue: h.Quantity * h.AverageCost,
			Sector:       h.Sector,
			Geography:    h.Geography,
		}

		// TODO: Refactor to Hasura GraphQL bulk mutation
		// mutation { insert_holdings(objects: [{...}]) { returning { id } affected_rows }}
		holdingQuery := `
			INSERT INTO holdings (id, portfolio_id, symbol, name, asset_class, quantity, average_cost, 
					current_price, current_value, sector, geography, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
		`
		_, err := s.db.ExecContext(ctx, holdingQuery,
			holding.ID, holding.PortfolioID, holding.Symbol, holding.Name, holding.AssetClass,
			holding.Quantity, holding.AverageCost, holding.CurrentPrice, holding.CurrentValue,
			holding.Sector, holding.Geography)
		if err != nil {
			return nil, err
		}

		portfolio.Holdings = append(portfolio.Holdings, holding)
	}

	return portfolio, nil
}

func (s *Service) createPortfolioWithHasura(ctx context.Context, req CreatePortfolioRequest, tenantID, clientID string) (*Portfolio, error) {
	mutation := `
		mutation InsertPortfolio($object: portfolios_insert_input!) {
			insert_portfolios_one(object: $object) {
				id
				tenant_id
				client_id
				type
				benchmark
				asset_allocation_targets
				performance_metrics
				advisor_discretion
				client_approval_required
				template_id
				custom_fields
				created_at
				updated_at
			}
		}
	`

	portfolioID := uuid.New()
	variables := map[string]interface{}{
		"object": map[string]interface{}{
			"id":                       portfolioID.String(),
			"tenant_id":                tenantID,
			"client_id":                clientID,
			"type":                     req.Type,
			"benchmark":                req.Benchmark,
			"asset_allocation_targets": req.AssetAllocationTargets,
			"performance_metrics":      req.PerformanceMetrics,
			"advisor_discretion":       req.AdvisorDiscretion,
			"client_approval_required": req.ClientApprovalRequired,
			"custom_fields":            req.CustomFields,
		},
	}

	result, err := s.hasura.Mutate(mutation, variables)
	if err != nil {
		return nil, err
	}

	portfolioData, ok := result["insert_portfolios_one"].(map[string]interface{})
	if !ok || portfolioData == nil {
		return nil, fmt.Errorf("failed to create portfolio")
	}

	portfolio := &Portfolio{
		ID:                     portfolioID,
		AdvisorDiscretion:      req.AdvisorDiscretion,
		ClientApprovalRequired: req.ClientApprovalRequired,
	}

	if tid, ok := portfolioData["tenant_id"].(string); ok {
		portfolio.TenantID = uuid.MustParse(tid)
	}
	if cid, ok := portfolioData["client_id"].(string); ok {
		portfolio.ClientID = uuid.MustParse(cid)
	}
	if typ, ok := portfolioData["type"].(string); ok {
		portfolio.Type = typ
	}
	if benchmark, ok := portfolioData["benchmark"].(string); ok {
		portfolio.Benchmark = benchmark
	}
	if targets, ok := portfolioData["asset_allocation_targets"].(string); ok {
		portfolio.AssetAllocationTargets = json.RawMessage(targets)
	}
	if metrics, ok := portfolioData["performance_metrics"].(string); ok {
		portfolio.PerformanceMetrics = json.RawMessage(metrics)
	}
	if customFields, ok := portfolioData["custom_fields"].(string); ok {
		portfolio.CustomFields = json.RawMessage(customFields)
	}

	return portfolio, nil
}

// ============================================================================
// Recommendation Operations
// ============================================================================

// CreateRecommendation creates a new recommendation
func (s *Service) CreateRecommendation(ctx context.Context, portfolioID, userID string, req CreateRecommendationRequest) (*Recommendation, error) {
	// TODO: Refactor to Hasura GraphQL
	// mutation { insert_recommendations_one(object: {id, portfolio_id, created_by, title, ...}) {
	//   id portfolio_id created_by title description type status priority
	// }}
	recID := uuid.New()

	allocJSON, _ := json.Marshal(req.TargetAllocations)
	actionsJSON, _ := json.Marshal(req.Actions)

	query := `
		INSERT INTO recommendations (id, portfolio_id, created_by, title, description, type, status, priority, 
			target_allocations, recommended_actions, rationale, risk_score, expected_return, time_horizon, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, 'draft', $7, $8, $9, $10, 0, 0, $11, NOW(), NOW())
		RETURNING id, portfolio_id, created_by, title, description, type, status, priority, created_at, updated_at
	`

	rec := &Recommendation{
		ID:                recID,
		PortfolioID:       uuid.MustParse(portfolioID),
		CreatedBy:         uuid.MustParse(userID),
		Title:             req.Title,
		Description:       req.Description,
		Type:              req.Type,
		Status:            "draft",
		Priority:          req.Priority,
		TargetAllocations: req.TargetAllocations,
		Actions:           req.Actions,
		Rationale:         req.Rationale,
		TimeHorizon:       req.TimeHorizon,
	}

	err := s.db.GetContext(ctx, rec, query,
		recID, portfolioID, userID, req.Title, req.Description, req.Type, req.Priority,
		string(allocJSON), string(actionsJSON), req.Rationale, req.TimeHorizon)

	if err != nil {
		return nil, err
	}

	return rec, nil
}

// GetRecommendation retrieves a specific recommendation
func (s *Service) GetRecommendation(ctx context.Context, recID string) (*Recommendation, error) {
	// TODO: Refactor to Hasura GraphQL
	// query { recommendations_by_pk(id: $id) {
	//   id portfolio_id created_by title description type status priority
	//   target_allocations recommended_actions rationale risk_score expected_return time_horizon backtest_id
	// }}
	rec := &Recommendation{}
	query := `
		SELECT id, portfolio_id, created_by, title, description, type, status, priority,
		       target_allocations, recommended_actions, rationale, risk_score, expected_return, 
		       time_horizon, backtest_id, created_at, updated_at
		FROM recommendations
		WHERE id = $1
	`
	err := s.db.GetContext(ctx, rec, query, recID)
	if err != nil {
		return nil, err
	}

	// Parse JSON fields
	if len(rec.Metadata) > 0 {
		var allocs []TargetAllocation
		json.Unmarshal(rec.Metadata, &allocs)
		rec.TargetAllocations = allocs
	}

	return rec, nil
}

// UpdateRecommendationStatus updates recommendation status
func (s *Service) UpdateRecommendationStatus(ctx context.Context, recID, status, notes string) error {
	// TODO: Refactor to Hasura GraphQL
	// mutation { update_recommendations_by_pk(
	//   pk_columns: {id: $id}
	//   _set: {status: $status, metadata: $metadata, updated_at: "now()"}
	// ) { id status }}
	query := `
		UPDATE recommendations 
		SET status = $1, metadata = jsonb_set(COALESCE(metadata, '{}'::jsonb), '{notes}', $2::jsonb), updated_at = NOW()
		WHERE id = $3
	`
	notesJSON, _ := json.Marshal(notes)
	_, err := s.db.ExecContext(ctx, query, status, string(notesJSON), recID)
	return err
}

// ============================================================================
// Backtest Operations
// ============================================================================

// RunBacktest executes a complete backtest simulation
func (s *Service) RunBacktest(ctx context.Context, req BacktestRequest) (*BacktestResult, error) {
	// Parse IDs
	recID, err := uuid.Parse(req.RecommendationID)
	if err != nil {
		return nil, fmt.Errorf("invalid recommendation ID: %w", err)
	}

	portID, err := uuid.Parse(req.PortfolioID)
	if err != nil {
		return nil, fmt.Errorf("invalid portfolio ID: %w", err)
	}

	// Fetch portfolio and recommendation
	portfolio, err := s.GetPortfolio(ctx, req.PortfolioID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch portfolio: %w", err)
	}

	recommendation, err := s.GetRecommendation(ctx, req.RecommendationID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recommendation: %w", err)
	}

	holdings, err := s.GetHoldings(ctx, req.PortfolioID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch holdings: %w", err)
	}

	// Fetch historical prices
	historicalPrices, err := s.fetchHistoricalPrices(ctx, holdings, req.StartDate, req.EndDate)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch historical prices: %w", err)
	}

	// Run historical simulation
	dailySimulations, baselineReturn, recommendedReturn, maxDrawdownBaseline, maxDrawdownRecommended :=
		s.runHistoricalSimulation(holdings, recommendation, historicalPrices, req.StartDate, req.EndDate)

	// Calculate metrics
	alpha := recommendedReturn - baselineReturn
	sharpeBaseline := s.calculateSharpeRatio(dailySimulations, true)
	sharpeRecommended := s.calculateSharpeRatio(dailySimulations, false)

	// Estimate tax and transaction costs
	taxSavings := s.estimateTaxSavings(portfolio, recommendation)
	transactionCosts := s.estimateTransactionCosts(portfolio, recommendation)

	// Calculate confidence score (0-1)
	confidence := s.calculateConfidenceScore(float64(len(dailySimulations)))

	netBenefit := alpha + taxSavings - transactionCosts

	// Create result
	simulationDataJSON, _ := json.Marshal(dailySimulations)

	result := &BacktestResult{
		ID:                     uuid.New(),
		RecommendationID:       recID,
		PortfolioID:            portID,
		SimulationType:         "historical",
		StartDate:              req.StartDate,
		EndDate:                req.EndDate,
		BaselineReturn:         baselineReturn,
		RecommendationReturn:   recommendedReturn,
		AlphaGenerated:         alpha,
		BetaAdjustedReturn:     recommendedReturn * 0.95, // Placeholder
		SharpeRatioBaseline:    sharpeBaseline,
		SharpeRatioRecommended: sharpeRecommended,
		MaxDrawdownBaseline:    maxDrawdownBaseline,
		MaxDrawdownRecommended: maxDrawdownRecommended,
		TaxSavingsAccumulated:  taxSavings,
		TransactionCosts:       transactionCosts,
		NetBenefit:             netBenefit,
		Confidence:             confidence,
		SimulationData:         simulationDataJSON,
		CreatedAt:              time.Now(),
	}

	// Save to database
	err = s.saveBacktestResult(ctx, result)
	if err != nil {
		return nil, fmt.Errorf("failed to save backtest: %w", err)
	}

	return result, nil
}

// ============================================================================
// Simulation Calculations
// ============================================================================

func (s *Service) runHistoricalSimulation(holdings []Holding, recommendation *Recommendation,
	historicalPrices map[string][]float64, startDate, endDate time.Time) (
	[]DailySimulation, float64, float64, float64, float64) {

	var dailySimulations []DailySimulation
	var baselineValues []float64
	var recommendedValues []float64

	// Calculate initial portfolio value
	initialValue := 0.0
	for _, h := range holdings {
		initialValue += h.CurrentValue
	}

	baselineValues = append(baselineValues, initialValue)
	recommendedValues = append(recommendedValues, initialValue)

	// Simulate daily changes
	daysToSimulate := endDate.Sub(startDate).Hours() / 24
	for day := 1; day <= int(daysToSimulate); day++ {
		currentDate := startDate.AddDate(0, 0, day)

		baselineValue := initialValue
		recommendedValue := initialValue

		// Apply price changes for each holding
		for _, h := range holdings {
			if prices, ok := historicalPrices[h.Symbol]; ok && len(prices) > day {
				priceChange := (prices[day] / prices[0]) - 1.0
				baselineValue += h.CurrentValue * priceChange
			}
		}

		// Apply recommendation changes
		for _, targetAlloc := range recommendation.TargetAllocations {
			if prices, ok := historicalPrices[targetAlloc.Symbol]; ok && len(prices) > day {
				priceChange := (prices[day] / prices[0]) - 1.0
				recommendedValue += initialValue * (targetAlloc.TargetAllocation / 100.0) * priceChange
			}
		}

		baselineValues = append(baselineValues, baselineValue)
		recommendedValues = append(recommendedValues, recommendedValue)

		sim := DailySimulation{
			Date:                 currentDate,
			BaselineValue:        baselineValue,
			RecommendationValue:  recommendedValue,
			BaselineReturn:       (baselineValue - initialValue) / initialValue,
			RecommendationReturn: (recommendedValue - initialValue) / initialValue,
			AlphaAccumulated:     (recommendedValue - baselineValue) / initialValue,
		}

		dailySimulations = append(dailySimulations, sim)
	}

	// Calculate aggregate metrics
	baselineReturn := (baselineValues[len(baselineValues)-1] - initialValue) / initialValue
	recommendedReturn := (recommendedValues[len(recommendedValues)-1] - initialValue) / initialValue
	maxDrawdownBaseline := s.calculateMaxDrawdown(baselineValues)
	maxDrawdownRecommended := s.calculateMaxDrawdown(recommendedValues)

	return dailySimulations, baselineReturn, recommendedReturn, maxDrawdownBaseline, maxDrawdownRecommended
}

func (s *Service) calculateSharpeRatio(simulations []DailySimulation, isBaseline bool) float64 {
	if len(simulations) == 0 {
		return 0
	}

	var returns []float64
	for _, sim := range simulations {
		var ret float64
		if isBaseline {
			ret = sim.BaselineReturn
		} else {
			ret = sim.RecommendationReturn
		}
		returns = append(returns, ret)
	}

	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))

	variance := 0.0
	for _, r := range returns {
		variance += (r - mean) * (r - mean)
	}
	variance /= float64(len(returns))

	stdDev := math.Sqrt(variance)
	if stdDev == 0 {
		return 0
	}

	// Annual Sharpe Ratio (assuming 252 trading days)
	return (mean * 252) / (stdDev * math.Sqrt(252))
}

func (s *Service) calculateMaxDrawdown(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	maxDrawdown := 0.0
	peak := values[0]

	for _, value := range values {
		if value > peak {
			peak = value
		}
		drawdown := (peak - value) / peak
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	return maxDrawdown
}

func (s *Service) estimateTaxSavings(portfolio *Portfolio, recommendation *Recommendation) float64 {
	// Placeholder: 0.5% tax optimization benefit
	totalValue := 0.0
	for _, h := range portfolio.Holdings {
		totalValue += h.CurrentValue
	}
	return totalValue * 0.005
}

func (s *Service) estimateTransactionCosts(portfolio *Portfolio, recommendation *Recommendation) float64 {
	// Placeholder: 0.1% of portfolio value for execution
	totalValue := 0.0
	for _, h := range portfolio.Holdings {
		totalValue += h.CurrentValue
	}
	return totalValue * 0.001
}

func (s *Service) calculateConfidenceScore(dataPoints float64) float64 {
	// Confidence increases with more data points, capped at 1.0
	confidence := math.Min(dataPoints/252.0, 1.0) // 252 trading days = 1 year
	return confidence
}

// ============================================================================
// Price Data Handling
// ============================================================================

func (s *Service) fetchHistoricalPrices(ctx context.Context, holdings []Holding, startDate, endDate time.Time) (map[string][]float64, error) {
	prices := make(map[string][]float64)

	// TODO: Refactor to Hasura GraphQL
	// query { historical_prices(
	//   where: {ticker: {_in: $tickers}, date: {_gte: $start_date, _lte: $end_date}}
	//   order_by: {date: asc}
	// ) { ticker close_price date }}
	for _, h := range holdings {
		query := `
			SELECT close_price FROM historical_prices
			WHERE ticker = $1 AND date BETWEEN $2 AND $3
			ORDER BY date ASC
		`

		var priceList []float64
		err := s.db.SelectContext(ctx, &priceList, query, h.Symbol, startDate, endDate)
		if err != nil {
			// Use current price as fallback
			for i := 0; i < int(endDate.Sub(startDate).Hours()/24); i++ {
				priceList = append(priceList, h.CurrentPrice)
			}
		}

		prices[h.Symbol] = priceList
	}

	return prices, nil
}

// ============================================================================
// Database Operations
// ============================================================================

func (s *Service) saveBacktestResult(ctx context.Context, result *BacktestResult) error {
	// TODO: Refactor to Hasura GraphQL
	// mutation { insert_backtest_results_one(object: {
	//   id, recommendation_id, portfolio_id, simulation_type, start_date, end_date,
	//   baseline_return, recommendation_return, alpha_generated, ..., simulation_data
	// }) { id }}
	query := `
		INSERT INTO backtest_results 
		(id, recommendation_id, portfolio_id, simulation_type, start_date, end_date,
		 baseline_return, recommendation_return, alpha_generated, beta_adjusted_return,
		 sharpe_ratio_baseline, sharpe_ratio_recommended, max_drawdown_baseline, max_drawdown_recommended,
		 tax_savings_accumulated, transaction_costs, net_benefit, confidence, simulation_data, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
	`

	_, err := s.db.ExecContext(ctx, query,
		result.ID, result.RecommendationID, result.PortfolioID, result.SimulationType,
		result.StartDate, result.EndDate, result.BaselineReturn, result.RecommendationReturn,
		result.AlphaGenerated, result.BetaAdjustedReturn, result.SharpeRatioBaseline, result.SharpeRatioRecommended,
		result.MaxDrawdownBaseline, result.MaxDrawdownRecommended, result.TaxSavingsAccumulated,
		result.TransactionCosts, result.NetBenefit, result.Confidence, result.SimulationData, result.CreatedAt)

	return err
}

// GetBacktestResults retrieves historical backtest results
func (s *Service) GetBacktestResults(ctx context.Context, portfolioID string, limit int) ([]BacktestResult, error) {
	// TODO: Refactor to Hasura GraphQL
	// query { backtest_results(
	//   where: {portfolio_id: {_eq: $portfolio_id}}
	//   order_by: {created_at: desc}
	//   limit: $limit
	// ) { id recommendation_id portfolio_id ... }}
	var results []BacktestResult
	query := `
		SELECT id, recommendation_id, portfolio_id, simulation_type, start_date, end_date,
		       baseline_return, recommendation_return, alpha_generated, beta_adjusted_return,
		       sharpe_ratio_baseline, sharpe_ratio_recommended, max_drawdown_baseline, max_drawdown_recommended,
		       tax_savings_accumulated, transaction_costs, net_benefit, confidence, simulation_data, created_at
		FROM backtest_results
		WHERE portfolio_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	err := s.db.SelectContext(ctx, &results, query, portfolioID, limit)
	return results, err
}

// GetBacktestByID retrieves a specific backtest result
func (s *Service) GetBacktestByID(ctx context.Context, backtestID string) (*BacktestResult, error) {
	// TODO: Refactor to Hasura GraphQL
	// query { backtest_results_by_pk(id: $id) {
	//   id recommendation_id portfolio_id simulation_type start_date end_date
	//   baseline_return recommendation_return alpha_generated ... simulation_data
	// }}
	result := &BacktestResult{}
	query := `
		SELECT id, recommendation_id, portfolio_id, simulation_type, start_date, end_date,
		       baseline_return, recommendation_return, alpha_generated, beta_adjusted_return,
		       sharpe_ratio_baseline, sharpe_ratio_recommended, max_drawdown_baseline, max_drawdown_recommended,
		       tax_savings_accumulated, transaction_costs, net_benefit, confidence, simulation_data, created_at
		FROM backtest_results
		WHERE id = $1
	`
	err := s.db.GetContext(ctx, result, query, backtestID)
	return result, err
}

// CompareBacktests compares two recommendations
func (s *Service) CompareBacktests(ctx context.Context, req ComparisonRequest) (*ComparisonResult, error) {
	// Get latest backtest for each recommendation
	// TODO: Refactor to Hasura GraphQL
	// query { backtest_results(
	//   where: {recommendation_id: {_eq: $rec_id}, portfolio_id: {_eq: $portfolio_id}}
	//   order_by: {created_at: desc}
	//   limit: 1
	// ) { ... all fields }}
	var backtest1, backtest2 *BacktestResult

	query := `
		SELECT id, recommendation_id, portfolio_id, simulation_type, start_date, end_date,
		       baseline_return, recommendation_return, alpha_generated, beta_adjusted_return,
		       sharpe_ratio_baseline, sharpe_ratio_recommended, max_drawdown_baseline, max_drawdown_recommended,
		       tax_savings_accumulated, transaction_costs, net_benefit, confidence, simulation_data, created_at
		FROM backtest_results
		WHERE recommendation_id = $1 AND portfolio_id = $2
		ORDER BY created_at DESC
		LIMIT 1
	`

	err := s.db.GetContext(ctx, backtest1, query, req.RecommendationID1, req.PortfolioID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch backtest 1: %w", err)
	}

	err = s.db.GetContext(ctx, backtest2, query, req.RecommendationID2, req.PortfolioID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch backtest 2: %w", err)
	}

	// Determine winner
	winner := "tie"
	if backtest1.NetBenefit > backtest2.NetBenefit {
		winner = "rec1"
	} else if backtest2.NetBenefit > backtest1.NetBenefit {
		winner = "rec2"
	}

	// Build comparison result
	comparison := &ComparisonResult{
		ID:                uuid.New(),
		PortfolioID:       uuid.MustParse(req.PortfolioID),
		RecommendationID1: uuid.MustParse(req.RecommendationID1),
		RecommendationID2: uuid.MustParse(req.RecommendationID2),
		Winner:            winner,
		PerformanceDiff:   backtest1.RecommendationReturn - backtest2.RecommendationReturn,
		RiskDiff:          backtest1.MaxDrawdownRecommended - backtest2.MaxDrawdownRecommended,
		SharpeRatioDiff:   backtest1.SharpeRatioRecommended - backtest2.SharpeRatioRecommended,
		DrawdownDiff:      backtest1.MaxDrawdownRecommended - backtest2.MaxDrawdownRecommended,
		TaxDiff:           backtest1.TaxSavingsAccumulated - backtest2.TaxSavingsAccumulated,
		CostDiff:          backtest1.TransactionCosts - backtest2.TransactionCosts,
		Reasoning:         fmt.Sprintf("Rec1 Net Benefit: %.2f%%, Rec2 Net Benefit: %.2f%%", backtest1.NetBenefit*100, backtest2.NetBenefit*100),
		CreatedAt:         time.Now(),
	}

	// Save comparison
	err = s.saveComparison(ctx, comparison)
	if err != nil {
		return nil, err
	}

	return comparison, nil
}

func (s *Service) saveComparison(ctx context.Context, comparison *ComparisonResult) error {
	// TODO: Refactor to Hasura GraphQL
	// mutation { insert_backtest_comparisons_one(object: {
	//   id, portfolio_id, recommendation_id_1, recommendation_id_2, winner,
	//   performance_diff, risk_diff, sharpe_ratio_diff, drawdown_diff, tax_diff, cost_diff, reasoning
	// }) { id }}
	query := `
		INSERT INTO backtest_comparisons
		(id, portfolio_id, recommendation_id_1, recommendation_id_2, winner, performance_diff, 
		 risk_diff, sharpe_ratio_diff, drawdown_diff, tax_diff, cost_diff, reasoning, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := s.db.ExecContext(ctx, query,
		comparison.ID, comparison.PortfolioID, comparison.RecommendationID1, comparison.RecommendationID2,
		comparison.Winner, comparison.PerformanceDiff, comparison.RiskDiff, comparison.SharpeRatioDiff,
		comparison.DrawdownDiff, comparison.TaxDiff, comparison.CostDiff, comparison.Reasoning, comparison.CreatedAt)

	return err
}

// ============================================================================
// Risk Analytics
// ============================================================================

// CalculateRiskMetrics calculates comprehensive risk metrics for a portfolio
func (s *Service) CalculateRiskMetrics(ctx context.Context, portfolioID string) (*PortfolioRiskMetrics, error) {
	portfolio, err := s.GetPortfolio(ctx, portfolioID)
	if err != nil {
		return nil, err
	}

	holdings, err := s.GetHoldings(ctx, portfolioID)
	if err != nil {
		return nil, err
	}

	// Calculate metrics
	metrics := &PortfolioRiskMetrics{
		ID:          uuid.New(),
		PortfolioID: portfolio.ID,
		AsOfDate:    time.Now(),
	}

	// Placeholder calculations
	metrics.ExpectedReturn = 0.08 // 8% annual
	metrics.Volatility = 0.12     // 12% annual
	metrics.SharpeRatio = (metrics.ExpectedReturn - 0.02) / metrics.Volatility
	metrics.MaxDrawdown = 0.15
	metrics.VaR95 = -0.15
	metrics.CVaR95 = -0.20

	// Concentration metrics
	totalValue := 0.0
	for _, h := range holdings {
		totalValue += h.CurrentValue
	}
	if len(holdings) > 0 && totalValue > 0 {
		sort.Slice(holdings, func(i, j int) bool {
			return holdings[i].CurrentValue > holdings[j].CurrentValue
		})

		top1 := holdings[0].CurrentValue / totalValue
		top5 := 0.0
		top10 := 0.0

		for i := 0; i < len(holdings) && i < 5; i++ {
			top5 += holdings[i].CurrentValue
		}
		top5 /= totalValue

		for i := 0; i < len(holdings) && i < 10; i++ {
			top10 += holdings[i].CurrentValue
		}
		top10 /= totalValue

		metrics.Concentration = Concentration{
			Top1Holding:   top1,
			Top5Holdings:  top5,
			Top10Holdings: top10,
		}
	}

	// Save metrics
	err = s.saveRiskMetrics(ctx, metrics)
	if err != nil {
		return nil, err
	}

	return metrics, nil
}

func (s *Service) saveRiskMetrics(ctx context.Context, metrics *PortfolioRiskMetrics) error {
	// TODO: Refactor to Hasura GraphQL
	// mutation { insert_portfolio_risk_metrics_one(object: {
	//   id, portfolio_id, as_of_date, expected_return, volatility, sharpe_ratio,
	//   beta, alpha, max_drawdown, var_95, cvar_95, diversification_ratio, herfindahl_index
	// }) { id }}
	query := `
		INSERT INTO portfolio_risk_metrics
		(id, portfolio_id, as_of_date, expected_return, volatility, sharpe_ratio, sortino_ratio,
		 beta, alpha, max_drawdown, var_95, cvar_95, diversification_ratio, herfindahl_index,
		 top_10_holdings, top_5_holdings, top_1_holding, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, NOW())
	`

	_, err := s.db.ExecContext(ctx, query,
		metrics.ID, metrics.PortfolioID, metrics.AsOfDate,
		metrics.ExpectedReturn, metrics.Volatility, metrics.SharpeRatio, metrics.SortinoRatio,
		metrics.Beta, metrics.Alpha, metrics.MaxDrawdown, metrics.VaR95, metrics.CVaR95,
		metrics.DiversificationRatio, metrics.HerfindahlIndex,
		metrics.Concentration.Top10Holdings, metrics.Concentration.Top5Holdings, metrics.Concentration.Top1Holding)

	return err
}

// GetRiskMetrics retrieves latest risk metrics for a portfolio
func (s *Service) GetRiskMetrics(ctx context.Context, portfolioID string) (*PortfolioRiskMetrics, error) {
	// TODO: Refactor to Hasura GraphQL
	// query { portfolio_risk_metrics(
	//   where: {portfolio_id: {_eq: $portfolio_id}}
	//   order_by: {as_of_date: desc}
	//   limit: 1
	// ) { id portfolio_id as_of_date expected_return volatility sharpe_ratio ... }}
	metrics := &PortfolioRiskMetrics{}
	query := `
		SELECT id, portfolio_id, as_of_date, expected_return, volatility, sharpe_ratio, sortino_ratio,
		       beta, alpha, max_drawdown, var_95, cvar_95, diversification_ratio, herfindahl_index, created_at
		FROM portfolio_risk_metrics
		WHERE portfolio_id = $1
		ORDER BY as_of_date DESC
		LIMIT 1
	`
	err := s.db.GetContext(ctx, metrics, query, portfolioID)
	return metrics, err
}
