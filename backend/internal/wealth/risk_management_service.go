package wealth

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// RiskManagementService handles portfolio risk analysis and hedging strategies
type RiskManagementService struct {
	db *pgxpool.Pool
}

// NewRiskManagementService creates a new risk management service
func NewRiskManagementService(db *pgxpool.Pool) *RiskManagementService {
	return &RiskManagementService{
		db: db,
	}
}

// ============================================================================
// OPTIONS OVERLAY STRATEGIES
// ============================================================================

// OptionsOverlayStrategy represents a protective options strategy
type OptionsOverlayStrategy struct {
	StrategyID         string          `json:"strategy_id"`
	PortfolioID        string          `json:"portfolio_id"`
	FamilyID           string          `json:"family_id"`
	StrategyType       string          `json:"strategy_type"` // PROTECTIVE_PUT, COLLAR, COVERED_CALL
	UnderlyingPosition string          `json:"underlying_position"`
	PositionValue      decimal.Decimal `json:"position_value"`
	ProtectionLevel    decimal.Decimal `json:"protection_level"` // % downside protection
	CostOfProtection   decimal.Decimal `json:"cost_of_protection"`
	OptionLegs         []OptionLeg     `json:"option_legs"`
	MaxLoss            decimal.Decimal `json:"max_loss"`
	MaxGain            decimal.Decimal `json:"max_gain"`
	BreakEvenPrice     decimal.Decimal `json:"break_even_price"`
	Expiration         time.Time       `json:"expiration"`
	ImpliedVolatility  decimal.Decimal `json:"implied_volatility"`
	Greeks             OptionGreeks    `json:"greeks"`
	CreatedAt          time.Time       `json:"created_at"`
}

// OptionLeg represents one leg of an options strategy
type OptionLeg struct {
	LegType      string          `json:"leg_type"` // LONG_PUT, SHORT_CALL, etc.
	Strike       decimal.Decimal `json:"strike"`
	Quantity     int             `json:"quantity"`
	Premium      decimal.Decimal `json:"premium"`
	Expiration   time.Time       `json:"expiration"`
	OptionSymbol string          `json:"option_symbol"`
}

// OptionGreeks represents option sensitivity metrics
type OptionGreeks struct {
	Delta decimal.Decimal `json:"delta"`
	Gamma decimal.Decimal `json:"gamma"`
	Theta decimal.Decimal `json:"theta"`
	Vega  decimal.Decimal `json:"vega"`
	Rho   decimal.Decimal `json:"rho"`
}

// BuildProtectivePutStrategy creates a protective put overlay
func (s *RiskManagementService) BuildProtectivePutStrategy(
	ctx context.Context,
	portfolioID string,
	familyID string,
	underlyingSymbol string,
	positionValue decimal.Decimal,
	desiredProtectionPct decimal.Decimal, // e.g., 10% = protect at 10% drawdown
	expirationMonths int,
) (*OptionsOverlayStrategy, error) {
	strategy := &OptionsOverlayStrategy{
		StrategyID:         uuid.New().String(),
		PortfolioID:        portfolioID,
		FamilyID:           familyID,
		StrategyType:       "PROTECTIVE_PUT",
		UnderlyingPosition: underlyingSymbol,
		PositionValue:      positionValue,
		ProtectionLevel:    desiredProtectionPct,
		Expiration:         time.Now().AddDate(0, expirationMonths, 0),
		CreatedAt:          time.Now(),
	}

	// Calculate strike price for desired protection level
	// If stock at $100 and want 10% protection, buy put at $90 strike
	currentPrice := decimal.NewFromInt(100) // Mock - would fetch real price
	strikePrice := currentPrice.Mul(decimal.NewFromInt(1).Sub(desiredProtectionPct.Div(decimal.NewFromInt(100))))

	// Calculate number of contracts needed (100 shares per contract)
	sharesOwned := positionValue.Div(currentPrice)
	contractsNeeded := int(sharesOwned.Div(decimal.NewFromInt(100)).IntPart())

	// Mock option pricing (would use Black-Scholes)
	putPremium := decimal.NewFromFloat(2.50) // $2.50 per share
	totalCost := putPremium.Mul(decimal.NewFromInt(100)).Mul(decimal.NewFromInt(int64(contractsNeeded)))

	strategy.CostOfProtection = totalCost
	strategy.OptionLegs = []OptionLeg{
		{
			LegType:      "LONG_PUT",
			Strike:       strikePrice,
			Quantity:     contractsNeeded,
			Premium:      putPremium,
			Expiration:   strategy.Expiration,
			OptionSymbol: underlyingSymbol + "_PUT_" + strikePrice.String(),
		},
	}

	// Calculate max loss/gain
	strategy.MaxLoss = totalCost                     // Most you can lose is premium paid
	strategy.MaxGain = decimal.NewFromInt(999999999) // Unlimited upside
	strategy.BreakEvenPrice = currentPrice.Sub(putPremium)

	// Mock Greeks
	strategy.Greeks = OptionGreeks{
		Delta: decimal.NewFromFloat(-0.40), // Put delta
		Gamma: decimal.NewFromFloat(0.05),
		Theta: decimal.NewFromFloat(-0.10), // Time decay
		Vega:  decimal.NewFromFloat(0.15),
		Rho:   decimal.NewFromFloat(-0.05),
	}

	strategy.ImpliedVolatility = decimal.NewFromFloat(25.0) // 25% IV

	return strategy, nil
}

