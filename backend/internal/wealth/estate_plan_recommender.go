package wealth

import (
	"context"
	"fmt"
	"sort"

	"github.com/shopspring/decimal"
)

// EstatePlanRecommender provides AI-powered estate planning recommendations
type EstatePlanRecommender struct {
	familyOfficeService *FamilyOfficeService
	taxCalcService      *TaxCalculationService
}

// NewEstatePlanRecommender creates a new recommender
func NewEstatePlanRecommender(
	familyOfficeService *FamilyOfficeService,
	taxCalcService *TaxCalculationService,
) *EstatePlanRecommender {
	return &EstatePlanRecommender{
		familyOfficeService: familyOfficeService,
		taxCalcService:      taxCalcService,
	}
}

// RecommendStrategies recommends estate planning strategies for a family
func (r *EstatePlanRecommender) RecommendStrategies(ctx context.Context, familyID string) ([]StrategyRecommendation, error) {
	// Get family profile
	profile, err := r.familyOfficeService.GetFamilyProfile(ctx, familyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get family profile: %w", err)
	}

	recommendations := []StrategyRecommendation{}

	// Rule 1: Annual Gifting - Always recommend if married with children
	if profile.MarriedCouple && profile.HasChildren {
		score := r.scoreAnnualGifting(profile)
		recommendations = append(recommendations, StrategyRecommendation{
			StrategyType: "ANNUAL_GIFTING",
			StrategyName: "Annual Exclusion Gifting",
			Score:        score,
			Reasoning: []string{
				"Married couple can gift $37,000 per recipient annually ($18,500 x 2)",
				fmt.Sprintf("With %d children, can transfer up to $%s/year tax-free",
					len(profile.Members), decimal.NewFromInt(int64(len(profile.Members)*37000))),
				"Removes future appreciation from estate",
				"No gift tax filing required if within annual exclusion",
			},
			EstimatedTaxSavings:   r.calculateAnnualGiftingSavings(profile),
			ComplexityScore:       2,
			ImplementationWeeks:   1,
			AnnualMaintenanceCost: decimal.NewFromInt(500),
		})
	}

	// Rule 2: SLAT - Recommend if married, high net worth, and lifetime exemption available
	if profile.MarriedCouple && profile.TotalNetworth.GreaterThan(decimal.NewFromInt(5000000)) {
		score := r.scoreSLAT(profile)
		if score > 0.6 {
			recommendations = append(recommendations, StrategyRecommendation{
				StrategyType: "SLAT",
				StrategyName: "Spousal Lifetime Access Trust",
				Score:        score,
				Reasoning: []string{
					"Married couple with significant assets",
					"Can use lifetime exemption ($13.99M in 2025)",
					"Spouse retains indirect access to assets",
					"Locks in current exemption before potential reduction",
					"Removes all future growth from taxable estate",
				},
				EstimatedTaxSavings:   r.calculateSLATSavings(profile),
				ComplexityScore:       6,
				ImplementationWeeks:   8,
				AnnualMaintenanceCost: decimal.NewFromInt(5000),
			})
		}
	}

	// Rule 3: Dynasty Trust - Recommend if has grandchildren and very high net worth
	if profile.HasGrandchildren && profile.TotalNetworth.GreaterThan(decimal.NewFromInt(15000000)) {
		score := r.scoreDynastyTrust(profile)
		if score > 0.7 {
			recommendations = append(recommendations, StrategyRecommendation{
				StrategyType: "DYNASTY_TRUST",
				StrategyName: "Dynasty Trust",
				Score:        score,
				Reasoning: []string{
					"Multi-generational family structure",
					"Eliminates estate tax at every generation",
					"Can last in perpetuity in many states",
					"Protects assets from creditors and divorces",
					fmt.Sprintf("Estimated benefit over 3 generations: $%s",
						r.calculateDynastyBenefit(profile)),
				},
				EstimatedTaxSavings:   r.calculateDynastyTrustSavings(profile),
				ComplexityScore:       8,
				ImplementationWeeks:   16,
				AnnualMaintenanceCost: decimal.NewFromInt(10000),
			})
		}
	}

	// Rule 4: GRAT - Recommend if significant appreciating assets
	if profile.BusinessInterestPct.GreaterThan(decimal.NewFromInt(20)) ||
		profile.TotalNetworth.GreaterThan(decimal.NewFromInt(10000000)) {
		score := r.scoreGRAT(profile)
		if score > 0.65 {
			recommendations = append(recommendations, StrategyRecommendation{
				StrategyType: "GRAT",
				StrategyName: "Grantor Retained Annuity Trust",
				Score:        score,
				Reasoning: []string{
					"Significant appreciating assets identified",
					"Transfers future appreciation tax-free",
					"Low gift tax cost (often zero)",
					"Ideal for pre-IPO stock or growing businesses",
					"Can be repeated with multiple GRATs",
				},
				EstimatedTaxSavings:   r.calculateGRATSavings(profile),
				ComplexityScore:       7,
				ImplementationWeeks:   12,
				AnnualMaintenanceCost: decimal.NewFromInt(7500),
			})
		}
	}

	// Rule 5: ILIT - Recommend if inadequate life insurance
	if r.needsMoreLifeInsurance(profile) {
		score := r.scoreILIT(profile)
		if score > 0.5 {
			recommendations = append(recommendations, StrategyRecommendation{
				StrategyType: "ILIT",
				StrategyName: "Irrevocable Life Insurance Trust",
				Score:        score,
				Reasoning: []string{
					"Life insurance proceeds would increase estate tax",
					"ILIT removes proceeds from taxable estate",
					"Provides liquidity to pay estate taxes",
					"Can use annual exclusion gifts to pay premiums",
					"Protects proceeds from creditors",
				},
				EstimatedTaxSavings:   r.calculateILITSavings(profile),
				ComplexityScore:       5,
				ImplementationWeeks:   6,
				AnnualMaintenanceCost: decimal.NewFromInt(3000),
			})
		}
	}

	// Sort by score (highest first)
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Score > recommendations[j].Score
	})

	return recommendations, nil
}

