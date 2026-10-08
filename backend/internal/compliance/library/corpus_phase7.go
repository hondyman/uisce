package library

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// getPhase7PostTradeScenarios returns deterministic scenario vectors for Phase 7 Ownership & Disclosure Rules (Rules 37-42).
func getPhase7PostTradeScenarios() []Scenario {
	return []Scenario{
		// ====================================================================
		// 87. POST_TRADE_SEC_SCHEDULE_13D_5PCT (SEC Schedule 13D/13G >= 5% Voting Equity)
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_SEC_SCHEDULE_13D_5PCT",
			Code:        "POST_TRADE_SEC_SCHEDULE_13D_5PCT:PASS",
			Description: "Compliant firm-wide voting equity below 5% threshold (4.0% <= 5.0%)",
			Input: OrderContext{
				FirmwideEquityVotingPct:      d("0.040000"),
				PriorFirmwideEquityVotingPct: d("0.035000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_SEC_SCHEDULE_13D_5PCT",
			Code:        "POST_TRADE_SEC_SCHEDULE_13D_5PCT:BOUNDARY",
			Description: "Boundary firm-wide voting equity exactly at 5.0% threshold (no crossing breach)",
			Input: OrderContext{
				FirmwideEquityVotingPct:      d("0.050000"),
				PriorFirmwideEquityVotingPct: d("0.050000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_SEC_SCHEDULE_13D_5PCT",
			Code:        "POST_TRADE_SEC_SCHEDULE_13D_5PCT:FAIL_INITIAL_CROSSING",
			Description: "Breached initial threshold crossing from 4.0% to 5.5% (triggers Schedule 13D filing clock)",
			Input: OrderContext{
				FirmwideEquityVotingPct:      d("0.055000"),
				PriorFirmwideEquityVotingPct: d("0.040000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "firmwide_equity_voting_pct"},
		},
		{
			RuleCode:    "POST_TRADE_SEC_SCHEDULE_13D_5PCT",
			Code:        "POST_TRADE_SEC_SCHEDULE_13D_5PCT:ADVERSARIAL_FIRST_EVALUATION",
			Description: "Adversarial first-ever portfolio evaluation at 6.0% with missing prior snapshot (crosses from 0.0)",
			Input: OrderContext{
				FirmwideEquityVotingPct:      d("0.060000"),
				PriorFirmwideEquityVotingPct: decimal.Zero, // Missing prior snapshot / first evaluation
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "firmwide_equity_voting_pct"},
		},

		// ====================================================================
		// 88. POST_TRADE_UK_FCA_DTR5_INITIAL_3PCT (UK FCA DTR 5 >= 3% + 1% Steps)
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_UK_FCA_DTR5_INITIAL_3PCT",
			Code:        "POST_TRADE_UK_FCA_DTR5_INITIAL_3PCT:PASS",
			Description: "Compliant UK issuer voting rights below 3% (2.5% <= 3.0%)",
			Input: OrderContext{
				FirmwideEquityVotingPct:      d("0.025000"),
				PriorFirmwideEquityVotingPct: d("0.020000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_UK_FCA_DTR5_INITIAL_3PCT",
			Code:        "POST_TRADE_UK_FCA_DTR5_INITIAL_3PCT:FAIL_INITIAL_CROSSING",
			Description: "Breached initial UK DTR5 crossing from 2.5% to 3.2% (triggers 2-trading-day filing clock)",
			Input: OrderContext{
				FirmwideEquityVotingPct:      d("0.032000"),
				PriorFirmwideEquityVotingPct: d("0.025000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "firmwide_equity_voting_pct"},
		},
		{
			RuleCode:    "POST_TRADE_UK_FCA_DTR5_INITIAL_3PCT",
			Code:        "POST_TRADE_UK_FCA_DTR5_INITIAL_3PCT:FAIL_STEP_CROSSING_4PCT",
			Description: "Breached integer step crossing from 3.2% to 4.1% (crosses 4% integer boundary requiring filing)",
			Input: OrderContext{
				FirmwideEquityVotingPct:      d("0.041000"),
				PriorFirmwideEquityVotingPct: d("0.032000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "firmwide_equity_voting_pct"},
		},
		{
			RuleCode:    "POST_TRADE_UK_FCA_DTR5_INITIAL_3PCT",
			Code:        "POST_TRADE_UK_FCA_DTR5_INITIAL_3PCT:ADVERSARIAL_DISPOSAL_CROSSING",
			Description: "Adversarial major shareholding reduction below threshold from 3.5% down to 2.2% (triggers cessation disclosure)",
			Input: OrderContext{
				FirmwideEquityVotingPct:      d("0.022000"),
				PriorFirmwideEquityVotingPct: d("0.035000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_UK_FCA_DTR5_INITIAL_3PCT",
			Code:        "POST_TRADE_UK_FCA_DTR5_INITIAL_3PCT:ADVERSARIAL_TENANT_OVERRIDE_4PCT",
			Description: "Adversarial tenant threshold override from 3.0% to 4.0%: asserts 3.5% passes under 4.0% override",
			Input: OrderContext{
				FirmwideEquityVotingPct:      d("0.035000"),
				PriorFirmwideEquityVotingPct: d("0.025000"),
				ParameterOverrides: map[string]string{
					"initial_disclosure_threshold_pct": "0.040000",
					"step_size_pct":                    "0.010000",
				},
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},

		// ====================================================================
		// 89. POST_TRADE_EU_TRANSPARENCY_DIR_5PCT (EU Transparency Directive >= 5%, 10%... Tiers)
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_EU_TRANSPARENCY_DIR_5PCT",
			Code:        "POST_TRADE_EU_TRANSPARENCY_DIR_5PCT:PASS",
			Description: "Compliant EU issuer voting rights below 5% (4.5% <= 5.0%)",
			Input: OrderContext{
				FirmwideEquityVotingPct:      d("0.045000"),
				PriorFirmwideEquityVotingPct: d("0.040000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_EU_TRANSPARENCY_DIR_5PCT",
			Code:        "POST_TRADE_EU_TRANSPARENCY_DIR_5PCT:BOUNDARY",
			Description: "Boundary EU voting rights exactly at 5.0% threshold",
			Input: OrderContext{
				FirmwideEquityVotingPct:      d("0.050000"),
				PriorFirmwideEquityVotingPct: d("0.050000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_EU_TRANSPARENCY_DIR_5PCT",
			Code:        "POST_TRADE_EU_TRANSPARENCY_DIR_5PCT:FAIL_CROSSING_5PCT",
			Description: "Breached initial EU Transparency Directive crossing from 4.5% to 5.8%",
			Input: OrderContext{
				FirmwideEquityVotingPct:      d("0.058000"),
				PriorFirmwideEquityVotingPct: d("0.045000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "firmwide_equity_voting_pct"},
		},
		{
			RuleCode:    "POST_TRADE_EU_TRANSPARENCY_DIR_5PCT",
			Code:        "POST_TRADE_EU_TRANSPARENCY_DIR_5PCT:ADVERSARIAL_CROSSING_FRIDAY_BEFORE_HOLIDAY",
			Description: "Adversarial threshold crossing on Friday before Good Friday holiday (asserts 4-trading-day clock)",
			Input: OrderContext{
				FirmwideEquityVotingPct:      d("0.112000"), // Crosses 10% tier from 8%
				PriorFirmwideEquityVotingPct: d("0.080000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "firmwide_equity_voting_pct"},
		},

		// ====================================================================
		// 90. POST_TRADE_UK_TAKEOVER_MANDATORY_BID_30 (UK Takeover Code Rule 9 >= 30%)
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_UK_TAKEOVER_MANDATORY_BID_30",
			Code:        "POST_TRADE_UK_TAKEOVER_MANDATORY_BID_30:PASS",
			Description: "Compliant voting control below 30% takeover threshold (25% <= 30%)",
			Input: OrderContext{
				FirmwideVotingControlPct: d("0.250000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_UK_TAKEOVER_MANDATORY_BID_30",
			Code:        "POST_TRADE_UK_TAKEOVER_MANDATORY_BID_30:BOUNDARY",
			Description: "Boundary target voting control exactly at 30.0% threshold",
			Input: OrderContext{
				FirmwideVotingControlPct: d("0.300000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_UK_TAKEOVER_MANDATORY_BID_30",
			Code:        "POST_TRADE_UK_TAKEOVER_MANDATORY_BID_30:FAIL_CROSSING_30PCT",
			Description: "Breached mandatory cash offer threshold (31.5% > 30.0% triggering Takeover Code Rule 9)",
			Input: OrderContext{
				FirmwideVotingControlPct: d("0.315000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "firmwide_voting_control_pct"},
		},
		{
			RuleCode:    "POST_TRADE_UK_TAKEOVER_MANDATORY_BID_30",
			Code:        "POST_TRADE_UK_TAKEOVER_MANDATORY_BID_30:ADVERSARIAL_CYCLE_PRUNING",
			Description: "Adversarial 3-tier cyclic feeder hierarchy fixture: asserts hand-computed exact non-duplicated 10.0% ownership",
			Input: OrderContext{
				FirmwideVotingControlPct: d("0.100000"), // Hand-computed exact: cycle pruned, no duplicate addition
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},

		// ====================================================================
		// 91. POST_TRADE_EU_SSR_SHORT_DISCLOSURE_01 (EU SSR Net Short Notification >= 0.10% + 0.1% Steps)
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_EU_SSR_SHORT_DISCLOSURE_01",
			Code:        "POST_TRADE_EU_SSR_SHORT_DISCLOSURE_01:PASS",
			Description: "Compliant EU net short position below 0.10% threshold (0.08% <= 0.10%)",
			Input: OrderContext{
				FirmwideNetShortPct: d("0.000800"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_EU_SSR_SHORT_DISCLOSURE_01",
			Code:        "POST_TRADE_EU_SSR_SHORT_DISCLOSURE_01:BOUNDARY",
			Description: "Boundary net short position exactly at 0.10% notification threshold",
			Input: OrderContext{
				FirmwideNetShortPct: d("0.001000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_EU_SSR_SHORT_DISCLOSURE_01",
			Code:        "POST_TRADE_EU_SSR_SHORT_DISCLOSURE_01:FAIL_NET_SHORT_CROSSING_01",
			Description: "Breached net short position crossing above 0.10% (0.15% > 0.10% triggers T+1 15:30 CET notification)",
			Input: OrderContext{
				FirmwideNetShortPct: d("0.001500"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "firmwide_net_short_pct"},
		},
		{
			RuleCode:    "POST_TRADE_EU_SSR_SHORT_DISCLOSURE_01",
			Code:        "POST_TRADE_EU_SSR_SHORT_DISCLOSURE_01:ADVERSARIAL_OPTIONS_DELTA_NETTING",
			Description: "Adversarial 100k gross short shares offset by 40k delta-equivalent long call options (net short 0.06% <= 0.10%)",
			Input: OrderContext{
				FirmwideNetShortPct: d("0.000600"), // 100k short - 40k delta long = 60k net short / 100M shares out = 0.06%
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},

		// ====================================================================
		// 92. POST_TRADE_UK_FCA_SSR_SHORT_DISCLOSURE_02 (UK FCA SSR Net Short Notification >= 0.20% + 0.1% Steps)
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_UK_FCA_SSR_SHORT_DISCLOSURE_02",
			Code:        "POST_TRADE_UK_FCA_SSR_SHORT_DISCLOSURE_02:PASS",
			Description: "Compliant UK net short position below 0.20% threshold (0.15% <= 0.20%)",
			Input: OrderContext{
				FirmwideNetShortPct: d("0.001500"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_UK_FCA_SSR_SHORT_DISCLOSURE_02",
			Code:        "POST_TRADE_UK_FCA_SSR_SHORT_DISCLOSURE_02:BOUNDARY",
			Description: "Boundary UK net short position exactly at 0.20% threshold",
			Input: OrderContext{
				FirmwideNetShortPct: d("0.002000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_UK_FCA_SSR_SHORT_DISCLOSURE_02",
			Code:        "POST_TRADE_UK_FCA_SSR_SHORT_DISCLOSURE_02:FAIL_NET_SHORT_CROSSING_02",
			Description: "Breached UK net short position crossing above 0.20% (0.28% > 0.20%)",
			Input: OrderContext{
				FirmwideNetShortPct: d("0.002800"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "firmwide_net_short_pct"},
		},
		{
			RuleCode:    "POST_TRADE_UK_FCA_SSR_SHORT_DISCLOSURE_02",
			Code:        "POST_TRADE_UK_FCA_SSR_SHORT_DISCLOSURE_02:ADVERSARIAL_STEADY_STATE_SUPPRESSION",
			Description: "Adversarial steady-state net short (0.25% today, 0.25% yesterday in same step bracket -> no re-alert)",
			Input: OrderContext{
				FirmwideNetShortPct:          d("0.002500"),
				PriorFirmwideNetShortPct:     d("0.002500"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},

		// ====================================================================
		// 93. POST_TRADE_SEC_SCHEDULE_13G_PASSIVE (SEC Schedule 13G Passive Institutional Disclosure >= 5% / >= 10%)
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_SEC_SCHEDULE_13G_PASSIVE",
			Code:        "POST_TRADE_SEC_SCHEDULE_13G_PASSIVE:PASS",
			Description: "Compliant passive holding below 5% initial threshold (4.0% <= 5.0%, passive intent)",
			Input: OrderContext{
				FirmwideEquityVotingPct:      d("0.040000"),
				PriorFirmwideEquityVotingPct: d("0.035000"),
				IsPassiveIntent:              boolPtr(true),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_SEC_SCHEDULE_13G_PASSIVE",
			Code:        "POST_TRADE_SEC_SCHEDULE_13G_PASSIVE:BOUNDARY",
			Description: "Boundary passive holding exactly at 5.0% threshold (no crossing breach)",
			Input: OrderContext{
				FirmwideEquityVotingPct:      d("0.050000"),
				PriorFirmwideEquityVotingPct: d("0.050000"),
				IsPassiveIntent:              boolPtr(true),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_SEC_SCHEDULE_13G_PASSIVE",
			Code:        "POST_TRADE_SEC_SCHEDULE_13G_PASSIVE:FAIL_INITIAL_5PCT",
			Description: "Breached initial passive 5% crossing from 4.0% to 6.5% (triggers annual Schedule 13G filing)",
			Input: OrderContext{
				FirmwideEquityVotingPct:      d("0.065000"),
				PriorFirmwideEquityVotingPct: d("0.040000"),
				IsPassiveIntent:              boolPtr(true),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "firmwide_equity_voting_pct"},
		},
		{
			RuleCode:    "POST_TRADE_SEC_SCHEDULE_13G_PASSIVE",
			Code:        "POST_TRADE_SEC_SCHEDULE_13G_PASSIVE:FAIL_ACCELERATED_10PCT",
			Description: "Breached accelerated 10% threshold crossing from 8.0% to 11.0% (triggers 5-business-day accelerated filing)",
			Input: OrderContext{
				FirmwideEquityVotingPct:      d("0.110000"),
				PriorFirmwideEquityVotingPct: d("0.080000"),
				IsPassiveIntent:              boolPtr(true),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "firmwide_equity_voting_pct"},
		},
		{
			RuleCode:    "POST_TRADE_SEC_SCHEDULE_13G_PASSIVE",
			Code:        "POST_TRADE_SEC_SCHEDULE_13G_PASSIVE:ADVERSARIAL_ACTIVIST_TRANSITION",
			Description: "Adversarial passive intent loss at 6.5% voting equity (is_passive_intent=false forces immediate Schedule 13D conversion)",
			Input: OrderContext{
				FirmwideEquityVotingPct:      d("0.065000"),
				PriorFirmwideEquityVotingPct: d("0.065000"),
				IsPassiveIntent:              boolPtr(false),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "is_passive_intent"},
		},

		// ====================================================================
		// 94. POST_TRADE_ERISA_PLAN_ASSET_25PCT (ERISA Benefit Plan Investor 25% Significant Participation Ceiling)
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_ERISA_PLAN_ASSET_25PCT",
			Code:        "POST_TRADE_ERISA_PLAN_ASSET_25PCT:PASS",
			Description: "Compliant Benefit Plan Investor equity participation below 25% (20.0% < 25.0%)",
			Input: OrderContext{
				ErisaBpiEquityPct: d("0.200000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_ERISA_PLAN_ASSET_25PCT",
			Code:        "POST_TRADE_ERISA_PLAN_ASSET_25PCT:BOUNDARY",
			Description: "Boundary Benefit Plan Investor equity participation at 24.9% (below 25.0% significant participation threshold)",
			Input: OrderContext{
				ErisaBpiEquityPct: d("0.249000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_ERISA_PLAN_ASSET_25PCT",
			Code:        "POST_TRADE_ERISA_PLAN_ASSET_25PCT:FAIL_CROSSING_25PCT",
			Description: "Breached Benefit Plan Investor 25% ceiling (26.5% >= 25.0% triggers Title I plan asset status)",
			Input: OrderContext{
				ErisaBpiEquityPct: d("0.265000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "erisa_bpi_equity_pct"},
		},
		{
			RuleCode:    "POST_TRADE_ERISA_PLAN_ASSET_25PCT",
			Code:        "POST_TRADE_ERISA_PLAN_ASSET_25PCT:ADVERSARIAL_GP_DISREGARDED_DENOMINATOR",
			Description: "Adversarial GP equity exclusion from denominator: $24M BPI / ($100M total - $20M GP) = 30.0% BPI equity >= 25.0%",
			Input: OrderContext{
				ErisaBpiEquityPct: d("0.300000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "erisa_bpi_equity_pct"},
		},
	}
}

func boolPtr(b bool) *bool {
	return &b
}

// evaluatePhase7Scenario evaluates Phase 7 Ownership & Disclosure rules.
func evaluatePhase7Scenario(sc Scenario) (string, string, bool) {
	in := sc.Input
	switch sc.RuleCode {
	case "POST_TRADE_SEC_SCHEDULE_13D_5PCT":
		limit := d("0.050000")
		if in.ParameterOverrides != nil {
			if v, ok := in.ParameterOverrides["max_voting_equity_pct"]; ok {
				limit = d(v)
			}
		}
		// Initial crossing trigger: prior < limit and current > limit
		if in.FirmwideEquityVotingPct.GreaterThan(limit) && in.PriorFirmwideEquityVotingPct.LessThanOrEqual(limit) {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: firmwide_equity_voting_pct %s crosses max_voting_equity_pct %s", sc.RuleCode, in.FirmwideEquityVotingPct, limit), true
		}
		return "PASSED", "Compliant", true

	case "POST_TRADE_UK_FCA_DTR5_INITIAL_3PCT":
		limit := d("0.030000")
		step := d("0.010000")
		if in.ParameterOverrides != nil {
			if v, ok := in.ParameterOverrides["initial_disclosure_threshold_pct"]; ok {
				limit = d(v)
			}
			if v, ok := in.ParameterOverrides["step_size_pct"]; ok {
				step = d(v)
			}
		}
		if in.FirmwideEquityVotingPct.GreaterThan(limit) {
			// Check initial crossing
			if in.PriorFirmwideEquityVotingPct.LessThanOrEqual(limit) {
				return "BLOCKED", fmt.Sprintf("Rule %s breached: firmwide_equity_voting_pct %s crosses initial_disclosure_threshold_pct %s", sc.RuleCode, in.FirmwideEquityVotingPct, limit), true
			}
			// Check integer step crossing (e.g. 3.2% -> 4.1%)
			currStep := in.FirmwideEquityVotingPct.Div(step).Floor()
			priorStep := in.PriorFirmwideEquityVotingPct.Div(step).Floor()
			if currStep.GreaterThan(priorStep) {
				return "BLOCKED", fmt.Sprintf("Rule %s breached: firmwide_equity_voting_pct %s crosses 1%% step boundary (prior %s)", sc.RuleCode, in.FirmwideEquityVotingPct, in.PriorFirmwideEquityVotingPct), true
			}
		}
		return "PASSED", "Compliant", true

	case "POST_TRADE_EU_TRANSPARENCY_DIR_5PCT":
		limit := d("0.050000")
		tierStep := d("0.050000")
		if in.ParameterOverrides != nil {
			if v, ok := in.ParameterOverrides["initial_threshold_pct"]; ok {
				limit = d(v)
			}
			if v, ok := in.ParameterOverrides["tier_step_size_pct"]; ok {
				tierStep = d(v)
			}
		}
		if in.FirmwideEquityVotingPct.GreaterThan(limit) {
			if in.PriorFirmwideEquityVotingPct.LessThanOrEqual(limit) {
				return "BLOCKED", fmt.Sprintf("Rule %s breached: firmwide_equity_voting_pct %s crosses initial_threshold_pct %s", sc.RuleCode, in.FirmwideEquityVotingPct, limit), true
			}
			currTier := in.FirmwideEquityVotingPct.Div(tierStep).Floor()
			priorTier := in.PriorFirmwideEquityVotingPct.Div(tierStep).Floor()
			if currTier.GreaterThan(priorTier) {
				return "BLOCKED", fmt.Sprintf("Rule %s breached: firmwide_equity_voting_pct %s crosses tier boundary %s", sc.RuleCode, in.FirmwideEquityVotingPct, currTier.Mul(tierStep)), true
			}
		}
		return "PASSED", "Compliant", true

	case "POST_TRADE_UK_TAKEOVER_MANDATORY_BID_30":
		limit := d("0.300000")
		if in.ParameterOverrides != nil {
			if v, ok := in.ParameterOverrides["mandatory_bid_threshold_pct"]; ok {
				limit = d(v)
			}
		}
		if in.FirmwideVotingControlPct.GreaterThan(limit) {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: firmwide_voting_control_pct %s exceeds mandatory_bid_threshold_pct %s", sc.RuleCode, in.FirmwideVotingControlPct, limit), true
		}
		return "PASSED", "Compliant", true

	case "POST_TRADE_EU_SSR_SHORT_DISCLOSURE_01":
		limit := d("0.001000")
		if in.ParameterOverrides != nil {
			if v, ok := in.ParameterOverrides["notification_threshold_pct"]; ok {
				limit = d(v)
			}
		}
		if in.FirmwideNetShortPct.GreaterThan(limit) {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: firmwide_net_short_pct %s exceeds notification_threshold_pct %s", sc.RuleCode, in.FirmwideNetShortPct, limit), true
		}
		return "PASSED", "Compliant", true

	case "POST_TRADE_UK_FCA_SSR_SHORT_DISCLOSURE_02":
		limit := d("0.002000")
		step := d("0.001000")
		if in.ParameterOverrides != nil {
			if v, ok := in.ParameterOverrides["notification_threshold_pct"]; ok {
				limit = d(v)
			}
			if v, ok := in.ParameterOverrides["step_increment_pct"]; ok {
				step = d(v)
			}
		}
		// Steady state suppression: if in.PriorFirmwideNetShortPct == in.FirmwideNetShortPct and within same step -> no re-alert
		if in.FirmwideNetShortPct.GreaterThan(limit) {
			if in.PriorFirmwideNetShortPct.LessThanOrEqual(limit) {
				return "BLOCKED", fmt.Sprintf("Rule %s breached: firmwide_net_short_pct %s exceeds notification_threshold_pct %s", sc.RuleCode, in.FirmwideNetShortPct, limit), true
			}
			currStep := in.FirmwideNetShortPct.Div(step).Floor()
			priorStep := in.PriorFirmwideNetShortPct.Div(step).Floor()
			if currStep.GreaterThan(priorStep) {
				return "BLOCKED", fmt.Sprintf("Rule %s breached: firmwide_net_short_pct %s crosses 0.1%% step boundary", sc.RuleCode, in.FirmwideNetShortPct), true
			}
		}
		return "PASSED", "Compliant", true

	case "POST_TRADE_SEC_SCHEDULE_13G_PASSIVE":
		initLimit := d("0.050000")
		accelLimit := d("0.100000")
		ceilLimit := d("0.200000")
		if in.ParameterOverrides != nil {
			if v, ok := in.ParameterOverrides["initial_passive_threshold_pct"]; ok {
				initLimit = d(v)
			}
			if v, ok := in.ParameterOverrides["accelerated_threshold_pct"]; ok {
				accelLimit = d(v)
			}
			if v, ok := in.ParameterOverrides["max_passive_ownership_ceiling_pct"]; ok {
				ceilLimit = d(v)
			}
		}
		isPassive := true
		if in.IsPassiveIntent != nil {
			isPassive = *in.IsPassiveIntent
		}
		if !isPassive && in.FirmwideEquityVotingPct.GreaterThan(initLimit) {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: is_passive_intent false forces Schedule 13D mandatory conversion for voting equity %s", sc.RuleCode, in.FirmwideEquityVotingPct), true
		}
		if in.FirmwideEquityVotingPct.GreaterThanOrEqual(ceilLimit) {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: firmwide_equity_voting_pct %s exceeds 20%% passive ceiling", sc.RuleCode, in.FirmwideEquityVotingPct), true
		}
		if in.FirmwideEquityVotingPct.GreaterThan(accelLimit) && in.PriorFirmwideEquityVotingPct.LessThanOrEqual(accelLimit) {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: firmwide_equity_voting_pct %s crosses 10%% accelerated threshold", sc.RuleCode, in.FirmwideEquityVotingPct), true
		}
		if in.FirmwideEquityVotingPct.GreaterThan(initLimit) && in.PriorFirmwideEquityVotingPct.LessThanOrEqual(initLimit) {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: firmwide_equity_voting_pct %s crosses initial_passive_threshold_pct %s", sc.RuleCode, in.FirmwideEquityVotingPct, initLimit), true
		}
		return "PASSED", "Compliant", true

	case "POST_TRADE_ERISA_PLAN_ASSET_25PCT":
		limit := d("0.250000")
		if in.ParameterOverrides != nil {
			if v, ok := in.ParameterOverrides["max_bpi_equity_pct"]; ok {
				limit = d(v)
			}
		}
		if in.ErisaBpiEquityPct.GreaterThanOrEqual(limit) {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: erisa_bpi_equity_pct %s reaches or exceeds max_bpi_equity_pct %s", sc.RuleCode, in.ErisaBpiEquityPct, limit), true
		}
		return "PASSED", "Compliant", true

	default:
		return "", "", false
	}
}
