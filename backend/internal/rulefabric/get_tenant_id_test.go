package rulefabric

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/security"
)

func TestGetTenantID(t *testing.T) {
	const (
		tenantA = "11111111-1111-1111-1111-111111111111"
		tenantB = "22222222-2222-2222-2222-222222222222"
		adminTenant = "33333333-3333-3333-3333-333333333333"
	)

	makeReq := func(auth security.AuthInfo, headerVal string) *http.Request {
		r := httptest.NewRequest("GET", "/", nil)
		if headerVal != "" {
			r.Header.Set("X-Tenant-ID", headerVal)
		}
		ctx := security.WithAuthInfo(context.Background(), auth)
		return r.WithContext(ctx)
	}

	cases := []struct {
		name       string
		auth       security.AuthInfo
		header     string
		wantTenant string
		wantErr    bool
	}{
		{
			name:       "JWT only — returns JWT tenant",
			auth:       security.AuthInfo{UserID: "user-1", TenantIDs: []string{tenantA}, Roles: []string{"user"}},
			header:     "",
			wantTenant: tenantA,
			wantErr:    false,
		},
		{
			name:       "header matches JWT tenant — returns header value",
			auth:       security.AuthInfo{UserID: "user-1", TenantIDs: []string{tenantA}, Roles: []string{"user"}},
			header:     tenantA,
			wantTenant: tenantA,
			wantErr:    false,
		},
		{
			name:       "non-admin spoof — header does NOT match JWT — fail-closed",
			auth:       security.AuthInfo{UserID: "user-1", TenantIDs: []string{tenantA}, Roles: []string{"user"}},
			header:     tenantB,
			wantTenant: "",
			wantErr:    true,
		},
		{
			name:       "admin JWT + different header — admin override honored (Variant A)",
			auth:       security.AuthInfo{UserID: "admin-1", TenantIDs: []string{adminTenant}, Roles: []string{"global_admin"}, IsGlobalAdmin: true},
			header:     tenantA,
			wantTenant: tenantA,
			wantErr:    false,
		},
		{
			name:       "admin JWT — no header — returns admin's own tenant",
			auth:       security.AuthInfo{UserID: "admin-1", TenantIDs: []string{adminTenant}, Roles: []string{"global_admin"}, IsGlobalAdmin: true},
			header:     "",
			wantTenant: adminTenant,
			wantErr:    false,
		},
		{
			name:       "no auth info — returns error",
			auth:       security.AuthInfo{UserID: "user-1", TenantIDs: []string{}},
			header:     "",
			wantTenant: "",
			wantErr:    true,
		},
		{
			name:       "malformed UUID in header — fail-closed",
			auth:       security.AuthInfo{UserID: "user-1", TenantIDs: []string{tenantA}, Roles: []string{"user"}},
			header:     "not-a-uuid",
			wantTenant: "",
			wantErr:    true,
		},
		{
			name:       "whitespace header treated as absent",
			auth:       security.AuthInfo{UserID: "user-1", TenantIDs: []string{tenantA}, Roles: []string{"user"}},
			header:     "   ",
			wantTenant: tenantA,
			wantErr:    false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := makeReq(c.auth, c.header)
			got, err := getTenantID(r)
			if c.wantErr {
				if err == nil {
					t.Errorf("getTenantID(): expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("getTenantID(): unexpected error: %v", err)
				return
			}
			if got.String() != c.wantTenant {
				t.Errorf("getTenantID(): got %s, want %s", got.String(), c.wantTenant)
			}
		})
	}
}

// TestGetTenantID_AfterHotfix verifies the post-hotfix behavior: spoof attempts
// (header != JWT tenant, non-admin) are rejected rather than honored.
func TestGetTenantID_AfterHotfix(t *testing.T) {
	// Setup: user authenticated as tenantA, tries to operate as tenantB
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Tenant-ID", "22222222-2222-2222-2222-222222222222") // victim tenant
	ctx := security.WithAuthInfo(context.Background(), security.AuthInfo{
		UserID:    "user-1",
		TenantIDs:  []string{"11111111-1111-1111-1111-111111111111"},
		Roles:      []string{"analyst"},
		IsGlobalAdmin: false,
	})
	r = r.WithContext(ctx)

	got, err := getTenantID(r)
	if err == nil {
		t.Errorf("getTenantID(): expected error for spoof attempt (header=tenantB, JWT=tenantA), got nil. Got: %s", got.String())
	} else {
		// Error is expected behavior — spoof was blocked
		t.Logf("getTenantID(): correctly rejected spoof attempt: %v", err)
	}
}

// TestGetTenantID_AdminOverride confirms Variant A: global_admin can specify any tenant.
func TestGetTenantID_AdminOverride(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Tenant-ID", "22222222-2222-2222-2222-222222222222")
	ctx := security.WithAuthInfo(context.Background(), security.AuthInfo{
		UserID:    "admin-1",
		TenantIDs:  []string{"33333333-3333-3333-3333-333333333333"},
		Roles:      []string{"global_admin"},
		IsGlobalAdmin: true,
	})
	r = r.WithContext(ctx)

	got, err := getTenantID(r)
	if err != nil {
		t.Errorf("getTenantID(): unexpected error for admin override: %v", err)
		return
	}
	if got != (uuid.UUID{}) && got.String() != "22222222-2222-2222-2222-222222222222" {
		// Admin specified tenantB, should get tenantB
		t.Errorf("getTenantID(): admin override: got %s, want 22222222-2222-2222-2222-222222222222", got.String())
	}
}
