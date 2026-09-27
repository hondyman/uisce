package datapipeline

import "testing"

func TestNormalizeAssetClass(t *testing.T) {
	cases := map[string]string{
		"Fixed Income": "FixedIncome",
		"fixed_income": "FixedIncome",
		"EQUITY":       "Equity",
		"Equity":       "Equity",
		"Bond":         "FixedIncome",
		"FX":           "FX",
		"Commodity":    "Commodity",
	}
	for in, want := range cases {
		got := normalizeAssetClass(in)
		if got != want {
			t.Fatalf("normalizeAssetClass(%q)=%v want %q", in, got, want)
		}
	}
}
