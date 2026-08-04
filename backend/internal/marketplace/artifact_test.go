package marketplace

import (
	"encoding/json"
	"testing"
)

func TestArtifact_Validate_ValidMinimal(t *testing.T) {
	a := &Artifact{
		SchemaVersion:    "uisce.mp/v1",
		ArtifactType:    ArtifactTypeRBACRole,
		ArtifactVersion: "1.0.0",
		Definition: ArtifactDef{
			"role_key": "role_viewer",
		},
	}
	errs := a.Validate()
	if len(errs) > 0 {
		t.Fatalf("expected valid, got errors: %v", errs)
	}
}

func TestArtifact_Validate_ValidFullRBAC(t *testing.T) {
	a := &Artifact{
		SchemaVersion:    "uisce.mp/v1",
		ArtifactType:    ArtifactTypeRBACRolePack,
		ArtifactVersion: "1.2.3",
		Definition: ArtifactDef{
			"role_key":    "sox_compliance_controller",
			"role_name":   "SOX Compliance Controller",
			"description": "Enforces SOX compliance controls",
			"permissions":  []any{"read", "write", "approve"},
			"role_type":   "controller",
			"categories":  []any{"compliance", "sox"},
		},
	}
	errs := a.Validate()
	if len(errs) > 0 {
		t.Fatalf("expected valid, got errors: %v", errs)
	}
}

func TestArtifact_Validate_ValidABAC(t *testing.T) {
	a := &Artifact{
		SchemaVersion:    "uisce.mp/v1",
		ArtifactType:    ArtifactTypeABACPolicy,
		ArtifactVersion: "1.0.0",
		Definition: ArtifactDef{
			"resource_type": "portfolio",
			"action":        "read",
		},
	}
	errs := a.Validate()
	if len(errs) > 0 {
		t.Fatalf("expected valid, got errors: %v", errs)
	}
}

func TestArtifact_Validate_InvalidSchemaVersion(t *testing.T) {
	a := &Artifact{
		SchemaVersion:    "uisce.mp/v2",
		ArtifactType:    ArtifactTypeRBACRole,
		ArtifactVersion: "1.0.0",
		Definition: ArtifactDef{
			"role_key": "role_viewer",
		},
	}
	errs := a.Validate()
	if len(errs) == 0 {
		t.Fatal("expected error for unsupported schema_version")
	}
}

func TestArtifact_Validate_InvalidArtifactType(t *testing.T) {
	a := &Artifact{
		SchemaVersion:    "uisce.mp/v1",
		ArtifactType:    "unknown_type",
		ArtifactVersion: "1.0.0",
		Definition: ArtifactDef{
			"role_key": "role_viewer",
		},
	}
	errs := a.Validate()
	if len(errs) == 0 {
		t.Fatal("expected error for unknown artifact_type")
	}
}

func TestArtifact_Validate_InvalidSemver(t *testing.T) {
	a := &Artifact{
		SchemaVersion:    "uisce.mp/v1",
		ArtifactType:    ArtifactTypeRBACRole,
		ArtifactVersion: "not-semver",
		Definition: ArtifactDef{
			"role_key": "role_viewer",
		},
	}
	errs := a.Validate()
	if len(errs) == 0 {
		t.Fatal("expected error for invalid semver")
	}
}

func TestArtifact_Validate_NilDefinition(t *testing.T) {
	a := &Artifact{
		SchemaVersion:    "uisce.mp/v1",
		ArtifactType:    ArtifactTypeRBACRole,
		ArtifactVersion: "1.0.0",
		Definition:      nil,
	}
	errs := a.Validate()
	if len(errs) == 0 {
		t.Fatal("expected error for nil definition")
	}
}

func TestArtifact_Validate_RoleKeyEmptyString(t *testing.T) {
	a := &Artifact{
		SchemaVersion:    "uisce.mp/v1",
		ArtifactType:    ArtifactTypeRBACRole,
		ArtifactVersion: "1.0.0",
		Definition: ArtifactDef{
			"role_key": "", // present but empty
		},
	}
	errs := a.Validate()
	found := false
	for _, e := range errs {
		if e == "definition.role_key is required" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected 'role_key is required' error, got: %v", errs)
	}
}

func TestArtifact_Validate_RoleKeyFormat(t *testing.T) {
	a := &Artifact{
		SchemaVersion:    "uisce.mp/v1",
		ArtifactType:    ArtifactTypeRBACRole,
		ArtifactVersion: "1.0.0",
		Definition: ArtifactDef{
			"role_key": "123_invalid", // must start with letter
		},
	}
	errs := a.Validate()
	found := false
	for _, e := range errs {
		if e == "definition.role_key must start with a letter and contain only alphanumeric or underscore characters" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected role_key format error, got: %v", errs)
	}
}

