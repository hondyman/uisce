package security

import (
	"fmt"
	"strings"
)

// DEPRECATED: MaskingPolicy is deprecated.
// Use bp_field_permissions with permission_level='mask' and masking_pattern instead.
// This struct and ApplyDynamicMasking function will be removed in v3.0.
// New code should use security.FieldPermissionRepository to get term permissions
// and apply masking based on bp_field_permissions.masking_pattern.
type MaskingPolicy struct {
	TargetField string `json:"target_field"`
	MaskType    string `json:"mask_type"` // PARTIAL_MASK, HASH, NULLIFY, TOKENIZE
	RoleExempt  string `json:"role_exempt"`
}

// DEPRECATED: Use bp_field_permissions permission_level='mask' with masking_pattern instead.
// See ApplyMaskingFromPermissions for the new implementation.
func ApplyDynamicMasking(fieldExpr string, fieldName string, userRoles []string, policies []MaskingPolicy) string {
	// Log deprecation warning in production
	// TODO: Add structured logging when logging is refactored

	for _, policy := range policies {
		if strings.EqualFold(policy.TargetField, fieldName) {
			// Check if user holds an exempt role (e.g. 'PII_VIEWER', 'ADMIN')
			if containsMaskingRole(userRoles, policy.RoleExempt) {
				return fieldExpr
			}

			// Apply native SQL transformation based on policy type
			switch policy.MaskType {
			case "PARTIAL_MASK":
				// Credit cards or account numbers: show last 4 digits
				return fmt.Sprintf("REGEXP_REPLACE(CAST(%s AS VARCHAR), '^.*(.{4})$', '****-****-****-\\1') AS %s", fieldExpr, fieldName)
			case "NULLIFY":
				return fmt.Sprintf("NULL AS %s", fieldName)
			case "HASH":
				return fmt.Sprintf("MD5(CAST(%s AS VARCHAR)) AS %s", fieldExpr, fieldName)
			case "TOKENIZE":
				return fmt.Sprintf("CONCAT('TOK_', SHA256(CAST(%s AS VARCHAR))) AS %s", fieldExpr, fieldName)
			default:
				return fmt.Sprintf("'[RESTRICTED]' AS %s", fieldName)
			}
		}
	}
	return fieldExpr
}

// DEPRECATED: Use ContainsRole from helpers.go instead
func containsMaskingRole(roles []string, target string) bool {
	if target == "" {
		return false
	}
	for _, r := range roles {
		if strings.EqualFold(r, target) {
			return true
		}
	}
	return false
}

// ApplyMaskingFromPermissions applies masking based on bp_field_permissions data.
// This is the new implementation that replaces ApplyDynamicMasking.
func ApplyMaskingFromPermissions(fieldExpr string, fieldName string, termNodeID string, userRoles []string, permissions []FieldPermission) string {
	for _, perm := range permissions {
		if perm.TermNodeID != nil && *perm.TermNodeID == termNodeID && perm.PermissionLevel == "mask" {
			// Check if user has an exempt role (would have 'read' permission instead of 'mask')
			// For now, apply masking pattern if available
			if perm.MaskingPattern != nil && *perm.MaskingPattern != "" {
				return applyMaskingPattern(fieldExpr, fieldName, *perm.MaskingPattern)
			}
			// Default to partial mask if no pattern
			return fmt.Sprintf("REGEXP_REPLACE(CAST(%s AS VARCHAR), '^.*(.{4})$', '****-****-****-\\1') AS %s", fieldExpr, fieldName)
		}
	}
	// No masking permission found, return unchanged
	return fieldExpr
}

// applyMaskingPattern applies a masking pattern like 'XXX-XX-####' for SSN
func applyMaskingPattern(fieldExpr string, fieldName string, pattern string) string {
	// Pattern: X = masked, # = visible from end
	// For now, use simple regex replacement
	// This is a simplified implementation
	visibleCount := 0
	for _, char := range pattern {
		if char == '#' {
			visibleCount++
		}
	}
	if visibleCount == 0 {
		masked := strings.ReplaceAll(pattern, "X", "X")
		return fmt.Sprintf("'%s' AS %s", masked, fieldName)
	}
	// Show last N characters based on # count
	return fmt.Sprintf("REGEXP_REPLACE(CAST(%s AS VARCHAR), '^.*(.{%d})$', '%s') AS %s",
		fieldExpr, visibleCount, pattern, fieldName)
}
