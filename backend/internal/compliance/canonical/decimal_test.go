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