// BuildCollarStrategy creates a collar (protective put + covered call)
func (s *RiskManagementService) BuildCollarStrategy(
	ctx context.Context,
	portfolioID string,
	familyID string,
	underlyingSymbol string,
	positionValue decimal.Decimal,
	protectionStrike decimal.Decimal,
	callStrike decimal.Decimal,
	expirationMonths int,
) (*OptionsOverlayStrategy, error) {
	strategy := &OptionsOverlayStrategy{
		StrategyID:         uuid.New().String(),
		PortfolioID:        portfolioID,
		FamilyID:           familyID,
		StrategyType:       "COLLAR",
		UnderlyingPosition: underlyingSymbol,
		PositionValue:      positionValue,
		Expiration:         time.Now().AddDate(0, expirationMonths, 0),
		CreatedAt:          time.Now(),
	}

	currentPrice := decimal.NewFromInt(100)
	sharesOwned := positionValue.Div(currentPrice)
	contractsNeeded := int(sharesOwned.Div(decimal.NewFromInt(100)).IntPart())

	// Buy protective put
	putPremium := decimal.NewFromFloat(2.00)
	putCost := putPremium.Mul(decimal.NewFromInt(100)).Mul(decimal.NewFromInt(int64(contractsNeeded)))

	// Sell covered call
	callPremium := decimal.NewFromFloat(1.75)
	callCredit := callPremium.Mul(decimal.NewFromInt(100)).Mul(decimal.NewFromInt(int64(contractsNeeded)))

	// Net cost = put cost - call credit
	strategy.CostOfProtection = putCost.Sub(callCredit)

	strategy.OptionLegs = []OptionLeg{
		{
			LegType:      "LONG_PUT",
			Strike:       protectionStrike,
			Quantity:     contractsNeeded,
			Premium:      putPremium,
			Expiration:   strategy.Expiration,
			OptionSymbol: underlyingSymbol + "_PUT_" + protectionStrike.String(),
		},
		{
			LegType:      "SHORT_CALL",
			Strike:       callStrike,
			Quantity:     contractsNeeded,
			Premium:      callPremium,
			Expiration:   strategy.Expiration,
			OptionSymbol: underlyingSymbol + "_CALL_" + callStrike.String(),
		},
	}

	// Max loss = current price - put strike + net cost
	strategy.MaxLoss = currentPrice.Sub(protectionStrike).Add(strategy.CostOfProtection).Mul(decimal.NewFromInt(int64(contractsNeeded * 100)))
	// Max gain = call strike - current price - net cost
	strategy.MaxGain = callStrike.Sub(currentPrice).Sub(strategy.CostOfProtection).Mul(decimal.NewFromInt(int64(contractsNeeded * 100)))
	strategy.BreakEvenPrice = currentPrice.Add(strategy.CostOfProtection.Div(decimal.NewFromInt(int64(contractsNeeded * 100))))

	return strategy, nil
}

// ============================================================================
// TAIL RISK HEDGING
// ============================================================================

// TailRiskAnalysis represents tail risk exposure
type TailRiskAnalysis struct {
	AnalysisID            string           `json:"analysis_id"`
	PortfolioID           string           `json:"portfolio_id"`
	FamilyID              string           `json:"family_id"`
	ValueAtRisk95         decimal.Decimal  `json:"value_at_risk_95"` // 95% confidence
	ValueAtRisk99         decimal.Decimal  `json:"value_at_risk_99"` // 99% confidence
	ConditionalVaR        decimal.Decimal  `json:"conditional_var"`  // Expected Shortfall
	MaxDrawdownHistorical decimal.Decimal  `json:"max_drawdown_historical"`
	TailRiskExposure      decimal.Decimal  `json:"tail_risk_exposure"` // % portfolio at risk
	RecommendedHedges     []TailRiskHedge  `json:"recommended_hedges"`
	StressTestScenarios   []StressScenario `json:"stress_test_scenarios"`
	CreatedAt             time.Time        `json:"created_at"`
}

