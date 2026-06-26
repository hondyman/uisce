package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/hondyman/semlayer/backend/internal/wealth"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"
)

// WealthVisionHandlers contains handlers for WealthVision features
type WealthVisionHandlers struct {
	taxOptService   *wealth.TaxOptimizationService
	multiGenService *wealth.MultiGenerationalService
	altInvService   *wealth.AlternativeInvestmentService
	aiService       *wealth.AIIntelligenceService
	esgService      *wealth.ESGIntelligenceService
}

// NewWealthVisionHandlers creates WealthVision handlers
func NewWealthVisionHandlers(
	taxOptService *wealth.TaxOptimizationService,
	multiGenService *wealth.MultiGenerationalService,
	altInvService *wealth.AlternativeInvestmentService,
	aiService *wealth.AIIntelligenceService,
	esgService *wealth.ESGIntelligenceService,
) *WealthVisionHandlers {
	return &WealthVisionHandlers{
		taxOptService:   taxOptService,
		multiGenService: multiGenService,
		altInvService:   altInvService,
		aiService:       aiService,
		esgService:      esgService,
	}
}

// RegisterRoutes registers all WealthVision routes
func (h *WealthVisionHandlers) RegisterRoutes(r chi.Router) {
	r.Route("/wealthvision", func(r chi.Router) {
		// Tax Optimization endpoints
		r.Post("/tax/state-residency-comparison", h.CompareStateResidencies)
		r.Post("/tax/niit-calculation", h.CalculateNIIT)
		r.Post("/tax/charitable-bunching", h.AnalyzeCharitableBunching)

		// Multi-Generational endpoints
		r.Post("/multigen/dynasty-trust-simulation", h.SimulateDynastyTrust)
		r.Post("/multigen/529-optimization", h.Optimize529Plan)
		r.Post("/multigen/legacy-impact", h.CalculateLegacyImpact)

		// Alternative Investment endpoints
		r.Post("/altinv/pe-metrics", h.CalculatePEMetrics)
		r.Post("/altinv/vc-exit-scenarios", h.ModelVCExitScenarios)
		r.Post("/altinv/1031-exchange", h.Calculate1031Exchange)
		r.Post("/altinv/art-appreciation", h.TrackArtAppreciation)

		// AI Intelligence endpoints
		r.Post("/ai/churn-prediction", h.PredictChurnRisk)
		r.Post("/ai/meeting-prep", h.GenerateMeetingPrep)
		r.Post("/ai/portfolio-optimization", h.GeneratePortfolioRecommendation)

		// ESG Intelligence endpoints
		r.Post("/esg/carbon-footprint", h.CalculateCarbonFootprint)
		r.Post("/esg/esg-score", h.CalculateESGScore)
		r.Post("/esg/impact-investment", h.TrackImpactInvestment)
	})
}

// ==============================================================================
// TAX OPTIMIZATION HANDLERS
// ==============================================================================

type CompareStateResidenciesRequest struct {
	FamilyID         string          `json:"family_id"`
	CurrentState     string          `json:"current_state"`
	GrossIncome      decimal.Decimal `json:"gross_income"`
	InvestmentIncome decimal.Decimal `json:"investment_income"`
	CapitalGains     decimal.Decimal `json:"capital_gains"`
	EstateValue      decimal.Decimal `json:"estate_value"`
	StatesToCompare  []string        `json:"states_to_compare"`
}

