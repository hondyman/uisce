package activities

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"

	"github.com/hondyman/semlayer/backend/internal/factor"
	"github.com/hondyman/semlayer/backend/internal/optimizer"
)

// RebalanceActivities provides all activities for the rebalance workflow.
type RebalanceActivities struct {
	// Dependencies would be injected here (DB, services, etc.)
}

// NewRebalanceActivities creates a new RebalanceActivities instance.
func NewRebalanceActivities() *RebalanceActivities {
	return &RebalanceActivities{}
}

// CheckDriftInput is the input for the CheckDriftActivity.
type CheckDriftInput struct {
	PortfolioID string  `json:"portfolio_id"`
	TenantID    string  `json:"tenant_id"`
	Threshold   float64 `json:"threshold"`
}

// CheckDriftOutput is the output of the CheckDriftActivity.
type CheckDriftOutput struct {
	DriftDetected bool                  `json:"drift_detected"`
	DriftReport   optimizer.DriftReport `json:"drift_report"`
}

// CheckDriftActivity checks if a portfolio has drifted beyond threshold.
func (a *RebalanceActivities) CheckDriftActivity(ctx context.Context, input CheckDriftInput) (*CheckDriftOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("CheckDriftActivity started", "portfolio_id", input.PortfolioID)

	// In production, this would query the portfolio and calculate drift
	// For now, return a mock drift report
	driftReport := optimizer.DriftReport{
		PortfolioID:   input.PortfolioID,
		TenantID:      input.TenantID,
		DriftPercent:  0.05,
		TrackingError: 0.02,
		ComputedAt:    time.Now(),
		Exposures: []optimizer.Exposure{
			{
				Symbol:       "SPY",
				CurrentWgt:   0.65,
				TargetWgt:    0.60,
				MarketValue:  650000.00,
				DriftPercent: 0.05,
			},
			{
				Symbol:       "EFA",
				CurrentWgt:   0.20,
				TargetWgt:    0.25,
				MarketValue:  200000.00,
				DriftPercent: -0.05,
			},
		},
	}

	return &CheckDriftOutput{
		DriftDetected: driftReport.DriftPercent > input.Threshold,
		DriftReport:   driftReport,
	}, nil
}

// LoadLotsInput is the input for the LoadLotsActivity.
type LoadLotsInput struct {
	PortfolioID string `json:"portfolio_id"`
	TenantID    string `json:"tenant_id"`
}

// LoadLotsOutput is the output of the LoadLotsActivity.
type LoadLotsOutput struct {
	Lots []optimizer.Lot `json:"lots"`
}

// LoadLotsActivity loads tax lots for a portfolio.
func (a *RebalanceActivities) LoadLotsActivity(ctx context.Context, input LoadLotsInput) (*LoadLotsOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("LoadLotsActivity started", "portfolio_id", input.PortfolioID)

	// Mock tax lots
	lots := []optimizer.Lot{
		{
			LotID:         "lot-001",
			Symbol:        "AAPL",
			Quantity:      100,
			CostBasis:     15000.00,
			PurchaseDate:  time.Now().AddDate(-2, 0, 0),
			UnrealizedPnL: 3500.00,
			Term:          "long",
		},
		{
			LotID:         "lot-002",
			Symbol:        "MSFT",
			Quantity:      50,
			CostBasis:     12000.00,
			PurchaseDate:  time.Now().AddDate(0, -6, 0),
			UnrealizedPnL: 3000.00,
			Term:          "short",
		},
	}

	return &LoadLotsOutput{Lots: lots}, nil
}

// LoadPricesInput is the input for the LoadPricesActivity.
type LoadPricesInput struct {
	SecurityIDs []string `json:"security_ids"`
}

// LoadPricesOutput is the output of the LoadPricesActivity.
type LoadPricesOutput struct {
	Prices map[string]float64 `json:"prices"`
}

// LoadPricesActivity loads current prices for securities.
func (a *RebalanceActivities) LoadPricesActivity(ctx context.Context, input LoadPricesInput) (*LoadPricesOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("LoadPricesActivity started", "security_count", len(input.SecurityIDs))

	// Mock prices
	prices := map[string]float64{
		"AAPL":  185.00,
		"MSFT":  300.00,
		"GOOGL": 140.00,
		"AMZN":  180.00,
		"META":  500.00,
	}

	return &LoadPricesOutput{Prices: prices}, nil
}

