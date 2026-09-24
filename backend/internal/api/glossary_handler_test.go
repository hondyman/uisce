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

func TestTokenizeColumnName(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"account_id", []string{"account", "id"}},
		{"customer_uuid", []string{"customer", "uuid"}},
		{"external_ref_id", []string{"external", "ref", "id"}},
		{"settlement_date", []string{"settlement", "date"}},
		{"postal_code", []string{"postal", "code"}},
		{"first_name", []string{"first", "name"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := tokenizeColumnName(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("tokenizeColumnName(%q) = %v, want %v", tt.input, result, tt.expected)
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("tokenizeColumnName(%q)[%d] = %q, want %q", tt.input, i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestPascaCase(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []string
		expected string
	}{
		{"account + IDENTIFIER", []string{"account", "IDENTIFIER"}, "AccountIdentifier"},
		{"customer + IDENTIFIER", []string{"customer", "IDENTIFIER"}, "CustomerIdentifier"},
		{"settlement + DATE (4 chars, not acronym)", []string{"settlement", "DATE"}, "SettlementDate"},
		{"external + REF + ID", []string{"external", "REF", "ID"}, "ExternalREFID"},
		{"account + CD (3 chars, kept as acronym)", []string{"account", "CD"}, "AccountCD"},
		{"bare id (lowercase)", []string{"id"}, "Id"},
		{"uuid", []string{"uuid"}, "Uuid"},
		{"ACCOUNT (all caps, > 3 chars, title-cased)", []string{"ACCOUNT"}, "Account"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pascalCase(tt.tokens)
			if result != tt.expected {
				t.Errorf("pascalCase(%v) = %q, want %q", tt.tokens, result, tt.expected)
			}
		})
	}
}

func TestTitleCase(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []string
		expected string
	}{
		{"account + IDENTIFIER", []string{"account", "IDENTIFIER"}, "Account Identifier"},
		{"customer + IDENTIFIER", []string{"customer", "IDENTIFIER"}, "Customer Identifier"},
		{"settlement + DATE", []string{"settlement", "DATE"}, "Settlement Date"},
		{"external + REF + ID", []string{"external", "REF", "ID"}, "External Ref Id"},
		{"bare id (lowercase)", []string{"id"}, "Id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := titleCase(tt.tokens)
			if result != tt.expected {
				t.Errorf("titleCase(%v) = %q, want %q", tt.tokens, result, tt.expected)
			}
		})
	}
}

func TestLooksLikeAbbreviation(t *testing.T) {
	tests := []struct {
		token    string
		expected bool
	}{
		{"ID", true},        // "ID" removed from commonShortWords
		{"CD", true},
		{"ACCT", true},
		{"EXCH", true},
		{"OF", false},       // commonShortWords
		{"OR", false},       // commonShortWords
		{"uuid", false},     // not all-caps
		{"name", false},     // not all-caps
		{"customer", false}, // not all-caps
		{"AND", true},       // all-caps, not in commonShortWords
		{"THE", true},       // all-caps, not in commonShortWords
	}

	for _, tt := range tests {
		t.Run(tt.token, func(t *testing.T) {
			result := looksLikeAbbreviation(tt.token)
			if result != tt.expected {
				t.Errorf("looksLikeAbbreviation(%q) = %v, want %v", tt.token, result, tt.expected)
			}
		})
	}
}

func TestPascalCaseToWords(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want []string
	}{
		{"SupplierCity", []string{"Supplier", "City"}},
		{"IssuerAddressCity", []string{"Issuer", "Address", "City"}},
		{"AccountIdentifier", []string{"Account", "Identifier"}},
		{"EmployeeCity", []string{"Employee", "City"}},
		{"City", []string{"City"}},
		{"ID", []string{"ID"}},
		{"XMLParser", []string{"XML", "Parser"}},
		{"ROE", []string{"ROE"}},
	} {
		got := pascalCaseToWords(tc.in)
		if len(got) != len(tc.want) {
			t.Errorf("pascalCaseToWords(%q) = %v, want %v", tc.in, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("pascalCaseToWords(%q)[%d] = %q, want %q", tc.in, i, got[i], tc.want[i])
			}
		}
	}
}

func TestQualifiedTermBusinessName(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want string
	}{
		{"SupplierCity", "Supplier City"},
		{"IssuerAddressCity", "Issuer Address City"},
		{"AccountIdentifier", "Account Identifier"},
		{"EmployeeCity", "Employee City"},
	} {
		got := titleCase(pascalCaseToWords(tc.in))
		if got != tc.want {
			t.Errorf("titleCase(pascalCaseToWords(%q)) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
