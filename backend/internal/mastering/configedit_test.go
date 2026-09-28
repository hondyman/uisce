package mastering

import (
	"encoding/json"
	"testing"
)

func TestConfigColumnsAndKeys(t *testing.T) {
	for dt, want := range map[string]string{
		"character varying": "text", "integer": "integer", "numeric": "number", "boolean": "boolean",
		"jsonb": "json", "ARRAY": "list", "uuid": "", "timestamp with time zone": "",
	} {
		if got := columnType(dt); got != want {
			t.Errorf("columnType(%q) = %q, want %q", dt, got, want)
		}
	}
	// Match rules override by rule code, the registry by code, a hierarchy
	// row by its scope, group and source.
	if !isKey(KindMatchRule, "rule_cd", "text") || isKey(KindMatchRule, "rule_name", "text") {
		t.Error("match rules are keyed by rule_cd only")
	}
	if !isKey(KindSourceSystem, "code", "text") || isKey(KindSourceSystem, "display_name", "text") {
		t.Error("the registry is keyed by code only")
	}
	if !isKey(KindSourcePriority, "field_group", "text") || isKey(KindSourcePriority, "priority", "integer") {
		t.Error("a hierarchy row is keyed by its text columns")
	}
}

func TestConfigKeyOfTreatsWildcardsAsEmpty(t *testing.T) {
	tg := &configTarget{columns: []ConfigColumn{
		{Name: "field_group", Type: "text", Key: true}, {Name: "asset_class_cd", Type: "text", Key: true},
		{Name: "source", Type: "source", Key: true}, {Name: "priority", Type: "integer"},
	}}
	a := tg.keyOf(map[string]any{"field_group": "name", "asset_class_cd": "*", "source": "bloomberg", "priority": 1.0})
	b := tg.keyOf(map[string]any{"field_group": "NAME", "asset_class_cd": nil, "source": "BLOOMBERG", "priority": 9.0})
	if a != b {
		t.Fatalf("same key expected: %q vs %q", a, b)
	}
}

func TestConfigRowValuesShowSourceAsCode(t *testing.T) {
	tg := &configTarget{sourceCol: "source_system_id", columns: []ConfigColumn{
		{Name: "field_group", Type: "text", Key: true}, {Name: "source", Type: "source", Key: true}, {Name: "priority", Type: "integer"},
	}}
	raw, _ := json.Marshal(map[string]any{"id": "r1", "tenant_id": "t", "field_group": "NAME", "source_system_id": "s1", "priority": 10})
	v := tg.rowValues(raw, map[string]string{"s1": "REFINITIV"})
	if v["source"] != "REFINITIV" || v["field_group"] != "NAME" || v["priority"] != 10.0 || v["id"] != nil {
		t.Fatalf("got %v", v)
	}
}
