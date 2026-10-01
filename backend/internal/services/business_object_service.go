package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/jmoiron/sqlx"
)

// BusinessObjectService handles business object operations with real database queries
type BusinessObjectService struct {
	db    *sqlx.DB
	rules AccessRuleRepository
}

// GetInstanceForValidation retrieves a business object instance and formats it for validation
// It merges core and custom field values into a single flat JSON object
func (s *BusinessObjectService) GetInstanceForValidation(ctx context.Context, tenantID, instanceID string) (map[string]interface{}, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database connection not initialized")
	}

	boID, err := s.getInstanceBusinessObjectID(ctx, tenantID, instanceID)
	if err != nil {
		return nil, err
	}

	decision, err := s.requireAccess(ctx, tenantID, boID, AccessLevelRead)
	if err != nil {
		return nil, fmt.Errorf("access denied for business object: %w", err)
	}

	whereParts := []string{"tenant_id = $1", "id = $2"}
	if decision.RowPredicate != "" {
		whereParts = append(whereParts, "("+decision.RowPredicate+")")
	}

	query := "SELECT id, business_object_id, core_attributes, custom_attributes, created_by, created_at, last_modified_at " +
		"FROM business_object_instances WHERE " + strings.Join(whereParts, " AND ")

	inst := &models.BusinessObjectInstance{}
	var coreJSON, customJSON []byte

	err = s.db.QueryRowContext(ctx, query, tenantID, instanceID).
		Scan(&inst.ID, &inst.BusinessObjectID, &coreJSON, &customJSON,
			&inst.CreatedBy, &inst.CreatedAt, &inst.LastModifiedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("instance not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	// Create merged flat JSON object for validation
	result := make(map[string]interface{})

	// Add core fields
	if len(coreJSON) > 0 {
		var coreFields map[string]interface{}
		if err := json.Unmarshal(coreJSON, &coreFields); err == nil {
			for k, v := range coreFields {
				result[k] = v
			}
		}
	}

	// Add custom fields (will override core if same key exists)
	if len(customJSON) > 0 {
		var customFields map[string]interface{}
		if err := json.Unmarshal(customJSON, &customFields); err == nil {
			for k, v := range customFields {
				result[k] = v
			}
		}
	}

	// Add metadata fields that validation rules might need
	result["_instanceId"] = inst.ID
	result["_businessObjectId"] = inst.BusinessObjectID
	result["_createdAt"] = inst.CreatedAt
	result["_lastModifiedAt"] = inst.LastModifiedAt

	for term, mask := range decision.ColumnMasks {
		switch mask {
		case "HIDE":
			delete(result, term)
		case "MASK":
			if _, ok := result[term]; ok {
				result[term] = "[MASKED]"
			}
		}
	}

	return result, nil
}

// getInstanceBusinessObjectID fetches the BO ID for an instance with tenant scoping.
func (s *BusinessObjectService) getInstanceBusinessObjectID(ctx context.Context, tenantID, instanceID string) (string, error) {
	var boID string
	err := s.db.QueryRowContext(ctx, `SELECT business_object_id FROM business_object_instances WHERE tenant_id = $1 AND id = $2`, tenantID, instanceID).Scan(&boID)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("instance not found")
	}
	if err != nil {
		return "", fmt.Errorf("failed to resolve business object id: %w", err)
	}
	return boID, nil
}

// ============================================================================
// WORKDAY-STYLE CORE/CUSTOM COMPOSITION HELPERS
// ============================================================================
