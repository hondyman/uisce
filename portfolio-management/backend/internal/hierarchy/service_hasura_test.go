package hierarchy

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// mockHasuraClient implements HasuraClient for testing
type mockHasuraClient struct {
	queryFunc  func(query string, variables map[string]interface{}) (map[string]interface{}, error)
	mutateFunc func(mutation string, variables map[string]interface{}) (map[string]interface{}, error)
}

func (m *mockHasuraClient) Query(query string, variables map[string]interface{}) (map[string]interface{}, error) {
	if m.queryFunc != nil {
		return m.queryFunc(query, variables)
	}
	return nil, nil
}

func (m *mockHasuraClient) Mutate(mutation string, variables map[string]interface{}) (map[string]interface{}, error) {
	if m.mutateFunc != nil {
		return m.mutateFunc(mutation, variables)
	}
	return nil, nil
}

func TestValidateHierarchyWithHasura(t *testing.T) {
	tenantID := uuid.New().String()
	ruleID := uuid.New().String()

	mockClient := &mockHasuraClient{
		queryFunc: func(query string, variables map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{
				"entity_hierarchy_rules": []interface{}{
					map[string]interface{}{
						"id":                ruleID,
						"tenant_id":         tenantID,
						"parent_model_type": "PORTFOLIO",
						"child_model_type":  "ACCOUNT",
						"allowed":           true,
						"ownership_types":   `["PERCENT_BASED"]`,
						"max_children":      float64(100),
						"description":       "Portfolio can own accounts",
						"notes":             "Test rule",
						"created_at":        "2025-01-01T00:00:00Z",
						"updated_at":        "2025-01-01T00:00:00Z",
					},
				},
			}, nil
		},
	}

	service := NewHierarchyServiceWithHasura(nil, mockClient)
	ctx := context.Background()

	result, err := service.ValidateHierarchy(ctx, tenantID, "PORTFOLIO", "ACCOUNT")
	if err != nil {
		t.Fatalf("ValidateHierarchy failed: %v", err)
	}

	if !result.Valid {
		t.Error("Expected validation to succeed")
	}

	if len(result.MatchingRules) != 1 {
		t.Errorf("Expected 1 matching rule, got %d", len(result.MatchingRules))
	}

	if result.MatchingRules[0].ParentModelType != "PORTFOLIO" {
		t.Errorf("Expected parent type PORTFOLIO, got %s", result.MatchingRules[0].ParentModelType)
	}
}

func TestGetHierarchyRulesWithHasura(t *testing.T) {
	tenantID := uuid.New().String()

	mockClient := &mockHasuraClient{
		queryFunc: func(query string, variables map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{
				"entity_hierarchy_rules": []interface{}{
					map[string]interface{}{
						"id":                uuid.New().String(),
						"tenant_id":         tenantID,
						"parent_model_type": "PORTFOLIO",
						"child_model_type":  "ACCOUNT",
						"allowed":           true,
						"ownership_types":   `["PERCENT_BASED"]`,
						"description":       "Portfolio owns accounts",
						"created_at":        "2025-01-01T00:00:00Z",
						"updated_at":        "2025-01-01T00:00:00Z",
					},
					map[string]interface{}{
						"id":                uuid.New().String(),
						"tenant_id":         tenantID,
						"parent_model_type": "ACCOUNT",
						"child_model_type":  "POSITION",
						"allowed":           true,
						"ownership_types":   `["SHARE_BASED"]`,
						"description":       "Account owns positions",
						"created_at":        "2025-01-01T00:00:00Z",
						"updated_at":        "2025-01-01T00:00:00Z",
					},
				},
			}, nil
		},
	}

	service := NewHierarchyServiceWithHasura(nil, mockClient)
	ctx := context.Background()

	rules, err := service.GetHierarchyRules(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetHierarchyRules failed: %v", err)
	}

	if len(rules) != 2 {
		t.Errorf("Expected 2 rules, got %d", len(rules))
	}
}

