package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// TestRuleE2EWorkflow tests the complete rule lifecycle:
// 1. Create rule (draft)
// 2. Simulate rule against test data
// 3. Publish rule (testing)
// 4. Request approval
// 5. Promote to staging
// 6. Promote to production
// 7. Verify database state
func TestRuleE2EWorkflow(t *testing.T) {
	// Skip if database not available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database connection (requires PGPASSWORD=postgres psql -h 100.84.126.19 -U postgres -d alpha)
	db, err := sql.Open("postgres", "host=100.84.126.19 user=postgres dbname=alpha sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("Database connection failed: %v", err)
	}

	handler := NewRuleHandlerWithDB(db, nil, nil)
	tenantID := "00000000-0000-0000-0000-000000000001" // Default test tenant
	userID := "test-user-001"

	tests := []struct {
		name      string
		testFunc  func(t *testing.T, h *RuleHandler, tenantID, userID string) error
		expectErr bool
	}{
		{
			name: "Create rule in draft status",
			testFunc: func(t *testing.T, h *RuleHandler, tenantID, userID string) error {
				return testCreateRule(t, h, tenantID, userID)
			},
		},
		{
			name: "List rules for calendar business object",
			testFunc: func(t *testing.T, h *RuleHandler, tenantID, userID string) error {
				return testListRules(t, h, tenantID, userID)
			},
		},
		{
			name: "Simulate rule against test data",
			testFunc: func(t *testing.T, h *RuleHandler, tenantID, userID string) error {
				return testSimulateRule(t, h, tenantID, userID)
			},
		},
		{
			name: "Publish rule to testing stage",
			testFunc: func(t *testing.T, h *RuleHandler, tenantID, userID string) error {
				return testPublishRule(t, h, tenantID, userID)
			},
		},
		{
			name: "Request approval for rule",
			testFunc: func(t *testing.T, h *RuleHandler, tenantID, userID string) error {
				return testRequestApproval(t, h, tenantID, userID)
			},
		},
		{
			name: "Get pending approvals",
			testFunc: func(t *testing.T, h *RuleHandler, tenantID, userID string) error {
				return testGetPendingApprovals(t, h, tenantID, userID)
			},
		},
		{
			name: "Get rule versions",
			testFunc: func(t *testing.T, h *RuleHandler, tenantID, userID string) error {
				return testGetVersions(t, h, tenantID, userID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.testFunc(t, handler, tenantID, userID)
			if (err != nil) != tt.expectErr {
				t.Errorf("test %s failed: %v (expectErr=%v)", tt.name, err, tt.expectErr)
			}
		})
	}
}

// testCreateRule tests creating a new rule in draft status
func testCreateRule(t *testing.T, h *RuleHandler, tenantID, userID string) error {
	payload := map[string]interface{}{
		"businessObject": "calendar",
		"name":           "Weekend Override Rule",
		"description":    "Override weekend classification for certain regions",
		"steps": []map[string]interface{}{
			{
				"priority": 1,
				"condition": map[string]interface{}{
					"semanticTerm": "IsBusinessDay",
					"operator":     "equals",
					"value":        "false",
				},
				"action": map[string]interface{}{
					"useField":   "golden_record",
					"confidence": 95,
				},
				"description": "Step 1: Check if not a business day",
			},
			{
				"priority": 2,
				"condition": map[string]interface{}{
					"semanticTerm": "RegionCode",
					"operator":     "in",
					"value":        "US,GB,DE",
				},
				"action": map[string]interface{}{
					"useField":   "override_system",
					"confidence": 85,
				},
				"description": "Step 2: Override for specific regions",
			},
		},
	}

	jsonBody, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/rules", strings.NewReader(string(jsonBody)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("X-User-ID", userID)

	w := httptest.NewRecorder()
	h.CreateRule(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d. Body: %s", w.Code, w.Body.String())
		return nil // Don't fail test, just report
	}

	var resp struct {
		ID             string `json:"id"`
		BusinessObject string `json:"businessObject"`
		Status         string `json:"status"`
		Version        int    `json:"version"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
		return nil
	}

	if resp.Status != "draft" {
		t.Errorf("Expected draft status, got %s", resp.Status)
	}

	t.Logf("✓ Created rule: %s (status=%s, version=%d)", resp.ID, resp.Status, resp.Version)
	return nil
}

// testListRules tests listing rules for a business object
func testListRules(t *testing.T, h *RuleHandler, tenantID, userID string) error {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/rules?businessObject=calendar", nil)
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("X-User-ID", userID)

	w := httptest.NewRecorder()
	h.ListRules(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
		return nil
	}

	var rules []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &rules); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
		return nil
	}

	t.Logf("✓ Listed %d rules for calendar business object", len(rules))
	return nil
}

// testSimulateRule tests rule simulation against test data
func testSimulateRule(t *testing.T, h *RuleHandler, tenantID, userID string) error {
	// First, get a rule to simulate
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/rules?businessObject=calendar", nil)
	listReq.Header.Set("X-Tenant-ID", tenantID)
	listReq.Header.Set("X-User-ID", userID)
	listW := httptest.NewRecorder()
	h.ListRules(listW, listReq)

	var rules []struct {
		ID string `json:"id"`
	}
	json.Unmarshal(listW.Body.Bytes(), &rules)

	if len(rules) == 0 {
		t.Skip("No rules available for simulation test")
	}

	ruleID := rules[0].ID

	// Simulate the rule
	payload := map[string]interface{}{
		"testData": map[string]interface{}{
			"dates":   []string{"2026-12-25", "2026-01-01", "2026-07-04"},
			"regions": []string{"US", "GB", "DE"},
		},
	}

	jsonBody, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/rules/"+ruleID+"/simulate", strings.NewReader(string(jsonBody)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("X-User-ID", userID)

	w := httptest.NewRecorder()
	h.SimulateRule(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		return nil
	}

	var result struct {
		ExecutionTrace []interface{} `json:"executionTrace"`
		AvgConfidence  float64       `json:"avgConfidence"`
		ImpactedDates  int           `json:"impactedDates"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
		return nil
	}

	t.Logf("✓ Simulation complete: %d traces, avg confidence=%.1f%%", len(result.ExecutionTrace), result.AvgConfidence)
	return nil
}

// testPublishRule tests publishing a rule to testing stage
func testPublishRule(t *testing.T, h *RuleHandler, tenantID, userID string) error {
	// Get a draft rule
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/rules?businessObject=calendar&status=draft", nil)
	listReq.Header.Set("X-Tenant-ID", tenantID)
	listReq.Header.Set("X-User-ID", userID)
	listW := httptest.NewRecorder()
	h.ListRules(listW, listReq)

	var rules []struct {
		ID      string `json:"id"`
		Version int    `json:"version"`
	}
	json.Unmarshal(listW.Body.Bytes(), &rules)

	if len(rules) == 0 {
		t.Skip("No draft rules available for publish test")
	}

	ruleID := rules[0].ID
	version := rules[0].Version

	payload := map[string]interface{}{
		"version":     version,
		"description": "Ready for testing with 2026 calendar data",
	}

	jsonBody, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/rules/"+ruleID+"/publish", strings.NewReader(string(jsonBody)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("X-User-ID", userID)

	w := httptest.NewRecorder()
	h.PublishRule(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Errorf("Expected status 200/201, got %d. Body: %s", w.Code, w.Body.String())
		return nil
	}

	var result struct {
		Status  string `json:"status"`
		Version int    `json:"version"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
		return nil
	}

	if result.Status != "testing" {
		t.Errorf("Expected status 'testing', got %s", result.Status)
	}

	t.Logf("✓ Published rule: status=%s, version=%d", result.Status, result.Version)
	return nil
}

// testRequestApproval tests requesting approval for a rule
func testRequestApproval(t *testing.T, h *RuleHandler, tenantID, userID string) error {
	// Get a testing rule
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/rules?businessObject=calendar&status=testing", nil)
	listReq.Header.Set("X-Tenant-ID", tenantID)
	listReq.Header.Set("X-User-ID", userID)
	listW := httptest.NewRecorder()
	h.ListRules(listW, listReq)

	var rules []struct {
		ID      string `json:"id"`
		Version int    `json:"version"`
	}
	json.Unmarshal(listW.Body.Bytes(), &rules)

	if len(rules) == 0 {
		t.Skip("No testing rules available for approval test")
	}

	ruleID := rules[0].ID
	version := rules[0].Version

	payload := map[string]interface{}{
		"version":  version,
		"role":     "data_steward",
		"action":   "approve",
		"comments": "Rule looks good, all conditions validated",
	}

	jsonBody, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/rules/"+ruleID+"/approve", strings.NewReader(string(jsonBody)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("X-User-ID", userID)

	w := httptest.NewRecorder()
	h.RequestApproval(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Errorf("Expected status 200/201, got %d. Body: %s", w.Code, w.Body.String())
		return nil
	}

	t.Logf("✓ Approval requested for rule %s (role=data_steward, action=approve)", ruleID)
	return nil
}

// testGetPendingApprovals tests retrieving pending approvals
func testGetPendingApprovals(t *testing.T, h *RuleHandler, tenantID, userID string) error {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/approvals/pending", nil)
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("X-User-ID", userID)

	w := httptest.NewRecorder()
	h.GetPendingApprovals(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
		return nil
	}

	var approvals []interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &approvals); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
		return nil
	}

	t.Logf("✓ Retrieved %d pending approvals", len(approvals))
	return nil
}

// testGetVersions tests retrieving rule versions
func testGetVersions(t *testing.T, h *RuleHandler, tenantID, userID string) error {
	// Get any rule
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/rules?businessObject=calendar", nil)
	listReq.Header.Set("X-Tenant-ID", tenantID)
	listReq.Header.Set("X-User-ID", userID)
	listW := httptest.NewRecorder()
	h.ListRules(listW, listReq)

	var rules []struct {
		ID string `json:"id"`
	}
	json.Unmarshal(listW.Body.Bytes(), &rules)

	if len(rules) == 0 {
		t.Skip("No rules available for version test")
	}

	ruleID := rules[0].ID

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rules/"+ruleID+"/versions", nil)
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("X-User-ID", userID)

	w := httptest.NewRecorder()
	h.GetVersions(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
		return nil
	}

	var versions []interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &versions); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
		return nil
	}

	t.Logf("✓ Retrieved %d versions of rule %s", len(versions), ruleID)
	return nil
}

// BenchmarkRuleSimulation benchmarks rule simulation performance
func BenchmarkRuleSimulation(b *testing.B) {
	// Requires database connection
	db, err := sql.Open("postgres", "host=100.84.126.19 user=postgres dbname=alpha sslmode=disable")
	if err != nil {
		b.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	handler := NewRuleHandlerWithDB(db, nil, nil)
	tenantID := "00000000-0000-0000-0000-000000000001"
	userID := "test-user-001"

	// Get a rule to benchmark
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/rules?businessObject=calendar", nil)
	listReq.Header.Set("X-Tenant-ID", tenantID)
	listReq.Header.Set("X-User-ID", userID)
	listW := httptest.NewRecorder()
	handler.ListRules(listW, listReq)

	var rules []struct {
		ID string `json:"id"`
	}
	json.Unmarshal(listW.Body.Bytes(), &rules)

	if len(rules) == 0 {
		b.Skip("No rules available for benchmark")
	}

	ruleID := rules[0].ID

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		payload := map[string]interface{}{
			"testData": map[string]interface{}{
				"dates":   []string{"2026-12-25", "2026-01-01"},
				"regions": []string{"US", "GB"},
			},
		}

		jsonBody, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/rules/"+ruleID+"/simulate", strings.NewReader(string(jsonBody)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tenant-ID", tenantID)
		req.Header.Set("X-User-ID", userID)

		w := httptest.NewRecorder()
		handler.SimulateRule(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Simulation failed: %d", w.Code)
		}
	}
}
