package backtest

import (
	"context"
	"encoding/json"
	"testing"

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

func TestGetPortfolioWithHasura(t *testing.T) {
	portfolioID := uuid.New()
	tenantID := uuid.New()
	clientID := uuid.New()

	mockClient := &mockHasuraClient{
		queryFunc: func(query string, variables map[string]interface{}) (map[string]interface{}, error) {
			// Return mock portfolio data
			return map[string]interface{}{
				"portfolios_by_pk": map[string]interface{}{
					"id":                       portfolioID.String(),
					"tenant_id":                tenantID.String(),
					"client_id":                clientID.String(),
					"type":                     "MANAGED",
					"benchmark":                "S&P 500",
					"asset_allocation_targets": `{"stocks": 60, "bonds": 30, "cash": 10}`,
					"performance_metrics":      `{"total_return": 0.15, "ytd_return": 0.08}`,
					"advisor_discretion":       true,
					"client_approval_required": false,
					"custom_fields":            `{"notes": "test portfolio"}`,
					"created_at":               "2025-01-01T00:00:00Z",
					"updated_at":               "2025-01-01T00:00:00Z",
				},
			}, nil
		},
	}

	service := NewServiceWithHasura(nil, mockClient)
	ctx := context.Background()

	portfolio, err := service.GetPortfolio(ctx, portfolioID.String())
	if err != nil {
		t.Fatalf("GetPortfolio failed: %v", err)
	}

	if portfolio.ID != portfolioID {
		t.Errorf("Expected portfolio ID %s, got %s", portfolioID, portfolio.ID)
	}
	if portfolio.TenantID != tenantID {
		t.Errorf("Expected tenant ID %s, got %s", tenantID, portfolio.TenantID)
	}
	if portfolio.ClientID != clientID {
		t.Errorf("Expected client ID %s, got %s", clientID, portfolio.ClientID)
	}
	if portfolio.Type != "MANAGED" {
		t.Errorf("Expected type MANAGED, got %s", portfolio.Type)
	}
	if portfolio.Benchmark != "S&P 500" {
		t.Errorf("Expected benchmark S&P 500, got %s", portfolio.Benchmark)
	}
	if !portfolio.AdvisorDiscretion {
		t.Error("Expected advisor_discretion to be true")
	}
	if portfolio.ClientApprovalRequired {
		t.Error("Expected client_approval_required to be false")
	}
}

func TestGetPortfolioNotFound(t *testing.T) {
	portfolioID := uuid.New()

	mockClient := &mockHasuraClient{
		queryFunc: func(query string, variables map[string]interface{}) (map[string]interface{}, error) {
			// Return null for portfolio not found
			return map[string]interface{}{
				"portfolios_by_pk": nil,
			}, nil
		},
	}

	service := NewServiceWithHasura(nil, mockClient)
	ctx := context.Background()

	_, err := service.GetPortfolio(ctx, portfolioID.String())
	if err == nil {
		t.Error("Expected error for portfolio not found, got nil")
	}
	if err.Error() != "portfolio not found" {
		t.Errorf("Expected 'portfolio not found' error, got: %v", err)
	}
}

func TestCreatePortfolioWithHasura(t *testing.T) {
	tenantID := uuid.New()
	clientID := uuid.New()

	mockClient := &mockHasuraClient{
		mutateFunc: func(mutation string, variables map[string]interface{}) (map[string]interface{}, error) {
			// Verify mutation contains correct data
			object := variables["object"].(map[string]interface{})

			portfolioID := object["id"].(string)

			// Return the created portfolio
			return map[string]interface{}{
				"insert_portfolios_one": map[string]interface{}{
					"id":                       portfolioID,
					"tenant_id":                tenantID.String(),
					"client_id":                clientID.String(),
					"type":                     object["type"],
					"benchmark":                object["benchmark"],
					"asset_allocation_targets": object["asset_allocation_targets"],
					"performance_metrics":      object["performance_metrics"],
					"advisor_discretion":       object["advisor_discretion"],
					"client_approval_required": object["client_approval_required"],
					"custom_fields":            object["custom_fields"],
					"created_at":               "2025-01-01T00:00:00Z",
					"updated_at":               "2025-01-01T00:00:00Z",
				},
			}, nil
		},
	}

	service := NewServiceWithHasura(nil, mockClient)
	ctx := context.Background()

	req := CreatePortfolioRequest{
		Type:                   "MANAGED",
		Benchmark:              "S&P 500",
		AssetAllocationTargets: json.RawMessage(`{"stocks": 60, "bonds": 30, "cash": 10}`),
		PerformanceMetrics:     json.RawMessage(`{"total_return": 0.15}`),
		AdvisorDiscretion:      true,
		ClientApprovalRequired: false,
		CustomFields:           json.RawMessage(`{"notes": "new portfolio"}`),
	}

	portfolio, err := service.CreatePortfolio(ctx, req, tenantID.String(), clientID.String())
	if err != nil {
		t.Fatalf("CreatePortfolio failed: %v", err)
	}

	if portfolio.ID == uuid.Nil {
		t.Error("Expected valid portfolio ID, got nil UUID")
	}
	if portfolio.TenantID != tenantID {
		t.Errorf("Expected tenant ID %s, got %s", tenantID, portfolio.TenantID)
	}
	if portfolio.ClientID != clientID {
		t.Errorf("Expected client ID %s, got %s", clientID, portfolio.ClientID)
	}
	if portfolio.Type != "MANAGED" {
		t.Errorf("Expected type MANAGED, got %s", portfolio.Type)
	}
	if portfolio.Benchmark != "S&P 500" {
		t.Errorf("Expected benchmark S&P 500, got %s", portfolio.Benchmark)
	}
	if !portfolio.AdvisorDiscretion {
		t.Error("Expected advisor_discretion to be true")
	}
	if portfolio.ClientApprovalRequired {
		t.Error("Expected client_approval_required to be false")
	}
}

func TestCreatePortfolioWithCustomFields(t *testing.T) {
	tenantID := uuid.New()
	clientID := uuid.New()

	customFields := map[string]interface{}{
		"risk_tolerance":  "moderate",
		"investment_goal": "retirement",
		"notes":           "test portfolio with custom fields",
	}
	customFieldsJSON, _ := json.Marshal(customFields)

	mockClient := &mockHasuraClient{
		mutateFunc: func(mutation string, variables map[string]interface{}) (map[string]interface{}, error) {
			object := variables["object"].(map[string]interface{})
			portfolioID := object["id"].(string)

			return map[string]interface{}{
				"insert_portfolios_one": map[string]interface{}{
					"id":                       portfolioID,
					"tenant_id":                tenantID.String(),
					"client_id":                clientID.String(),
					"type":                     "DISCRETIONARY",
					"benchmark":                "NASDAQ",
					"advisor_discretion":       true,
					"client_approval_required": true,
					"custom_fields":            string(customFieldsJSON),
				},
			}, nil
		},
	}

	service := NewServiceWithHasura(nil, mockClient)
	ctx := context.Background()

	req := CreatePortfolioRequest{
		Type:                   "DISCRETIONARY",
		Benchmark:              "NASDAQ",
		AdvisorDiscretion:      true,
		ClientApprovalRequired: true,
		CustomFields:           customFieldsJSON,
	}

	portfolio, err := service.CreatePortfolio(ctx, req, tenantID.String(), clientID.String())
	if err != nil {
		t.Fatalf("CreatePortfolio failed: %v", err)
	}

	// Verify custom fields were stored
	var storedFields map[string]interface{}
	if err := json.Unmarshal(portfolio.CustomFields, &storedFields); err != nil {
		t.Fatalf("Failed to unmarshal custom fields: %v", err)
	}

	if storedFields["risk_tolerance"] != "moderate" {
		t.Errorf("Expected risk_tolerance 'moderate', got %v", storedFields["risk_tolerance"])
	}
	if storedFields["investment_goal"] != "retirement" {
		t.Errorf("Expected investment_goal 'retirement', got %v", storedFields["investment_goal"])
	}
}

func TestCreatePortfolioFailure(t *testing.T) {
	mockClient := &mockHasuraClient{
		mutateFunc: func(mutation string, variables map[string]interface{}) (map[string]interface{}, error) {
			// Simulate mutation failure (e.g., constraint violation)
			return map[string]interface{}{
				"insert_portfolios_one": nil,
			}, nil
		},
	}

	service := NewServiceWithHasura(nil, mockClient)
	ctx := context.Background()

	req := CreatePortfolioRequest{
		Type:              "MANAGED",
		AdvisorDiscretion: true,
	}

	_, err := service.CreatePortfolio(ctx, req, uuid.New().String(), uuid.New().String())
	if err == nil {
		t.Error("Expected error for failed portfolio creation, got nil")
	}
	if err.Error() != "failed to create portfolio" {
		t.Errorf("Expected 'failed to create portfolio' error, got: %v", err)
	}
}
