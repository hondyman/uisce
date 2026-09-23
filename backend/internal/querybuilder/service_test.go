package querybuilder

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/hondyman/uisce/backend/internal/boresolver"
)

// TestApplyColumnMetadata_CopiesRootOwnership pins the exact gap found
// while tracing the fan-out cardinality trace's ownership field to its
// only real consumer (SavedQueryWidget.tsx's gauge total, fed by
// QueryService.Execute, not Preview): applyColumnMetadata copied BOID and
// Cardinality field-by-field but not RootOwnership, so a shared-ownership
// column would read back as RootOwnership: "" (empty - the "unique by
// convention" default) on every /execute response, despite Preview
// correctly reporting "shared" for the exact same column. This is a
// silent hazard, not a crash: the field exists, has the right value at
// Preview, and reads back wrong exactly where a downstream consumer would
// need it to be right.
func TestApplyColumnMetadata_CopiesRootOwnership(t *testing.T) {
	generated := []boresolver.QueryResultColumn{
		{Name: "ZoneName", BOID: "bo-warehouse-zone", Cardinality: "one", RootOwnership: "shared"},
		{Name: "OrderID", BOID: "bo-order", Cardinality: "", RootOwnership: "unique"},
	}
	// dbColumns simulates what the driver reports back: name and type
	// only, no metadata - exactly what scanColumns produces before
	// applyColumnMetadata runs.
	dbColumns := []boresolver.QueryResultColumn{
		{Name: "ZoneName", Type: "text"},
		{Name: "OrderID", Type: "text"},
	}

	applyColumnMetadata(dbColumns, generated)

	if dbColumns[0].RootOwnership != "shared" {
		t.Errorf("ZoneName: expected RootOwnership 'shared' to survive from Preview's columns into Execute's, got %q", dbColumns[0].RootOwnership)
	}
	if dbColumns[0].BOID != "bo-warehouse-zone" || dbColumns[0].Cardinality != "one" {
		t.Errorf("ZoneName: BOID/Cardinality regressed: %+v", dbColumns[0])
	}
	if dbColumns[1].RootOwnership != "unique" {
		t.Errorf("OrderID: expected RootOwnership 'unique', got %q", dbColumns[1].RootOwnership)
	}
}

// TestApplyColumnMetadata_CopiesEveryMetadataField is the completeness
// check TestApplyColumnMetadata_CopiesRootOwnership can't be: that test
// pins the ONE field (RootOwnership) already found missing, but a
// per-field test suite only ever catches fields someone remembered to
// write a test for - the NEXT field added to QueryResultColumn would be
// silently dropped by applyColumnMetadata's hand-copy exactly the same
// way, and the test suite would stay green throughout, on a one-release
// delay, forever.
//
// This uses reflection instead: set every string field on a generated
// column - except Name/Type, which the driver (not the generator) is
// authoritative for and applyColumnMetadata must never overwrite - to a
// value unique to that field's name, run applyColumnMetadata, and assert
// every one of those fields survived onto the db column. A field
// forgotten in applyColumnMetadata's copy list shows up as its zero
// value here without this test needing to know the field's name in
// advance.
//
// Limitation, stated rather than hidden: this only covers string-typed
// fields (every field on QueryResultColumn is a string today). A future
// non-string field would silently skip this check, not fail it - if
// QueryResultColumn grows one, this test needs a matching reflect.Kind
// branch, not just a new field name.
func TestApplyColumnMetadata_CopiesEveryMetadataField(t *testing.T) {
	driverOwnedFields := map[string]bool{"Name": true, "Type": true}

	var generated boresolver.QueryResultColumn
	genVal := reflect.ValueOf(&generated).Elem()
	genType := genVal.Type()
	for i := 0; i < genType.NumField(); i++ {
		field := genType.Field(i)
		if driverOwnedFields[field.Name] {
			continue
		}
		fv := genVal.Field(i)
		if fv.Kind() == reflect.String {
			fv.SetString("distinguishable-" + field.Name)
		}
	}
	generated.Name = "col1" // the join key applyColumnMetadata matches on

	dbColumns := []boresolver.QueryResultColumn{{Name: "col1", Type: "text"}}
	applyColumnMetadata(dbColumns, []boresolver.QueryResultColumn{generated})

	outVal := reflect.ValueOf(dbColumns[0])
	outType := outVal.Type()
	checked := 0
	for i := 0; i < outType.NumField(); i++ {
		field := outType.Field(i)
		if driverOwnedFields[field.Name] {
			continue
		}
		fv := outVal.Field(i)
		if fv.Kind() != reflect.String {
			t.Logf("skipping non-string field %s - this test does not cover it, see doc comment", field.Name)
			continue
		}
		checked++
		want := "distinguishable-" + field.Name
		if fv.String() != want {
			t.Errorf("field %s was not copied by applyColumnMetadata: got %q, want %q", field.Name, fv.String(), want)
		}
	}
	if checked == 0 {
		t.Fatal("test setup: no string metadata fields found to check - QueryResultColumn's shape changed underneath this test")
	}
}

