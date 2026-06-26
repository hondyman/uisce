package handlers

import (
	"context"
	"fmt"
	"regexp"

	"github.com/jmoiron/sqlx"
)

// ============================================================================
// Validation Types
// ============================================================================

// ValidationResult represents the result of BO validation
type ValidationResult struct {
	Valid    bool              `json:"valid"`
	Errors   []ValidationError `json:"errors"`
	Warnings []ValidationError `json:"warnings"`
}

// ValidationError represents a single validation error or warning
type ValidationError struct {
	Code    string `json:"code"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationCode constants
const (
	CodeBOKeyRequired       = "BO_KEY_REQUIRED"
	CodeBOKeyFormat         = "BO_KEY_FORMAT"
	CodeBOKeyUnique         = "BO_KEY_UNIQUE"
	CodeBONameRequired      = "BO_NAME_REQUIRED"
	CodeDriverTableNotFound = "DRIVER_TABLE_NOT_FOUND"
	CodeTermNotFound        = "TERM_NOT_FOUND"
	CodeLinkedBONotFound    = "LINKED_BO_NOT_FOUND"
	CodeCircularReference   = "CIRCULAR_REFERENCE"
	CodeDuplicateTerm       = "DUPLICATE_TERM"
)

// boKeyRegex validates BO key format: lowercase letters, numbers, underscores, starts with letter
var boKeyRegex = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// ============================================================================
// Validation Functions
// ============================================================================

// ValidateBusinessObject performs comprehensive validation on a BO save request
func ValidateBusinessObject(ctx context.Context, db *sqlx.DB, req SaveWizardRequest, tenantID, datasourceID string) ValidationResult {
	result := ValidationResult{
		Valid:    true,
		Errors:   []ValidationError{},
		Warnings: []ValidationError{},
	}

	// 1. Required field validation
	if req.BOKey == "" {
		result.addError(CodeBOKeyRequired, "bo_key", "Business Object key is required")
	} else if !boKeyRegex.MatchString(req.BOKey) {
		result.addError(CodeBOKeyFormat, "bo_key", "Key must start with a letter and contain only lowercase letters, numbers, and underscores")
	}

	if req.Name == "" {
		result.addError(CodeBONameRequired, "name", "Business Object name is required")
	}

	// Early return if basic validation fails
	if !result.Valid {
		return result
	}

	// 2. BO key uniqueness check
	var existingCount int
	err := db.GetContext(ctx, &existingCount, `
		SELECT COUNT(*) FROM public.business_objects
		WHERE tenant_id = $1
		  AND tenant_datasource_id = $2
		  AND key = $3
	`, tenantID, datasourceID, req.BOKey)
	if err == nil && existingCount > 0 {
		result.addError(CodeBOKeyUnique, "bo_key", fmt.Sprintf("A Business Object with key '%s' already exists", req.BOKey))
	}

	// 3. Driver table existence check
	if req.DriverTableID != "" {
		var tableExists bool
		err := db.GetContext(ctx, &tableExists, `
			SELECT EXISTS(
				SELECT 1 FROM catalog_node 
				WHERE id = $1 
				  AND (tenant_id = $2 OR tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999')
				  AND tenant_datasource_id = $3
			)
		`, req.DriverTableID, tenantID, datasourceID)
		if err != nil || !tableExists {
			result.addError(CodeDriverTableNotFound, "driver_table_id", "Driver table not found")
		}
	}

	// 4. Validate selected terms exist
	if len(req.SelectedTerms) > 0 {
		// Check for duplicates
		termSet := make(map[string]bool)
		for _, termID := range req.SelectedTerms {
			if termSet[termID] {
				result.addWarning(CodeDuplicateTerm, "selected_terms", fmt.Sprintf("Duplicate term ID: %s", termID))
			}
			termSet[termID] = true
		}

		// Check terms exist
		query, args, _ := sqlx.In(`
			SELECT id FROM catalog_node WHERE id IN (?)
		`, req.SelectedTerms)
		query = db.Rebind(query)
		var foundTerms []string
		if err := db.SelectContext(ctx, &foundTerms, query, args...); err == nil {
			foundSet := make(map[string]bool)
			for _, id := range foundTerms {
				foundSet[id] = true
			}
			for _, termID := range req.SelectedTerms {
				if !foundSet[termID] {
					result.addError(CodeTermNotFound, "selected_terms", fmt.Sprintf("Term not found: %s", termID))
				}
			}
		}
	}

	// 5. Validate linked BOs exist and check for circular references
	if len(req.LinkedBOs) > 0 {
		linkedBOIDs := make([]string, len(req.LinkedBOs))
		for i, linked := range req.LinkedBOs {
			linkedBOIDs[i] = linked.BOID
		}

		query, args, _ := sqlx.In(`
			SELECT id FROM catalog_node cn
			INNER JOIN catalog_node_type cnt ON cn.node_type_id = cnt.id
			WHERE cnt.catalog_type_name = 'business_object'
			  AND cn.id IN (?)
		`, linkedBOIDs)
		query = db.Rebind(query)
		var foundBOs []string
		if err := db.SelectContext(ctx, &foundBOs, query, args...); err == nil {
			foundSet := make(map[string]bool)
			for _, id := range foundBOs {
				foundSet[id] = true
			}
			for _, linked := range req.LinkedBOs {
				if !foundSet[linked.BOID] {
					result.addError(CodeLinkedBONotFound, "linked_bos", fmt.Sprintf("Linked BO not found: %s", linked.BOID))
				}
			}
		}

		// TODO: Add circular reference detection for more complex graphs
	}

	return result
}

// addError adds an error and marks result as invalid
func (r *ValidationResult) addError(code, field, message string) {
	r.Valid = false
	r.Errors = append(r.Errors, ValidationError{
		Code:    code,
		Field:   field,
		Message: message,
	})
}

// addWarning adds a warning (does not affect validity)
func (r *ValidationResult) addWarning(code, field, message string) {
	r.Warnings = append(r.Warnings, ValidationError{
		Code:    code,
		Field:   field,
		Message: message,
	})
}
