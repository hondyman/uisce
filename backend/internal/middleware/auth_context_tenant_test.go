package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/stretchr/testify/require"
)

// observed is what a downstream handler sees after AuthContextMiddleware ran.
type observed struct {
	tenantHeader string
	userHeader   string
	auth         security.AuthInfo
	hasAuth      bool
	active       string
	hasActive    bool
}

func runTenantCase(t *testing.T, claims jwt.MapClaims, token bool, headers map[string]string) observed {
	t.Helper()
	sm := newTestSecurityManager()
	var got observed
	h := AuthContextMiddleware(sm)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.tenantHeader = r.Header.Get("X-Tenant-ID")
		got.userHeader = r.Header.Get("X-User-ID")
		got.auth, got.hasAuth = security.AuthInfoFromContext(r.Context())
		if got.hasAuth {
			got.active, got.hasActive = got.auth.ActiveTenant()
		}
	}))
	req := httptest.NewRequest("GET", "/x", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if token {
		req.Header.Set("Authorization", "Bearer "+signTestToken(sm, claims))
	}
	h.ServeHTTP(httptest.NewRecorder(), req)
	return got
}

func tokenClaims(extra jwt.MapClaims) jwt.MapClaims {
	c := jwt.MapClaims{
		"user_id": uuid.NewString(),
		"sub":     uuid.NewString(),
		"exp":     time.Now().Add(time.Hour).Unix(),
	}
	for k, v := range extra {
		c[k] = v
	}
	return c
}

func TestNoToken_ClientIdentityHeadersAreDiscarded(t *testing.T) {
	got := runTenantCase(t, nil, false, map[string]string{
		"X-Tenant-ID": uuid.NewString(), "X-User-ID": "attacker",
	})
	require.False(t, got.hasAuth)
	require.Empty(t, got.tenantHeader, "an unauthenticated request must not carry a tenant")
	require.Empty(t, got.userHeader, "an unauthenticated request must not carry a user")
}

func TestInvalidToken_ClientIdentityHeadersAreDiscarded(t *testing.T) {
	sm := newTestSecurityManager()
	var tenant, user string
	h := AuthContextMiddleware(sm)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenant, user = r.Header.Get("X-Tenant-ID"), r.Header.Get("X-User-ID")
	}))
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Bearer not-a-valid-jwt")
	req.Header.Set("X-Tenant-ID", uuid.NewString())
	req.Header.Set("X-User-ID", "attacker")
	h.ServeHTTP(httptest.NewRecorder(), req)
	require.Empty(t, tenant)
	require.Empty(t, user)
}

func TestSingleTenantToken_ClaimOverridesSpoofedHeader(t *testing.T) {
	mine, victim := uuid.NewString(), uuid.NewString()
	got := runTenantCase(t, tokenClaims(jwt.MapClaims{"tenant_id": mine}), true, map[string]string{"X-Tenant-ID": victim})
	require.Equal(t, mine, got.tenantHeader)
	require.True(t, got.hasActive)
	require.Equal(t, mine, got.active)
}

func TestMultiTenantToken_RequiresAnAuthorizedExplicitSelection(t *testing.T) {
	a, b, outsider := uuid.NewString(), uuid.NewString(), uuid.NewString()
	claims := func() jwt.MapClaims { return tokenClaims(jwt.MapClaims{"tenant_ids": []string{a, b}}) }

	t.Run("authorized selection is honored", func(t *testing.T) {
		got := runTenantCase(t, claims(), true, map[string]string{"X-Tenant-ID": b})
		require.True(t, got.hasActive)
		require.Equal(t, b, got.active)
		require.Equal(t, b, got.tenantHeader)
	})
	t.Run("a tenant the token does not authorize is rejected, not guessed around", func(t *testing.T) {
		got := runTenantCase(t, claims(), true, map[string]string{"X-Tenant-ID": outsider})
		require.False(t, got.hasActive)
		require.Empty(t, got.tenantHeader)
	})
	t.Run("no selection means no tenant, not the first one", func(t *testing.T) {
		got := runTenantCase(t, claims(), true, nil)
		require.False(t, got.hasActive, "several tenants and no selection is ambiguous")
		require.Empty(t, got.tenantHeader)
	})
}

func TestTenantlessNonAdminToken_HeaderIsNotATenant(t *testing.T) {
	got := runTenantCase(t, tokenClaims(nil), true, map[string]string{"X-Tenant-ID": uuid.NewString()})
	require.True(t, got.hasAuth)
	require.False(t, got.hasActive, "a token without a tenant must not gain one from a client header")
	require.Empty(t, got.tenantHeader)
}

func TestGlobalAdmin_MustSelectTenantExplicitlyAndValidly(t *testing.T) {
	admin := func() jwt.MapClaims { return tokenClaims(jwt.MapClaims{"roles": []string{"global_admin"}}) }
	sel := uuid.NewString()

	got := runTenantCase(t, admin(), true, map[string]string{"X-Tenant-ID": sel})
	require.True(t, got.hasActive)
	require.Equal(t, sel, got.active)

	got = runTenantCase(t, admin(), true, map[string]string{"X-Tenant-ID": "not-a-uuid"})
	require.False(t, got.hasActive)

	got = runTenantCase(t, admin(), true, nil)
	require.False(t, got.hasActive, "an admin with no selection has no tenant")
}
