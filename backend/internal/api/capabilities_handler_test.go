package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/hondyman/uisce/backend/internal/security"
)

// capabilitiesRequest builds a request that already carries verified auth
// context, mirroring what AuthContextMiddleware injects in production. There is
// no JWT here on purpose: resolveProfileKey must read the profile from AuthInfo
// and the IAM tables, never from a client-supplied header.
func capabilitiesRequest(method, path, tenantID string, auth security.AuthInfo, extraHeaders map[string]string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	req = req.WithContext(security.WithAuthInfo(req.Context(), auth))
	if tenantID != "" {
		req.Header.Set("X-Tenant-ID", tenantID)
	}
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}
	return req
}

// newCapsTestRouter wires the handler over a sqlmock DB and returns the mock so
// each test can script the gold-copy lookup, the optional IAM lookup, and the
// policy query in order.
func newCapsTestRouter(t *testing.T) (*chi.Mux, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	h := NewCapabilitiesHandler(sqlx.NewDb(db, "sqlmock"))
	r := chi.NewRouter()
	h.RegisterRoutes(r)
	return r, mock
}

// expectGoldCopy scripts the gold-copy tenant lookup, which the handler always
// makes before evaluating policies.
func expectGoldCopy(mock sqlmock.Sqlmock, gold uuid.UUID) {
	mock.ExpectQuery(`uisce_gold_copy_tenant_id`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(gold))
}

func decodeCaps(t *testing.T, body []byte) map[string]bool {
	t.Helper()
	var caps map[string]bool
	if err := json.Unmarshal(body, &caps); err != nil {
		t.Fatalf("decode capabilities: %v (body: %s)", err, body)
	}
	return caps
}

// The endpoint must refuse a caller with no identifiable tenant rather than
// falling back to a guessed one. This is the same rule security.ResolveTenantID
// enforces platform-wide.
func TestCapabilities_RequiresTenant(t *testing.T) {
	r, _ := newCapsTestRouter(t)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, capabilitiesRequest(http.MethodGet, "/capabilities", "", security.AuthInfo{UserID: "u"}, nil))

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for a request with no tenant, got %d (body: %s)", w.Code, w.Body.String())
	}
}