// LoadTaxRulesInput is the input for the LoadTaxRulesActivity.
type LoadTaxRulesInput struct {
	AccountID string `json:"account_id"`
	TenantID  string `json:"tenant_id"`
}

// LoadTaxRulesOutput is the output of the LoadTaxRulesActivity.
type LoadTaxRulesOutput struct {
	TaxRules optimizer.TaxRules `json:"tax_rules"`
}

// LoadTaxRulesActivity loads tax rules for an account.
func (a *RebalanceActivities) LoadTaxRulesActivity(ctx context.Context, input LoadTaxRulesInput) (*LoadTaxRulesOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("LoadTaxRulesActivity started", "account_id", input.AccountID)

	rules := optimizer.TaxRules{
		ShortTermRate:     0.37,
		LongTermRate:      0.20,
		WashSaleDays:      30,
		HarvestThreshold:  1000.00,
		PreferLongTerm:    true,
		TransactionCostBp: 5,
		UpdatedAt:         time.Now(),
	}

	return &LoadTaxRulesOutput{TaxRules: rules}, nil
}

// TaxAwareOptimizeInput is the input for the TaxAwareOptimizeActivity.
type TaxAwareOptimizeInput struct {
	DriftReport optimizer.DriftReport `json:"drift_report"`
	Lots        []optimizer.Lot       `json:"lots"`
	Prices      map[string]float64    `json:"prices"`
	TaxRules    optimizer.TaxRules    `json:"tax_rules"`
}

// TaxAwareOptimizeOutput is the output of the TaxAwareOptimizeActivity.
type TaxAwareOptimizeOutput struct {
	Plan optimizer.Plan `json:"plan"`
}

// TaxAwareOptimizeActivity runs the tax-aware optimizer.
func (a *RebalanceActivities) TaxAwareOptimizeActivity(ctx context.Context, input TaxAwareOptimizeInput) (*TaxAwareOptimizeOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("TaxAwareOptimizeActivity started")

	inputs := optimizer.Inputs{
		Drift:  input.DriftReport,
		Lots:   input.Lots,
		Prices: input.Prices,
		Rules:  input.TaxRules,
		Weights: optimizer.ScoreWeights{
			TEWeight:        0.5,
			TaxAlphaWeight:  0.3,
			TransCostWeight: 0.2,
		},
	}
	plan := optimizer.Optimize(inputs)

	return &TaxAwareOptimizeOutput{Plan: plan}, nil
}

// FactorSelectInput is the input for the FactorSelectActivity.
type FactorSelectInput struct {
	SoldPositions []factor.SoldPosition `json:"sold_positions"`
	Universe      factor.Universe       `json:"universe"`
	Constraints   factor.Constraints    `json:"constraints"`
	Weights       factor.Weights        `json:"weights"`
	Correlations  map[string]float64    `json:"correlations"`
}

// FactorSelectOutput is the output of the FactorSelectActivity.
type FactorSelectOutput struct {
	Replacements map[string][]factor.Replacement `json:"replacements"` // SecurityID -> replacements
}

// FactorSelectActivity selects factor-aware replacements for sold positions.
func (a *RebalanceActivities) FactorSelectActivity(ctx context.Context, input FactorSelectInput) (*FactorSelectOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("FactorSelectActivity started", "positions", len(input.SoldPositions))

	replacements := make(map[string][]factor.Replacement)

	for _, sold := range input.SoldPositions {
		candidates := factor.SelectReplacements(sold, input.Universe, input.Constraints, input.Weights, input.Correlations)
		sized := factor.SizeReplacements(candidates, sold.SaleProceeds, "score_weighted")
		replacements[sold.SecurityID] = sized
	}

	return &FactorSelectOutput{Replacements: replacements}, nil
}

// MonteCarloSimInput is the input for the MonteCarloSimActivity.
type MonteCarloSimInput struct {
	Plan     optimizer.Plan     `json:"plan"`
	Lots     []optimizer.Lot    `json:"lots"`
	Prices   map[string]float64 `json:"prices"`
	TaxRules optimizer.TaxRules `json:"tax_rules"`
	NumRuns  int                `json:"num_runs"`
}

