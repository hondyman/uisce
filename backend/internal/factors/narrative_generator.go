package factors

import (
	"context"
	"fmt"
	"math"
	"strings"
)

// NarrativeGenerator creates plain English explanations for factor exposures
type NarrativeGenerator struct {
	model FactorModel
}

// NewNarrativeGenerator creates a new narrative generator
func NewNarrativeGenerator(model FactorModel) *NarrativeGenerator {
	return &NarrativeGenerator{
		model: model,
	}
}

// GenerateExposureNarratives creates narratives for factor exposures
func (ng *NarrativeGenerator) GenerateExposureNarratives(ctx context.Context, exposures []FactorExposure, holdings []Holding) []FactorExposure {
	enriched := make([]FactorExposure, len(exposures))
	
	for i, exp := range exposures {
		enriched[i] = exp
		enriched[i].Narrative = ng.generateSingleNarrative(exp, holdings)
		enriched[i].Sources = ng.identifyDrivingHoldings(exp, holdings)
	}
	
	return enriched
}

// generateSingleNarrative creates a narrative for a single factor
func (ng *NarrativeGenerator) generateSingleNarrative(exp FactorExposure, holdings []Holding) string {
	// Determine magnitude and direction
	magnitude := math.Abs(exp.Contribution)
	direction := "neutral"
	if exp.Contribution > 0.1 {
		direction = "overweight"
	} else if exp.Contribution < -0.1 {
		direction = "underweight"
	}
	
	// Determine statistical significance
	isSignificant := exp.PValue < 0.05
	confidence := "with moderate confidence"
	if exp.PValue < 0.01 {
		confidence = "with high confidence"
	} else if exp.PValue > 0.10 {
		confidence = "but not statistically significant"
	}
	
	// Factor-specific narratives
	switch exp.Factor {
	case "Market":
		return ng.marketNarrative(exp.Contribution, magnitude, direction, confidence, isSignificant)
	case "SMB":
		return ng.smbNarrative(exp.Contribution, magnitude, direction, confidence, isSignificant)
	case "HML":
		return ng.hmlNarrative(exp.Contribution, magnitude, direction, confidence, isSignificant)
	case "RMW":
		return ng.rmwNarrative(exp.Contribution, magnitude, direction, confidence, isSignificant)
	case "CMA":
		return ng.cmaNarrative(exp.Contribution, magnitude, direction, confidence, isSignificant)
	default:
		return ng.genericNarrative(exp.Factor, exp.Contribution, magnitude, direction, confidence)
	}
}

// Factor-specific narrative generators

func (ng *NarrativeGenerator) marketNarrative(beta, magnitude float64, direction, confidence string, significant bool) string {
	if magnitude < 0.1 {
		return fmt.Sprintf("Portfolio has minimal market exposure (beta: %.2f), indicating significant hedging or market-neutral positioning %s", beta, confidence)
	}
	
	if beta > 1.2 {
		return fmt.Sprintf("Portfolio is highly sensitive to market movements (beta: %.2f), %s. Expect amplified gains in bull markets and larger losses in downturns %s", 
			beta, direction, confidence)
	}
	
	if beta < 0.8 {
		return fmt.Sprintf("Portfolio has defensive characteristics (beta: %.2f), showing lower volatility than the market %s", beta, confidence)
	}
	
	return fmt.Sprintf("Portfolio tracks the market closely (beta: %.2f) %s", beta, confidence)
}

func (ng *NarrativeGenerator) smbNarrative(beta, magnitude float64, direction, confidence string, significant bool) string {
	if magnitude < 0.05 {
		return fmt.Sprintf("Portfolio is size-neutral, with balanced exposure to large and small cap stocks %s", confidence)
	}
	
	if beta > 0.1 {
		return fmt.Sprintf("Portfolio tilts toward small-cap stocks (%.1f%% exposure), potentially capturing the size premium but with higher volatility %s",
			beta*100, confidence)
	}
	
	return fmt.Sprintf("Portfolio favors large-cap stocks (%.1f%% underweight small-cap), prioritizing stability and liquidity %s",
		math.Abs(beta)*100, confidence)
}

func (ng *NarrativeGenerator) hmlNarrative(beta, magnitude float64, direction, confidence string, significant bool) string {
	if magnitude < 0.05 {
		return fmt.Sprintf("Portfolio is balanced between value and growth stocks %s", confidence)
	}
	
	if beta > 0.1 {
		return fmt.Sprintf("Portfolio has a value tilt (%.1f%% exposure), focusing on stocks with lower price-to-book ratios %s",
			beta*100, confidence)
	}
	
	return fmt.Sprintf("Portfolio leans toward growth stocks (%.1f%% underweight value), emphasizing future earnings potential over current valuations %s",
		math.Abs(beta)*100, confidence)
}

