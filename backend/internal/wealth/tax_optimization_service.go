package wealth

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// TaxOptimizationService handles advanced tax optimization strategies
type TaxOptimizationService struct {
	db *pgxpool.Pool
}

// NewTaxOptimizationService creates a new tax optimization service
func NewTaxOptimizationService(db *pgxpool.Pool) *TaxOptimizationService {
	return &TaxOptimizationService{
		db: db,
	}
}

// ============================================================================
// STATE RESIDENCY OPTIMIZER
// ============================================================================

// CompareStateResidencies compares tax implications across multiple states
func (s *TaxOptimizationService) CompareStateResidencies(
	ctx context.Context,
	familyID string,
	currentState string,
	grossIncome decimal.Decimal,
	investmentIncome decimal.Decimal,
	capitalGains decimal.Decimal,
	estateValue decimal.Decimal,
	statesToCompare []string,
) (*StateResidencyComparison, error) {
	comparison := &StateResidencyComparison{
		FamilyID:         familyID,
		AnalysisDate:     time.Now(),
		CurrentState:     currentState,
		GrossIncome:      grossIncome,
		InvestmentIncome: investmentIncome,
		CapitalGains:     capitalGains,
		EstateValue:      estateValue,
		StateComparisons: []StateResidencyDetail{},
	}

	// Get current state data
	currentDetail, err := s.calculateStateDetail(ctx, currentState, grossIncome, investmentIncome, capitalGains, estateValue)
	if err != nil {
		return nil, err
	}

	comparison.StateComparisons = append(comparison.StateComparisons, *currentDetail)

	// Compare with other states
	for _, stateCode := range statesToCompare {
		if stateCode == currentState {
			continue
		}

		detail, err := s.calculateStateDetail(ctx, stateCode, grossIncome, investmentIncome, capitalGains, estateValue)
		if err != nil {
			return nil, err
		}

		// Calculate savings vs. current state
		detail.AnnualSavingsVsCurrent = currentDetail.TotalAnnualTax.Sub(detail.TotalAnnualTax)
		detail.LifetimeSavings30yr = detail.AnnualSavingsVsCurrent.Mul(decimal.NewFromInt(30))

		comparison.StateComparisons = append(comparison.StateComparisons, *detail)
	}

	// Sort by savings (highest first)
	sort.Slice(comparison.StateComparisons, func(i, j int) bool {
		return comparison.StateComparisons[i].AnnualSavingsVsCurrent.GreaterThan(
			comparison.StateComparisons[j].AnnualSavingsVsCurrent,
		)
	})

	// Top 3 recommendations
	if len(comparison.StateComparisons) > 3 {
		comparison.TopRecommendations = comparison.StateComparisons[:3]
	} else {
		comparison.TopRecommendations = comparison.StateComparisons
	}

	// Add considerations
	comparison.AdditionalConsiderations = []string{
		"Consider total cost of living including housing, property taxes, and sales tax",
		"Verify residency requirements (typically 183+ days/year)",
		"Consult with tax advisor regarding domicile change procedures",
		"Estate tax may sunset in 2026 - monitor for legislative changes",
		"Some states have 'exit taxes' or special rules for high-income earners",
	}

	return comparison, nil
}

