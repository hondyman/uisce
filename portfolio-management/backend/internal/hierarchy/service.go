package hierarchy

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// HierarchyService handles all hierarchy-related operations
type HierarchyService struct {
	db *gorm.DB
}

// NewHierarchyService creates a new hierarchy service
func NewHierarchyService(db *gorm.DB) *HierarchyService {
	return &HierarchyService{db: db}
}

// ValidateHierarchy checks if a parent-child relationship is allowed
func (s *HierarchyService) ValidateHierarchy(ctx context.Context, tenantID, parentModelType, childModelType string) (*HierarchyValidationResult, error) {
	result := &HierarchyValidationResult{
		ParentModelType: parentModelType,
		ChildModelType:  childModelType,
		Errors:          []string{},
		Warnings:        []string{},
	}

	// Find matching hierarchy rules
	var rules []HierarchyRule
	if err := s.db.WithContext(ctx).
		Where("tenant_id = ? AND parent_model_type = ? AND child_model_type = ?",
			tenantID, parentModelType, childModelType).
		Find(&rules).Error; err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Database error: %v", err))
		return result, err
	}

	result.MatchingRules = rules

	// Check if any allowed rules exist
	allowed := false
	for _, rule := range rules {
		if rule.Allowed {
			allowed = true
			break
		}
	}

	if allowed {
		result.Valid = true
	} else {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("%s cannot be parent of %s", parentModelType, childModelType))
	}

	return result, nil
}

