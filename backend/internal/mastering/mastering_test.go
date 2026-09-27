package mastering

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
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
	d, issues := Survive(contribs, rules, map[string]any{"aum": 60.0}, now)

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
		map[string]SurvivalRule{"domicile": {Strategy: "SOURCE_PRIORITY", Priority: []string{"BLOOMBERG"}, StalenessSec: 3600}}, nil, now)
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
	d, _ := Survive(c, map[string]SurvivalRule{"inception_date": {Strategy: "PROVIDER_AUTHORITATIVE", Priority: []string{"BLOOMBERG"}}}, nil, now)
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
