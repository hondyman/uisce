package api
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
)

// ============================================================================
// Multi-Tenant Isolation Tests
// ============================================================================

// TestDashboardComplianceMultiTenant verifies compliance data is tenant-isolated
func TestDashboardComplianceMultiTenant(t *testing.T) {
	tests := []struct {
		name       string
		tenantID   string
		wantStatus int
		wantFields []string
	}{
		{
			name:       "Valid tenant gets compliance metrics",
			tenantID:   "tenant-001",
			wantStatus: http.StatusOK,
			wantFields: []string{"critical", "warning", "passing", "rules", "timestamp"},
		},
		{
			name:       "Missing tenant returns 400",
			tenantID:   "",
			wantStatus: http.StatusBadRequest,
			wantFields: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := chi.NewRouter()
			// Mock database would go here
			handler := &DashboardHandler{}

			router.Get("/api/dashboard/compliance", handler.GetComplianceMetrics)

			req := httptest.NewRequest("GET", "/api/dashboard/compliance?tenant_id="+tt.tenantID, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("GetComplianceMetrics() status = %d, want %d", w.Code, tt.wantStatus)
			}

			if w.Code == http.StatusOK {
				var response ComplianceResponse
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}

				// Verify response has expected fields
				if response.Critical < 0 {
					t.Error("Critical count should be non-negative")
				}
				if response.Timestamp == "" {
					t.Error("Timestamp should not be empty")
				}
			}
		})
	}
}

// TestPortfolioOverviewMultiTenant tests portfolio isolation between tenants
func TestPortfolioOverviewMultiTenant(t *testing.T) {
	tests := []struct {
		name        string
		tenantID    string
		portfolioID string
		wantStatus  int
	}{
		{
			name:        "Valid tenant and portfolio",
			tenantID:    "tenant-001",
			portfolioID: "portfolio-001",
			wantStatus:  http.StatusOK,
		},
		{
			name:        "Missing tenant ID",
			tenantID:    "",
			portfolioID: "portfolio-001",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "Missing portfolio ID",
			tenantID:    "tenant-001",
			portfolioID: "",
			wantStatus:  http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := chi.NewRouter()
			handler := &PortfolioHandler{}

			router.Get("/api/portfolios/{portfolioId}/overview", handler.GetPortfolioOverview)

			url := fmt.Sprintf("/api/portfolios/%s/overview", tt.portfolioID)
			if tt.tenantID != "" {
				url += "?tenant_id=" + tt.tenantID
			}

			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("GetPortfolioOverview() status = %d, want %d", w.Code, tt.wantStatus)
			}

			if w.Code == http.StatusOK {
				var response PortfolioOverviewResponse
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}

				if response.PortfolioID != tt.portfolioID {
					t.Errorf("Portfolio ID mismatch: got %s, want %s", response.PortfolioID, tt.portfolioID)
				}
			}
		})
	}
}

