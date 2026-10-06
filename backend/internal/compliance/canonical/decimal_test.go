package canonical

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"
)

func TestDecimal6Normalization(t *testing.T) {
	tests := []struct {
		name     string
		input    decimal.Decimal
		expected string
	}{
		{
			name:     "Integer to 6 decimals",
			input:    decimal.NewFromInt(100),
			expected: "100.000000",
		},
		{
			name:     "Single decimal place",
			input:    decimal.NewFromFloat(123.4),
			expected: "123.400000",
		},
		{
			name:     "Small fractional decimal at 6 places",
			input:    decimal.NewFromFloat(0.000001),
			expected: "0.000001",
		},
		{
			name:     "Trailing zeros beyond 6 are accepted and formatted to 6",
			input:    decimal.RequireFromString("123.45000000"),
			expected: "123.450000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FormatDecimal6(tt.input)
			if err != nil {
				t.Fatalf("FormatDecimal6() unexpected error: %v", err)
			}
			if got != tt.expected {
				t.Errorf("FormatDecimal6() mismatch: got %s, expected %s", got, tt.expected)
			}
		})
	}
}

func TestDecimal6ScaleRejection(t *testing.T) {
	// Values with non-zero fractional digits beyond 6 decimal places must be rejected!
	exceededValues := []string{
		"123.4567891",
		"0.0000001",
		"100.0000009",
	}

	for _, s := range exceededValues {
		d := decimal.RequireFromString(s)
		_, err := FormatDecimal6(d)
		if err == nil {
			t.Errorf("FormatDecimal6(%s) expected ErrDecimalScaleExceeded, got nil", s)
		}
		if !errors.Is(err, ErrDecimalScaleExceeded) {
			t.Errorf("FormatDecimal6(%s) expected ErrDecimalScaleExceeded, got %v", s, err)
		}
	}
}

func TestCanonicalDecimalMap(t *testing.T) {
	inputMap := map[string]interface{}{
		"limit":       decimal.NewFromFloat(0.05),
		"marketValue": 1250000.5,
		"symbol":      "AAPL",
		"nested": map[string]interface{}{
			"nav": decimal.NewFromInt(10000000),
		},
	}

	res, err := CanonicalDecimalMap(inputMap)
	if err != nil {
		t.Fatalf("CanonicalDecimalMap() error: %v", err)
	}
	raw, err := Marshal(res)
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	expected := `{"limit":"0.050000","marketValue":"1250000.500000","nested":{"nav":"10000000.000000"},"symbol":"AAPL"}`
	if string(raw) != expected {
		t.Errorf("CanonicalDecimalMap() JSON mismatch:\n got:      %s\n expected: %s", string(raw), expected)
	}
}

