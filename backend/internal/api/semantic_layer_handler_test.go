package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hondyman/semlayer/backend/internal/analytics"
)

func TestPlanHandler_RejectsMissingRegion(t *testing.T) {
	// Set up handler with analytics service (TranslateNaturalLanguageToQuery returns a query without region by default)
	analyticsSvc := analytics.NewSemanticService(nil)
	h := NewSemanticLayerHandler(nil, analyticsSvc, nil)

	// Build request
	body := bytes.NewBufferString(`{"datasource":"customers","prompt":"Show me revenue by region","mode":"exploratory"}`)
	req := httptest.NewRequest("POST", "/api/semantic/plan", body)
	req.Header.Set("X-Tenant-ID", "tenant-123")
	// req.Header.Set("X-Tenant-Region", "EMEA") // Intentionally missing to trigger 400

	rw := httptest.NewRecorder()

	// Call handler
	h.PlanHandler(rw, req)

	res := rw.Result()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400; got %d", res.StatusCode)
	}

	// Optionally verify body contains our error message
	// (we simply assert non-empty body to keep the test resilient)
	if rw.Body.Len() == 0 {
		t.Fatalf("expected error body, got empty")
	}
}