// calculateStateDetail calculates tax details for a specific state
func (s *TaxOptimizationService) calculateStateDetail(
	ctx context.Context,
	stateCode string,
	grossIncome decimal.Decimal,
	investmentIncome decimal.Decimal,
	capitalGains decimal.Decimal,
	estateValue decimal.Decimal,
) (*StateResidencyDetail, error) {
	// State tax data (simplified - in production, pull from database)
	stateTaxData := s.getStateTaxData()

	data, exists := stateTaxData[stateCode]
	if !exists {
		return nil, fmt.Errorf("state code %s not found", stateCode)
	}

	detail := &StateResidencyDetail{
		StateCode:                stateCode,
		StateName:                data.Name,
		HasIncomeTax:             data.IncomeTaxRate.GreaterThan(decimal.Zero),
		HasEstateTax:             data.EstateTaxRate.GreaterThan(decimal.Zero),
		ResidencyRequirementDays: data.ResidencyDays,
		IncomeTaxRate:            data.IncomeTaxRate,
		CapitalGainsTaxRate:      data.CapitalGainsTaxRate,
	}

	// Calculate income tax
	detail.IncomeTax = grossIncome.Mul(data.IncomeTaxRate)

	// Calculate capital gains tax
	detail.CapitalGainsTax = capitalGains.Mul(data.CapitalGainsTaxRate)

	// Estimate property tax (1% of estate value as proxy)
	detail.PropertyTax = estateValue.Mul(data.PropertyTaxRate)

	// Estimate sales tax
	detail.SalesTax = grossIncome.Mul(data.SalesTaxRate).Mul(decimal.NewFromFloat(0.02)) // Assume 2% of income spent on taxable goods

	// Calculate estate tax
	if data.EstateTaxExemption.GreaterThan(decimal.Zero) && estateValue.GreaterThan(data.EstateTaxExemption) {
		taxableEstate := estateValue.Sub(data.EstateTaxExemption)
		detail.EstateTax = taxableEstate.Mul(data.EstateTaxRate)
	} else {
		detail.EstateTax = decimal.Zero
	}

	// Total annual tax
	detail.TotalAnnualTax = detail.IncomeTax.Add(detail.CapitalGainsTax).Add(detail.PropertyTax).Add(detail.SalesTax)

	detail.Notes = data.Notes

	return detail, nil
}

// StateTaxData represents tax rates for a state
type StateTaxData struct {
	Name                string
	IncomeTaxRate       decimal.Decimal
	CapitalGainsTaxRate decimal.Decimal
	PropertyTaxRate     decimal.Decimal
	SalesTaxRate        decimal.Decimal
	EstateTaxRate       decimal.Decimal
	EstateTaxExemption  decimal.Decimal
	ResidencyDays       int
	Notes               string
}

// getStateTaxData returns state tax data (in production, this would query the database)
func (s *TaxOptimizationService) getStateTaxData() map[string]StateTaxData {
	return map[string]StateTaxData{
		"CA": {
			Name:                "California",
			IncomeTaxRate:       decimal.NewFromFloat(0.133), // Top marginal rate
			CapitalGainsTaxRate: decimal.NewFromFloat(0.133), // Same as income
			PropertyTaxRate:     decimal.NewFromFloat(0.0076),
			SalesTaxRate:        decimal.NewFromFloat(0.0825),
			EstateTaxRate:       decimal.Zero,
			EstateTaxExemption:  decimal.Zero,
			ResidencyDays:       183,
			Notes:               "Highest state income tax; no estate tax",
		},
		"NY": {
			Name:                "New York",
			IncomeTaxRate:       decimal.NewFromFloat(0.109), // Top marginal rate
			CapitalGainsTaxRate: decimal.NewFromFloat(0.109),
			PropertyTaxRate:     decimal.NewFromFloat(0.017),
			SalesTaxRate:        decimal.NewFromFloat(0.0852),
			EstateTaxRate:       decimal.NewFromFloat(0.16),
			EstateTaxExemption:  decimal.NewFromInt(6580000),
			ResidencyDays:       183,
			Notes:               "High state income tax; estate tax applies above $6.58M",
		},
		"FL": {
			Name:                "Florida",
			IncomeTaxRate:       decimal.Zero,
			CapitalGainsTaxRate: decimal.Zero,
			PropertyTaxRate:     decimal.NewFromFloat(0.0098),
			SalesTaxRate:        decimal.NewFromFloat(0.065),
			EstateTaxRate:       decimal.Zero,
			EstateTaxExemption:  decimal.Zero,
			ResidencyDays:       183,
			Notes:               "No state income tax; no estate tax; popular for UHNW",
		},
		"TX": {
			Name:                "Texas",
			IncomeTaxRate:       decimal.Zero,
			CapitalGainsTaxRate: decimal.Zero,
			PropertyTaxRate:     decimal.NewFromFloat(0.0181), // Among highest property tax
			SalesTaxRate:        decimal.NewFromFloat(0.0625),
			EstateTaxRate:       decimal.Zero,
			EstateTaxExemption:  decimal.Zero,
			ResidencyDays:       183,
			Notes:               "No state income tax; no estate tax; high property tax",
		},
		"NV": {
			Name:                "Nevada",
			IncomeTaxRate:       decimal.Zero,
			CapitalGainsTaxRate: decimal.Zero,
			PropertyTaxRate:     decimal.NewFromFloat(0.0069),
			SalesTaxRate:        decimal.NewFromFloat(0.0685),
			EstateTaxRate:       decimal.Zero,
			EstateTaxExemption:  decimal.Zero,
			ResidencyDays:       183,
			Notes:               "No state income tax; no estate tax; low property tax",
		},
		"WA": {
			Name:                "Washington",
			IncomeTaxRate:       decimal.Zero,
			CapitalGainsTaxRate: decimal.NewFromFloat(0.07), // On capital gains > $250k
			PropertyTaxRate:     decimal.NewFromFloat(0.0092),
			SalesTaxRate:        decimal.NewFromFloat(0.0929),
			EstateTaxRate:       decimal.NewFromFloat(0.20),
			EstateTaxExemption:  decimal.NewFromInt(2193000),
			ResidencyDays:       183,
			Notes:               "No state income tax but 7% capital gains tax on gains >$250k; estate tax",
		},
	}
}