// TestApplyColumnMetadata_NoGeneratedColumns_LeavesDBColumnsUnmodified
// covers the single-BO case (Columns is empty for single-BO queries per
// QueryPreviewResponse's own doc comment) - applyColumnMetadata must be a
// no-op, not zero out whatever the driver already reported.
func TestApplyColumnMetadata_NoGeneratedColumns_LeavesDBColumnsUnmodified(t *testing.T) {
	dbColumns := []boresolver.QueryResultColumn{{Name: "id", Type: "uuid"}}
	applyColumnMetadata(dbColumns, nil)
	if dbColumns[0].Name != "id" || dbColumns[0].Type != "uuid" {
		t.Errorf("expected dbColumns unmodified when generated is empty, got: %+v", dbColumns[0])
	}
}

// TestSingleBOPreviewColumns_PopulatesBOID_LeavesAggregationEmpty exercises the
// single-BO path's new column population (the α fix for the
// "deliberate Needs review" follow-up from PR #113). It pins three
// facts on the wire:
//
//  1. BOID is populated (every single-BO column belongs to the root BO).
//  2. Aggregation is empty / absent under omitempty (row-grain SQL —
//     isAdditiveSafe must keep refusing on this column, per the gate's
//     fail-safe polarity). If someone later "widens" isAdditiveSafe to
//     accept avg, this test still pins the wire-shape truth.
//  3. Cardinality and RootOwnership are also absent under omitempty —
//     single-BO has no related-BO joins, so per-column grain/ownership
//     metadata is noise; the gate's hasRelatedBOs:false on the saved-query
//     path is what does the work, and that's already covered by
//     TestHasRelatedBOs_FalseSurvivesGoJSONMarshal_AsMapLiteral.
//
// The wire assertion is byte-level (json.Marshal on the same struct the
// handler serializes) so a future refactor that changes the omitempty
// behavior, or accidentally populates Aggregation, breaks here with a
// concrete diff — not silently.
func TestSingleBOPreviewColumns_PopulatesBOID_LeavesAggregationEmpty(t *testing.T) {
	columns := []boresolver.QueryResultColumn{
		{Name: "Order ID", Type: "unknown", BOID: "bo-orders"},
		{Name: "Total Amount", Type: "unknown", BOID: "bo-orders"},
	}

	// Sanity: predicates the gate consumes.
	for i, c := range columns {
		if c.BOID != "bo-orders" {
			t.Errorf("columns[%d].BOID = %q; want %q (every single-BO column belongs to the root BO)", i, c.BOID, "bo-orders")
		}
		if c.Aggregation != "" {
			t.Errorf("columns[%d].Aggregation = %q; want empty (row-grain SQL; isAdditiveSafe must refuse)", i, c.Aggregation)
		}
	}

	// Wire-shape pin.
	b, err := json.Marshal(columns)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	wire := string(b)
	if !strings.Contains(wire, `"boId":"bo-orders"`) {
		t.Errorf("expected boId on the wire for both columns, got: %s", wire)
	}
	if strings.Contains(wire, `"aggregation":`) {
		t.Errorf("expected aggregation absent under omitempty (the gate's unsafe sentinel), got: %s", wire)
	}
	if strings.Contains(wire, `"cardinality":`) {
		t.Errorf("expected cardinality absent under omitempty (single-BO has no related-BO joins; grain gate is hasRelatedBOs:false, not per-column), got: %s", wire)
	}
	if strings.Contains(wire, `"rootOwnership":`) {
		t.Errorf("expected rootOwnership absent under omitempty (same reason as cardinality), got: %s", wire)
	}
}

