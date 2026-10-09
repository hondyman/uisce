package api_test

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	httpapi "github.com/hondyman/uisce/backend/internal/api"
	"github.com/hondyman/uisce/backend/internal/handlers"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

const (
	scopeTenant = "11111111-1111-4111-8111-111111111111"
	scopeOther  = "22222222-2222-4222-8222-222222222222"
	scopeGold   = "99e99e99-99e9-49e9-89e9-99e99e99e999"
	scopeRole   = "33333333-3333-4333-8333-333333333333"
	scopeUser   = "44444444-4444-4444-8444-444444444444"
)

// roleScopeResolver resolves the datasource this test uses to the caller's tenant, as a
// real resolver does for a datasource the tenant owns. Any other id is unknown.
type roleScopeResolver struct{}

const scopeDatasource = "99999999-9999-4999-8999-999999999999"

func (roleScopeResolver) Resolve(_ context.Context, id string) (*security.ResolvedDatasource, error) {
	if id != scopeDatasource {
		return nil, errors.New("unknown datasource in these tests")
	}
	return &security.ResolvedDatasource{TenantID: scopeTenant, DatasourceID: id, InstanceID: "instance", ProductID: "product", AllowedRegions: []string{"us-east-1"}}, nil
}

// roleScopeRouter mounts the real RBAC handlers over a sqlmock database.
func roleScopeRouter(t *testing.T) (sqlmock.Sqlmock, *chi.Mux) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	h := httpapi.NewRBACHandlers(sqlx.NewDb(db, "sqlmock"), handlers.SecurityContextDeps{Resolver: roleScopeResolver{}})
	r := chi.NewRouter()
	h.RegisterRoutes(r)
	return mock, r
}

// scopeRequest is a request from a signed-in user of tenant, stating the region as a real client does.
func scopeRequest(method, path, tenant, body string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("X-Region", "us-east-1")
	auth := security.AuthInfo{UserID: scopeUser, TenantIDs: []string{tenant}, Roles: []string{"user"}}
	return req.WithContext(security.WithAuthInfo(req.Context(), auth))
}

// expectOwner answers the authorization lookup for the role: its owning tenant and whether it is a shared gold-copy role.
func expectOwner(mock sqlmock.Sqlmock, owner string, shared bool) {
	mock.ExpectQuery(`FROM bp_roles\s+WHERE id = \$1`).WithArgs(scopeRole).
		WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "shared"}).AddRow(owner, shared))
}

var roleColumns = []string{
	"id", "tenant_id", "role_key", "role_name", "description", "role_type", "role_level", "is_active", "is_template",
	"parent_role_id", "security_profile_id", "tenant_instance_id", "created_by", "created_at", "updated_at",
}

func TestRBACRole_UpdateOwnRoleIsAllowed(t *testing.T) {
	mock, r := roleScopeRouter(t)
	expectOwner(mock, scopeTenant, false)
	mock.ExpectExec(`UPDATE bp_roles`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), scopeRole, scopeTenant).
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, scopeRequest(http.MethodPut, "/rbac/roles/"+scopeRole, scopeTenant, `{"role_name":"Auditor"}`))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

// A gold-copy role is shared: the tenant can see it but must not change it. No write is attempted.
func TestRBACRole_UpdateGoldCopyRoleIsReadOnly(t *testing.T) {
	mock, r := roleScopeRouter(t)
	expectOwner(mock, scopeGold, true)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, scopeRequest(http.MethodPut, "/rbac/roles/"+scopeRole, scopeTenant, `{"role_name":"Changed"}`))
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

