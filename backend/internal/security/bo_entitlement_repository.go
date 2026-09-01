package security

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type BOEntitlement struct {
	ResourceType    string  `db:"resource_type"`
	ResourceID      string  `db:"resource_id"`
	FieldName       string  `db:"field_name"`
	PermissionLevel string  `db:"permission_level"`
	MaskingPattern  *string `db:"masking_pattern"`
}

type BOEntitlementRepository struct {
	db *sqlx.DB
}

func NewBOEntitlementRepository(db *sqlx.DB) *BOEntitlementRepository {
	return &BOEntitlementRepository{db: db}
}

type EntitlementKey struct {
	ResourceID string
	FieldName  string
}

type EntitlementsResult struct {
	Entitlements    map[EntitlementKey]string // (resource_id, field_name) -> permission_level
	MaskingPatterns map[EntitlementKey]string // (resource_id, field_name) -> masking_pattern
	HiddenBOs       map[string]struct{}        // bo_ids explicitly denied
	MaskedFields    map[EntitlementKey]struct{} // fields that should be masked
}

func (r *BOEntitlementRepository) GetEntitlementsForUser(
	ctx context.Context,
	userID string,
	roles []string,
	tenantID string,
	tenantInstanceID string,
) (*EntitlementsResult, error) {
	if len(roles) == 0 {
		return &EntitlementsResult{
			Entitlements:    map[EntitlementKey]string{},
			MaskingPatterns: map[EntitlementKey]string{},
			HiddenBOs:       map[string]struct{}{},
			MaskedFields:    map[EntitlementKey]struct{}{},
		}, nil
	}

	instanceFilter := ""
	rolesArgPos := 4
	if tenantInstanceID != "" && tenantInstanceID != "default" {
		instanceFilter = "AND (fp.tenant_instance_id IS NULL OR fp.tenant_instance_id = $5)"
		rolesArgPos = 4
	}

	query := fmt.Sprintf(`
		SELECT
			fp.resource_id,
			fp.resource_type,
			fp.field_name,
			fp.permission_level,
			fp.masking_pattern
		FROM bp_field_permissions fp
		JOIN bp_user_roles ur ON ur.role_id = fp.role_id
		WHERE ur.user_id = $1
		  AND ur.tenant_id = $2
		  AND fp.resource_type = 'business_object'
		  AND fp.tenant_id = $3
		  AND ur.role_key = ANY($%d::text[])
		  AND ur.is_active = true
		  %s
	`, rolesArgPos, instanceFilter)

	var args []interface{}
	args = append(args, userID, tenantID, tenantID, pq.Array(roles))
	if tenantInstanceID != "" && tenantInstanceID != "default" {
		args = append(args, tenantInstanceID)
	}

	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get BO entitlements: %w", err)
	}
	defer rows.Close()

	result := &EntitlementsResult{
		Entitlements:    map[EntitlementKey]string{},
		MaskingPatterns: map[EntitlementKey]string{},
		HiddenBOs:       map[string]struct{}{},
		MaskedFields:    map[EntitlementKey]struct{}{},
	}

	for rows.Next() {
		var resType, resID, fieldName, permLevel string
		var maskingPattern *string

		if err := rows.Scan(&resID, &resType, &fieldName, &permLevel, &maskingPattern); err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			return nil, fmt.Errorf("failed to scan entitlement row: %w", err)
		}

		key := EntitlementKey{ResourceID: resID, FieldName: fieldName}

		if fieldName == "*" && permLevel == "none" {
			result.HiddenBOs[resID] = struct{}{}
		} else {
			existing, ok := result.Entitlements[key]
			if !ok || permLevel == "write" || (existing != "write" && permLevel == "read") ||
				(existing != "write" && existing != "read" && permLevel == "mask") {
				result.Entitlements[key] = permLevel
			}
			if maskingPattern != nil && *maskingPattern != "" {
				result.MaskingPatterns[key] = *maskingPattern
			}
		}
	}

	return result, rows.Err()
}

