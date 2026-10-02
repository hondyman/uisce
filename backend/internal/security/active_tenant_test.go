package security

import "testing"

func TestActiveTenant_NeverGuesses(t *testing.T) {
	cases := []struct {
		name string
		auth AuthInfo
		want string
		ok   bool
	}{
		{"no tenants", AuthInfo{}, "", false},
		{"one tenant is unambiguous", AuthInfo{TenantIDs: []string{"a"}}, "a", true},
		{"several tenants, none selected", AuthInfo{TenantIDs: []string{"a", "b"}}, "", false},
		{"several tenants, one selected", AuthInfo{TenantIDs: []string{"a", "b"}, ActiveTenantID: "b"}, "b", true},
		{"a blank tenant is not a tenant", AuthInfo{TenantIDs: []string{"  "}}, "", false},
	}
	for _, tc := range cases {
		got, ok := tc.auth.ActiveTenant()
		if got != tc.want || ok != tc.ok {
			t.Errorf("%s: got (%q,%v), want (%q,%v)", tc.name, got, ok, tc.want, tc.ok)
		}
	}
}

func TestResolveTenantID_NoSelectionOnlyIfUnambiguous(t *testing.T) {
	if _, ok := ResolveTenantID(AuthInfo{TenantIDs: []string{"a", "b"}}, ""); ok {
		t.Fatal("several tenants and no selection must not resolve to the first")
	}
	if got, ok := ResolveTenantID(AuthInfo{TenantIDs: []string{"a"}}, ""); !ok || got != "a" {
		t.Fatalf("a single tenant is unambiguous, got (%q,%v)", got, ok)
	}
	if got, ok := ResolveTenantID(AuthInfo{TenantIDs: []string{"a", "b"}}, "b"); !ok || got != "b" {
		t.Fatalf("an authorized selection must be honored, got (%q,%v)", got, ok)
	}
	if _, ok := ResolveTenantID(AuthInfo{TenantIDs: []string{"a", "b"}}, "c"); ok {
		t.Fatal("an unauthorized selection must be refused")
	}
	if got, ok := ResolveTenantID(AuthInfo{IsGlobalAdmin: true}, "c"); !ok || got != "c" {
		t.Fatalf("a global admin may select explicitly, got (%q,%v)", got, ok)
	}
}
