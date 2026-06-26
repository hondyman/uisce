package wealth

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// AlternativeInvestmentService handles alternative investment tracking
type AlternativeInvestmentService struct {
	db *pgxpool.Pool
}

// NewAlternativeInvestmentService creates a new alternative investment service
func NewAlternativeInvestmentService(db *pgxpool.Pool) *AlternativeInvestmentService {
	return &AlternativeInvestmentService{
		db: db,
	}
}

// ============================================================================
// PRIVATE EQUITY TRACKING
// ============================================================================

// PrivateEquityInvestment represents a PE fund investment
type PrivateEquityInvestment struct {
	InvestmentID       string                 `json:"investment_id"`
	FamilyID           string                 `json:"family_id"`
	FundName           string                 `json:"fund_name"`
	GeneralPartner     string                 `json:"general_partner"`
	VintageYear        int                    `json:"vintage_year"`
	CommitmentAmount   decimal.Decimal        `json:"commitment_amount"`
	CapitalCalled      decimal.Decimal        `json:"capital_called"`
	CapitalRemaining   decimal.Decimal        `json:"capital_remaining"`
	Distributions      decimal.Decimal        `json:"distributions"`
	CurrentNAV         decimal.Decimal        `json:"current_nav"`
	TotalValue         decimal.Decimal        `json:"total_value"`   // NAV + Distributions
	IRR                decimal.Decimal        `json:"irr"`           // Internal Rate of Return
	MOIC               decimal.Decimal        `json:"moic"`          // Multiple on Invested Capital
	DPI                decimal.Decimal        `json:"dpi"`           // Distributions to Paid-In
	RVPI               decimal.Decimal        `json:"rvpi"`          // Residual Value to Paid-In
	TVPI               decimal.Decimal        `json:"tvpi"`          // Total Value to Paid-In
	JCurvePhase        string                 `json:"j_curve_phase"` // DRAWDOWN, TROUGH, RECOVERY, MATURE
	ExpectedFinalClose time.Time              `json:"expected_final_close"`
	CashFlowProjection []PECashFlowProjection `json:"cash_flow_projection"`
	CreatedAt          time.Time              `json:"created_at"`
}

// PECashFlowProjection represents projected cash flows for PE investment
type PECashFlowProjection struct {
	Year                   int             `json:"year"`
	ProjectedCalls         decimal.Decimal `json:"projected_calls"`
	ProjectedDistributions decimal.Decimal `json:"projected_distributions"`
	NetCashFlow            decimal.Decimal `json:"net_cash_flow"`
	CumulativeNAV          decimal.Decimal `json:"cumulative_nav"`
}

// CalculatePEMetrics calculates private equity performance metrics
func (s *AlternativeInvestmentService) CalculatePEMetrics(
	ctx context.Context,
	investmentID string,
	commitmentAmount decimal.Decimal,
	capitalCalled decimal.Decimal,
	distributions decimal.Decimal,
	currentNAV decimal.Decimal,
) (*PrivateEquityInvestment, error) {
	investment := &PrivateEquityInvestment{
		InvestmentID:     investmentID,
		CommitmentAmount: commitmentAmount,
		CapitalCalled:    capitalCalled,
		CapitalRemaining: commitmentAmount.Sub(capitalCalled),
		Distributions:    distributions,
		CurrentNAV:       currentNAV,
		TotalValue:       currentNAV.Add(distributions),
	}

	// Calculate key PE metrics
	if capitalCalled.GreaterThan(decimal.Zero) {
		// DPI = Distributions / Capital Called
		investment.DPI = distributions.Div(capitalCalled)

		// RVPI = NAV / Capital Called
		investment.RVPI = currentNAV.Div(capitalCalled)

		// TVPI = (NAV + Distributions) / Capital Called
		investment.TVPI = investment.TotalValue.Div(capitalCalled)

		// MOIC = Total Value / Capital Called (same as TVPI)
		investment.MOIC = investment.TVPI
	}

	// Determine J-Curve phase
	investment.JCurvePhase = s.determineJCurvePhase(investment)

	return investment, nil
}

