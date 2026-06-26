package rdl

import (
	"context"
	"encoding/json"
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

func TestGetRulesByTenantWithHasura(t *testing.T) {
	tenantID := uuid.New()
	ruleID1 := uuid.New()
	ruleID2 := uuid.New()

	mockClient := &mockHasuraClient{
		queryFunc: func(query string, variables map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{
				"rule_definitions": []interface{}{
					map[string]interface{}{
						"id":                     ruleID1.String(),
						"tenant_id":              tenantID.String(),
						"rule_id":                "TLH_001",
						"type":                   "tax_loss_harvesting",
						"version":                "1.0",
						"name":                   "TLH Basic Rule",
						"description":            "Basic tax loss harvesting rule",
						"jurisdiction":           "US",
						"parameters":             `{"min_loss_percentage": 10}`,
						"expression":             "input.unrealized_loss_pct >= params.min_loss_percentage",
						"scoring_formula":        "input.unrealized_loss_usd * 0.35",
						"wash_sale_config":       `{"enabled": true}`,
						"substitute_asset_rules": `{}`,
						"schedule":               `{}`,
						"notifications":          `{}`,
						"active":                 true,
						"audit":                  `{}`,
						"created_at":             "2025-01-01T00:00:00Z",
						"updated_at":             "2025-01-01T00:00:00Z",
					},
					map[string]interface{}{
						"id":           ruleID2.String(),
						"tenant_id":    tenantID.String(),
						"rule_id":      "WS_001",
						"type":         "wash_sale",
						"version":      "1.0",
						"name":         "Wash Sale Rule",
						"description":  "Prevent wash sales",
						"jurisdiction": "US",
						"parameters":   `{}`,
						"expression":   "true",
						"active":       true,
						"created_at":   "2025-01-01T00:00:00Z",
						"updated_at":   "2025-01-01T00:00:00Z",
					},
				},
			}, nil
		},
	}

	service := NewService(nil)
	service.hasura = mockClient
	ctx := context.Background()

	rules, err := service.GetRulesByTenant(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetRulesByTenant failed: %v", err)
	}

	if len(rules) != 2 {
		t.Errorf("Expected 2 rules, got %d", len(rules))
	}

	if rules[0].RuleID != "TLH_001" {
		t.Errorf("Expected rule_id TLH_001, got %s", rules[0].RuleID)
	}
	if rules[0].Type != RuleTypeTaxLossHarvesting {
		t.Errorf("Expected type tax_loss_harvesting, got %s", rules[0].Type)
	}
}

func TestCreateRuleWithHasura(t *testing.T) {
	tenantID := uuid.New()

	mockClient := &mockHasuraClient{
		mutateFunc: func(mutation string, variables map[string]interface{}) (map[string]interface{}, error) {
			object := variables["object"].(map[string]interface{})
			ruleID := object["id"].(string)

			return map[string]interface{}{
				"insert_rule_definitions_one": map[string]interface{}{
					"id":         ruleID,
					"created_at": "2025-01-01T00:00:00Z",
					"updated_at": "2025-01-01T00:00:00Z",
				},
			}, nil
		},
	}

	service := NewService(nil)
	service.hasura = mockClient
	ctx := context.Background()

	rule := &RuleDefinition{
		TenantID:     tenantID,
		RuleID:       "TLH_TEST",
		Type:         RuleTypeTaxLossHarvesting,
		Version:      "1.0",
		Name:         "Test TLH Rule",
		Description:  "Test rule for tax loss harvesting",
		Jurisdiction: "US",
		Parameters:   json.RawMessage(`{"min_loss_percentage": 15}`),
		Expression:   "input.unrealized_loss_pct >= 15",
		Active:       true,
	}

	err := service.CreateRule(ctx, rule)
	if err != nil {
		t.Fatalf("CreateRule failed: %v", err)
	}

	if rule.ID == uuid.Nil {
		t.Error("Expected valid rule ID, got nil UUID")
	}
}

func TestUpdateRuleWithHasura(t *testing.T) {
	tenantID := uuid.New()

	mockClient := &mockHasuraClient{
		mutateFunc: func(mutation string, variables map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{
				"update_rule_definitions": map[string]interface{}{
					"affected_rows": float64(1),
					"returning": []interface{}{
						map[string]interface{}{
							"id":         uuid.New().String(),
							"updated_at": time.Now().Format(time.RFC3339),
						},
					},
				},
			}, nil
		},
	}

	service := NewService(nil)
	service.hasura = mockClient
	ctx := context.Background()

	rule := &RuleDefinition{
		TenantID:    tenantID,
		RuleID:      "TLH_001",
		Version:     "1.0",
		Name:        "Updated TLH Rule",
		Description: "Updated description",
		Expression:  "input.unrealized_loss_pct >= 20",
		Active:      true,
	}

	err := service.UpdateRule(ctx, rule)
	if err != nil {
		t.Fatalf("UpdateRule failed: %v", err)
	}
}

func TestUpdateRuleNotFound(t *testing.T) {
	tenantID := uuid.New()

	mockClient := &mockHasuraClient{
		mutateFunc: func(mutation string, variables map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{
				"update_rule_definitions": map[string]interface{}{
					"affected_rows": float64(0),
					"returning":     []interface{}{},
				},
			}, nil
		},
	}

	service := NewService(nil)
	service.hasura = mockClient
	ctx := context.Background()

	rule := &RuleDefinition{
		TenantID: tenantID,
		RuleID:   "NONEXISTENT",
		Version:  "1.0",
		Name:     "Nonexistent Rule",
	}

	err := service.UpdateRule(ctx, rule)
	if err == nil {
		t.Fatal("Expected error for nonexistent rule, got nil")
	}
}

func TestGetRulesByTypeWithHasura(t *testing.T) {
	tenantID := uuid.New()
	ruleID1 := uuid.New()
	ruleID2 := uuid.New()

	mockClient := &mockHasuraClient{
		queryFunc: func(query string, variables map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{
				"rule_definitions": []interface{}{
					map[string]interface{}{
						"id":                     ruleID1.String(),
						"tenant_id":              tenantID.String(),
						"rule_id":                "TLH_001",
						"type":                   "tax_loss_harvesting",
						"version":                "1.0",
						"name":                   "TLH Rule 1",
						"description":            "Tax loss harvesting rule 1",
						"jurisdiction":           "US",
						"parameters":             `{"min_loss_percentage": 10}`,
						"expression":             "input.unrealized_loss_pct >= 10",
						"scoring_formula":        "input.unrealized_loss_usd * 0.35",
						"wash_sale_config":       `{"enabled": true}`,
						"substitute_asset_rules": `{}`,
						"schedule":               `{}`,
						"notifications":          `{}`,
						"active":                 true,
						"audit":                  `{}`,
						"created_at":             "2025-01-01T00:00:00Z",
						"updated_at":             "2025-01-01T00:00:00Z",
					},
					map[string]interface{}{
						"id":                     ruleID2.String(),
						"tenant_id":              tenantID.String(),
						"rule_id":                "TLH_002",
						"type":                   "tax_loss_harvesting",
						"version":                "1.0",
						"name":                   "TLH Rule 2",
						"description":            "Tax loss harvesting rule 2",
						"jurisdiction":           "CA",
						"parameters":             `{"min_loss_percentage": 15}`,
						"expression":             "input.unrealized_loss_pct >= 15",
						"scoring_formula":        "input.unrealized_loss_usd * 0.30",
						"wash_sale_config":       `{"enabled": false}`,
						"substitute_asset_rules": `{}`,
						"schedule":               `{}`,
						"notifications":          `{}`,
						"active":                 true,
						"audit":                  `{}`,
						"created_at":             "2025-01-01T00:00:00Z",
						"updated_at":             "2025-01-01T00:00:00Z",
					},
				},
			}, nil
		},
	}

	service := NewService(nil)
	service.hasura = mockClient
	ctx := context.Background()

	rules, err := service.GetRulesByType(ctx, tenantID, RuleTypeTaxLossHarvesting)
	if err != nil {
		t.Fatalf("GetRulesByType failed: %v", err)
	}

	if len(rules) != 2 {
		t.Errorf("Expected 2 rules, got %d", len(rules))
	}

	if rules[0].RuleID != "TLH_001" {
		t.Errorf("Expected rule_id TLH_001, got %s", rules[0].RuleID)
	}
	if rules[1].RuleID != "TLH_002" {
		t.Errorf("Expected rule_id TLH_002, got %s", rules[1].RuleID)
	}
}

func TestGetRuleByIDWithHasura(t *testing.T) {
	tenantID := uuid.New()
	ruleID := uuid.New()

	mockClient := &mockHasuraClient{
		queryFunc: func(query string, variables map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{
				"rule_definitions": []interface{}{
					map[string]interface{}{
						"id":                     ruleID.String(),
						"tenant_id":              tenantID.String(),
						"rule_id":                "TLH_001",
						"type":                   "tax_loss_harvesting",
						"version":                "2.0",
						"name":                   "TLH Rule Latest Version",
						"description":            "Latest version of TLH rule",
						"jurisdiction":           "US",
						"parameters":             `{"min_loss_percentage": 12}`,
						"expression":             "input.unrealized_loss_pct >= 12",
						"scoring_formula":        "input.unrealized_loss_usd * 0.35",
						"wash_sale_config":       `{"enabled": true}`,
						"substitute_asset_rules": `{}`,
						"schedule":               `{}`,
						"notifications":          `{}`,
						"active":                 true,
						"audit":                  `{}`,
						"created_at":             "2025-01-01T00:00:00Z",
						"updated_at":             "2025-01-02T00:00:00Z",
					},
				},
			}, nil
		},
	}

	service := NewService(nil)
	service.hasura = mockClient
	ctx := context.Background()

	rule, err := service.GetRuleByID(ctx, tenantID, "TLH_001")
	if err != nil {
		t.Fatalf("GetRuleByID failed: %v", err)
	}

	if rule.RuleID != "TLH_001" {
		t.Errorf("Expected rule_id TLH_001, got %s", rule.RuleID)
	}
	if rule.Version != "2.0" {
		t.Errorf("Expected version 2.0, got %s", rule.Version)
	}
}

func TestGetRuleByIDNotFound(t *testing.T) {
	tenantID := uuid.New()

	mockClient := &mockHasuraClient{
		queryFunc: func(query string, variables map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{
				"rule_definitions": []interface{}{},
			}, nil
		},
	}

	service := NewService(nil)
	service.hasura = mockClient
	ctx := context.Background()

	_, err := service.GetRuleByID(ctx, tenantID, "NONEXISTENT")
	if err == nil {
		t.Fatal("Expected error for nonexistent rule, got nil")
	}
}

func TestDeactivateRuleWithHasura(t *testing.T) {
	tenantID := uuid.New()

	mockClient := &mockHasuraClient{
		mutateFunc: func(mutation string, variables map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{
				"update_rule_definitions": map[string]interface{}{
					"affected_rows": float64(1),
				},
			}, nil
		},
	}

	service := NewService(nil)
	service.hasura = mockClient
	ctx := context.Background()

	err := service.DeactivateRule(ctx, tenantID, "TLH_001")
	if err != nil {
		t.Fatalf("DeactivateRule failed: %v", err)
	}
}

func TestDeactivateRuleNotFound(t *testing.T) {
	tenantID := uuid.New()

	mockClient := &mockHasuraClient{
		mutateFunc: func(mutation string, variables map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{
				"update_rule_definitions": map[string]interface{}{
					"affected_rows": float64(0),
				},
			}, nil
		},
	}

	service := NewService(nil)
	service.hasura = mockClient
	ctx := context.Background()

	err := service.DeactivateRule(ctx, tenantID, "NONEXISTENT")
	if err == nil {
		t.Fatal("Expected error for nonexistent rule, got nil")
	}
}
