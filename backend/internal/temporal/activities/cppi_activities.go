package activities

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"go.temporal.io/sdk/activity"
)

// CPPIActivities contains activities for CPPI floor protection
type CPPIActivities struct {
	db *sqlx.DB
}

// NewCPPIActivities creates a new CPPIActivities instance
func NewCPPIActivities(db *sqlx.DB) *CPPIActivities {
	return &CPPIActivities{db: db}
}

// CPPITrade represents a trade for CPPI rebalancing
type CPPITrade struct {
	Side        string  `json:"side"`
	Ticker      string  `json:"ticker"`
	Quantity    float64 `json:"quantity"`
	EstValueUSD float64 `json:"est_value_usd"`
	Reason      string  `json:"reason"`
}

// GetPortfolioNAVInput is the input for GetPortfolioNAVActivity
type GetPortfolioNAVInput struct {
	TenantID    string `json:"tenant_id"`
	PortfolioID string `json:"portfolio_id"`
}

// GetPortfolioNAVOutput is the output from GetPortfolioNAVActivity
type GetPortfolioNAVOutput struct {
	NAV           float64 `json:"nav"`
	PositionCount int     `json:"position_count"`
	AsOfTime      string  `json:"as_of_time"`
}

// GetPortfolioNAVActivity retrieves the current NAV for a portfolio
func (a *CPPIActivities) GetPortfolioNAVActivity(ctx context.Context, input GetPortfolioNAVInput) (*GetPortfolioNAVOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Getting portfolio NAV", "portfolioID", input.PortfolioID)

	// Query to calculate NAV from positions
	query := `
		SELECT 
			COALESCE(SUM(p.quantity * COALESCE(pr.last_price, p.cost_basis / NULLIF(p.quantity, 0))), 0) as nav,
			COUNT(*) as position_count
		FROM positions p
		LEFT JOIN prices pr ON p.ticker = pr.ticker
		WHERE p.tenant_id = $1 AND p.portfolio_id = $2 AND p.quantity > 0
	`

	var nav float64
	var positionCount int
	row := a.db.QueryRowContext(ctx, query, input.TenantID, input.PortfolioID)
	if err := row.Scan(&nav, &positionCount); err != nil {
		return nil, fmt.Errorf("failed to get portfolio NAV: %w", err)
	}

	return &GetPortfolioNAVOutput{
		NAV:           nav,
		PositionCount: positionCount,
	}, nil
}

// GetCurrentAllocationsInput is the input for GetCurrentAllocationsActivity
type GetCurrentAllocationsInput struct {
	TenantID       string `json:"tenant_id"`
	PortfolioID    string `json:"portfolio_id"`
	RiskFreeTicker string `json:"risk_free_ticker"`
}

// GetCurrentAllocationsOutput is the output from GetCurrentAllocationsActivity
type GetCurrentAllocationsOutput struct {
	RiskyAllocation    float64 `json:"risky_allocation"`
	RiskFreeAllocation float64 `json:"risk_free_allocation"`
	CashBalance        float64 `json:"cash_balance"`
}

// GetCurrentAllocationsActivity retrieves the current risky vs risk-free allocation
func (a *CPPIActivities) GetCurrentAllocationsActivity(ctx context.Context, input GetCurrentAllocationsInput) (*GetCurrentAllocationsOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Getting current allocations", "portfolioID", input.PortfolioID)

	// Query to calculate allocation split
	query := `
		SELECT 
			COALESCE(SUM(CASE WHEN p.ticker = $3 OR p.asset_class = 'CASH' THEN p.quantity * COALESCE(pr.last_price, 1) ELSE 0 END), 0) as risk_free,
			COALESCE(SUM(CASE WHEN p.ticker != $3 AND p.asset_class != 'CASH' THEN p.quantity * COALESCE(pr.last_price, p.cost_basis / NULLIF(p.quantity, 0)) ELSE 0 END), 0) as risky,
			COALESCE(SUM(CASE WHEN p.asset_class = 'CASH' THEN p.quantity ELSE 0 END), 0) as cash
		FROM positions p
		LEFT JOIN prices pr ON p.ticker = pr.ticker
		WHERE p.tenant_id = $1 AND p.portfolio_id = $2 AND p.quantity > 0
	`

	var riskFree, risky, cash float64
	row := a.db.QueryRowContext(ctx, query, input.TenantID, input.PortfolioID, input.RiskFreeTicker)
	if err := row.Scan(&riskFree, &risky, &cash); err != nil {
		return nil, fmt.Errorf("failed to get allocations: %w", err)
	}

	return &GetCurrentAllocationsOutput{
		RiskyAllocation:    risky,
		RiskFreeAllocation: riskFree,
		CashBalance:        cash,
	}, nil
}