// determineJCurvePhase determines where the investment is in the J-curve
func (s *AlternativeInvestmentService) determineJCurvePhase(pe *PrivateEquityInvestment) string {
	calledPct := decimal.Zero
	if pe.CommitmentAmount.GreaterThan(decimal.Zero) {
		calledPct = pe.CapitalCalled.Div(pe.CommitmentAmount)
	}

	tvpi := pe.TVPI

	switch {
	case calledPct.LessThan(decimal.NewFromFloat(0.30)):
		return "DRAWDOWN" // Still calling capital
	case tvpi.LessThan(decimal.NewFromFloat(0.80)):
		return "TROUGH" // In the J-curve trough (negative returns)
	case tvpi.LessThan(decimal.NewFromFloat(1.20)):
		return "RECOVERY" // Recovering to 1.0x
	default:
		return "MATURE" // Generating returns
	}
}

// ============================================================================
// VENTURE CAPITAL TRACKING
// ============================================================================

// VentureCapitalInvestment represents a VC investment
type VentureCapitalInvestment struct {
	InvestmentID       string           `json:"investment_id"`
	FamilyID           string           `json:"family_id"`
	CompanyName        string           `json:"company_name"`
	Round              string           `json:"round"` // SEED, SERIES_A, SERIES_B, etc.
	InvestmentDate     time.Time        `json:"investment_date"`
	InitialInvestment  decimal.Decimal  `json:"initial_investment"`
	SharesOwned        int64            `json:"shares_owned"`
	SharePriceInitial  decimal.Decimal  `json:"share_price_initial"`
	SharePriceCurrent  decimal.Decimal  `json:"share_price_current"`
	CurrentValuation   decimal.Decimal  `json:"current_valuation"`
	OwnershipPct       decimal.Decimal  `json:"ownership_pct"`
	PreMoneyValuation  decimal.Decimal  `json:"pre_money_valuation"`
	PostMoneyValuation decimal.Decimal  `json:"post_money_valuation"`
	FullyDilutedShares int64            `json:"fully_diluted_shares"`
	DilutionRisk       decimal.Decimal  `json:"dilution_risk"`
	ExitScenarios      []VCExitScenario `json:"exit_scenarios"`
	CreatedAt          time.Time        `json:"created_at"`
}

// VCExitScenario represents a potential exit scenario
type VCExitScenario struct {
	Scenario         string          `json:"scenario"` // ACQUISITION, IPO, DOWN_ROUND, WRITE_OFF
	Probability      decimal.Decimal `json:"probability"`
	ExitValuation    decimal.Decimal `json:"exit_valuation"`
	ExpectedReturn   decimal.Decimal `json:"expected_return"`
	ExpectedMultiple decimal.Decimal `json:"expected_multiple"`
}

// ModelVCExitScenarios creates exit scenarios for VC investment
func (s *AlternativeInvestmentService) ModelVCExitScenarios(
	ctx context.Context,
	initialInvestment decimal.Decimal,
	currentOwnershipPct decimal.Decimal,
) []VCExitScenario {
	scenarios := []VCExitScenario{
		{
			Scenario:         "WRITE_OFF",
			Probability:      decimal.NewFromFloat(0.40), // 40% fail
			ExitValuation:    decimal.Zero,
			ExpectedReturn:   initialInvestment.Mul(decimal.NewFromInt(-1)),
			ExpectedMultiple: decimal.Zero,
		},
		{
			Scenario:         "ACQUISITION_MODEST",
			Probability:      decimal.NewFromFloat(0.30), // 30% modest exit
			ExitValuation:    initialInvestment.Div(currentOwnershipPct).Mul(decimal.NewFromInt(3)),
			ExpectedReturn:   initialInvestment.Mul(decimal.NewFromInt(2)),
			ExpectedMultiple: decimal.NewFromInt(3),
		},
		{
			Scenario:         "ACQUISITION_STRONG",
			Probability:      decimal.NewFromFloat(0.20), // 20% strong exit
			ExitValuation:    initialInvestment.Div(currentOwnershipPct).Mul(decimal.NewFromInt(10)),
			ExpectedReturn:   initialInvestment.Mul(decimal.NewFromInt(9)),
			ExpectedMultiple: decimal.NewFromInt(10),
		},
		{
			Scenario:         "IPO",
			Probability:      decimal.NewFromFloat(0.10), // 10% IPO
			ExitValuation:    initialInvestment.Div(currentOwnershipPct).Mul(decimal.NewFromInt(25)),
			ExpectedReturn:   initialInvestment.Mul(decimal.NewFromInt(24)),
			ExpectedMultiple: decimal.NewFromInt(25),
		},
	}

	return scenarios
}

