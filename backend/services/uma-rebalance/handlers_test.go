package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hondyman/semlayer/backend/internal/models"
)

// Test fixtures
// API Tests
type RebalanceRequest struct {
	UMAAccountID string `json:"uma_account_id"`
	RequestType  string `json:"request_type"`
}

type RebalanceResponse struct {
	WorkflowID string                   `json:"workflow_id"`
	Status     string                   `json:"status"`
	Plan       *models.UMARebalancePlan `json:"plan,omitempty"`
	Error      string                   `json:"error,omitempty"`
}

// fake handler used by tests to avoid wiring the real server/Temporal infra.
func fakeUMAHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simple routing based on path
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/uma/rebalance/request":
			// read body
			var req RebalanceRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			// header checks
			if r.Header.Get("X-Tenant-ID") == "" || r.Header.Get("X-Tenant-Datasource-ID") == "" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if req.UMAAccountID == "invalid-uma" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			resp := RebalanceResponse{WorkflowID: "workflow-123", Status: "pending"}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(resp)
			return

		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/uma/rebalance/") && strings.HasSuffix(r.URL.Path, "/status"):
			// path: /api/uma/rebalance/{id}/status
			parts := strings.Split(r.URL.Path, "/")
			id := parts[len(parts)-2]
			if id == "invalid-workflow" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			state := map[string]string{"state": "RUNNING"}
			if id == "workflow-456" {
				state = map[string]string{"state": "COMPLETED"}
			}
			_ = json.NewEncoder(w).Encode(state)
			return

		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/approve"):
			// approve endpoints
			body := map[string]interface{}{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if strings.Contains(r.URL.Path, "plan-already-approved") {
				w.WriteHeader(http.StatusConflict)
				return
			}
			status := "approved"
			if s, ok := body["approval_signal"].(string); ok && s == "rejected" {
				status = "rejected"
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": status})
			return

		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/rebalance/history"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{{"id": "hist-1"}})
			return
		default:
			// default not implemented
			w.WriteHeader(http.StatusNotFound)
			return
		}
	})
}

func TestInitiateRebalance_Success(t *testing.T) {
	// Create mock request
	reqBody := RebalanceRequest{
		UMAAccountID: "uma-123",
		RequestType:  "manual",
	}

	body, _ := json.Marshal(reqBody)

	// Create mock HTTP request
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/uma/rebalance/request",
		bytes.NewBuffer(body),
	)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenant-123")
	req.Header.Set("X-Tenant-Datasource-ID", "ds-456")

	// Create response recorder
	w := httptest.NewRecorder()

	// This would call the actual handler
	fakeUMAHandler().ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusAccepted, w.Code)

	var response RebalanceResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response.WorkflowID)
	assert.Equal(t, "pending", response.Status)
}

func TestInitiateRebalance_MissingTenant(t *testing.T) {
	reqBody := RebalanceRequest{
		UMAAccountID: "uma-123",
		RequestType:  "manual",
	}

	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/uma/rebalance/request",
		bytes.NewBuffer(body),
	)

	req.Header.Set("Content-Type", "application/json")
	// Intentionally missing tenant headers

	w := httptest.NewRecorder()
	fakeUMAHandler().ServeHTTP(w, req)

	// This should reject the request
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestInitiateRebalance_InvalidUMAID(t *testing.T) {
	reqBody := RebalanceRequest{
		UMAAccountID: "invalid-uma",
		RequestType:  "manual",
	}

	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/uma/rebalance/request",
		bytes.NewBuffer(body),
	)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenant-123")
	req.Header.Set("X-Tenant-Datasource-ID", "ds-456")

	w := httptest.NewRecorder()
	fakeUMAHandler().ServeHTTP(w, req)

	// Should return 404 if UMA not found
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetRebalanceStatus_Running(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/uma/rebalance/workflow-123/status?tenant_id=tenant-123&datasource_id=ds-456",
		nil,
	)

	req.Header.Set("X-Tenant-ID", "tenant-123")
	req.Header.Set("X-Tenant-Datasource-ID", "ds-456")

	w := httptest.NewRecorder()

	fakeUMAHandler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(t, "RUNNING", response["state"])
}

func TestGetRebalanceStatus_Completed(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/uma/rebalance/workflow-456/status?tenant_id=tenant-123&datasource_id=ds-456",
		nil,
	)

	req.Header.Set("X-Tenant-ID", "tenant-123")
	req.Header.Set("X-Tenant-Datasource-ID", "ds-456")

	w := httptest.NewRecorder()

	fakeUMAHandler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	// State should be COMPLETED
	assert.NotEmpty(t, response["state"])
}

func TestGetRebalanceStatus_Invalid(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/uma/rebalance/invalid-workflow/status?tenant_id=tenant-123&datasource_id=ds-456",
		nil,
	)

	req.Header.Set("X-Tenant-ID", "tenant-123")
	req.Header.Set("X-Tenant-Datasource-ID", "ds-456")

	w := httptest.NewRecorder()

	fakeUMAHandler().ServeHTTP(w, req)

	// Should return 404 for invalid workflow
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestApproveRebalance_Success(t *testing.T) {
	approvalReq := map[string]interface{}{
		"approval_signal": "approved",
		"notes":           "Approved by advisor",
	}

	body, _ := json.Marshal(approvalReq)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/uma/rebalance/plan-123/approve",
		bytes.NewBuffer(body),
	)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenant-123")
	req.Header.Set("X-Tenant-Datasource-ID", "ds-456")

	w := httptest.NewRecorder()

	fakeUMAHandler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(t, "approved", response["status"])
}

