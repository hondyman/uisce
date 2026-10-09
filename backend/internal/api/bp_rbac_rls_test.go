package api

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"

	dbpkg "github.com/hondyman/uisce/backend/internal/db"
)

// TestRBACMembership_RestrictedRole runs the membership check as a restricted database
// role, the way the application's login must run. It needs a database prepared by
// db/verify/rbac_rls_fixture.sql, and is skipped unless UISCE_RBAC_RLS_DSN is set.
//
// The mapped-member case is asserted as it behaves TODAY, a known gap: user_tenant is
// row-level-security forced with no policy, so a restricted role cannot see the mapping
// even with the tenant set. When a policy for user_tenant lands, this case flips to a
// member, and the assertion below must be changed with it.
func TestRBACMembership_RestrictedRole(t *testing.T) {
	dsn := os.Getenv("UISCE_RBAC_RLS_DSN")
	if dsn == "" {
		t.Skip("set UISCE_RBAC_RLS_DSN to a restricted-role connection on a db prepared by db/verify/rbac_rls_fixture.sql")
	}
	raw, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer raw.Close()
	db := sqlx.NewDb(raw, "postgres")
	ctx := context.Background()
	const tenant = "11111111-1111-4111-8111-111111111111"

	t.Run("home-tenant member is a member", func(t *testing.T) {
		require.NoError(t, authorizeUser(ctx, db, tenant, "home-member"))
	})

	t.Run("other tenant's user is not a member", func(t *testing.T) {
		require.ErrorIs(t, authorizeUser(ctx, db, tenant, "other-tenant"), errNotInTenant)
	})

	t.Run("unknown user is not a member", func(t *testing.T) {
		require.ErrorIs(t, authorizeUser(ctx, db, tenant, "nobody"), errNotInTenant)
	})

	// The picker and updateUserTenant bring an unassigned user in: no home tenant and no
	// mapping. Under the app_user policy a NULL tenant_id is never visible, so the unassigned
	// user cannot be seen at all. Known gap, asserted as it behaves today.
	t.Run("unassigned user: KNOWN GAP (app_user policy hides NULL tenant_id rows)", func(t *testing.T) {
		var unassigned bool
		err := dbpkg.WithTenantTransaction(ctx, db.DB, tenant, func(tx *sql.Tx) error {
			return tx.QueryRowContext(ctx, `
				SELECT EXISTS (SELECT 1 FROM app_user u WHERE u.id = $1 AND u.tenant_id IS NULL
				               AND NOT EXISTS (SELECT 1 FROM user_tenant ut WHERE ut.user_id = u.id))`, "unassigned").Scan(&unassigned)
		})
		require.NoError(t, err)
		require.False(t, unassigned,
			"if this now reports true, the app_user policy admits unassigned rows: flip this assertion to true")
	})

	t.Run("mapped member: KNOWN GAP (user_tenant has no policy for the restricted role)", func(t *testing.T) {
		require.ErrorIs(t, authorizeUser(ctx, db, tenant, "mapped-member"), errNotInTenant,
			"if this now passes, the user_tenant policy has landed: flip this assertion to NoError")
	})
}
