package cube

import (
	"testing"
)

func TestMergeCube_AllowsSafeOverrides(t *testing.T) {
	base := Cube{
		Name:     "users",
		SQLTable: "public.users",
		Dimensions: map[string]map[string]any{
			"user_id": {"sql": "id", "type": "number", "primary_key": true},
			"email":   {"sql": "email", "type": "string"},
		},
		Measures: map[string]map[string]any{
			"count": {"type": "count", "sql": "id"},
		},
	}
	ext := Cube{
		Name:        "users_ext",
		Extends:     "users",
		Title:       "Users (Custom)",
		Description: "Extended users",
		Dimensions: map[string]map[string]any{
			"region": {"sql": "region", "type": "string"},
			"email":  {"title": "Email Address"}, // override safe attr
		},
		Measures: map[string]map[string]any{
			"active_user_count": {"type": "count", "filters": []any{"last_login > now() - interval '30 days'"}},
		},
	}
	merged, issues := MergeCube(base, ext)
	if merged.Name != "users" { // published name stays base; caller can adjust if desired
		t.Fatalf("unexpected merged name: %s", merged.Name)
	}
	if len(issues) > 0 {
		// only warnings acceptable; no errors expected
		for _, is := range issues {
			if is.Level == "error" {
				t.Fatalf("unexpected error: %+v", is)
			}
		}
	}
	if _, ok := merged.Dimensions["region"]; !ok {
		t.Fatalf("expected new dimension 'region' in merged cube")
	}
	if merged.Dimensions["email"]["title"] != "Email Address" {
		t.Fatalf("expected email title override")
	}
}

func TestValidateExtension_DisallowPrimaryKeyOverride(t *testing.T) {
	base := Cube{
		Name:     "orders",
		SQLTable: "public.orders",
		Dimensions: map[string]map[string]any{
			"order_id":  {"sql": "id", "type": "number", "primary_key": true},
			"tenant_id": {"sql": "tenant_id", "type": "number"},
		},
	}
	ext := Cube{
		Name:    "orders_ext",
		Extends: "orders",
		Dimensions: map[string]map[string]any{
			"order_id": {"primary_key": false}, // attempt to change
		},
	}
	issues := ValidateExtension(base, ext)
	hasError := false
	for _, is := range issues {
		if is.Code == "DISALLOWED_PK_OVERRIDE" {
			hasError = true
		}
	}
	if !hasError {
		t.Fatalf("expected DISALLOWED_PK_OVERRIDE error")
	}
}

func TestMergeCube_AuditMetadata(t *testing.T) {
	base := Cube{
		Name:     "orders",
		SQLTable: "public.orders",
		Dimensions: map[string]map[string]any{
			"order_id":  {"sql": "id", "type": "number", "primary_key": true},
			"tenant_id": {"sql": "tenant_id", "type": "number"},
		},
		Measures: map[string]map[string]any{
			"count": {"type": "count"},
		},
		Metadata: map[string]any{"core_version": 5},
	}
	ext := Cube{
		Name:    "orders_ext",
		Extends: "orders",
		Dimensions: map[string]map[string]any{
			"region":    {"sql": "region", "type": "string"},
			"tenant_id": {"title": "Tenant"},
		},
		Joins: map[string]map[string]any{
			"customers": {"relationship": "belongsTo"},
		},
		Metadata: map[string]any{"core_version": 5},
	}
	merged, _ := MergeCube(base, ext)
	if merged.Metadata == nil {
		t.Fatalf("expected metadata on merged cube")
	}
	if merged.Metadata["inherits_from"] != "orders" {
		t.Fatalf("expected inherits_from=orders")
	}
	// extension_changes should exist and reflect additions/overrides
	changes, ok := merged.Metadata["extension_changes"].(map[string]any)
	if !ok {
		t.Fatalf("expected extension_changes in metadata")
	}
	if _, ok := changes["dimensions_added"]; !ok {
		t.Fatalf("expected dimensions_added in extension_changes")
	}
}
