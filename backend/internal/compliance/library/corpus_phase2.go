package library

import "fmt"

// getPhase2PostTradeScenarios returns deterministic scenario vectors for Phase 2 Post-Trade Rules.
func getPhase2PostTradeScenarios() []Scenario {
	return []Scenario{
		// ====================================================================
		// 58. POST_TRADE_UNCLASSIFIED_CEILING_5 (Unclassified Securities Exposure Ceiling 5%)
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_UNCLASSIFIED_CEILING_5",
			Code:        "POST_TRADE_UNCLASSIFIED_CEILING_5:PASS",
			Description: "Compliant unclassified securities exposure (2% <= 5%)",
			Input: OrderContext{
				UnclassifiedSecuritiesPct: d("0.020000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_UNCLASSIFIED_CEILING_5",
			Code:        "POST_TRADE_UNCLASSIFIED_CEILING_5:BOUNDARY",
			Description: "Boundary unclassified exposure exactly at 5% ceiling",
			Input: OrderContext{
				UnclassifiedSecuritiesPct: d("0.050000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_UNCLASSIFIED_CEILING_5",
			Code:        "POST_TRADE_UNCLASSIFIED_CEILING_5:FAIL",
			Description: "Breached unclassified exposure above 5% (8.5% > 5%)",
			Input: OrderContext{
				UnclassifiedSecuritiesPct: d("0.085000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "unclassified_securities_pct"},
		},
		{
			RuleCode:    "POST_TRADE_UNCLASSIFIED_CEILING_5",
			Code:        "POST_TRADE_UNCLASSIFIED_CEILING_5:ADVERSARIAL",
			Description: "Adversarial missing security-master row hiding unclassified concentration (25% > 5% unjoined holdings)",
			Input: OrderContext{
				UnclassifiedSecuritiesPct: d("0.250000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "unclassified_securities_pct"},
		},

		// ====================================================================
		// 59. POST_TRADE_SECTOR_CONCENTRATION_25 (GICS/ICB Sector Concentration Limit 25%)
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_SECTOR_CONCENTRATION_25",
			Code:        "POST_TRADE_SECTOR_CONCENTRATION_25:PASS",
			Description: "Compliant single sector concentration (18% <= 25%)",
			Input: OrderContext{
				MaxSectorExposurePct: d("0.180000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_SECTOR_CONCENTRATION_25",
			Code:        "POST_TRADE_SECTOR_CONCENTRATION_25:BOUNDARY",
			Description: "Boundary sector exposure exactly at 25% limit",
			Input: OrderContext{
				MaxSectorExposurePct: d("0.250000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_SECTOR_CONCENTRATION_25",
			Code:        "POST_TRADE_SECTOR_CONCENTRATION_25:FAIL",
			Description: "Breached sector exposure above 25% (28.5% > 25%)",
			Input: OrderContext{
				MaxSectorExposurePct: d("0.285000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "max_sector_exposure_pct"},
		},
		{
			RuleCode:    "POST_TRADE_SECTOR_CONCENTRATION_25",
			Code:        "POST_TRADE_SECTOR_CONCENTRATION_25:ADVERSARIAL",
			Description: "Extreme sector concentration (55% > 25%)",
			Input: OrderContext{
				MaxSectorExposurePct: d("0.550000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "max_sector_exposure_pct"},
		},

		// ====================================================================
		// 60. POST_TRADE_INDUSTRY_GROUP_15 (Industry Group Concentration Limit 15%)
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_INDUSTRY_GROUP_15",
			Code:        "POST_TRADE_INDUSTRY_GROUP_15:PASS",
			Description: "Compliant industry group concentration (10% <= 15%)",
			Input: OrderContext{
				MaxIndustryGroupPct: d("0.100000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_INDUSTRY_GROUP_15",
			Code:        "POST_TRADE_INDUSTRY_GROUP_15:BOUNDARY",
			Description: "Boundary industry group exposure exactly at 15% limit",
			Input: OrderContext{
				MaxIndustryGroupPct: d("0.150000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_INDUSTRY_GROUP_15",
			Code:        "POST_TRADE_INDUSTRY_GROUP_15:FAIL",
			Description: "Breached industry group exposure above 15% (18.5% > 15%)",
			Input: OrderContext{
				MaxIndustryGroupPct: d("0.185000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "max_industry_group_pct"},
		},
		{
			RuleCode:    "POST_TRADE_INDUSTRY_GROUP_15",
			Code:        "POST_TRADE_INDUSTRY_GROUP_15:ADVERSARIAL",
			Description: "Extreme industry group concentration (40% > 15%)",
			Input: OrderContext{
				MaxIndustryGroupPct: d("0.400000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "max_industry_group_pct"},
		},

		// ====================================================================
		// 61. POST_TRADE_CYCLICAL_SECTOR_35 (Cyclical Sector Aggregate Exposure Limit 35%)
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_CYCLICAL_SECTOR_35",
			Code:        "POST_TRADE_CYCLICAL_SECTOR_35:PASS",
			Description: "Compliant cyclical sector aggregate exposure (25% <= 35%)",
			Input: OrderContext{
				CyclicalSectorsAggregatePct: d("0.250000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_CYCLICAL_SECTOR_35",
			Code:        "POST_TRADE_CYCLICAL_SECTOR_35:BOUNDARY",
			Description: "Boundary cyclical sector exposure exactly at 35% threshold",
			Input: OrderContext{
				CyclicalSectorsAggregatePct: d("0.350000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_CYCLICAL_SECTOR_35",
			Code:        "POST_TRADE_CYCLICAL_SECTOR_35:FAIL",
			Description: "Warning cyclical sector exposure above 35% (42% > 35%)",
			Input: OrderContext{
				CyclicalSectorsAggregatePct: d("0.420000"),
			},
			Expected: ExpectedOutcome{Status: "WARNING", MustContain: "cyclical_sectors_aggregate_pct"},
		},
		{
			RuleCode:    "POST_TRADE_CYCLICAL_SECTOR_35",
			Code:        "POST_TRADE_CYCLICAL_SECTOR_35:ADVERSARIAL",
			Description: "Extreme cyclical sector exposure (70% > 35%)",
			Input: OrderContext{
				CyclicalSectorsAggregatePct: d("0.700000"),
			},
			Expected: ExpectedOutcome{Status: "WARNING", MustContain: "cyclical_sectors_aggregate_pct"},
		},

		// ====================================================================
		// 62. POST_TRADE_EMERGING_MARKET_20 (Emerging Market Country Exposure Limit 20%)
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_EMERGING_MARKET_20",
			Code:        "POST_TRADE_EMERGING_MARKET_20:PASS",
			Description: "Compliant emerging market exposure (12% <= 20%)",
			Input: OrderContext{
				EmergingMarketsPct: d("0.120000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_EMERGING_MARKET_20",
			Code:        "POST_TRADE_EMERGING_MARKET_20:BOUNDARY",
			Description: "Boundary emerging market exposure exactly at 20% limit",
			Input: OrderContext{
				EmergingMarketsPct: d("0.200000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_EMERGING_MARKET_20",
			Code:        "POST_TRADE_EMERGING_MARKET_20:FAIL",
			Description: "Breached emerging market exposure above 20% (24.5% > 20%)",
			Input: OrderContext{
				EmergingMarketsPct: d("0.245000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "emerging_markets_pct"},
		},
		{
			RuleCode:    "POST_TRADE_EMERGING_MARKET_20",
			Code:        "POST_TRADE_EMERGING_MARKET_20:ADVERSARIAL",
			Description: "Extreme emerging market exposure (60% > 20%)",
			Input: OrderContext{
				EmergingMarketsPct: d("0.600000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "emerging_markets_pct"},
		},

		// ====================================================================
		// 63. POST_TRADE_NON_OECD_EXPOSURE_10 (Non-OECD Country Aggregate Exposure Ceiling 10%)
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_NON_OECD_EXPOSURE_10",
			Code:        "POST_TRADE_NON_OECD_EXPOSURE_10:PASS",
			Description: "Compliant non-OECD exposure (6% <= 10%)",
			Input: OrderContext{
				NonOecdExposurePct: d("0.060000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_NON_OECD_EXPOSURE_10",
			Code:        "POST_TRADE_NON_OECD_EXPOSURE_10:BOUNDARY",
			Description: "Boundary non-OECD exposure exactly at 10% ceiling",
			Input: OrderContext{
				NonOecdExposurePct: d("0.100000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_NON_OECD_EXPOSURE_10",
			Code:        "POST_TRADE_NON_OECD_EXPOSURE_10:FAIL",
			Description: "Breached non-OECD exposure above 10% (13.5% > 10%)",
			Input: OrderContext{
				NonOecdExposurePct: d("0.135000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "non_oecd_exposure_pct"},
		},
		{
			RuleCode:    "POST_TRADE_NON_OECD_EXPOSURE_10",
			Code:        "POST_TRADE_NON_OECD_EXPOSURE_10:ADVERSARIAL",
			Description: "Extreme non-OECD exposure (45% > 10%)",
			Input: OrderContext{
				NonOecdExposurePct: d("0.450000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "non_oecd_exposure_pct"},
		},

		// ====================================================================
		// 64. POST_TRADE_FRONTIER_MARKET_5 (Frontier Market Country Sub-Ceiling 5%)
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_FRONTIER_MARKET_5",
			Code:        "POST_TRADE_FRONTIER_MARKET_5:PASS",
			Description: "Compliant frontier market exposure (2.5% <= 5%)",
			Input: OrderContext{
				FrontierMarketsPct: d("0.025000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_FRONTIER_MARKET_5",
			Code:        "POST_TRADE_FRONTIER_MARKET_5:BOUNDARY",
			Description: "Boundary frontier market exposure exactly at 5% sub-ceiling",
			Input: OrderContext{
				FrontierMarketsPct: d("0.050000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_FRONTIER_MARKET_5",
			Code:        "POST_TRADE_FRONTIER_MARKET_5:FAIL",
			Description: "Breached frontier market exposure above 5% (7.5% > 5%)",
			Input: OrderContext{
				FrontierMarketsPct: d("0.075000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "frontier_markets_pct"},
		},
		{
			RuleCode:    "POST_TRADE_FRONTIER_MARKET_5",
			Code:        "POST_TRADE_FRONTIER_MARKET_5:ADVERSARIAL",
			Description: "Extreme frontier market exposure (30% > 5%)",
			Input: OrderContext{
				FrontierMarketsPct: d("0.300000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "frontier_markets_pct"},
		},

		// ====================================================================
		// 65. POST_TRADE_SANCTION_LIST_ZERO_TOLERANCE
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_SANCTION_LIST_ZERO_TOLERANCE",
			Code:        "POST_TRADE_SANCTION_LIST_ZERO_TOLERANCE:PASS",
			Description: "Compliant portfolio with zero sanctioned entity holdings",
			Input: OrderContext{
				SanctionedEntityMatchesCount: 0,
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_SANCTION_LIST_ZERO_TOLERANCE",
			Code:        "POST_TRADE_SANCTION_LIST_ZERO_TOLERANCE:BOUNDARY",
			Description: "Boundary portfolio with exactly zero sanctioned entity matches",
			Input: OrderContext{
				SanctionedEntityMatchesCount: 0,
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_SANCTION_LIST_ZERO_TOLERANCE",
			Code:        "POST_TRADE_SANCTION_LIST_ZERO_TOLERANCE:FAIL",
			Description: "Direct match of OFAC SDN sanctioned entity holding",
			Input: OrderContext{
				SanctionedEntityMatchesCount: 1,
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "sanctioned_entity_matches_count"},
		},
		{
			RuleCode:    "POST_TRADE_SANCTION_LIST_ZERO_TOLERANCE",
			Code:        "POST_TRADE_SANCTION_LIST_ZERO_TOLERANCE:ADVERSARIAL",
			Description: "Adversarial fuzzy match / 50% rule subsidiary of designated entity",
			Input: OrderContext{
				SanctionedEntityMatchesCount: 3,
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "sanctioned_entity_matches_count"},
		},

		// ====================================================================
		// 66. POST_TRADE_FX_FORWARD_UNHEDGED_30
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_FX_FORWARD_UNHEDGED_30",
			Code:        "POST_TRADE_FX_FORWARD_UNHEDGED_30:PASS",
			Description: "Compliant unhedged currency exposure (20% <= 30%)",
			Input: OrderContext{
				UnhedgedFxExposurePct: d("0.200000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_FX_FORWARD_UNHEDGED_30",
			Code:        "POST_TRADE_FX_FORWARD_UNHEDGED_30:BOUNDARY",
			Description: "Boundary unhedged FX exposure exactly at 30% ceiling",
			Input: OrderContext{
				UnhedgedFxExposurePct: d("0.300000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_FX_FORWARD_UNHEDGED_30",
			Code:        "POST_TRADE_FX_FORWARD_UNHEDGED_30:FAIL",
			Description: "Breached unhedged currency exposure above 30% (35% > 30%)",
			Input: OrderContext{
				UnhedgedFxExposurePct: d("0.350000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "unhedged_fx_exposure_pct"},
		},
		{
			RuleCode:    "POST_TRADE_FX_FORWARD_UNHEDGED_30",
			Code:        "POST_TRADE_FX_FORWARD_UNHEDGED_30:ADVERSARIAL",
			Description: "Extreme unhedged foreign exchange exposure (65% > 30%)",
			Input: OrderContext{
				UnhedgedFxExposurePct: d("0.650000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "unhedged_fx_exposure_pct"},
		},

		// ====================================================================
		// 67. POST_TRADE_HIGH_YIELD_CEILING_10
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_HIGH_YIELD_CEILING_10",
			Code:        "POST_TRADE_HIGH_YIELD_CEILING_10:PASS",
			Description: "Compliant high-yield debt holding (6% <= 10%)",
			Input: OrderContext{
				HighYieldDebtExposurePct: d("0.060000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_HIGH_YIELD_CEILING_10",
			Code:        "POST_TRADE_HIGH_YIELD_CEILING_10:BOUNDARY",
			Description: "Boundary high-yield debt holding exactly at 10% ceiling",
			Input: OrderContext{
				HighYieldDebtExposurePct: d("0.100000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_HIGH_YIELD_CEILING_10",
			Code:        "POST_TRADE_HIGH_YIELD_CEILING_10:FAIL",
			Description: "Breached high-yield exposure above 10% (14% > 10%)",
			Input: OrderContext{
				HighYieldDebtExposurePct: d("0.140000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "high_yield_debt_exposure_pct"},
		},
		{
			RuleCode:    "POST_TRADE_HIGH_YIELD_CEILING_10",
			Code:        "POST_TRADE_HIGH_YIELD_CEILING_10:ADVERSARIAL",
			Description: "Extreme high-yield debt concentration (35% > 10%)",
			Input: OrderContext{
				HighYieldDebtExposurePct: d("0.350000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "high_yield_debt_exposure_pct"},
		},

		// ====================================================================
		// 68. POST_TRADE_SPLIT_RATING_CONSERVATIVE_FLOOR (Conservative Rating Floor BBB-/Baa3, Max Rank 10)
		// Ordinal Scale: 1=AAA ... 10=BBB- (Lowest IG) ... 11=BB+ (Sub-IG) ... 22=D
		// Concrete Cases:
		// (a) PASS: S&P AA (3) / Moody's Baa2 (9) -> Conservative worst rank is 9 <= 10 (IG PASS)
		// (b) FAIL: S&P BBB- (10) / Moody's BB+ (11) -> Conservative worst rank is 11 > 10 (Sub-IG FAIL)
		// (c) BOUNDARY: S&P BBB- (10) / Moody's BBB- (10) -> Conservative worst rank is exactly 10 <= 10 (BOUNDARY PASS)
		// (d) ADVERSARIAL: S&P BBB- (10) / Fitch BB+ (11) / Moody's unrated -> Conservative worst rank is 11 > 10 (ADVERSARIAL FAIL)
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_SPLIT_RATING_CONSERVATIVE_FLOOR",
			Code:        "POST_TRADE_SPLIT_RATING_CONSERVATIVE_FLOOR:PASS",
			Description: "Compliant split rating: S&P AA (3) / Moody's Baa2 (9) -> conservative worst rank is 9 <= 10 (IG)",
			Input: OrderContext{
				SplitRatingWorstGradeRank: 9,
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_SPLIT_RATING_CONSERVATIVE_FLOOR",
			Code:        "POST_TRADE_SPLIT_RATING_CONSERVATIVE_FLOOR:BOUNDARY",
			Description: "Boundary split rating: S&P BBB- (10) / Moody's BBB- (10) -> conservative worst rank is exactly 10 <= 10",
			Input: OrderContext{
				SplitRatingWorstGradeRank: 10,
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_SPLIT_RATING_CONSERVATIVE_FLOOR",
			Code:        "POST_TRADE_SPLIT_RATING_CONSERVATIVE_FLOOR:FAIL",
			Description: "Breached split rating: S&P BBB- (10) / Moody's BB+ (11) -> conservative worst rank is 11 > 10 (sub-IG)",
			Input: OrderContext{
				SplitRatingWorstGradeRank: 11,
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "split_rating_worst_grade_rank"},
		},
		{
			RuleCode:    "POST_TRADE_SPLIT_RATING_CONSERVATIVE_FLOOR",
			Code:        "POST_TRADE_SPLIT_RATING_CONSERVATIVE_FLOOR:ADVERSARIAL",
			Description: "Adversarial notch-straddling: S&P BBB- (10) / Fitch BB+ (11) / Moody's unrated -> conservative worst rank is 11 > 10",
			Input: OrderContext{
				SplitRatingWorstGradeRank: 11,
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "split_rating_worst_grade_rank"},
		},

		// ====================================================================
		// 69. POST_TRADE_ESG_CONTROVERSIAL_WEAPONS_0
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_ESG_CONTROVERSIAL_WEAPONS_0",
			Code:        "POST_TRADE_ESG_CONTROVERSIAL_WEAPONS_0:PASS",
			Description: "Compliant portfolio with zero controversial weapons holdings",
			Input: OrderContext{
				EsgControversialWeaponsPct: d("0.000000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_ESG_CONTROVERSIAL_WEAPONS_0",
			Code:        "POST_TRADE_ESG_CONTROVERSIAL_WEAPONS_0:BOUNDARY",
			Description: "Boundary portfolio with exactly zero controversial weapons exposure",
			Input: OrderContext{
				EsgControversialWeaponsPct: d("0.000000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_ESG_CONTROVERSIAL_WEAPONS_0",
			Code:        "POST_TRADE_ESG_CONTROVERSIAL_WEAPONS_0:FAIL",
			Description: "Breached controversial weapons holding (0.5% > 0%)",
			Input: OrderContext{
				EsgControversialWeaponsPct: d("0.005000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "esg_controversial_weapons_pct"},
		},
		{
			RuleCode:    "POST_TRADE_ESG_CONTROVERSIAL_WEAPONS_0",
			Code:        "POST_TRADE_ESG_CONTROVERSIAL_WEAPONS_0:ADVERSARIAL",
			Description: "Direct holding in cluster munitions component manufacturer (5% > 0%)",
			Input: OrderContext{
				EsgControversialWeaponsPct: d("0.050000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "esg_controversial_weapons_pct"},
		},

		// ====================================================================
		// 70. POST_TRADE_ESG_THERMAL_COAL_REVENUE_5
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_ESG_THERMAL_COAL_REVENUE_5",
			Code:        "POST_TRADE_ESG_THERMAL_COAL_REVENUE_5:PASS",
			Description: "Compliant thermal coal revenue exposure (2% <= 5%)",
			Input: OrderContext{
				EsgThermalCoalRevenuePct: d("0.020000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_ESG_THERMAL_COAL_REVENUE_5",
			Code:        "POST_TRADE_ESG_THERMAL_COAL_REVENUE_5:BOUNDARY",
			Description: "Boundary thermal coal revenue exactly at 5% threshold",
			Input: OrderContext{
				EsgThermalCoalRevenuePct: d("0.050000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_ESG_THERMAL_COAL_REVENUE_5",
			Code:        "POST_TRADE_ESG_THERMAL_COAL_REVENUE_5:FAIL",
			Description: "Breached thermal coal revenue above 5% threshold (8% > 5%)",
			Input: OrderContext{
				EsgThermalCoalRevenuePct: d("0.080000"),
			},
			Expected: ExpectedOutcome{Status: "WARNING", MustContain: "esg_thermal_coal_revenue_pct"},
		},
		{
			RuleCode:    "POST_TRADE_ESG_THERMAL_COAL_REVENUE_5",
			Code:        "POST_TRADE_ESG_THERMAL_COAL_REVENUE_5:ADVERSARIAL",
			Description: "High thermal coal revenue exposure (45% > 5%)",
			Input: OrderContext{
				EsgThermalCoalRevenuePct: d("0.450000"),
			},
			Expected: ExpectedOutcome{Status: "WARNING", MustContain: "esg_thermal_coal_revenue_pct"},
		},

		// ====================================================================
		// 71. POST_TRADE_ESG_TOBACCO_REVENUE_5
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_ESG_TOBACCO_REVENUE_5",
			Code:        "POST_TRADE_ESG_TOBACCO_REVENUE_5:PASS",
			Description: "Compliant tobacco revenue exposure (1% <= 5%)",
			Input: OrderContext{
				EsgTobaccoRevenuePct: d("0.010000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_ESG_TOBACCO_REVENUE_5",
			Code:        "POST_TRADE_ESG_TOBACCO_REVENUE_5:BOUNDARY",
			Description: "Boundary tobacco revenue exactly at 5% threshold",
			Input: OrderContext{
				EsgTobaccoRevenuePct: d("0.050000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_ESG_TOBACCO_REVENUE_5",
			Code:        "POST_TRADE_ESG_TOBACCO_REVENUE_5:FAIL",
			Description: "Breached tobacco revenue above 5% threshold (9% > 5%)",
			Input: OrderContext{
				EsgTobaccoRevenuePct: d("0.090000"),
			},
			Expected: ExpectedOutcome{Status: "WARNING", MustContain: "esg_tobacco_revenue_pct"},
		},
		{
			RuleCode:    "POST_TRADE_ESG_TOBACCO_REVENUE_5",
			Code:        "POST_TRADE_ESG_TOBACCO_REVENUE_5:ADVERSARIAL",
			Description: "High tobacco manufacturing revenue exposure (30% > 5%)",
			Input: OrderContext{
				EsgTobaccoRevenuePct: d("0.300000"),
			},
			Expected: ExpectedOutcome{Status: "WARNING", MustContain: "esg_tobacco_revenue_pct"},
		},

		// ====================================================================
		// 72. POST_TRADE_ILLIQUID_TIER3_ASSETS_10
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_ILLIQUID_TIER3_ASSETS_10",
			Code:        "POST_TRADE_ILLIQUID_TIER3_ASSETS_10:PASS",
			Description: "Compliant Level 3 illiquid assets exposure (4% <= 10%)",
			Input: OrderContext{
				IlliquidLevel3AssetsPct: d("0.040000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_ILLIQUID_TIER3_ASSETS_10",
			Code:        "POST_TRADE_ILLIQUID_TIER3_ASSETS_10:BOUNDARY",
			Description: "Boundary Level 3 illiquid assets exactly at 10% ceiling",
			Input: OrderContext{
				IlliquidLevel3AssetsPct: d("0.100000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_ILLIQUID_TIER3_ASSETS_10",
			Code:        "POST_TRADE_ILLIQUID_TIER3_ASSETS_10:FAIL",
			Description: "Breached Level 3 illiquid assets above 10% (13% > 10%)",
			Input: OrderContext{
				IlliquidLevel3AssetsPct: d("0.130000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "illiquid_level3_assets_pct"},
		},
		{
			RuleCode:    "POST_TRADE_ILLIQUID_TIER3_ASSETS_10",
			Code:        "POST_TRADE_ILLIQUID_TIER3_ASSETS_10:ADVERSARIAL",
			Description: "Extreme Level 3 illiquid assets concentration (40% > 10%)",
			Input: OrderContext{
				IlliquidLevel3AssetsPct: d("0.400000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "illiquid_level3_assets_pct"},
		},

		// ====================================================================
		// 73. POST_TRADE_SETTLEMENT_FAIL_CONCENTRATION_5
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_SETTLEMENT_FAIL_CONCENTRATION_5",
			Code:        "POST_TRADE_SETTLEMENT_FAIL_CONCENTRATION_5:PASS",
			Description: "Compliant settlement failure exposure (1% <= 5%)",
			Input: OrderContext{
				SettlementFailExposurePct: d("0.010000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_SETTLEMENT_FAIL_CONCENTRATION_5",
			Code:        "POST_TRADE_SETTLEMENT_FAIL_CONCENTRATION_5:BOUNDARY",
			Description: "Boundary settlement failure exposure exactly at 5% threshold",
			Input: OrderContext{
				SettlementFailExposurePct: d("0.050000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_SETTLEMENT_FAIL_CONCENTRATION_5",
			Code:        "POST_TRADE_SETTLEMENT_FAIL_CONCENTRATION_5:FAIL",
			Description: "Breached settlement failure exposure above 5% threshold (8% > 5%)",
			Input: OrderContext{
				SettlementFailExposurePct: d("0.080000"),
			},
			Expected: ExpectedOutcome{Status: "WARNING", MustContain: "settlement_fail_exposure_pct"},
		},
		{
			RuleCode:    "POST_TRADE_SETTLEMENT_FAIL_CONCENTRATION_5",
			Code:        "POST_TRADE_SETTLEMENT_FAIL_CONCENTRATION_5:ADVERSARIAL",
			Description: "High settlement failure exposure (25% > 5%)",
			Input: OrderContext{
				SettlementFailExposurePct: d("0.250000"),
			},
			Expected: ExpectedOutcome{Status: "WARNING", MustContain: "settlement_fail_exposure_pct"},
		},

		// ====================================================================
		// 74. POST_TRADE_ESG_WACI_PORTFOLIO_CEILING (WACI 150 tCO2e/M$, 75% Coverage Floor)
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_ESG_WACI_PORTFOLIO_CEILING",
			Code:        "POST_TRADE_ESG_WACI_PORTFOLIO_CEILING:PASS",
			Description: "Compliant WACI intensity (120 tCO2e/M$ <= 150) with sufficient data coverage (85% >= 75%)",
			Input: OrderContext{
				EsgWaciTco2ePerMRevenue:     d("120.000000"),
				EsgEmissionsDataCoveragePct: d("0.850000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_ESG_WACI_PORTFOLIO_CEILING",
			Code:        "POST_TRADE_ESG_WACI_PORTFOLIO_CEILING:BOUNDARY",
			Description: "Boundary WACI intensity (150 tCO2e/M$) with exact 75% coverage floor",
			Input: OrderContext{
				EsgWaciTco2ePerMRevenue:     d("150.000000"),
				EsgEmissionsDataCoveragePct: d("0.750000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_ESG_WACI_PORTFOLIO_CEILING",
			Code:        "POST_TRADE_ESG_WACI_PORTFOLIO_CEILING:FAIL",
			Description: "Breached WACI intensity (185.5 tCO2e/M$ > 150) with 80% data coverage",
			Input: OrderContext{
				EsgWaciTco2ePerMRevenue:     d("185.500000"),
				EsgEmissionsDataCoveragePct: d("0.800000"),
			},
			Expected: ExpectedOutcome{Status: "WARNING", MustContain: "esg_waci_tco2e_per_m_revenue"},
		},
		{
			RuleCode:    "POST_TRADE_ESG_WACI_PORTFOLIO_CEILING",
			Code:        "POST_TRADE_ESG_WACI_PORTFOLIO_CEILING:ADVERSARIAL",
			Description: "Adversarial low data coverage probe: WACI 110 tCO2e/M$ understated due to 60% coverage (< 75% floor)",
			Input: OrderContext{
				EsgWaciTco2ePerMRevenue:     d("110.000000"),
				EsgEmissionsDataCoveragePct: d("0.600000"),
			},
			Expected: ExpectedOutcome{Status: "WARNING", MustContain: "esg_emissions_data_coverage_pct"},
		},

		// ====================================================================
		// 75. POST_TRADE_ESG_SCOPE_1_2_EMISSIONS_CEILING (Scope 1+2 100 tCO2e/M$)
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_ESG_SCOPE_1_2_EMISSIONS_CEILING",
			Code:        "POST_TRADE_ESG_SCOPE_1_2_EMISSIONS_CEILING:PASS",
			Description: "Compliant Scope 1+2 GHG emissions intensity (75 tCO2e/M$ <= 100)",
			Input: OrderContext{
				EsgGhgScope12Intensity: d("75.000000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_ESG_SCOPE_1_2_EMISSIONS_CEILING",
			Code:        "POST_TRADE_ESG_SCOPE_1_2_EMISSIONS_CEILING:BOUNDARY",
			Description: "Boundary Scope 1+2 emissions exactly at 100 tCO2e/M$ ceiling",
			Input: OrderContext{
				EsgGhgScope12Intensity: d("100.000000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_ESG_SCOPE_1_2_EMISSIONS_CEILING",
			Code:        "POST_TRADE_ESG_SCOPE_1_2_EMISSIONS_CEILING:FAIL",
			Description: "Breached Scope 1+2 emissions intensity (135 tCO2e/M$ > 100)",
			Input: OrderContext{
				EsgGhgScope12Intensity: d("135.000000"),
			},
			Expected: ExpectedOutcome{Status: "WARNING", MustContain: "esg_ghg_scope_1_2_intensity"},
		},
		{
			RuleCode:    "POST_TRADE_ESG_SCOPE_1_2_EMISSIONS_CEILING",
			Code:        "POST_TRADE_ESG_SCOPE_1_2_EMISSIONS_CEILING:ADVERSARIAL",
			Description: "Extreme Scope 1+2 emissions intensity (280 tCO2e/M$ > 100)",
			Input: OrderContext{
				EsgGhgScope12Intensity: d("280.000000"),
			},
			Expected: ExpectedOutcome{Status: "WARNING", MustContain: "esg_ghg_scope_1_2_intensity"},
		},

		// ====================================================================
		// 76. POST_TRADE_ESG_BOARD_GENDER_DIVERSITY_FLOOR (Female Board Representation Floor 30%)
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_ESG_BOARD_GENDER_DIVERSITY_FLOOR",
			Code:        "POST_TRADE_ESG_BOARD_GENDER_DIVERSITY_FLOOR:PASS",
			Description: "Compliant board gender diversity (38% female representation >= 30%)",
			Input: OrderContext{
				EsgBoardGenderDiversityPct: d("0.380000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_ESG_BOARD_GENDER_DIVERSITY_FLOOR",
			Code:        "POST_TRADE_ESG_BOARD_GENDER_DIVERSITY_FLOOR:BOUNDARY",
			Description: "Boundary board gender diversity exactly at 30% floor",
			Input: OrderContext{
				EsgBoardGenderDiversityPct: d("0.300000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_ESG_BOARD_GENDER_DIVERSITY_FLOOR",
			Code:        "POST_TRADE_ESG_BOARD_GENDER_DIVERSITY_FLOOR:FAIL",
			Description: "Breached board gender diversity below 30% floor (22% < 30%)",
			Input: OrderContext{
				EsgBoardGenderDiversityPct: d("0.220000"),
			},
			Expected: ExpectedOutcome{Status: "WARNING", MustContain: "esg_board_gender_diversity_pct"},
		},
		{
			RuleCode:    "POST_TRADE_ESG_BOARD_GENDER_DIVERSITY_FLOOR",
			Code:        "POST_TRADE_ESG_BOARD_GENDER_DIVERSITY_FLOOR:ADVERSARIAL",
			Description: "Adversarial 0% female board representation (all-male board 0% < 30%)",
			Input: OrderContext{
				EsgBoardGenderDiversityPct: d("0.000000"),
			},
			Expected: ExpectedOutcome{Status: "WARNING", MustContain: "esg_board_gender_diversity_pct"},
		},

		// ====================================================================
		// 77. POST_TRADE_ESG_HAZARDOUS_WASTE_RATIO_CEILING (Hazardous Waste 5.0 Tonnes/M$)
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_ESG_HAZARDOUS_WASTE_RATIO_CEILING",
			Code:        "POST_TRADE_ESG_HAZARDOUS_WASTE_RATIO_CEILING:PASS",
			Description: "Compliant hazardous waste ratio (2.5 tonnes/M$ <= 5.0)",
			Input: OrderContext{
				EsgHazardousWasteRatio: d("2.500000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_ESG_HAZARDOUS_WASTE_RATIO_CEILING",
			Code:        "POST_TRADE_ESG_HAZARDOUS_WASTE_RATIO_CEILING:BOUNDARY",
			Description: "Boundary hazardous waste ratio exactly at 5.0 tonnes/M$ ceiling",
			Input: OrderContext{
				EsgHazardousWasteRatio: d("5.000000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_ESG_HAZARDOUS_WASTE_RATIO_CEILING",
			Code:        "POST_TRADE_ESG_HAZARDOUS_WASTE_RATIO_CEILING:FAIL",
			Description: "Breached hazardous waste ratio above 5.0 tonnes/M$ (8.4 > 5.0)",
			Input: OrderContext{
				EsgHazardousWasteRatio: d("8.400000"),
			},
			Expected: ExpectedOutcome{Status: "WARNING", MustContain: "esg_hazardous_waste_ratio"},
		},
		{
			RuleCode:    "POST_TRADE_ESG_HAZARDOUS_WASTE_RATIO_CEILING",
			Code:        "POST_TRADE_ESG_HAZARDOUS_WASTE_RATIO_CEILING:ADVERSARIAL",
			Description: "Extreme hazardous waste ratio exposure (35.0 tonnes/M$ > 5.0)",
			Input: OrderContext{
				EsgHazardousWasteRatio: d("35.000000"),
			},
			Expected: ExpectedOutcome{Status: "WARNING", MustContain: "esg_hazardous_waste_ratio"},
		},

		// ====================================================================
		// 78. POST_TRADE_EU_TAXONOMY_GREEN_REVENUE_FLOOR (EU Taxonomy Green Revenue Floor 15%)
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_EU_TAXONOMY_GREEN_REVENUE_FLOOR",
			Code:        "POST_TRADE_EU_TAXONOMY_GREEN_REVENUE_FLOOR:PASS",
			Description: "Compliant EU Taxonomy green revenue alignment (22% >= 15%)",
			Input: OrderContext{
				EuTaxonomyAlignmentPct: d("0.220000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_EU_TAXONOMY_GREEN_REVENUE_FLOOR",
			Code:        "POST_TRADE_EU_TAXONOMY_GREEN_REVENUE_FLOOR:BOUNDARY",
			Description: "Boundary EU Taxonomy green revenue alignment exactly at 15% floor",
			Input: OrderContext{
				EuTaxonomyAlignmentPct: d("0.150000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_EU_TAXONOMY_GREEN_REVENUE_FLOOR",
			Code:        "POST_TRADE_EU_TAXONOMY_GREEN_REVENUE_FLOOR:FAIL",
			Description: "Breached EU Taxonomy green revenue alignment below 15% (8% < 15%)",
			Input: OrderContext{
				EuTaxonomyAlignmentPct: d("0.080000"),
			},
			Expected: ExpectedOutcome{Status: "WARNING", MustContain: "eu_taxonomy_alignment_pct"},
		},
		{
			RuleCode:    "POST_TRADE_EU_TAXONOMY_GREEN_REVENUE_FLOOR",
			Code:        "POST_TRADE_EU_TAXONOMY_GREEN_REVENUE_FLOOR:ADVERSARIAL",
			Description: "Adversarial 0% green revenue alignment in SFDR Article 9 portfolio (0% < 15%)",
			Input: OrderContext{
				EuTaxonomyAlignmentPct: d("0.000000"),
			},
			Expected: ExpectedOutcome{Status: "WARNING", MustContain: "eu_taxonomy_alignment_pct"},
		},

		// ====================================================================
		// 79. POST_TRADE_LIQUIDITY_COVERAGE_RATIO_BUFFER (Stress LCR Minimum 105%)
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_LIQUIDITY_COVERAGE_RATIO_BUFFER",
			Code:        "POST_TRADE_LIQUIDITY_COVERAGE_RATIO_BUFFER:PASS",
			Description: "Compliant stress Liquidity Coverage Ratio buffer (125% >= 105%)",
			Input: OrderContext{
				LiquidityCoverageRatio: d("1.250000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_LIQUIDITY_COVERAGE_RATIO_BUFFER",
			Code:        "POST_TRADE_LIQUIDITY_COVERAGE_RATIO_BUFFER:BOUNDARY",
			Description: "Boundary stress Liquidity Coverage Ratio buffer exactly at 105% floor",
			Input: OrderContext{
				LiquidityCoverageRatio: d("1.050000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_LIQUIDITY_COVERAGE_RATIO_BUFFER",
			Code:        "POST_TRADE_LIQUIDITY_COVERAGE_RATIO_BUFFER:FAIL",
			Description: "Breached stress Liquidity Coverage Ratio buffer (95% < 105%)",
			Input: OrderContext{
				LiquidityCoverageRatio: d("0.950000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "liquidity_coverage_ratio"},
		},
		{
			RuleCode:    "POST_TRADE_LIQUIDITY_COVERAGE_RATIO_BUFFER",
			Code:        "POST_TRADE_LIQUIDITY_COVERAGE_RATIO_BUFFER:ADVERSARIAL",
			Description: "Severe stress Liquidity Coverage Ratio deficit under liquidity run (60% < 105%)",
			Input: OrderContext{
				LiquidityCoverageRatio: d("0.600000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "liquidity_coverage_ratio"},
		},
	}
}

func evaluatePhase2Scenario(sc Scenario) (string, string, bool) {
	in := sc.Input
	switch sc.RuleCode {
	case "POST_TRADE_UNCLASSIFIED_CEILING_5":
		limit := d("0.050000")
		if in.UnclassifiedSecuritiesPct.GreaterThan(limit) {
			return "BLOCKED", fmt.Sprintf("Hard Block: unclassified_securities_pct %s exceeds ceiling %s", in.UnclassifiedSecuritiesPct.StringFixed(4), limit.StringFixed(4)), true
		}
		return "PASSED", "Compliant: unclassified securities exposure within 5% ceiling", true

	case "POST_TRADE_SECTOR_CONCENTRATION_25":
		limit := d("0.250000")
		if in.MaxSectorExposurePct.GreaterThan(limit) {
			return "BLOCKED", fmt.Sprintf("Hard Block: max_sector_exposure_pct %s exceeds limit %s", in.MaxSectorExposurePct.StringFixed(4), limit.StringFixed(4)), true
		}
		return "PASSED", "Compliant: single sector concentration within 25% limit", true

	case "POST_TRADE_INDUSTRY_GROUP_15":
		limit := d("0.150000")
		if in.MaxIndustryGroupPct.GreaterThan(limit) {
			return "BLOCKED", fmt.Sprintf("Hard Block: max_industry_group_pct %s exceeds limit %s", in.MaxIndustryGroupPct.StringFixed(4), limit.StringFixed(4)), true
		}
		return "PASSED", "Compliant: industry group concentration within 15% limit", true

	case "POST_TRADE_CYCLICAL_SECTOR_35":
		limit := d("0.350000")
		if in.CyclicalSectorsAggregatePct.GreaterThan(limit) {
			return "WARNING", fmt.Sprintf("Soft Warning: cyclical_sectors_aggregate_pct %s exceeds warning threshold %s", in.CyclicalSectorsAggregatePct.StringFixed(4), limit.StringFixed(4)), true
		}
		return "PASSED", "Compliant: cyclical sector aggregate exposure within 35% threshold", true

	case "POST_TRADE_EMERGING_MARKET_20":
		limit := d("0.200000")
		if in.EmergingMarketsPct.GreaterThan(limit) {
			return "BLOCKED", fmt.Sprintf("Hard Block: emerging_markets_pct %s exceeds limit %s", in.EmergingMarketsPct.StringFixed(4), limit.StringFixed(4)), true
		}
		return "PASSED", "Compliant: emerging market country exposure within 20% limit", true

	case "POST_TRADE_NON_OECD_EXPOSURE_10":
		limit := d("0.100000")
		if in.NonOecdExposurePct.GreaterThan(limit) {
			return "BLOCKED", fmt.Sprintf("Hard Block: non_oecd_exposure_pct %s exceeds ceiling %s", in.NonOecdExposurePct.StringFixed(4), limit.StringFixed(4)), true
		}
		return "PASSED", "Compliant: non-OECD country exposure within 10% ceiling", true

	case "POST_TRADE_FRONTIER_MARKET_5":
		limit := d("0.050000")
		if in.FrontierMarketsPct.GreaterThan(limit) {
			return "BLOCKED", fmt.Sprintf("Hard Block: frontier_markets_pct %s exceeds sub-ceiling %s", in.FrontierMarketsPct.StringFixed(4), limit.StringFixed(4)), true
		}
		return "PASSED", "Compliant: frontier market country exposure within 5% sub-ceiling", true

	case "POST_TRADE_SANCTION_LIST_ZERO_TOLERANCE":
		if in.SanctionedEntityMatchesCount > 0 {
			return "BLOCKED", fmt.Sprintf("Hard Block: sanctioned_entity_matches_count %d exceeds 0", in.SanctionedEntityMatchesCount), true
		}
		return "PASSED", "Compliant: zero sanctioned entity matches", true

	case "POST_TRADE_FX_FORWARD_UNHEDGED_30":
		limit := d("0.300000")
		if in.UnhedgedFxExposurePct.GreaterThan(limit) {
			return "BLOCKED", fmt.Sprintf("Hard Block: unhedged_fx_exposure_pct %s exceeds limit %s", in.UnhedgedFxExposurePct.StringFixed(4), limit.StringFixed(4)), true
		}
		return "PASSED", "Compliant: unhedged FX exposure within 30% limit", true

	case "POST_TRADE_HIGH_YIELD_CEILING_10":
		limit := d("0.100000")
		if in.HighYieldDebtExposurePct.GreaterThan(limit) {
			return "BLOCKED", fmt.Sprintf("Hard Block: high_yield_debt_exposure_pct %s exceeds ceiling %s", in.HighYieldDebtExposurePct.StringFixed(4), limit.StringFixed(4)), true
		}
		return "PASSED", "Compliant: high-yield debt exposure within 10% ceiling", true

	case "POST_TRADE_SPLIT_RATING_CONSERVATIVE_FLOOR":
		maxRank := 10 // BBB-/Baa3 floor (Rank 1=AAA ... 10=BBB- ... 22=D)
		rank := in.SplitRatingWorstGradeRank
		if rank == 0 {
			rank = in.SplitRatingConservativeGradeRank
		}
		if rank > maxRank {
			return "BLOCKED", fmt.Sprintf("Hard Block: split_rating_worst_grade_rank %d exceeds maximum grade threshold %d (sub-investment grade breach)", rank, maxRank), true
		}
		return "PASSED", "Compliant: split credit rating satisfies conservative floor rank (investment grade)", true

	case "POST_TRADE_ESG_CONTROVERSIAL_WEAPONS_0":
		limit := d("0.000000")
		if in.EsgControversialWeaponsPct.GreaterThan(limit) {
			return "BLOCKED", fmt.Sprintf("Hard Block: esg_controversial_weapons_pct %s exceeds zero tolerance limit", in.EsgControversialWeaponsPct.StringFixed(4)), true
		}
		return "PASSED", "Compliant: zero controversial weapons exposure", true

	case "POST_TRADE_ESG_THERMAL_COAL_REVENUE_5":
		limit := d("0.050000")
		if in.EsgThermalCoalRevenuePct.GreaterThan(limit) {
			return "WARNING", fmt.Sprintf("Soft Warning: esg_thermal_coal_revenue_pct %s exceeds threshold %s", in.EsgThermalCoalRevenuePct.StringFixed(4), limit.StringFixed(4)), true
		}
		return "PASSED", "Compliant: thermal coal revenue within 5% threshold", true

	case "POST_TRADE_ESG_TOBACCO_REVENUE_5":
		limit := d("0.050000")
		if in.EsgTobaccoRevenuePct.GreaterThan(limit) {
			return "WARNING", fmt.Sprintf("Soft Warning: esg_tobacco_revenue_pct %s exceeds threshold %s", in.EsgTobaccoRevenuePct.StringFixed(4), limit.StringFixed(4)), true
		}
		return "PASSED", "Compliant: tobacco revenue within 5% threshold", true

	case "POST_TRADE_ILLIQUID_TIER3_ASSETS_10":
		limit := d("0.100000")
		if in.IlliquidLevel3AssetsPct.GreaterThan(limit) {
			return "BLOCKED", fmt.Sprintf("Hard Block: illiquid_level3_assets_pct %s exceeds ceiling %s", in.IlliquidLevel3AssetsPct.StringFixed(4), limit.StringFixed(4)), true
		}
		return "PASSED", "Compliant: Level 3 illiquid assets within 10% ceiling", true

	case "POST_TRADE_SETTLEMENT_FAIL_CONCENTRATION_5":
		limit := d("0.050000")
		if in.SettlementFailExposurePct.GreaterThan(limit) {
			return "WARNING", fmt.Sprintf("Soft Warning: settlement_fail_exposure_pct %s exceeds threshold %s", in.SettlementFailExposurePct.StringFixed(4), limit.StringFixed(4)), true
		}
		return "PASSED", "Compliant: settlement failure exposure within 5% threshold", true

	case "POST_TRADE_ESG_WACI_PORTFOLIO_CEILING":
		waciLimit := d("150.000000")
		covFloor := d("0.750000")
		if in.EsgEmissionsDataCoveragePct.LessThan(covFloor) {
			return "WARNING", fmt.Sprintf("Soft Warning: esg_emissions_data_coverage_pct %s is below mandatory threshold %s (understates portfolio WACI)", in.EsgEmissionsDataCoveragePct.StringFixed(4), covFloor.StringFixed(4)), true
		}
		if in.EsgWaciTco2ePerMRevenue.GreaterThan(waciLimit) {
			return "WARNING", fmt.Sprintf("Soft Warning: esg_waci_tco2e_per_m_revenue %s exceeds carbon ceiling %s", in.EsgWaciTco2ePerMRevenue.StringFixed(2), waciLimit.StringFixed(2)), true
		}
		return "PASSED", "Compliant: WACI within portfolio carbon ceiling", true

	case "POST_TRADE_ESG_SCOPE_1_2_EMISSIONS_CEILING":
		limit := d("100.000000")
		if in.EsgGhgScope12Intensity.GreaterThan(limit) {
			return "WARNING", fmt.Sprintf("Soft Warning: esg_ghg_scope_1_2_intensity %s exceeds ceiling %s", in.EsgGhgScope12Intensity.StringFixed(2), limit.StringFixed(2)), true
		}
		return "PASSED", "Compliant: Scope 1+2 emissions intensity within ceiling", true

	case "POST_TRADE_ESG_BOARD_GENDER_DIVERSITY_FLOOR":
		floor := d("0.300000")
		if in.EsgBoardGenderDiversityPct.LessThan(floor) {
			return "WARNING", fmt.Sprintf("Soft Warning: esg_board_gender_diversity_pct %s is below minimum diversity floor %s (SFDR PAI 13)", in.EsgBoardGenderDiversityPct.StringFixed(4), floor.StringFixed(4)), true
		}
		return "PASSED", "Compliant: board female representation satisfies minimum diversity floor", true

	case "POST_TRADE_ESG_HAZARDOUS_WASTE_RATIO_CEILING":
		limit := d("5.000000")
		if in.EsgHazardousWasteRatio.GreaterThan(limit) {
			return "WARNING", fmt.Sprintf("Soft Warning: esg_hazardous_waste_ratio %s exceeds ceiling %s (SFDR PAI 9)", in.EsgHazardousWasteRatio.StringFixed(2), limit.StringFixed(2)), true
		}
		return "PASSED", "Compliant: hazardous waste ratio within ceiling", true

	case "POST_TRADE_EU_TAXONOMY_GREEN_REVENUE_FLOOR":
		floor := d("0.150000")
		if in.EuTaxonomyAlignmentPct.LessThan(floor) {
			return "WARNING", fmt.Sprintf("Soft Warning: eu_taxonomy_alignment_pct %s is below minimum floor %s", in.EuTaxonomyAlignmentPct.StringFixed(4), floor.StringFixed(4)), true
		}
		return "PASSED", "Compliant: EU Taxonomy green revenue satisfies minimum floor", true

	case "POST_TRADE_LIQUIDITY_COVERAGE_RATIO_BUFFER":
		floor := d("1.050000")
		if in.LiquidityCoverageRatio.LessThan(floor) {
			return "BLOCKED", fmt.Sprintf("Hard Block: liquidity_coverage_ratio %s is below mandatory buffer %s (Basel III / UCITS)", in.LiquidityCoverageRatio.StringFixed(4), floor.StringFixed(4)), true
		}
		return "PASSED", "Compliant: stress liquidity coverage ratio satisfies mandatory buffer", true

	default:
		return "", "", false
	}
}