// MonteCarloSimOutput is the output of the MonteCarloSimActivity.
type MonteCarloSimOutput struct {
	Summary optimizer.MonteCarloSummary `json:"summary"`
}

// MonteCarloSimActivity runs Monte Carlo simulation on a plan.
func (a *RebalanceActivities) MonteCarloSimActivity(ctx context.Context, input MonteCarloSimInput) (*MonteCarloSimOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("MonteCarloSimActivity started", "num_runs", input.NumRuns)

	numRuns := input.NumRuns
	if numRuns <= 0 {
		numRuns = 1000
	}

	summary := optimizer.MonteCarloSimulate(input.Plan, input.Lots, input.Prices, input.TaxRules, numRuns)

	return &MonteCarloSimOutput{Summary: summary}, nil
}

// PolicyCheckInput is the input for the PolicyCheckActivity.
type PolicyCheckInput struct {
	Plan        optimizer.Plan `json:"plan"`
	PortfolioID string         `json:"portfolio_id"`
	TenantID    string         `json:"tenant_id"`
}

// PolicyViolation represents a policy violation.
type PolicyViolation struct {
	PolicyID    string `json:"policy_id"`
	PolicyName  string `json:"policy_name"`
	Severity    string `json:"severity"` // "error", "warning", "info"
	Description string `json:"description"`
	TradeID     string `json:"trade_id,omitempty"`
}

// PolicyCheckOutput is the output of the PolicyCheckActivity.
type PolicyCheckOutput struct {
	Passed     bool              `json:"passed"`
	Violations []PolicyViolation `json:"violations"`
}

// PolicyCheckActivity checks the plan against compliance policies.
func (a *RebalanceActivities) PolicyCheckActivity(ctx context.Context, input PolicyCheckInput) (*PolicyCheckOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("PolicyCheckActivity started", "portfolio_id", input.PortfolioID)

	// Mock policy check - in production would call compliance engine
	violations := []PolicyViolation{}

	// Example: Check for concentration limits
	for _, trade := range input.Plan.Trades {
		if trade.Side == "BUY" && trade.EstValue > 100000 {
			violations = append(violations, PolicyViolation{
				PolicyID:    "POL-001",
				PolicyName:  "Large Trade Review",
				Severity:    "warning",
				Description: fmt.Sprintf("Trade %s exceeds $100,000 threshold", trade.Symbol),
				TradeID:     trade.Symbol,
			})
		}
	}

	passed := true
	for _, v := range violations {
		if v.Severity == "error" {
			passed = false
			break
		}
	}

	return &PolicyCheckOutput{
		Passed:     passed,
		Violations: violations,
	}, nil
}

// NotifyAdvisorInput is the input for the NotifyAdvisorActivity.
type NotifyAdvisorInput struct {
	ProposalID  string                      `json:"proposal_id"`
	PortfolioID string                      `json:"portfolio_id"`
	AdvisorID   string                      `json:"advisor_id"`
	Plan        optimizer.Plan              `json:"plan"`
	MonteCarlo  optimizer.MonteCarloSummary `json:"monte_carlo"`
	Violations  []PolicyViolation           `json:"violations"`
}

// NotifyAdvisorOutput is the output of the NotifyAdvisorActivity.
type NotifyAdvisorOutput struct {
	NotificationID string    `json:"notification_id"`
	SentAt         time.Time `json:"sent_at"`
}

// NotifyAdvisorActivity sends a notification to the advisor about a pending proposal.
func (a *RebalanceActivities) NotifyAdvisorActivity(ctx context.Context, input NotifyAdvisorInput) (*NotifyAdvisorOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("NotifyAdvisorActivity started", "advisor_id", input.AdvisorID, "proposal_id", input.ProposalID)

	// In production, this would:
	// 1. Create a proposal record in the database
	// 2. Send push notification / email to advisor
	// 3. Update the advisor dashboard

	notificationID := uuid.New().String()

	return &NotifyAdvisorOutput{
		NotificationID: notificationID,
		SentAt:         time.Now(),
	}, nil
}

// ExecuteTradeSagaInput is the input for the ExecuteTradeSagaActivity.
type ExecuteTradeSagaInput struct {
	ProposalID  string         `json:"proposal_id"`
	PortfolioID string         `json:"portfolio_id"`
	Plan        optimizer.Plan `json:"plan"`
	ApprovedBy  string         `json:"approved_by"`
	ApprovedAt  time.Time      `json:"approved_at"`
}

