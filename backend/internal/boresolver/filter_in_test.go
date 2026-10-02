package boresolver

import (
	"reflect"
	"testing"
)

// IN is what a multi-select slicer or parameter becomes: a list value on a
// filter. These pin how a list compiles - one bound parameter per member,
// never interpolated - for every value shape callers send (resolved
// parameters arrive as []string, JSON bodies as []interface{}, a typed
// shorthand as a comma string) and every dialect's placeholder style.

func compileIn(t *testing.T, d Dialect, op string, value interface{}) (string, []interface{}) {
	t.Helper()
	g := &BOSQLGenerator{Dialect: d}
	ctx := &GenerationContext{}
	sql, err := CompileFilterPredicate(g, ctx, "t0.region", FilterClause{Operator: op, Value: value})
	if err != nil {
		t.Fatalf("%s %v: %v", op, value, err)
	}
	return finalizeParams(t, g, ctx, sql)
}

func TestCompileFilterPredicate_InBindsEachMemberForEveryValueShape(t *testing.T) {
	want := []interface{}{"EU", "US"}
	for _, tc := range []struct {
		name  string
		value interface{}
	}{
		{"[]string, as a resolved slicer parameter", []string{"EU", "US"}},
		{"[]interface{}, as a JSON body", []interface{}{"EU", "US"}},
		{"comma string, blanks dropped", "EU, ,US"},
	} {
		for _, op := range []string{"IN", "in"} {
			sql, args := compileIn(t, PostgresDialect{}, op, tc.value)
			if sql != "t0.region IN ($1, $2)" || !reflect.DeepEqual(args, want) {
				t.Errorf("%s / %s: got %q args=%v", tc.name, op, sql, args)
			}
		}
	}
}

func TestCompileFilterPredicate_EqualsWithAListIsIn(t *testing.T) {
	// A saved filter keeps its "equals" operator when a slicer supplies several values.
	for _, op := range []string{"=", "equals"} {
		sql, args := compileIn(t, PostgresDialect{}, op, []string{"EU", "US"})
		if sql != "t0.region IN ($1, $2)" || !reflect.DeepEqual(args, []interface{}{"EU", "US"}) {
			t.Errorf("%s: got %q args=%v", op, sql, args)
		}
	}
	// One value stays a plain equality.
	if sql, args := compileIn(t, PostgresDialect{}, "=", "EU"); sql != "t0.region = $1" || !reflect.DeepEqual(args, []interface{}{"EU"}) {
		t.Errorf("single value: got %q args=%v", sql, args)
	}
}

func TestCompileFilterPredicate_InUsesEachDialectsPlaceholders(t *testing.T) {
	for _, tc := range []struct {
		name string
		d    Dialect
		want string
	}{
		{"postgres", PostgresDialect{}, "t0.region IN ($1, $2, $3)"},
		{"sqlserver", SQLServerDialect{}, "t0.region IN (@p1, @p2, @p3)"},
		{"snowflake", SnowflakeDialect{}, "t0.region IN (?, ?, ?)"},
	} {
		sql, args := compileIn(t, tc.d, "IN", []string{"EU", "US", "APAC"})
		if sql != tc.want || len(args) != 3 {
			t.Errorf("%s: got %q args=%v", tc.name, sql, args)
		}
	}
}

func TestCompileFilterPredicate_InNeverInterpolatesValues(t *testing.T) {
	hostile := []string{"x'); DROP TABLE t; --", "EU"}
	sql, args := compileIn(t, PostgresDialect{}, "IN", hostile)
	if sql != "t0.region IN ($1, $2)" || args[0] != hostile[0] {
		t.Fatalf("got %q args=%v", sql, args)
	}
}

func TestCompileFilterPredicate_EmptyInMatchesNothingAndEmptyNotInEverything(t *testing.T) {
	// An empty list must not become "IN ()" (invalid SQL) or silently drop the filter.
	if sql, args := compileIn(t, PostgresDialect{}, "IN", []string{}); sql != "1=0" || len(args) != 0 {
		t.Errorf("empty IN: got %q args=%v", sql, args)
	}
	if sql, args := compileIn(t, PostgresDialect{}, "NOT IN", []string{}); sql != "1=1" || len(args) != 0 {
		t.Errorf("empty NOT IN: got %q args=%v", sql, args)
	}
}