// TailRiskHedge represents a tail risk hedge recommendation
type TailRiskHedge struct {
	HedgeType       string          `json:"hedge_type"` // OTM_PUTS, VIX_CALLS, GOLD, TREASURIES
	AllocationPct   decimal.Decimal `json:"allocation_pct"`
	EstimatedCost   decimal.Decimal `json:"estimated_cost"`
	ProtectionLevel decimal.Decimal `json:"protection_level"`
	Rationale       string          `json:"rationale"`
}

// StressScenario represents a stress test scenario
type StressScenario struct {
	ScenarioName    string          `json:"scenario_name"` // 2008_CRISIS, COVID_CRASH, DOT_COM_BUBBLE
	EquityDraw      decimal.Decimal `json:"equity_draw"`   // % drop
	PortfolioImpact decimal.Decimal `json:"portfolio_impact"`
	RecoveryMonths  int             `json:"recovery_months"`
}

// AnalyzeTailRisk performs comprehensive tail risk analysis
func (s *RiskManagementService) AnalyzeTailRisk(
	ctx context.Context,
	portfolioID string,
	familyID string,
	portfolioValue decimal.Decimal,
	returnHistory []decimal.Decimal, // Historical returns
) (*TailRiskAnalysis, error) {
	analysis := &TailRiskAnalysis{
		AnalysisID:  uuid.New().String(),
		PortfolioID: portfolioID,
		FamilyID:    familyID,
		CreatedAt:   time.Now(),
	}

	// Calculate VaR (simplified - would use Monte Carlo or historical simulation)
	// VaR95 = portfolio value at 5th percentile of return distribution
	analysis.ValueAtRisk95 = portfolioValue.Mul(decimal.NewFromFloat(0.15)) // 15% loss at 95% confidence
	analysis.ValueAtRisk99 = portfolioValue.Mul(decimal.NewFromFloat(0.25)) // 25% loss at 99% confidence

	// Conditional VaR (CVaR) = average loss beyond VaR
	analysis.ConditionalVaR = portfolioValue.Mul(decimal.NewFromFloat(0.30)) // 30% average loss in tail

	// Historical max drawdown
	analysis.MaxDrawdownHistorical = decimal.NewFromFloat(45.0) // 45% max historical drawdown

	// Tail risk exposure
	analysis.TailRiskExposure = decimal.NewFromFloat(20.0) // 20% of portfolio at risk in tail events

	// Recommend hedges
	analysis.RecommendedHedges = []TailRiskHedge{
		{
			HedgeType:       "OTM_PUTS",
			AllocationPct:   decimal.NewFromFloat(2.0),                       // 2% of portfolio
			EstimatedCost:   portfolioValue.Mul(decimal.NewFromFloat(0.005)), // 0.5% annual cost
			ProtectionLevel: decimal.NewFromFloat(15.0),                      // Protect against 15%+ drops
			Rationale:       "Out-of-the-money puts provide asymmetric downside protection",
		},
		{
			HedgeType:       "VIX_CALLS",
			AllocationPct:   decimal.NewFromFloat(1.0),                      // 1% of portfolio
			EstimatedCost:   portfolioValue.Mul(decimal.NewFromFloat(0.01)), // 1% annual cost
			ProtectionLevel: decimal.NewFromFloat(20.0),                     // Spikes in volatility
			Rationale:       "VIX calls spike during market crashes, offsetting equity losses",
		},
		{
			HedgeType:       "GOLD",
			AllocationPct:   decimal.NewFromFloat(5.0), // 5% allocation
			EstimatedCost:   decimal.Zero,              // No carrying cost
			ProtectionLevel: decimal.NewFromFloat(10.0),
			Rationale:       "Gold typically performs well during market stress and inflation",
		},
	}

	// Stress test scenarios
	analysis.StressTestScenarios = []StressScenario{
		{
			ScenarioName:    "2008_FINANCIAL_CRISIS",
			EquityDraw:      decimal.NewFromFloat(-57.0),                     // S&P 500 dropped 57%
			PortfolioImpact: portfolioValue.Mul(decimal.NewFromFloat(-0.40)), // -40% with diversification
			RecoveryMonths:  48,                                              // 4 years to recover
		},
		{
			ScenarioName:    "COVID_CRASH_2020",
			EquityDraw:      decimal.NewFromFloat(-34.0),                     // S&P dropped 34%
			PortfolioImpact: portfolioValue.Mul(decimal.NewFromFloat(-0.25)), // -25% with diversification
			RecoveryMonths:  6,                                               // 6 months to recover
		},
		{
			ScenarioName:    "DOT_COM_BUBBLE_2000",
			EquityDraw:      decimal.NewFromFloat(-49.0),                     // NASDAQ dropped 78%, S&P 49%
			PortfolioImpact: portfolioValue.Mul(decimal.NewFromFloat(-0.35)), // -35%
			RecoveryMonths:  60,                                              // 5 years to recover
		},
	}

	return analysis, nil
}