// TradeExecution represents the execution result of a single trade.
type TradeExecution struct {
	TradeID       string    `json:"trade_id"`
	SecurityID    string    `json:"security_id"`
	Status        string    `json:"status"` // "executed", "failed", "pending"
	ExecutedAt    time.Time `json:"executed_at,omitempty"`
	ExecutedPrice float64   `json:"executed_price,omitempty"`
	ExecutedQty   float64   `json:"executed_qty,omitempty"`
	ErrorMessage  string    `json:"error_message,omitempty"`
}

// ExecuteTradeSagaOutput is the output of the ExecuteTradeSagaActivity.
type ExecuteTradeSagaOutput struct {
	SagaID      string           `json:"saga_id"`
	Status      string           `json:"status"` // "completed", "partial", "failed", "rolled_back"
	Executions  []TradeExecution `json:"executions"`
	CompletedAt time.Time        `json:"completed_at"`
}

// ExecuteTradeSagaActivity executes the approved trades as a saga.
func (a *RebalanceActivities) ExecuteTradeSagaActivity(ctx context.Context, input ExecuteTradeSagaInput) (*ExecuteTradeSagaOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("ExecuteTradeSagaActivity started", "proposal_id", input.ProposalID)

	sagaID := uuid.New().String()
	executions := make([]TradeExecution, 0, len(input.Plan.Trades))

	// Execute each trade
	for i, trade := range input.Plan.Trades {
		tradeID := fmt.Sprintf("%s-trade-%d", sagaID, i)

		// In production, this would call the OMS/EMS
		// For now, simulate successful execution
		executions = append(executions, TradeExecution{
			TradeID:       tradeID,
			SecurityID:    trade.Symbol,
			Status:        "executed",
			ExecutedAt:    time.Now(),
			ExecutedPrice: trade.EstValue / trade.Qty,
			ExecutedQty:   trade.Qty,
		})
	}

	return &ExecuteTradeSagaOutput{
		SagaID:      sagaID,
		Status:      "completed",
		Executions:  executions,
		CompletedAt: time.Now(),
	}, nil
}

// PersistUARInput is the input for the PersistUARActivity.
type PersistUARInput struct {
	ProposalID  string                      `json:"proposal_id"`
	PortfolioID string                      `json:"portfolio_id"`
	TenantID    string                      `json:"tenant_id"`
	Plan        optimizer.Plan              `json:"plan"`
	MonteCarlo  optimizer.MonteCarloSummary `json:"monte_carlo"`
	Violations  []PolicyViolation           `json:"violations"`
	Decision    string                      `json:"decision"` // "approved", "rejected", "clarify"
	DecisionBy  string                      `json:"decision_by"`
	DecisionAt  time.Time                   `json:"decision_at"`
	Executions  []TradeExecution            `json:"executions,omitempty"`
}

// PersistUAROutput is the output of the PersistUARActivity.
type PersistUAROutput struct {
	UARID     string    `json:"uar_id"`
	CreatedAt time.Time `json:"created_at"`
}

// PersistUARActivity persists the User Action Record for compliance.
func (a *RebalanceActivities) PersistUARActivity(ctx context.Context, input PersistUARInput) (*PersistUAROutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("PersistUARActivity started", "proposal_id", input.ProposalID)

	uarID := uuid.New().String()

	// Serialize the full record
	uarData := map[string]interface{}{
		"uar_id":       uarID,
		"proposal_id":  input.ProposalID,
		"portfolio_id": input.PortfolioID,
		"tenant_id":    input.TenantID,
		"plan":         input.Plan,
		"monte_carlo":  input.MonteCarlo,
		"violations":   input.Violations,
		"decision":     input.Decision,
		"decision_by":  input.DecisionBy,
		"decision_at":  input.DecisionAt,
		"executions":   input.Executions,
		"created_at":   time.Now(),
	}

	// In production, this would be persisted to the database
	uarJSON, _ := json.Marshal(uarData)
	logger.Info("UAR persisted", "uar_id", uarID, "size_bytes", len(uarJSON))

	return &PersistUAROutput{
		UARID:     uarID,
		CreatedAt: time.Now(),
	}, nil
}
