package wealth

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// MultiGenerationalService handles dynasty trust modeling and multi-gen planning
type MultiGenerationalService struct {
	db      *pgxpool.Pool
	taxCalc *TaxCalculationService
}

// NewMultiGenerationalService creates a new multi-generational service
func NewMultiGenerationalService(db *pgxpool.Pool, taxCalc *TaxCalculationService) *MultiGenerationalService {
	return &MultiGenerationalService{
		db:      db,
		taxCalc: taxCalc,
	}
}

// ============================================================================
// DYNASTY TRUST SIMULATION
// ============================================================================

// DynastyTrustSimulation represents a multi-generational dynasty trust projection
type DynastyTrustSimulation struct {
	SimulationID       string                 `json:"simulation_id"`
	FamilyID           string                 `json:"family_id"`
	TrustName          string                 `json:"trust_name"`
	InitialFunding     decimal.Decimal        `json:"initial_funding"`
	AssumedGrowthRate  decimal.Decimal        `json:"assumed_growth_rate"`
	AssumedTaxRate     decimal.Decimal        `json:"assumed_tax_rate"`
	GenerationCount    int                    `json:"generation_count"`
	YearsPerGeneration int                    `json:"years_per_generation"`
	Generations        []GenerationProjection `json:"generations"`
	TotalTaxSavings    decimal.Decimal        `json:"total_tax_savings"`
	WealthMultiplier   decimal.Decimal        `json:"wealth_multiplier"`
	CreatedAt          time.Time              `json:"created_at"`
}

// GenerationProjection represents wealth projection for one generation
type GenerationProjection struct {
	Generation         int             `json:"generation"`
	StartYear          int             `json:"start_year"`
	EndYear            int             `json:"end_year"`
	StartingValue      decimal.Decimal `json:"starting_value"`
	GrowthValue        decimal.Decimal `json:"growth_value"`
	DistributionsTotal decimal.Decimal `json:"distributions_total"`
	EndingValue        decimal.Decimal `json:"ending_value"`
	TaxIfNoTrust       decimal.Decimal `json:"tax_if_no_trust"`
	TaxWithTrust       decimal.Decimal `json:"tax_with_trust"`
	TaxSavings         decimal.Decimal `json:"tax_savings"`
	BeneficiaryCount   int             `json:"beneficiary_count"`
	PerCapitaValue     decimal.Decimal `json:"per_capita_value"`
}

// SimulateDynastyTrust creates a multi-generational wealth projection
func (s *MultiGenerationalService) SimulateDynastyTrust(
	ctx context.Context,
	familyID string,
	trustName string,
	initialFunding decimal.Decimal,
	growthRate decimal.Decimal,
	generationCount int,
	yearsPerGeneration int,
) (*DynastyTrustSimulation, error) {
	simulation := &DynastyTrustSimulation{
		SimulationID:       uuid.New().String(),
		FamilyID:           familyID,
		TrustName:          trustName,
		InitialFunding:     initialFunding,
		AssumedGrowthRate:  growthRate,
		AssumedTaxRate:     decimal.NewFromFloat(0.40), // 40% estate tax
		GenerationCount:    generationCount,
		YearsPerGeneration: yearsPerGeneration,
		Generations:        []GenerationProjection{},
		TotalTaxSavings:    decimal.Zero,
		CreatedAt:          time.Now(),
	}

	currentValue := initialFunding
	currentYear := time.Now().Year()
	beneficiaryCount := 2 // Start with 2 G1 beneficiaries

	for gen := 1; gen <= generationCount; gen++ {
		projection := GenerationProjection{
			Generation:       gen,
			StartYear:        currentYear,
			EndYear:          currentYear + yearsPerGeneration,
			StartingValue:    currentValue,
			BeneficiaryCount: beneficiaryCount,
		}

		// Calculate growth over period
		growthMultiplier := decimal.NewFromFloat(1).Add(growthRate).Pow(decimal.NewFromInt(int64(yearsPerGeneration)))
		projection.GrowthValue = currentValue.Mul(growthMultiplier).Sub(currentValue)

		// Calculate distributions (assume 3% annual distribution)
		distributionRate := decimal.NewFromFloat(0.03)
		projection.DistributionsTotal = currentValue.Mul(distributionRate).Mul(decimal.NewFromInt(int64(yearsPerGeneration)))

		// Ending value
		projection.EndingValue = currentValue.Add(projection.GrowthValue).Sub(projection.DistributionsTotal)

		// Calculate tax if NO dynasty trust (taxed at each generation)
		projection.TaxIfNoTrust = projection.EndingValue.Mul(simulation.AssumedTaxRate)

		// Tax WITH dynasty trust (only distributions taxed, not principal)
		projection.TaxWithTrust = projection.DistributionsTotal.Mul(decimal.NewFromFloat(0.37)) // Income tax on distributions

		// Tax savings
		projection.TaxSavings = projection.TaxIfNoTrust.Sub(projection.TaxWithTrust)
		simulation.TotalTaxSavings = simulation.TotalTaxSavings.Add(projection.TaxSavings)

		// Per capita value
		projection.PerCapitaValue = projection.EndingValue.Div(decimal.NewFromInt(int64(beneficiaryCount)))

		simulation.Generations = append(simulation.Generations, projection)

		// Next generation
		currentValue = projection.EndingValue
		currentYear += yearsPerGeneration
		beneficiaryCount = beneficiaryCount * 2 // Double each generation
	}

	// Wealth multiplier
	if initialFunding.GreaterThan(decimal.Zero) {
		finalValue := simulation.Generations[len(simulation.Generations)-1].EndingValue
		simulation.WealthMultiplier = finalValue.Div(initialFunding)
	}

	// TODO: Persist to database
	return simulation, nil
}