// StrategyRecommendation represents a recommended strategy
type StrategyRecommendation struct {
	StrategyType          string          `json:"strategy_type"`
	StrategyName          string          `json:"strategy_name"`
	Score                 float64         `json:"score"` // 0.0 to 1.0
	Reasoning             []string        `json:"reasoning"`
	EstimatedTaxSavings   decimal.Decimal `json:"estimated_tax_savings"`
	ComplexityScore       int             `json:"complexity_score"` // 1-10
	ImplementationWeeks   int             `json:"implementation_weeks"`
	AnnualMaintenanceCost decimal.Decimal `json:"annual_maintenance_cost"`
	Prerequisites         []string        `json:"prerequisites,omitempty"`
	Risks                 []string        `json:"risks,omitempty"`
}

// Scoring functions for each strategy

func (r *EstatePlanRecommender) scoreAnnualGifting(profile *FamilyProfile) float64 {
	score := 0.5 // Base score

	if profile.MarriedCouple {
		score += 0.2
	}
	if profile.HasChildren {
		score += 0.2
	}
	if profile.HasGrandchildren {
		score += 0.1
	}

	return score
}

func (r *EstatePlanRecommender) scoreSLAT(profile *FamilyProfile) float64 {
	score := 0.0

	if !profile.MarriedCouple {
		return 0.0 // Must be married
	}

	// Net worth scoring
	if profile.TotalNetworth.GreaterThan(decimal.NewFromInt(5000000)) {
		score += 0.3
	}
	if profile.TotalNetworth.GreaterThan(decimal.NewFromInt(10000000)) {
		score += 0.2
	}
	if profile.TotalNetworth.GreaterThan(decimal.NewFromInt(20000000)) {
		score += 0.2
	}

	// Liquid assets for funding
	if profile.LiquidAssetPct.GreaterThan(decimal.NewFromInt(40)) {
		score += 0.2
	}

	// Age consideration (better when younger)
	if profile.OldestMemberAge < 60 {
		score += 0.1
	}

	return score
}

func (r *EstatePlanRecommender) scoreDynastyTrust(profile *FamilyProfile) float64 {
	score := 0.0

	if !profile.HasGrandchildren {
		return 0.5 // Lower score without grandchildren
	}

	// High net worth requirement
	if profile.TotalNetworth.GreaterThan(decimal.NewFromInt(15000000)) {
		score += 0.4
	}
	if profile.TotalNetworth.GreaterThan(decimal.NewFromInt(30000000)) {
		score += 0.2
	}

	// Multi-generational structure
	if profile.GenerationCount >= 3 {
		score += 0.3
	} else if profile.GenerationCount >= 2 {
		score += 0.1
	}

	return score
}