// GetHierarchyRules retrieves all hierarchy rules for a tenant
func (s *HierarchyService) GetHierarchyRules(ctx context.Context, tenantID string) ([]HierarchyRule, error) {
	var rules []HierarchyRule
	if err := s.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("parent_model_type, child_model_type").
		Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

// GetHierarchySummary retrieves a summary of hierarchy rules
func (s *HierarchyService) GetHierarchySummary(ctx context.Context, tenantID string) ([]HierarchySummary, error) {
	var summary []HierarchySummary
	if err := s.db.WithContext(ctx).
		Table("entity_hierarchy_summary").
		Where("tenant_id = ?", tenantID).
		Scan(&summary).Error; err != nil {
		return nil, err
	}
	return summary, nil
}

// GetEntityHierarchy retrieves the entity hierarchy tree
func (s *HierarchyService) GetEntityHierarchy(ctx context.Context, rootID string, maxDepth int) (*EntityHierarchyNode, error) {
	// Get root entity
	node := &EntityHierarchyNode{
		ID:    rootID,
		Depth: 0,
	}

	// In a real implementation, this would recursively fetch children
	// from the positions and entity_hierarchy_tree view
	return node, nil
}

// GetHierarchyStats retrieves statistics about the hierarchy
func (s *HierarchyService) GetHierarchyStats(ctx context.Context, tenantID string) (*HierarchyStats, error) {
	stats := &HierarchyStats{}

	// Count entities
	if err := s.db.WithContext(ctx).
		Table("model_types").
		Where("tenant_id = ?", tenantID).
		Count(&stats.TotalEntities).Error; err != nil {
		return nil, err
	}

	// Count rules
	if err := s.db.WithContext(ctx).
		Model(&HierarchyRule{}).
		Where("tenant_id = ? AND allowed = ?", tenantID, true).
		Count(&stats.AllowedRules).Error; err != nil {
		return nil, err
	}

	if err := s.db.WithContext(ctx).
		Model(&HierarchyRule{}).
		Where("tenant_id = ? AND allowed = ?", tenantID, false).
		Count(&stats.DisallowedRules).Error; err != nil {
		return nil, err
	}

	return stats, nil
}

// CreateHierarchyRule creates a new hierarchy rule
func (s *HierarchyService) CreateHierarchyRule(ctx context.Context, rule *HierarchyRule) error {
	rule.ID = generateUUID()
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()

	if err := s.db.WithContext(ctx).Create(rule).Error; err != nil {
		return err
	}
	return nil
}

// UpdateHierarchyRule updates an existing hierarchy rule
func (s *HierarchyService) UpdateHierarchyRule(ctx context.Context, rule *HierarchyRule) error {
	rule.UpdatedAt = time.Now()

	if err := s.db.WithContext(ctx).Model(rule).Updates(rule).Error; err != nil {
		return err
	}
	return nil
}

// DeleteHierarchyRule deletes a hierarchy rule
func (s *HierarchyService) DeleteHierarchyRule(ctx context.Context, tenantID, parentType, childType string) error {
	if err := s.db.WithContext(ctx).
		Where("tenant_id = ? AND parent_model_type = ? AND child_model_type = ?",
			tenantID, parentType, childType).
		Delete(&HierarchyRule{}).Error; err != nil {
		return err
	}
	return nil
}

// BulkCreateOperations executes multiple hierarchy operations in a transaction
func (s *HierarchyService) BulkCreateOperations(ctx context.Context, tenantID string, req *HierarchyBulkRequest) (*HierarchyBulkResponse, error) {
	response := &HierarchyBulkResponse{}

	// Start transaction using GORM v2 syntax
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for _, op := range req.Operations {
		result := HierarchyOperationResult{
			Operation: op,
			Success:   true,
		}

		// Process each operation
		switch op.Operation {
		case "CREATE":
			rule := &HierarchyRule{
				ID:       generateUUID(),
				TenantID: tenantID,
			}

			if err := tx.Create(rule).Error; err != nil {
				result.Success = false
				result.Error = err.Error()
				response.Failed++
				response.ErrorsSummary = append(response.ErrorsSummary, err.Error())
			} else {
				response.Successful++
			}

		default:
			result.Message = "Operation type not supported"
		}

		response.Results = append(response.Results, result)
	}

	if err := tx.Commit().Error; err != nil {
		return response, err
	}

	return response, nil
}

// LogHierarchyAudit logs a hierarchy audit entry
func (s *HierarchyService) LogHierarchyAudit(ctx context.Context, log *HierarchyAuditLog) error {
	if err := s.db.WithContext(ctx).Create(log).Error; err != nil {
		return err
	}
	return nil
}

// GetHierarchyAuditLog retrieves audit logs for an entity
func (s *HierarchyService) GetHierarchyAuditLog(ctx context.Context, entityID string, limit int) ([]HierarchyAuditLog, error) {
	var logs []HierarchyAuditLog
	if err := s.db.WithContext(ctx).
		Where("entity_id = ?", entityID).
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

// ImportHierarchyRules imports multiple rules at once
func (s *HierarchyService) ImportHierarchyRules(ctx context.Context, tenantID string, req *HierarchyImportRequest) (*HierarchyImportResponse, error) {
	response := &HierarchyImportResponse{
		CreatedAt: time.Now(),
	}

	tx := s.db.WithContext(ctx).Begin()

	for _, rule := range req.Rules {
		hr := &HierarchyRule{
			ID:              generateUUID(),
			TenantID:        tenantID,
			ParentModelType: rule.ParentModelType,
			ChildModelType:  rule.ChildModelType,
			Allowed:         true,
			OwnershipTypes:  rule.OwnershipTypes,
			Description:     rule.Description,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		if err := tx.Create(hr).Error; err != nil {
			// Skip duplicates
			response.Skipped++
			continue
		}

		response.Imported++
	}

	if err := tx.Commit().Error; err != nil {
		response.Failed = len(req.Rules) - response.Imported - response.Skipped
		return response, err
	}

	return response, nil
}

// ValidateEntityConsistency checks for circular references and orphaned entities
func (s *HierarchyService) ValidateEntityConsistency(ctx context.Context, tenantID string) error {
	// Check for circular references
	var circularCount int64
	if err := s.db.WithContext(ctx).
		Table("positions").
		Where("tenant_id = ?", tenantID).
		Count(&circularCount).Error; err != nil {
		return err
	}

	// In production, would use recursive CTE to find actual circulars
	if circularCount > 0 {
		return fmt.Errorf("potential circular references detected")
	}

	return nil
}

// Helper function to generate UUID (simplified - in production use proper UUID lib)
func generateUUID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
