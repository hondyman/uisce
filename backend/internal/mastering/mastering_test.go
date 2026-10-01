package mastering

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/hondyman/uisce/backend/internal/rules/vm"
)

func productProfile(t *testing.T) *Profile {
	t.Helper()
	ident := "mdm.product_identifier"
	p := &Profile{EntityCd: "PRODUCT", BOKey: "product", TablePrefix: "product", AnchorTable: "mdm.product",
		AnchorCodeColumn: "product_cd", IdentifierTable: &ident, IncomingTable: "staging.product_incoming", CodePrefix: "PRD-",
		RawSettings: json.RawMessage(`{"defaults": {"status_cd": "LIVE"}, "record_columns": {"product_type_cd": "product_type_cd"},
			"references": [{"column": "product_type_id", "attribute": "product_type_cd", "required": true,
			"ref_table": "mdm.product_type", "ref_code_column": "type_cd", "map_table": "mdm.product_type_mapping",
			"map_source_column": "mdm_source_system_id", "map_vendor_column": "vendor_type_cd", "map_code_column": "internal_type_cd"}]}`)}
	if err := p.decode(); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestProfileRejectsUnsafeNames(t *testing.T) {
	p := productProfile(t)
	if p.table("golden_record") != `"mdm"."product_golden_record"` || p.keyColumn() != `"product_id"` {
		t.Errorf("tables: %s %s", p.table("golden_record"), p.keyColumn())
	}
	bad := *p
	bad.AnchorTable = "mdm.product; drop table x"
	if err := bad.decode(); err == nil {
		t.Error("an unsafe table name must be refused")
	}
	bad = *p
	bad.RawSettings = json.RawMessage(`{"references": [{"column": "x\"", "attribute": "a", "ref_table": "mdm.t", "ref_code_column": "c"}]}`)
	if err := bad.decode(); err == nil {
		t.Error("an unsafe reference column must be refused")
	}
}

func TestCanonicalRecord(t *testing.T) {
	c := &Canonicalizer{
		Profile: productProfile(t),
		Binding: map[string]string{
			"ProductName": "fund_name", "ProductBaseCurrency": "base_currency", "ProductTypeId": "fund_type",
			"ProductInceptionDate": "inception_date", "ProductShortName": "fund_name_short", "Unmapped": "ticker",
			"id:ISIN": "isin", "id:SEDOL": "sedol", "@source_key": "fsym_id",
		},
		FieldAttr: map[string]string{"ProductName": "name", "ProductBaseCurrency": "base_currency",
			"ProductTypeId": "product_type_id", "ProductInceptionDate": "inception_date", "ProductShortName": "short_name"},
		ColumnTypes: map[string]string{"inception_date": "date"},
	}
	at := time.Date(2026, 9, 25, 14, 9, 45, 0, time.UTC)
	rec := c.Record(map[string]any{
		"fund_name": []byte(" Uisce Global Equity Income Fund "), "base_currency": "EUR", "fund_type": "ETF",
		"inception_date": time.Date(2013, 1, 17, 0, 0, 0, 0, time.UTC), "fund_name_short": "",
		"isin": "ie0019722233", "sedol": "", "fsym_id": "H29PZ16-R", "ticker": "FND000-IE",
	}, at)
	if !rec.Valid() || rec.SourceKey != "H29PZ16-R" || !rec.AsOf.Equal(at) {
		t.Fatalf("record: %+v", rec)
	}
	want := map[string]any{"name": "Uisce Global Equity Income Fund", "base_currency": "EUR", "product_type_cd": "ETF", "inception_date": "2013-01-17"}
	if len(rec.Attrs) != len(want) {
		t.Errorf("attrs %v, want %v (empty is absent; a reference arrives under its code attribute)", rec.Attrs, want)
	}
	for k, v := range want {
		if rec.Attrs[k] != v {
			t.Errorf("%s = %v, want %v", k, rec.Attrs[k], v)
		}
	}
	if rec.Identifiers["ISIN"] != "IE0019722233" || len(rec.Identifiers) != 1 {
		t.Errorf("identifiers %v: upper-cased, empty skipped", rec.Identifiers)
	}
	if got := c.Unmastered(); len(got) != 1 || got[0] != "Unmapped" {
		t.Errorf("unmastered %v", got)
	}

	delete(c.Binding, "@source_key")
	if rec := c.Record(map[string]any{"fund_name": "x"}, at); rec.Valid() {
		t.Error("a record with no source key can't be linked and must be rejected")
	}
}

// A staging binding may name its fields by golden attribute instead of BO
// field (staging.security_data and staging.party_data do). Those keys must
// still reach the golden record: when they did not, mint() rejected every
// security record for a missing security_name and the run reported COMPLETED
// with zero published.
func TestCanonicalRecordAttributeNamedBinding(t *testing.T) {
	c := &Canonicalizer{
		Profile: productProfile(t),
		Binding: map[string]string{
			"name": "fund_name", "base_currency": "base_currency",
			"id:ISIN": "isin", "@source_key": "fsym_id",
		},
		FieldAttr: map[string]string{
			"ProductName": "name", "ProductBaseCurrency": "base_currency",
			// both key spellings, as prepare() builds them
			"name": "name", "base_currency": "base_currency",
		},
		ColumnTypes: map[string]string{},
	}
	rec := c.Record(map[string]any{
		"fund_name": "Uisce Global Equity Income Fund", "base_currency": "EUR",
		"isin": "ie0019722233", "fsym_id": "H29PZ16-R",
	}, time.Date(2026, 9, 25, 14, 9, 45, 0, time.UTC))
	if !rec.Valid() {
		t.Fatalf("record rejected: %v", rec.Issues)
	}
	if rec.Attrs["name"] != "Uisce Global Equity Income Fund" {
		t.Errorf("name = %v, want the fund name", rec.Attrs["name"])
	}
	if rec.Attrs["base_currency"] != "EUR" {
		t.Errorf("base_currency = %v, want EUR", rec.Attrs["base_currency"])
	}
}

func TestExtractCol(t *testing.T) {
	jsonBytes := []byte(`{"esg_score": 85.5, "lot_size": 100, "lei": "5493000M6T3", "details": {"callable": true}}`)
	jsonStr := string(jsonBytes)
	jsonMap := map[string]any{
		"esg_score": 85.5,
		"lot_size":  float64(100),
		"lei":       "5493000M6T3",
		"details":   map[string]any{"callable": true},
	}

	tests := []struct {
		name     string
		row      map[string]any
		col      string
		expected any
	}{
		{
			name:     "plain column",
			row:      map[string]any{"security_name": "Apple Inc."},
			col:      "security_name",
			expected: "Apple Inc.",
		},
		{
			name:     "json bytes extraction",
			row:      map[string]any{"custom_attributes": jsonBytes},
			col:      "custom_attributes->>'esg_score'",
			expected: 85.5,
		},
		{
			name:     "json string extraction",
			row:      map[string]any{"custom_attributes": jsonStr},
			col:      "custom_attributes->>'lei'",
			expected: "5493000M6T3",
		},
		{
			name:     "json map extraction",
			row:      map[string]any{"custom_attributes": jsonMap},
			col:      "custom_attributes->>'lot_size'",
			expected: float64(100),
		},
		{
			name:     "nested json path extraction",
			row:      map[string]any{"custom_attributes": jsonBytes},
			col:      "custom_attributes->'details'->>'callable'",
			expected: true,
		},
		{
			name:     "missing key in json",
			row:      map[string]any{"custom_attributes": jsonBytes},
			col:      "custom_attributes->>'nonexistent'",
			expected: nil,
		},
		{
			name:     "missing base column",
			row:      map[string]any{"other_col": "abc"},
			col:      "custom_attributes->>'esg_score'",
			expected: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := extractCol(tc.row, tc.col)
			if got != tc.expected {
				t.Errorf("extractCol(%v, %q) = %v (%T), want %v (%T)", tc.row, tc.col, got, got, tc.expected, tc.expected)
			}
		})
	}
}