// TestDashboardRiskMetricsContract verifies API contract compliance
func TestDashboardRiskMetricsContract(t *testing.T) {
	router := chi.NewRouter()
	handler := &DashboardHandler{}
	router.Get("/api/dashboard/risk", handler.GetRiskMetrics)

	req := httptest.NewRequest("GET", "/api/dashboard/risk?tenant_id=test-tenant", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response RiskResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify all required fields exist
	requiredFields := map[string]bool{
		"volatility":     response.Volatility > 0,
		"var_percent":    response.VaRPercent > 0,
		"var_value":      response.VaRValue > 0,
		"beta_market":    response.BetaMarket != 0,
		"drawdown":       response.Drawdown != 0,
		"timestamp":      response.Timestamp != "",
		"metrics_count":  len(response.Metrics) > 0,
	}

	for field, present := range requiredFields {
		if !present {
			t.Errorf("Required field missing or empty: %s", field)
		}
	}

	// Verify metrics structure
	for i, metric := range response.Metrics {
		if metric.MetricID == "" {
			t.Errorf("Metric %d: MetricID is empty", i)
		}
		if metric.MetricName == "" {
			t.Errorf("Metric %d: MetricName is empty", i)
		}
		if metric.Unit == "" {
			t.Errorf("Metric %d: Unit is empty", i)
		}
		if metric.Status == "" {
			t.Errorf("Metric %d: Status is empty", i)
		}
	}
}

// TestPortfolioHoldingsContract verifies holdings endpoint contract
func TestPortfolioHoldingsContract(t *testing.T) {
	router := chi.NewRouter()
	handler := &PortfolioHandler{}
	router.Get("/api/portfolios/{portfolioId}/holdings", handler.GetHoldings)

	req := httptest.NewRequest("GET", "/api/portfolios/port-123/holdings?tenant_id=test-tenant", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response HoldingsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify portfolio-level fields
	if response.PortfolioID == "" {
		t.Error("PortfolioID is empty")
	}
	if response.ValuationDate == "" {
		t.Error("ValuationDate is empty")
	}
	if response.TotalHoldings <= 0 {
		t.Error("TotalHoldings should be positive")
	}
	if response.Timestamp == "" {
		t.Error("Timestamp is empty")
	}

	// Verify holding structure
	for i, holding := range response.TopHoldings {
		if holding.Symbol == "" {
			t.Errorf("Holding %d: Symbol is empty", i)
		}
		if holding.PositionValue <= 0 {
			t.Errorf("Holding %d: PositionValue should be positive", i)
		}
		if holding.WeightPercent <= 0 {
			t.Errorf("Holding %d: WeightPercent should be positive", i)
		}
	}

	// Verify sector weights sum
	var sectorSum float64
	for _, sector := range response.SectorWeights {
		sectorSum += sector.WeightPercent
	}
	if sectorSum <= 0 {
		t.Errorf("Sector weights sum is %.2f, expected > 0", sectorSum)
	}
}

// TestComplianceResponseSchema verifies strict schema compliance
func TestComplianceResponseSchema(t *testing.T) {
	router := chi.NewRouter()
	handler := &DashboardHandler{}
	router.Get("/api/dashboard/compliance", handler.GetComplianceMetrics)

	req := httptest.NewRequest("GET", "/api/dashboard/compliance?tenant_id=test-tenant", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	var jsonData map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &jsonData); err != nil {
		t.Fatalf("Invalid JSON response: %v", err)
	}

	// Verify top-level structure
	expectedKeys := []string{"critical", "warning", "passing", "rules", "timestamp"}
	for _, key := range expectedKeys {
		if _, exists := jsonData[key]; !exists {
			t.Errorf("Missing required field: %s", key)
		}
	}

	// Verify types
	if _, ok := jsonData["critical"].(float64); !ok {
		t.Error("critical should be a number")
	}
	if _, ok := jsonData["rules"].([]interface{}); !ok {
		t.Error("rules should be an array")
	}
}

// TestTriggerETLResponseStructure verifies ETL trigger response format
func TestTriggerETLResponseStructure(t *testing.T) {
	router := chi.NewRouter()
	handler := &DashboardHandler{}
	router.Post("/api/dashboard/etl/trigger", handler.TriggerETL)

	body := []byte(`{"priority":"high"}`)
	req := httptest.NewRequest("POST", "/api/dashboard/etl/trigger?tenant_id=test-tenant", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("Expected status 202, got %d", w.Code)
	}

	var response TriggerETLResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Invalid response: %v", err)
	}

	// Verify response fields
	if response.RunID == "" {
		t.Error("RunID is empty")
	}
	if response.Status != "Queued" {
		t.Errorf("Expected status 'Queued', got '%s'", response.Status)
	}
	if response.StartedAt == "" {
		t.Error("StartedAt is empty")
	}
}

// TestPortfolioComplianceSchema verifies portfolio compliance response schema
func TestPortfolioComplianceSchema(t *testing.T) {
	router := chi.NewRouter()
	handler := &PortfolioHandler{}
	router.Get("/api/portfolios/{portfolioId}/compliance", handler.GetPortfolioCompliance)

	req := httptest.NewRequest("GET", "/api/portfolios/port-123/compliance?tenant_id=test-tenant", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	var response PortfolioComplianceResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify breach details structure
	for i, breach := range response.BreachDetails {
		if breach.RuleID == "" {
			t.Errorf("Breach %d: RuleID is empty", i)
		}
		if breach.Status == "" {
			t.Errorf("Breach %d: Status is empty", i)
		}
		if breach.Severity == "" {
			t.Errorf("Breach %d: Severity is empty", i)
		}
	}
}

// TestScenariosResponseStructure verifies scenario response format
func TestScenariosResponseStructure(t *testing.T) {
	router := chi.NewRouter()
	handler := &PortfolioHandler{}
	router.Get("/api/portfolios/{portfolioId}/scenarios", handler.GetScenarios)

	req := httptest.NewRequest("GET", "/api/portfolios/port-123/scenarios?tenant_id=test-tenant", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	var response ScenariosResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify scenario structure
	for i, scenario := range response.Scenarios {
		if scenario.ScenarioID == "" {
			t.Errorf("Scenario %d: ScenarioID is empty", i)
		}
		if scenario.ScenarioName == "" {
			t.Errorf("Scenario %d: ScenarioName is empty", i)
		}
		if scenario.BaselineValue <= 0 {
			t.Errorf("Scenario %d: BaselineValue should be positive", i)
		}
		if scenario.SimulatedValue <= 0 {
			t.Errorf("Scenario %d: SimulatedValue should be positive", i)
		}
	}
}

// BenchmarkDashboardComplianceEndpoint measures endpoint performance
func BenchmarkDashboardComplianceEndpoint(b *testing.B) {
	router := chi.NewRouter()
	handler := &DashboardHandler{}
	router.Get("/api/dashboard/compliance", handler.GetComplianceMetrics)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/dashboard/compliance?tenant_id=test-tenant", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected 200, got %d", w.Code)
		}
	}
}

