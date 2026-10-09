package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	dbpkg "github.com/hondyman/uisce/backend/internal/db"
)

// Scope checks for the RBAC handlers. Every handler that takes an id from the
// request answers "may this tenant touch it" before it reads or writes anything,
// and writes its own statements with the tenant in the WHERE clause as well.
//
// The answers are deliberately uniform: a resource the tenant may not use is
// reported as not found, exactly like a resource that does not exist, so ids and
// user names cannot be probed across tenants.
var (
	// errNotInTenant: the resource does not exist for this tenant (or is not visible to it).
	errNotInTenant = errors.New("not found in this tenant")
	// errSharedReadOnly: a shared gold-copy role the tenant may read but not change.
	errSharedReadOnly = errors.New("gold-copy roles are read-only for tenants")
)

// Role access. A tenant reads its own roles and the shared gold-copy roles (the
// gold-copy tenant owns the template roles, and every tenant assigns them). It
// changes its own roles only.
func authorizeRole(ctx context.Context, db *sqlx.DB, tenantID, roleID string, write bool) error {
	tid, rid, err := parseScopeIDs(tenantID, roleID)
	if err != nil {
		return errNotInTenant
	}
	var owner struct {
		TenantID string `db:"tenant_id"`
		Shared   bool   `db:"shared"`
	}
	err = db.GetContext(ctx, &owner, `
		SELECT tenant_id::text AS tenant_id, (tenant_id = public.uisce_gold_copy_tenant_id()) AS shared
		FROM bp_roles
		WHERE id = $1
	`, rid)
	if errors.Is(err, sql.ErrNoRows) {
		return errNotInTenant
	}
	if err != nil {
		return fmt.Errorf("load role: %w", err)
	}
	ownerID, err := uuid.Parse(owner.TenantID)
	if err != nil {
		return errNotInTenant
	}
	switch {
	case ownerID == tid:
		return nil
	case owner.Shared:
		if write {
			return errSharedReadOnly
		}
		return nil
	default:
		return errNotInTenant
	}
}

// authorizeOwned: the row with this id belongs to the tenant. query must select the
// row's tenant_id as text, filtered by id; it is always a constant in this file.
func authorizeOwned(ctx context.Context, db *sqlx.DB, query, id, tenantID string) error {
	tid, rid, err := parseScopeIDs(tenantID, id)
	if err != nil {
		return errNotInTenant
	}
	var owner string
	err = db.GetContext(ctx, &owner, query, rid)
	if errors.Is(err, sql.ErrNoRows) {
		return errNotInTenant
	}
	if err != nil {
		return fmt.Errorf("load resource: %w", err)
	}
	ownerID, err := uuid.Parse(owner)
	if err != nil || ownerID != tid {
		return errNotInTenant
	}
	return nil
}

// authorizeVisibleNode: a catalog term the tenant may refer to in a grant. Its own
// terms, and the shared gold-copy terms every tenant reads. A shared term is usable,
// never changed here: the permission row is the tenant's own, the term is not.
func authorizeVisibleNode(ctx context.Context, db *sqlx.DB, nodeID, tenantID string) error {
	tid, rid, err := parseScopeIDs(tenantID, nodeID)
	if err != nil {
		return errNotInTenant
	}
	var owner struct {
		TenantID string `db:"tenant_id"`
		Shared   bool   `db:"shared"`
	}
	err = db.GetContext(ctx, &owner, `
		SELECT tenant_id::text AS tenant_id, (tenant_id = public.uisce_gold_copy_tenant_id()) AS shared
		FROM catalog_node
		WHERE id = $1
	`, rid)
	if errors.Is(err, sql.ErrNoRows) {
		return errNotInTenant
	}
	if err != nil {
		return fmt.Errorf("load term: %w", err)
	}
	ownerID, err := uuid.Parse(owner.TenantID)
	if err != nil {
		return errNotInTenant
	}
	if ownerID == tid || owner.Shared {
		return nil
	}
	return errNotInTenant
}

// authorizeTeam and authorizeDelegation: the team or delegation belongs to the tenant.
func authorizeTeam(ctx context.Context, db *sqlx.DB, teamID, tenantID string) error {
	return authorizeOwned(ctx, db, `SELECT tenant_id::text FROM bp_teams WHERE id = $1`, teamID, tenantID)
}

func authorizeDelegation(ctx context.Context, db *sqlx.DB, delegationID, tenantID string) error {
	return authorizeOwned(ctx, db, `SELECT tenant_id::text FROM bp_approval_delegations WHERE id = $1`, delegationID, tenantID)
}

// authorizeUser: the user belongs to the tenant, through the tenant's membership
// mapping (user_tenant) or as the user's home tenant (app_user.tenant_id). Users are
// identified by the text id the client sent, which is how both tables store them.
// A user who is not a member is not found for this tenant.
func authorizeUser(ctx context.Context, db *sqlx.DB, tenantID, userID string) error {
	tid, err := uuid.Parse(tenantID)
	if err != nil || userID == "" {
		return errNotInTenant
	}
	var member bool
	// Read inside a transaction that sets the tenant on the session. user_tenant and
	// app_user are row-level-security forced: with no tenant set, a restricted role sees
	// no rows, and every user would look like a non-member.
	err = dbpkg.WithTenantTransaction(ctx, db.DB, tenantID, func(tx *sql.Tx) error {
		return tx.QueryRowContext(ctx, `
			SELECT EXISTS (SELECT 1 FROM user_tenant WHERE user_id = $1 AND tenant_id = $2)
			    OR EXISTS (SELECT 1 FROM app_user WHERE id = $1 AND tenant_id = $2)
		`, userID, tid).Scan(&member)
	})
	if err != nil {
		return fmt.Errorf("check membership: %w", err)
	}
	if !member {
		return errNotInTenant
	}
	return nil
}

func parseScopeIDs(tenantID, id string) (uuid.UUID, uuid.UUID, error) {
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	rid, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	return tid, rid, nil
}

// writeScopeError answers a refused scope check: 404 for anything the tenant may not
// see (the same answer as a missing resource), 403 for a shared role it may only read,
// 500 when the check itself failed.
func writeScopeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errNotInTenant):
		http.Error(w, "Not found", http.StatusNotFound)
	case errors.Is(err, errSharedReadOnly):
		http.Error(w, errSharedReadOnly.Error(), http.StatusForbidden)
	default:
		http.Error(w, fmt.Sprintf("Failed to check access: %v", err), http.StatusInternalServerError)
	}
}
