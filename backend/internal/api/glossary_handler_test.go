package api

import (
	"encoding/json"
	"testing"
)

func TestValidateTermProperties_StringMinLength(t *testing.T) {
	props := []NodeProperty{{Name: "data_type", Label: "Data Type", DataType: "string", Nullable: false, InputType: "text", Validation: map[string]interface{}{"minLength": 2}}}
	values := map[string]interface{}{"data_type": "x"}
	if errs, ok := validateTermProperties(props, values); ok {
		t.Fatalf("Expected validation to fail for minLength but succeeded: %v", errs)
	}
}

func TestValidateTermProperties_JSONParse(t *testing.T) {
	props := []NodeProperty{{Name: "meta_json", Label: "Meta JSON", DataType: "json", Nullable: true, InputType: "json-editor"}}
	values := map[string]interface{}{"meta_json": "{invalid: }"}
	if errs, ok := validateTermProperties(props, values); ok {
		t.Fatalf("Expected validation to fail for invalid JSON but succeeded: %v", errs)
	}
}

func TestValidateTermProperties_NumberMinMax(t *testing.T) {
	props := []NodeProperty{{Name: "score", Label: "Score", DataType: "integer", Nullable: true, InputType: "number", Validation: map[string]interface{}{"min": 1, "max": 10}}}
	values := map[string]interface{}{"score": 0}
	if errs, ok := validateTermProperties(props, values); ok {
		t.Fatalf("Expected validation to fail for min but succeeded: %v", errs)
	}

	values["score"] = 11
	if errs, ok := validateTermProperties(props, values); ok {
		t.Fatalf("Expected validation to fail for max but succeeded: %v", errs)
	}

	values["score"] = 5
	if errs, ok := validateTermProperties(props, values); !ok {
		t.Fatalf("Expected validation to succeed but failed: %v", errs)
	}
}

func TestValidateTermProperties_MultipleArray(t *testing.T) {
	props := []NodeProperty{{Name: "tags", Label: "Tags", DataType: "array", Nullable: true, InputType: "chips", Validation: map[string]interface{}{"multiple": true, "minLength": 1}}}
	values := map[string]interface{}{"tags": []interface{}{}}
	if errs, ok := validateTermProperties(props, values); ok {
		t.Fatalf("Expected validation to fail for empty array but succeeded: %v", errs)
	}
	values["tags"] = []interface{}{"tag1"}
	if errs, ok := validateTermProperties(props, values); !ok {
		t.Fatalf("Expected validation to succeed but failed: %v", errs)
	}
}

func TestJSONUnmarshalForProperties(t *testing.T) {
	// Ensure the helper doesn't panic when dealing with JSON numbers etc
	b, _ := json.Marshal([]NodeProperty{{Name: "n", Label: "N", DataType: "integer", Nullable: false, InputType: "number"}})
	var props []NodeProperty
	if err := json.Unmarshal(b, &props); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}
	values := map[string]interface{}{"n": 1}
	if errs, ok := validateTermProperties(props, values); !ok {
		t.Fatalf("Expected validation to pass for integer: %v", errs)
	}
}
