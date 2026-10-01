package analytics

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"testing"
)

// TestScanForSharedOwnership_MixedCorpus exercises the scanner against a
// mix of shapes in one call, not just isolated single-column cases: two
// reports, five columns, spanning shape B/D (unique - no finding) and
// shape A/C (shared - finding), so a scanner bug that only fires on the
// first matching column or the first report would show up here.
func TestScanForSharedOwnership_MixedCorpus(t *testing.T) {
	columns := []ScannedColumn{
		{ // shape B: single 1:M hop - unique
			ReportID:   "rpt-orders-lines",
			ColumnName: "line_item_amount",
			Path:       &JoinPath{Steps: []JoinPathStep{step("1:M")}},
		},
		{ // shape D: chain of two 1:M hops - unique
			ReportID:   "rpt-orders-lines",
			ColumnName: "allocation_qty",
			Path:       &JoinPath{Steps: []JoinPathStep{step("1:M"), step("1:M")}},
		},
		{ // shape A: two M:1 hops - shared. Deliberately not the
			// customer/region story from the trace doc, so the test
			// doesn't just re-confirm the one example already worked
			// through by hand.
			ReportID:   "rpt-customer-hierarchy",
			ColumnName: "warehouse_zone_name",
			Path: &JoinPath{Steps: []JoinPathStep{
				{LeftTable: "orders", RightTable: "warehouses", Cardinality: "M:1"},
				{LeftTable: "warehouses", RightTable: "zones", Cardinality: "M:1"},
			}},
		},
		{ // shape C: M:1 then 1:M - shared, via the FIRST hop.
			ReportID:   "rpt-customer-hierarchy",
			ColumnName: "category_display_name",
			Path: &JoinPath{Steps: []JoinPathStep{
				{LeftTable: "orders", RightTable: "skus", Cardinality: "M:1"},
				{LeftTable: "skus", RightTable: "categories", Cardinality: "1:M"},
			}},
		},
		{ // 1:1 - unique
			ReportID:   "rpt-shipments-only",
			ColumnName: "shipment_carrier",
			Path:       &JoinPath{Steps: []JoinPathStep{step("1:1")}},
		},
	}

	findings := ScanForSharedOwnership(columns)

	if len(findings) != 2 {
		t.Fatalf("expected 2 findings, got %d: %+v", len(findings), findings)
	}

	sort.Slice(findings, func(i, j int) bool { return findings[i].ColumnName < findings[j].ColumnName })

	if findings[0].ReportID != "rpt-customer-hierarchy" || findings[0].ColumnName != "category_display_name" {
		t.Errorf("finding[0]: got report=%s column=%s", findings[0].ReportID, findings[0].ColumnName)
	}
	if findings[0].OffendingHop.RightTable != "skus" {
		t.Errorf("finding[0] should name the FIRST shared hop (orders->skus), got offending hop targeting %q", findings[0].OffendingHop.RightTable)
	}

	if findings[1].ReportID != "rpt-customer-hierarchy" || findings[1].ColumnName != "warehouse_zone_name" {
		t.Errorf("finding[1]: got report=%s column=%s", findings[1].ReportID, findings[1].ColumnName)
	}

	// Neither unique-ownership report should produce a finding under
	// either name.
	for _, f := range findings {
		if f.ReportID == "rpt-orders-lines" || f.ReportID == "rpt-shipments-only" {
			t.Errorf("unique-ownership report %q incorrectly flagged: %+v", f.ReportID, f)
		}
	}
}

func TestScanForSharedOwnership_EmptyCorpus(t *testing.T) {
	if findings := ScanForSharedOwnership(nil); len(findings) != 0 {
		t.Errorf("expected no findings for an empty corpus, got %+v", findings)
	}
}

// TestScanForSharedOwnership_AgreesWithRootOwnership is a BEHAVIORAL
// check only: the scanner's verdict on these two paths matches what
// RootOwnership() itself returns for them. It does NOT prove the scanner
// calls RootOwnership() rather than reimplementing the same switch
// locally - a hand-copied reimplementation would pass this test exactly
// as well, since it's indistinguishable from the real thing by output
// alone. See TestOwnershipScanner_SourceCallsRootOwnership below for the
// actual drift-proof check; this test is kept because behavioral
// coverage is still useful, just not for the claim its old name made.
func TestScanForSharedOwnership_AgreesWithRootOwnership(t *testing.T) {
	sharedPath := &JoinPath{Steps: []JoinPathStep{step("M:1")}}
	uniquePath := &JoinPath{Steps: []JoinPathStep{step("1:M")}}

	if sharedPath.RootOwnership() != "shared" || uniquePath.RootOwnership() != "unique" {
		t.Fatal("test setup: RootOwnership itself disagrees with the fixture's intent")
	}

	findings := ScanForSharedOwnership([]ScannedColumn{
		{ReportID: "r1", ColumnName: "c1", Path: sharedPath},
		{ReportID: "r2", ColumnName: "c2", Path: uniquePath},
	})

	if len(findings) != 1 || findings[0].ReportID != "r1" {
		t.Fatalf("expected exactly one finding, matching RootOwnership's own verdict, got: %+v", findings)
	}
}