func TestComputePilotRuleHashes(t *testing.T) {
	h1, err := ComputeRuleContentHashFromRaw(
		[]byte(`{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.ucits_aggregate_above_5pct_exposure"},"op":"LTE","right":{"type":"PARAM","name":"max_aggregate_above_5pct_pct"}}`),
		[]byte(`{"max_aggregate_above_5pct_pct":"0.400000","max_single_issuer_pct":"0.100000"}`),
		"UCITS Directive 2009/65/EC Art. 52(1)-(2)",
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("UCITS_5_10_40 hash: %s", h1)

	h2, err := ComputeRuleContentHashFromRaw(
		[]byte(`{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.restricted_144a_exposure_pct"},"op":"LTE","right":{"type":"PARAM","name":"max_144a_non_qib_pct"}}`),
		[]byte(`{"max_144a_non_qib_pct":"0.150000"}`),
		"SEC Rule 144A / Investment Company Act Rule 22e-4",
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("SEC_144A_QIB_HOLDING hash: %s", h2)

	h3, err := ComputeRuleContentHashFromRaw(
		[]byte(`{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.margin_utilization_pct"},"op":"LTE","right":{"type":"PARAM","name":"max_margin_utilization_pct"}}`),
		[]byte(`{"max_margin_utilization_pct":"0.800000"}`),
		"FINRA Rule 4210 / House Margin Policy",
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("MARGIN_UTILIZATION_80 hash: %s", h3)
}

func TestComputePhase1Tranche1RuleHashes(t *testing.T) {
	h1, err := ComputeRuleContentHashFromRaw(
		[]byte(`{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.max_group_issuer_exposure_pct"},"op":"LTE","right":{"type":"PARAM","name":"max_group_issuer_pct"}}`),
		[]byte(`{"max_group_issuer_pct":"0.200000"}`),
		"UCITS Directive 2009/65/EC Art. 52(3) / Investment Company Act Sec. 12(d)",
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("POST_TRADE_GROUP_ISSUER_20 hash: %s", h1)

	h2, err := ComputeRuleContentHashFromRaw(
		[]byte(`{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.max_issuer_debt_exposure_pct"},"op":"LTE","right":{"type":"PARAM","name":"max_issuer_debt_pct"}}`),
		[]byte(`{"max_issuer_debt_pct":"0.150000"}`),
		"FINRA Rule 4210 / Institutional Fixed Income Mandate",
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("POST_TRADE_ISSUER_DEBT_15 hash: %s", h2)

	h3, err := ComputeRuleContentHashFromRaw(
		[]byte(`{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.max_counterparty_pfe_exposure_pct"},"op":"LTE","right":{"type":"PARAM","name":"max_counterparty_pfe_pct"}}`),
		[]byte(`{"max_counterparty_pfe_pct":"0.100000"}`),
		"BCBS 279 Standardised Approach for Counterparty Credit Risk (SA-CCR) / EMIR Art. 11",
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("POST_TRADE_COUNTERPARTY_PFE_10 hash: %s", h3)

	h4, err := ComputeRuleContentHashFromRaw(
		[]byte(`{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.cash_and_equivalent_pct"},"op":"GTE","right":{"type":"PARAM","name":"min_cash_pct"}}`),
		[]byte(`{"min_cash_pct":"0.050000"}`),
		"ESMA Guidelines on Liquidity Stress Testing / UCITS Liquidity Management",
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("POST_TRADE_CASH_MIN_5 hash: %s", h4)
}

func TestComputePhase1Tranche2RuleHashes(t *testing.T) {
	rules := []struct {
		code      string
		ast       string
		params    string
		citation  string
	}{
		{
			code:     "POST_TRADE_SOVEREIGN_EXPOSURE_35",
			ast:      `{"left":{"path":"portfolio.max_sovereign_exposure_pct","type":"METRIC"},"operator":"LESS_THAN_OR_EQUAL","right":{"name":"max_sovereign_exposure_pct","type":"PARAM"},"type":"COMPARISON"}`,
			params:   `{"max_sovereign_exposure_pct":"0.350000","require_oecd_member":false}`,
			citation: "UCITS Directive 2009/65/EC Art. 54(1); SEC Rule 2a-7(d)(2); ESMA Guidelines ESMA/2014/937 §43",
		},
		{
			code:     "POST_TRADE_AGENCY_SUPRA_25",
			ast:      `{"left":{"path":"portfolio.max_agency_supra_exposure_pct","type":"METRIC"},"operator":"LESS_THAN_OR_EQUAL","right":{"name":"max_agency_supra_pct","type":"PARAM"},"type":"COMPARISON"}`,
			params:   `{"include_multilateral_dev_banks":true,"max_agency_supra_pct":"0.250000"}`,
			citation: "ESMA Guidelines on Money Market Funds (ESMA34-49-115) §4.2; UCITS Directive 2009/65/EC Art. 52(2)",
		},
		{
			code:     "POST_TRADE_MUNI_OBLIGOR_10",
			ast:      `{"left":{"path":"portfolio.max_muni_obligor_exposure_pct","type":"METRIC"},"operator":"LESS_THAN_OR_EQUAL","right":{"name":"max_muni_obligor_pct","type":"PARAM"},"type":"COMPARISON"}`,
			params:   `{"max_muni_obligor_pct":"0.100000","revenue_bond_lookthrough":true}`,
			citation: "17 CFR § 270.2a-7(d)(3)(i); MSRB Rules G-17 & G-19; US Investment Company Act 1940 §8(b)(1)",
		},
		{
			code:     "POST_TRADE_CCP_CLEARING_EXPOSURE_15",
			ast:      `{"left":{"path":"portfolio.max_ccp_exposure_pct","type":"METRIC"},"operator":"LESS_THAN_OR_EQUAL","right":{"name":"max_ccp_exposure_pct","type":"PARAM"},"type":"COMPARISON"}`,
			params:   `{"max_ccp_exposure_pct":"0.150000","qualifying_ccp_only":true}`,
			citation: "EMIR Regulation (EU) No 648/2012 Art. 47; CSDR Regulation (EU) No 909/2014 Art. 12; CFTC 17 CFR Part 39",
		},
		{
			code:     "POST_TRADE_CUSTODIAN_CONCENTRATION_20",
			ast:      `{"left":{"path":"portfolio.max_custodian_concentration_pct","type":"METRIC"},"operator":"LESS_THAN_OR_EQUAL","right":{"name":"max_custodian_pct","type":"PARAM"},"type":"COMPARISON"}`,
			params:   `{"max_custodian_pct":"0.200000","segregated_sub_custody_exemption":false}`,
			citation: "AIFMD Directive 2011/61/EU Art. 21(8); UCITS Directive 2009/65/EC Art. 22; SEC Custody Rule 17 CFR § 275.206(4)-2",
		},
		{
			code:     "POST_TRADE_BANK_DEPOSIT_20",
			ast:      `{"left":{"path":"portfolio.max_bank_deposit_pct","type":"METRIC"},"operator":"LESS_THAN_OR_EQUAL","right":{"name":"max_bank_deposit_pct","type":"PARAM"},"type":"COMPARISON"}`,
			params:   `{"credit_institution_tier1_only":true,"max_bank_deposit_pct":"0.200000"}`,
			citation: "Institutional Cash Management Policy & FCA COLL 5.2.10R (House limit applying to aggregate operational and sweep bank cash; cross-ref statutory UCITS_DEPOSIT_20)",
		},
		{
			code:     "POST_TRADE_SEC_LENDING_COLLATERAL_102",
			ast:      `{"left":{"path":"portfolio.sec_lending_collateral_ratio","type":"METRIC"},"operator":"GREATER_THAN_OR_EQUAL","right":{"name":"min_collateral_ratio","type":"PARAM"},"type":"COMPARISON"}`,
			params:   `{"daily_mark_to_market":true,"min_collateral_ratio":"1.020000"}`,
			citation: "ESMA Guidelines on ETFs and other UCITS issues (ESMA/2014/937) §43(e); SEC Rule 17 CFR § 240.15c3-3; ICMA GMSLA Schedule",
		},
	}

	for _, r := range rules {
		h, err := ComputeRuleContentHashFromRaw([]byte(r.ast), []byte(r.params), r.citation)
		if err != nil {
			t.Fatalf("ComputeRuleContentHashFromRaw for %s failed: %v", r.code, err)
		}
		t.Logf("%s hash: %s", r.code, h)
	}
}

func TestComputePhase2Tranche1RuleHashes(t *testing.T) {
	rules := []struct {
		code     string
		ast      string
		params   string
		citation string
	}{
		{
			code:     "POST_TRADE_UNCLASSIFIED_CEILING_5",
			ast:      `{"left":{"path":"portfolio.unclassified_securities_pct","type":"METRIC"},"operator":"LESS_THAN_OR_EQUAL","right":{"name":"max_unclassified_pct","type":"PARAM"},"type":"COMPARISON"}`,
			params:   `{"max_unclassified_pct":"0.050000"}`,
			citation: "Institutional Risk Policy & ESMA Guidelines on Portfolio Transparency (ESMA34-49-115) §6; SEC Rule 22e-4",
		},
		{
			code:     "POST_TRADE_SECTOR_CONCENTRATION_25",
			ast:      `{"left":{"path":"portfolio.max_sector_exposure_pct","type":"METRIC"},"operator":"LESS_THAN_OR_EQUAL","right":{"name":"max_sector_pct","type":"PARAM"},"type":"COMPARISON"}`,
			params:   `{"max_sector_pct":"0.250000"}`,
			citation: "Institutional Mandate Concentration Standards; UCITS Directive 2009/65/EC Art. 52(1); SEC Investment Company Act §8(b)(1)",
		},
		{
			code:     "POST_TRADE_INDUSTRY_GROUP_15",
			ast:      `{"left":{"path":"portfolio.max_industry_group_pct","type":"METRIC"},"operator":"LESS_THAN_OR_EQUAL","right":{"name":"max_industry_group_pct","type":"PARAM"},"type":"COMPARISON"}`,
			params:   `{"max_industry_group_pct":"0.150000"}`,
			citation: "Institutional Investment Mandate Standard; US Investment Company Act 1940 §8(b)(1); 17 CFR § 270.8b-16",
		},
		{
			code:     "POST_TRADE_CYCLICAL_SECTOR_35",
			ast:      `{"left":{"path":"portfolio.cyclical_sectors_aggregate_pct","type":"METRIC"},"operator":"LESS_THAN_OR_EQUAL","right":{"name":"max_cyclical_sector_pct","type":"PARAM"},"type":"COMPARISON"}`,
			params:   `{"max_cyclical_sector_pct":"0.350000"}`,
			citation: "Institutional Macro Risk Policy; ESMA Guidelines on Liquidity Stress Testing (ESMA34-49-286) §4",
		},
		{
			code:     "POST_TRADE_EMERGING_MARKET_20",
			ast:      `{"left":{"path":"portfolio.emerging_markets_pct","type":"METRIC"},"operator":"LESS_THAN_OR_EQUAL","right":{"name":"max_emerging_market_pct","type":"PARAM"},"type":"COMPARISON"}`,
			params:   `{"max_emerging_market_pct":"0.200000"}`,
			citation: "MSCI Emerging Markets Classification Standard; UCITS Directive 2009/65/EC Art. 50(1)(f); ESMA Guidelines 2014/937",
		},
		{
			code:     "POST_TRADE_NON_OECD_EXPOSURE_10",
			ast:      `{"left":{"path":"portfolio.non_oecd_exposure_pct","type":"METRIC"},"operator":"LESS_THAN_OR_EQUAL","right":{"name":"max_non_oecd_pct","type":"PARAM"},"type":"COMPARISON"}`,
			params:   `{"max_non_oecd_pct":"0.100000"}`,
			citation: "OECD Code of Liberalisation of Capital Movements; UCITS Directive 2009/65/EC Art. 50(1)(d)",
		},
		{
			code:     "POST_TRADE_FRONTIER_MARKET_5",
			ast:      `{"left":{"path":"portfolio.frontier_markets_pct","type":"METRIC"},"operator":"LESS_THAN_OR_EQUAL","right":{"name":"max_frontier_market_pct","type":"PARAM"},"type":"COMPARISON"}`,
			params:   `{"max_frontier_market_pct":"0.050000"}`,
			citation: "MSCI Frontier Markets Index Methodology §2; ESMA Liquidity Risk Management Guidelines",
		},
		{
			code:     "POST_TRADE_SANCTION_LIST_ZERO_TOLERANCE",
			ast:      `{"left":{"path":"portfolio.sanctioned_entity_matches_count","type":"METRIC"},"operator":"EQUAL_TO","right":{"name":"max_sanctioned_matches","type":"PARAM"},"type":"COMPARISON"}`,
			params:   `{"max_sanctioned_matches":0}`,
			citation: "OFAC Sanctions Regulations 31 CFR Part 500; EU Council Regulation (EU) No 269/2014 Art. 2; UK Sanctions and Anti-Money Laundering Act 2018 §1",
		},
		{
			code:     "POST_TRADE_FX_FORWARD_UNHEDGED_30",
			ast:      `{"left":{"path":"portfolio.unhedged_fx_exposure_pct","type":"METRIC"},"operator":"LESS_THAN_OR_EQUAL","right":{"name":"max_unhedged_fx_pct","type":"PARAM"},"type":"COMPARISON"}`,
			params:   `{"max_unhedged_fx_pct":"0.300000"}`,
			citation: "UCITS Directive 2009/65/EC Art. 51(3); ESMA/2014/937 Guidelines on ETF and Currency Hedged Sub-Funds; Institutional Mandate Risk Policy §4.2",
		},
		{
			code:     "POST_TRADE_HIGH_YIELD_CEILING_10",
			ast:      `{"left":{"path":"portfolio.high_yield_debt_exposure_pct","type":"METRIC"},"operator":"LESS_THAN_OR_EQUAL","right":{"name":"max_high_yield_pct","type":"PARAM"},"type":"COMPARISON"}`,
			params:   `{"max_high_yield_pct":"0.100000"}`,
			citation: "House Fixed Income Policy (High-Yield Ceiling); UCITS Directive 2009/65/EC Art. 52(1) (Non-Investment Grade Debt Concentration); SEC Investment Company Act §8(b)(1)",
		},
		{
			code:     "POST_TRADE_SPLIT_RATING_CONSERVATIVE_FLOOR",
			ast:      `{"left":{"path":"portfolio.split_rating_worst_grade_rank","type":"METRIC"},"operator":"LESS_THAN_OR_EQUAL","right":{"name":"worst_permissible_rank","type":"PARAM"},"type":"COMPARISON"}`,
			params:   `{"worst_permissible_rank":10}`,
			citation: "House Credit Policy (Conservative Rating Floor); SEC Rule 2a-7(d)(2) (Second Tier Security Evaluation); ESMA Guidelines on Money Market Funds (ESMA34-49-115) §4.1; UCITS Directive 2009/65/EC Art. 51",
		},
		{
			code:     "POST_TRADE_ESG_CONTROVERSIAL_WEAPONS_0",
			ast:      `{"left":{"path":"portfolio.esg_controversial_weapons_pct","type":"METRIC"},"operator":"LESS_THAN_OR_EQUAL","right":{"name":"max_weapons_exposure_pct","type":"PARAM"},"type":"COMPARISON"}`,
			params:   `{"max_weapons_exposure_pct":"0.000000"}`,
			citation: "Convention on Cluster Munitions (CCM) Art. 2; Anti-Personnel Mine Ban Convention (Ottawa Treaty); SFDR Regulation (EU) 2019/2088 Art. 8/9 Mandatory PAI 14",
		},
		{
			code:     "POST_TRADE_ESG_THERMAL_COAL_REVENUE_5",
			ast:      `{"left":{"path":"portfolio.esg_thermal_coal_revenue_pct","type":"METRIC"},"operator":"LESS_THAN_OR_EQUAL","right":{"name":"max_coal_revenue_pct","type":"PARAM"},"type":"COMPARISON"}`,
			params:   `{"max_coal_revenue_pct":"0.050000"}`,
			citation: "SFDR Regulation (EU) 2019/2088 Regulatory Technical Standards PAI 4; Paris Aligned Benchmark Regulation (EU) 2020/1818 Art. 12(1)(d)",
		},
		{
			code:     "POST_TRADE_ESG_TOBACCO_REVENUE_5",
			ast:      `{"left":{"path":"portfolio.esg_tobacco_revenue_pct","type":"METRIC"},"operator":"LESS_THAN_OR_EQUAL","right":{"name":"max_tobacco_revenue_pct","type":"PARAM"},"type":"COMPARISON"}`,
			params:   `{"max_tobacco_revenue_pct":"0.050000"}`,
			citation: "WHO Framework Convention on Tobacco Control (FCTC) Art. 5.3; Paris Aligned Benchmark Regulation (EU) 2020/1818 Art. 12(1)(e); UN Global Compact Principle 7",
		},
		{
			code:     "POST_TRADE_ILLIQUID_TIER3_ASSETS_10",
			ast:      `{"left":{"path":"portfolio.illiquid_level3_assets_pct","type":"METRIC"},"operator":"LESS_THAN_OR_EQUAL","right":{"name":"max_illiquid_tier3_pct","type":"PARAM"},"type":"COMPARISON"}`,
			params:   `{"max_illiquid_tier3_pct":"0.100000"}`,
			citation: "SEC Rule 22e-4 Liquidity Risk Management Program (Illiquid Investments Ceiling); IFRS 13 Fair Value Measurement Level 3 Inputs §72; UCITS Eligible Assets Directive 2007/16/EC Art. 2",
		},
		{
			code:     "POST_TRADE_SETTLEMENT_FAIL_CONCENTRATION_5",
			ast:      `{"left":{"path":"portfolio.settlement_fail_exposure_pct","type":"METRIC"},"operator":"LESS_THAN_OR_EQUAL","right":{"name":"max_settlement_fail_pct","type":"PARAM"},"type":"COMPARISON"}`,
			params:   `{"max_settlement_fail_pct":"0.050000"}`,
			citation: "CSDR Regulation (EU) No 909/2014 Settlement Discipline Regime Art. 6-7; US SEC Rule 15c6-1 (T+1 Settlement Integrity & Fails Monitoring)",
		},
		{
			code:     "POST_TRADE_ESG_WACI_PORTFOLIO_CEILING",
			ast:      `{"left":{"path":"portfolio.esg_waci_tco2e_per_m_revenue","type":"METRIC"},"operator":"LESS_THAN_OR_EQUAL","right":{"name":"max_waci_tco2e_per_m_revenue","type":"PARAM"},"type":"COMPARISON"}`,
			params:   `{"max_waci_tco2e_per_m_revenue":"150.000000","min_emissions_data_coverage_pct":"0.750000"}`,
			citation: "TCFD Recommendations on Metrics and Targets §C.2; SFDR Regulation (EU) 2019/2088 Regulatory Technical Standards PAI 2 (Carbon Footprint / WACI); EU Paris-Aligned Benchmarks Regulation (EU) 2020/1818 Art. 9",
		},
		{
			code:     "POST_TRADE_ESG_SCOPE_1_2_EMISSIONS_CEILING",
			ast:      `{"left":{"path":"portfolio.esg_ghg_scope_1_2_intensity","type":"METRIC"},"operator":"LESS_THAN_OR_EQUAL","right":{"name":"max_ghg_scope_1_2_intensity","type":"PARAM"},"type":"COMPARISON"}`,
			params:   `{"max_ghg_scope_1_2_intensity":"100.000000"}`,
			citation: "GHG Protocol Corporate Standard (Scope 1 & 2 Emissions); SFDR Regulation (EU) 2019/2088 PAI 1 (GHG Emissions); ESRS E1 Climate Change §44",
		},
		{
			code:     "POST_TRADE_ESG_BOARD_GENDER_DIVERSITY_FLOOR",
			ast:      `{"left":{"path":"portfolio.esg_board_gender_diversity_pct","type":"METRIC"},"operator":"GREATER_THAN_OR_EQUAL","right":{"name":"min_board_gender_diversity_pct","type":"PARAM"},"type":"COMPARISON"}`,
			params:   `{"min_board_gender_diversity_pct":"0.300000"}`,
			citation: "EU Corporate Sustainability Reporting Directive (CSRD) / ESRS S1-9; SFDR Regulation (EU) 2019/2088 Mandatory PAI 13 (Board Gender Diversity); UK FCA Listing Rules (PS22/3) LR 9.8.6R",
		},
		{
			code:     "POST_TRADE_ESG_HAZARDOUS_WASTE_RATIO_CEILING",
			ast:      `{"left":{"path":"portfolio.esg_hazardous_waste_ratio","type":"METRIC"},"operator":"LESS_THAN_OR_EQUAL","right":{"name":"max_hazardous_waste_ratio","type":"PARAM"},"type":"COMPARISON"}`,
			params:   `{"max_hazardous_waste_ratio":"5.000000"}`,
			citation: "SFDR Regulation (EU) 2019/2088 Mandatory PAI 9 (Hazardous Waste and Radioactive Waste Ratio); Basel Convention on the Control of Transboundary Movements of Hazardous Wastes",
		},
		{
			code:     "POST_TRADE_EU_TAXONOMY_GREEN_REVENUE_FLOOR",
			ast:      `{"left":{"path":"portfolio.eu_taxonomy_alignment_pct","type":"METRIC"},"operator":"GREATER_THAN_OR_EQUAL","right":{"name":"min_taxonomy_alignment_pct","type":"PARAM"},"type":"COMPARISON"}`,
			params:   `{"min_taxonomy_alignment_pct":"0.150000"}`,
			citation: "EU Taxonomy Regulation (EU) 2020/852 Art. 3, Art. 5 (Substantial Contribution to Climate Change Mitigation); SFDR Regulation (EU) 2019/2088 Art. 8(2a) & Art. 9(4a)",
		},
		{
			code:     "POST_TRADE_LIQUIDITY_COVERAGE_RATIO_BUFFER",
			ast:      `{"left":{"path":"portfolio.liquidity_coverage_ratio","type":"METRIC"},"operator":"GREATER_THAN_OR_EQUAL","right":{"name":"min_lcr_buffer_ratio","type":"PARAM"},"type":"COMPARISON"}`,
			params:   `{"min_lcr_buffer_ratio":"1.050000"}`,
			citation: "Basel III Liquidity Coverage Ratio (LCR) Supervisory Standard (BCBS 238); UCITS Liquidity Risk Management Guidelines (ESMA34-49-286) §5; SEC Rule 22e-4",
		},
	}

	for _, r := range rules {
		h, err := ComputeRuleContentHashFromRaw([]byte(r.ast), []byte(r.params), r.citation)
		if err != nil {
			t.Fatalf("ComputeRuleContentHashFromRaw for %s failed: %v", r.code, err)
		}
		t.Logf("%s hash: %s", r.code, h)
	}
}