// ============================================================================
// NIIT (NET INVESTMENT INCOME TAX) CALCULATOR
// ============================================================================

// CalculateNIIT calculates Net Investment Income Tax (3.8%)
func (s *TaxOptimizationService) CalculateNIIT(
	ctx context.Context,
	familyID string,
	memberID string,
	taxYear int,
	filingStatus string,
	modifiedAGI decimal.Decimal,
	investmentIncomeComponents InvestmentIncomeBreakdown,
) (*NIITCalculation, error) {
	// NIIT thresholds based on filing status
	niitThreshold := s.getNIITThreshold(filingStatus)

	calc := &NIITCalculation{
		FamilyID:                   familyID,
		MemberID:                   memberID,
		TaxYear:                    taxYear,
		FilingStatus:               filingStatus,
		ModifiedAGI:                modifiedAGI,
		NIITThreshold:              niitThreshold,
		NetInvestmentIncome:        investmentIncomeComponents.NetInvestmentIncome,
		InvestmentIncomeComponents: investmentIncomeComponents,
	}

	// Calculate excess MAGI over threshold
	calc.ExcessOverThreshold = modifiedAGI.Sub(niitThreshold)
	if calc.ExcessOverThreshold.LessThan(decimal.Zero) {
		calc.ExcessOverThreshold = decimal.Zero
	}

	// Taxable NII is the LESSER of:
	// 1. Net investment income
	// 2. Excess MAGI over threshold
	if investmentIncomeComponents.NetInvestmentIncome.LessThan(calc.ExcessOverThreshold) {
		calc.TaxableNII = investmentIncomeComponents.NetInvestmentIncome
	} else {
		calc.TaxableNII = calc.ExcessOverThreshold
	}

	// NIIT = 3.8% of taxable NII
	niitRate := decimal.NewFromFloat(0.038)
	calc.NIITax = calc.TaxableNII.Mul(niitRate)

	// Effective rate
	if modifiedAGI.GreaterThan(decimal.Zero) {
		calc.EffectiveNIITRate = calc.NIITax.Div(modifiedAGI).Mul(decimal.NewFromInt(100))
	}

	// Generate mitigation strategies
	calc.MitigationStrategies = s.generateNIITMitigationStrategies(calc)

	return calc, nil
}