// ============================================================================
// ART & COLLECTIBLES TRACKING
// ============================================================================

// ArtCollectible represents art and collectible assets
type ArtCollectible struct {
	AssetID             string          `json:"asset_id"`
	FamilyID            string          `json:"family_id"`
	ArtistName          string          `json:"artist_name"`
	ArtworkTitle        string          `json:"artwork_title"`
	Medium              string          `json:"medium"`
	YearCreated         int             `json:"year_created"`
	AcquisitionDate     time.Time       `json:"acquisition_date"`
	AcquisitionPrice    decimal.Decimal `json:"acquisition_price"`
	CurrentValuation    decimal.Decimal `json:"current_valuation"`
	ValuationDate       time.Time       `json:"valuation_date"`
	AppraisalFirm       string          `json:"appraisal_firm"`
	InsuranceValue      decimal.Decimal `json:"insurance_value"`
	InsuranceProvider   string          `json:"insurance_provider"`
	Location            string          `json:"location"`
	Provenance          string          `json:"provenance"`
	Condition           string          `json:"condition"` // EXCELLENT, GOOD, FAIR, POOR
	AnnualAppreciation  decimal.Decimal `json:"annual_appreciation"`
	FractionalOwnership bool            `json:"fractional_ownership"`
	OwnershipPct        decimal.Decimal `json:"ownership_pct"`
	CreatedAt           time.Time       `json:"created_at"`
}

// TrackArtAppreciation calculates art appreciation over time
func (s *AlternativeInvestmentService) TrackArtAppreciation(
	ctx context.Context,
	acquisitionPrice decimal.Decimal,
	currentValuation decimal.Decimal,
	yearsHeld int,
) decimal.Decimal {
	if yearsHeld == 0 || acquisitionPrice.Equal(decimal.Zero) {
		return decimal.Zero
	}

	// CAGR = (Ending Value / Beginning Value)^(1/years) - 1
	ratio := currentValuation.Div(acquisitionPrice)
	cagr := ratio.Pow(decimal.NewFromInt(1).Div(decimal.NewFromInt(int64(yearsHeld)))).Sub(decimal.NewFromInt(1))

	return cagr.Mul(decimal.NewFromInt(100)) // Convert to percentage
}

// ============================================================================
// HEDGE FUND ANALYTICS
// ============================================================================

// HedgeFundInvestment represents hedge fund investment
type HedgeFundInvestment struct {
	InvestmentID       string          `json:"investment_id"`
	FamilyID           string          `json:"family_id"`
	FundName           string          `json:"fund_name"`
	Strategy           string          `json:"strategy"` // LONG_SHORT, GLOBAL_MACRO, EVENT_DRIVEN, etc.
	InitialInvestment  decimal.Decimal `json:"initial_investment"`
	CurrentNAV         decimal.Decimal `json:"current_nav"`
	MonthlyReturns     []MonthlyReturn `json:"monthly_returns"`
	AnnualizedReturn   decimal.Decimal `json:"annualized_return"`
	Volatility         decimal.Decimal `json:"volatility"`
	SharpeRatio        decimal.Decimal `json:"sharpe_ratio"`
	MaxDrawdown        decimal.Decimal `json:"max_drawdown"`
	CorrelationToSP500 decimal.Decimal `json:"correlation_to_sp500"`
	StyleDriftAlert    bool            `json:"style_drift_alert"`
	ManagementFee      decimal.Decimal `json:"management_fee"`
	PerformanceFee     decimal.Decimal `json:"performance_fee"`
	HighWaterMark      decimal.Decimal `json:"high_water_mark"`
	CreatedAt          time.Time       `json:"created_at"`
}

// MonthlyReturn represents monthly performance
type MonthlyReturn struct {
	Month  time.Time       `json:"month"`
	Return decimal.Decimal `json:"return"`
}