func TestCanonicalRecordJSONPathBinding(t *testing.T) {
	c := &Canonicalizer{
		Profile: productProfile(t),
		Binding: map[string]string{
			"@source_key": "sec_id",
			"id:LEI":      "custom_attributes->>'lei'",
			"esg_score":   "custom_attributes->>'esg_score'",
			"lot_size":    "custom_attributes->>'lot_size'",
			"name":        "sec_name",
		},
		FieldAttr: map[string]string{
			"name":      "name",
			"esg_score": "esg_score",
			"lot_size":  "lot_size",
		},
		ColumnTypes: map[string]string{
			"custom_attributes->>'esg_score'": "numeric",
			"custom_attributes->>'lot_size'":  "integer",
		},
	}

	at := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	row := map[string]any{
		"sec_id":            "SEC-001",
		"sec_name":          "Apple Global Corp",
		"custom_attributes": `{"esg_score": 88.5, "lot_size": 1000, "lei": "5493006M6T3X75N"}`,
	}

	rec := c.Record(row, at)
	if !rec.Valid() {
		t.Fatalf("record rejected: %v", rec.Issues)
	}

	if rec.SourceKey != "SEC-001" {
		t.Errorf("source key = %v, want SEC-001", rec.SourceKey)
	}
	if rec.Attrs["name"] != "Apple Global Corp" {
		t.Errorf("name = %v, want Apple Global Corp", rec.Attrs["name"])
	}
	if rec.Attrs["esg_score"] != 88.5 {
		t.Errorf("esg_score = %v, want 88.5", rec.Attrs["esg_score"])
	}
	if rec.Attrs["lot_size"] != int64(1000) && rec.Attrs["lot_size"] != 1000.0 && rec.Attrs["lot_size"] != 1000 {
		t.Errorf("lot_size = %v, want 1000", rec.Attrs["lot_size"])
	}
	if rec.Identifiers["LEI"] != "5493006M6T3X75N" {
		t.Errorf("identifier LEI = %v, want 5493006M6T3X75N", rec.Identifiers["LEI"])
	}
}