// getNIITThreshold returns NIIT threshold for filing status
func (s *TaxOptimizationService) getNIITThreshold(filingStatus string) decimal.Decimal {
	thresholds := map[string]decimal.Decimal{
		"SINGLE":            decimal.NewFromInt(200000),
		"MARRIED_JOINT":     decimal.NewFromInt(250000),
		"MARRIED_SEPARATE":  decimal.NewFromInt(125000),
		"HEAD_OF_HOUSEHOLD": decimal.NewFromInt(200000),
	}

	threshold, exists := thresholds[filingStatus]
	if !exists {
		return decimal.NewFromInt(200000) // Default to single
	}

	return threshold
}

// generateNIITMitigationStrategies generates strategies to reduce NIIT
func (s *TaxOptimizationService) generateNIITMitigationStrategies(calc *NIITCalculation) []MitigationStrategy {
	strategies := []MitigationStrategy{}

	// Strategy 1: Tax-loss harvesting
	if calc.InvestmentIncomeComponents.CapitalGains.GreaterThan(decimal.NewFromInt(10000)) {
		estimatedReduction := calc.InvestmentIncomeComponents.CapitalGains.Mul(decimal.NewFromFloat(0.20)).Mul(decimal.NewFromFloat(0.038))
		strategies = append(strategies, MitigationStrategy{
			StrategyName:       "Tax-Loss Harvesting",
			EstimatedReduction: estimatedReduction,
			ImplementationCost: decimal.NewFromInt(500),
			NetBenefit:         estimatedReduction.Sub(decimal.NewFromInt(500)),
			Difficulty:         "LOW",
			Description:        "Harvest investment losses to offset capital gains and reduce NII",
			RequiresCPAConsult: false,
		})
	}

	// Strategy 2: Municipal bonds
	if calc.InvestmentIncomeComponents.Interest.GreaterThan(decimal.NewFromInt(50000)) {
		estimatedReduction := calc.InvestmentIncomeComponents.Interest.Mul(decimal.NewFromFloat(0.50)).Mul(decimal.NewFromFloat(0.038))
		strategies = append(strategies, MitigationStrategy{
			StrategyName:       "Municipal Bond Allocation",
			EstimatedReduction: estimatedReduction,
			ImplementationCost: decimal.NewFromInt(1000),
			NetBenefit:         estimatedReduction.Sub(decimal.NewFromInt(1000)),
			Difficulty:         "MEDIUM",
			Description:        "Shift taxable bond allocation to tax-exempt municipal bonds",
			RequiresCPAConsult: true,
		})
	}

	// Strategy 3: Qualified Opportunity Zones
	if calc.InvestmentIncomeComponents.CapitalGains.GreaterThan(decimal.NewFromInt(100000)) {
		estimatedReduction := calc.InvestmentIncomeComponents.CapitalGains.Mul(decimal.NewFromFloat(0.15)).Mul(decimal.NewFromFloat(0.038))
		strategies = append(strategies, MitigationStrategy{
			StrategyName:       "Qualified Opportunity Zone Investment",
			EstimatedReduction: estimatedReduction,
			ImplementationCost: decimal.NewFromInt(5000),
			NetBenefit:         estimatedReduction.Sub(decimal.NewFromInt(5000)),
			Difficulty:         "HIGH",
			Description:        "Defer capital gains via QOZ investment (10+ year hold for full benefit)",
			RequiresCPAConsult: true,
		})
	}

	// Strategy 4: Real estate professional status
	if calc.InvestmentIncomeComponents.RentalIncome.GreaterThan(decimal.NewFromInt(50000)) {
		estimatedReduction := calc.InvestmentIncomeComponents.RentalIncome.Mul(decimal.NewFromFloat(0.038))
		strategies = append(strategies, MitigationStrategy{
			StrategyName:       "Real Estate Professional Status",
			EstimatedReduction: estimatedReduction,
			ImplementationCost: decimal.NewFromInt(2000),
			NetBenefit:         estimatedReduction.Sub(decimal.NewFromInt(2000)),
			Difficulty:         "HIGH",
			Description:        "Qualify as real estate professional to convert passive rental income to active (requires 750+ hours/year)",
			RequiresCPAConsult: true,
		})
	}

	return strategies
}

