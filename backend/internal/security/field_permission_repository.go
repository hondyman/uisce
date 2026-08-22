package security

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// FieldPermission represents a semantic-level field permission
// Canonical store for all field-level permissions using term_node_id
type FieldPermission struct {
	ID               string  `db:"id"`
	TenantID         string  `db:"tenant_id"`
	DatasourceID     string  `db:"datasource_id"`
	RoleID           string  `db:"role_id"`
	TermNodeID       *string `db:"term_node_id"`       // FK to catalog_node (semantic term)
	ResourceType     *string `db:"resource_type"`      // Optional: 'process' | 'step' | etc
	ResourceID       *string `db:"resource_id"`       // Optional: specific resource instance
	PermissionLevel  string  `db:"permission_level"`   // 'none' | 'read' | 'write' | 'mask'
	MaskingPattern  *string `db:"masking_pattern"`   // For 'mask' permission: e.g., 'XXX-XX-####' for SSN
}

// FieldPermissionRepository handles database operations for field permissions.
type FieldPermissionRepository struct {
	db *sqlx.DB
}

// NewFieldPermissionRepository creates a new repository.
func NewFieldPermissionRepository(db *sqlx.DB) *FieldPermissionRepository {
	return &FieldPermissionRepository{db: db}
}

// GetTermPermissionsForUser retrieves all term-level permissions for a user via their roles.
func (r *FieldPermissionRepository) GetTermPermissionsForUser(ctx context.Context, userID, tenantID, datasourceID string) ([]FieldPermission, error) {
	query := `
		SELECT DISTINCT ON (fp.term_node_id, fp.resource_type, fp.resource_id)
			fp.id, fp.tenant_id, fp.datasource_id, fp.role_id,
			fp.term_node_id, fp.resource_type, fp.resource_id,
			fp.permission_level, fp.masking_pattern
		FROM bp_user_roles ur
		JOIN bp_field_permissions fp ON ur.role_id = fp.role_id
		WHERE ur.user_id = $1
		  AND ur.tenant_id = $2
		  AND ur.datasource_id = $3
		  AND ur.is_active = true
		  AND fp.term_node_id IS NOT NULL
		ORDER BY fp.term_node_id, fp.resource_type, fp.resource_id,
			CASE fp.permission_level
				WHEN 'write' THEN 1
				WHEN 'read' THEN 2
				WHEN 'mask' THEN 3
				WHEN 'none' THEN 4
			END
	`

	rows, err := r.db.QueryxContext(ctx, query, userID, tenantID, datasourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get term permissions for user: %w", err)
	}
	defer rows.Close()

	var permissions []FieldPermission
	for rows.Next() {
		var fp FieldPermission
		if err := rows.Scan(
			&fp.ID, &fp.TenantID, &fp.DatasourceID, &fp.RoleID,
			&fp.TermNodeID, &fp.ResourceType, &fp.ResourceID,
			&fp.PermissionLevel, &fp.MaskingPattern,
		); err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			return nil, fmt.Errorf("failed to scan field permission: %w", err)
		}
		permissions = append(permissions, fp)
	}

	return permissions, rows.Err()
}

// GetTermPermissionsForRoles retrieves all term-level permissions for given role IDs.
func (r *FieldPermissionRepository) GetTermPermissionsForRoles(ctx context.Context, roleIDs []string, tenantID, datasourceID string) ([]FieldPermission, error) {
	if len(roleIDs) == 0 {
		return nil, nil
	}

	query := `
		SELECT DISTINCT ON (fp.term_node_id, fp.resource_type, fp.resource_id)
			fp.id, fp.tenant_id, fp.datasource_id, fp.role_id,
			fp.term_node_id, fp.resource_type, fp.resource_id,
			fp.permission_level, fp.masking_pattern
		FROM bp_field_permissions fp
		WHERE fp.role_id = ANY($1)
		  AND fp.tenant_id = $2
		  AND fp.datasource_id = $3
		  AND fp.term_node_id IS NOT NULL
		ORDER BY fp.term_node_id, fp.resource_type, fp.resource_id,
			CASE fp.permission_level
				WHEN 'write' THEN 1
				WHEN 'read' THEN 2
				WHEN 'mask' THEN 3
				WHEN 'none' THEN 4
			END
	`

	rows, err := r.db.QueryxContext(ctx, query, roleIDs, tenantID, datasourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get term permissions for roles: %w", err)
	}
	defer rows.Close()

	var permissions []FieldPermission
	for rows.Next() {
		var fp FieldPermission
		if err := rows.Scan(
			&fp.ID, &fp.TenantID, &fp.DatasourceID, &fp.RoleID,
			&fp.TermNodeID, &fp.ResourceType, &fp.ResourceID,
			&fp.PermissionLevel, &fp.MaskingPattern,
		); err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			return nil, fmt.Errorf("failed to scan field permission: %w", err)
		}
		permissions = append(permissions, fp)
	}

	return permissions, rows.Err()
}

// CheckTermAccess checks if a user has access to a specific term_node_id.
func (r *FieldPermissionRepository) CheckTermAccess(ctx context.Context, userID, termNodeID, tenantID, datasourceID string) (string, error) {
	// Returns the permission level ('none', 'read', 'write', 'mask') or empty string if no permission found
	query := `
		SELECT COALESCE(fp.permission_level, 'none') as permission_level
		FROM bp_user_roles ur
		JOIN bp_field_permissions fp ON ur.role_id = fp.role_id
		WHERE ur.user_id = $1
		  AND fp.term_node_id = $2
		  AND ur.tenant_id = $3
		  AND ur.datasource_id = $4
		  AND ur.is_active = true
		ORDER BY
			CASE fp.permission_level
				WHEN 'write' THEN 1
				WHEN 'read' THEN 2
				WHEN 'mask' THEN 3
				WHEN 'none' THEN 4
			END
		LIMIT 1
	`

	var permissionLevel string
	err := r.db.QueryRowxContext(ctx, query, userID, termNodeID, tenantID, datasourceID).Scan(&permissionLevel)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil // No permission found
		}
		return "", fmt.Errorf("failed to check term access: %w", err)
	}

	return permissionLevel, nil
}