// A caller must never receive another profile's grants. This is the regression
// that motivated the target_profile_key predicate in the policy query: without
// it a BASE_USER was told it held PLATFORM_OPERATOR's menu grants.
func TestCapabilities_DoesNotLeakOtherProfileGrants(t *testing.T) {
	r, mock := newCapsTestRouter(t)
	tenant, gold := uuid.New(), uuid.New()

	expectGoldCopy(mock, gold)
	// The seed keeps PLATFORM_OPERATOR allows at priority 100 and the
	// BASE_USER deny at 200. Only the BASE_USER rows may reach a BASE_USER.
	mock.ExpectQuery(`FROM studio.tenant_abac_policies`).
		WithArgs(profileBaseUser, tenant, gold).
		WillReturnRows(sqlmock.NewRows([]string{"action_attribute", "effect", "priority"}).
			AddRow("menu:platform", "deny", 200))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, capabilitiesRequest(http.MethodGet, "/capabilities", tenant.String(),
		security.AuthInfo{UserID: "u"}, nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	caps := decodeCaps(t, w.Body.Bytes())
	if v, ok := caps["menu:platform"]; !ok || v {
		t.Fatalf("BASE_USER must be denied menu:platform, got %v (all: %v)", v, caps)
	}
	for _, key := range []string{"menu:organization", "menu:security", "menu:system", "menu:entitlements"} {
		if _, leaked := caps[key]; leaked {
			t.Fatalf("BASE_USER received PLATFORM_OPERATOR grant %q: %v", key, caps)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// Deny must win over an allow of equal priority regardless of row order, so a
// tenant-level allow can never silently re-grant a denied menu.
func TestCapabilities_DenyWinsAtEqualPriority(t *testing.T) {
	r, mock := newCapsTestRouter(t)
	tenant, gold := uuid.New(), uuid.New()

	expectGoldCopy(mock, gold)
	// Deliberately ordered allow-then-deny at the same priority: a naive
	// last-writer-wins merge would return true here.
	mock.ExpectQuery(`FROM studio.tenant_abac_policies`).
		WithArgs(profileBaseUser, tenant, gold).
		WillReturnRows(sqlmock.NewRows([]string{"action_attribute", "effect", "priority"}).
			AddRow("menu:platform", "allow", 100).
			AddRow("menu:platform", "deny", 100))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, capabilitiesRequest(http.MethodGet, "/capabilities", tenant.String(),
		security.AuthInfo{UserID: "u"}, nil))

	caps := decodeCaps(t, w.Body.Bytes())
	if caps["menu:platform"] {
		t.Fatalf("deny must win at equal priority, got %v", caps)
	}
}

// A higher-priority deny must not be undone by a lower-priority allow.
func TestCapabilities_HigherPriorityDenyWins(t *testing.T) {
	r, mock := newCapsTestRouter(t)
	tenant, gold := uuid.New(), uuid.New()

	expectGoldCopy(mock, gold)
	mock.ExpectQuery(`FROM studio.tenant_abac_policies`).
		WithArgs(profileBaseUser, tenant, gold).
		WillReturnRows(sqlmock.NewRows([]string{"action_attribute", "effect", "priority"}).
			AddRow("menu:platform", "deny", 200).
			AddRow("menu:platform", "allow", 100))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, capabilitiesRequest(http.MethodGet, "/capabilities", tenant.String(),
		security.AuthInfo{UserID: "u"}, nil))

	caps := decodeCaps(t, w.Body.Bytes())
	if caps["menu:platform"] {
		t.Fatalf("priority 200 deny must beat priority 100 allow, got %v", caps)
	}
}

// The profile is authorization-critical: a client that could name its own
// profile would simply pick the one with the widest grants. This asserts an
// operator role only takes effect when it arrives via verified auth context.
func TestCapabilities_ProfileNeverComesFromClientHeader(t *testing.T) {
	r, mock := newCapsTestRouter(t)
	tenant, gold := uuid.New(), uuid.New()

	expectGoldCopy(mock, gold)
	// No IAM row (user ID is not a UUID, so resolveProfileKey skips that query),
	// and no operator role in the verified claims -> BASE_USER.
	mock.ExpectQuery(`FROM studio.tenant_abac_policies`).
		WithArgs(profileBaseUser, tenant, gold).
		WillReturnRows(sqlmock.NewRows([]string{"action_attribute", "effect", "priority"}).
			AddRow("menu:platform", "deny", 200))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, capabilitiesRequest(http.MethodGet, "/capabilities", tenant.String(),
		security.AuthInfo{UserID: "not-a-uuid"},
		map[string]string{
			"X-Profile":       profilePlatformOperator,
			"target_profile":  profilePlatformOperator,
			"profile_key":     profilePlatformOperator,
			"X-Tenant-Region": "us-west",
		}))

	caps := decodeCaps(t, w.Body.Bytes())
	if caps["menu:platform"] {
		t.Fatalf("client-supplied profile header escalated the caller: %v", caps)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// A verified operator claim resolves to PLATFORM_OPERATOR and sees the
// operator grants.
func TestCapabilities_OperatorClaimResolvesOperatorProfile(t *testing.T) {
	r, mock := newCapsTestRouter(t)
	tenant, gold := uuid.New(), uuid.New()

	expectGoldCopy(mock, gold)
	mock.ExpectQuery(`FROM studio.tenant_abac_policies`).
		WithArgs(profilePlatformOperator, tenant, gold).
		WillReturnRows(sqlmock.NewRows([]string{"action_attribute", "effect", "priority"}).
			AddRow("menu:platform", "allow", 100).
			AddRow("menu:organization", "allow", 100))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, capabilitiesRequest(http.MethodGet, "/capabilities", tenant.String(),
		security.AuthInfo{UserID: "not-a-uuid", Roles: []string{"global_admin"}}, nil))

	caps := decodeCaps(t, w.Body.Bytes())
	if !caps["menu:platform"] || !caps["menu:organization"] {
		t.Fatalf("operator should hold Platform grants, got %v", caps)
	}
}

// An unknown caller with no roles must fall back to the most restrictive
// profile, never to the widest.
func TestCapabilities_UnknownCallerFallsBackToBaseUser(t *testing.T) {
	r, mock := newCapsTestRouter(t)
	tenant, gold := uuid.New(), uuid.New()

	expectGoldCopy(mock, gold)
	mock.ExpectQuery(`FROM studio.tenant_abac_policies`).
		WithArgs(profileBaseUser, tenant, gold).
		WillReturnRows(sqlmock.NewRows([]string{"action_attribute", "effect", "priority"}).
			AddRow("menu:platform", "deny", 200))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, capabilitiesRequest(http.MethodGet, "/capabilities", tenant.String(),
		security.AuthInfo{UserID: "not-a-uuid", Roles: []string{"some_unknown_role"}}, nil))

	caps := decodeCaps(t, w.Body.Bytes())
	if caps["menu:platform"] {
		t.Fatalf("unknown role must fall back to BASE_USER, got %v", caps)
	}
}

// The tenant's own IAM assignment is authoritative when present, and it lets a
// tenant mint profile keys beyond the two built-in ones.
func TestCapabilities_IAMAssignmentOverridesClaims(t *testing.T) {
	r, mock := newCapsTestRouter(t)
	tenant, gold := uuid.New(), uuid.New()
	userID := uuid.New()

	// getCapabilities resolves the profile BEFORE the gold-copy lookup, so the
	// IAM query is the first one the handler issues.
	mock.ExpectQuery(`FROM iam.user_roles`).
		WithArgs(userID, tenant).
		WillReturnRows(sqlmock.NewRows([]string{"role_name"}).AddRow("TENANT_RISK_OFFICER"))
	expectGoldCopy(mock, gold)
	mock.ExpectQuery(`FROM studio.tenant_abac_policies`).
		WithArgs("TENANT_RISK_OFFICER", tenant, gold).
		WillReturnRows(sqlmock.NewRows([]string{"action_attribute", "effect", "priority"}).
			AddRow("menu:security", "allow", 100))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, capabilitiesRequest(http.MethodGet, "/capabilities", tenant.String(),
		security.AuthInfo{UserID: userID.String(), Roles: []string{"global_admin"}}, nil))

	caps := decodeCaps(t, w.Body.Bytes())
	if !caps["menu:security"] {
		t.Fatalf("tenant IAM profile should have been used, got %v", caps)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// The frontend enforces Menu Designer required_entitlement (a
// target_profile_key) using this header, so it must carry the same
// server-resolved profile the policy query was scoped to. It is a header
// rather than a body field so the capability map keeps its flat bool shape.
func TestCapabilities_ExposesResolvedProfileHeader(t *testing.T) {
	r, mock := newCapsTestRouter(t)
	tenant, gold := uuid.New(), uuid.New()

	expectGoldCopy(mock, gold)
	mock.ExpectQuery(`FROM studio.tenant_abac_policies`).
		WithArgs(profilePlatformOperator, tenant, gold).
		WillReturnRows(sqlmock.NewRows([]string{"action_attribute", "effect", "priority"}).
			AddRow("menu:platform", "allow", 100))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, capabilitiesRequest(http.MethodGet, "/capabilities", tenant.String(),
		security.AuthInfo{UserID: "not-a-uuid", Roles: []string{"global_admin"}}, nil))

	if got := w.Header().Get("X-Resolved-Profile"); got != profilePlatformOperator {
		t.Fatalf("expected X-Resolved-Profile %q, got %q", profilePlatformOperator, got)
	}
}

// An unresolvable caller must be reported as BASE_USER so the frontend can
// fail closed rather than assuming the widest grant.
func TestCapabilities_ResolvedProfileHeaderFallsBackToBaseUser(t *testing.T) {
	r, mock := newCapsTestRouter(t)
	tenant, gold := uuid.New(), uuid.New()

	expectGoldCopy(mock, gold)
	mock.ExpectQuery(`FROM studio.tenant_abac_policies`).
		WithArgs(profileBaseUser, tenant, gold).
		WillReturnRows(sqlmock.NewRows([]string{"action_attribute", "effect", "priority"}).
			AddRow("menu:platform", "deny", 200))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, capabilitiesRequest(http.MethodGet, "/capabilities", tenant.String(),
		security.AuthInfo{UserID: "u"}, nil))

	if got := w.Header().Get("X-Resolved-Profile"); got != profileBaseUser {
		t.Fatalf("expected X-Resolved-Profile %q, got %q", profileBaseUser, got)
	}
}