// CPPIRebalanceInput is the input for GenerateCPPIRebalanceTradesActivity
type CPPIRebalanceInput struct {
	TenantID                  string  `json:"tenant_id"`
	PortfolioID               string  `json:"portfolio_id"`
	TargetRiskyAllocation     float64 `json:"target_risky_allocation"`
	TargetRiskFreeAllocation  float64 `json:"target_risk_free_allocation"`
	CurrentRiskyAllocation    float64 `json:"current_risky_allocation"`
	CurrentRiskFreeAllocation float64 `json:"current_risk_free_allocation"`
	RiskFreeTicker            string  `json:"risk_free_ticker"`
}

// CPPIRebalanceOutput is the output from GenerateCPPIRebalanceTradesActivity
type CPPIRebalanceOutput struct {
	Trades []CPPITrade `json:"trades"`
}

// GenerateCPPIRebalanceTradesActivity generates trades to rebalance to CPPI targets
func (a *CPPIActivities) GenerateCPPIRebalanceTradesActivity(ctx context.Context, input CPPIRebalanceInput) (*CPPIRebalanceOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Generating CPPI rebalance trades",
		"portfolioID", input.PortfolioID,
		"targetRisky", input.TargetRiskyAllocation,
		"currentRisky", input.CurrentRiskyAllocation)

	var trades []CPPITrade

	riskyDelta := input.TargetRiskyAllocation - input.CurrentRiskyAllocation

	if riskyDelta < 0 {
		// Need to reduce risky, increase risk-free
		// Sell risky assets proportionally, buy risk-free
		sellAmount := -riskyDelta

		// Get risky positions to sell
		query := `
			SELECT p.ticker, p.quantity, COALESCE(pr.last_price, p.cost_basis / NULLIF(p.quantity, 0)) as price
			FROM positions p
			LEFT JOIN prices pr ON p.ticker = pr.ticker
			WHERE p.tenant_id = $1 AND p.portfolio_id = $2 
			  AND p.ticker != $3 AND p.asset_class != 'CASH'
			  AND p.quantity > 0
			ORDER BY p.quantity * COALESCE(pr.last_price, p.cost_basis / NULLIF(p.quantity, 0)) DESC
		`

		rows, err := a.db.QueryContext(ctx, query, input.TenantID, input.PortfolioID, input.RiskFreeTicker)
		if err != nil {
			return nil, fmt.Errorf("failed to query risky positions: %w", err)
		}
		defer rows.Close()

		remaining := sellAmount
		for rows.Next() && remaining > 0 {
			var ticker string
			var quantity, price float64
			if err := rows.Scan(&ticker, &quantity, &price); err != nil {
				continue
			}

			posValue := quantity * price
			sellValue := min(posValue, remaining)
			sellQty := sellValue / price

			trades = append(trades, CPPITrade{
				Side:        "SELL",
				Ticker:      ticker,
				Quantity:    sellQty,
				EstValueUSD: sellValue,
				Reason:      "CPPI_RISK_REDUCTION",
			})

			remaining -= sellValue
		}

		// Buy risk-free asset
		if sellAmount > 0 {
			// Get risk-free price
			var rfPrice float64
			err := a.db.QueryRowContext(ctx,
				"SELECT COALESCE(last_price, 100) FROM prices WHERE ticker = $1",
				input.RiskFreeTicker).Scan(&rfPrice)
			if err != nil {
				rfPrice = 100.0 // Default
			}

			trades = append(trades, CPPITrade{
				Side:        "BUY",
				Ticker:      input.RiskFreeTicker,
				Quantity:    sellAmount / rfPrice,
				EstValueUSD: sellAmount,
				Reason:      "CPPI_FLOOR_PROTECTION",
			})
		}
	} else if riskyDelta > 0 {
		// Need to increase risky, reduce risk-free
		buyAmount := riskyDelta

		// Sell risk-free first
		var rfQty, rfPrice float64
		err := a.db.QueryRowContext(ctx, `
			SELECT p.quantity, COALESCE(pr.last_price, 100)
			FROM positions p
			LEFT JOIN prices pr ON p.ticker = pr.ticker
			WHERE p.tenant_id = $1 AND p.portfolio_id = $2 AND p.ticker = $3
		`, input.TenantID, input.PortfolioID, input.RiskFreeTicker).Scan(&rfQty, &rfPrice)
		if err == nil && rfQty > 0 {
			sellValue := min(rfQty*rfPrice, buyAmount)
			sellQty := sellValue / rfPrice

			trades = append(trades, CPPITrade{
				Side:        "SELL",
				Ticker:      input.RiskFreeTicker,
				Quantity:    sellQty,
				EstValueUSD: sellValue,
				Reason:      "CPPI_CUSHION_DEPLOYMENT",
			})
		}

		// Buy risky assets (for now, use a diversified ETF approach)
		// In production, this would be more sophisticated
		trades = append(trades, CPPITrade{
			Side:        "BUY",
			Ticker:      "VTI",           // Total market ETF as placeholder
			Quantity:    buyAmount / 200, // Approximate price
			EstValueUSD: buyAmount,
			Reason:      "CPPI_RISK_INCREASE",
		})
	}

	return &CPPIRebalanceOutput{Trades: trades}, nil
}

