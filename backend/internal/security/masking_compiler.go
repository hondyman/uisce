package security

import (
	"fmt"
	"strings"
)

type MaskingPolicy struct {
	TargetField string `json:"target_field"`
	MaskType    string `json:"mask_type"` // PARTIAL_MASK, HASH, NULLIFY, TOKENIZE
	RoleExempt  string `json:"role_exempt"`
}

// ApplyDynamicMasking evaluates user entitlements and injects masking functions into the AST projection
func ApplyDynamicMasking(fieldExpr string, fieldName string, userRoles []string, policies []MaskingPolicy) string {
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
