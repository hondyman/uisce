package querybuilder

import (
	"reflect"
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
