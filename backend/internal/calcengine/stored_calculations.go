package calcengine

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"
)

// ============================================================================
// STORED CALCULATIONS: API → Compute → Store → Access
// ============================================================================
// Pattern: Client calls API → Engine computes → Result stored in StarRocks
//          → Client (or Cube) reads from stored results
// Benefits:
//   - Complex calcs (XIRR, TWR) computed once, served many times
//   - Cube can access pre-computed results for BI
//   - No recalculation on every request
// ============================================================================

// StoredCalcRequest represents a request to compute and store a calculation
type StoredCalcRequest struct {
	// Required: tenant isolation
	TenantID     string `json:"tenant_id"`
	DatasourceID string `json:"datasource_id"`

	// Calculation details
	CalcType   string                 `json:"calc_type"`   // XIRR, TWR, IRR, etc.
	EntityType string                 `json:"entity_type"` // portfolio, account, family
	EntityID   string                 `json:"entity_id"`
	AsOfDate   time.Time              `json:"as_of_date"`
	Params     map[string]interface{} `json:"params,omitempty"`

	// Options
	ForceRecalc bool `json:"force_recalc"` // Recalculate even if cached
	Async       bool `json:"async"`        // Return immediately, compute in background
}

// StoredCalcResult represents a stored calculation result
type StoredCalcResult struct {
	// Identity
	ResultID     string    `json:"result_id"`
	TenantID     string    `json:"tenant_id"`
	DatasourceID string    `json:"datasource_id"`
	CalcType     string    `json:"calc_type"`
	EntityType   string    `json:"entity_type"`
	EntityID     string    `json:"entity_id"`
	AsOfDate     time.Time `json:"as_of_date"`

	// Result
	Value        float64                `json:"value"`
	Breakdown    map[string]interface{} `json:"breakdown,omitempty"`
	Status       string                 `json:"status"` // pending, success, failed
	ErrorMessage string                 `json:"error_message,omitempty"`

	// Metadata
	ComputedAt    time.Time `json:"computed_at"`
	ComputeTimeMS int64     `json:"compute_time_ms"`
	InputHash     string    `json:"input_hash"` // Hash of inputs for cache invalidation
	ExpiresAt     time.Time `json:"expires_at"`
}

// ComputeAndStore computes a calculation and stores the result
func (e *UnifiedCalcEngine) ComputeAndStore(ctx context.Context, req *StoredCalcRequest) (*StoredCalcResult, error) {
	start := time.Now()

	// Check for existing valid result (unless force recalc)
	if !req.ForceRecalc {
		existing, err := e.getStoredResult(ctx, req)
		if err == nil && existing != nil && existing.Status == "success" {
			return existing, nil
		}
	}

	// Create result record (status: pending)
	result := &StoredCalcResult{
		ResultID:     generateResultID(req),
		TenantID:     req.TenantID,
		DatasourceID: req.DatasourceID,
		CalcType:     req.CalcType,
		EntityType:   req.EntityType,
		EntityID:     req.EntityID,
		AsOfDate:     req.AsOfDate,
		Status:       "pending",
		InputHash:    hashInputs(req),
	}

	// If async, save pending and return
	if req.Async {
		if err := e.saveStoredResult(ctx, result); err != nil {
			return nil, err
		}
		// Queue for background processing
		go e.computeAsync(context.Background(), req, result)
		return result, nil
	}

	// Compute synchronously
	if err := e.computeCalc(ctx, req, result); err != nil {
		result.Status = "failed"
		result.ErrorMessage = err.Error()
	} else {
		result.Status = "success"
	}

	result.ComputedAt = time.Now()
	result.ComputeTimeMS = time.Since(start).Milliseconds()
	result.ExpiresAt = time.Now().Add(24 * time.Hour) // Default 24h TTL

	// Store result
	if err := e.saveStoredResult(ctx, result); err != nil {
		return nil, fmt.Errorf("failed to store result: %w", err)
	}

	return result, nil
}

// GetStoredResult retrieves a stored calculation result
func (e *UnifiedCalcEngine) GetStoredResult(ctx context.Context, tenantID, datasourceID, calcType, entityType, entityID string, asOfDate time.Time) (*StoredCalcResult, error) {
	req := &StoredCalcRequest{
		TenantID:     tenantID,
		DatasourceID: datasourceID,
		CalcType:     calcType,
		EntityType:   entityType,
		EntityID:     entityID,
		AsOfDate:     asOfDate,
	}
	return e.getStoredResult(ctx, req)
}

