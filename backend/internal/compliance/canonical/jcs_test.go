package canonical

import (
	"testing"
)

func TestRFC8785CanonicalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Key sorting (ASCII)",
			input:    `{"z": 1, "a": 2, "m": 3}`,
			expected: `{"a":2,"m":3,"z":1}`,
		},
		{
			name:     "Whitespace removal",
			input:    `{ "foo" : [ 1 , 2 , { "bar" : "baz" } ] }`,
			expected: `{"foo":[1,2,{"bar":"baz"}]}`,
		},
		{
			name:     "Negative zero normalization",
			input:    `{"zero": -0, "neg": -0.0}`,
			expected: `{"neg":0,"zero":0}`,
		},
		{
			name:     "RFC 8785 Appendix B.1 - 1e-7 exponential threshold",
			input:    `{"val": 0.0000001}`,
			expected: `{"val":1e-7}`,
		},
		{
			name:     "RFC 8785 Appendix B.1 - 1e-6 fixed threshold",
			input:    `{"val": 0.000001}`,
			expected: `{"val":0.000001}`,
		},
		{
			name:     "RFC 8785 Appendix B.1 - 1e20 fixed threshold",
			input:    `{"val": 100000000000000000000.0}`,
			expected: `{"val":100000000000000000000}`,
		},
		{
			name:     "RFC 8785 Appendix B.1 - 1e21 exponential threshold",
			input:    `{"val": 1000000000000000000000.0}`,
			expected: `{"val":1e+21}`,
		},
		{
			name:     "Exact 2^53 - 1 integer boundary",
			input:    `{"maxSafeInt": 9007199254740991}`,
			expected: `{"maxSafeInt":9007199254740991}`,
		},
		{
			name:     "Exact 2^53 integer boundary",
			input:    `{"beyondSafeInt": 9007199254740992}`,
			expected: `{"beyondSafeInt":9007199254740992}`,
		},
		{
			name:     "Nested object and UTF-16 code unit ordering",
			input:    `{"\u00e9": "accent", "e": "plain", "\ud83d\ude00": "emoji"}`,
			expected: `{"e":"plain","é":"accent","😀":"emoji"}`,
		},
		{
			name:     "Escaping rules",
			input:    `{"escapes": "line1\nline2\t\"quoted\"\\backslash\u0007bell"}`,
			expected: `{"escapes":"line1\nline2\t\"quoted\"\\backslash\u0007bell"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Transform([]byte(tt.input))
			if err != nil {
				t.Fatalf("Transform() unexpected error: %v", err)
			}
			if string(got) != tt.expected {
				t.Errorf("Transform() mismatch:\n got:      %s\n expected: %s", string(got), tt.expected)
			}
		})
	}
}

func TestRFC8785InvalidJSON(t *testing.T) {
	badInputs := []string{
		`{bad: json}`,
		`{"unclosed": "string`,
		`{"trailing": "data"} trailing`,
	}

	for _, bad := range badInputs {
		_, err := Transform([]byte(bad))
		if err == nil {
			t.Errorf("Transform(%q) expected error, got nil", bad)
		}
	}
}

