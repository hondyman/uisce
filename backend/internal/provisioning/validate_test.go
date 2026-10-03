package provisioning

import (
	"strings"
	"testing"
)

func TestValidateTenantCode(t *testing.T) {
	tests := []struct {
		name string
		code string
		ok   bool
	}{
		{"simple", "acme", true},
		{"digits and underscore", "acme_2", true},
		{"min length", "ab", true},
		{"max length", "a" + strings.Repeat("b", 40), true},
		{"overlong", "a" + strings.Repeat("b", 41), false},
		{"empty", "", false},
		{"single char", "a", false},
		{"leading digit", "1acme", false},
		{"leading underscore", "_acme", false},
		{"semicolon injection", "x; DROP DATABASE alpha;--", false},
		{"semicolon no spaces", "x;drop", false},
		{"double quote", `x"y`, false},
		{"single quote", "x'y", false},
		{"space", "ac me", false},
		{"uppercase", "Acme", false},
		{"hyphen", "ac-me", false},
		{"dot", "ac.me", false},
		{"slash", "ac/me", false},
		{"backslash", `ac\me`, false},
		{"comment", "ac--me", false},
		{"newline", "acme\n", false},
		{"nul byte", "ac\x00me", false},
		{"unicode letter", "acmé", false},
		{"unicode digit", "acme٣", false},
		{"fullwidth", "ａｃｍｅ", false},
		{"leading dash (flag)", "-acme", false},
		{"connection string", "a=b host=evil", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateTenantCode(tc.code)
			if (err == nil) != tc.ok {
				t.Fatalf("ValidateTenantCode(%q) err=%v, want ok=%v", tc.code, err, tc.ok)
			}
			if _, derr := TenantDatabaseName(tc.code); (derr == nil) != tc.ok {
				t.Fatalf("TenantDatabaseName(%q) err=%v, want ok=%v", tc.code, derr, tc.ok)
			}
		})
	}
}

func TestTenantDatabaseName(t *testing.T) {
	got, err := TenantDatabaseName("acme_2")
	if err != nil || got != "tenant_acme_2" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestValidateDatabaseName(t *testing.T) {
	tests := []struct {
		name string
		db   string
		ok   bool
	}{
		{"tenant db", "tenant_acme", true},
		{"gold copy", "alpha", true},
		{"mixed case", "Alpha", true},
		{"max length", "d" + strings.Repeat("x", 62), true},
		{"overlong", "d" + strings.Repeat("x", 63), false},
		{"empty", "", false},
		{"semicolon", "x; DROP DATABASE alpha;--", false},
		{"double quote", `x" ; drop`, false},
		{"single quote", "x'y", false},
		{"space", "a b", false},
		{"leading digit", "1a", false},
		{"leading dash", "-h", false},
		{"conninfo", "dbname=x host=evil", false},
		{"unicode", "tenant_é", false},
		{"newline", "a\nb", false},
		{"nul", "a\x00b", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateDatabaseName(tc.db); (err == nil) != tc.ok {
				t.Fatalf("ValidateDatabaseName(%q) err=%v, want ok=%v", tc.db, err, tc.ok)
			}
		})
	}
}
