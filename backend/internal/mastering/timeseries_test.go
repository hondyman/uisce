package mastering

import (
	"database/sql"
	"reflect"
	"testing"
	"time"
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

func TestPriceOverridePolicy(t *testing.T) {
	p := &Policy{Mode: ModeApproval, ApprovalsRequired: 1, HighRiskApprovals: 2, HighRiskAttributes: []string{"NAV"}}
	if n := p.required("NAV@2026-09-26"); n != 2 {
		t.Errorf("high-risk price type: %d approvals, want 2", n)
	}
	if n := p.required("LAST@2026-09-26"); n != 1 {
		t.Errorf("other price type: %d, want 1", n)
	}
	if n := p.required("NAVIGATOR"); n != 1 {
		t.Errorf("a record attribute that starts like a price type: %d, want 1", n)
	}
	if pt, d, ok := parsePriceAttr("OFFICIAL_CLOSE@2026-09-25"); !ok || pt != "OFFICIAL_CLOSE" || d != "2026-09-25" {
		t.Errorf("parse: %s %s %v", pt, d, ok)
	}
	if _, _, ok := parsePriceAttr("LAST@yesterday"); ok {
		t.Error("a date that is not a date must not parse")
	}
}

func TestCompletenessTargetAndDate(t *testing.T) {
	if e, tbl, ok := ParseTarget("price:completeness"); !ok || e != "price" || tbl != completenessTarget {
		t.Errorf("completeness target: %s %s %v", e, tbl, ok)
	}
	if _, _, ok := ParseTarget("price:somethingelse"); ok {
		t.Error("an unknown target must not parse")
	}
	// 01:30 UTC on the 27th is still the 26th in New York.
	at := time.Date(2026, 9, 27, 1, 30, 0, 0, time.UTC)
	if d := completenessDate(at, nil); d != "2026-09-26" {
		t.Errorf("default: %s", d)
	}
	if d := completenessDate(at, map[string]any{"days_back": float64(1)}); d != "2026-09-25" {
		t.Errorf("days_back: %s", d)
	}
	if d := completenessDate(at, map[string]any{"time_zone": "UTC"}); d != "2026-09-27" {
		t.Errorf("time_zone: %s", d)
	}
	if d := completenessDate(at, map[string]any{"valuation_date": "2026-09-24"}); d != "2026-09-24" {
		t.Errorf("valuation_date: %s", d)
	}
}
