package attribute

import (
	"testing"
)

func TestDehydrateAndHydrateRoundTrip(t *testing.T) {
	projections := []FieldProjection{
		{LogicalName: "SFDR Article", FieldCd: "sfdr_article", DataType: "integer"},
		{LogicalName: "sfdr_article", FieldCd: "sfdr_article", DataType: "integer"},
		{LogicalName: "esg_score", FieldCd: "esg_score", DataType: "decimal"},
	}

	rec := map[string]any{
		"id":           "1",
		"product_cd":   "VFINX",
		"sfdr_article": 8,
		"esg_score":    7.5,
		"name":         "Vanguard",
	}

	out := DehydrateRecord(rec, projections)
	if _, ok := out["sfdr_article"]; ok {
		t.Fatal("expected sfdr_article removed from top-level")
	}
	custom, ok := out["custom_attributes"].(map[string]any)
	if !ok {
		t.Fatalf("custom_attributes missing: %#v", out["custom_attributes"])
	}
	if custom["sfdr_article"] != 8 || custom["esg_score"] != 7.5 {
		t.Fatalf("unexpected custom map: %#v", custom)
	}
	if out["product_cd"] != "VFINX" {
		t.Fatal("core column should remain")
	}

	hydrated := HydrateRecord(map[string]any{
		"id":                "1",
		"product_cd":        "VFINX",
		"custom_attributes": custom,
	}, projections)
	if hydrated["sfdr_article"] != 8 {
		t.Fatalf("hydrate field_cd failed: %#v", hydrated)
	}
	if hydrated["SFDR Article"] != 8 {
		t.Fatalf("hydrate logical name failed: %#v", hydrated)
	}
}