// EmergencyLiquidationInput is the input for EmergencyLiquidationActivity
type EmergencyLiquidationInput struct {
	TenantID       string  `json:"tenant_id"`
	PortfolioID    string  `json:"portfolio_id"`
	TargetCash     float64 `json:"target_cash"`
	RiskFreeTicker string  `json:"risk_free_ticker"`
	Reason         string  `json:"reason"`
}

// EmergencyLiquidationOutput is the output from EmergencyLiquidationActivity
type EmergencyLiquidationOutput struct {
	Trades          []CPPITrade `json:"trades"`
	TotalLiquidated float64     `json:"total_liquidated"`
}

// EmergencyLiquidationActivity generates trades to liquidate risky positions to floor
func (a *CPPIActivities) EmergencyLiquidationActivity(ctx context.Context, input EmergencyLiquidationInput) (*EmergencyLiquidationOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Warn("Executing emergency liquidation",
		"portfolioID", input.PortfolioID,
		"targetCash", input.TargetCash,
		"reason", input.Reason)

	var trades []CPPITrade
	var totalLiquidated float64

	// Sell ALL risky positions
	query := `
		SELECT p.ticker, p.quantity, COALESCE(pr.last_price, p.cost_basis / NULLIF(p.quantity, 0)) as price
		FROM positions p
		LEFT JOIN prices pr ON p.ticker = pr.ticker
		WHERE p.tenant_id = $1 AND p.portfolio_id = $2 
		  AND p.ticker != $3 AND p.asset_class != 'CASH'
		  AND p.quantity > 0
	`

	rows, err := a.db.QueryContext(ctx, query, input.TenantID, input.PortfolioID, input.RiskFreeTicker)
	if err != nil {
		return nil, fmt.Errorf("failed to query positions for liquidation: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ticker string
		var quantity, price float64
		if err := rows.Scan(&ticker, &quantity, &price); err != nil {
			continue
		}

		value := quantity * price
		trades = append(trades, CPPITrade{
			Side:        "SELL",
			Ticker:      ticker,
			Quantity:    quantity,
			EstValueUSD: value,
			Reason:      "EMERGENCY_FLOOR_PROTECTION",
		})
		totalLiquidated += value
	}

	// Convert proceeds to risk-free asset
	if totalLiquidated > 0 {
		var rfPrice float64
		err := a.db.QueryRowContext(ctx,
			"SELECT COALESCE(last_price, 100) FROM prices WHERE ticker = $1",
			input.RiskFreeTicker).Scan(&rfPrice)
		if err != nil {
			rfPrice = 100.0
		}

		trades = append(trades, CPPITrade{
			Side:        "BUY",
			Ticker:      input.RiskFreeTicker,
			Quantity:    totalLiquidated / rfPrice,
			EstValueUSD: totalLiquidated,
			Reason:      "FLOOR_PROTECTION_CONVERSION",
		})
	}

	return &EmergencyLiquidationOutput{
		Trades:          trades,
		TotalLiquidated: totalLiquidated,
	}, nil
}

// CPPINotificationInput is the input for NotifyFloorEventActivity
type CPPINotificationInput struct {
	TenantID    string      `json:"tenant_id"`
	PortfolioID string      `json:"portfolio_id"`
	AdvisorID   string      `json:"advisor_id"`
	EventType   string      `json:"event_type"` // FLOOR_BREACH, REBALANCE_REQUIRED, etc.
	NAV         float64     `json:"nav"`
	Floor       float64     `json:"floor"`
	Cushion     float64     `json:"cushion"`
	Purpose     string      `json:"purpose"`
	Trades      []CPPITrade `json:"trades,omitempty"`
}

// NotifyFloorEventActivity sends notifications about CPPI events
func (a *CPPIActivities) NotifyFloorEventActivity(ctx context.Context, input CPPINotificationInput) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Sending CPPI notification",
		"portfolioID", input.PortfolioID,
		"eventType", input.EventType,
		"nav", input.NAV,
		"floor", input.Floor)

	// In production, this would:
	// 1. Send email/SMS to advisor
	// 2. Create in-app notification
	// 3. Update dashboard widgets
	// 4. Potentially notify client based on preferences

	// For now, just log the event
	_, err := a.db.ExecContext(ctx, `
		INSERT INTO cppi_events (
			tenant_id, portfolio_id, advisor_id, event_type, 
			nav, floor_value, cushion, purpose, trades_json, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
	`, input.TenantID, input.PortfolioID, input.AdvisorID, input.EventType,
		input.NAV, input.Floor, input.Cushion, input.Purpose, "[]")

	if err != nil {
		logger.Warn("Failed to log CPPI event (table may not exist)", "error", err)
	}

	return nil
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
