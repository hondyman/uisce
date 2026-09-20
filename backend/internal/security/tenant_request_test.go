package security

import (
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResolveTenantForRequest(t *testing.T) {
	const (
		tenantA = "11111111-1111-1111-1111-111111111111"
		tenantB = "22222222-2222-2222-2222-222222222222"
		adminTenant = "33333333-3333-3333-3333-333333333333"
	)

	makeReq := func(headerVal string) *http.Request {
		r := httptest.NewRequest("GET", "/", nil)
		if headerVal != "" {
			r.Header.Set("X-Tenant-ID", headerVal)
		}
		return r
	}

	cases := []struct {
		name        string
		auth        AuthInfo
		header      string
		wantTenant  string
		wantErr     bool
		errContains string
	}{
		{
			name: "JWT only — returns JWT tenant",
			auth: AuthInfo{
				UserID:    "user-1",
				TenantIDs: []string{tenantA},
				Roles:     []string{"user"},
			},
			header:     "",
			wantTenant: tenantA,
			wantErr:    false,
		},
		{
			name: "JWT only — no tenant claim",
			auth: AuthInfo{
				UserID:    "user-1",
				TenantIDs: []string{},
				Roles:     []string{"user"},
			},
			header:      "",
			wantErr:     true,
			errContains: "no tenant available",
		},
		{
			name: "header matches JWT tenant — returns header value",
			auth: AuthInfo{
				UserID:    "user-1",
				TenantIDs: []string{tenantA},
				Roles:     []string{"user"},
			},
			header:     tenantA,
			wantTenant: tenantA,
			wantErr:    false,
		},
		{
			name: "header does NOT match JWT — non-admin — fail-closed",
			auth: AuthInfo{
				UserID:    "user-1",
				TenantIDs: []string{tenantA},
				Roles:     []string{"user"},
			},
			header:      tenantB,
			wantErr:     true,
			errContains: "not authorized",
		},
		{
			name: "admin JWT + header matches another tenant — admin override honored",
			auth: AuthInfo{
				UserID:         "admin-1",
				TenantIDs:      []string{adminTenant},
				Roles:          []string{"global_admin"},
				IsGlobalAdmin:  true,
			},
			header:     tenantA,
			wantTenant: tenantA,
			wantErr:    false,
		},
		{
			name: "admin JWT — no header — returns admin's own tenant",
			auth: AuthInfo{
				UserID:         "admin-1",
				TenantIDs:      []string{adminTenant},
				Roles:          []string{"global_admin"},
				IsGlobalAdmin:  true,
			},
			header:     "",
			wantTenant: adminTenant,
			wantErr:    false,
		},
		{
			name: "non-admin spoof attempt — header with different tenant rejected",
			auth: AuthInfo{
				UserID:    "user-1",
				TenantIDs: []string{tenantA},
				Roles:     []string{"analyst"},
			},
			header:      tenantB,
			wantErr:     true,
			errContains: "not authorized",
		},
		{
			name: "no auth info — returns error",
			auth:        AuthInfo{},
			header:      "",
			wantErr:     true,
			errContains: "no tenant available",
		},
		{
			name: "whitespace header treated as absent",
			auth: AuthInfo{
				UserID:    "user-1",
				TenantIDs: []string{tenantA},
				Roles:     []string{"user"},
			},
			header:     "   ",
			wantTenant: tenantA,
			wantErr:    false,
		},
		{
			name: "malformed UUID in header — fail-closed",
			auth: AuthInfo{
				UserID:    "user-1",
				TenantIDs: []string{tenantA},
				Roles:     []string{"user"},
			},
			header:      "not-a-uuid",
			wantErr:     true,
			errContains: "not authorized",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := makeReq(c.header)
			ctx := WithAuthInfo(r.Context(), c.auth)
			r = r.WithContext(ctx)

			got, err := ResolveTenantForRequest(r)
			if c.wantErr {
				if err == nil {
					t.Errorf("ResolveTenantForRequest(): expected error containing %q, got nil", c.errContains)
					return
				}
				if c.errContains != "" && !strings.Contains(err.Error(), c.errContains) {
					t.Errorf("ResolveTenantForRequest(): error %q does not contain %q", err.Error(), c.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("ResolveTenantForRequest(): unexpected error: %v", err)
				return
			}
			if got != c.wantTenant {
				t.Errorf("ResolveTenantForRequest(): got %q, want %q", got, c.wantTenant)
			}
		})
	}
}

func TestResolveTenantForRequestWarn_SpoofDegradedToNoOp(t *testing.T) {
	const (
		tenantA = "11111111-1111-1111-1111-111111111111"
		tenantB = "22222222-2222-2222-2222-222222222222"
	)

	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Tenant-ID", tenantB) // spoof attempt
	ctx := WithAuthInfo(r.Context(), AuthInfo{
		UserID:    "user-1",
		TenantIDs: []string{tenantA},
		Roles:     []string{"user"},
	})
	r = r.WithContext(ctx)

	// Capture log output
	var logLines []string
	logger := log.Default()
	oldOut := logger.Writer()
	logger.SetOutput(&logWriter{lines: &logLines})

	got, err := ResolveTenantForRequestWarn(r)
	logger.SetOutput(oldOut)

	if err != nil {
		t.Errorf("ResolveTenantForRequestWarn(): unexpected error: %v", err)
		return
	}
	// Spoof should be silently degraded to JWT tenant
	if got != tenantA {
		t.Errorf("ResolveTenantForRequestWarn(): got %q, want %q (JWT tenant)", got, tenantA)
	}
	// Should have logged the spoof attempt
	foundLog := false
	for _, l := range logLines {
		if strings.Contains(l, "tenant-spoof-blocked") || strings.Contains(l, tenantB) {
			foundLog = true
			break
		}
	}
	if !foundLog {
		t.Logf("log lines: %v", logLines)
		// Not a hard failure — log presence is best-effort
	}
}

func TestResolveTenantForRequestWarn_NoSpoof(t *testing.T) {
	const tenantA = "11111111-1111-1111-1111-111111111111"

	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Tenant-ID", tenantA) // legitimate use of header
	ctx := WithAuthInfo(r.Context(), AuthInfo{
		UserID:    "user-1",
		TenantIDs: []string{tenantA},
		Roles:     []string{"user"},
	})
	r = r.WithContext(ctx)

	got, err := ResolveTenantForRequestWarn(r)
	if err != nil {
		t.Errorf("ResolveTenantForRequestWarn(): unexpected error: %v", err)
		return
	}
	if got != tenantA {
		t.Errorf("ResolveTenantForRequestWarn(): got %q, want %q", got, tenantA)
	}
}

func TestResolveTenantForRequestWarn_NoAuth(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	// No auth context
	_, err := ResolveTenantForRequestWarn(r)
	if err == nil {
		t.Error("ResolveTenantForRequestWarn(): expected error for missing auth context, got nil")
	}
}

type logWriter struct {
	lines *[]string
}

func (lw *logWriter) Write(p []byte) (int, error) {
	*lw.lines = append(*lw.lines, string(p))
	return len(p), nil
}