func TestNormalizeNumbers(t *testing.T) {
	if v := normalize([]byte("3567039317.1100"), "numeric"); v != 3567039317.11 {
		t.Errorf("numeric: %v", v)
	}
	if v := normalize([]byte("201071364"), "character varying"); v != "201071364" {
		t.Errorf("a code stays text: %v (%T)", v, v)
	}
}

func TestSurvivePriorityStalenessAndProvenance(t *testing.T) {
	now := time.Date(2026, 9, 27, 0, 0, 0, 0, time.UTC)
	fresh, old := now.Add(-time.Hour), now.Add(-72*time.Hour)
	contribs := []Contribution{
		{SourceID: "f", SourceCd: "FACTSET", SourceKey: "F1", AsOf: fresh, Attrs: map[string]any{"name": "Global Equity Fund", "base_currency": "EUR", "aum": 100.0}},
		{SourceID: "b", SourceCd: "BLOOMBERG", SourceKey: "B1", AsOf: old, Attrs: map[string]any{"name": "Global Equity Fd", "base_currency": "EUR"}},
		{SourceID: "r", SourceCd: "REFINITIV", SourceKey: "R1", AsOf: fresh, Attrs: map[string]any{"name": "Global Equity Fund"}},
	}
	rules := map[string]SurvivalRule{
		"name":          {ID: "r-name", Strategy: "SOURCE_PRIORITY", Priority: []string{"BLOOMBERG", "REFINITIV", "FACTSET"}, StalenessSec: 86400},
		"base_currency": {Strategy: "SOURCE_PRIORITY", Priority: []string{"BLOOMBERG", "REFINITIV", "FACTSET"}, StalenessSec: 86400},
		"aum":           {Strategy: "MOST_RECENT", TolerancePct: 25},
	}
	d, issues := Survive(contribs, rules, map[string]any{"aum": 60.0}, now, nil)

	// Bloomberg ranks first but is stale: the fresh Refinitiv value wins.
	if d["name"].Value != "Global Equity Fund" || d["name"].Winner.SourceCd != "REFINITIV" || d["name"].RuleID != "r-name" {
		t.Errorf("name: %+v", d["name"])
	}
	if len(d["name"].Competing) != 3 || d["name"].Confidence != 0.6667 {
		t.Errorf("provenance keeps every competing value and the agreement: %+v", d["name"])
	}
	// Bloomberg is ranked first but stale; FactSet is the only fresh source.
	if d["base_currency"].Winner.SourceCd != "FACTSET" {
		t.Errorf("base_currency: %+v", d["base_currency"])
	}
	// AUM moved 67% against a 25% tolerance: held at the previous value.
	if !d["aum"].Held || d["aum"].Value != 60.0 || len(issues) != 1 || issues[0].Code != "ANOMALY" {
		t.Errorf("aum: %+v issues %+v", d["aum"], issues)
	}
}

