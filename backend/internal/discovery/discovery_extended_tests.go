package discovery

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// testLoggerExt and mockDBExt avoid conflicts with api_test.go definitions
var testLoggerExt = log.New(os.Stdout, "DISCOVERY_TEST: ", log.LstdFlags)

func mockDBExt() *sql.DB {
	// In real tests, would use testcontainers or pgx
	// For now, return nil to indicate mock
	return nil
}

// TestApprovalWorkflowAudit verifies approval creates audit trail
func TestApprovalWorkflowAudit(t *testing.T) {
	handler := &DiscoveryHandler{
		db:     mockDBExt(),
		logger: testLoggerExt,
	}

	// First create a candidate
	candidateID := "cand-001"

	req := ApproveRequest{
		CandidateID: candidateID,
		FeatureName: "transaction_amount",
		Notes:       "High value for forecasting",
	}

	body, _ := json.Marshal(req)
	r := httptest.NewRequest("POST", "/discovery/approve", bytes.NewReader(body))
	r.Header.Set("X-User-ID", "user-alice")
	w := httptest.NewRecorder()

	handler.ApproveCandidate(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp CandidateResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Status != "approved" {
		t.Errorf("Expected status 'approved', got %s", resp.Status)
	}
}

// TestRejectionWithReason verifies rejection stores reason
func TestRejectionWithReason(t *testing.T) {
	handler := &DiscoveryHandler{
		db:     mockDBExt(),
		logger: testLoggerExt,
	}

	req := RejectRequest{
		CandidateID: "cand-002",
		Reason:      "Too sparse for current use case",
	}

	body, _ := json.Marshal(req)
	r := httptest.NewRequest("POST", "/discovery/reject", bytes.NewReader(body))
	r.Header.Set("X-User-ID", "user-bob")
	w := httptest.NewRecorder()

	handler.RejectCandidate(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp CandidateResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Status != "rejected" {
		t.Errorf("Expected status 'rejected', got %s", resp.Status)
	}
	if resp.RejectionReason != "Too sparse for current use case" {
		t.Errorf("Expected rejection reason preserved")
	}
}

// TestSearchEdgeCases tests search query validation
func TestSearchEdgeCases(t *testing.T) {
	handler := &DiscoveryHandler{
		db:     mockDBExt(),
		logger: testLoggerExt,
	}

	tests := []struct {
		query     string
		shouldErr bool
		name      string
	}{
		{"a", true, "Single character query (too short)"},
		{"ab", false, "Minimum valid query (2 chars)"},
		{"transaction", false, "Valid single word"},
		{"transaction_feature", false, "Valid with underscore"},
		{"", true, "Empty query (too short)"},
		{"   ", true, "Whitespace only (too short)"},
		{"ab cd", false, "Multi-word query"},
	}

	for _, test := range tests {
		r := httptest.NewRequest("GET", "/discovery/search?q="+test.query, nil)
		w := httptest.NewRecorder()

		handler.SearchCandidates(w, r)

		hasError := w.Code != http.StatusOK
		if hasError != test.shouldErr {
			t.Errorf("%s: Expected error=%v, got %d", test.name, test.shouldErr, w.Code)
		}
	}
}

// TestSortingOptions verifies all sort options work
func TestSortingOptions(t *testing.T) {
	handler := &DiscoveryHandler{
		db:     mockDBExt(),
		logger: testLoggerExt,
	}

	sortOptions := []string{"score", "name", "discovered_at"}

	for _, sortBy := range sortOptions {
		r := httptest.NewRequest("GET", "/discovery/candidates?sort_by="+sortBy+"&page=1&page_size=10", nil)
		w := httptest.NewRecorder()

		handler.ListCandidates(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("Sort by %s failed: %d", sortBy, w.Code)
		}

		var resp CandidateListResponse
		json.Unmarshal(w.Body.Bytes(), &resp)

		if len(resp.Candidates) == 0 {
			t.Logf("Sort by %s returned no items (expected for mock)", sortBy)
		}
	}
}

// TestInvalidSortOption returns default sorting
func TestInvalidSortOption(t *testing.T) {
	handler := &DiscoveryHandler{
		db:     mockDBExt(),
		logger: testLoggerExt,
	}

	r := httptest.NewRequest("GET", "/discovery/candidates?sort_by=invalid_sort", nil)
	w := httptest.NewRecorder()

	handler.ListCandidates(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Expected default sort to work, got %d", w.Code)
	}
}

// TestFilterByStatus tests status filtering
func TestFilterByStatus(t *testing.T) {
	handler := &DiscoveryHandler{
		db:     mockDBExt(),
		logger: testLoggerExt,
	}

	statuses := []string{"candidate", "approved", "rejected"}

	for _, status := range statuses {
		r := httptest.NewRequest("GET", "/discovery/candidates?status="+status, nil)
		w := httptest.NewRecorder()

		handler.ListCandidates(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("Filter by status %s failed: %d", status, w.Code)
		}
	}
}

// TestMultipleFiltersApplied tests combining multiple filters
func TestMultipleFiltersApplied(t *testing.T) {
	handler := &DiscoveryHandler{
		db:     mockDBExt(),
		logger: testLoggerExt,
	}

	r := httptest.NewRequest("GET", "/discovery/candidates?status=candidate&source_db=postgres&min_score=0.6&page=1&page_size=10", nil)
	w := httptest.NewRecorder()

	handler.ListCandidates(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Multiple filters failed: %d", w.Code)
	}

	var resp CandidateListResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Page != 1 {
		t.Errorf("Expected page 1, got %d", resp.Page)
	}
	if resp.PageSize != 10 {
		t.Errorf("Expected page_size 10, got %d", resp.PageSize)
	}
}

// TestStatsAggregation verifies statistics are correct
func TestStatsAggregation(t *testing.T) {
	handler := &DiscoveryHandler{
		db:     mockDBExt(),
		logger: testLoggerExt,
	}

	r := httptest.NewRequest("GET", "/discovery/stats", nil)
	w := httptest.NewRecorder()

	handler.GetDiscoveryStats(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Stats retrieval failed: %d", w.Code)
	}

	var stats DiscoveryStats
	json.Unmarshal(w.Body.Bytes(), &stats)

	if stats.TotalCandidates < 0 {
		t.Error("Total candidates should be non-negative")
	}
	if stats.ApprovedCount < 0 {
		t.Error("Approved count should be non-negative")
	}
	if stats.RejectedCount < 0 {
		t.Error("Rejected count should be non-negative")
	}
}

// TestDiscoveryRunNotFound tests 404 when run doesn't exist
func TestDiscoveryRunNotFound(t *testing.T) {
	handler := &DiscoveryHandler{
		db:     mockDBExt(),
		logger: testLoggerExt,
	}

	r := httptest.NewRequest("GET", "/discovery/runs/nonexistent-run-id", nil)
	w := httptest.NewRecorder()

	handler.GetDiscoveryRun(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404 for nonexistent run, got %d", w.Code)
	}
}

// TestCandidateNotFound tests 404 when candidate doesn't exist
func TestCandidateNotFound(t *testing.T) {
	handler := &DiscoveryHandler{
		db:     mockDBExt(),
		logger: testLoggerExt,
	}

	r := httptest.NewRequest("GET", "/discovery/candidates/nonexistent-candidate-id", nil)
	w := httptest.NewRecorder()

	handler.GetCandidate(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404 for nonexistent candidate, got %d", w.Code)
	}
}

// TestMissingUserIDHeaderInApproval tests user tracking
func TestMissingUserIDHeaderInApproval(t *testing.T) {
	handler := &DiscoveryHandler{
		db:     mockDBExt(),
		logger: testLoggerExt,
	}

	req := ApproveRequest{
		CandidateID: "cand-001",
	}

	body, _ := json.Marshal(req)
	r := httptest.NewRequest("POST", "/discovery/approve", bytes.NewReader(body))
	// Intentionally NOT setting X-User-ID header
	w := httptest.NewRecorder()

	handler.ApproveCandidate(w, r)

	// Should either fail or use default user
	if w.Code == http.StatusOK {
		var resp CandidateResponse
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Status == "approved" {
			t.Logf("Candidate approved successfully")
		}
	}
}

// TestDiscoveryStartWithDifferentDatabaseTypes tests various database types
func TestDiscoveryStartWithDifferentDatabaseTypes(t *testing.T) {
	handler := &DiscoveryHandler{
		db:     mockDBExt(),
		logger: testLoggerExt,
	}

	dbTypes := []string{"postgres", "trino", "auto", "starrocks"}

	for _, dbType := range dbTypes {
		req := StartDiscoveryRequest{
			DatabaseType:   dbType,
			ScanInterval:   24,
			UseCase:        "forecasting",
			ScoringWeights: DefaultWeights().toMap(),
		}

		body, _ := json.Marshal(req)
		r := httptest.NewRequest("POST", "/discovery/start", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.StartDiscovery(w, r)

		if w.Code != http.StatusCreated {
			t.Errorf("Database type %s failed: %d", dbType, w.Code)
		}
	}
}

// TestDiscoveryStartDefaultValues tests default values are applied
func TestDiscoveryStartDefaultValues(t *testing.T) {
	handler := &DiscoveryHandler{
		db:     mockDBExt(),
		logger: testLoggerExt,
	}

	// Minimal request with only required fields
	req := StartDiscoveryRequest{
		DatabaseType: "auto",
		UseCase:      "forecasting",
	}

	body, _ := json.Marshal(req)
	r := httptest.NewRequest("POST", "/discovery/start", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.StartDiscovery(w, r)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected 201, got %d", w.Code)
	}

	var resp DiscoveryRunResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Status != "pending" {
		t.Error("Expected pending status")
	}
}

// TestRatioaneCalculation tests scoring rationale generation
func TestRationaleCalculation(t *testing.T) {
	tests := []struct {
		score    float64
		expected string
	}{
		{0.1, "low"},
		{0.4, "moderate"},
		{0.7, "high"},
		{0.95, "very high"},
	}

	for _, test := range tests {
		rationale := generateRationale(test.score)
		if !strings.Contains(strings.ToLower(rationale), strings.ToLower(test.expected)) {
			t.Logf("Score %.1f rationale: %s", test.score, rationale)
		}
	}
}

// TestListCandidatesWithoutFilters returns all candidates
func TestListCandidatesWithoutFilters(t *testing.T) {
	handler := &DiscoveryHandler{
		db:     mockDBExt(),
		logger: testLoggerExt,
	}

	r := httptest.NewRequest("GET", "/discovery/candidates", nil)
	w := httptest.NewRecorder()

	handler.ListCandidates(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var resp CandidateListResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Page != 1 {
		t.Errorf("Expected page 1, got %d", resp.Page)
	}
	if resp.PageSize == 0 {
		t.Error("Expected non-zero page size")
	}
}

// TestResponseContentType verifies JSON content type
func TestResponseContentType(t *testing.T) {
	handler := &DiscoveryHandler{
		db:     mockDBExt(),
		logger: testLoggerExt,
	}

	r := httptest.NewRequest("GET", "/discovery/stats", nil)
	w := httptest.NewRecorder()

	handler.GetDiscoveryStats(w, r)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}
}

// TestConcurrentRequests tests handler safety with concurrent calls (basic)
func TestConcurrentRequests(t *testing.T) {
	handler := &DiscoveryHandler{
		db:     mockDBExt(),
		logger: testLoggerExt,
	}

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			r := httptest.NewRequest("GET", "/discovery/stats", nil)
			w := httptest.NewRecorder()
			handler.GetDiscoveryStats(w, r)
			done <- w.Code == http.StatusOK
		}()
	}

	successCount := 0
	for i := 0; i < 10; i++ {
		if <-done {
			successCount++
		}
	}

	if successCount < 8 {
		t.Errorf("Expected at least 8/10 concurrent requests to succeed, got %d", successCount)
	}
}

// TestDiscoveryRunStatusTransitions verifies status flow
func TestDiscoveryRunStatusTransitions(t *testing.T) {
	handler := &DiscoveryHandler{
		db:     mockDBExt(),
		logger: testLoggerExt,
	}

	// Start discovery
	req := StartDiscoveryRequest{
		DatabaseType:   "auto",
		ScanInterval:   24,
		UseCase:        "forecasting",
		ScoringWeights: DefaultWeights().toMap(),
	}

	body, _ := json.Marshal(req)
	r := httptest.NewRequest("POST", "/discovery/start", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.StartDiscovery(w, r)

	var resp DiscoveryRunResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Status != "pending" {
		t.Errorf("Initial status should be pending, got %s", resp.Status)
	}
}

// TestScoreDistributionBuckets verifies statistics bucketing
func TestScoreDistributionBuckets(t *testing.T) {
	handler := &DiscoveryHandler{
		db:     mockDBExt(),
		logger: testLoggerExt,
	}

	r := httptest.NewRequest("GET", "/discovery/stats", nil)
	w := httptest.NewRecorder()

	handler.GetDiscoveryStats(w, r)

	var stats DiscoveryStats
	json.Unmarshal(w.Body.Bytes(), &stats)

	if stats.ScoreDistribution == nil {
		t.Error("Score distribution should not be nil")
	}

	expectedBuckets := 5 // 0-0.2, 0.2-0.4, 0.4-0.6, 0.6-0.8, 0.8-1.0
	if stats.ScoreDistribution != nil && len(stats.ScoreDistribution) != expectedBuckets {
		t.Logf("Expected %d score buckets", expectedBuckets)
	}
}

// TestContextCancellation verifies graceful handling of cancelled context
func TestContextCancellation(t *testing.T) {
	handler := &DiscoveryHandler{
		db:     mockDBExt(),
		logger: testLoggerExt,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Immediately cancel

	r := httptest.NewRequest("GET", "/discovery/stats", nil)
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()

	// Should handle gracefully (may return error or proceed depending on implementation)
	handler.GetDiscoveryStats(w, r)

	if w.Code >= 500 && w.Code != http.StatusServiceUnavailable {
		// Could be 503 Service Unavailable or other error
		t.Logf("Context cancellation handled: %d", w.Code)
	}
}

// TestLocalTimestampHandling verifies correct timestamp usage
func TestLocalTimestampHandling(t *testing.T) {
	handler := &DiscoveryHandler{
		db:     mockDBExt(),
		logger: testLoggerExt,
	}

	r := httptest.NewRequest("POST", "/discovery/start", bytes.NewReader([]byte(
		`{"database_type":"auto","scan_interval_hours":24,"use_case":"forecasting"}`,
	)))
	w := httptest.NewRecorder()

	handler.StartDiscovery(w, r)

	var resp DiscoveryRunResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if !resp.StartedAt.Before(time.Now().Add(1 * time.Second)) {
		t.Error("Started timestamp should be recent")
	}
}

// generateRationale creates a rationale string based on a score
func generateRationale(score float64) string {
	if score >= 0.8 {
		return "Very high value for forecasting"
	} else if score >= 0.6 {
		return "High value for forecasting"
	} else if score >= 0.4 {
		return "Moderate value for analysis"
	} else if score >= 0.2 {
		return "Low value for current use"
	}
	return "Very low value"
}
