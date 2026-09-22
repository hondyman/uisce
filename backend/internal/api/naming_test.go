package api

import (
	"testing"
)

func TestNaming_TokenizeColumnName(t *testing.T) {
	cases := []struct {
		input    string
		expected []string
	}{
		{"address_line_1", []string{"address", "line", "1"}},
		{"customer_acct_no", []string{"customer", "acct", "no"}},
		{"AcctCd", []string{"Acct", "Cd"}},
		{"issuer_address", []string{"issuer", "address"}},
		{"ack_at", []string{"ack", "at"}},
		{"ack_status", []string{"ack", "status"}},
		{"account_id", []string{"account", "id"}},
		{"XMLParser", []string{"XMLParser"}},
		{"CustomerIdentifier", []string{"Customer", "Identifier"}},
		{"CUSIP", []string{"CUSIP"}},
	}

	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			got := tokenizeColumnName(c.input)
			if len(got) != len(c.expected) {
				t.Errorf("tokenize(%q) = %v; want %v", c.input, got, c.expected)
				return
			}
			for i := range got {
				if got[i] != c.expected[i] {
					t.Errorf("tokenize(%q)[%d] = %q; want %q", c.input, i, got[i], c.expected[i])
				}
			}
		})
	}
}

func TestNaming_PascalCase(t *testing.T) {
	cases := []struct {
		input    []string
		expected string
	}{
		{[]string{"address", "line", "1"}, "AddressLine1"},
		{[]string{"customer", "acct", "no"}, "CustomerAcctNo"},
		{[]string{"acknowledged"}, "Acknowledged"},
		{[]string{"acknowledgement", "status"}, "AcknowledgementStatus"},
		{[]string{"issuer", "address"}, "IssuerAddress"},
		{[]string{"XML"}, "XML"},
		{[]string{"account", "identifier"}, "AccountIdentifier"},
		{[]string{"CUSIP"}, "Cusip"},
	}

	for _, c := range cases {
		t.Run(c.expected, func(t *testing.T) {
			got := pascalCase(c.input)
			if got != c.expected {
				t.Errorf("pascalCase(%v) = %q; want %q", c.input, got, c.expected)
			}
		})
	}
}

func TestNaming_TitleCase(t *testing.T) {
	cases := []struct {
		input    []string
		expected string
	}{
		{[]string{"address", "line", "1"}, "Address Line 1"},
		{[]string{"acknowledged", "at"}, "Acknowledged At"},
		{[]string{"acknowledgement", "status"}, "Acknowledgement Status"},
	}

	for _, c := range cases {
		t.Run(c.expected, func(t *testing.T) {
			got := titleCase(c.input)
			if got != c.expected {
				t.Errorf("titleCase(%v) = %q; want %q", c.input, got, c.expected)
			}
		})
	}
}

func TestNaming_LooksLikeAbbreviation(t *testing.T) {
	cases := []struct {
		token    string
		expected bool
	}{
		{"ACCT", true},
		{"CD", true},
		{"XML", true},
		{"customer", false},
		{"account", false},
		{"ID", true},
		{"NO", false},
		{"OF", false},
	}

	for _, c := range cases {
		t.Run(c.token, func(t *testing.T) {
			got := looksLikeAbbreviation(c.token)
			if got != c.expected {
				t.Errorf("looksLikeAbbreviation(%q) = %v; want %v", c.token, got, c.expected)
			}
		})
	}
}

func TestNaming_PascalCaseToWords(t *testing.T) {
	cases := []struct {
		input    string
		expected []string
	}{
		{"IssuerAddress1", []string{"Issuer", "Address1"}},
		{"AcknowledgedAt", []string{"Acknowledged", "At"}},
		{"AcknowledgementStatus", []string{"Acknowledgement", "Status"}},
		{"CustomerIdentifier", []string{"Customer", "Identifier"}},
		{"AccountNumber", []string{"Account", "Number"}},
		{"XMLParser", []string{"XML", "Parser"}},
	}

	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			got := pascalCaseToWords(c.input)
			if len(got) != len(c.expected) {
				t.Errorf("pascalCaseToWords(%q) = %v; want %v", c.input, got, c.expected)
				return
			}
			for i := range got {
				if got[i] != c.expected[i] {
					t.Errorf("pascalCaseToWords(%q)[%d] = %q; want %q", c.input, i, got[i], c.expected[i])
				}
			}
		})
	}
}

func TestNaming_PinnedRegressions(t *testing.T) {
	tableCtx := `table "issuer_address" in schema "orm"`

	result := deriveTermNamesDeterministic([]string{"ADDRESS", "line", "1"}, "address_line_1", tableCtx)
	if result.SemanticName != "IssuerAddress1" {
		t.Errorf("address_line_1 + issuer_address → SemanticName = %q; want %q", result.SemanticName, "IssuerAddress1")
	}
	if result.BusinessName != "Issuer Address Line 1" {
		t.Errorf("address_line_1 + issuer_address → BusinessName = %q; want %q", result.BusinessName, "Issuer Address Line 1")
	}

	result = deriveTermNamesDeterministic([]string{"ACKNOWLEDGED", "at"}, "ack_at", "")
	if result.SemanticName != "AcknowledgedAt" {
		t.Errorf("ack_at → SemanticName = %q; want %q", result.SemanticName, "AcknowledgedAt")
	}

	result = deriveTermNamesDeterministic([]string{"ACKNOWLEDGED", "STATUS"}, "ack_status", "")
	if result.SemanticName != "AcknowledgedStatus" {
		t.Errorf("ack_status → SemanticName = %q; want %q", result.SemanticName, "AcknowledgedStatus")
	}
}

func TestNaming_ExtractTableNameFromContext(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{`table "issuer_address" in schema "orm"`, "issuer_address"},
		{`table "customer" in schema "public"`, "customer"},
		{`public.customer`, "customer"},
		{`metadata.issuer_address`, "issuer_address"},
		{`table "orders" in schema "sales"`, "orders"},
	}

	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			got := extractTableNameFromContext(c.input)
			if got != c.expected {
				t.Errorf("extractTableNameFromContext(%q) = %q; want %q", c.input, got, c.expected)
			}
		})
	}
}

func TestNaming_IsGenericWord(t *testing.T) {
	cases := []struct {
		word     string
		expected bool
	}{
		{"city", true},
		{"CITY", true},
		{"Status", true},
		{"name", true},
		{"code", true},
		{"number", true},
		{"amount", true},
		{"id", true},
		{"customer", false},
		{"issuer", false},
		{"account", false},
		{"address", false},
	}

	for _, c := range cases {
		t.Run(c.word, func(t *testing.T) {
			got := isGenericWord(c.word)
			if got != c.expected {
				t.Errorf("isGenericWord(%q) = %v; want %v", c.word, got, c.expected)
			}
		})
	}
}
