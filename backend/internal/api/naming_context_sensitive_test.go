package api

import (
	"testing"
)

// TestDeriveTermNamesDeterministic_ContextSensitive verifies the ContextSensitive
// flag on derivedTermNames — the linchpin for PR-β's pre-grouped dispatch.
// The flag controls whether the grouping key includes table context: context-free
// columns (same name, different tables) collapse into one group; context-sensitive
// columns (address-line, bare-generic, abbreviation expansion) split per context.
func TestDeriveTermNamesDeterministic_ContextSensitive(t *testing.T) {
	tests := []struct {
		name            string
		resolvedTokens  []string
		rawName         string
		tableCtx        string
		wantCS          bool
		wantSemantic    string
	}{
		{
			name:           "address_line_context_fires",
			resolvedTokens: []string{"address", "line", "1"},
			rawName:        "address_line_1",
			tableCtx:       `table "issuer_address" in schema "orm"`,
			wantCS:         true,
			wantSemantic:   "IssuerAddress1",
		},
		{
			name:           "bare_generic_fires",
			resolvedTokens: []string{"city"},
			rawName:        "city",
			tableCtx:       `table "employees" in schema "hr"`,
			wantCS:         true,
			wantSemantic:   "City",
		},
		{
			name:           "pure_pascal_no_context",
			resolvedTokens: []string{"customer", "identifier"},
			rawName:        "customer_id",
			tableCtx:       `table "orders" in schema "orm"`,
			wantCS:         false,
			wantSemantic:   "CustomerIdentifier",
		},
		{
			name:           "address_line_different_table",
			resolvedTokens: []string{"address", "line", "1"},
			rawName:        "address_line_1",
			tableCtx:       `table "customer_address" in schema "orm"`,
			wantCS:         true,
			wantSemantic:   "CustomerAddress1",
		},
		{
			name:           "bare_generic_same_name_as_table",
			resolvedTokens: []string{"status"},
			rawName:        "status",
			tableCtx:       `table "status" in schema "orm"`,
			wantCS:         false,
			wantSemantic:   "Status",
		},
		{
			name:           "empty_tokens",
			resolvedTokens: []string{},
			rawName:        "",
			tableCtx:       "",
			wantCS:         false,
			wantSemantic:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveTermNamesDeterministic(tt.resolvedTokens, tt.rawName, tt.tableCtx)
			if got.ContextSensitive != tt.wantCS {
				t.Errorf("ContextSensitive = %v, want %v (source=%q)", got.ContextSensitive, tt.wantCS, got.source)
			}
			if got.SemanticName != tt.wantSemantic {
				t.Errorf("SemanticName = %q, want %q", got.SemanticName, tt.wantSemantic)
			}
		})
	}
}

// TestDeriveTermNamesDeterministic_AddressLineRuleTwoTables proves the key
// correctness property: same column name in two different tables with the
// address-line rule produces two different derived names AND both have
// ContextSensitive = true. This is the case that drove the grouping-key fix.
func TestDeriveTermNamesDeterministic_AddressLineRuleTwoTables(t *testing.T) {
	tokens := []string{"address", "line", "1"}

	issuer := deriveTermNamesDeterministic(tokens, "address_line_1", `table "issuer_address" in schema "orm"`)
	customer := deriveTermNamesDeterministic(tokens, "address_line_1", `table "customer_address" in schema "orm"`)

	if !issuer.ContextSensitive {
		t.Error("issuer: ContextSensitive should be true")
	}
	if !customer.ContextSensitive {
		t.Error("customer: ContextSensitive should be true")
	}
	if issuer.SemanticName == customer.SemanticName {
		t.Errorf("same SemanticName for different tables: %q", issuer.SemanticName)
	}
	if issuer.SemanticName != "IssuerAddress1" {
		t.Errorf("issuer SemanticName = %q, want IssuerAddress1", issuer.SemanticName)
	}
	if customer.SemanticName != "CustomerAddress1" {
		t.Errorf("customer SemanticName = %q, want CustomerAddress1", customer.SemanticName)
	}
}

// TestDeriveTermNamesDeterministic_CustomerIdInThreeTables proves the dedup
// property: customer_id in three different tables → all ContextSensitive=false,
// all derive the same name. Under the refined grouping key, these three items
// collapse into one group (one LLM call), not three.
func TestDeriveTermNamesDeterministic_CustomerIdInThreeTables(t *testing.T) {
	tokens := []string{"customer", "identifier"}

	tables := []string{
		`table "orders" in schema "orm"`,
		`table "invoices" in schema "orm"`,
		`table "payments" in schema "orm"`,
	}

	for _, tableCtx := range tables {
		got := deriveTermNamesDeterministic(tokens, "customer_id", tableCtx)
		if got.ContextSensitive {
			t.Errorf("ContextSensitive=true for tableCtx=%q — dedup will fail", tableCtx)
		}
		if got.SemanticName != "CustomerIdentifier" {
			t.Errorf("SemanticName=%q, want CustomerIdentifier for tableCtx=%q", got.SemanticName, tableCtx)
		}
	}
}