// computeCalc performs the actual calculation
func (e *UnifiedCalcEngine) computeCalc(ctx context.Context, req *StoredCalcRequest, result *StoredCalcResult) error {
	switch req.CalcType {
	case "XIRR":
		return e.computeXIRR(ctx, req, result)
	case "TWR":
		return e.computeTWR(ctx, req, result)
	case "IRR":
		return e.computeIRR(ctx, req, result)
	case "CAGR":
		return e.computeCAGR(ctx, req, result)
	case "SHARPE":
		return e.computeSharpe(ctx, req, result)
	case "SORTINO":
		return e.computeSortino(ctx, req, result)
	default:
		return fmt.Errorf("unsupported calculation type: %s", req.CalcType)
	}
}

// ============================================================================
// XIRR CALCULATION
// ============================================================================

// CashFlow represents a cash flow for XIRR calculation
type CashFlow struct {
	Date   time.Time `json:"date"`
	Amount float64   `json:"amount"` // Negative = outflow, Positive = inflow
}

// computeXIRR calculates the Internal Rate of Return with irregular cash flows
func (e *UnifiedCalcEngine) computeXIRR(ctx context.Context, req *StoredCalcRequest, result *StoredCalcResult) error {
	// Fetch cash flows from StarRocks
	cashFlows, err := e.fetchCashFlows(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to fetch cash flows: %w", err)
	}

	if len(cashFlows) < 2 {
		return fmt.Errorf("XIRR requires at least 2 cash flows")
	}

	// Add terminal value (current portfolio value as positive inflow)
	terminalValue, err := e.fetchTerminalValue(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to fetch terminal value: %w", err)
	}

	cashFlows = append(cashFlows, CashFlow{
		Date:   req.AsOfDate,
		Amount: terminalValue,
	})

	// Sort by date
	sort.Slice(cashFlows, func(i, j int) bool {
		return cashFlows[i].Date.Before(cashFlows[j].Date)
	})

	// Calculate XIRR using Newton-Raphson method
	xirr, err := calculateXIRR(cashFlows)
	if err != nil {
		return fmt.Errorf("XIRR calculation failed: %w", err)
	}

	result.Value = xirr
	result.Breakdown = map[string]interface{}{
		"cash_flow_count": len(cashFlows),
		"first_date":      cashFlows[0].Date.Format("2006-01-02"),
		"last_date":       cashFlows[len(cashFlows)-1].Date.Format("2006-01-02"),
		"total_invested":  sumNegative(cashFlows),
		"total_returned":  sumPositive(cashFlows),
		"terminal_value":  terminalValue,
	}

	return nil
}

// calculateXIRR implements Newton-Raphson method for XIRR
func calculateXIRR(cashFlows []CashFlow) (float64, error) {
	if len(cashFlows) == 0 {
		return 0, fmt.Errorf("no cash flows provided")
	}

	// Initial guess
	rate := 0.1

	// Reference date (first cash flow)
	refDate := cashFlows[0].Date

	// Newton-Raphson iteration
	maxIterations := 100
	tolerance := 1e-10

	for i := 0; i < maxIterations; i++ {
		npv := 0.0
		dnpv := 0.0 // Derivative of NPV

		for _, cf := range cashFlows {
			years := cf.Date.Sub(refDate).Hours() / (24 * 365.25)

			if rate == -1 {
				rate = -0.999999 // Avoid division by zero
			}

			factor := math.Pow(1+rate, years)
			npv += cf.Amount / factor
			dnpv -= years * cf.Amount / (factor * (1 + rate))
		}

		if math.Abs(npv) < tolerance {
			return rate, nil
		}

		if dnpv == 0 {
			return 0, fmt.Errorf("derivative is zero, cannot continue")
		}

		newRate := rate - npv/dnpv

		// Bound the rate to reasonable values
		if newRate < -0.999 {
			newRate = -0.999
		} else if newRate > 10 {
			newRate = 10
		}

		if math.Abs(newRate-rate) < tolerance {
			return newRate, nil
		}

		rate = newRate
	}

	return rate, fmt.Errorf("XIRR did not converge after %d iterations", maxIterations)
}

