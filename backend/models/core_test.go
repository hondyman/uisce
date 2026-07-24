package models

import (
	"testing"
)

// TestGenerate_InvalidInputs verifies that Generate returns errors for empty or malformed datasource_id
func TestGenerate_InvalidInputs(t *testing.T) {
	// Use a Services struct with a nil SemanticModelService; Generate parses the datasource_id
	// before calling any service methods, so this is sufficient for validation tests.
	svcs := Services{SemanticModelService: nil}

	// Case 1: empty datasource_id
	reqEmpty := Request{DatasourceID: "", Scope: map[string]interface{}{"type": "tables", "names": []string{"public.categories"}}}
	if _, err := Generate(svcs, reqEmpty); err == nil {
		t.Fatalf("expected error for empty datasource_id, got nil")
	}

	// Case 2: malformed datasource_id
	reqBad := Request{DatasourceID: "not-a-uuid", Scope: map[string]interface{}{"type": "tables", "names": []string{"public.categories"}}}
	if _, err := Generate(svcs, reqBad); err == nil {
		t.Fatalf("expected error for malformed datasource_id, got nil")
	}
}
