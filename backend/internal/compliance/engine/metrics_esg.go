package engine

import (
	"github.com/shopspring/decimal"
)

// computeESGAndSustainabilityMetrics calculates ESG, emissions, diversity, waste, and taxonomy metrics.
func computeESGAndSustainabilityMetrics(state *PortfolioState, metrics map[string]decimal.Decimal) {
	// 1. Controversial Weapons Zero-Tolerance Exposure
	var weaponsWeight decimal.Decimal
	for _, pos := range state.Positions {
		if pos.IsControversialWeapons {
			weaponsWeight = weaponsWeight.Add(pos.Weight)
		}
	}
	metrics["portfolio.esg_controversial_weapons_pct"] = weaponsWeight

	// 2. Thermal Coal Revenue Exposure
	var maxCoalRevenue decimal.Decimal
	for _, pos := range state.Positions {
		if pos.ThermalCoalRevenuePct.GreaterThan(maxCoalRevenue) {
			maxCoalRevenue = pos.ThermalCoalRevenuePct
		}
	}
	metrics["portfolio.esg_thermal_coal_revenue_pct"] = maxCoalRevenue

	// 3. Tobacco Revenue Exposure
	var maxTobaccoRevenue decimal.Decimal
	for _, pos := range state.Positions {
		if pos.TobaccoRevenuePct.GreaterThan(maxTobaccoRevenue) {
			maxTobaccoRevenue = pos.TobaccoRevenuePct
		}
	}
	metrics["portfolio.esg_tobacco_revenue_pct"] = maxTobaccoRevenue

	// 4-8. ESG & Sustainability Metrics (Normalized over Invested Portfolio Assets)
	var totalInvestedMV decimal.Decimal
	for _, pos := range state.Positions {
		totalInvestedMV = totalInvestedMV.Add(pos.MarketValue)
	}

	if totalInvestedMV.GreaterThan(decimal.Zero) {
		var waciSum, coveredEmissionsMV, scope12Sum, diversitySum, wasteSum, taxonomySum decimal.Decimal
		for _, pos := range state.Positions {
			invWeight := pos.MarketValue.Div(totalInvestedMV)
			if pos.HasEmissionsData || pos.WaciIntensity.GreaterThan(decimal.Zero) {
				coveredEmissionsMV = coveredEmissionsMV.Add(pos.MarketValue)
				waciSum = waciSum.Add(invWeight.Mul(pos.WaciIntensity))
			}
			if pos.GhgScope12Intensity.GreaterThan(decimal.Zero) {
				scope12Sum = scope12Sum.Add(invWeight.Mul(pos.GhgScope12Intensity))
			}
			if pos.BoardGenderDiversityPct.GreaterThan(decimal.Zero) {
				diversitySum = diversitySum.Add(invWeight.Mul(pos.BoardGenderDiversityPct))
			}
			if pos.HazardousWasteRatio.GreaterThan(decimal.Zero) {
				wasteSum = wasteSum.Add(invWeight.Mul(pos.HazardousWasteRatio))
			}
			if pos.EuTaxonomyAlignmentPct.GreaterThan(decimal.Zero) {
				taxonomySum = taxonomySum.Add(invWeight.Mul(pos.EuTaxonomyAlignmentPct))
			}
		}
		metrics["portfolio.esg_waci_tco2e_per_m_revenue"] = waciSum
		metrics["portfolio.esg_emissions_data_coverage_pct"] = coveredEmissionsMV.Div(totalInvestedMV)
		metrics["portfolio.esg_ghg_scope_1_2_intensity"] = scope12Sum
		metrics["portfolio.esg_board_gender_diversity_pct"] = diversitySum
		metrics["portfolio.esg_hazardous_waste_ratio"] = wasteSum
		metrics["portfolio.eu_taxonomy_alignment_pct"] = taxonomySum
	} else {
		metrics["portfolio.esg_waci_tco2e_per_m_revenue"] = decimal.Zero
		metrics["portfolio.esg_emissions_data_coverage_pct"] = decimal.RequireFromString("1.000000")
		metrics["portfolio.esg_ghg_scope_1_2_intensity"] = decimal.Zero
		metrics["portfolio.esg_board_gender_diversity_pct"] = decimal.RequireFromString("0.500000")
		metrics["portfolio.esg_hazardous_waste_ratio"] = decimal.Zero
		metrics["portfolio.eu_taxonomy_alignment_pct"] = decimal.RequireFromString("0.500000")
	}
}