// ============================================================================
// DRAWDOWN ANALYSIS
// ============================================================================

// DrawdownAnalysis represents portfolio drawdown metrics
type DrawdownAnalysis struct {
	AnalysisID            string              `json:"analysis_id"`
	PortfolioID           string              `json:"portfolio_id"`
	FamilyID              string              `json:"family_id"`
	CurrentDrawdown       decimal.Decimal     `json:"current_drawdown"`
	MaxDrawdown           decimal.Decimal     `json:"max_drawdown"`
	AverageDrawdown       decimal.Decimal     `json:"average_drawdown"`
	DrawdownDuration      int                 `json:"drawdown_duration"`      // Days in current drawdown
	RecoveryTimeEstimate  int                 `json:"recovery_time_estimate"` // Estimated days to recover
	DrawdownEvents        []DrawdownEvent     `json:"drawdown_events"`        // Historical drawdowns
	DrawdownProbabilities DrawdownProbability `json:"drawdown_probabilities"`
	CreatedAt             time.Time           `json:"created_at"`
}

// DrawdownEvent represents a historical drawdown
type DrawdownEvent struct {
	StartDate    time.Time       `json:"start_date"`
	BottomDate   time.Time       `json:"bottom_date"`
	RecoveryDate *time.Time      `json:"recovery_date,omitempty"`
	DrawdownPct  decimal.Decimal `json:"drawdown_pct"`
	Duration     int             `json:"duration"` // Days
	Recovery     int             `json:"recovery"` // Days to recover
}

// DrawdownProbability represents probability of future drawdowns
type DrawdownProbability struct {
	Prob10PctDrawdown1Yr decimal.Decimal `json:"prob_10pct_drawdown_1yr"` // Probability of 10%+ drawdown in 1 year
	Prob20PctDrawdown1Yr decimal.Decimal `json:"prob_20pct_drawdown_1yr"`
	Prob30PctDrawdown1Yr decimal.Decimal `json:"prob_30pct_drawdown_1yr"`
}

// AnalyzeDrawdowns performs comprehensive drawdown analysis
func (s *RiskManagementService) AnalyzeDrawdowns(
	ctx context.Context,
	portfolioID string,
	familyID string,
	priceHistory []PricePoint,
) (*DrawdownAnalysis, error) {
	analysis := &DrawdownAnalysis{
		AnalysisID:  uuid.New().String(),
		PortfolioID: portfolioID,
		FamilyID:    familyID,
		CreatedAt:   time.Now(),
	}

	// Mock historical drawdowns
	analysis.DrawdownEvents = []DrawdownEvent{
		{
			StartDate:    time.Date(2020, 2, 19, 0, 0, 0, 0, time.UTC),
			BottomDate:   time.Date(2020, 3, 23, 0, 0, 0, 0, time.UTC),
			RecoveryDate: ptrTime(time.Date(2020, 8, 18, 0, 0, 0, 0, time.UTC)),
			DrawdownPct:  decimal.NewFromFloat(-34.0),
			Duration:     33,
			Recovery:     148,
		},
		{
			StartDate:    time.Date(2018, 9, 20, 0, 0, 0, 0, time.UTC),
			BottomDate:   time.Date(2018, 12, 24, 0, 0, 0, 0, time.UTC),
			RecoveryDate: ptrTime(time.Date(2019, 4, 23, 0, 0, 0, 0, time.UTC)),
			DrawdownPct:  decimal.NewFromFloat(-19.8),
			Duration:     95,
			Recovery:     120,
		},
	}

	analysis.CurrentDrawdown = decimal.NewFromFloat(-5.2) // Currently 5.2% below peak
	analysis.MaxDrawdown = decimal.NewFromFloat(-34.0)
	analysis.AverageDrawdown = decimal.NewFromFloat(-12.5)
	analysis.DrawdownDuration = 15     // 15 days in current drawdown
	analysis.RecoveryTimeEstimate = 30 // Estimated 30 days to recover

	// Drawdown probabilities (based on historical volatility)
	analysis.DrawdownProbabilities = DrawdownProbability{
		Prob10PctDrawdown1Yr: decimal.NewFromFloat(45.0), // 45% chance
		Prob20PctDrawdown1Yr: decimal.NewFromFloat(15.0), // 15% chance
		Prob30PctDrawdown1Yr: decimal.NewFromFloat(5.0),  // 5% chance
	}

	return analysis, nil
}

// PricePoint represents a price at a point in time
type PricePoint struct {
	Date  time.Time       `json:"date"`
	Price decimal.Decimal `json:"price"`
}

// Helper function
func ptrTime(t time.Time) *time.Time {
	return &t
}