// fetchCashFlows retrieves cash flows from StarRocks
func (e *UnifiedCalcEngine) fetchCashFlows(ctx context.Context, req *StoredCalcRequest) ([]CashFlow, error) {
	query := fmt.Sprintf(`
		SELECT transaction_date, 
		       CASE 
		           WHEN transaction_type IN ('BUY', 'DEPOSIT', 'CONTRIBUTION') THEN -ABS(quantity * price + COALESCE(fees, 0))
		           WHEN transaction_type IN ('SELL', 'WITHDRAWAL', 'DISTRIBUTION') THEN ABS(quantity * price - COALESCE(fees, 0))
		           WHEN transaction_type = 'DIVIDEND' THEN ABS(quantity * price)
		           ELSE 0
		       END as amount
		FROM %s.transactions
		WHERE tenant_id = '%s'
		  AND datasource_id = '%s'
		  AND %s = '%s'
		  AND transaction_date <= '%s'
		ORDER BY transaction_date
	`, e.config.HotDatabase, req.TenantID, req.DatasourceID,
		req.EntityType+"_id", req.EntityID,
		req.AsOfDate.Format("2006-01-02"))

	rows, err := e.starrocks.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cashFlows []CashFlow
	for rows.Next() {
		var cf CashFlow
		if err := rows.Scan(&cf.Date, &cf.Amount); err != nil {
			return nil, err
		}
		if cf.Amount != 0 {
			cashFlows = append(cashFlows, cf)
		}
	}

	return cashFlows, rows.Err()
}

// fetchTerminalValue gets current portfolio value
func (e *UnifiedCalcEngine) fetchTerminalValue(ctx context.Context, req *StoredCalcRequest) (float64, error) {
	query := fmt.Sprintf(`
		SELECT COALESCE(SUM(h.quantity * p.price * COALESCE(fx.rate, 1.0)), 0)
		FROM %s.holdings h
		LEFT JOIN %s.prices p ON h.ticker = p.ticker AND p.price_date = '%s'
		LEFT JOIN %s.fx_rates fx ON h.currency = fx.from_currency 
		    AND fx.to_currency = 'USD' AND fx.rate_date = '%s'
		WHERE h.tenant_id = '%s'
		  AND h.datasource_id = '%s'
		  AND h.%s = '%s'
	`, e.config.HotDatabase, e.config.HotDatabase, req.AsOfDate.Format("2006-01-02"),
		e.config.HotDatabase, req.AsOfDate.Format("2006-01-02"),
		req.TenantID, req.DatasourceID,
		req.EntityType+"_id", req.EntityID)

	var value float64
	err := e.starrocks.QueryRowContext(ctx, query).Scan(&value)
	return value, err
}

// ============================================================================
// TIME-WEIGHTED RETURN (TWR)
// ============================================================================

func (e *UnifiedCalcEngine) computeTWR(ctx context.Context, req *StoredCalcRequest, result *StoredCalcResult) error {
	startDate := getTimeParam(req.Params, "start_date", time.Now().AddDate(-1, 0, 0))

	query := fmt.Sprintf(`
		WITH daily_values AS (
			SELECT 
				as_of_date,
				nav_value,
				inflows,
				outflows,
				LAG(nav_value) OVER (ORDER BY as_of_date) as prev_nav
			FROM %s.portfolio_nav
			WHERE tenant_id = '%s'
			  AND datasource_id = '%s'
			  AND portfolio_id = '%s'
			  AND as_of_date BETWEEN '%s' AND '%s'
		),
		daily_returns AS (
			SELECT 
				as_of_date,
				CASE 
					WHEN prev_nav + inflows > 0 
					THEN (nav_value - prev_nav - inflows + outflows) / (prev_nav + inflows)
					ELSE 0 
				END as daily_return
			FROM daily_values
			WHERE prev_nav IS NOT NULL
		)
		SELECT 
			EXP(SUM(LN(1 + daily_return))) - 1 as twr,
			COUNT(*) as periods
		FROM daily_returns
		WHERE daily_return > -1
	`, e.config.HotDatabase, req.TenantID, req.DatasourceID, req.EntityID,
		startDate.Format("2006-01-02"), req.AsOfDate.Format("2006-01-02"))

	var twr float64
	var periods int
	if err := e.starrocks.QueryRowContext(ctx, query).Scan(&twr, &periods); err != nil {
		return err
	}

	result.Value = twr
	result.Breakdown = map[string]interface{}{
		"start_date": startDate.Format("2006-01-02"),
		"end_date":   req.AsOfDate.Format("2006-01-02"),
		"periods":    periods,
	}

	return nil
}

// ============================================================================
// OTHER CALCULATIONS (IRR, CAGR, SHARPE, SORTINO)
// ============================================================================

func (e *UnifiedCalcEngine) computeIRR(ctx context.Context, req *StoredCalcRequest, result *StoredCalcResult) error {
	// IRR is XIRR with periodic cash flows - delegate to XIRR
	return e.computeXIRR(ctx, req, result)
}

