package semantic

import (
	"testing"
)

func TestMergeCubes(t *testing.T) {
	// 1. Setup Base Cube (Core)
	baseCube := &Cube{
		ID:          "core_cube_1",
		TenantID:    "system",
		Name:        "base_c",
		DisplayName: "Base Cube",
		IsSystem:    true,
		Dimensions: []Dimension{
			{ID: "d1", Name: "dim1", DisplayName: "Dimension 1", Shown: true, Type: "string"},
			{ID: "d2", Name: "dim2", DisplayName: "Dimension 2", Shown: true, Type: "number"},
		},
		Measures: []Measure{
			{ID: "m1", Name: "meas1", DisplayName: "Measure 1"},
		},
	}

	// 2. Setup Custom Cube (Override)
	customCube := &Cube{
		ID:           "custom_cube_1",
		TenantID:     "tenant_A",
		Name:         "custom_c",
		DisplayName:  "Custom Cube",
		IsSystem:     false,
		SourceCubeID: &baseCube.ID,
		// Overriding dimension 1
		Dimensions: []Dimension{
			{ID: "d1_custom", Name: "dim1", DisplayName: "Dimension 1 (Custom)", Shown: true, Type: "string"}, // Same name, new display name
			{ID: "d3", Name: "dim3", DisplayName: "Dimension 3", Shown: true, Type: "time"},                   // New dimension
		},
		// Inheriting Measure 1 implicitly (not listed here means no override in this struct,
		// BUT wait, the logic in service.go: "merged.Dimensions = append(base... override...)"
		// Let's check logic:
		// for _, d := range base.Dimensions { dimMap[d.Name] = d }
		// for _, d := range override.Dimensions { dimMap[d.Name] = d }
		// This implies if I want to inherit, I don't need to put it in override.
		// But if I want to KEEP it, it should be in base.
	}

	// 3. Execute Merge using the service helper (we need to access the private method or replicate logic if it tests package internals)
	// Since we are in package semantic, we can call s.mergeCubes if we have an s.
	// But mergeCubes is a method on *Service. I need a dummy service.
	s := &Service{} // No DB needed for pure logic test of mergeCubes

	merged := s.mergeCubes(baseCube, customCube)

	// 4. Assertions
	if merged.Name != customCube.Name {
		t.Errorf("Expected name %s, got %s", customCube.Name, merged.Name)
	}
	if merged.DisplayName != customCube.DisplayName {
		t.Errorf("Expected display name %s, got %s", customCube.DisplayName, merged.DisplayName)
	}

	// Verify Dimensions
	if len(merged.Dimensions) != 3 {
		t.Errorf("Expected 3 dimensions, got %d", len(merged.Dimensions))
	}

	dimMap := make(map[string]Dimension)
	for _, d := range merged.Dimensions {
		dimMap[d.Name] = d
	}

	// Check Override
	if dimMap["dim1"].DisplayName != "Dimension 1 (Custom)" {
		t.Errorf("Expected overridden display name, got %s", dimMap["dim1"].DisplayName)
	}
	// Check Inheritance
	if dimMap["dim2"].DisplayName != "Dimension 2" {
		t.Errorf("Expected inherited display name, got %s", dimMap["dim2"].DisplayName)
	}
	// Check New
	if _, ok := dimMap["dim3"]; !ok {
		t.Errorf("Expected new dimension dim3")
	}

	// Verify Measures (Inheritance only)
	if len(merged.Measures) != 1 {
		t.Errorf("Expected 1 measure, got %d", len(merged.Measures))
	}
	if merged.Measures[0].Name != "meas1" {
		t.Errorf("Expected inherited measure meas1")
	}
}