// DetectStyleDrift analyzes if hedge fund is drifting from stated strategy
func (s *AlternativeInvestmentService) DetectStyleDrift(
	ctx context.Context,
	fundStrategy string,
	monthlyReturns []MonthlyReturn,
	correlationToSP500 decimal.Decimal,
) bool {
	// Simple heuristic: if a long/short equity fund has >0.70 correlation to S&P500,
	// it may be drifting to just being long-only
	if fundStrategy == "LONG_SHORT" && correlationToSP500.GreaterThan(decimal.NewFromFloat(0.70)) {
		return true
	}

	// Global macro should have low correlation (<0.30)
	if fundStrategy == "GLOBAL_MACRO" && correlationToSP500.GreaterThan(decimal.NewFromFloat(0.30)) {
		return true
	}

	return false
}

// ============================================================================
// REAL ESTATE SYNDICATION
// ============================================================================

// RealEstateSyndication represents real estate syndication investment
type RealEstateSyndication struct {
	InvestmentID         string          `json:"investment_id"`
	FamilyID             string          `json:"family_id"`
	PropertyName         string          `json:"property_name"`
	PropertyType         string          `json:"property_type"` // MULTIFAMILY, OFFICE, RETAIL, INDUSTRIAL
	Location             string          `json:"location"`
	Sponsor              string          `json:"sponsor"`
	InitialInvestment    decimal.Decimal `json:"initial_investment"`
	OwnershipPct         decimal.Decimal `json:"ownership_pct"`
	CurrentValue         decimal.Decimal `json:"current_value"`
	AnnualCashFlow       decimal.Decimal `json:"annual_cash_flow"`
	CashOnCashReturn     decimal.Decimal `json:"cash_on_cash_return"`
	TotialDepreciation   decimal.Decimal `json:"total_depreciation"`
	BonusDepreciation    decimal.Decimal `json:"bonus_depreciation"`
	TaxLossPassThrough   decimal.Decimal `json:"tax_loss_pass_through"`
	K1Received           bool            `json:"k1_received"`
	K1DocumentID         *string         `json:"k1_document_id,omitempty"`
	Exchange1031Eligible bool            `json:"exchange_1031_eligible"`
	ExpectedExitYear     int             `json:"expected_exit_year"`
	ProjectedIRR         decimal.Decimal `json:"projected_irr"`
	CreatedAt            time.Time       `json:"created_at"`
}

// Calculate1031ExchangeOpportunity identifies 1031 exchange opportunities
func (s *AlternativeInvestmentService) Calculate1031ExchangeOpportunity(
	ctx context.Context,
	propertyValue decimal.Decimal,
	costBasis decimal.Decimal,
	expectedSalePrice decimal.Decimal,
) *Exchange1031Analysis {
	capitalGain := expectedSalePrice.Sub(costBasis)
	capitalGainsTax := capitalGain.Mul(decimal.NewFromFloat(0.20)) // 20% federal
	niitTax := capitalGain.Mul(decimal.NewFromFloat(0.038))        // 3.8% NIIT
	totalTaxOwe := capitalGainsTax.Add(niitTax)

	return &Exchange1031Analysis{
		PropertyValue:     propertyValue,
		CostBasis:         costBasis,
		ExpectedSalePrice: expectedSalePrice,
		CapitalGain:       capitalGain,
		TaxIfSold:         totalTaxOwe,
		TaxDeferred1031:   totalTaxOwe,
		NetProceedsIfSold: expectedSalePrice.Sub(totalTaxOwe),
		NetProceeds1031:   expectedSalePrice, // Full amount available for reinvestment
		BenefitOf1031:     totalTaxOwe,
	}
}

// Exchange1031Analysis represents 1031 exchange analysis
type Exchange1031Analysis struct {
	PropertyValue     decimal.Decimal `json:"property_value"`
	CostBasis         decimal.Decimal `json:"cost_basis"`
	ExpectedSalePrice decimal.Decimal `json:"expected_sale_price"`
	CapitalGain       decimal.Decimal `json:"capital_gain"`
	TaxIfSold         decimal.Decimal `json:"tax_if_sold"`
	TaxDeferred1031   decimal.Decimal `json:"tax_deferred_1031"`
	NetProceedsIfSold decimal.Decimal `json:"net_proceeds_if_sold"`
	NetProceeds1031   decimal.Decimal `json:"net_proceeds_1031"`
	BenefitOf1031     decimal.Decimal `json:"benefit_of_1031"`
}
