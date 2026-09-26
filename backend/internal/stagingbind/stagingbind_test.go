package stagingbind

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestWhoMayAdminister(t *testing.T) {
	cases := []struct {
		a    Actor
		gold bool
		want bool
	}{
		{Actor{PlatformAdmin: true}, true, true},
		{Actor{TenantAdmin: true}, true, false}, // core bindings: platform administrators only
		{Actor{TenantAdmin: true}, false, true},
		{Actor{PlatformAdmin: true}, false, true},
		{Actor{}, false, false},
	}
	for _, c := range cases {
		if got := c.a.canAdminister(c.gold); got != c.want {
			t.Errorf("%+v gold=%v: got %v", c.a, c.gold, got)
		}
	}
}

// Approval re-checks the binding against the one the change was proposed
// on, so a change never overwrites a later edit.
func TestSameFields(t *testing.T) {
	a := map[string]string{"Isin": "isin", "Cusip": "cusip"}
	if !sameFields(a, map[string]string{"Cusip": "cusip", "Isin": "isin"}) {
		t.Error("order must not matter")
	}
	if sameFields(a, map[string]string{"Isin": "isin_cd", "Cusip": "cusip"}) {
		t.Error("a changed column is a different binding")
	}
	if !sameFields(nil, map[string]string{}) {
		t.Error("none and empty are both 'no binding'")
	}
	if sameFields(nil, a) {
		t.Error("a binding created since the proposal must be caught")
	}
}

func TestStagingTableNames(t *testing.T) {
	for name, ok := range map[string]bool{
		"staging.ff_product": true, "staging._load_run": true, "staging.FF": false, "public.ff": false,
		"staging.ff;drop": false, "staging.": false, "ff_product": false,
	} {
		if stagingTable.MatchString(name) != ok {
			t.Errorf("%q: want %v", name, ok)
		}
	}
}

func TestChangeJSONFlattensAudit(t *testing.T) {
	c := Change{ID: "c1", Status: "applied", Fields: map[string]string{"Isin": "isin"}}
	c.ReviewedBy.String, c.ReviewedBy.Valid = "u2", true
	c.ReviewedAt.Time, c.ReviewedAt.Valid = time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC), true
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{`"reviewed_by":"u2"`, `"reviewed_at":"2026-09-26T00:00:00Z"`, `"fields":{"Isin":"isin"}`} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %s in %s", want, s)
		}
	}
	if strings.Contains(s, `"String"`) || strings.Contains(s, `"Valid"`) {
		t.Errorf("nullable columns leaked: %s", s)
	}
}
