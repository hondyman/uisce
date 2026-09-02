package security

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// UserProvisioner just-in-time creates an app_user row for an identity that
// authenticated successfully (JWT signature + issuer/tenant trust already
// verified upstream) but has never been seen before. Access itself is still
// governed entirely by bp_user_roles / bp_field_permissions and
// idp_group_role_mappings — provisioning a bare user row grants nothing on
// its own, it only ensures the row FK'd by those tables exists so a new
// hire doesn't need a manual "create the user" step before role assignment.
type UserProvisioner struct {
	db *sqlx.DB
}

// NewUserProvisioner creates a provisioner backed by app_user.
func NewUserProvisioner(db *sqlx.DB) *UserProvisioner {
	return &UserProvisioner{db: db}
}

// EnsureUser inserts an app_user row for userID if one doesn't already
// exist. tenantID is stored as the user's default/home tenant for display
// purposes only (e.g. the legacy public.users view) — it has no bearing on
// authorization, which is resolved fresh per request from the token's
// tenant_ids/groups claims. Safe to call on every request: a no-op once the
// row exists.
func (p *UserProvisioner) EnsureUser(ctx context.Context, userID, email, name, tenantID string) error {
	if p == nil || p.db == nil || userID == "" {
		return nil
	}

	// username has no natural source from a JWT and is nullable in the
	// schema, but the admin "assign a role to a user" picker (listUsers)
	// scans it into a non-nullable Go string and silently drops any row
	// that fails to scan — so a NULL username would make a freshly
	// provisioned user invisible to admins trying to grant them access.
	// email is already unique and stable; falling back to it (or the user
	// id) keeps every provisioned user selectable.
	username := email
	if username == "" {
		username = userID
	}

	if _, err := p.db.ExecContext(ctx, `
		INSERT INTO app_user (id, email, name, username, tenant_id, is_active, status)
		VALUES ($1, NULLIF($2, ''), NULLIF($3, ''), $4, NULLIF($5, ''), true, 'active')
		ON CONFLICT (id) DO NOTHING
	`, userID, email, name, username, tenantID); err != nil {
		return fmt.Errorf("provisioning user %q: %w", userID, err)
	}
	return nil
}
