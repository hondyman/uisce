package wealth

import (
	"context"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// EstatePlanNarrativeGenerator creates natural language explanations for estate plans
type EstatePlanNarrativeGenerator struct {
	taxCalcService *TaxCalculationService
}

// NewEstatePlanNarrativeGenerator creates a new narrative generator
func NewEstatePlanNarrativeGenerator(taxCalcService *TaxCalculationService) *EstatePlanNarrativeGenerator {
	return &EstatePlanNarrativeGenerator{
		taxCalcService: taxCalcService,
	}
}

// GenerateScenarioNarrative generates a narrative for a single scenario
func (ng *EstatePlanNarrativeGenerator) GenerateScenarioNarrative(
	ctx context.Context,
	scenario *EstateScenario,
	profile *FamilyProfile,
) (string, error) {
	var parts []string

	// Introduction
	parts = append(parts, ng.generateIntroduction(scenario, profile))

	// Strategy explanation
	parts = append(parts, ng.generateStrategyExplanation(scenario))

	// Tax savings breakdown
	parts = append(parts, ng.generateTaxSavingsBreakdown(scenario))

	// Structures and entities
	parts = append(parts, ng.generateStructuresOverview(scenario))

	// Implementation steps
	parts = append(parts, ng.generateImplementationSteps(scenario))

	// Annual maintenance
	parts = append(parts, ng.generateMaintenanceRequirements(scenario))

	// Risks and considerations
	parts = append(parts, ng.generateRisksAndConsiderations(scenario))

	return strings.Join(parts, "\n\n"), nil
}

// EstateScenario represents an estate planning scenario
type EstateScenario struct {
	ScenarioID      string          `json:"scenario_id"`
	ScenarioName    string          `json:"scenario_name"`
	StrategyType    string          `json:"strategy_type"`
	TaxSavings      decimal.Decimal `json:"tax_savings"`
	TaxSavingsPct   float64         `json:"tax_savings_pct"`
	ComplexityScore int             `json:"complexity_score"`
	StructuresUsed  []string        `json:"structures_used"`
	Implementation  []string        `json:"implementation_steps"`
	AnnualCost      decimal.Decimal `json:"annual_cost"`
	Risks           []string        `json:"risks"`
}

func (ng *EstatePlanNarrativeGenerator) generateIntroduction(scenario *EstateScenario, profile *FamilyProfile) string {
	networthStr := profile.TotalNetworth.Div(decimal.NewFromInt(1000000)).StringFixed(1)
	savingsStr := scenario.TaxSavings.Div(decimal.NewFromInt(1000000)).StringFixed(1)

	return fmt.Sprintf(`**%s Estate Planning Strategy**

This %s strategy is designed for the %s family with an estimated net worth of $%sM. The plan is projected to save approximately $%sM in estate taxes (%.1f%% reduction compared to baseline).`,
		scenario.ScenarioName,
		ng.complexityLabel(scenario.ComplexityScore),
		profile.FamilyName,
		networthStr,
		savingsStr,
		scenario.TaxSavingsPct,
	)
}

func (ng *EstatePlanNarrativeGenerator) generateStrategyExplanation(scenario *EstateScenario) string {
	explanations := map[string]string{
		"ANNUAL_GIFTING": `**What This Plan Does:**
Annual exclusion gifting leverages the IRS annual gift tax exclusion ($18,500 per recipient in 2025, or $37,000 if married) to transfer wealth tax-free each year. By making systematic annual gifts to children and grandchildren, the family can reduce the taxable estate over time while removing future appreciation from the estate.`,

		"SLAT": `**What This Plan Does:**
A Spousal Lifetime Access Trust (SLAT) allows one spouse to make a large gift (up to the lifetime exemption of $13.99M) to an irrevocable trust for the benefit of the other spouse and descendants. The donating spouse "uses up" their exemption, but the non-donor spouse can still access trust assets, providing flexibility while   removing assets and future growth from the taxable estate.`,

		"DYNASTY_TRUST": `**What This Plan Does:**
A Dynasty Trust is designed to last for multiple generations (often in perpetuity in certain states). Assets placed in the trust grow tax-free and are never subject to estate tax at any generation, creating a legacy that can benefit great-grandchildren and beyond. This is particularly powerful for families with significant wealth who want to establish multi-generational financial security.`,

		"GRAT": `**What This Plan Does:**
A Grantor Retained Annuity Trust (GRAT) allows you to transfer appreciating assets (like pre-IPO stock or a growing business) while minimizing gift tax. You receive an annuity payment for a set term (typically 2-10 years), and any appreciation above the IRS hurdle rate transfers to beneficiaries tax-free. The gift tax cost is often zero or minimal.`,

		"ILIT": `**What This Plan Does:**
An Irrevocable Life Insurance Trust (ILIT) owns life insurance policies outside your taxable estate. Death benefits paid to the ILIT are estate-tax-free and can provide liquidity to pay estate taxes, equalize inheritances, or fund other planning goals. Annual exclusion gifts can be used to pay premiums.`,
	}

	if exp, ok := explanations[scenario.StrategyType]; ok {
		return exp
	}

	return "**What This Plan Does:**\n" + scenario.ScenarioName + " strategy details."
}

func (ng *EstatePlanNarrativeGenerator) generateTaxSavingsBreakdown(scenario *EstateScenario) string {
	savingsStr := scenario.TaxSavings.StringFixed(0)

	return fmt.Sprintf(`**Why This Saves Taxes:**

• **Direct tax savings**: $%s in reduced federal and state estate taxes
• **Removes future appreciation**: Assets and all future growth are excluded from your taxable estate
• **Locks in current exemption**: Uses today's $13.99M exemption before potential reduction in 2026
• **Compound benefit**: The sooner assets are removed, the more tax-free growth over time`,
		savingsStr,
	)
}

func (ng *EstatePlanNarrativeGenerator) generateStructuresOverview(scenario *EstateScenario) string {
	if len(scenario.StructuresUsed) == 0 {
		return "**Estate Planning Structures:**\n\nNo complex structures required for this strategy."
	}

	structures := strings.Join(scenario.StructuresUsed, "\n• ")
	return fmt.Sprintf("**Estate Planning Structures:**\n\nThis strategy utilizes the following legal entities:\n• %s", structures)
}

func (ng *EstatePlanNarrativeGenerator) generateImplementationSteps(scenario *EstateScenario) string {
	if len(scenario.Implementation) == 0 {
		return "**Implementation Steps:**\n\n1. Consult with estate planning attorney\n2. Prepare and sign required documents\n3. Fund the strategy as appropriate"
	}

	steps := ""
	for i, step := range scenario.Implementation {
		steps += fmt.Sprintf("%d. %s\n", i+1, step)
	}

	return "**Implementation Steps:**\n\n" + steps
}

func (ng *EstatePlanNarrativeGenerator) generateMaintenanceRequirements(scenario *EstateScenario) string {
	costStr := scenario.AnnualCost.StringFixed(0)

	return fmt.Sprintf(`**Annual Maintenance:**

• **Cost**: Approximately $%s per year
• **Tax filings**: Annual trust tax returns (Form 1041)
• **Valuations**: Periodic asset appraisals 
• **Review**: Annual planning review with advisors
• **Record keeping**: Gift tracking and documentation`,
		costStr,
	)
}

func (ng *EstatePlanNarrativeGenerator) generateRisksAndConsiderations(scenario *EstateScenario) string {
	if len(scenario.Risks) == 0 {
		return "**Risks and Considerations:**\n\n• Tax law changes could reduce benefits\n• Requires professional legal and tax guidance"
	}

	risks := strings.Join(scenario.Risks, "\n• ")
	return fmt.Sprintf("**Risks and Considerations:**\n\n• %s", risks)
}

func (ng *EstatePlanNarrativeGenerator) complexityLabel(score int) string {
	if score <= 3 {
		return "straightforward"
	} else if score <= 6 {
		return "moderate complexity"
	}
	return "sophisticated"
}

// GenerateComparisonNarrative generates a narrative comparing multiple scenarios
func (ng *EstatePlanNarrativeGenerator) GenerateComparisonNarrative(
	ctx context.Context,
	scenarios []EstateScenario,
	profile *FamilyProfile,
) (string, error) {
	if len(scenarios) < 2 {
		return "", fmt.Errorf("need at least 2 scenarios to compare")
	}

	var parts []string

	// Header
	parts = append(parts, fmt.Sprintf("**Estate Planning Scenario Comparison for the %s Family**\n", profile.FamilyName))

	// Find best on different dimensions
	bestTaxSavings := scenarios[0]
	simplest := scenarios[0]
	bestROI := scenarios[0]

	for _, s := range scenarios {
		if s.TaxSavings.GreaterThan(bestTaxSavings.TaxSavings) {
			bestTaxSavings = s
		}
		if s.ComplexityScore < simplest.ComplexityScore {
			simplest = s
		}

		// ROI = savings / cost
		currentROI := s.TaxSavings.Div(s.AnnualCost)
		bestCurrentROI := bestROI.TaxSavings.Div(bestROI.AnnualCost)
		if currentROI.GreaterThan(bestCurrentROI) {
			bestROI = s
		}
	}

	// Comparison summary
	parts = append(parts, fmt.Sprintf(`**Key Insights:**

• **Maximum Tax Savings**: %s saves $%s (%.1f%% reduction)
• **Simplest Strategy**: %s (complexity: %d/10)
• **Best ROI**: %s ($%s saved per $1 annual cost)`,
		bestTaxSavings.ScenarioName,
		bestTaxSavings.TaxSavings.Div(decimal.NewFromInt(1000000)).StringFixed(1)+"M",
		bestTaxSavings.TaxSavingsPct,
		simplest.ScenarioName,
		simplest.ComplexityScore,
		bestROI.ScenarioName,
		bestROI.TaxSavings.Div(bestROI.AnnualCost).StringFixed(0),
	))

	// Scenario-by-scenario breakdown
	parts = append(parts, "\n**Detailed Comparison:**\n")
	for i, s := range scenarios {
		parts = append(parts, fmt.Sprintf(`
%d. **%s**
   - Tax Savings: $%sM (%.1f%% reduction)
   - Complexity: %d/10 (%s)
   - Annual Cost: $%s
   - Implementation: %s`,
			i+1,
			s.ScenarioName,
			s.TaxSavings.Div(decimal.NewFromInt(1000000)).StringFixed(1),
			s.TaxSavingsPct,
			s.ComplexityScore,
			ng.complexityLabel(s.ComplexityScore),
			s.AnnualCost.StringFixed(0),
			strings.Join(s.StructuresUsed, ", "),
		))
	}

	// Recommendation
	parts = append(parts, ng.generateRecommendation(bestTaxSavings, simplest, bestROI, profile))

	return strings.Join(parts, "\n"), nil
}

func (ng *EstatePlanNarrativeGenerator) generateRecommendation(
	bestSavings, simplest, bestROI EstateScenario,
	profile *FamilyProfile,
) string {
	// Simple heuristic-based recommendation
	if profile.TotalNetworth.GreaterThan(decimal.NewFromInt(20000000)) {
		// High net worth: prioritize tax savings
		return fmt.Sprintf(`

**Recommended Approach:**

Given your family's significant net worth ($%sM), we recommend prioritizing maximum tax savings with the **%s** strategy. While more complex, the $%sM in savings substantially outweighs the implementation costs.`,
			profile.TotalNetworth.Div(decimal.NewFromInt(1000000)).StringFixed(1),
			bestSavings.ScenarioName,
			bestSavings.TaxSavings.Div(decimal.NewFromInt(1000000)).StringFixed(1),
		)
	} else if simplest.ComplexityScore <= 4 {
		// Moderate net worth: balance savings and simplicity
		return fmt.Sprintf(`

**Recommended Approach:**

For your family situation, we recommend the **%s** strategy. It offers a strong balance of tax savings ($%sM) with manageable complexity (%d/10), making it easier to implement and maintain.`,
			simplest.ScenarioName,
			simplest.TaxSavings.Div(decimal.NewFromInt(1000000)).StringFixed(1),
			simplest.ComplexityScore,
		)
	}

	return `

**Recommended Approach:**

We suggest starting with a simpler strategy and adding complexity over time as you become comfortable with the estate planning process.`
}

// GenerateExecutiveSummary generates a one-page executive summary
func (ng *EstatePlanNarrativeGenerator) GenerateExecutiveSummary(
	ctx context.Context,
	topScenario EstateScenario,
	profile *FamilyProfile,
) (string, error) {
	networthStr := profile.TotalNetworth.Div(decimal.NewFromInt(1000000)).StringFixed(1)
	savingsStr := topScenario.TaxSavings.Div(decimal.NewFromInt(1000000)).StringFixed(1)

	summary := fmt.Sprintf(`# %s Family Estate Plan - Executive Summary

## Family Overview
• Family Net Worth: $%sM
• Generations: %d
• Primary State: %s

## Recommended Strategy: %s

### Tax Impact
• **Estimated Tax Savings**: $%sM (%.1f%% reduction vs. no planning)
• **Effective Estate Tax Rate**: Reduced from 40%% to %.1f%%

### Implementation
• **Complexity**: %d/10 (%s)
• **Timeline**: %d weeks to implement
• **Annual Maintenance**: $%s

### Key Benefits
%s

### Next Steps
1. Review and approve this plan
2. Engage estate planning attorney
3. Prepare trust documents and valuations
4. Execute and fund structures
5. Establish annual review schedule

*This is a preliminary analysis. Final implementation requires professional legal and tax counsel.*`,
		profile.FamilyName,
		networthStr,
		profile.GenerationCount,
		profile.PrimaryState,
		topScenario.ScenarioName,
		savingsStr,
		topScenario.TaxSavingsPct,
		40.0-topScenario.TaxSavingsPct,
		topScenario.ComplexityScore,
		ng.complexityLabel(topScenario.ComplexityScore),
		len(topScenario.Implementation),
		topScenario.AnnualCost.StringFixed(0),
		ng.formatImplementationSummary(topScenario.Implementation),
	)

	return summary, nil
}

func (ng *EstatePlanNarrativeGenerator) formatImplementationSummary(steps []string) string {
	if len(steps) == 0 {
		return "No implementation steps defined."
	}
	limit := 3
	if len(steps) < limit {
		limit = len(steps)
	}
	return strings.Join(steps[:limit], "\n")
}