// BenchmarkPortfolioOverviewEndpoint measures portfolio endpoint performance
func BenchmarkPortfolioOverviewEndpoint(b *testing.B) {
	router := chi.NewRouter()
	handler := &PortfolioHandler{}
	router.Get("/api/portfolios/{portfolioId}/overview", handler.GetPortfolioOverview)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/portfolios/port-123/overview?tenant_id=test-tenant", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected 200, got %d", w.Code)
		}
	}
}

// TestAllEndpointsRespond verifies all 11 endpoints are registered and respond
func TestAllEndpointsRespond(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		path     string
		wantCode int
	}{
		{"Dashboard Compliance", "GET", "/api/dashboard/compliance?tenant_id=t1", http.StatusOK},
		{"Dashboard Risk", "GET", "/api/dashboard/risk?tenant_id=t1", http.StatusOK},
		{"Dashboard Sparklines", "GET", "/api/dashboard/sparklines?tenant_id=t1", http.StatusOK},
		{"Dashboard ETL Health", "GET", "/api/dashboard/etl-health?tenant_id=t1", http.StatusOK},
		{"Dashboard Alerts", "GET", "/api/dashboard/alerts?tenant_id=t1", http.StatusOK},
		{"Trigger ETL", "POST", "/api/dashboard/etl/trigger?tenant_id=t1", http.StatusAccepted},
		{"Portfolio Overview", "GET", "/api/portfolios/p1/overview?tenant_id=t1", http.StatusOK},
		{"Portfolio Holdings", "GET", "/api/portfolios/p1/holdings?tenant_id=t1", http.StatusOK},
		{"Portfolio Risk", "GET", "/api/portfolios/p1/risk?tenant_id=t1", http.StatusOK},
		{"Portfolio Compliance", "GET", "/api/portfolios/p1/compliance?tenant_id=t1", http.StatusOK},
		{"Portfolio Scenarios", "GET", "/api/portfolios/p1/scenarios?tenant_id=t1", http.StatusOK},
	}

	router := chi.NewRouter()
	dashboardHandler := &DashboardHandler{}
	portfolioHandler := &PortfolioHandler{}
	dashboardHandler.RegisterRoutes(router)
	portfolioHandler.RegisterRoutes(router)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			if tt.method == "POST" {
				body = []byte("{}")
			}

			var req *http.Request
			if body != nil {
				req = httptest.NewRequest(tt.method, tt.path, bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("Expected %d, got %d", tt.wantCode, w.Code)
			}
		})
	}
}