// ============================================================================
// CHARITABLE BUNCHING ANALYZER
// ============================================================================

// AnalyzeCharitableBunching analyzes bunching vs. annual giving strategies
func (s *TaxOptimizationService) AnalyzeCharitableBunching(
	ctx context.Context,
	familyID string,
	memberID string,
	analysisYears int,
	annualCharitableGiving decimal.Decimal,
	standardDeduction decimal.Decimal,
	itemizedDeductions decimal.Decimal,
	marginalTaxRate decimal.Decimal,
) (*CharitableBunchingAnalysis, error) {
	analysis := &CharitableBunchingAnalysis{
		FamilyID:               familyID,
		MemberID:               memberID,
		AnalysisYears:          analysisYears,
		AnnualCharitableGiving: annualCharitableGiving,
		StandardDeduction:      standardDeduction,
		ItemizedDeductions:     itemizedDeductions,
		MarginalTaxRate:        marginalTaxRate,
	}

	// Baseline: Annual giving
	analysis.BaselineScenario = s.simulateAnnualGiving(
		analysisYears,
		annualCharitableGiving,
		standardDeduction,
		itemizedDeductions,
		marginalTaxRate,
	)

	// Bunching: 3-year bunching strategy
	analysis.BunchingScenario = s.simulate3YearBunching(
		analysisYears,
		annualCharitableGiving,
		standardDeduction,
		itemizedDeductions,
		marginalTaxRate,
	)

	// Calculate savings
	analysis.EstimatedTaxSavings = analysis.BunchingScenario.TotalTaxSavings.Sub(analysis.BaselineScenario.TotalTaxSavings)

	// Recommendation
	if analysis.EstimatedTaxSavings.GreaterThan(decimal.NewFromInt(5000)) {
		analysis.RecommendedStrategy = "BUNCHING_3YR"

		// DAF recommendation
		analysis.DAFRecommendation = &DAFRecommendation{
			RecommendedProvider: "Fidelity Charitable or Schwab Charitable",
			InitialContribution: annualCharitableGiving.Mul(decimal.NewFromInt(3)),
			ProjectedGrowth:     annualCharitableGiving.Mul(decimal.NewFromInt(3)).Mul(decimal.NewFromFloat(0.06)),
			AnnualGrantBudget:   annualCharitableGiving,
			EstimatedFees:       annualCharitableGiving.Mul(decimal.NewFromInt(3)).Mul(decimal.NewFromFloat(0.006)),
			NetCharitableImpact: annualCharitableGiving.Mul(decimal.NewFromInt(int64(analysisYears))),
		}
	} else {
		analysis.RecommendedStrategy = "ANNUAL"
	}

	return analysis, nil
}