func TestArtifact_Validate_DisallowedDefinitionKeys(t *testing.T) {
	a := &Artifact{
		SchemaVersion:    "uisce.mp/v1",
		ArtifactType:    ArtifactTypeRBACRole,
		ArtifactVersion: "1.0.0",
		Definition: ArtifactDef{
			"role_key": "role_viewer",
			"code":     "executeThis()", // not in allowedRBACDefKeys
		},
	}
	errs := a.Validate()
	found := false
	for _, e := range errs {
		if e == `definition contains disallowed key "code"` {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected disallowed key error, got: %v", errs)
	}
}

func TestArtifact_Validate_PermissionsMustBeStrings(t *testing.T) {
	a := &Artifact{
		SchemaVersion:    "uisce.mp/v1",
		ArtifactType:    ArtifactTypeRBACRole,
		ArtifactVersion: "1.0.0",
		Definition: ArtifactDef{
			"role_key":   "role_viewer",
			"permissions": []any{"read", 123, "write"}, // 123 is not a string
		},
	}
	errs := a.Validate()
	found := false
	for _, e := range errs {
		if e == "definition.permissions[1] must be a string" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected permissions type error, got: %v", errs)
	}
}

func TestArtifact_Validate_DefinitionOver256KiB(t *testing.T) {
	large := make([]byte, 256*1024+1)
	for i := range large {
		large[i] = 'a'
	}
	a := &Artifact{
		SchemaVersion:    "uisce.mp/v1",
		ArtifactType:    ArtifactTypeRBACRole,
		ArtifactVersion: "1.0.0",
		Definition: ArtifactDef{
			"role_key": "role_viewer",
			"large":    string(large),
		},
	}
	errs := a.Validate()
	found := false
	for _, e := range errs {
		if e == "definition exceeds maximum size of 262144 bytes" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected size error, got: %v", errs)
	}
}

func TestArtifact_Validate_DefinitionExactly256KiB(t *testing.T) {
	sz := 256 * 1024
	largeField := make([]byte, sz-50) // leave room for role_key + overhead
	for i := range largeField {
		largeField[i] = 'a'
	}
	a := &Artifact{
		SchemaVersion:    "uisce.mp/v1",
		ArtifactType:    ArtifactTypeRBACRole,
		ArtifactVersion: "1.0.0",
		Definition: ArtifactDef{
			"role_key":    "role_viewer",
			"description": string(largeField),
		},
	}
	errs := a.Validate()
	if len(errs) > 0 {
		t.Fatalf("256 KiB should be valid, got errors: %v", errs)
	}
}

func TestValidateListing_Valid(t *testing.T) {
	errs := ValidateListing(KindRBAC, StatusPublished, PeriodMonthly, "SOX Role Pack", "Description", 9999)
	if len(errs) > 0 {
		t.Fatalf("expected valid, got errors: %v", errs)
	}
}

func TestValidateListing_InvalidKind(t *testing.T) {
	errs := ValidateListing("invalid_kind", StatusPublished, PeriodMonthly, "Title", "Desc", 0)
	if len(errs) == 0 {
		t.Fatal("expected error for invalid kind")
	}
}

func TestValidateListing_InvalidStatus(t *testing.T) {
	errs := ValidateListing(KindRBAC, "invalid_status", PeriodMonthly, "Title", "Desc", 0)
	if len(errs) == 0 {
		t.Fatal("expected error for invalid status")
	}
}

func TestValidateListing_InvalidBillingPeriod(t *testing.T) {
	errs := ValidateListing(KindRBAC, StatusPublished, "invalid_period", "Title", "Desc", 0)
	if len(errs) == 0 {
		t.Fatal("expected error for invalid billing period")
	}
}

func TestValidateListing_EmptyTitle(t *testing.T) {
	errs := ValidateListing(KindRBAC, StatusPublished, PeriodMonthly, "", "Desc", 0)
	if len(errs) == 0 {
		t.Fatal("expected error for empty title")
	}
}

func TestValidateListing_TitleTooLong(t *testing.T) {
	longTitle := make([]byte, 151)
	for i := range longTitle {
		longTitle[i] = 'a'
	}
	errs := ValidateListing(KindRBAC, StatusPublished, PeriodMonthly, string(longTitle), "Desc", 0)
	if len(errs) == 0 {
		t.Fatal("expected error for title > 150 chars")
	}
}

func TestValidateListing_NegativePrice(t *testing.T) {
	errs := ValidateListing(KindRBAC, StatusPublished, PeriodMonthly, "Title", "Desc", -1)
	if len(errs) == 0 {
		t.Fatal("expected error for negative price")
	}
}

func TestArtifact_Validate_ABAC_DisallowedResourceType(t *testing.T) {
	a := &Artifact{
		SchemaVersion:    "uisce.mp/v1",
		ArtifactType:    ArtifactTypeABACPolicy,
		ArtifactVersion: "1.0.0",
		Definition: ArtifactDef{
			"resource_type": "unknown_resource",
			"action":        "read",
		},
	}
	errs := a.Validate()
	found := false
	for _, e := range errs {
		if e == `definition.resource_type "unknown_resource" is not in the allowed list` {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected resource_type error, got: %v", errs)
	}
}

func TestArtifact_Validate_ABAC_DisallowedAction(t *testing.T) {
	a := &Artifact{
		SchemaVersion:    "uisce.mp/v1",
		ArtifactType:    ArtifactTypeABACPolicy,
		ArtifactVersion: "1.0.0",
		Definition: ArtifactDef{
			"resource_type": "portfolio",
			"action":        "unknown_action",
		},
	}
	errs := a.Validate()
	found := false
	for _, e := range errs {
		if e == `definition.action "unknown_action" is not in the allowed list` {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected action error, got: %v", errs)
	}
}

func TestArtifact_Validate_DefinitionCanBeJSONMarshaled(t *testing.T) {
	a := &Artifact{
		SchemaVersion:    "uisce.mp/v1",
		ArtifactType:    ArtifactTypeRBACRole,
		ArtifactVersion: "1.0.0",
		Definition: ArtifactDef{
			"role_key":    "role_viewer",
			"permissions": []any{"read", "write"},
			"nested": map[string]any{
				"key": "value",
			},
		},
	}
	data, err := json.Marshal(a.Definition)
	if err != nil {
		t.Fatalf("definition must be JSON marshalable: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty JSON output")
	}
}
