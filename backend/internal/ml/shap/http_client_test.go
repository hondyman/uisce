package shap

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestHTTPClient_NewHTTPClient tests client creation
func TestHTTPClient_NewHTTPClient(t *testing.T) {
	client := NewHTTPClient("http://localhost:8000", 5*time.Second, 3)

	if client == nil {
		t.Error("Client should not be nil")
	}

	if client.baseURL != "http://localhost:8000" {
		t.Errorf("Expected baseURL http://localhost:8000, got %s", client.baseURL)
	}

	if client.timeout != 5*time.Second {
		t.Errorf("Expected timeout 5s, got %v", client.timeout)
	}

	if client.retries != 3 {
		t.Errorf("Expected retries 3, got %d", client.retries)
	}
}

func TestHTTPClient_HealthCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		resp := ServiceHealthDTO{
			Status:        "healthy",
			SHAPAvailable: true,
			UptimeSeconds: 1.0,
			Version:       "1.0",
			Timestamp:     time.Now().UTC().Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, 2*time.Second, 1)
	healthy, err := client.HealthCheck(context.Background())
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}
	if !healthy {
		t.Error("Service should be healthy")
	}
}

func TestHTTPClient_GetMetrics(t *testing.T) {
	metricsResp := map[string]interface{}{"predictions_processed": 1000}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metrics" {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(metricsResp)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, 2*time.Second, 1)
	metrics, err := client.GetMetrics(context.Background())
	if err != nil {
		t.Fatalf("GetMetrics failed: %v", err)
	}
	if metrics == nil {
		t.Error("Metrics should not be nil")
	}
	if v, ok := metrics["predictions_processed"]; !ok || v.(float64) != 1000 {
		t.Error("Expected 1000 predictions processed")
	}
}

func TestHTTPClient_ComputeBatchSHAP(t *testing.T) {
	response := ExplainBatchResponseDTO{
		TotalRequests:          2,
		SuccessfulExplanations: 2,
		FailedExplanations:     0,
		Explanations: []ExplainResponseDTO{
			{
				ChainID:   "chain-1",
				BaseValue: 0.5,
				SHAPValues: []SHAPCoefficientDTO{
					{Feature: "health_score", Index: 0, Coefficient: -0.1, Baseline: 0.5},
				},
				FeatureImportance: map[string]float64{"health_score": 1.0},
				ComputeTimeMs:     40.0,
				Timestamp:         time.Now().UTC().Format(time.RFC3339),
			},
			{
				ChainID:   "chain-2",
				BaseValue: 0.5,
				SHAPValues: []SHAPCoefficientDTO{
					{Feature: "health_score", Index: 0, Coefficient: 0.2, Baseline: 0.5},
				},
				FeatureImportance: map[string]float64{"health_score": 1.0},
				ComputeTimeMs:     38.0,
				Timestamp:         time.Now().UTC().Format(time.RFC3339),
			},
		},
		TotalComputeTimeMs: 78.0,
		Timestamp:          time.Now().UTC().Format(time.RFC3339),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/explain/batch" {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, 5*time.Second, 1)
	requests := []struct {
		ChainID, Region string
		Features        map[string]float64
		ModelVersion    string
	}{
		{ChainID: "chain-1", Region: "us-east-1", Features: map[string]float64{"health_score": 0.4}, ModelVersion: "1.0"},
		{ChainID: "chain-2", Region: "us-west-2", Features: map[string]float64{"health_score": 0.8}, ModelVersion: "1.0"},
	}

	result, err := client.ComputeBatchSHAP(context.Background(), requests, 1)
	if err != nil {
		t.Fatalf("ComputeBatchSHAP failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 results, got %d", len(result))
	}
	if _, ok := result["chain-1"]; !ok {
		t.Error("Expected chain-1 in results")
	}
	if _, ok := result["chain-2"]; !ok {
		t.Error("Expected chain-2 in results")
	}
}
