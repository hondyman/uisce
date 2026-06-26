package ai

import (
	"fmt"
	"strings"

	"github.com/hondyman/semlayer/backend/pkg/factors"
)

// NarrativeGenerator generates explanations for financial data
type NarrativeGenerator struct{}

func NewNarrativeGenerator() *NarrativeGenerator {
	return &NarrativeGenerator{}
}

// GenerateFactorNarrative creates a natural language explanation of factor exposures
func (g *NarrativeGenerator) GenerateFactorNarrative(analysis factors.FactorAnalysisResult) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Analysis of **%s** using the **%s** model (R²=%.2f):\n\n", analysis.EntityID, analysis.ModelName, analysis.R2))

	// Alpha check
	if analysis.Alpha > 0.001 { // > 10 bps
		sb.WriteString(fmt.Sprintf("✅ **Positive Alpha**: The portfolio is generating %.2f%% monthly excess return unexplained by risk factors.\n", analysis.Alpha*100))
	} else if analysis.Alpha < -0.001 {
		sb.WriteString(fmt.Sprintf("⚠️ **Negative Alpha**: The portfolio is underperforming by %.2f%% monthly after adjusting for risk.\n", analysis.Alpha*100))
	}

	sb.WriteString("\n**Key Factor Exposures:**\n")

	for _, exp := range analysis.Exposures {
		// Only comment on significant exposures (t-stat > 2 or beta magnitude > 0.1)
		if exp.TStat > 2.0 || exp.Beta > 0.1 || exp.Beta < -0.1 {
			interpretation := interpretBeta(exp.FactorName, exp.Beta)
			sb.WriteString(fmt.Sprintf("- **%s** (β=%.2f): %s\n", exp.FactorName, exp.Beta, interpretation))
		}
	}

	return sb.String()
}

func interpretBeta(factor string, beta float64) string {
	switch factor {
	case factors.FactorMarket:
		if beta > 1.1 {
			return "High market sensitivity (aggressive)."
		} else if beta < 0.9 {
			return "Low market sensitivity (defensive)."
		}
		return "Market-like risk."
	case factors.FactorSMB:
		if beta > 0.2 {
			return "Tilt towards Small Cap stocks."
		} else if beta < -0.2 {
			return "Tilt towards Large Cap stocks."
		}
	case factors.FactorHML:
		if beta > 0.2 {
			return "Tilt towards Value stocks."
		} else if beta < -0.2 {
			return "Tilt towards Growth stocks."
		}
	case factors.FactorRMW:
		if beta > 0.2 {
			return "Bias towards high profitability firms."
		}
	case factors.FactorCMA:
		if beta > 0.2 {
			return "Bias towards conservative investment firms."
		}
	}
	return "Neutral exposure."
}
