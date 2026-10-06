package canonical

import (
	"testing"
)

func TestRFC8785CanonicalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Key sorting (ASCII)",
			input:    `{"z": 1, "a": 2, "m": 3}`,
			expected: `{"a":2,"m":3,"z":1}`,
		},
		{
			name:     "Whitespace removal",
			input:    `{ "foo" : [ 1 , 2 , { "bar" : "baz" } ] }`,
			expected: `{"foo":[1,2,{"bar":"baz"}]}`,
		},
		{
			name:     "Negative zero normalization",
			input:    `{"zero": -0, "neg": -0.0}`,
			expected: `{"neg":0,"zero":0}`,
		},
		{
			name:     "RFC 8785 Appendix B.1 - 1e-7 exponential threshold",
			input:    `{"val": 0.0000001}`,
			expected: `{"val":1e-7}`,
		},
		{
			name:     "RFC 8785 Appendix B.1 - 1e-6 fixed threshold",
			input:    `{"val": 0.000001}`,
			expected: `{"val":0.000001}`,
		},
		{
			name:     "RFC 8785 Appendix B.1 - 1e20 fixed threshold",
			input:    `{"val": 100000000000000000000.0}`,
			expected: `{"val":100000000000000000000}`,
		},
		{
			name:     "RFC 8785 Appendix B.1 - 1e21 exponential threshold",
			input:    `{"val": 1000000000000000000000.0}`,
			expected: `{"val":1e+21}`,
		},
		{
			name:     "Exact 2^53 - 1 integer boundary",
			input:    `{"maxSafeInt": 9007199254740991}`,
			expected: `{"maxSafeInt":9007199254740991}`,
		},
		{
			name:     "Exact 2^53 integer boundary",
			input:    `{"beyondSafeInt": 9007199254740992}`,
			expected: `{"beyondSafeInt":9007199254740992}`,
		},
		{
			name:     "Nested object and UTF-16 code unit ordering",
			input:    `{"\u00e9": "accent", "e": "plain", "\ud83d\ude00": "emoji"}`,
			expected: `{"e":"plain","é":"accent","😀":"emoji"}`,
		},
		{
			name:     "Escaping rules",
			input:    `{"escapes": "line1\nline2\t\"quoted\"\\backslash\u0007bell"}`,
			expected: `{"escapes":"line1\nline2\t\"quoted\"\\backslash\u0007bell"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Transform([]byte(tt.input))
			if err != nil {
				t.Fatalf("Transform() unexpected error: %v", err)
			}
			if string(got) != tt.expected {
				t.Errorf("Transform() mismatch:\n got:      %s\n expected: %s", string(got), tt.expected)
			}
		})
	}
}

func TestRFC8785InvalidJSON(t *testing.T) {
	badInputs := []string{
		`{bad: json}`,
		`{"unclosed": "string`,
		`{"trailing": "data"} trailing`,
	}

	for _, bad := range badInputs {
		_, err := Transform([]byte(bad))
		if err == nil {
			t.Errorf("Transform(%q) expected error, got nil", bad)
		}
	}
}