func TestGetHierarchyStatsWithHasura(t *testing.T) {
	tenantID := uuid.New().String()

	mockClient := &mockHasuraClient{
		queryFunc: func(query string, variables map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{
				"entities_aggregate": map[string]interface{}{
					"aggregate": map[string]interface{}{
						"count": float64(42),
					},
				},
			}, nil
		},
	}

	service := NewHierarchyServiceWithHasura(nil, mockClient)
	ctx := context.Background()

	stats, err := service.GetHierarchyStats(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetHierarchyStats failed: %v", err)
	}

	if stats.TotalEntities != 42 {
		t.Errorf("Expected 42 entities, got %d", stats.TotalEntities)
	}
}

func TestCreateHierarchyRuleWithHasura(t *testing.T) {
	tenantID := uuid.New().String()

	mockClient := &mockHasuraClient{
		mutateFunc: func(mutation string, variables map[string]interface{}) (map[string]interface{}, error) {
			object := variables["object"].(map[string]interface{})
			return map[string]interface{}{
				"insert_entity_hierarchy_rules_one": map[string]interface{}{
					"id":         object["id"],
					"created_at": time.Now().Format(time.RFC3339),
					"updated_at": time.Now().Format(time.RFC3339),
				},
			}, nil
		},
	}

	service := NewHierarchyServiceWithHasura(nil, mockClient)
	ctx := context.Background()

	rule := &HierarchyRule{
		TenantID:        tenantID,
		ParentModelType: "PORTFOLIO",
		ChildModelType:  "ACCOUNT",
		Allowed:         true,
		OwnershipTypes:  StringArray{"PERCENT_BASED"},
		Description:     "Test rule",
		Notes:           "Created via Hasura",
	}

	err := service.CreateHierarchyRule(ctx, rule)
	if err != nil {
		t.Fatalf("CreateHierarchyRule failed: %v", err)
	}

	if rule.ID == "" {
		t.Error("Expected rule ID to be set")
	}
}

func TestUpdateHierarchyRuleWithHasura(t *testing.T) {
	tenantID := uuid.New().String()
	ruleID := uuid.New().String()

	mockClient := &mockHasuraClient{
		mutateFunc: func(mutation string, variables map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{
				"update_entity_hierarchy_rules": map[string]interface{}{
					"affected_rows": float64(1),
					"returning": []interface{}{
						map[string]interface{}{
							"id":         ruleID,
							"updated_at": time.Now().Format(time.RFC3339),
						},
					},
				},
			}, nil
		},
	}

	service := NewHierarchyServiceWithHasura(nil, mockClient)
	ctx := context.Background()

	rule := &HierarchyRule{
		ID:              ruleID,
		TenantID:        tenantID,
		ParentModelType: "PORTFOLIO",
		ChildModelType:  "ACCOUNT",
		Allowed:         false,
		OwnershipTypes:  StringArray{"SHARE_BASED"},
		Description:     "Updated rule",
	}

	err := service.UpdateHierarchyRule(ctx, rule)
	if err != nil {
		t.Fatalf("UpdateHierarchyRule failed: %v", err)
	}
}

func TestUpdateHierarchyRuleNotFound(t *testing.T) {
	tenantID := uuid.New().String()
	ruleID := uuid.New().String()

	mockClient := &mockHasuraClient{
		mutateFunc: func(mutation string, variables map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{
				"update_entity_hierarchy_rules": map[string]interface{}{
					"affected_rows": float64(0),
					"returning":     []interface{}{},
				},
			}, nil
		},
	}

	service := NewHierarchyServiceWithHasura(nil, mockClient)
	ctx := context.Background()

	rule := &HierarchyRule{
		ID:              ruleID,
		TenantID:        tenantID,
		ParentModelType: "PORTFOLIO",
		ChildModelType:  "ACCOUNT",
		Allowed:         false,
	}

	err := service.UpdateHierarchyRule(ctx, rule)
	if err == nil {
		t.Fatal("Expected error for nonexistent rule, got nil")
	}
}