func TestSurviveAllStaleStillSurvives(t *testing.T) {
	now := time.Now()
	d, _ := Survive([]Contribution{{SourceCd: "FACTSET", AsOf: now.Add(-48 * time.Hour), Attrs: map[string]any{"domicile": "IE"}}},
		map[string]SurvivalRule{"domicile": {Strategy: "SOURCE_PRIORITY", Priority: []string{"BLOOMBERG"}, StalenessSec: 3600}}, nil, now, nil)
	if d["domicile"].Value != "IE" || !strings.Contains(d["domicile"].Reason, "stale") || d["domicile"].Confidence != 0.5 {
		t.Errorf("an old feed must not blank a value, but is flagged: %+v", d["domicile"])
	}
}

func TestSurviveAuthoritative(t *testing.T) {
	now := time.Now()
	c := []Contribution{
		{SourceCd: "FACTSET", AsOf: now, Attrs: map[string]any{"inception_date": "2013-01-17"}},
		{SourceCd: "BLOOMBERG", AsOf: now.Add(-time.Minute), Attrs: map[string]any{"inception_date": "2013-01-18"}},
	}
	d, _ := Survive(c, map[string]SurvivalRule{"inception_date": {Strategy: "PROVIDER_AUTHORITATIVE", Priority: []string{"BLOOMBERG"}}}, nil, now, nil)
	if d["inception_date"].Winner.SourceCd != "BLOOMBERG" {
		t.Errorf("the authoritative provider wins over a more recent one: %+v", d["inception_date"])
	}
}

func TestFuzzyScore(t *testing.T) {
	keys := []FuzzyKey{{Field: "name", Method: "trigram", Weight: 0.7}, {Field: "base_currency", Method: "exact", Weight: 0.15}, {Field: "domicile", Method: "exact", Weight: 0.15}}
	rec := map[string]any{"name": "Uisce Euro Bond", "base_currency": "EUR", "domicile": "LU"}
	cand := map[string]any{"name": "present", "base_currency": "eur", "domicile": "IE"}
	if s := FuzzyScore(keys, rec, cand, map[string]float64{"name": 0.9}); s != 0.78 {
		t.Errorf("score %v", s)
	}
	// A comparison one side lacks is left out, weights renormalized.
	if s := FuzzyScore(keys, map[string]any{"name": "x", "base_currency": "EUR"}, cand, map[string]float64{"name": 0.9}); s != 0.9176 {
		t.Errorf("sparse score %v", s)
	}
	// No name, no match: sharing a currency and a domicile is not identity.
	if s := FuzzyScore(keys, map[string]any{"base_currency": "EUR", "domicile": "IE"}, cand, nil); s != 0 {
		t.Errorf("no-name score %v", s)
	}
	mr := MatchRule{ThresholdAuto: 0.95, ThresholdReview: 0.8}
	if mr.Classify(0.96) != MatchFuzzy || mr.Classify(0.85) != MatchReview || mr.Classify(0.5) != MatchNew {
		t.Error("classify")
	}
}

func TestRulesAtEachStage(t *testing.T) {
	var rs RuleSet
	// Canonical skips a rule whose field the source doesn't send; golden runs it on null.
	rs.rules = []rule{{id: "r1", name: "name present", severity: "BLOCK", fields: []string{"ProductName"}}}
	if got := rs.Canonical(map[string]any{"ProductBaseCurrency": "EUR"}); len(got) != 0 {
		t.Errorf("canonical: a source that doesn't send the field can't fail the rule: %+v", got)
	}
	if !hasAll(map[string]any{"a": 1}, []string{"a"}) || hasAll(map[string]any{"a": nil}, []string{"a"}) {
		t.Error("hasAll")
	}
}

func TestDQScore(t *testing.T) {
	rules := map[string]SurvivalRule{"name": {}, "domicile": {}, "legal_name": {}, "aum": {}}
	attrs := map[string]any{"name": "x", "domicile": "IE", "aum": 1.0}
	if s := dqScore(attrs, rules, nil); s != 75 {
		t.Errorf("3 of 4 mastered attributes: %v", s)
	}
	if s := dqScore(attrs, rules, []Issue{{Severity: SevWarning}}); s != 65 {
		t.Errorf("a warning costs 10: %v", s)
	}
}