// ============================================================================
// EDUCATION PLANNING (529 OPTIMIZATION)
// ============================================================================

// EducationPlan represents a 529 college savings plan optimization
type EducationPlan struct {
	PlanID                  string          `json:"plan_id"`
	FamilyID                string          `json:"family_id"`
	StudentMemberID         string          `json:"student_member_id"`
	StudentName             string          `json:"student_name"`
	StudentAge              int             `json:"student_age"`
	YearsUntilCollege       int             `json:"years_until_college"`
	TargetFunding           decimal.Decimal `json:"target_funding"`
	CurrentSavings          decimal.Decimal `json:"current_savings"`
	MonthlyContribution     decimal.Decimal `json:"monthly_contribution"`
	AssumedReturn           decimal.Decimal `json:"assumed_return"`
	ProjectedValue          decimal.Decimal `json:"projected_value"`
	Overfunded              bool            `json:"overfunded"`
	Gap                     decimal.Decimal `json:"gap"`
	StateTaxBenefit         decimal.Decimal `json:"state_tax_benefit"`
	RecommendedState        string          `json:"recommended_state"`
	FederalTaxBenefit       decimal.Decimal `json:"federal_tax_benefit"`
	TotalTaxBenefitLifetime decimal.Decimal `json:"total_tax_benefit_lifetime"`
	CreatedAt               time.Time       `json:"created_at"`
}

// Optimize529Plan creates an optimized education savings plan
func (s *MultiGenerationalService) Optimize529Plan(
	ctx context.Context,
	familyID string,
	studentMemberID string,
	studentAge int,
	targetFunding decimal.Decimal,
	currentSavings decimal.Decimal,
	monthlyContribution decimal.Decimal,
	homeState string,
) (*EducationPlan, error) {
	yearsUntilCollege := 18 - studentAge
	if yearsUntilCollege < 0 {
		yearsUntilCollege = 0
	}

	plan := &EducationPlan{
		PlanID:              uuid.New().String(),
		FamilyID:            familyID,
		StudentMemberID:     studentMemberID,
		StudentAge:          studentAge,
		YearsUntilCollege:   yearsUntilCollege,
		TargetFunding:       targetFunding,
		CurrentSavings:      currentSavings,
		MonthlyContribution: monthlyContribution,
		AssumedReturn:       decimal.NewFromFloat(0.07), // 7% annual return
		CreatedAt:           time.Now(),
	}

	// Calculate future value of current savings
	if yearsUntilCollege > 0 {
		futureValueCurrentSavings := currentSavings.Mul(
			decimal.NewFromFloat(1.07).Pow(decimal.NewFromInt(int64(yearsUntilCollege))),
		)

		// Calculate future value of monthly contributions (annuity)
		monthsUntilCollege := yearsUntilCollege * 12
		monthlyRate := decimal.NewFromFloat(0.07).Div(decimal.NewFromInt(12))

		// FV of annuity = PMT × ((1 + r)^n - 1) / r
		futureValueContributions := monthlyContribution.Mul(
			decimal.NewFromFloat(1).Add(monthlyRate).Pow(decimal.NewFromInt(int64(monthsUntilCollege))).
				Sub(decimal.NewFromInt(1)).
				Div(monthlyRate),
		)

		plan.ProjectedValue = futureValueCurrentSavings.Add(futureValueContributions)
	} else {
		plan.ProjectedValue = currentSavings
	}

	// Check if overfunded or gap
	if plan.ProjectedValue.GreaterThan(targetFunding) {
		plan.Overfunded = true
		plan.Gap = plan.ProjectedValue.Sub(targetFunding)
	} else {
		plan.Overfunded = false
		plan.Gap = targetFunding.Sub(plan.ProjectedValue)
	}

	// Calculate state tax benefit
	stateBenefits := s.get529StateBenefits()
	if benefit, exists := stateBenefits[homeState]; exists {
		annualContribution := monthlyContribution.Mul(decimal.NewFromInt(12))
		plan.StateTaxBenefit = annualContribution.Mul(benefit.DeductionRate).Mul(decimal.NewFromInt(int64(yearsUntilCollege)))
		plan.RecommendedState = homeState
	} else {
		// Recommend Nevada (Vanguard) or New York (low fees) if no state benefit
		plan.RecommendedState = "NV" // Nevada - Vanguard 529
		plan.StateTaxBenefit = decimal.Zero
	}

	// Federal tax benefit (tax-free growth and withdrawals)
	growthAmount := plan.ProjectedValue.Sub(currentSavings).Sub(monthlyContribution.Mul(decimal.NewFromInt(12)).Mul(decimal.NewFromInt(int64(yearsUntilCollege))))
	plan.FederalTaxBenefit = growthAmount.Mul(decimal.NewFromFloat(0.20)) // Assume 20% capital gains tax saved

	plan.TotalTaxBenefitLifetime = plan.StateTaxBenefit.Add(plan.FederalTaxBenefit)

	return plan, nil
}

