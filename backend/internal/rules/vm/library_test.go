package vm

import "testing"

// The library used to be described by a separate registry_test.go that
// cross-checked a FunctionCapability list against the two hand-maintained
// nativeFuncs/starrocksFuncs maps. Now that Library IS the source (not a
// description of two other sources), that cross-check has nothing left to
// check against - so these tests instead verify DERIVED properties: every
// registered function is actually callable natively, financial functions
// with no closed form are honestly non-pushdownable, and aggregates/NPV
// (which do have a closed-form SQL expansion) are pushdownable. TVPI/DPI/
// MOIC need no entry here or in Library - they're compositions of SUM
// (BinaryExpr over FuncCall), proved directly against CompileToSQL in
// TestCompileToSQL_AggregateFuncCall (sql_compiler_test.go).

func TestLibrary_EveryEntryHasNativeImpl(t *testing.T) {
	for _, spec := range LibraryEntries() {
		if spec.Native == nil {
			t.Errorf("%s: Native is nil - every registered function must be callable natively (native/WASM eval has no other implementation to fall back to)", spec.Name)
		}
	}
}

func TestLibrary_EveryEntryHasNameCategoryDescription(t *testing.T) {
	for _, spec := range LibraryEntries() {
		if spec.Name == "" || spec.Category == "" || spec.Description == "" {
			t.Errorf("%+v: Name/Category/Description must all be set - these back editor autocomplete and capability badges", spec)
		}
	}
}

func TestLibrary_NoClosedFormFunctionsAreNotPushdownable(t *testing.T) {
	for _, name := range []string{"IRR", "XIRR"} {
		spec, ok := LookupFunction(name)
		if !ok {
			t.Fatalf("%s not registered", name)
		}
		if !spec.NoClosedForm {
			t.Errorf("%s: expected NoClosedForm=true", name)
		}
		if spec.Pushdownable(DialectStarRocks) {
			t.Errorf("%s: NoClosedForm functions must never be pushdownable - there is no SQL expansion for a numerically-solved root", name)
		}
	}
}

func TestLibrary_AggregatesAndNPVArePushdownable(t *testing.T) {
	for _, name := range []string{"SUM", "AVG", "MIN", "MAX", "NPV"} {
		spec, ok := LookupFunction(name)
		if !ok {
			t.Fatalf("%s not registered", name)
		}
		if !spec.Pushdownable(DialectStarRocks) {
			t.Errorf("%s: expected a StarRocks SQL emitter", name)
		}
	}
}

func TestLibrary_LookupIsCaseInsensitive(t *testing.T) {
	for _, name := range []string{"sum", "Sum", "SUM"} {
		if _, ok := LookupFunction(name); !ok {
			t.Errorf("LookupFunction(%q) should resolve regardless of case", name)
		}
	}
}

func TestLibrary_UnknownFunctionNotRegistered(t *testing.T) {
	if _, ok := LookupFunction("NOT_A_REAL_FUNCTION"); ok {
		t.Fatal("expected NOT_A_REAL_FUNCTION to be unregistered")
	}
}