func (h *WealthVisionHandlers) CompareStateResidencies(w http.ResponseWriter, r *http.Request) {
	var req CompareStateResidenciesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := h.taxOptService.CompareStateResidencies(
		r.Context(),
		req.FamilyID,
		req.CurrentState,
		req.GrossIncome,
		req.InvestmentIncome,
		req.CapitalGains,
		req.EstateValue,
		req.StatesToCompare,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

type CalculateNIITRequest struct {
	FamilyID                   string          `json:"family_id"`
	MemberID                   string          `json:"member_id"`
	TaxYear                    int             `json:"tax_year"`
	FilingStatus               string          `json:"filing_status"`
	ModifiedAGI                decimal.Decimal `json:"modified_agi"`
	InvestmentIncomeComponents struct {
		Interest            decimal.Decimal `json:"interest"`
		Dividends           decimal.Decimal `json:"dividends"`
		CapitalGains        decimal.Decimal `json:"capital_gains"`
		RentalIncome        decimal.Decimal `json:"rental_income"`
		PassiveIncome       decimal.Decimal `json:"passive_income"`
		NetInvestmentIncome decimal.Decimal `json:"net_investment_income"`
	} `json:"investment_income_components"`
}

func (h *WealthVisionHandlers) CalculateNIIT(w http.ResponseWriter, r *http.Request) {
	var req CalculateNIITRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	components := wealth.InvestmentIncomeBreakdown{
		Interest:            req.InvestmentIncomeComponents.Interest,
		Dividends:           req.InvestmentIncomeComponents.Dividends,
		CapitalGains:        req.InvestmentIncomeComponents.CapitalGains,
		RentalIncome:        req.InvestmentIncomeComponents.RentalIncome,
		PassiveIncome:       req.InvestmentIncomeComponents.PassiveIncome,
		NetInvestmentIncome: req.InvestmentIncomeComponents.NetInvestmentIncome,
	}

	result, err := h.taxOptService.CalculateNIIT(
		r.Context(),
		req.FamilyID,
		req.MemberID,
		req.TaxYear,
		req.FilingStatus,
		req.ModifiedAGI,
		components,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

type AnalyzeCharitableBunchingRequest struct {
	FamilyID                string          `json:"family_id"`
	MemberID                string          `json:"member_id"`
	AnalysisYears           int             `json:"analysis_years"`
	AnnualGiving            decimal.Decimal `json:"annual_giving"`
	StandardDeduction       decimal.Decimal `json:"standard_deduction"`
	OtherItemizedDeductions decimal.Decimal `json:"other_itemized_deductions"`
	MarginalTaxRate         decimal.Decimal `json:"marginal_tax_rate"`
}

func (h *WealthVisionHandlers) AnalyzeCharitableBunching(w http.ResponseWriter, r *http.Request) {
	var req AnalyzeCharitableBunchingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := h.taxOptService.AnalyzeCharitableBunching(
		r.Context(),
		req.FamilyID,
		req.MemberID,
		req.AnalysisYears,
		req.AnnualGiving,
		req.StandardDeduction,
		req.OtherItemizedDeductions,
		req.MarginalTaxRate,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// ==============================================================================
// MULTI-GENERATIONAL HANDLERS
// ==============================================================================

type SimulateDynastyTrustRequest struct {
	FamilyID           string          `json:"family_id"`
	TrustName          string          `json:"trust_name"`
	InitialFunding     decimal.Decimal `json:"initial_funding"`
	GrowthRate         decimal.Decimal `json:"growth_rate"`
	GenerationCount    int             `json:"generation_count"`
	YearsPerGeneration int             `json:"years_per_generation"`
}

func (h *WealthVisionHandlers) SimulateDynastyTrust(w http.ResponseWriter, r *http.Request) {
	var req SimulateDynastyTrustRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := h.multiGenService.SimulateDynastyTrust(
		r.Context(),
		req.FamilyID,
		req.TrustName,
		req.InitialFunding,
		req.GrowthRate,
		req.GenerationCount,
		req.YearsPerGeneration,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

type Optimize529PlanRequest struct {
	FamilyID            string          `json:"family_id"`
	StudentMemberID     string          `json:"student_member_id"`
	StudentAge          int             `json:"student_age"`
	TargetFunding       decimal.Decimal `json:"target_funding"`
	CurrentSavings      decimal.Decimal `json:"current_savings"`
	MonthlyContribution decimal.Decimal `json:"monthly_contribution"`
	HomeState           string          `json:"home_state"`
}

func (h *WealthVisionHandlers) Optimize529Plan(w http.ResponseWriter, r *http.Request) {
	var req Optimize529PlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := h.multiGenService.Optimize529Plan(
		r.Context(),
		req.FamilyID,
		req.StudentMemberID,
		req.StudentAge,
		req.TargetFunding,
		req.CurrentSavings,
		req.MonthlyContribution,
		req.HomeState,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

type CalculateLegacyImpactRequest struct {
	FamilyID             string          `json:"family_id"`
	PhilanthropicFocus   string          `json:"philanthropic_focus"`
	AnnualGiving         decimal.Decimal `json:"annual_giving"`
	YearsOfGiving        int             `json:"years_of_giving"`
	IncludesDynastyTrust bool            `json:"includes_dynasty_trust"`
	DynastyGivingPct     decimal.Decimal `json:"dynasty_giving_pct"`
}

func (h *WealthVisionHandlers) CalculateLegacyImpact(w http.ResponseWriter, r *http.Request) {
	var req CalculateLegacyImpactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := h.multiGenService.CalculateLegacyImpact(
		r.Context(),
		req.FamilyID,
		req.PhilanthropicFocus,
		req.AnnualGiving,
		req.YearsOfGiving,
		req.IncludesDynastyTrust,
		req.DynastyGivingPct,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// ==============================================================================
// ALTERNATIVE INVESTMENT HANDLERS
// ==============================================================================

type CalculatePEMetricsRequest struct {
	InvestmentID     string          `json:"investment_id"`
	CommitmentAmount decimal.Decimal `json:"commitment_amount"`
	CapitalCalled    decimal.Decimal `json:"capital_called"`
	Distributions    decimal.Decimal `json:"distributions"`
	CurrentNAV       decimal.Decimal `json:"current_nav"`
}

func (h *WealthVisionHandlers) CalculatePEMetrics(w http.ResponseWriter, r *http.Request) {
	var req CalculatePEMetricsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := h.altInvService.CalculatePEMetrics(
		r.Context(),
		req.InvestmentID,
		req.CommitmentAmount,
		req.CapitalCalled,
		req.Distributions,
		req.CurrentNAV,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

type ModelVCExitScenariosRequest struct {
	InitialInvestment   decimal.Decimal `json:"initial_investment"`
	CurrentOwnershipPct decimal.Decimal `json:"current_ownership_pct"`
}

func (h *WealthVisionHandlers) ModelVCExitScenarios(w http.ResponseWriter, r *http.Request) {
	var req ModelVCExitScenariosRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result := h.altInvService.ModelVCExitScenarios(
		r.Context(),
		req.InitialInvestment,
		req.CurrentOwnershipPct,
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

type Calculate1031ExchangeRequest struct {
	PropertyValue     decimal.Decimal `json:"property_value"`
	CostBasis         decimal.Decimal `json:"cost_basis"`
	ExpectedSalePrice decimal.Decimal `json:"expected_sale_price"`
}

func (h *WealthVisionHandlers) Calculate1031Exchange(w http.ResponseWriter, r *http.Request) {
	var req Calculate1031ExchangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result := h.altInvService.Calculate1031ExchangeOpportunity(
		r.Context(),
		req.PropertyValue,
		req.CostBasis,
		req.ExpectedSalePrice,
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

type TrackArtAppreciationRequest struct {
	AcquisitionPrice decimal.Decimal `json:"acquisition_price"`
	CurrentValuation decimal.Decimal `json:"current_valuation"`
	YearsHeld        int             `json:"years_held"`
}

func (h *WealthVisionHandlers) TrackArtAppreciation(w http.ResponseWriter, r *http.Request) {
	var req TrackArtAppreciationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cagr := h.altInvService.TrackArtAppreciation(
		r.Context(),
		req.AcquisitionPrice,
		req.CurrentValuation,
		req.YearsHeld,
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"acquisition_price": req.AcquisitionPrice,
		"current_valuation": req.CurrentValuation,
		"years_held":        req.YearsHeld,
		"cagr_percent":      cagr,
	})
}

// ==============================================================================
// AI INTELLIGENCE HANDLERS
// ==============================================================================

type PredictChurnRiskRequest struct {
	FamilyID                string          `json:"family_id"`
	FamilyName              string          `json:"family_name"`
	AUM                     decimal.Decimal `json:"aum"`
	LastLoginDays           int             `json:"last_login_days"`
	LastContactDays         int             `json:"last_contact_days"`
	PortfolioPerformance    decimal.Decimal `json:"portfolio_performance"`
	ServiceIssuesCount      int             `json:"service_issues_count"`
	AgeOfRelationshipMonths int             `json:"age_of_relationship_months"`
	AdvisorChangesCount     int             `json:"advisor_changes_count"`
	NetNewAssets            decimal.Decimal `json:"net_new_assets"`
}

func (h *WealthVisionHandlers) PredictChurnRisk(w http.ResponseWriter, r *http.Request) {
	var req PredictChurnRiskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := h.aiService.PredictChurnRisk(
		r.Context(),
		req.FamilyID,
		req.FamilyName,
		req.AUM,
		req.LastLoginDays,
		req.LastContactDays,
		req.PortfolioPerformance,
		req.ServiceIssuesCount,
		req.AgeOfRelationshipMonths,
		req.AdvisorChangesCount,
		req.NetNewAssets,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

type GenerateMeetingPrepRequest struct {
	FamilyID    string `json:"family_id"`
	MeetingDate string `json:"meeting_date"`
	MeetingType string `json:"meeting_type"`
}

func (h *WealthVisionHandlers) GenerateMeetingPrep(w http.ResponseWriter, r *http.Request) {
	var req GenerateMeetingPrepRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Parse meeting date
	meetingDate, err := parseDate(req.MeetingDate)
	if err != nil {
		http.Error(w, "invalid meeting_date format", http.StatusBadRequest)
		return
	}

	result, err := h.aiService.GenerateMeetingPrep(
		r.Context(),
		req.FamilyID,
		meetingDate,
		req.MeetingType,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

type GeneratePortfolioRecommendationRequest struct {
	FamilyID          string                     `json:"family_id"`
	CurrentAllocation map[string]decimal.Decimal `json:"current_allocation"`
	RiskTolerance     string                     `json:"risk_tolerance"`
}

func (h *WealthVisionHandlers) GeneratePortfolioRecommendation(w http.ResponseWriter, r *http.Request) {
	var req GeneratePortfolioRecommendationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := h.aiService.GeneratePortfolioRecommendation(
		r.Context(),
		req.FamilyID,
		req.CurrentAllocation,
		req.RiskTolerance,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// ==============================================================================
// ESG INTELLIGENCE HANDLERS
// ==============================================================================

type CalculateCarbonFootprintRequest struct {
	FamilyID       string                     `json:"family_id"`
	PortfolioValue decimal.Decimal            `json:"portfolio_value"`
	Holdings       map[string]decimal.Decimal `json:"holdings"`
}

func (h *WealthVisionHandlers) CalculateCarbonFootprint(w http.ResponseWriter, r *http.Request) {
	var req CalculateCarbonFootprintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := h.esgService.CalculateCarbonFootprint(
		r.Context(),
		req.FamilyID,
		req.PortfolioValue,
		req.Holdings,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

type CalculateESGScoreRequest struct {
	FamilyID       string                     `json:"family_id"`
	PortfolioValue decimal.Decimal            `json:"portfolio_value"`
	Holdings       map[string]decimal.Decimal `json:"holdings"`
}

func (h *WealthVisionHandlers) CalculateESGScore(w http.ResponseWriter, r *http.Request) {
	var req CalculateESGScoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := h.esgService.CalculateESGScore(
		r.Context(),
		req.FamilyID,
		req.PortfolioValue,
		req.Holdings,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

type TrackImpactInvestmentRequest struct {
	FamilyID         string          `json:"family_id"`
	InvestmentName   string          `json:"investment_name"`
	InvestmentAmount decimal.Decimal `json:"investment_amount"`
	ImpactTheme      string          `json:"impact_theme"`
}

func (h *WealthVisionHandlers) TrackImpactInvestment(w http.ResponseWriter, r *http.Request) {
	var req TrackImpactInvestmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := h.esgService.TrackImpactInvestment(
		r.Context(),
		req.FamilyID,
		req.InvestmentName,
		req.InvestmentAmount,
		req.ImpactTheme,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// ==============================================================================
// HELPER FUNCTIONS
// ==============================================================================

func parseDate(dateStr string) (time.Time, error) {
	// Try various date formats
	formats := []string{
		"2006-01-02",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
	}

	var t time.Time
	var err error
	for _, format := range formats {
		t, err = time.Parse(format, dateStr)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, err
}
