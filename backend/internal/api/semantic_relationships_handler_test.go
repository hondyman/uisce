package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/analytics"
)

func TestSemanticRelationshipsHandler_Routes(t *testing.T) {
	relService := analytics.NewTermRelationshipService(nil)
	handler := NewSemanticRelationshipsHandler(relService, nil)

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	// 1. Test GET /semantic-terms/{id}/related
	req1 := httptest.NewRequest(http.MethodGet, "/semantic-terms/Account%20Code/related", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("expected status 200 from related terms, got %d", w1.Code)
	}

	var disambig analytics.TermDisambiguation
	if err := json.NewDecoder(w1.Body).Decode(&disambig); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if disambig.PrimaryTerm.TermName != "Account Code" {
		t.Errorf("expected primary term 'Account Code', got '%s'", disambig.PrimaryTerm.TermName)
	}
	if len(disambig.RelatedTerms) == 0 {
		t.Errorf("expected related terms for Account Code, got none")
	}

	// 2. Test GET /semantic-mapper/suggest-related
	req2 := httptest.NewRequest(http.MethodGet, "/semantic-mapper/suggest-related?column=cusip_number&entity=security_master", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected status 200 from suggest-related, got %d", w2.Code)
	}

	var sugResp struct {
		Column      string                      `json:"column"`
		Entity      string                      `json:"entity"`
		Suggestions []analytics.RelatedTermInfo `json:"suggestions"`
	}
	if err := json.NewDecoder(w2.Body).Decode(&sugResp); err != nil {
		t.Fatalf("failed to decode suggest response: %v", err)
	}
	if len(sugResp.Suggestions) == 0 {
		t.Errorf("expected suggestions for cusip_number, got 0")
	}

	// 3. Test POST /semantic-mapper/ai-context
	bodyBytes, _ := json.Marshal(map[string]interface{}{
		"domain": "Capital Markets",
	})
	req3 := httptest.NewRequest(http.MethodPost, "/semantic-mapper/ai-context", bytes.NewReader(bodyBytes))
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)

	if w3.Code != http.StatusOK {
		t.Fatalf("expected status 200 from ai-context, got %d", w3.Code)
	}

	var aiPayload analytics.AIContextPayload
	if err := json.NewDecoder(w3.Body).Decode(&aiPayload); err != nil {
		t.Fatalf("failed to decode AI context payload: %v", err)
	}
	if aiPayload.PromptContextBlock == "" {
		t.Errorf("expected prompt context block, got empty")
	}

	// 4. Test POST /semantic-mapper/rejections
	rejBody, _ := json.Marshal(map[string]interface{}{
		"source_node_id":     "11111111-1111-1111-1111-111111111111",
		"rejected_target_id": "22222222-2222-2222-2222-222222222222",
		"reason":             "Not applicable for trade allocations",
	})
	req4 := httptest.NewRequest(http.MethodPost, "/semantic-mapper/rejections", bytes.NewReader(rejBody))
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)

	if w4.Code != http.StatusOK {
		t.Fatalf("expected status 200 from rejections post, got %d", w4.Code)
	}

	// 5. Test GET /semantic-mapper/rejections
	req5 := httptest.NewRequest(http.MethodGet, "/semantic-mapper/rejections", nil)
	w5 := httptest.NewRecorder()
	r.ServeHTTP(w5, req5)

	if w5.Code != http.StatusOK {
		t.Fatalf("expected status 200 from rejections get, got %d", w5.Code)
	}

	// 6. Test GET /taxonomy/l3-classifications
	req6 := httptest.NewRequest(http.MethodGet, "/taxonomy/l3-classifications", nil)
	w6 := httptest.NewRecorder()
	r.ServeHTTP(w6, req6)

	if w6.Code != http.StatusOK {
		t.Fatalf("expected status 200 from l3-classifications get, got %d", w6.Code)
	}

	var l3Resp struct {
		Data  []analytics.L3ClassificationInfo `json:"data"`
		Count int                              `json:"count"`
	}
	if err := json.NewDecoder(w6.Body).Decode(&l3Resp); err != nil {
		t.Fatalf("failed decoding l3 classifications: %v", err)
	}
	if len(l3Resp.Data) == 0 {
		t.Errorf("expected L3 classifications, got 0")
	}

	// 7. Test GET /taxonomy/suggest-l3
	req7 := httptest.NewRequest(http.MethodGet, "/taxonomy/suggest-l3?term=Allocation%20Account%20Code&column=alloc_code", nil)
	w7 := httptest.NewRecorder()
	r.ServeHTTP(w7, req7)

	if w7.Code != http.StatusOK {
		t.Fatalf("expected status 200 from suggest-l3, got %d", w7.Code)
	}

	var l3Sug analytics.L3ClassificationInfo
	if err := json.NewDecoder(w7.Body).Decode(&l3Sug); err != nil {
		t.Fatalf("failed decoding suggest-l3 response: %v", err)
	}
	if l3Sug.Name != "Trade Allocation" {
		t.Errorf("expected suggest-l3 to return 'Trade Allocation', got '%s'", l3Sug.Name)
	}
}