// State529Benefit represents tax benefits for a state's 529 plan
type State529Benefit struct {
	StateCode     string          `json:"state_code"`
	StateName     string          `json:"state_name"`
	DeductionRate decimal.Decimal `json:"deduction_rate"` // % of contribution deductible
	MaxDeduction  decimal.Decimal `json:"max_deduction"`
	HasDeduction  bool            `json:"has_deduction"`
}

// get529StateBenefits returns state-specific 529 tax benefits
func (s *MultiGenerationalService) get529StateBenefits() map[string]State529Benefit {
	return map[string]State529Benefit{
		"NY": {
			StateCode:     "NY",
			StateName:     "New York",
			DeductionRate: decimal.NewFromFloat(0.109), // 10.9% state income tax
			MaxDeduction:  decimal.NewFromInt(10000),   // $10K max deduction
			HasDeduction:  true,
		},
		"CA": {
			StateCode:     "CA",
			StateName:     "California",
			DeductionRate: decimal.Zero, // No state deduction
			MaxDeduction:  decimal.Zero,
			HasDeduction:  false,
		},
		"IL": {
			StateCode:     "IL",
			StateName:     "Illinois",
			DeductionRate: decimal.NewFromFloat(0.0495), // 4.95% state income tax
			MaxDeduction:  decimal.NewFromInt(20000),    // $20K max deduction (married)
			HasDeduction:  true,
		},
		"VA": {
			StateCode:     "VA",
			StateName:     "Virginia",
			DeductionRate: decimal.NewFromFloat(0.0575), // 5.75% state income tax
			MaxDeduction:  decimal.NewFromInt(4000),     // $4K per account
			HasDeduction:  true,
		},
	}
}

// ============================================================================
// LEGACY IMPACT CALCULATOR
// ============================================================================

// LegacyImpact represents philanthropic legacy impact calculation
type LegacyImpact struct {
	ImpactID                 string          `json:"impact_id"`
	FamilyID                 string          `json:"family_id"`
	PhilanthropicFocus       string          `json:"philanthropic_focus"`
	AnnualGiving             decimal.Decimal `json:"annual_giving"`
	DynastyGivingComponent   decimal.Decimal `json:"dynasty_giving_component"`
	TotalProjectedGiving     decimal.Decimal `json:"total_projected_giving"`
	GenerationsImpacted      int             `json:"generations_impacted"`
	DirectBeneficiariesEst   int             `json:"direct_beneficiaries_est"`
	IndirectBeneficiariesEst int             `json:"indirect_beneficiaries_est"`
	CompoundImpactMultiplier decimal.Decimal `json:"compound_impact_multiplier"`
	LegacyRating             string          `json:"legacy_rating"` // MODEST, SIGNIFICANT, TRANSFORMATIVE, GENERATIONAL
	RecommendedStructures    []string        `json:"recommended_structures"`
	EstimatedTaxDeductions   decimal.Decimal `json:"estimated_tax_deductions"`
	NetCostAfterTax          decimal.Decimal `json:"net_cost_after_tax"`
	CreatedAt                time.Time       `json:"created_at"`
}

