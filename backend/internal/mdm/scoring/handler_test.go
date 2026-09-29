package scoring

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

type mockScoringRepository struct {
	tolerances    map[string]AttributeTolerance
	goldenRecords []GoldenRecord
	candidates    []VendorCandidate
	overrides     []ValueOverrideRecord
}

func (m *mockScoringRepository) GetAttributeTolerances(ctx context.Context) (map[string]AttributeTolerance, error) {
	return m.tolerances, nil
}

func (m *mockScoringRepository) GetGoldenRecords(ctx context.Context, asOf time.Time) ([]GoldenRecord, error) {
	return m.goldenRecords, nil
}

func (m *mockScoringRepository) GetVendorCandidates(ctx context.Context, asOf time.Time) ([]VendorCandidate, error) {
	return m.candidates, nil
}

func (m *mockScoringRepository) GetValueOverrides(ctx context.Context, from time.Time) ([]ValueOverrideRecord, error) {
	return m.overrides, nil
}

func TestHandlerGetScorecard(t *testing.T) {
	now := time.Now()
	mockRepo := &mockScoringRepository{
		tolerances: map[string]AttributeTolerance{
			"LEI": {AttributeCode: "LEI", Tier: 1, MatchType: MatchExact, TierWeight: 0.5},
		},
		goldenRecords: []GoldenRecord{
			{EntityID: 1, AttributeCode: "LEI", GoldenValue: "ABC", WinningVendorID: "BBG"},
		},
		candidates: []VendorCandidate{
			{EntityID: 1, AttributeCode: "LEI", VendorID: "BBG", NormalizedValue: "ABC", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},
			{EntityID: 1, AttributeCode: "LEI", VendorID: "RFT", NormalizedValue: "ABC", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},
		},
		overrides: []ValueOverrideRecord{},
	}

	service := NewService(mockRepo)
	handler := NewHandler(service)

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/api/mdm/scoring/scorecard", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("handler returned status %d; want %d", rr.Code, http.StatusOK)
	}

	var report VendorScorecardReport
	if err := json.NewDecoder(rr.Body).Decode(&report); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(report.SubstitutionMatrix) == 0 {
		t.Errorf("expected non-empty substitution matrix in scorecard report")
	}
}

func TestHandlerSimulateDisplacement(t *testing.T) {
	now := time.Now()
	mockRepo := &mockScoringRepository{
		tolerances: map[string]AttributeTolerance{
			"LEI": {AttributeCode: "LEI", Tier: 1, MatchType: MatchExact, TierWeight: 0.5},
		},
		goldenRecords: []GoldenRecord{
			{EntityID: 1, AttributeCode: "LEI", GoldenValue: "ABC", WinningVendorID: "BBG"},
		},
		candidates: []VendorCandidate{
			{EntityID: 1, AttributeCode: "LEI", VendorID: "BBG", NormalizedValue: "ABC", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},
			{EntityID: 1, AttributeCode: "LEI", VendorID: "RFT", NormalizedValue: "ABC", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},
		},
	}

	service := NewService(mockRepo)
	handler := NewHandler(service)

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	payload := `{"dropped_vendor_id":"BBG","dropped_vendor_name":"Bloomberg","annual_cost":2140000,"vendor_hierarchy":["BBG","RFT"]}`
	req := httptest.NewRequest("POST", "/api/mdm/scoring/displacement", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("handler returned status %d; want %d", rr.Code, http.StatusOK)
	}

	var result VendorDisplacementResult
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.DroppedVendorID != "BBG" {
		t.Errorf("dropped vendor = %s; want BBG", result.DroppedVendorID)
	}
}
