package scoring

import (
	"testing"
)

func TestEvaluateToleranceExact(t *testing.T) {
	tol := AttributeTolerance{
		AttributeCode: "LEI",
		MatchType:     MatchExact,
	}

	tests := []struct {
		cand   string
		gold   string
		expect bool
	}{
		{"5493006MHB84DD0ZWV18", "5493006MHB84DD0ZWV18", true},
		{"5493006mhb84dd0zwv18", "5493006MHB84DD0ZWV18", true}, // case insensitive
		{" 5493006MHB84DD0ZWV18 ", "5493006MHB84DD0ZWV18", true}, // trimmed
		{"5493006MHB84DD0ZWV18", "5493006MHB84DD0ZWV19", false},
		{"", "5493006MHB84DD0ZWV18", false},
	}

	for _, tt := range tests {
		got := EvaluateTolerance(tol, tt.cand, tt.gold)
		if got != tt.expect {
			t.Errorf("EvaluateTolerance(%s, %s) = %v; want %v", tt.cand, tt.gold, got, tt.expect)
		}
	}
}

func TestEvaluateToleranceJaroWinkler(t *testing.T) {
	tol := AttributeTolerance{
		AttributeCode: "LEGAL_NAME",
		MatchType:     MatchFuzzyJaro,
		ToleranceVal:  0.92,
	}

	tests := []struct {
		cand   string
		gold   string
		expect bool
	}{
		{"Apple Inc.", "Apple Inc.", true},
		{"Apple Inc", "Apple Inc.", true},
		{"Microsoft Corp.", "Microsoft Corp", true},
		{"Tavistock Rail Group", "Tavistock Rail Group Ltd", true},
		{"Alphabet Inc.", "Microsoft Corp", false},
		{"", "Apple Inc.", false},
	}

	for _, tt := range tests {
		got := EvaluateTolerance(tol, tt.cand, tt.gold)
		if got != tt.expect {
			sim := JaroWinklerSimilarity(tt.cand, tt.gold)
			t.Errorf("EvaluateTolerance(%q, %q) [sim=%.3f] = %v; want %v", tt.cand, tt.gold, sim, got, tt.expect)
		}
	}
}

func TestEvaluateToleranceNumericBP(t *testing.T) {
	tol := AttributeTolerance{
		AttributeCode: "CLOSING_PRICE",
		MatchType:     MatchNumericBP,
		ToleranceVal:  0.0001, // 1 basis point (0.01%)
	}

	tests := []struct {
		cand   string
		gold   string
		expect bool
	}{
		{"100.0000", "100.0000", true},
		{"100.00005", "100.00000", true}, // 0.5 bp diff
		{"100.0001", "100.0000", true},  // 1.0 bp diff
		{"100.0002", "100.0000", false}, // 2.0 bp diff
		{"not_a_number", "100.0000", false},
	}

	for _, tt := range tests {
		got := EvaluateTolerance(tol, tt.cand, tt.gold)
		if got != tt.expect {
			t.Errorf("EvaluateTolerance(%s, %s) = %v; want %v", tt.cand, tt.gold, got, tt.expect)
		}
	}
}

func TestEvaluateToleranceNumericPct(t *testing.T) {
	tol := AttributeTolerance{
		AttributeCode: "MARKET_CAP",
		MatchType:     MatchNumericPct,
		ToleranceVal:  0.01, // 0.01%
	}

	tests := []struct {
		cand   string
		gold   string
		expect bool
	}{
		{"1000000000", "1000000000", true},
		{"1000050000", "1000000000", true}, // 0.005% diff
		{"1000100000", "1000000000", true}, // 0.01% diff
		{"1000500000", "1000000000", false}, // 0.05% diff
		{"0", "0", true},
	}

	for _, tt := range tests {
		got := EvaluateTolerance(tol, tt.cand, tt.gold)
		if got != tt.expect {
			t.Errorf("EvaluateTolerance(%s, %s) = %v; want %v", tt.cand, tt.gold, got, tt.expect)
		}
	}
}

func TestEvaluateToleranceDateLag(t *testing.T) {
	tol := AttributeTolerance{
		AttributeCode: "ANNOUNCE_DATE",
		MatchType:     MatchDateLag,
		ToleranceVal:  1.0, // ±1 day
	}

	tests := []struct {
		cand   string
		gold   string
		expect bool
	}{
		{"2026-09-15", "2026-09-15", true},
		{"2026-09-14", "2026-09-15", true},  // 1 day lag
		{"2026-09-16", "2026-09-15", true},  // 1 day advance
		{"2026-09-12", "2026-09-15", false}, // 3 days diff
		{"invalid-date", "2026-09-15", false},
	}

	for _, tt := range tests {
		got := EvaluateTolerance(tol, tt.cand, tt.gold)
		if got != tt.expect {
			t.Errorf("EvaluateTolerance(%s, %s) = %v; want %v", tt.cand, tt.gold, got, tt.expect)
		}
	}
}