// simulateAnnualGiving simulates annual charitable giving
func (s *TaxOptimizationService) simulateAnnualGiving(
	years int,
	annualGiving decimal.Decimal,
	standardDeduction decimal.Decimal,
	otherItemizedDeductions decimal.Decimal,
	marginalTaxRate decimal.Decimal,
) CharitableGivingScenario {
	scenario := CharitableGivingScenario{
		ScenarioName:        "Annual Giving",
		YearByYearBreakdown: []CharitableYearDetail{},
		TotalContributions:  decimal.Zero,
		TotalDeductions:     decimal.Zero,
		TotalTaxSavings:     decimal.Zero,
	}

	for year := 1; year <= years; year++ {
		totalItemized := otherItemizedDeductions.Add(annualGiving)
		itemizes := totalItemized.GreaterThan(standardDeduction)

		var deductionTaken decimal.Decimal
		if itemizes {
			deductionTaken = totalItemized
		} else {
			deductionTaken = standardDeduction
		}

		// Tax savings from charitable deduction
		var taxSavings decimal.Decimal
		if itemizes {
			taxSavings = annualGiving.Mul(marginalTaxRate)
		} else {
			taxSavings = decimal.Zero
		}

		yearDetail := CharitableYearDetail{
			Year:              year,
			Contribution:      annualGiving,
			ItemizesDeduction: itemizes,
			DeductionTaken:    deductionTaken,
			TaxSavings:        taxSavings,
		}

		scenario.YearByYearBreakdown = append(scenario.YearByYearBreakdown, yearDetail)
		scenario.TotalContributions = scenario.TotalContributions.Add(annualGiving)
		scenario.TotalDeductions = scenario.TotalDeductions.Add(deductionTaken)
		scenario.TotalTaxSavings = scenario.TotalTaxSavings.Add(taxSavings)
	}

	if scenario.TotalContributions.GreaterThan(decimal.Zero) {
		scenario.EffectiveDeductionRate = scenario.TotalTaxSavings.Div(scenario.TotalContributions)
	}

	return scenario
}

// simulate3YearBunching simulates 3-year bunching strategy
func (s *TaxOptimizationService) simulate3YearBunching(
	years int,
	annualGiving decimal.Decimal,
	standardDeduction decimal.Decimal,
	otherItemizedDeductions decimal.Decimal,
	marginalTaxRate decimal.Decimal,
) CharitableGivingScenario {
	scenario := CharitableGivingScenario{
		ScenarioName:        "3-Year Bunching",
		YearByYearBreakdown: []CharitableYearDetail{},
		TotalContributions:  decimal.Zero,
		TotalDeductions:     decimal.Zero,
		TotalTaxSavings:     decimal.Zero,
	}

	for year := 1; year <= years; year++ {
		var contribution decimal.Decimal

		// Bunch every 3 years
		if (year-1)%3 == 0 {
			contribution = annualGiving.Mul(decimal.NewFromInt(3))
		} else {
			contribution = decimal.Zero
		}

		totalItemized := otherItemizedDeductions.Add(contribution)
		itemizes := totalItemized.GreaterThan(standardDeduction)

		var deductionTaken decimal.Decimal
		if itemizes {
			deductionTaken = totalItemized
		} else {
			deductionTaken = standardDeduction
		}

		var taxSavings decimal.Decimal
		if itemizes {
			taxSavings = contribution.Mul(marginalTaxRate)
		} else {
			taxSavings = decimal.Zero
		}

		yearDetail := CharitableYearDetail{
			Year:              year,
			Contribution:      contribution,
			ItemizesDeduction: itemizes,
			DeductionTaken:    deductionTaken,
			TaxSavings:        taxSavings,
		}

		scenario.YearByYearBreakdown = append(scenario.YearByYearBreakdown, yearDetail)
		scenario.TotalContributions = scenario.TotalContributions.Add(contribution)
		scenario.TotalDeductions = scenario.TotalDeductions.Add(deductionTaken)
		scenario.TotalTaxSavings = scenario.TotalTaxSavings.Add(taxSavings)
	}

	if scenario.TotalContributions.GreaterThan(decimal.Zero) {
		scenario.EffectiveDeductionRate = scenario.TotalTaxSavings.Div(scenario.TotalContributions)
	}

	return scenario
}

// ============================================================================
// SAVE TAX STRATEGY
// ============================================================================

// SaveTaxStrategy persists a tax strategy to the database
func (s *TaxOptimizationService) SaveTaxStrategy(
	ctx context.Context,
	strategy *TaxStrategy,
) error {
	if strategy.StrategyID == "" {
		strategy.StrategyID = uuid.New().String()
	}

	// TODO: Implement database persistence once migration is run
	// For now, just return nil to allow service to compile
	return nil
}
