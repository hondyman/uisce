package metadata

import (
	"reflect"
	"testing"
)

func TestGoldCopyRebaseService_Compute3WayMerge(t *testing.T) {
	service := NewGoldCopyRebaseService(nil)

	// Baseline (Base v1)
	baseV1 := map[string]interface{}{
		"data_type":       "VARCHAR(50)",
		"regex_pattern":   "^[A-Z0-9]+$",
		"validation_rule": "LENGTH <= 50",
		"description":     "Legacy account code definition",
	}

	// ─────────────────────────────────────────────
	// Test 1: Clean Merge - Only Gold Copy updated (Base v2 upgrade applied)
	// ─────────────────────────────────────────────
	baseV2Upgrade := map[string]interface{}{
		"data_type":       "VARCHAR(50)",
		"regex_pattern":   "^[A-Z]{2}[0-9]{6}$", // Upgraded ISO regex in Gold Copy
		"validation_rule": "LENGTH <= 50",
		"description":     "Legacy account code definition",
	}
	tenantUnchanged := map[string]interface{}{
		"data_type":       "VARCHAR(50)",
		"regex_pattern":   "^[A-Z0-9]+$",
		"validation_rule": "LENGTH <= 50",
		"description":     "Legacy account code definition",
	}

	merged, conflicts := service.Compute3WayMerge(baseV1, baseV2Upgrade, tenantUnchanged)
	if len(conflicts) > 0 {
		t.Fatalf("expected 0 conflicts, got %v", conflicts)
	}
	if merged["regex_pattern"] != "^[A-Z]{2}[0-9]{6}$" {
		t.Errorf("expected upgraded regex_pattern '^[A-Z]{2}[0-9]{6}$', got %v", merged["regex_pattern"])
	}

	// ─────────────────────────────────────────────
	// Test 2: Clean Merge - Only Tenant customized (Tenant customization preserved)
	// ─────────────────────────────────────────────
	baseV2Unchanged := map[string]interface{}{
		"data_type":       "VARCHAR(50)",
		"regex_pattern":   "^[A-Z0-9]+$",
		"validation_rule": "LENGTH <= 50",
		"description":     "Legacy account code definition",
	}
	tenantCustomized := map[string]interface{}{
		"data_type":       "VARCHAR(50)",
		"regex_pattern":   "^[A-Z0-9]+$",
		"validation_rule": "LENGTH <= 100", // Custom tenant relaxation
		"description":     "Custom tenant description",
	}

	merged, conflicts = service.Compute3WayMerge(baseV1, baseV2Unchanged, tenantCustomized)
	if len(conflicts) > 0 {
		t.Fatalf("expected 0 conflicts, got %v", conflicts)
	}
	if merged["validation_rule"] != "LENGTH <= 100" {
		t.Errorf("expected tenant validation_rule 'LENGTH <= 100', got %v", merged["validation_rule"])
	}
	if merged["description"] != "Custom tenant description" {
		t.Errorf("expected tenant description, got %v", merged["description"])
	}

	// ─────────────────────────────────────────────
	// Test 3: Conflict - Both modified same key with different values
	// ─────────────────────────────────────────────
	baseV2Conflicting := map[string]interface{}{
		"data_type":       "VARCHAR(50)",
		"regex_pattern":   "^[A-Z]{3}[0-9]{5}$", // Core changed pattern
		"validation_rule": "LENGTH <= 50",
		"description":     "Core updated description",
	}
	tenantConflicting := map[string]interface{}{
		"data_type":       "VARCHAR(50)",
		"regex_pattern":   "^[A-Z0-9_]+$", // Tenant also changed pattern
		"validation_rule": "LENGTH <= 50",
		"description":     "Tenant custom description",
	}

	merged, conflicts = service.Compute3WayMerge(baseV1, baseV2Conflicting, tenantConflicting)
	if len(conflicts) != 2 {
		t.Fatalf("expected 2 conflicts (regex_pattern, description), got %v", conflicts)
	}
	// Fallback to tenant safety
	if merged["regex_pattern"] != "^[A-Z0-9_]+$" {
		t.Errorf("expected tenant fallback for conflicted regex_pattern, got %v", merged["regex_pattern"])
	}

	// ─────────────────────────────────────────────
	// Test 4: Converged Values - Both updated to same value
	// ─────────────────────────────────────────────
	baseV2Converged := map[string]interface{}{
		"data_type": "TEXT",
	}
	tenantConverged := map[string]interface{}{
		"data_type": "TEXT",
	}

	merged, conflicts = service.Compute3WayMerge(baseV1, baseV2Converged, tenantConverged)
	if len(conflicts) > 0 {
		t.Fatalf("expected 0 conflicts on converged values, got %v", conflicts)
	}
	if merged["data_type"] != "TEXT" {
		t.Errorf("expected converged data_type 'TEXT', got %v", merged["data_type"])
	}
}

func TestGoldCopyRebaseService_ThreeWayMathematicalInvariants(t *testing.T) {
	service := NewGoldCopyRebaseService(nil)

	// Mathematical Invariant: Clean Merge = Base_v2 ⊕ Δ_Tenant
	baseV1 := map[string]interface{}{"a": 1, "b": 2, "c": 3}
	baseV2 := map[string]interface{}{"a": 1, "b": 20, "c": 3} // Δ_Core = {b: 20}
	tenantCustom := map[string]interface{}{"a": 1, "b": 2, "c": 300, "d": 400} // Δ_Tenant = {c: 300, d: 400}

	expectedMerged := map[string]interface{}{
		"a": 1,
		"b": 20,  // From Δ_Core
		"c": 300, // From Δ_Tenant
		"d": 400, // From Δ_Tenant
	}

	merged, conflicts := service.Compute3WayMerge(baseV1, baseV2, tenantCustom)
	if len(conflicts) != 0 {
		t.Fatalf("expected no conflicts, got %v", conflicts)
	}
	if !reflect.DeepEqual(merged, expectedMerged) {
		t.Errorf("expected %v, got %v", expectedMerged, merged)
	}
}
