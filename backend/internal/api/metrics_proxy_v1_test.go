package api

import (
	"encoding/json"
	"github.com/hondyman/uisce/backend/internal/handlers"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCommitMetricsV1Handler(t *testing.T) {
	// Mock Prometheus server
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("query")
		w.Header().Set("Content-Type", "application/json")
		var resp PrometheusQueryResult
		resp.Status = "success"
		resp.Data.ResultType = "vector"
		// produce appropriate single-item result based on query content
		val := "0"
		metric := map[string]string{}
		if q == "sum(rate(commit_service_commits_success_total[5m]))" || containsStr(q, "commits_success_total") {
			val = "100"
		}
		if containsStr(q, "commits_failed_total") {
			val = "5"
		}
		if containsStr(q, "s3_validation_failures_total") {
			val = "2"
		}
		if containsStr(q, "idempotency_hits_total") {
			val = "12"
		}
		if containsStr(q, "histogram_quantile(0.95") {
			val = "120.5"
		}
		resp.Data.Result = []struct {
			Metric map[string]string `json:"metric"`
			Value  [2]interface{}    `json:"value"`
		}{{Metric: metric, Value: [2]interface{}{0, val}}}

		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mock.Close()

	t.Setenv("PROMETHEUS_URL", mock.URL)

	// commitMetricsV1Handler resolves the caller's security context (auth + datasource +
	// region) and scopes the Prometheus query to that tenant, so the test supplies all three.
	server := &Server{SecurityContextDeps: handlers.SecurityContextDeps{Resolver: &testMockResolver{}}}
	req := withAuthContext(httptest.NewRequest("GET", "/api/v1/metrics/commit?window=5m", nil), "test-tenant")
	req.Header.Set("X-Datasource-Id", "ds-1")
	req.Header.Set("X-Region", "us-east-1")
	w := httptest.NewRecorder()

	server.commitMetricsV1Handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d body: %s", w.Code, w.Body.String())
	}

	var res CommitMetricsV1Response
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("invalid response: %v", err)
	}

	if res.Global.SuccessCount != 100 {
		t.Fatalf("expected success count 100, got %v", res.Global.SuccessCount)
	}
	if res.Global.LatencyMs.P95 != 120.5 {
		t.Fatalf("expected p95 120.5, got %v", res.Global.LatencyMs.P95)
	}
}

func containsStr(s, sub string) bool {
	if len(s) < len(sub) {
		return false
	}
	if s == sub {
		return true
	}
	return indexOfStr(s, sub) >= 0
}

func indexOfStr(s, sub string) int {
	for i := range s {
		if len(s)-i < len(sub) {
			break
		}
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