func (r *EstatePlanRecommender) scoreGRAT(profile *FamilyProfile) float64 {
	score := 0.3 // Base score

	// Business interests or appreciating assets
	if profile.BusinessInterestPct.GreaterThan(decimal.NewFromInt(20)) {
		score += 0.4
	}

	// High net worth
	if profile.TotalNetworth.GreaterThan(decimal.NewFromInt(10000000)) {
		score += 0.2
	}

	// Age (better when younger for longer GRAT terms)
	if profile.OldestMemberAge < 65 {
		score += 0.1
	}

	return score
}

func (r *EstatePlanRecommender) scoreILIT(profile *FamilyProfile) float64 {
	score := 0.4 // Base score

	// Estate tax threshold
	if profile.TotalNetworth.GreaterThan(decimal.NewFromInt(13990000)) {
		score += 0.3
	}

	// Married couples benefit more (spousal exemption portability)
	if profile.MarriedCouple {
		score += 0.2
	}

	// Younger age (more years of coverage needed)
	if profile.OldestMemberAge < 55 {
		score += 0.1
	}

	return score
}

// Tax savings calculations (simplified estimates)

func (r *EstatePlanRecommender) calculateAnnualGiftingSavings(profile *FamilyProfile) decimal.Decimal {
	// Assume 10 years of gifting, 7% growth, 40% tax rate
	childCount := decimal.NewFromInt(int64(len(profile.Members) - 2)) // Exclude parents
	if childCount.LessThan(decimal.NewFromInt(1)) {
		childCount = decimal.NewFromInt(2) // Assume 2 children
	}

	annualGift := decimal.NewFromInt(37000).Mul(childCount) // $37K per child (spousal split)
	tenYearTotal := annualGift.Mul(decimal.NewFromInt(10))

	// Future value with 7% growth
	futureValue := tenYearTotal.Mul(decimal.NewFromFloat(1.967)) // (1.07^10)

	// Tax savings = 40% of future value
	return futureValue.Mul(decimal.NewFromFloat(0.40))
}

func (r *EstatePlanRecommender) calculateSLATSavings(profile *FamilyProfile) decimal.Decimal {
	// Assume funding 80% of exemption ($11M), 7% growth over 20 years
	funding := decimal.NewFromInt(11000000)
	futureValue := funding.Mul(decimal.NewFromFloat(3.87)) // (1.07^20)

	// Estate tax on growth = 40%
	return futureValue.Sub(funding).Mul(decimal.NewFromFloat(0.40))
}

func (r *EstatePlanRecommender) calculateDynastyTrustSavings(profile *FamilyProfile) decimal.Decimal {
	// Conservative: $10M growing at 7% for 60 years (2 generations)
	// Eliminates 40% tax at each generation

	gen1Value := decimal.NewFromInt(10000000).Mul(decimal.NewFromFloat(5.42)) // 30 years
	gen1Tax := gen1Value.Mul(decimal.NewFromFloat(0.40))

	gen2Value := gen1Value.Mul(decimal.NewFromFloat(5.42)) // Another 30 years
	gen2Tax := gen2Value.Mul(decimal.NewFromFloat(0.40))

	return gen1Tax.Add(gen2Tax)
}

func (r *EstatePlanRecommender) calculateGRATSavings(profile *FamilyProfile) decimal.Decimal {
	// Assume $5M in GRAT, appreciation above 7520 rate (assume 3% excess)
	principal := decimal.NewFromInt(5000000)
	excessReturn := decimal.NewFromFloat(0.03) // 3% above hurdle rate
	years := decimal.NewFromInt(10)

	appreciation := principal.Mul(excessReturn).Mul(years)
	return appreciation.Mul(decimal.NewFromFloat(0.40)) // 40% tax saved
}

func (r *EstatePlanRecommender) calculateILITSavings(profile *FamilyProfile) decimal.Decimal {
	// Assume $3M death benefit, estate tax rate 40%
	return decimal.NewFromInt(3000000).Mul(decimal.NewFromFloat(0.40))
}

func (r *EstatePlanRecommender) calculateDynastyBenefit(profile *FamilyProfile) decimal.Decimal {
	return r.calculateDynastyTrustSavings(profile)
}

func (r *EstatePlanRecommender) needsMoreLifeInsurance(profile *FamilyProfile) bool {
	// Simple heuristic: if net worth > $10M, likely needs life insurance
	return profile.TotalNetworth.GreaterThan(decimal.NewFromInt(10000000))
}