// CalculateLegacyImpact projects multi-generational philanthropic impact
func (s *MultiGenerationalService) CalculateLegacyImpact(
	ctx context.Context,
	familyID string,
	philanthropicFocus string,
	annualGiving decimal.Decimal,
	yearsOfGiving int,
	includesDynastyTrust bool,
	dynastyGivingPct decimal.Decimal,
) (*LegacyImpact, error) {
	impact := &LegacyImpact{
		ImpactID:           uuid.New().String(),
		FamilyID:           familyID,
		PhilanthropicFocus: philanthropicFocus,
		AnnualGiving:       annualGiving,
		CreatedAt:          time.Time{},
	}

	// Calculate total projected giving
	impact.TotalProjectedGiving = annualGiving.Mul(decimal.NewFromInt(int64(yearsOfGiving)))

	// Add dynasty trust giving component if applicable
	if includesDynastyTrust {
		// Assume dynasty trust distributes 3% annually for charitable purposes
		dynastyValue := annualGiving.Mul(decimal.NewFromInt(20))                                        // Estimate
		impact.DynastyGivingComponent = dynastyValue.Mul(dynastyGivingPct).Mul(decimal.NewFromInt(100)) // 100 years
		impact.TotalProjectedGiving = impact.TotalProjectedGiving.Add(impact.DynastyGivingComponent)
		impact.GenerationsImpacted = 4 // 100 years ≈ 4 generations
	} else {
		impact.GenerationsImpacted = 1
	}

	// Estimate beneficiaries based on focus area
	beneficiaryMultipliers := map[string]struct {
		Direct   int
		Indirect int
	}{
		"EDUCATION":   {Direct: 50, Indirect: 200},    // Students and families
		"HEALTHCARE":  {Direct: 30, Indirect: 150},    // Patients and communities
		"POVERTY":     {Direct: 100, Indirect: 400},   // Families and communities
		"ENVIRONMENT": {Direct: 1000, Indirect: 5000}, // Broader impact
		"ARTS":        {Direct: 500, Indirect: 2000},
		"GENERAL":     {Direct: 50, Indirect: 200},
	}

	multiplier := beneficiaryMultipliers["GENERAL"]
	if m, exists := beneficiaryMultipliers[philanthropicFocus]; exists {
		multiplier = m
	}

	// Beneficiaries = (Total Giving / $1000) × Multiplier
	grantEquivalent := impact.TotalProjectedGiving.Div(decimal.NewFromInt(1000))
	impact.DirectBeneficiariesEst = int(grantEquivalent.IntPart()) * multiplier.Direct
	impact.IndirectBeneficiariesEst = int(grantEquivalent.IntPart()) * multiplier.Indirect

	// Compound impact multiplier (ripple effect)
	if includesDynastyTrust {
		impact.CompoundImpactMultiplier = decimal.NewFromFloat(5.0) // Generational giving has 5× ripple effect
	} else {
		impact.CompoundImpactMultiplier = decimal.NewFromFloat(2.5)
	}

	// Legacy rating
	totalInMillions := impact.TotalProjectedGiving.Div(decimal.NewFromInt(1000000))
	switch {
	case totalInMillions.LessThan(decimal.NewFromInt(1)):
		impact.LegacyRating = "MODEST"
	case totalInMillions.LessThan(decimal.NewFromInt(10)):
		impact.LegacyRating = "SIGNIFICANT"
	case totalInMillions.LessThan(decimal.NewFromInt(100)):
		impact.LegacyRating = "TRANSFORMATIVE"
	default:
		impact.LegacyRating = "GENERATIONAL"
	}

	// Recommended structures
	impact.RecommendedStructures = []string{}
	if totalInMillions.GreaterThan(decimal.NewFromInt(5)) {
		impact.RecommendedStructures = append(impact.RecommendedStructures, "PRIVATE_FOUNDATION")
	}
	if totalInMillions.GreaterThanOrEqual(decimal.NewFromInt(1)) {
		impact.RecommendedStructures = append(impact.RecommendedStructures, "DONOR_ADVISED_FUND")
	}
	if includesDynastyTrust {
		impact.RecommendedStructures = append(impact.RecommendedStructures, "CHARITABLE_DYNASTY_TRUST")
	}

	// Tax deductions (assume 37% marginal rate)
	impact.EstimatedTaxDeductions = impact.TotalProjectedGiving.Mul(decimal.NewFromFloat(0.37))
	impact.NetCostAfterTax = impact.TotalProjectedGiving.Sub(impact.EstimatedTaxDeductions)

	return impact, nil
}