func TestSameJSONIsOrderFree(t *testing.T) {
	if !sameJSON(map[string]any{"a": 1.0, "b": "x"}, map[string]any{"b": "x", "a": 1.0}) {
		t.Error("key order must not create a new version")
	}
}

func TestParseTarget(t *testing.T) {
	if e, tb, ok := ParseTarget("Product:staging.ff_product"); !ok || e != "product" || tb != "staging.ff_product" {
		t.Errorf("got %q %q %v", e, tb, ok)
	}
	for _, bad := range []string{"product", "product:", ":staging.x", "product:mdm.product", "product:staging.x;drop", "product:staging.X"} {
		if _, _, ok := ParseTarget(bad); ok {
			t.Errorf("%q must be refused", bad)
		}
	}
}

func TestCountsAdd(t *testing.T) {
	var c Counts
	c.add(Counts{Records: 22, Published: 9, Invalid: 13})
	c.add(Counts{Records: 22, Unchanged: 9, Invalid: 13})
	if c.Records != 44 || c.Published != 9 || c.Unchanged != 9 || c.Invalid != 26 {
		t.Errorf("%+v", c)
	}
}

func TestOverridableAttributes(t *testing.T) {
	p := productProfile(t)
	anchor := map[string]string{"id": "uuid", "product_cd": "character varying", "name": "character varying", "manager_id": "uuid", "aum": "numeric"}
	for attr, want := range map[string]bool{"name": true, "aum": true, "product_type_cd": true, "id": false, "product_cd": false, "tenant_id": false, "manager_id": false} {
		if got := overridable(p, anchor, attr); got != want {
			t.Errorf("%s: got %v", attr, got)
		}
	}
}

func TestJSONTextMatchesPostgres(t *testing.T) {
	for v, want := range map[any]string{"EUR": "EUR", 3567039317.11: "3567039317.11", true: "true"} {
		if got := jsonText(v); got != want {
			t.Errorf("%v: got %q want %q", v, got, want)
		}
	}
}

func TestDefaultHierarchyAndNewStrategies(t *testing.T) {
	now := time.Now()
	c := []Contribution{
		{SourceCd: "FACTSET", AsOf: now, Attrs: map[string]any{"name": "Uisce Global Equity Income Fund", "domicile": "IE", "manager_name": "Uisce AM"}},
		{SourceCd: "BLOOMBERG", AsOf: now.Add(-time.Minute), Attrs: map[string]any{"name": "UISCE GLOBAL EQ INC", "domicile": "IE", "manager_name": "Uisce Asset Management"}},
		{SourceCd: "REFINITIV", AsOf: now.Add(-2 * time.Minute), Attrs: map[string]any{"name": "Uisce Global Equity Income", "domicile": "LU", "manager_name": "Uisce AM"}},
	}
	hier := func(attr string) []string {
		if attr == "name" {
			return []string{"REFINITIV", "BLOOMBERG", "FACTSET"}
		}
		return []string{"BLOOMBERG", "REFINITIV", "FACTSET"}
	}
	d, _ := Survive(c, map[string]SurvivalRule{
		"domicile":     {Strategy: "MOST_FREQUENT"},
		"manager_name": {Strategy: "MOST_COMPLETE"},
	}, nil, now, &SurviveOptions{Hierarchy: hier})
	// No rule: the entity hierarchy decides, not recency.
	if d["name"].Winner.SourceCd != "REFINITIV" || !strings.Contains(d["name"].Reason, "entity hierarchy") {
		t.Errorf("name: %+v", d["name"])
	}
	if d["domicile"].Value != "IE" || !strings.Contains(d["domicile"].Reason, "2 of 3") {
		t.Errorf("most frequent: %+v", d["domicile"])
	}
	if d["manager_name"].Value != "Uisce Asset Management" {
		t.Errorf("most complete: %+v", d["manager_name"])
	}
	// No rule and no hierarchy: recency, as before.
	d, _ = Survive(c, nil, nil, now, nil)
	if d["name"].Winner.SourceCd != "FACTSET" {
		t.Errorf("no hierarchy: %+v", d["name"])
	}
}

