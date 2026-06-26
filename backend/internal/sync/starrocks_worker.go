package sync

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

// StarRocksSyncWorker handles synchronization to StarRocks
type StarRocksSyncWorker struct {
	db *sql.DB
}

// NewStarRocksSyncWorker creates a new StarRocks sync worker
func NewStarRocksSyncWorker(starrocksURL, user, password string) (*StarRocksSyncWorker, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/", user, password, starrocksURL)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	return &StarRocksSyncWorker{
		db: db,
	}, nil
}

// SyncRole creates or updates a StarRocks role
func (w *StarRocksSyncWorker) SyncRole(ctx context.Context, roleData map[string]interface{}) error {
	roleName, _ := roleData["role_name"].(string)
	tenantID, _ := roleData["tenant_id"].(string)
	isGlobalAdmin, _ := roleData["is_global_admin"].(bool)

	srRoleName := fmt.Sprintf("tenant_%s_role_%s", tenantID, roleName)

	// Create role in StarRocks
	_, err := w.db.ExecContext(ctx, fmt.Sprintf("CREATE ROLE IF NOT EXISTS '%s'", srRoleName))
	if err != nil {
		return fmt.Errorf("failed to create StarRocks role: %w", err)
	}

	// Grant privileges
	if isGlobalAdmin {
		// Global admins get access to all databases
		_, err = w.db.ExecContext(ctx, fmt.Sprintf("GRANT ALL ON *.* TO ROLE '%s'", srRoleName))
	} else {
		// Tenant users get access to tenant-specific database
		tenantDB := fmt.Sprintf("tenant_%s", tenantID)
		_, err = w.db.ExecContext(ctx, fmt.Sprintf("GRANT SELECT ON %s.* TO ROLE '%s'", tenantDB, srRoleName))
	}

	if err != nil {
		return fmt.Errorf("failed to grant privileges: %w", err)
	}

	return nil
}

// AssignUserToRole assigns a user to a StarRocks role
func (w *StarRocksSyncWorker) AssignUserToRole(ctx context.Context, userID, roleID string) error {
	// Get role name
	var roleName, tenantID string
	// Note: This would need to query the PostgreSQL IAM database
	// For now, we'll skip the actual implementation

	srRoleName := fmt.Sprintf("tenant_%s_role_%s", tenantID, roleName)
	srUserName := fmt.Sprintf("user_%s", userID)

	// Grant role to user
	_, err := w.db.ExecContext(ctx, fmt.Sprintf("GRANT ROLE '%s' TO '%s'", srRoleName, srUserName))
	if err != nil {
		return fmt.Errorf("failed to assign role to user: %w", err)
	}

	return nil
}

// Close closes the StarRocks connection
func (w *StarRocksSyncWorker) Close() error {
	return w.db.Close()
}