func (ng *NarrativeGenerator) rmwNarrative(beta, magnitude float64, direction, confidence string, significant bool) string {
	if magnitude < 0.05 {
		return fmt.Sprintf("Portfolio is neutral on profitability, with mixed exposure to high and low profitability firms %s", confidence)
	}
	
	if beta > 0.1 {
		return fmt.Sprintf("Portfolio emphasizes highly profitable companies (%.1f%% exposure), focusing on firms with robust operating margins %s",
			beta*100, confidence)
	}
	
	return fmt.Sprintf("Portfolio includes lower profitability companies (%.1f%% underweight), potentially pursuing growth over current earnings %s",
		math.Abs(beta)*100, confidence)
}

func (ng *NarrativeGenerator) cmaNarrative(beta, magnitude float64, direction, confidence string, significant bool) string {
	if magnitude < 0.05 {
		return fmt.Sprintf("Portfolio is neutral on investment style %s", confidence)
	}
	
	if beta > 0.1 {
		return fmt.Sprintf("Portfolio favors conservative investment firms (%.1f%% exposure), companies that invest cautiously %s",
			beta*100, confidence)
	}
	
	return fmt.Sprintf("Portfolio tilts toward aggressive investors (%.1f%% underweight conservative), companies actively expanding assets %s",
		math.Abs(beta)*100, confidence)
}

func (ng *NarrativeGenerator) genericNarrative(factor string, beta, magnitude float64, direction, confidence string) string {
	return fmt.Sprintf("%s factor shows %.1f%% exposure (%s) %s", 
		factor, beta*100, direction, confidence)
}

// identifyDrivingHoldings finds the top holdings contributing to a factor exposure
func (ng *NarrativeGenerator) identifyDrivingHoldings(exp FactorExposure, holdings []Holding) []string {
	// TODO: Implement actual attribution logic
	// This would analyze individual security factor loadings
	
	// Placeholder: return largest holdings
	sources := make([]string, 0, 3)
	for i := 0; i < len(holdings) && i < 3; i++ {
		sources = append(sources, holdings[i].Ticker)
	}
	return sources
}

// GenerateAttributionNarrative creates a narrative for factor attribution
func (ng *NarrativeGenerator) GenerateAttributionNarrative(ctx context.Context, attribution *AttributionResult) string {
	var parts []string
	
	parts = append(parts, fmt.Sprintf("Total portfolio return: %.2f%%", attribution.TotalReturn*100))
	
	// Explain factor contributions
	if len(attribution.FactorReturns) > 0 {
		parts = append(parts, "\nFactor contributions:")
		for factor, ret := range attribution.FactorReturns {
			pct := (ret / attribution.TotalReturn) * 100
			parts = append(parts, fmt.Sprintf("  - %s: %.2f%% (%.0f%% of total)", factor, ret*100, pct))
		}
	}
	
	// Explain selection effect
	if attribution.SelectionReturn != 0 {
		pct := (attribution.SelectionReturn / attribution.TotalReturn) * 100
		parts = append(parts, fmt.Sprintf("\nStock selection: %.2f%% (%.0f%% of total)", 
			attribution.SelectionReturn*100, pct))
	}
	
	return strings.Join(parts, "\n")
}

// GenerateScenarioNarrative creates a narrative for scenario analysis
func (ng *NarrativeGenerator) GenerateScenarioNarrative(ctx context.Context, scenario *ScenarioResult, exposures []FactorExposure) string {
	// Find the exposure for the shocked factor
	var targetExp *FactorExposure
	for i := range exposures {
		if exposures[i].Factor == scenario.Scenario.Factor {
			targetExp = &exposures[i]
			break
		}
	}
	
	impact := scenario.PortfolioImpact * 100
	shock := scenario.Scenario.ShockPct * 100
	
	narrative := fmt.Sprintf("Scenario: %s experiences a %.1f%% shock.\n", 
		scenario.Scenario.Factor, shock)
	
	if targetExp != nil {
		narrative += fmt.Sprintf("Portfolio has %.1f%% exposure to this factor.\n", 
			targetExp.Contribution*100)
	}
	
	narrative += fmt.Sprintf("Estimated portfolio impact: %+.2f%%\n", impact)
	
	// Add context
	if math.Abs(impact) > 5 {
		narrative += "This represents a significant portfolio risk."
	} else if math.Abs(impact) > 2 {
		narrative += "This represents a moderate portfolio risk."
	} else {
		narrative += "Portfolio is relatively well-hedged against this scenario."
	}
	
	return narrative
}

// EnhanceWithLLM optionally enhances narratives using an LLM
func (ng *NarrativeGenerator) EnhanceWithLLM(ctx context.Context, baseNarrative string, exposures []FactorExposure) (string, error) {
	// TODO: Integrate with LLM service for more sophisticated narratives
	// This would pass the base narrative + data to an LLM and request
	// a more detailed, context-aware explanation
	
	return baseNarrative, nil
}