// A price-style gate: Bloomberg first, unless it is more than 20% from the
// median of the other sources - written as a selection rule on the rule
// engine, evaluated through selector().
func TestSelectionRuleConsensusGate(t *testing.T) {
	few, _ := vm.ParseExpression("COUNT(peers.value) < 2")
	near, _ := vm.ParseExpression("ABS(value - MEDIAN(peers.value)) <= 0.2 * MEDIAN(peers.value)")
	gate := vm.RuleNode{Type: vm.NodeTypeGroup, Group: &vm.RuleGroup{Operator: "OR", Conditions: []vm.RuleNode{
		{Type: vm.NodeTypeExpression, Expression: few}, {Type: vm.NodeTypeExpression, Expression: near}}}}
	sel := selector(map[string]rule{"r-gate": {id: "r-gate", name: "price.consensus_20pct", node: gate, version: "2026-09-27T00:00:00Z"}})
	now := time.Now()
	prices := func(bbg float64) []Contribution {
		return []Contribution{
			{SourceCd: "BLOOMBERG", SourceKey: "B", AsOf: now, Attrs: map[string]any{"price": bbg}},
			{SourceCd: "REFINITIV", SourceKey: "R", AsOf: now, Attrs: map[string]any{"price": 100.0}},
			{SourceCd: "FACTSET", SourceKey: "F", AsOf: now, Attrs: map[string]any{"price": 102.0}},
		}
	}
	priceRule := SurvivalRule{ID: "s-price", Strategy: "SOURCE_PRIORITY", Priority: []string{"BLOOMBERG", "REFINITIV", "FACTSET"}, SelectionRuleID: "r-gate"}
	opts := &SurviveOptions{Select: sel}

	d, issues := Survive(prices(101), map[string]SurvivalRule{"price": priceRule}, nil, now, opts)
	if d["price"].Winner.SourceCd != "BLOOMBERG" || len(issues) != 0 {
		t.Errorf("in line with the others: %+v %+v", d["price"], issues)
	}

	d, issues = Survive(prices(128), map[string]SurvivalRule{"price": priceRule}, nil, now, opts)
	if d["price"].Winner.SourceCd != "REFINITIV" || !strings.Contains(d["price"].Reason, "excluded BLOOMBERG") {
		t.Errorf("outlier excluded, next ranked wins: %+v", d["price"])
	}
	for _, c := range d["price"].Competing {
		if c.SourceCd == "BLOOMBERG" && (c.Selected == nil || *c.Selected || !strings.Contains(c.Note, "price.consensus_20pct")) {
			t.Errorf("the excluded candidate keeps its reason: %+v", c)
		}
	}

	flag := priceRule
	flag.SelectionMode = "FLAG"
	d, issues = Survive(prices(128), map[string]SurvivalRule{"price": flag}, nil, now, opts)
	if d["price"].Winner.SourceCd != "BLOOMBERG" || len(issues) != 1 || issues[0].Code != IssueSelectionFlag {
		t.Errorf("flag only: %+v %+v", d["price"], issues)
	}

	// Nobody passes a rule that fails for all: hold the previous value.
	never, _ := vm.ParseExpression("value < 0")
	hold := SurvivalRule{ID: "s", Strategy: "SOURCE_PRIORITY", SelectionRuleID: "r-never"}
	selNever := selector(map[string]rule{"r-never": {id: "r-never", name: "never", node: vm.RuleNode{Type: vm.NodeTypeExpression, Expression: never}}})
	d, issues = Survive(prices(101), map[string]SurvivalRule{"price": hold}, map[string]any{"price": 99.5}, now, &SurviveOptions{Select: selNever})
	if !d["price"].Held || d["price"].Value != 99.5 || len(issues) != 1 || issues[0].Code != IssueSelectionHold {
		t.Errorf("hold: %+v %+v", d["price"], issues)
	}
	allow := hold
	allow.OnNoneSelected = "ALLOW"
	d, issues = Survive(prices(101), map[string]SurvivalRule{"price": allow}, map[string]any{"price": 99.5}, now, &SurviveOptions{Select: selNever})
	if d["price"].Held || d["price"].Value != 101.0 || len(issues) != 1 || issues[0].Code != IssueSelectionFlag {
		t.Errorf("allow: %+v %+v", d["price"], issues)
	}

	// An unknown or broken rule never reads as a pass.
	d, _ = Survive(prices(101), map[string]SurvivalRule{"price": {SelectionRuleID: "missing"}}, map[string]any{"price": 99.5}, now, opts)
	if !d["price"].Held {
		t.Errorf("a rule that can't be evaluated must not pass: %+v", d["price"])
	}
}

