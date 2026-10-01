package querybuilder

import (
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/hondyman/uisce/backend/internal/boresolver"
)

// resolveParams turns the values a Page Studio slicer or a REST caller sends
// into the filters a saved query runs: the contract between "what the user
// picked" and the SQL. These pin it, including multi-select becoming an IN.

func state(filters []SavedQueryFilter, params ...SavedQueryParameter) SavedQueryState {
	return SavedQueryState{Filters: filters, Parameters: params}
}

func byRegion(op string, value interface{}) SavedQueryFilter {
	return SavedQueryFilter{TermNodeID: "region", Operator: op, Value: value, ParamRef: "region"}
}

func TestResolveParams_LiteralFiltersPassThrough(t *testing.T) {
	got, err := resolveParams(state([]SavedQueryFilter{{TermNodeID: "status", Operator: "=", Value: "ACTIVE"}}), nil)
	want := []boresolver.FilterDef{{TermNodeID: "status", Operator: "=", Value: "ACTIVE"}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestResolveParams_OneValueIsAStringSeveralAreAList(t *testing.T) {
	st := state([]SavedQueryFilter{byRegion("=", nil)}, SavedQueryParameter{Name: "region"})
	one, _ := resolveParams(st, url.Values{"region": {"EU"}})
	if one[0].Value != "EU" {
		t.Errorf("one value: got %#v", one[0].Value)
	}
	many, _ := resolveParams(st, url.Values{"region": {"EU", "US"}})
	if !reflect.DeepEqual(many[0].Value, []string{"EU", "US"}) {
		t.Errorf("several values: got %#v", many[0].Value)
	}
	// The saved operator is kept: it is the compiler that reads a list as IN.
	if many[0].Operator != "=" {
		t.Errorf("operator changed to %q", many[0].Operator)
	}
}

func TestResolveParams_AnArrayParameterIsAlwaysAList(t *testing.T) {
	st := state([]SavedQueryFilter{byRegion("IN", nil)}, SavedQueryParameter{Name: "region", Type: "array"})
	got, _ := resolveParams(st, url.Values{"region": {"EU"}})
	if !reflect.DeepEqual(got[0].Value, []string{"EU"}) {
		t.Fatalf("got %#v", got[0].Value)
	}
}

func TestResolveParams_ABlankValueCountsAsNotGiven(t *testing.T) {
	st := state([]SavedQueryFilter{byRegion("=", nil)}, SavedQueryParameter{Name: "region", Default: "GLOBAL"})
	got, _ := resolveParams(st, url.Values{"region": {""}})
	if got[0].Value != "GLOBAL" {
		t.Fatalf("a blank slicer should fall back to the default; got %#v", got[0].Value)
	}
}

func TestResolveParams_DefaultThenRequiredThenDropped(t *testing.T) {
	filter := []SavedQueryFilter{byRegion("=", nil)}
	// A default is used when nothing is given.
	if got, err := resolveParams(state(filter, SavedQueryParameter{Name: "region", Default: "EU"}), nil); err != nil || got[0].Value != "EU" {
		t.Errorf("default: got %v, %v", got, err)
	}
	// Required with no value and no default is an error naming the parameter.
	if _, err := resolveParams(state(filter, SavedQueryParameter{Name: "region", Required: true}), nil); err == nil || !strings.Contains(err.Error(), `"region"`) {
		t.Errorf("required: got %v", err)
	}
	// Optional, unset, no default: the filter is dropped, not sent as a nil predicate.
	if got, err := resolveParams(state(filter, SavedQueryParameter{Name: "region"}), nil); err != nil || len(got) != 0 {
		t.Errorf("optional unset: got %v, %v", got, err)
	}
}

// The whole chain, as a slicer drives it: the picked values, resolved, then compiled.
func TestSlicerSelectionCompilesToSQL(t *testing.T) {
	st := state([]SavedQueryFilter{byRegion("=", nil)}, SavedQueryParameter{Name: "region"})
	compile := func(values url.Values) (string, []interface{}) {
		t.Helper()
		filters, err := resolveParams(st, values)
		if err != nil || len(filters) != 1 {
			t.Fatalf("resolve: %v, %v", filters, err)
		}
		g := &boresolver.BOSQLGenerator{Dialect: boresolver.PostgresDialect{}}
		ctx := &boresolver.GenerationContext{}
		sql, err := boresolver.CompileFilterPredicate(g, ctx, "t0.region", boresolver.FilterClause{Operator: filters[0].Operator, Value: filters[0].Value})
		if err != nil {
			t.Fatal(err)
		}
		return sql, ctx.Args
	}

	if sql, args := compile(url.Values{"region": {"EU"}}); sql != "t0.region = $1" || !reflect.DeepEqual(args, []interface{}{"EU"}) {
		t.Errorf("one pick: got %q args=%v", sql, args)
	}
	if sql, args := compile(url.Values{"region": {"EU", "US"}}); sql != "t0.region IN ($1, $2)" || !reflect.DeepEqual(args, []interface{}{"EU", "US"}) {
		t.Errorf("two picks: got %q args=%v", sql, args)
	}
}
