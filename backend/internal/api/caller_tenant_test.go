package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/stretchr/testify/require"
)

func reqWithAuth(auth *security.AuthInfo) *http.Request {
	r := httptest.NewRequest("GET", "/x", nil)
	if auth != nil {
		r = r.WithContext(security.WithAuthInfo(r.Context(), *auth))
	}
	return r
}

func TestCallerTenant(t *testing.T) {
	mine, other := uuid.NewString(), uuid.NewString()
	user := &security.AuthInfo{UserID: "u", TenantIDs: []string{mine}, ActiveTenantID: mine}
	admin := &security.AuthInfo{UserID: "a", IsGlobalAdmin: true, TenantIDs: []string{mine}, ActiveTenantID: mine}
	multi := &security.AuthInfo{UserID: "m", TenantIDs: []string{mine, other}} // no selection
	adminNone := &security.AuthInfo{UserID: "a2", IsGlobalAdmin: true}          // no tenant at all

	cases := []struct {
		name      string
		auth      *security.AuthInfo
		requested []string
		wantOK    bool
		wantCode  int
		want      string
	}{
		{"unauthenticated is refused", nil, nil, false, 401, ""},
		{"no tenant and no request is refused, with no gold-copy default", &security.AuthInfo{UserID: "u"}, nil, false, 401, ""},
		{"verified tenant, nothing requested", user, nil, true, 200, mine},
		{"requested equals verified", user, []string{mine}, true, 200, mine},
		{"another tenant requested by a non-admin is refused", user, []string{other}, false, 403, ""},
		{"a malformed tenant id is refused", user, []string{"not-a-uuid"}, false, 400, ""},
		{"a global admin may select another tenant explicitly", admin, []string{other}, true, 200, other},
		{"a global admin with no tenant must name one", adminNone, nil, false, 401, ""},
		{"a global admin with no tenant may name one", adminNone, []string{other}, true, 200, other},
		{"several tenants and none selected is refused, not the first", multi, nil, false, 401, ""},
		{"several tenants, an authorized one named", multi, []string{other}, true, 200, other},
		{"several tenants, an unauthorized one named", multi, []string{uuid.NewString()}, false, 403, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			got, ok := callerTenant(w, reqWithAuth(tc.auth), tc.requested...)
			require.Equal(t, tc.wantOK, ok)
			if tc.wantOK {
				require.Equal(t, tc.want, got)
			} else {
				require.Equal(t, tc.wantCode, w.Code)
			}
		})
	}
}