func TestDeleteHierarchyRuleWithHasura(t *testing.T) {
	tenantID := uuid.New().String()

	mockClient := &mockHasuraClient{
		mutateFunc: func(mutation string, variables map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{
				"delete_entity_hierarchy_rules": map[string]interface{}{
					"affected_rows": float64(1),
				},
			}, nil
		},
	}

	service := NewHierarchyServiceWithHasura(nil, mockClient)
	ctx := context.Background()

	err := service.DeleteHierarchyRule(ctx, tenantID, "PORTFOLIO", "ACCOUNT")
	if err != nil {
		t.Fatalf("DeleteHierarchyRule failed: %v", err)
	}
}

func TestDeleteHierarchyRuleNotFound(t *testing.T) {
	tenantID := uuid.New().String()

	mockClient := &mockHasuraClient{
		mutateFunc: func(mutation string, variables map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{
				"delete_entity_hierarchy_rules": map[string]interface{}{
					"affected_rows": float64(0),
				},
			}, nil
		},
	}

	service := NewHierarchyServiceWithHasura(nil, mockClient)
	ctx := context.Background()

	err := service.DeleteHierarchyRule(ctx, tenantID, "NONEXISTENT", "NONEXISTENT")
	if err == nil {
		t.Fatal("Expected error for nonexistent rule, got nil")
	}
}

func TestLogHierarchyAuditWithHasura(t *testing.T) {
	tenantID := uuid.New().String()
	entityID := uuid.New().String()

	mockClient := &mockHasuraClient{
		mutateFunc: func(mutation string, variables map[string]interface{}) (map[string]interface{}, error) {
			object := variables["object"].(map[string]interface{})
			return map[string]interface{}{
				"insert_entity_hierarchy_audit_log_one": map[string]interface{}{
					"id":         object["id"],
					"created_at": time.Now().Format(time.RFC3339),
				},
			}, nil
		},
	}

	service := NewHierarchyServiceWithHasura(nil, mockClient)
	ctx := context.Background()

	log := &HierarchyAuditLog{
		EntityID:        entityID,
		TenantID:        tenantID,
		Action:          "CREATE",
		ParentModelType: "PORTFOLIO",
		ChildModelType:  "ACCOUNT",
		Reason:          "Test audit log",
	}

	err := service.LogHierarchyAudit(ctx, log)
	if err != nil {
		t.Fatalf("LogHierarchyAudit failed: %v", err)
	}

	if log.ID == "" {
		t.Error("Expected log ID to be set")
	}
}

func TestGetHierarchyAuditLogWithHasura(t *testing.T) {
	entityID := uuid.New().String()
	tenantID := uuid.New().String()

	mockClient := &mockHasuraClient{
		queryFunc: func(query string, variables map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{
				"entity_hierarchy_audit_log": []interface{}{
					map[string]interface{}{
						"id":                uuid.New().String(),
						"entity_id":         entityID,
						"tenant_id":         tenantID,
						"action":            "CREATE",
						"parent_model_type": "PORTFOLIO",
						"child_model_type":  "ACCOUNT",
						"reason":            "Created relationship",
						"created_at":        "2025-01-01T00:00:00Z",
					},
					map[string]interface{}{
						"id":                uuid.New().String(),
						"entity_id":         entityID,
						"tenant_id":         tenantID,
						"action":            "UPDATE",
						"parent_model_type": "PORTFOLIO",
						"child_model_type":  "ACCOUNT",
						"reason":            "Updated ownership",
						"created_at":        "2025-01-02T00:00:00Z",
					},
				},
			}, nil
		},
	}

	service := NewHierarchyServiceWithHasura(nil, mockClient)
	ctx := context.Background()

	logs, err := service.GetHierarchyAuditLog(ctx, entityID, 10)
	if err != nil {
		t.Fatalf("GetHierarchyAuditLog failed: %v", err)
	}

	if len(logs) != 2 {
		t.Errorf("Expected 2 audit logs, got %d", len(logs))
	}

	if logs[0].Action != "CREATE" {
		t.Errorf("Expected first log action CREATE, got %s", logs[0].Action)
	}
}
