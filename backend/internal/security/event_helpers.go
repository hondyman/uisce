package security

import (
	"os"
	"time"

	"github.com/hondyman/semlayer/backend/internal/events"
	"github.com/hondyman/semlayer/backend/internal/models"
)

// Helper functions for event publishing

func ruleToMap(rule *models.AccessRule) map[string]interface{} {
	m := map[string]interface{}{
		"rule_id":            rule.RuleID,
		"tenant_id":          rule.TenantID,
		"business_object_id": rule.BusinessObjectID,
		"group_dn":           rule.GroupDn,
		"access_level":       rule.AccessLevel,
		"status":             rule.Status,
		"row_filter_dsl":     rule.RowFilterDsl,
		"created_by":         rule.CreatedBy,
		"created_at":         rule.CreatedAt,
	}

	if rule.AppliesToApis != nil {
		m["applies_to_apis"] = *rule.AppliesToApis
	}
	if rule.AppliesToBi != nil {
		m["applies_to_bi"] = *rule.AppliesToBi
	}
	if rule.AppliesToAi != nil {
		m["applies_to_ai"] = *rule.AppliesToAi
	}
	if rule.UpdatedBy != "" {
		m["updated_by"] = rule.UpdatedBy
	}
	if !rule.UpdatedAt.IsZero() {
		m["updated_at"] = rule.UpdatedAt
	}

	if len(rule.ColumnMasks) > 0 {
		masks := make([]map[string]string, len(rule.ColumnMasks))
		for i, mask := range rule.ColumnMasks {
			masks[i] = map[string]string{
				"semantic_term_id": mask.SemanticTermID,
				"mask_type":        mask.MaskType,
			}
		}
		m["column_masks"] = masks
	}

	return m
}

func buildSnapshotEvent(rule *models.AccessRule) events.SecuritySnapshotEvent {
	event := events.SecuritySnapshotEvent{
		TenantID:         rule.TenantID,
		RuleID:           rule.RuleID,
		BusinessObjectID: rule.BusinessObjectID,
		GroupDN:          rule.GroupDn,
		AccessLevel:      rule.AccessLevel,
		Status:           rule.Status,
		RowFilterDsl:     rule.RowFilterDsl,
		CreatedBy:        rule.CreatedBy,
		CreatedAt:        rule.CreatedAt,
		SnapshotTime:     time.Now(),
		Version:          1, // TODO: track actual version
	}

	if rule.AppliesToApis != nil {
		event.AppliesToApis = *rule.AppliesToApis
	}
	if rule.AppliesToBi != nil {
		event.AppliesToBi = *rule.AppliesToBi
	}
	if rule.AppliesToAi != nil {
		event.AppliesToAi = *rule.AppliesToAi
	}
	if rule.UpdatedBy != "" {
		event.UpdatedBy = rule.UpdatedBy
	}
	if !rule.UpdatedAt.IsZero() {
		event.UpdatedAt = rule.UpdatedAt
	}

	// Convert column masks
	for _, mask := range rule.ColumnMasks {
		event.ColumnMasks = append(event.ColumnMasks, events.ColumnMaskSnapshot{
			SemanticTermID: mask.SemanticTermID,
			MaskType:       mask.MaskType,
		})
	}

	return event
}

func getEnvironment() string {
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		return "dev"
	}
	return env
}