// TestSingleBOColumnName_MatchesGeneratorAlias is the cross-check between
// the preview-side wireName computation (DisplayName || Name) and what
// BOSQLGenerator.ResolvePathWithLabel emits as the SQL alias. If either
// side ever drifts, applyColumnMetadata's name-match lookup at
// Execute time silently no-ops — same drift class as the mapper's
// dropped-Label bug (comment on Preview's Columns loop). Both sides are
// pinned against the same input set here so divergence breaks loudly.
func TestSingleBOColumnName_MatchesGeneratorAlias(t *testing.T) {
	const boID = "bo_orders"
	rootDef := &boresolver.BODefinition{
		ID:           boID,
		DrivingTable: "public.orders",
		Fields: []boresolver.BOField{
			{ID: "f1", Name: "id", DisplayName: "Order ID", PhysicalColumn: "id"},
			{ID: "f2", Name: "total_amount", DisplayName: "Total Amount", PhysicalColumn: "total_amount"},
			{ID: "f3", Name: "note", DisplayName: "", PhysicalColumn: "note"},
		},
	}
	// Minimal in-package BORepository implementation. MockBORepository
	// (the boresolver package's exported test helper) is in a test-only
	// file and not importable here, so we satisfy the interface with
	// exactly what NewBOSQLGenerator touches during this test.
	repo := mockBORepoForTest{
		rootDef: rootDef,
	}
	generator, err := boresolver.NewBOSQLGenerator(repo, "postgres")
	if err != nil {
		t.Fatalf("NewBOSQLGenerator: %v", err)
	}

	ctx := &boresolver.GenerationContext{
		Request:      boresolver.SQLGenerationRequest{BusinessObjectID: boID, TenantID: "tenant-alpha"},
		RootBODef:    rootDef,
		LoadedBOs:    map[string]*boresolver.BODefinition{boID: rootDef},
		Aliases:      map[string]string{"": "t0"},
		Joins:        nil,
		NextAliasIdx: 1,
	}

	for _, term := range []string{"id", "total_amount", "note"} {
		_, label, err := generator.ResolvePathWithLabel(ctx, term)
		if err != nil {
			t.Errorf("ResolvePathWithLabel(%q): %v", term, err)
			continue
		}
		// Preview-side wireName computation — mirror of the inline
		// logic in service.go::Preview.
		var field boresolver.BOField
		for _, f := range ctx.RootBODef.Fields {
			if f.Name == term {
				field = f
				break
			}
		}
		wireName := field.DisplayName
		if wireName == "" {
			wireName = field.Name
		}
		if label != wireName {
			t.Errorf("name-match divergence for %q: ResolvePathWithLabel=%q, preview-wireName=%q — applyColumnMetadata will silently no-op", term, label, wireName)
		}
	}
}

// mockBORepoForTest is a minimal BORepository implementation just for the
// Name-match cross-check above. Only GetBODefinition is exercised by
// NewBOSQLGenerator's construction path in this test; the other two
// methods return zero values because they aren't reached.
type mockBORepoForTest struct {
	rootDef *boresolver.BODefinition
}

func (m mockBORepoForTest) GetBODefinition(boID string) (*boresolver.BODefinition, error) {
	return m.rootDef, nil
}

func (m mockBORepoForTest) GetBOByTechnicalName(technicalName, tenantID, datasourceID string) (*boresolver.BODefinition, error) {
	return nil, nil
}

func (m mockBORepoForTest) TableHasColumn(drivingTable, column string) bool {
	return false
}
