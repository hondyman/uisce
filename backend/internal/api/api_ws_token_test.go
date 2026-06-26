package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGetWsTokenAndValidate(t *testing.T) {
	// Setup a server instance (use nil DB since getWsToken doesn't touch DB)
	srv := &Server{}

	// Create a request body
	body := `{"jobId":"job-123","purpose":"profiler","ttl_seconds":60}`
	req := httptest.NewRequest("POST", "/api/ws/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	// Call handler
	srv.getWsToken(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var js map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&js); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	token, ok := js["token"]
	if !ok || token == "" {
		t.Fatalf("token not present in response")
	}

	// Now validate token using validateWsToken
	claims, err := srv.validateWsToken(token, "job-123")
	if err != nil {
		t.Fatalf("validateWsToken failed: %v", err)
	}

	if claims["job_id"] != "job-123" {
		t.Fatalf("unexpected job_id claim: %v", claims["job_id"])
	}
	if claims["purpose"] != "profiler" {
		t.Fatalf("unexpected purpose claim: %v", claims["purpose"])
	}
}

func TestGetWsTokenExpired(t *testing.T) {
	srv := &Server{}
	body := `{ "jobId": "job-exp", "purpose": "profiler", "ttl_seconds": 1 }`
	req := httptest.NewRequest("POST", "/api/ws/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.getWsToken(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
	var js map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&js); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	token := js["token"]
	if token == "" {
		t.Fatalf("no token returned")
	}

	// Wait for it to expire
	time.Sleep(2 * time.Second)

	if _, err := srv.validateWsToken(token, "job-exp"); err == nil {
		t.Fatalf("expected token to be expired, but validateWsToken succeeded")
	}
}

func TestHandleWebSocketAuthFailures(t *testing.T) {
	srv := &Server{}

	// Missing token
	req1 := httptest.NewRequest("GET", "/ws/profiler/job-x", nil)
	w1 := httptest.NewRecorder()
	srv.handleWebSocket(w1, req1)
	if w1.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing token, got %d", w1.Result().StatusCode)
	}

	// Invalid token
	req2 := httptest.NewRequest("GET", "/ws/profiler/job-x?token=invalid-token", nil)
	w2 := httptest.NewRecorder()
	srv.handleWebSocket(w2, req2)
	if w2.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid token, got %d", w2.Result().StatusCode)
	}
}