func (e *UnifiedCalcEngine) computeCAGR(ctx context.Context, req *StoredCalcRequest, result *StoredCalcResult) error {
	startDate := getTimeParam(req.Params, "start_date", time.Now().AddDate(-1, 0, 0))

	query := fmt.Sprintf(`
		SELECT 
			first.nav_value as start_value,
			last.nav_value as end_value
		FROM 
			(SELECT nav_value FROM %s.portfolio_nav 
			 WHERE tenant_id = '%s' AND datasource_id = '%s' AND portfolio_id = '%s'
			 AND as_of_date >= '%s' ORDER BY as_of_date LIMIT 1) first,
			(SELECT nav_value FROM %s.portfolio_nav 
			 WHERE tenant_id = '%s' AND datasource_id = '%s' AND portfolio_id = '%s'
			 AND as_of_date <= '%s' ORDER BY as_of_date DESC LIMIT 1) last
	`, e.config.HotDatabase, req.TenantID, req.DatasourceID, req.EntityID, startDate.Format("2006-01-02"),
		e.config.HotDatabase, req.TenantID, req.DatasourceID, req.EntityID, req.AsOfDate.Format("2006-01-02"))

	var startValue, endValue float64
	if err := e.starrocks.QueryRowContext(ctx, query).Scan(&startValue, &endValue); err != nil {
		return err
	}

	years := req.AsOfDate.Sub(startDate).Hours() / (24 * 365.25)
	if years <= 0 || startValue <= 0 {
		return fmt.Errorf("invalid date range or start value")
	}

	cagr := math.Pow(endValue/startValue, 1/years) - 1

	result.Value = cagr
	result.Breakdown = map[string]interface{}{
		"start_value": startValue,
		"end_value":   endValue,
		"years":       years,
	}

	return nil
}

func (e *UnifiedCalcEngine) computeSharpe(ctx context.Context, req *StoredCalcRequest, result *StoredCalcResult) error {
	riskFreeRate := 0.04 // Default 4% risk-free rate
	if rfr, ok := req.Params["risk_free_rate"].(float64); ok {
		riskFreeRate = rfr
	}

	query := fmt.Sprintf(`
		WITH daily_returns AS (
			SELECT 
				(nav_value - LAG(nav_value) OVER (ORDER BY as_of_date)) / 
					NULLIF(LAG(nav_value) OVER (ORDER BY as_of_date), 0) as daily_return
			FROM %s.portfolio_nav
			WHERE tenant_id = '%s'
			  AND datasource_id = '%s'
			  AND portfolio_id = '%s'
			  AND as_of_date >= DATE_SUB('%s', INTERVAL 1 YEAR)
		)
		SELECT 
			AVG(daily_return) * 252 as annualized_return,
			STDDEV(daily_return) * SQRT(252) as annualized_vol
		FROM daily_returns
		WHERE daily_return IS NOT NULL
	`, e.config.HotDatabase, req.TenantID, req.DatasourceID, req.EntityID,
		req.AsOfDate.Format("2006-01-02"))

	var annReturn, annVol float64
	if err := e.starrocks.QueryRowContext(ctx, query).Scan(&annReturn, &annVol); err != nil {
		return err
	}

	if annVol == 0 {
		return fmt.Errorf("volatility is zero, cannot compute Sharpe ratio")
	}

	sharpe := (annReturn - riskFreeRate) / annVol

	result.Value = sharpe
	result.Breakdown = map[string]interface{}{
		"annualized_return": annReturn,
		"annualized_vol":    annVol,
		"risk_free_rate":    riskFreeRate,
	}

	return nil
}

func (e *UnifiedCalcEngine) computeSortino(ctx context.Context, req *StoredCalcRequest, result *StoredCalcResult) error {
	targetReturn := 0.0
	if tr, ok := req.Params["target_return"].(float64); ok {
		targetReturn = tr
	}

	query := fmt.Sprintf(`
		WITH daily_returns AS (
			SELECT 
				(nav_value - LAG(nav_value) OVER (ORDER BY as_of_date)) / 
					NULLIF(LAG(nav_value) OVER (ORDER BY as_of_date), 0) as daily_return
			FROM %s.portfolio_nav
			WHERE tenant_id = '%s'
			  AND datasource_id = '%s'
			  AND portfolio_id = '%s'
			  AND as_of_date >= DATE_SUB('%s', INTERVAL 1 YEAR)
		),
		downside AS (
			SELECT 
				AVG(daily_return) * 252 as annualized_return,
				SQRT(AVG(CASE WHEN daily_return < %f/252 THEN POW(daily_return - %f/252, 2) ELSE 0 END) * 252) as downside_dev
			FROM daily_returns
			WHERE daily_return IS NOT NULL
		)
		SELECT annualized_return, downside_dev FROM downside
	`, e.config.HotDatabase, req.TenantID, req.DatasourceID, req.EntityID,
		req.AsOfDate.Format("2006-01-02"), targetReturn, targetReturn)

	var annReturn, downsideDev float64
	if err := e.starrocks.QueryRowContext(ctx, query).Scan(&annReturn, &downsideDev); err != nil {
		return err
	}

	if downsideDev == 0 {
		return fmt.Errorf("downside deviation is zero")
	}

	sortino := (annReturn - targetReturn) / downsideDev

	result.Value = sortino
	result.Breakdown = map[string]interface{}{
		"annualized_return": annReturn,
		"downside_dev":      downsideDev,
		"target_return":     targetReturn,
	}

	return nil
}