// No policies configured is a legitimate empty result, not an error: the
// frontend treats {} as "nothing is gated" and falls back safely.
func TestCapabilities_EmptyPolicySetReturnsEmptyMap(t *testing.T) {
	r, mock := newCapsTestRouter(t)
	tenant, gold := uuid.New(), uuid.New()

	expectGoldCopy(mock, gold)
	mock.ExpectQuery(`FROM studio.tenant_abac_policies`).
		WithArgs(profileBaseUser, tenant, gold).
		WillReturnRows(sqlmock.NewRows([]string{"action_attribute", "effect", "priority"}))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, capabilitiesRequest(http.MethodGet, "/capabilities", tenant.String(),
		security.AuthInfo{UserID: "u"}, nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with no policies, got %d", w.Code)
	}
	caps := decodeCaps(t, w.Body.Bytes())
	if len(caps) != 0 {
		t.Fatalf("expected empty capability map, got %v", caps)
	}
}

// A missing table (migration not yet applied) must degrade to {} with 200 so
// the frontend falls through rather than seeing a hard error.
func TestCapabilities_MissingTableDegradesToEmptyMap(t *testing.T) {
	r, mock := newCapsTestRouter(t)
	tenant, gold := uuid.New(), uuid.New()

	expectGoldCopy(mock, gold)
	mock.ExpectQuery(`FROM studio.tenant_abac_policies`).
		WithArgs(profileBaseUser, tenant, gold).
		WillReturnError(context.DeadlineExceeded)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, capabilitiesRequest(http.MethodGet, "/capabilities", tenant.String(),
		security.AuthInfo{UserID: "u"}, nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on policy query failure, got %d", w.Code)
	}
	if len(decodeCaps(t, w.Body.Bytes())) != 0 {
		t.Fatalf("expected empty map on failure, got %s", w.Body.String())
	}
}