func (r *BOEntitlementRepository) GetEntitlementsForRoles(
	ctx context.Context,
	roleIDs []string,
	tenantID string,
	tenantInstanceID string,
) (*EntitlementsResult, error) {
	if len(roleIDs) == 0 {
		return &EntitlementsResult{
			Entitlements:    map[EntitlementKey]string{},
			MaskingPatterns: map[EntitlementKey]string{},
			HiddenBOs:       map[string]struct{}{},
			MaskedFields:    map[EntitlementKey]struct{}{},
		}, nil
	}

	instanceFilter := ""
	if tenantInstanceID != "" && tenantInstanceID != "default" {
		instanceFilter = "AND (fp.tenant_instance_id IS NULL OR fp.tenant_instance_id = $3)"
	}

	query := fmt.Sprintf(`
		SELECT
			fp.resource_id,
			fp.resource_type,
			fp.field_name,
			fp.permission_level,
			fp.masking_pattern
		FROM bp_field_permissions fp
		WHERE fp.role_id = ANY($1::uuid[])
		  AND fp.tenant_id = $2
		  AND fp.resource_type = 'business_object'
		  %s
	`, instanceFilter)

	var args []interface{}
	args = append(args, pq.Array(roleIDs), tenantID)
	if tenantInstanceID != "" {
		args = append(args, tenantInstanceID)
	}

	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get BO entitlements for roles: %w", err)
	}
	defer rows.Close()

	result := &EntitlementsResult{
		Entitlements:    map[EntitlementKey]string{},
		MaskingPatterns: map[EntitlementKey]string{},
		HiddenBOs:       map[string]struct{}{},
		MaskedFields:    map[EntitlementKey]struct{}{},
	}

	for rows.Next() {
		var resType, resID, fieldName, permLevel string
		var maskingPattern *string

		if err := rows.Scan(&resID, &resType, &fieldName, &permLevel, &maskingPattern); err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			return nil, fmt.Errorf("failed to scan entitlement row: %w", err)
		}

		key := EntitlementKey{ResourceID: resID, FieldName: fieldName}

		if fieldName == "*" && permLevel == "none" {
			result.HiddenBOs[resID] = struct{}{}
		} else {
			existing, ok := result.Entitlements[key]
			if !ok || permLevel == "write" || (existing != "write" && permLevel == "read") ||
				(existing != "write" && existing != "read" && permLevel == "mask") {
				result.Entitlements[key] = permLevel
			}
			if maskingPattern != nil && *maskingPattern != "" {
				result.MaskingPatterns[key] = *maskingPattern
			}
		}
	}

	return result, rows.Err()
}

func (r *BOEntitlementRepository) GetHiddenBOIDs(
	ctx context.Context,
	userID string,
	roles []string,
	tenantID string,
	tenantInstanceID string,
) (map[string]struct{}, error) {
	if len(roles) == 0 {
		return map[string]struct{}{}, nil
	}

	instanceFilter := ""
	args := []interface{}{userID, tenantID, tenantID, pq.Array(roles)}
	if tenantInstanceID != "" && tenantInstanceID != "default" {
		instanceFilter = "AND (fp.tenant_instance_id IS NULL OR fp.tenant_instance_id = $5)"
		args = append(args, tenantInstanceID)
	}

	query := fmt.Sprintf(`
		SELECT DISTINCT fp.resource_id
		FROM bp_field_permissions fp
		JOIN bp_user_roles ur ON ur.role_id = fp.role_id
		WHERE ur.user_id = $1
		  AND ur.tenant_id = $2
		  AND fp.resource_type = 'business_object'
		  AND fp.tenant_id = $3
		  AND fp.field_name = '*'
		  AND fp.permission_level = 'none'
		  AND ur.role_key = ANY($4::text[])
		  AND ur.is_active = true
		  %s
	`, instanceFilter)

	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get hidden BO IDs: %w", err)
	}
	defer rows.Close()

	hidden := make(map[string]struct{})
	for rows.Next() {
		var boID string
		if err := rows.Scan(&boID); err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			return nil, fmt.Errorf("failed to scan hidden BO ID: %w", err)
		}
		hidden[boID] = struct{}{}
	}

	return hidden, rows.Err()
}