// ============================================================================
// STORAGE OPERATIONS
// ============================================================================

func (e *UnifiedCalcEngine) saveStoredResult(ctx context.Context, result *StoredCalcResult) error {
	breakdownJSON, _ := json.Marshal(result.Breakdown)

	query := `
		REPLACE INTO stored_calc_results (
			result_id, tenant_id, datasource_id, calc_type, entity_type, entity_id,
			as_of_date, value, breakdown, status, error_message, computed_at,
			compute_time_ms, input_hash, expires_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := e.starrocks.ExecContext(ctx, query,
		result.ResultID, result.TenantID, result.DatasourceID, result.CalcType,
		result.EntityType, result.EntityID, result.AsOfDate, result.Value,
		string(breakdownJSON), result.Status, result.ErrorMessage, result.ComputedAt,
		result.ComputeTimeMS, result.InputHash, result.ExpiresAt,
	)
	return err
}

func (e *UnifiedCalcEngine) getStoredResult(ctx context.Context, req *StoredCalcRequest) (*StoredCalcResult, error) {
	query := `
		SELECT result_id, tenant_id, datasource_id, calc_type, entity_type, entity_id,
		       as_of_date, value, breakdown, status, error_message, computed_at,
		       compute_time_ms, input_hash, expires_at
		FROM stored_calc_results
		WHERE tenant_id = ?
		  AND datasource_id = ?
		  AND calc_type = ?
		  AND entity_type = ?
		  AND entity_id = ?
		  AND as_of_date = ?
		  AND expires_at > NOW()
		ORDER BY computed_at DESC
		LIMIT 1
	`

	var result StoredCalcResult
	var breakdownJSON string

	err := e.starrocks.QueryRowContext(ctx, query,
		req.TenantID, req.DatasourceID, req.CalcType,
		req.EntityType, req.EntityID, req.AsOfDate,
	).Scan(
		&result.ResultID, &result.TenantID, &result.DatasourceID, &result.CalcType,
		&result.EntityType, &result.EntityID, &result.AsOfDate, &result.Value,
		&breakdownJSON, &result.Status, &result.ErrorMessage, &result.ComputedAt,
		&result.ComputeTimeMS, &result.InputHash, &result.ExpiresAt,
	)
	if err != nil {
		return nil, err
	}

	json.Unmarshal([]byte(breakdownJSON), &result.Breakdown)
	return &result, nil
}

// computeAsync runs calculation in background
func (e *UnifiedCalcEngine) computeAsync(ctx context.Context, req *StoredCalcRequest, result *StoredCalcResult) {
	start := time.Now()

	if err := e.computeCalc(ctx, req, result); err != nil {
		result.Status = "failed"
		result.ErrorMessage = err.Error()
	} else {
		result.Status = "success"
	}

	result.ComputedAt = time.Now()
	result.ComputeTimeMS = time.Since(start).Milliseconds()
	result.ExpiresAt = time.Now().Add(24 * time.Hour)

	_ = e.saveStoredResult(ctx, result)
}

// Helper functions
func generateResultID(req *StoredCalcRequest) string {
	return fmt.Sprintf("%s_%s_%s_%s_%s_%s",
		req.TenantID[:8], req.DatasourceID[:8], req.CalcType,
		req.EntityType, req.EntityID, req.AsOfDate.Format("20060102"))
}

func hashInputs(req *StoredCalcRequest) string {
	data, _ := json.Marshal(req)
	return fmt.Sprintf("%x", data)[:16]
}

func sumNegative(flows []CashFlow) float64 {
	var sum float64
	for _, cf := range flows {
		if cf.Amount < 0 {
			sum += cf.Amount
		}
	}
	return sum
}

func sumPositive(flows []CashFlow) float64 {
	var sum float64
	for _, cf := range flows {
		if cf.Amount > 0 {
			sum += cf.Amount
		}
	}
	return sum
}
