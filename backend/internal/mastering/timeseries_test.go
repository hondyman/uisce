package mastering

import (
	"database/sql"
	"reflect"
	"testing"
)

func ns(s string) sql.NullString   { return sql.NullString{String: s, Valid: s != ""} }
func nf(f float64) sql.NullFloat64 { return sql.NullFloat64{Float64: f, Valid: true} }

func TestSeriesRankingMostSpecificScope(t *testing.T) {
	c := &seriesConfig{priority: []priorityRow{
		{SourceCd: "BLOOMBERG", Priority: 10}, {SourceCd: "REFINITIV", Priority: 20}, {SourceCd: "ICE", Priority: 30},
		{AssetClass: ns("FixedIncome"), SourceCd: "ICE", Priority: 10},
		{AssetClass: ns("FixedIncome"), SourceCd: "BLOOMBERG", Priority: 20},
		{AssetClass: ns("*"), PriceType: ns("NAV"), SourceCd: "MANUAL", Priority: 1, Fallback: true},
		{AssetClass: ns("Equity"), Currency: ns("JPY"), SourceCd: "REFINITIV", Priority: 5, MaxStaleMin: sql.NullInt64{Int64: 60, Valid: true}},
	}}
	cases := []struct {
		name  string
		inst  instrumentFacts
		ptype string
		ccy   string
		want  []string
	}{
		{"default", instrumentFacts{AssetClass: "Equity"}, "LAST", "GBP", []string{"BLOOMBERG", "REFINITIV", "ICE"}},
		// The asset-class ranking beats the default for the sources it names.
		{"fixed income", instrumentFacts{AssetClass: "FixedIncome"}, "MID", "GBP", []string{"ICE", "BLOOMBERG", "REFINITIV"}},
		// A fallback ranks after every primary source.
		{"fallback last", instrumentFacts{AssetClass: "Equity"}, "NAV", "USD", []string{"BLOOMBERG", "REFINITIV", "ICE", "MANUAL"}},
		{"currency scope", instrumentFacts{AssetClass: "Equity"}, "LAST", "JPY", []string{"REFINITIV", "BLOOMBERG", "ICE"}},
	}
	for _, tc := range cases {
		got, stale := c.ranking(tc.inst, tc.ptype, tc.ccy)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: %v, want %v", tc.name, got, tc.want)
		}
		if tc.name == "currency scope" && stale["REFINITIV"] != 60 {
			t.Errorf("staleness of the scoped row: %v", stale)
		}
	}
}

func TestSeriesThresholds(t *testing.T) {
	c := &seriesConfig{thresholds: []thresholdRow{
		{ID: "any", AssetClass: ns("*"), Warning: nf(5), Error: nf(10), Critical: nf(25)},
		{ID: "fi", AssetClass: ns("FixedIncome"), Warning: nf(1), Error: nf(3), Critical: nf(10)},
		{ID: "fi-mid", AssetClass: ns("FixedIncome"), PriceType: ns("MID"), Warning: nf(0.5), Error: nf(2), Critical: nf(5)},
	}}
	if th := c.threshold(instrumentFacts{AssetClass: "Equity"}, "LAST"); th.ID != "any" {
		t.Errorf("equity: %s", th.ID)
	}
	if th := c.threshold(instrumentFacts{AssetClass: "FixedIncome"}, "BID"); th.ID != "fi" {
		t.Errorf("fixed income bid: %s", th.ID)
	}
	th := c.threshold(instrumentFacts{AssetClass: "FixedIncome"}, "MID")
	if th.ID != "fi-mid" {
		t.Fatalf("fixed income mid: %s", th.ID)
	}
	for pct, want := range map[float64]string{0.4: "", 1: "WARNING", 3: "ERROR", 6: "CRITICAL"} {
		if got := th.level(pct); got != want {
			t.Errorf("level(%v) = %q, want %q", pct, got, want)
		}
	}
	var none *thresholdRow
	if none.level(99) != "" || len(none.data()) != 0 {
		t.Error("no threshold must flag nothing")
	}
	// Selection rules read it as threshold.error etc.
	if d := th.data(); d["error"] != 2.0 || d["type"] != "" {
		t.Errorf("data: %v", d)
	}
}

func TestSelectionContextExtra(t *testing.T) {
	d := SelectionContext{Candidate: Candidate{SourceCd: "ICE", Value: 99.5},
		Extra: map[string]any{"threshold": map[string]any{"error": 3.0}, "value": "must not replace the candidate's"}}.Data()
	if d["value"] != 99.5 {
		t.Errorf("extra context overwrote value: %v", d["value"])
	}
	if th, ok := d["threshold"].(map[string]any); !ok || th["error"] != 3.0 {
		t.Errorf("threshold: %v", d["threshold"])
	}
}