// The context a selection rule reads.
func TestSelectionContextData(t *testing.T) {
	now := time.Now()
	c := Candidate{SourceCd: "BLOOMBERG", SourceKey: "B", Value: 101.0, AsOf: now.Add(-2 * time.Hour), record: map[string]any{"asset_class": "Equity"}}
	p := Candidate{SourceCd: "REFINITIV", Value: 100.0, AsOf: now}
	d := SelectionContext{Candidate: c, Peers: []Candidate{p}, All: []Candidate{c, p}, Previous: 99.0, Now: now, Ranking: []string{"BLOOMBERG", "REFINITIV"}}.Data()
	if d["value"] != 101.0 || d["age_hours"] != 2.0 || d["rank"] != 1.0 || d["has_previous"] != true || len(d["peers"].([]any)) != 1 {
		t.Errorf("context: %+v", d)
	}
	if d["record"].(map[string]any)["asset_class"] != "Equity" {
		t.Errorf("record: %+v", d["record"])
	}
}

// The same gate as one expression, with the "too few peers" half as the
// rule's minimum-peers setting.
func TestSelectionMinPeers(t *testing.T) {
	near, err := vm.ParseExpression("ABS(value - MEDIAN(peers.value)) <= 0.2 * MEDIAN(peers.value)")
	if err != nil {
		t.Fatal(err)
	}
	sel := selector(map[string]rule{"g": {id: "g", name: "price.consensus_20pct", node: vm.RuleNode{Type: vm.NodeTypeExpression, Expression: near}}})
	now := time.Now()
	gate := SurvivalRule{Strategy: "SOURCE_PRIORITY", Priority: []string{"BLOOMBERG", "REFINITIV"}, SelectionRuleID: "g", SelectionMinPeers: 2}
	two := []Contribution{
		{SourceCd: "BLOOMBERG", AsOf: now, Attrs: map[string]any{"price": 128.0}},
		{SourceCd: "REFINITIV", AsOf: now, Attrs: map[string]any{"price": 100.0}},
	}
	d, issues := Survive(two, map[string]SurvivalRule{"price": gate}, nil, now, &SurviveOptions{Select: sel})
	if d["price"].Winner.SourceCd != "BLOOMBERG" || len(issues) != 0 || !strings.Contains(d["price"].Reason, "not applied") {
		t.Errorf("one peer: the gate doesn't apply: %+v %+v", d["price"], issues)
	}
	three := append(two, Contribution{SourceCd: "FACTSET", AsOf: now, Attrs: map[string]any{"price": 101.0}})
	d, _ = Survive(three, map[string]SurvivalRule{"price": gate}, nil, now, &SurviveOptions{Select: sel})
	if d["price"].Winner.SourceCd != "REFINITIV" {
		t.Errorf("two peers: the outlier is excluded: %+v", d["price"])
	}
}

func TestProfileLayouts(t *testing.T) {
	p := productProfile(t)
	if p.entityCol() != "id" || p.current("a") != "true" || p.bitemporal() {
		t.Errorf("product anchor: %s %s", p.entityCol(), p.current("a"))
	}
	id := p.ident()
	if id.KeyColumn != "product_id" || id.identActive("i") != `i."effective_to" IS NULL` || id.identRetire() != `"effective_to" = CURRENT_DATE` {
		t.Errorf("product identifiers: %+v %s", id, id.identActive("i"))
	}
	s := *p
	s.RawSettings = json.RawMessage(`{"entity_id_column": "master_id", "versioning": "bitemporal",
		"identifiers": {"key_column": "security_id", "type_column": "id_type", "value_column": "id_value",
		"source_column": "source_system_id", "source_is_id": true, "active_column": "is_valid", "active_is_flag": true}}`)
	if err := s.decode(); err != nil {
		t.Fatal(err)
	}
	if s.entityCol() != "master_id" || s.current("a") != `a."valid_to" IS NULL` {
		t.Errorf("security anchor: %s %s", s.entityCol(), s.current("a"))
	}
	si := s.ident()
	if si.identActive("") != `"is_valid"` || si.identRetire() != `"is_valid" = false` || !si.SourceIsID {
		t.Errorf("security identifiers: %+v", si)
	}
	bad := *p
	bad.RawSettings = json.RawMessage(`{"versioning": "sideways"}`)
	if err := bad.decode(); err == nil {
		t.Error("an unknown versioning must be refused")
	}
}