func TestComputePhase7Hashes(t *testing.T) {
	rules := []struct {
		code         string
		ast          string
		params       string
		citation     string
		expectedHash string
	}{
		{
			code:         "POST_TRADE_SEC_SCHEDULE_13D_5PCT",
			ast:          `{"type":"COMPARISON","operator":"LESS_THAN","left":{"type":"METRIC","path":"portfolio.firmwide_equity_voting_pct"},"right":{"type":"PARAM","name":"max_voting_equity_pct"}}`,
			params:       `{"max_voting_equity_pct":"0.050000"}`,
			citation:     "Securities Exchange Act of 1934 Section 13(d)(1) (15 U.S.C. § 78m(d)); SEC Rule 13d-1(a) (17 CFR § 240.13d-1(a)); SEC Modernized Beneficial Ownership Reporting Release No. 33-11253",
			expectedHash: "07cf6f49dbae8e504a023b524e219304d70643e68b578d0bed03ea682bfa9f0d",
		},
		{
			code:         "POST_TRADE_UK_FCA_DTR5_INITIAL_3PCT",
			ast:          `{"type":"COMPARISON","operator":"LESS_THAN","left":{"type":"METRIC","path":"portfolio.firmwide_equity_voting_pct"},"right":{"type":"PARAM","name":"initial_disclosure_threshold_pct"}}`,
			params:       `{"initial_disclosure_threshold_pct":"0.030000","step_size_pct":"0.010000"}`,
			citation:     "UK FCA Disclosure Guidance and Transparency Rules (DTR) Sourcebook 5.1.2R & 5.8.3R; UK Companies Act 2006 Part 43",
			expectedHash: "d3a84cfe55cc68b5dec1745b18a3654a85b9aeb16b332ce9db3cb3e7a80e272e",
		},
		{
			code:         "POST_TRADE_EU_TRANSPARENCY_DIR_5PCT",
			ast:          `{"type":"COMPARISON","operator":"LESS_THAN","left":{"type":"METRIC","path":"portfolio.firmwide_equity_voting_pct"},"right":{"type":"PARAM","name":"initial_threshold_pct"}}`,
			params:       `{"initial_threshold_pct":"0.050000","tier_step_size_pct":"0.050000"}`,
			citation:     "Directive 2004/109/EC of the European Parliament and of the Council (Transparency Directive) Art. 9(1) & Art. 12; Commission Delegated Regulation (EU) 2015/761",
			expectedHash: "0a8c5e70ef00a764812ac4cbe0ca0dac3165b68bb9916e2f0554602e0edd24df",
		},
		{
			code:         "POST_TRADE_UK_TAKEOVER_MANDATORY_BID_30",
			ast:          `{"type":"COMPARISON","operator":"LESS_THAN","left":{"type":"METRIC","path":"portfolio.firmwide_voting_control_pct"},"right":{"type":"PARAM","name":"mandatory_bid_threshold_pct"}}`,
			params:       `{"mandatory_bid_threshold_pct":"0.300000"}`,
			citation:     "The Takeover Code (The City Code on Takeovers and Mergers) Rule 9.1(a) & Rule 9.5; Companies Act 2006 Part 28",
			expectedHash: "44f9f009f478a4f5e9faffac72abb096d61660753a8f4dea132824b149d4bbec",
		},
		{
			code:         "POST_TRADE_EU_SSR_SHORT_DISCLOSURE_01",
			ast:          `{"type":"COMPARISON","operator":"LESS_THAN","left":{"type":"METRIC","path":"portfolio.firmwide_net_short_pct"},"right":{"type":"PARAM","name":"notification_threshold_pct"}}`,
			params:       `{"notification_threshold_pct":"0.001000","step_increment_pct":"0.001000"}`,
			citation:     "Regulation (EU) No 236/2012 on short selling and certain aspects of credit default swaps (SSR) Art. 5(1) & Art. 6(1); Commission Delegated Regulation (EU) 2022/27",
			expectedHash: "1d55c8582845eca1cbdabf972743a74c7972da46c6515e3f1f32a32633fd2a63",
		},
		{
			code:         "POST_TRADE_UK_FCA_SSR_SHORT_DISCLOSURE_02",
			ast:          `{"type":"COMPARISON","operator":"LESS_THAN","left":{"type":"METRIC","path":"portfolio.firmwide_net_short_pct"},"right":{"type":"PARAM","name":"notification_threshold_pct"}}`,
			params:       `{"notification_threshold_pct":"0.002000","step_increment_pct":"0.001000"}`,
			citation:     "UK Short Selling Regulation (SI 2012/2911) Art. 5 & Art. 6; FCA Handbook Short Selling Sourcebook",
			expectedHash: "16b6af93f5c454fa845b70bfd3446cd9d7bee5246bb0b4d2255e5db4a5cec549",
		},
		{
			code:         "POST_TRADE_SEC_SCHEDULE_13G_PASSIVE",
			ast:          `{"type":"LOGICAL_AND","operator":"AND","conditions":[{"type":"COMPARISON","operator":"LESS_THAN","left":{"type":"METRIC","path":"portfolio.firmwide_equity_voting_pct"},"right":{"type":"PARAM","name":"initial_passive_threshold_pct"}},{"type":"COMPARISON","operator":"EQUAL","left":{"type":"METRIC","path":"portfolio.is_passive_intent"},"right":{"type":"PARAM","name":"is_passive_intent"}}]}`,
			params:       `{"accelerated_threshold_pct":"0.100000","filer_category":"QII_QUALIFIED_INSTITUTIONAL","initial_passive_threshold_pct":"0.050000","is_passive_intent":true,"max_passive_ownership_ceiling_pct":"0.200000"}`,
			citation:     "Securities Exchange Act of 1934 Section 13(g) (15 U.S.C. § 78m(g)); SEC Rule 13d-1(b), (c), (d) (17 CFR § 240.13d-1); SEC Release No. 33-11253",
			expectedHash: "5a82c7d5241f092f1bb27218cae8cb97cadb9c7d2b173b45c9bf29ccf1e50cb4",
		},
		{
			code:         "POST_TRADE_ERISA_PLAN_ASSET_25PCT",
			ast:          `{"type":"COMPARISON","operator":"LESS_THAN","left":{"type":"METRIC","path":"portfolio.erisa_bpi_equity_pct"},"right":{"type":"PARAM","name":"max_bpi_equity_pct"}}`,
			params:       `{"disregard_gp_interests":true,"max_bpi_equity_pct":"0.250000"}`,
			citation:     "Employee Retirement Income Security Act of 1974 (ERISA) § 3(42) (29 U.S.C. § 1002(42)); 29 CFR § 2510.3-101 (DOL Plan Asset Regulation); ERISA § 406 Prohibited Transactions",
			expectedHash: "9b51f1a16678c5030f365ad8afbd7995cb3192f42830e9e90eea1d188d678ecf",
		},
	}

	for _, r := range rules {
		t.Run(r.code, func(t *testing.T) {
			hash, err := ComputeRuleContentHashFromRaw([]byte(r.ast), []byte(r.params), r.citation)
			if err != nil {
				t.Fatalf("ComputeRuleContentHashFromRaw failed: %v", err)
			}
			t.Logf("Rule %s computed canonical hash: %s", r.code, hash)
			if r.expectedHash != "" && hash != r.expectedHash {
				t.Errorf("Rule %s hash mismatch:\n got:      %s\n expected: %s", r.code, hash, r.expectedHash)
			}
		})
	}
}


