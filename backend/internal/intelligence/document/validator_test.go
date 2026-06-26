package document

import (
	"testing"
)

func TestValidateAgainstODS(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"revenue": { "type": "number" },
			"company_name": { "type": "string" }
		},
		"required": ["revenue", "company_name"]
	}`

	tests := []struct {
		name      string
		json      string
		wantValid bool
	}{
		{
			name:      "Valid JSON",
			json:      `{"revenue": 1000000, "company_name": "Acme Corp"}`,
			wantValid: true,
		},
		{
			name:      "Invalid Type",
			json:      `{"revenue": "1M", "company_name": "Acme Corp"}`,
			wantValid: false,
		},
		{
			name:      "Missing Field",
			json:      `{"revenue": 1000000}`,
			wantValid: false,
		},
		{
			name:      "Malformed JSON",
			json:      `{"revenue": 100`,
			wantValid: false, // Should return error or invalid result depending on implementation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateAgainstODS(tt.json, schema)
			if tt.name == "Malformed JSON" {
				// Malformed JSON might return an error from the loader or be treated as invalid
				if err == nil && result.IsValid {
					t.Errorf("Expected error or invalid result for malformed JSON")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result.IsValid != tt.wantValid {
				t.Errorf("ValidateAgainstODS() IsValid = %v, want %v", result.IsValid, tt.wantValid)
			}
		})
	}
}
