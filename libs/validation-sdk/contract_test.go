package validationsdk

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestSDKRouteContract verifies that the SDK's request paths strictly match the
// backend router's canonical route paths (/api/validation-rule-nodes/*).
func TestSDKRouteContract(t *testing.T) {
	recordedPaths := make(map[string]bool)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recordedPaths[r.URL.Path] = true
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/validation-rule-nodes/evaluate-record" {
			_ = json.NewEncoder(w).Encode(EvaluateRecordResponse{Valid: true})
			return
		}
		if r.URL.Path == "/api/validation-rule-nodes/evaluate-batch" {
			_ = json.NewEncoder(w).Encode(EvaluateBatchResponse{
				Summary: BatchSummary{TotalRecords: 1, ValidCount: 1},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	client := New(srv.URL)

	// 1. EvaluateRecord contract
	_, err := client.ValidateRecord(context.Background(), EvaluateRecordRequest{
		BOName: "trade_order",
		Record: map[string]any{"id": "rec-1"},
	})
	if err != nil {
		t.Fatalf("ValidateRecord error: %v", err)
	}
	if !recordedPaths["/api/validation-rule-nodes/evaluate-record"] {
		t.Errorf("expected path /api/validation-rule-nodes/evaluate-record to be called, got paths: %v", recordedPaths)
	}

	// 2. EvaluateBatch contract
	_, err = client.ValidateBatch(context.Background(), EvaluateBatchRequest{
		BOName:  "trade_order",
		Records: []map[string]any{{"id": "rec-1"}},
	})
	if err != nil {
		t.Fatalf("ValidateBatch error: %v", err)
	}
	if !recordedPaths["/api/validation-rule-nodes/evaluate-batch"] {
		t.Errorf("expected path /api/validation-rule-nodes/evaluate-batch to be called, got paths: %v", recordedPaths)
	}
}