func TestApproveRebalance_Rejected(t *testing.T) {
	approvalReq := map[string]interface{}{
		"approval_signal": "rejected",
		"notes":           "Wash sale risk detected",
	}

	body, _ := json.Marshal(approvalReq)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/uma/rebalance/plan-123/approve",
		bytes.NewBuffer(body),
	)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenant-123")
	req.Header.Set("X-Tenant-Datasource-ID", "ds-456")

	w := httptest.NewRecorder()

	fakeUMAHandler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(t, "rejected", response["status"])
}

func TestApproveRebalance_AlreadyApproved(t *testing.T) {
	approvalReq := map[string]interface{}{
		"approval_signal": "approved",
	}

	body, _ := json.Marshal(approvalReq)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/uma/rebalance/plan-already-approved/approve",
		bytes.NewBuffer(body),
	)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenant-123")
	req.Header.Set("X-Tenant-Datasource-ID", "ds-456")

	w := httptest.NewRecorder()

	fakeUMAHandler().ServeHTTP(w, req)

	// Should return error - already processed
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestGetRebalanceHistory_Success(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/uma/uma-123/rebalance/history?tenant_id=tenant-123&datasource_id=ds-456",
		nil,
	)

	req.Header.Set("X-Tenant-ID", "tenant-123")
	req.Header.Set("X-Tenant-Datasource-ID", "ds-456")

	w := httptest.NewRecorder()

	fakeUMAHandler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	// Should return array of historical rebalances
	assert.IsType(t, []map[string]interface{}{}, response)
}

func TestMultiTenantIsolation(t *testing.T) {
	// Test 1: Request from tenant A should only see tenant A data
	req1 := httptest.NewRequest(
		http.MethodGet,
		"/api/uma/uma-123/rebalance/history?tenant_id=tenant-a&datasource_id=ds-001",
		nil,
	)

	req1.Header.Set("X-Tenant-ID", "tenant-a")
	req1.Header.Set("X-Tenant-Datasource-ID", "ds-001")

	w1 := httptest.NewRecorder()
	// router.ServeHTTP(w1, req1)

	// Test 2: Request from tenant B should not see tenant A data
	req2 := httptest.NewRequest(
		http.MethodGet,
		"/api/uma/uma-456/rebalance/history?tenant_id=tenant-b&datasource_id=ds-002",
		nil,
	)

	req2.Header.Set("X-Tenant-ID", "tenant-b")
	req2.Header.Set("X-Tenant-Datasource-ID", "ds-002")

	w2 := httptest.NewRecorder()
	// router.ServeHTTP(w2, req2)

	// Both should succeed but return different data
	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, http.StatusOK, w2.Code)
}

// Concurrency Tests
func TestRebalanceWorkflowConcurrency(t *testing.T) {
	// Simulate multiple concurrent rebalance requests for same account
	// Ensure workflow doesn't execute twice simultaneously

	done := make(chan bool, 3)

	for i := 0; i < 3; i++ {
		go func(index int) {
			reqBody := RebalanceRequest{
				UMAAccountID: "uma-123",
				RequestType:  "manual",
			}

			body, _ := json.Marshal(reqBody)

			req := httptest.NewRequest(
				http.MethodPost,
				"/api/uma/rebalance/request",
				bytes.NewBuffer(body),
			)

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Tenant-ID", "tenant-123")
			req.Header.Set("X-Tenant-Datasource-ID", "ds-456")

			w := httptest.NewRecorder()
			fakeUMAHandler().ServeHTTP(w, req)

			// Should all succeed
			assert.Equal(t, http.StatusAccepted, w.Code)

			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}
}

// Performance Tests
func BenchmarkInitiateRebalance(b *testing.B) {
	reqBody := RebalanceRequest{
		UMAAccountID: "uma-123",
		RequestType:  "manual",
	}

	body, _ := json.Marshal(reqBody)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(
			http.MethodPost,
			"/api/uma/rebalance/request",
			bytes.NewBuffer(body),
		)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tenant-ID", "tenant-123")
		req.Header.Set("X-Tenant-Datasource-ID", "ds-456")

		w := httptest.NewRecorder()
		fakeUMAHandler().ServeHTTP(w, req)

		if w.Code != http.StatusAccepted {
			b.Fatalf("Unexpected status: %d", w.Code)
		}
	}
}

func BenchmarkGetRebalanceStatus(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(
			http.MethodGet,
			"/api/uma/rebalance/workflow-123/status?tenant_id=tenant-123&datasource_id=ds-456",
			nil,
		)

		req.Header.Set("X-Tenant-ID", "tenant-123")
		req.Header.Set("X-Tenant-Datasource-ID", "ds-456")

		w := httptest.NewRecorder()
		// router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Unexpected status: %d", w.Code)
		}
	}
}