// TestOwnershipScanner_SourceCallsAnalyze is a structural check, not a
// behavioral one: it parses ownership_scanner.go's actual source and
// asserts a call to JoinPath.Analyze() (or, failing that, RootOwnership/
// TraversalCardinality, its two thin wrappers) is present in it. This is
// the test that would catch someone replacing ScanForSharedOwnership's
// call with a local reimplementation of the same cardinality fold "for
// performance" or "to avoid the extra hop" - a change that behavioral
// tests (including the one above) cannot distinguish from the real
// thing, since a faithful reimplementation produces identical output
// right up until the fold's own rule changes and the copy doesn't.
//
// This test itself already caught one such drift, harmlessly: when
// ScanForSharedOwnership moved from two separate RootOwnership()/
// TraversalCardinality() calls to one Analyze() call (to stop computing
// the same fold twice per column), the OLD version of this test - which
// checked specifically for a RootOwnership() call - correctly went red,
// because ownership_scanner.go genuinely no longer calls it directly.
// That's the test doing its job, not a false alarm; it needed updating to
// match the new (still real, still non-reimplemented) call site.
func TestOwnershipScanner_SourceCallsAnalyze(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "ownership_scanner.go", nil, 0)
	if err != nil {
		t.Fatalf("parsing ownership_scanner.go: %v", err)
	}

	recognized := map[string]bool{"Analyze": true, "RootOwnership": true, "TraversalCardinality": true}
	called := false
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if ok && recognized[sel.Sel.Name] {
			called = true
		}
		return true
	})

	if !called {
		t.Fatal("ownership_scanner.go no longer calls Analyze()/RootOwnership()/TraversalCardinality() anywhere - " +
			"if the ownership/cardinality predicate was reimplemented locally instead, a future " +
			"change to Analyze's classification rule would silently stop being " +
			"reflected in scan results")
	}
}

// TestLoadScannedColumnsFromFixture exercises every axis the fixture was
// seeded to cover: shared ownership with a bare dimension (exposed, not
// at risk), shared ownership with SUM (at risk), shared ownership with
// MAX (exposed, not at risk - the negative case Finding 2 of the roll-up
// table called for), M:M (shared AND root-grain-corrupted, distinct from
// a plain M:1 finding), and an unclassifiable hop (unresolved, not
// silently "unique").
func TestLoadScannedColumnsFromFixture(t *testing.T) {
	columns, err := LoadScannedColumnsFromFixture("testdata/ownership_fixture_sample.json")
	if err != nil {
		t.Fatalf("LoadScannedColumnsFromFixture failed: %v", err)
	}
	if len(columns) != 9 {
		t.Fatalf("expected 9 columns in the sample fixture, got %d", len(columns))
	}

	findings := ScanForSharedOwnership(columns)
	if len(findings) != 6 {
		t.Fatalf("expected 6 findings from the sample fixture, got %d: %+v", len(findings), findings)
	}

	byColumn := make(map[string]OwnershipFinding, len(findings))
	for _, f := range findings {
		byColumn[f.ColumnName] = f
	}

	// warehouse_zone_name is the ONLY shared-ownership column with no
	// fan-out at all (shape A: two M:1 hops, nothing 1:N anywhere) - the
	// case that separates Cardinality from Ownership entirely.
	// category_revenue_total/category_max_price/category_distinct_customers
	// are shape C (root -(N:1)-> a -(1:N)-> b): the SECOND hop fans out
	// in the traversal direction exactly like "M:M" does, so Cardinality
	// must be "many" for all three, not just for tag_label's literal
	// "M:M" hop - that's the predicate an earlier round of this test got
	// wrong (as a bool that could only ever be true/false, not
	// "unresolved" - see external_region_name below, where the single
	// unrecognized hop makes BOTH axes "unresolved", not just Ownership).
	cases := []struct {
		column      string
		ownership   string
		atRisk      bool
		cardinality string
	}{
		{"warehouse_zone_name", "shared", false, "one"},            // shape A: shared, ZERO fan-out
		{"category_revenue_total", "shared", true, "many"},         // shape C, SUM - at risk, AND fans out
		{"category_max_price", "shared", false, "many"},            // shape C, MAX - exposed/fans out, NOT at risk
		{"category_distinct_customers", "shared", false, "many"},   // shape C, COUNT_DISTINCT - fans out, NOT at risk (immune to duplication)
		{"tag_label", "shared", true, "many"},                      // M:M, COUNT - at risk AND fans out
		{"external_region_name", "unresolved", true, "unresolved"}, // unclassifiable hop, SUM - at risk, not "unique"/"one"
	}
	for _, c := range cases {
		f, ok := byColumn[c.column]
		if !ok {
			t.Errorf("expected a finding for column %q, got none (findings: %+v)", c.column, findings)
			continue
		}
		if f.Ownership != c.ownership {
			t.Errorf("%s: expected Ownership %q, got %q", c.column, c.ownership, f.Ownership)
		}
		if f.AtRisk != c.atRisk {
			t.Errorf("%s: expected AtRisk %v, got %v", c.column, c.atRisk, f.AtRisk)
		}
		if f.Cardinality != c.cardinality {
			t.Errorf("%s: expected Cardinality %q, got %q", c.column, c.cardinality, f.Cardinality)
		}
	}

	// line_item_amount, allocation_qty, shipment_carrier are all unique
	// ownership and must not appear as findings at all.
	for _, safe := range []string{"line_item_amount", "allocation_qty", "shipment_carrier"} {
		if _, ok := byColumn[safe]; ok {
			t.Errorf("column %q has unique ownership and must not be flagged", safe)
		}
	}
}

func TestLoadScannedColumnsFromFixture_MissingFile(t *testing.T) {
	if _, err := LoadScannedColumnsFromFixture("testdata/does_not_exist.json"); err == nil {
		t.Fatal("expected an error for a missing fixture file, got nil")
	}
}