// A role owned by another tenant is indistinguishable from a missing one.
func TestRBACRole_UpdateOtherTenantRoleIsNotFound(t *testing.T) {
	mock, r := roleScopeRouter(t)
	expectOwner(mock, scopeOther, false)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, scopeRequest(http.MethodPut, "/rbac/roles/"+scopeRole, scopeTenant, `{"role_name":"Hijacked"}`))
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRBACRole_UpdateMissingRoleIsNotFound(t *testing.T) {
	mock, r := roleScopeRouter(t)
	mock.ExpectQuery(`FROM bp_roles\s+WHERE id = \$1`).WithArgs(scopeRole).WillReturnError(errNoRowsForScope())

	w := httptest.NewRecorder()
	r.ServeHTTP(w, scopeRequest(http.MethodPut, "/rbac/roles/"+scopeRole, scopeTenant, `{"role_name":"x"}`))
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRBACRole_DeleteOtherTenantRoleIsNotFound(t *testing.T) {
	mock, r := roleScopeRouter(t)
	expectOwner(mock, scopeOther, false)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, scopeRequest(http.MethodDelete, "/rbac/roles/"+scopeRole, scopeTenant, ""))
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRBACRole_DeleteGoldCopyRoleIsForbidden(t *testing.T) {
	mock, r := roleScopeRouter(t)
	expectOwner(mock, scopeGold, true)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, scopeRequest(http.MethodDelete, "/rbac/roles/"+scopeRole, scopeTenant, ""))
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRBACRole_DeleteOwnRoleIsAllowed(t *testing.T) {
	mock, r := roleScopeRouter(t)
	expectOwner(mock, scopeTenant, false)
	mock.ExpectExec(`UPDATE bp_roles SET is_active = false WHERE id = \$1 AND tenant_id = \$2`).
		WithArgs(scopeRole, scopeTenant).WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, scopeRequest(http.MethodDelete, "/rbac/roles/"+scopeRole, scopeTenant, ""))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

// A shared gold-copy role stays readable to every tenant, as the role list shows it.
func TestRBACRole_GetGoldCopyRoleIsReadable(t *testing.T) {
	mock, r := roleScopeRouter(t)
	expectOwner(mock, scopeGold, true)
	now := time.Now()
	mock.ExpectQuery(`SELECT \* FROM bp_roles`).WithArgs(scopeRole, scopeTenant).
		WillReturnRows(sqlmock.NewRows(roleColumns).AddRow(
			scopeRole, scopeGold, "controller", "Controller", "", "system", "approver", true, true,
			nil, nil, nil, nil, now, now,
		))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, scopeRequest(http.MethodGet, "/rbac/roles/"+scopeRole, scopeTenant, ""))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

// Unassigning a role touches only this tenant's assignments.
func TestRBACRole_UnassignIsScopedToTenant(t *testing.T) {
	mock, r := roleScopeRouter(t)
	expectOwner(mock, scopeTenant, false)
	mock.ExpectExec(`UPDATE bp_user_roles\s+SET is_active = false\s+WHERE role_id = \$1 AND user_id = \$2 AND tenant_id = \$3`).
		WithArgs(scopeRole, scopeUser, scopeTenant).WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, scopeRequest(http.MethodDelete, "/rbac/roles/"+scopeRole+"/unassign/"+scopeUser, scopeTenant, ""))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

// Assigning a role another tenant owns is refused before any insert.
func TestRBACRole_AssignOtherTenantRoleIsNotFound(t *testing.T) {
	mock, r := roleScopeRouter(t)
	expectOwner(mock, scopeOther, false)

	w := httptest.NewRecorder()
	body := `{"user_id":"` + scopeUser + `"}`
	r.ServeHTTP(w, scopeRequest(http.MethodPost, "/rbac/roles/"+scopeRole+"/assign", scopeTenant, body))
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

// With no verified identity the handler refuses before it reads or writes anything.
func TestRBACRole_NoIdentityIsUnauthorized(t *testing.T) {
	mock, r := roleScopeRouter(t)

	req := httptest.NewRequest(http.MethodPut, "/rbac/roles/"+scopeRole, strings.NewReader(`{"role_name":"x"}`))
	req.Header.Set("X-Region", "us-east-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

// A malformed role id is not a role the tenant can see; no query runs.
func TestRBACRole_MalformedIDIsNotFound(t *testing.T) {
	mock, r := roleScopeRouter(t)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, scopeRequest(http.MethodPut, "/rbac/roles/not-a-uuid", scopeTenant, `{"role_name":"x"}`))
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

// errNoRowsForScope is what the authorization lookup returns for a role that does not exist.
func errNoRowsForScope() error { return sql.ErrNoRows }

// A caller of tenant A who names tenant B in the X-Tenant-ID header is refused at
// the security context, before any role is read or written.
func TestRBACRole_SpoofedTenantHeaderIsRefused(t *testing.T) {
	for _, target := range []string{scopeOther, scopeGold} {
		mock, r := roleScopeRouter(t)
		req := scopeRequest(http.MethodPut, "/rbac/roles/"+scopeRole, scopeTenant, `{"role_name":"Hijacked"}`)
		req.Header.Set("X-Tenant-ID", target)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusForbidden, w.Code, "header %s: %s", target, w.Body.String())
		require.Contains(t, w.Body.String(), "requested tenant does not match")
		require.NoError(t, mock.ExpectationsWereMet())
	}
}

// The same spoof through the tenant_id query parameter is refused the same way.
func TestRBACRole_SpoofedTenantQueryIsRefused(t *testing.T) {
	for _, target := range []string{scopeOther, scopeGold} {
		mock, r := roleScopeRouter(t)
		path := "/rbac/roles/" + scopeRole + "?tenant_id=" + target

		w := httptest.NewRecorder()
		r.ServeHTTP(w, scopeRequest(http.MethodDelete, path, scopeTenant, ""))
		require.Equal(t, http.StatusForbidden, w.Code, "query %s: %s", target, w.Body.String())
		require.Contains(t, w.Body.String(), "requested tenant does not match")
		require.NoError(t, mock.ExpectationsWereMet())
	}
}

// Naming the caller's own tenant is the normal case and still works.
func TestRBACRole_NamingOwnTenantIsAccepted(t *testing.T) {
	mock, r := roleScopeRouter(t)
	expectOwner(mock, scopeTenant, false)
	mock.ExpectExec(`UPDATE bp_roles`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), scopeRole, scopeTenant).
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := scopeRequest(http.MethodPut, "/rbac/roles/"+scopeRole, scopeTenant, `{"role_name":"Auditor"}`)
	req.Header.Set("X-Tenant-ID", scopeTenant)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

const (
	scopeOtherUser = "55555555-5555-4555-8555-555555555555"
	scopeTeam      = "66666666-6666-4666-8666-666666666666"
	scopeDelegate  = "77777777-7777-4777-8777-777777777777"
	scopeFieldPerm = "88888888-8888-4888-8888-888888888888"
)

// userMember answers the membership lookup for a user the tenant may act on.
// beginTenantTx expects the transaction a user-table read runs in: it sets the tenant on the session.
func beginTenantTx(mock sqlmock.Sqlmock, tenant string) {
	mock.ExpectBegin()
	mock.ExpectExec(`set_config\('uisce.current_tenant'`).WithArgs(tenant).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`set_config\('app.tenant_id'`).WithArgs(tenant).WillReturnResult(sqlmock.NewResult(0, 0))
}

// expectUserMember answers the membership lookup for a user the tenant may act on.
func expectUserMember(mock sqlmock.Sqlmock, user string, member bool) {
	beginTenantTx(mock, scopeTenant)
	mock.ExpectQuery(`FROM user_tenant WHERE user_id = \$1 AND tenant_id = \$2`).WithArgs(user, scopeTenant).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(member))
	mock.ExpectCommit()
}

// Assigning one of the tenant's roles to a user of another tenant is refused: the role
// check passes, the user check fails, and no assignment is written.
func TestRBACRole_AssignOwnRoleToOtherTenantsUserIsNotFound(t *testing.T) {
	mock, r := roleScopeRouter(t)
	expectOwner(mock, scopeTenant, false)
	expectUserMember(mock, scopeOtherUser, false)

	w := httptest.NewRecorder()
	body := `{"user_id":"` + scopeOtherUser + `"}`
	r.ServeHTTP(w, scopeRequest(http.MethodPost, "/rbac/roles/"+scopeRole+"/assign", scopeTenant, body))
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

// The same assignment to a member of the tenant is written.
func TestRBACRole_AssignOwnRoleToMemberIsAllowed(t *testing.T) {
	mock, r := roleScopeRouter(t)
	expectOwner(mock, scopeTenant, false)
	expectUserMember(mock, scopeUser, true)
	mock.ExpectExec(`INSERT INTO bp_user_roles`).WillReturnResult(sqlmock.NewResult(1, 1))

	w := httptest.NewRecorder()
	body := `{"user_id":"` + scopeUser + `"}`
	r.ServeHTTP(w, scopeRequest(http.MethodPost, "/rbac/roles/"+scopeRole+"/assign", scopeTenant, body))
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

// A global admin may act for an arbitrary tenant they name: that is the rule, and it works.
func TestRBACRole_GlobalAdminNamingAnyTenantSucceeds(t *testing.T) {
	mock, r := roleScopeRouter(t)
	expectOwner(mock, scopeOther, false)
	mock.ExpectExec(`UPDATE bp_roles`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), scopeRole, scopeOther).
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest(http.MethodPut, "/rbac/roles/"+scopeRole, strings.NewReader(`{"role_name":"Ops"}`))
	req.Header.Set("X-Region", "us-east-1")
	req.Header.Set("X-Tenant-ID", scopeOther)
	auth := security.AuthInfo{UserID: scopeUser, Roles: []string{"global_admin"}}
	req = req.WithContext(security.WithAuthInfo(req.Context(), auth))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

// The gold-copy tenant owns the template roles, so it may change them.
func TestRBACRole_GoldTenantMayEditItsOwnGoldCopyRole(t *testing.T) {
	mock, r := roleScopeRouter(t)
	expectOwner(mock, scopeGold, true)
	mock.ExpectExec(`UPDATE bp_roles`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), scopeRole, scopeGold).
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, scopeRequest(http.MethodPut, "/rbac/roles/"+scopeRole, scopeGold, `{"role_name":"Controller"}`))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

// If the scoped write finds nothing (the role went away between check and write), it is
// a 404, not a silent success.
func TestRBACRole_UpdateWithNoRowsAffectedIsNotFound(t *testing.T) {
	mock, r := roleScopeRouter(t)
	expectOwner(mock, scopeTenant, false)
	mock.ExpectExec(`UPDATE bp_roles`).WillReturnResult(sqlmock.NewResult(0, 0))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, scopeRequest(http.MethodPut, "/rbac/roles/"+scopeRole, scopeTenant, `{"role_name":"x"}`))
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRBACRole_DeleteWithNoRowsAffectedIsNotFound(t *testing.T) {
	mock, r := roleScopeRouter(t)
	expectOwner(mock, scopeTenant, false)
	mock.ExpectExec(`UPDATE bp_roles SET is_active = false`).WillReturnResult(sqlmock.NewResult(0, 0))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, scopeRequest(http.MethodDelete, "/rbac/roles/"+scopeRole, scopeTenant, ""))
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

// checkPermission answers only about the caller's own tenant: a body naming another is refused
// before anything is read, and a user of another tenant is not found, with no grant looked up.
func TestRBACPermission_CheckRefusesOtherTenantInBody(t *testing.T) {
	mock, r := roleScopeRouter(t)
	w := httptest.NewRecorder()
	body := `{"user_id":"` + scopeUser + `","tenant_id":"` + scopeOther + `","permission_key":"x"}`
	r.ServeHTTP(w, scopeRequest(http.MethodPost, "/rbac/permissions/check", scopeTenant, body))
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRBACPermission_CheckForOtherTenantsUserIsNotFound(t *testing.T) {
	mock, r := roleScopeRouter(t)
	expectUserMember(mock, scopeOtherUser, false)

	w := httptest.NewRecorder()
	body := `{"user_id":"` + scopeOtherUser + `","permission_key":"x"}`
	req := scopeRequest(http.MethodPost, "/rbac/permissions/check", scopeTenant, body)
	req.Header.Set("X-Datasource-Id", scopeDatasource)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

// A member of the tenant gets the grant answered for the caller's own tenant and datasource.
func TestRBACPermission_CheckOwnUserAnswers(t *testing.T) {
	mock, r := roleScopeRouter(t)
	expectUserMember(mock, scopeUser, true)
	mock.ExpectQuery(`SELECT bp_user_has_permission\(\$1, \$2, \$3, \$4\)`).
		WithArgs(scopeUser, scopeTenant, scopeDatasource, "reports.read").
		WillReturnRows(sqlmock.NewRows([]string{"has"}).AddRow(true))

	w := httptest.NewRecorder()
	body := `{"user_id":"` + scopeUser + `","permission_key":"reports.read"}`
	req := scopeRequest(http.MethodPost, "/rbac/permissions/check", scopeTenant, body)
	req.Header.Set("X-Datasource-Id", scopeDatasource)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), `"has_permission":true`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRBACDelegation_LogOnOtherTenantsDelegationIsNotFound(t *testing.T) {
	mock, r := roleScopeRouter(t)
	mock.ExpectQuery(`FROM bp_approval_delegations WHERE id = \$1`).WithArgs(scopeDelegate).
		WillReturnRows(sqlmock.NewRows([]string{"tenant_id"}).AddRow(scopeOther))

	w := httptest.NewRecorder()
	body := `{"delegate_user_id":"` + scopeUser + `","action_type":"approve"}`
	r.ServeHTTP(w, scopeRequest(http.MethodPost, "/rbac/delegations/"+scopeDelegate+"/log", scopeTenant, body))
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRBACDelegation_UpdateOtherTenantsDelegationIsNotFound(t *testing.T) {
	mock, r := roleScopeRouter(t)
	mock.ExpectQuery(`FROM bp_approval_delegations WHERE id = \$1`).WithArgs(scopeDelegate).
		WillReturnRows(sqlmock.NewRows([]string{"tenant_id"}).AddRow(scopeOther))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, scopeRequest(http.MethodPut, "/rbac/delegations/"+scopeDelegate, scopeTenant, `{"is_active":false}`))
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRBACTeam_AddMemberToOtherTenantsTeamIsNotFound(t *testing.T) {
	mock, r := roleScopeRouter(t)
	mock.ExpectQuery(`FROM bp_teams WHERE id = \$1`).WithArgs(scopeTeam).
		WillReturnRows(sqlmock.NewRows([]string{"tenant_id"}).AddRow(scopeOther))

	w := httptest.NewRecorder()
	body := `{"user_id":"` + scopeUser + `","role_in_team":"member"}`
	r.ServeHTTP(w, scopeRequest(http.MethodPost, "/rbac/teams/"+scopeTeam+"/members", scopeTenant, body))
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

// A field permission can't be granted on a shared gold-copy role.
func TestRBACFieldPermission_GrantOnGoldRoleIsForbidden(t *testing.T) {
	mock, r := roleScopeRouter(t)
	expectOwner(mock, scopeGold, true)

	w := httptest.NewRecorder()
	body := `{"role_id":"` + scopeRole + `","term_node_id":"` + scopeFieldPerm + `","permission_level":"read"}`
	r.ServeHTTP(w, scopeRequest(http.MethodPost, "/rbac/field-permissions", scopeTenant, body))
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

// adminRequest is a request from a user with the given roles, acting for tenant.
func adminRequest(method, path, tenant, body string, roles ...string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("X-Region", "us-east-1")
	req.Header.Set("X-Tenant-ID", tenant)
	auth := security.AuthInfo{UserID: scopeUser, TenantIDs: []string{tenant}, Roles: roles}
	return req.WithContext(security.WithAuthInfo(req.Context(), auth))
}

// A tenant admin cannot move a user who belongs to another tenant: not found, no write.
func TestRBACUserTenant_TenantAdminCannotMoveOtherTenantsUser(t *testing.T) {
	mock, r := roleScopeRouter(t)
	expectUserMember(mock, scopeOtherUser, false)
	beginTenantTx(mock, scopeTenant)
	mock.ExpectQuery(`FROM app_user u WHERE u.id = \$1 AND u.tenant_id IS NULL`).WithArgs(scopeOtherUser).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectCommit()

	w := httptest.NewRecorder()
	body := `{"tenant_id":"` + scopeTenant + `"}`
	r.ServeHTTP(w, adminRequest(http.MethodPut, "/rbac/users/"+scopeOtherUser+"/tenant", scopeTenant, body, "admin"))
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

// A tenant admin may bring in a user who has no home tenant yet, into their own tenant.
func TestRBACUserTenant_TenantAdminBringsInUnassignedUser(t *testing.T) {
	mock, r := roleScopeRouter(t)
	expectUserMember(mock, scopeOtherUser, false)
	beginTenantTx(mock, scopeTenant)
	mock.ExpectQuery(`FROM app_user u WHERE u.id = \$1 AND u.tenant_id IS NULL`).WithArgs(scopeOtherUser).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectCommit()
	mock.ExpectExec(`UPDATE users`).WithArgs(scopeTenant, scopeOtherUser, false, scopeTenant).
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	body := `{"tenant_id":"` + scopeTenant + `"}`
	r.ServeHTTP(w, adminRequest(http.MethodPut, "/rbac/users/"+scopeOtherUser+"/tenant", scopeTenant, body, "admin"))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

// A global admin may move any user to any tenant they name.
func TestRBACUserTenant_GlobalAdminMovesAnyUser(t *testing.T) {
	mock, r := roleScopeRouter(t)
	mock.ExpectExec(`UPDATE users`).WithArgs(scopeOther, scopeOtherUser, true, scopeTenant).
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	body := `{"tenant_id":"` + scopeOther + `"}`
	r.ServeHTTP(w, adminRequest(http.MethodPut, "/rbac/users/"+scopeOtherUser+"/tenant", scopeTenant, body, "global_admin"))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

// The admin list is for admins only: an ordinary member gets 403 and no query runs.
func TestRBACUsers_ListRefusesNonAdmin(t *testing.T) {
	mock, r := roleScopeRouter(t)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, adminRequest(http.MethodGet, "/rbac/users", scopeTenant, "", "user"))
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

// The admin list is scoped to the caller's tenant: the query is bound to it.
func TestRBACUsers_ListIsScopedToTenant(t *testing.T) {
	mock, r := roleScopeRouter(t)
	beginTenantTx(mock, scopeTenant)
	mock.ExpectQuery(`FROM users u\s+WHERE u.is_active = true\s+AND \(u.tenant_id = \$1::uuid\s+OR EXISTS \(SELECT 1 FROM user_tenant ut`).
		WithArgs(scopeTenant).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email", "name", "first_name", "last_name", "status", "is_active", "created_at", "tenant_id"}).
			AddRow(scopeUser, "ann", "ann@example.com", "Ann", nil, nil, "active", true, time.Now(), scopeTenant))
	mock.ExpectCommit()

	w := httptest.NewRecorder()
	r.ServeHTTP(w, adminRequest(http.MethodGet, "/rbac/users", scopeTenant, "", "admin"))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "ann@example.com")
	require.NoError(t, mock.ExpectationsWereMet())
}

// The picker refuses non-admins too.
func TestRBACUsers_AssignableRefusesNonAdmin(t *testing.T) {
	mock, r := roleScopeRouter(t)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, adminRequest(http.MethodGet, "/rbac/users/assignable", scopeTenant, "", "user"))
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

// The picker sends email for members only. An unassigned user is identified by username,
// even if the row came back with an email, because the mapping drops it on the way out.
func TestRBACUsers_AssignableSendsEmailForMembersOnly(t *testing.T) {
	mock, r := roleScopeRouter(t)
	beginTenantTx(mock, scopeTenant)
	mock.ExpectQuery(`FROM users u`).WithArgs(scopeTenant).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email", "name", "is_member"}).
			AddRow(scopeUser, "ann", "ann@example.com", "Ann", true).
			AddRow(scopeOtherUser, "cy", "cy@example.com", "Cy", false))
	mock.ExpectCommit()

	w := httptest.NewRecorder()
	r.ServeHTTP(w, adminRequest(http.MethodGet, "/rbac/users/assignable", scopeTenant, "", "admin"))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	body := w.Body.String()
	require.Contains(t, body, "ann@example.com")
	require.NotContains(t, body, "cy@example.com")
	require.Contains(t, body, `"is_member":false`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// The identity gate runs before the body is read: an anonymous caller with a body that
// does not parse still gets 401, not a description of the body. Ordering is structural.
func TestRBACRoutes_IdentityGateRunsBeforeBody(t *testing.T) {
	mock, r := roleScopeRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/rbac/users", strings.NewReader("not json"))
	req.Header.Set("X-Region", "us-east-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

// A field permission may not point at a term the tenant cannot see: refused before the insert.
func TestRBACFieldPermission_TermOfOtherTenantIsNotFound(t *testing.T) {
	mock, r := roleScopeRouter(t)
	expectOwner(mock, scopeTenant, false)
	mock.ExpectQuery(`FROM catalog_node\s+WHERE id = \$1`).WithArgs(scopeFieldPerm).
		WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "shared"}).AddRow(scopeOther, false))

	w := httptest.NewRecorder()
	body := `{"role_id":"` + scopeRole + `","term_node_id":"` + scopeFieldPerm + `","permission_level":"read"}`
	r.ServeHTTP(w, scopeRequest(http.MethodPost, "/rbac/field-permissions", scopeTenant, body))
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

// A shared gold-copy term is usable in a grant by any tenant.
func TestRBACFieldPermission_SharedTermIsUsable(t *testing.T) {
	mock, r := roleScopeRouter(t)
	expectOwner(mock, scopeTenant, false)
	mock.ExpectQuery(`FROM catalog_node\s+WHERE id = \$1`).WithArgs(scopeFieldPerm).
		WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "shared"}).AddRow(scopeGold, true))
	mock.ExpectQuery(`INSERT INTO bp_field_permissions`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(scopeFieldPerm))

	w := httptest.NewRecorder()
	body := `{"role_id":"` + scopeRole + `","term_node_id":"` + scopeFieldPerm + `","permission_level":"read"}`
	r.ServeHTTP(w, scopeRequest(http.MethodPost, "/rbac/field-permissions", scopeTenant, body))
	require.Less(t, w.Code, 300, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}
