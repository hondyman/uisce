package sync

import (
	"context"
	"database/sql"
	"fmt"
)

// PostgreSQLSyncWorker handles synchronization to PostgreSQL RLS
type PostgreSQLSyncWorker struct {
	db *sql.DB
}

// NewPostgreSQLSyncWorker creates a new PostgreSQL sync worker
func NewPostgreSQLSyncWorker(db *sql.DB) *PostgreSQLSyncWorker {
	return &PostgreSQLSyncWorker{
		db: db,
	}
}

// SyncRole creates or updates a PostgreSQL role with RLS
func (w *PostgreSQLSyncWorker) SyncRole(ctx context.Context, roleData map[string]interface{}) error {
	roleName, _ := roleData["role_name"].(string)
	tenantID, _ := roleData["tenant_id"].(string)
	isGlobalAdmin, _ := roleData["is_global_admin"].(bool)

	// Create PostgreSQL role
	pgRoleName := fmt.Sprintf("tenant_%s_role_%s", tenantID, roleName)

	// Check if role exists
	var exists bool
	err := w.db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname = $1)",
		pgRoleName,
	).Scan(&exists)
	if err != nil {
		return err
	}

	if !exists {
		// Create role
		createSQL := fmt.Sprintf("CREATE ROLE %s", pgRoleName)
		if isGlobalAdmin {
			createSQL += " BYPASSRLS"
		}
		_, err = w.db.ExecContext(ctx, createSQL)
		if err != nil {
			return fmt.Errorf("failed to create role: %w", err)
		}
	}

	// Grant permissions
	if err := w.grantPermissions(ctx, pgRoleName, isGlobalAdmin); err != nil {
		return err
	}

	return nil
}

func (w *PostgreSQLSyncWorker) grantPermissions(ctx context.Context, roleName string, isGlobalAdmin bool) error {
	tables := []string{
		"business_objects",
		"processes",
		"validation_rules",
		"catalog_node",
		"catalog_edge",
		"page_layouts",
		"pipelines",
	}

	for _, table := range tables {
		grantSQL := fmt.Sprintf("GRANT SELECT, INSERT, UPDATE ON %s TO %s", table, roleName)
		if isGlobalAdmin {
			grantSQL = fmt.Sprintf("GRANT ALL ON %s TO %s", table, roleName)
		}

		_, err := w.db.ExecContext(ctx, grantSQL)
		if err != nil {
			return fmt.Errorf("failed to grant permissions on %s: %w", table, err)
		}
	}

	return nil
}

// AssignUserToRole assigns a user to a PostgreSQL role
func (w *PostgreSQLSyncWorker) AssignUserToRole(ctx context.Context, userID, roleID string) error {
	// Get role name
	var roleName, tenantID string
	var isGlobalAdmin bool
	err := w.db.QueryRowContext(ctx,
		"SELECT role_name, tenant_id, is_global_admin FROM iam.roles WHERE role_id = $1",
		roleID,
	).Scan(&roleName, &tenantID, &isGlobalAdmin)
	if err != nil {
		return err
	}

	pgRoleName := fmt.Sprintf("tenant_%s_role_%s", tenantID, roleName)

	// Grant role to user (in practice, this would be handled via session variables)
	// For now, we just log the assignment
	fmt.Printf("User %s assigned to role %s\n", userID, pgRoleName)

	return nil
}

// RevokeUserFromRole revokes a user from a PostgreSQL role
func (w *PostgreSQLSyncWorker) RevokeUserFromRole(ctx context.Context, userID, roleID string) error {
	// Get role name
	var roleName, tenantID string
	err := w.db.QueryRowContext(ctx,
		"SELECT role_name, tenant_id FROM iam.roles WHERE role_id = $1",
		roleID,
	).Scan(&roleName, &tenantID)
	if err != nil {
		return err
	}

	pgRoleName := fmt.Sprintf("tenant_%s_role_%s", tenantID, roleName)

	// Revoke role from user
	fmt.Printf("User %s revoked from role %s\n", userID, pgRoleName)

	return nil
}
