package vm

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
)

func TestCompileConditionSQL(t *testing.T) {
	for _, tc := range []struct {
		name     string
		op       string
		value    interface{}
		wantSQL  string
		wantArgs []interface{}
		wantErr  string
	}{
		{"canonical equals", "equals", "x", "c = $1", []interface{}{"x"}, ""},
		{"short alias", "gte", 5, "c >= $1", []interface{}{5}, ""},
		{"query-builder spelling", "GT", 5, "c > $1", []interface{}{5}, ""},
		{"empty operator is equals", "", "x", "c = $1", []interface{}{"x"}, ""},
		{"<> alias", "<>", "x", "c != $1", []interface{}{"x"}, ""},
		{"equals list is IN", "=", []interface{}{"a", "b"}, "c IN ($1, $2)", []interface{}{"a", "b"}, ""},
		{"not_equals list is NOT IN", "!=", []string{"a"}, "c NOT IN ($1)", []interface{}{"a"}, ""},
		{"equals nil is IS NULL", "eq", nil, "c IS NULL", nil, ""},
		{"not_equals nil is IS NOT NULL", "neq", nil, "c IS NOT NULL", nil, ""},
		{"in comma string drops blanks", "IN", "x, ,y", "c IN ($1, $2)", []interface{}{"x", "y"}, ""},
		{"empty IN is false", "in", []interface{}{}, "1=0", nil, ""},
		{"empty NOT IN is true", "NOT IN", []interface{}{}, "1=1", nil, ""},
		{"not_in underscore", "NOT_IN", []interface{}{1.0}, "c NOT IN ($1)", []interface{}{1.0}, ""},
		{"contains escapes wildcards", "contains", "50%_off", "c ILIKE $1", []interface{}{`%50\%\_off%`}, ""},
		{"like is contains", "like", "ab", "c ILIKE $1", []interface{}{"%ab%"}, ""},
		{"not_contains", "not_contains", "ab", "c NOT ILIKE $1", []interface{}{"%ab%"}, ""},
		{"starts with spaced", "STARTS WITH", "ab", "c ILIKE $1", []interface{}{"ab%"}, ""},
		{"ends_with", "ends_with", "ab", "c ILIKE $1", []interface{}{"%ab"}, ""},
		{"between", "BETWEEN", []interface{}{1, 9}, "c BETWEEN $1 AND $2", []interface{}{1, 9}, ""},
		{"not between", "NOT BETWEEN", []interface{}{1, 9}, "c NOT BETWEEN $1 AND $2", []interface{}{1, 9}, ""},
		{"is null spaced", "IS NOT NULL", nil, "c IS NOT NULL", nil, ""},
		{"is_true", "IS_TRUE", nil, "c IS TRUE", nil, ""},
		{"date before", "before", "2026-01-01", "c < $1", []interface{}{"2026-01-01"}, ""},
		{"date on_or_after", "on_or_after", "2026-01-01", "c >= $1", []interface{}{"2026-01-01"}, ""},
		{"relative date has no pushdown", "is_today", nil, "", nil, `unsupported filter operator "is_today"`},

		{"ordering with a list is an error", ">", []interface{}{1, 2}, "", nil, "greater_than requires a single value, not a list"},
		{"ordering with nil is an error", "lt", nil, "", nil, "less_than requires a value"},
		{"contains with a list is an error", "contains", []interface{}{"a"}, "", nil, "contains requires a single value"},
		{"between needs two bounds", "between", []interface{}{1}, "", nil, "between requires a two-element value array [low, high]"},
		{"in needs a list", "in", 5, "", nil, "in requires a list value (array or comma-separated string)"},
		{"injection attempt", "1=1; --", "x", "", nil, `unsupported filter operator "1=1; --"`},
		{"VM-only operator has no pushdown", "length_equals", 3, "", nil, `unsupported filter operator "length_equals"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var args []interface{}
			sql, err := CompileConditionSQL("c", tc.op, tc.value, func(v interface{}) string {
				args = append(args, v)
				return fmt.Sprintf("$%d", len(args))
			})
			if tc.wantErr != "" {
				if err == nil || err.Error() != tc.wantErr {
					t.Fatalf("err = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if sql != tc.wantSQL || !reflect.DeepEqual(args, tc.wantArgs) {
				t.Errorf("got %q %v, want %q %v", sql, args, tc.wantSQL, tc.wantArgs)
			}
		})
	}
}

func TestComparisonSQL(t *testing.T) {
	for op, want := range map[string]string{"": "=", "GTE": ">=", "not_equals": "!=", "<>": "!=", "less_than": "<"} {
		if got, err := ComparisonSQL(op); err != nil || got != want {
			t.Errorf("ComparisonSQL(%q) = %q, %v; want %q", op, got, err, want)
		}
	}
	var uerr *UnsupportedOperatorError
	for _, op := range []string{"contains", "IN", "1=1; --", "IS NULL"} {
		if _, err := ComparisonSQL(op); !errors.As(err, &uerr) {
			t.Errorf("ComparisonSQL(%q) err = %v, want UnsupportedOperatorError", op, err)
		}
	}
}
