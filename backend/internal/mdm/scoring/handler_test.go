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
	costs         map[string]float64
	domainCosts   map[string]map[string]float64
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

func (m *mockScoringRepository) GetVendorCosts(ctx context.Context) (map[string]float64, error) {
	if m.costs == nil {
		return map[string]float64{"BBG": 2140000, "RFT": 1180000}, nil
	}
	return m.costs, nil
}

func (m *mockScoringRepository) GetVendorDomainCosts(ctx context.Context) (map[string]map[string]float64, error) {
	if m.domainCosts == nil {
		return map[string]map[string]float64{"BBG": {"pricing": 900000}}, nil
	}
	return m.domainCosts, nil
}

func (m *mockScoringRepository) SetVendorCost(ctx context.Context, vendorID string, cost float64, entityDomain string) error {
	if m.costs == nil {
		m.costs = make(map[string]float64)
	}
	m.costs[vendorID] = cost
	return nil
}

func (m *mockScoringRepository) GetScoringSettings(ctx context.Context) (*ScoringSettings, error) {
	return &ScoringSettings{
		HourlyLaborRate: 150.00,
		StabilityDecayK: 50.00,
		FrictionBudget:  100000.00,
		ColdStartDays:   30,
	}, nil
}

func (m *mockScoringRepository) GetActiveWeightProfile(ctx context.Context) (*WeightProfile, error) {
	return &WeightProfile{
		ProfileID:   1,
		ProfileName: "Balanced Institutional Standard",
		IsActive:    true,
		WeightSuff:  0.300,
		WeightCov:   0.200,
		WeightSLA:   0.150,
		WeightStab:  0.150,
		WeightOER:   0.100,
		WeightLic:   0.100,
	}, nil
}

func (m *mockScoringRepository) GetWeightProfiles(ctx context.Context) ([]WeightProfile, error) {
	p, _ := m.GetActiveWeightProfile(ctx)
	return []WeightProfile{*p}, nil
}

func (m *mockScoringRepository) SaveWeightProfile(ctx context.Context, profile WeightProfile) (*WeightProfile, error) {
	profile.ProfileID = 123
	return &profile, nil
}

func (m *mockScoringRepository) GetVendorFeedLogs(ctx context.Context, asOf time.Time, windowDays int) ([]VendorFeedLog, error) {
	return nil, nil
}

func (m *mockScoringRepository) GetVendorRevisions(ctx context.Context, asOf time.Time, windowDays int) ([]VendorRevisionLog, error) {
	return nil, nil
}

func (m *mockScoringRepository) GetVendorFrictions(ctx context.Context, asOf time.Time) ([]VendorOperationalFriction, error) {
	return nil, nil
}

func (m *mockScoringRepository) GetVendorContractRights(ctx context.Context) ([]VendorContractRights, error) {
	return nil, nil
}

func (m *mockScoringRepository) LogShadowRun(ctx context.Context, entry ShadowRunLogEntry) error {
	return nil
}

func (m *mockScoringRepository) GetShadowRuns(ctx context.Context, tenantID string, limit int) ([]ShadowRunLogEntry, error) {
	return []ShadowRunLogEntry{
		{
			RunID:              1,
			TenantID:           tenantID,
			ExecutedAt:         time.Now(),
			CandidateVendorIDs: []string{"BBG", "RFT"},
			UniverseSize:       42000,
			Status:             "COMPLETED",
		},
	}, nil
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

func TestHandler_HandleRunIngest(t *testing.T) {
	now := time.Now()
	mockRepo := &mockScoringRepository{
		tolerances: DefaultTolerances(),
		goldenRecords: []GoldenRecord{
			{EntityID: 1, AttributeCode: "LEI", GoldenValue: "ABC", WinningVendorID: "BBG"},
		},
		candidates: []VendorCandidate{
			{EntityID: 1, AttributeCode: "LEI", VendorID: "BBG", NormalizedValue: "ABC", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},
		},
		overrides: []ValueOverrideRecord{},
	}

	service := NewService(mockRepo)
	handler := NewHandler(service)

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	payload := `{"universe_size":10000,"pipeline_id":"a11c0001-0001-4000-8000-000000000099"}`
	req := httptest.NewRequest("POST", "/api/mdm/scoring/run-ingest", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("handler returned status %d; want %d: %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var res map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res["status"] != "COMPLETED" {
		t.Errorf("expected status COMPLETED, got %v", res["status"])
	}
	if res["records_ingested"] != float64(10000) {
		t.Errorf("expected 10000 records ingested, got %v", res["records_ingested"])
	}
}

func TestHandlerUpdateSpend(t *testing.T) {
	mockRepo := &mockScoringRepository{
		costs: map[string]float64{"BBG": 2140000},
	}
	service := NewService(mockRepo)
	handler := NewHandler(service)

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	payload := `{"vendor_id":"BBG","annual_cost":2500000,"entity_domain":"pricing"}`
	req := httptest.NewRequest("POST", "/api/mdm/scoring/spend", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("handler returned status %d; want %d: %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var res map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res["status"] != "SUCCESS" {
		t.Errorf("expected status SUCCESS, got %v", res["status"])
	}
	if res["annual_cost"] != float64(2500000) {
		t.Errorf("expected annual_cost 2500000, got %v", res["annual_cost"])
	}
	if mockRepo.costs["BBG"] != 2500000 {
		t.Errorf("expected mockRepo.costs[BBG] to be 2500000, got %v", mockRepo.costs["BBG"])
	}
}

func TestHandlerGetDimensions(t *testing.T) {
	mockRepo := &mockScoringRepository{
		tolerances: map[string]AttributeTolerance{
			"LEI": {AttributeCode: "LEI", Tier: 1, MatchType: MatchExact, TierWeight: 0.5},
		},
		goldenRecords: []GoldenRecord{
			{EntityID: 1, AttributeCode: "LEI", GoldenValue: "ABC", WinningVendorID: "BBG"},
		},
		candidates: []VendorCandidate{
			{EntityID: 1, AttributeCode: "LEI", VendorID: "BBG", NormalizedValue: "ABC", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: time.Now()},
		},
	}
	service := NewService(mockRepo)
	handler := NewHandler(service)

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/api/mdm/scoring/dimensions", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("handler returned status %d; want %d: %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var profiles []VendorDimensionProfile
	if err := json.NewDecoder(rr.Body).Decode(&profiles); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(profiles) == 0 {
		t.Errorf("expected dimension profiles to be non-empty")
	}
}

func TestHandlerOptimizeBundle(t *testing.T) {
	now := time.Now()
	mockRepo := &mockScoringRepository{
		tolerances: map[string]AttributeTolerance{
			"ISIN": {AttributeCode: "ISIN", Tier: 1, MatchType: MatchExact, TierWeight: 0.5},
		},
		candidates: []VendorCandidate{
			{EntityID: 1, AttributeCode: "ISIN", VendorID: "BBG", NormalizedValue: "VAL1", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},
			{EntityID: 2, AttributeCode: "ISIN", VendorID: "RFT", NormalizedValue: "VAL2", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},
		},
		costs: map[string]float64{
			"BBG": 2140000,
			"RFT": 1180000,
		},
	}
	service := NewService(mockRepo)
	handler := NewHandler(service)

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	payload := `{"target_t1_coverage":0.50,"universe_size":2}`
	req := httptest.NewRequest("POST", "/api/mdm/scoring/optimize-bundle", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("handler returned status %d; want %d: %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var res OptimalBundleResult
	if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(res.SelectedVendors) == 0 {
		t.Errorf("expected selected vendors to be non-empty")
	}
	if res.SolverStrategy != "BITMASK_BRANCH_AND_BOUND" {
		t.Errorf("expected BITMASK_BRANCH_AND_BOUND, got %s", res.SolverStrategy)
	}
}

func TestHandlerWeightProfiles(t *testing.T) {
	mockRepo := &mockScoringRepository{}
	service := NewService(mockRepo)
	handler := NewHandler(service)

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	// GET profiles
	req := httptest.NewRequest("GET", "/api/mdm/scoring/weight-profiles", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("GET /weight-profiles returned %d", rr.Code)
	}

	// POST valid profile
	validPayload := `{"profile_name":"Custom Test Profile","is_active":true,"weight_suff":0.40,"weight_cov":0.20,"weight_sla":0.10,"weight_stab":0.10,"weight_oer":0.10,"weight_lic":0.10}`
	reqPost := httptest.NewRequest("POST", "/api/mdm/scoring/weight-profiles", strings.NewReader(validPayload))
	reqPost.Header.Set("Content-Type", "application/json")
	rrPost := httptest.NewRecorder()
	r.ServeHTTP(rrPost, reqPost)

	if rrPost.Code != http.StatusOK {
		t.Fatalf("POST valid profile returned %d: %s", rrPost.Code, rrPost.Body.String())
	}

	// POST invalid profile (sum != 1.0)
	invalidPayload := `{"profile_name":"Invalid Profile","weight_suff":0.50,"weight_cov":0.50,"weight_sla":0.50,"weight_stab":0.10,"weight_oer":0.10,"weight_lic":0.10}`
	reqInvalid := httptest.NewRequest("POST", "/api/mdm/scoring/weight-profiles", strings.NewReader(invalidPayload))
	reqInvalid.Header.Set("Content-Type", "application/json")
	rrInvalid := httptest.NewRecorder()
	r.ServeHTTP(rrInvalid, reqInvalid)

	if rrInvalid.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for invalid weight sum, got %d", rrInvalid.Code)
	}
}

func TestHandlerMultiDisplacement(t *testing.T) {
	mockRepo := &mockScoringRepository{
		tolerances: map[string]AttributeTolerance{
			"ISIN": {AttributeCode: "ISIN", Tier: 1, MatchType: MatchExact, TierWeight: 0.50},
		},
		candidates: []VendorCandidate{
			{EntityID: 1, AttributeCode: "ISIN", VendorID: "BBG", NormalizedValue: "US123", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: time.Now()},
			{EntityID: 1, AttributeCode: "ISIN", VendorID: "RFT", NormalizedValue: "US123", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: time.Now()},
		},
		goldenRecords: []GoldenRecord{
			{EntityID: 1, AttributeCode: "ISIN", GoldenValue: "US123", WinningVendorID: "BBG", RuleApplied: "SOURCE_PRIORITY"},
		},
	}
	service := NewService(mockRepo)
	handler := NewHandler(service)

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	// Valid POST
	payload := `{"dropped_vendor_ids":["BBG"],"replacement_vendor_ids":["ICE"]}`
	req := httptest.NewRequest("POST", "/api/mdm/scoring/displacement-multi", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("POST /displacement-multi returned %d: %s", rr.Code, rr.Body.String())
	}

	var res MultiVendorDisplacementResult
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(res.DroppedVendorIDs) != 1 || res.DroppedVendorIDs[0] != "BBG" {
		t.Errorf("expected dropped vendor BBG, got %v", res.DroppedVendorIDs)
	}

	// Invalid POST with no dropped vendors
	badPayload := `{"dropped_vendor_ids":[],"replacement_vendor_ids":["ICE"]}`
	badReq := httptest.NewRequest("POST", "/api/mdm/scoring/displacement-multi", strings.NewReader(badPayload))
	badReq.Header.Set("Content-Type", "application/json")
	badRr := httptest.NewRecorder()
	r.ServeHTTP(badRr, badReq)

	if badRr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for empty dropped_vendor_ids, got %d", badRr.Code)
	}
}

func TestHandlerGetShadowValidation(t *testing.T) {
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
		},
		costs: map[string]float64{
			"BBG": 2140000,
			"RFT": 1180000,
		},
	}

	service := NewService(mockRepo)
	handler := NewHandler(service)

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/api/mdm/scoring/shadow-eval", nil)
	req.Header.Set("X-Tenant-ID", "tenant-test")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("GET /shadow-eval returned %d: %s", rr.Code, rr.Body.String())
	}

	var report ShadowValidationReport
	if err := json.Unmarshal(rr.Body.Bytes(), &report); err != nil {
		t.Fatalf("failed to decode shadow-eval response: %v", err)
	}

	if report.TenantID != "tenant-test" {
		t.Errorf("expected tenant-test, got %s", report.TenantID)
	}
	if len(report.Comparisons) == 0 {
		t.Errorf("expected comparisons to be populated")
	}
}

func TestHandler_HandleGetTrends(t *testing.T) {
	mockRepo := &mockScoringRepository{
		tolerances: DefaultTolerances(),
	}
	service := NewService(mockRepo)
	handler := NewHandler(service)

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	t.Run("DefaultParameters", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/mdm/scoring/trends?dimension=COMPOSITE", nil)
		req.Header.Set("X-Tenant-ID", "tenant-test")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("GET /trends returned %d: %s", rr.Code, rr.Body.String())
		}

		var report TrendAnalysisReport
		if err := json.Unmarshal(rr.Body.Bytes(), &report); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if report.Dimension != "COMPOSITE" {
			t.Errorf("expected dimension COMPOSITE, got %s", report.Dimension)
		}
		if len(report.Series) == 0 {
			t.Fatalf("expected non-empty series")
		}
	})

	t.Run("InvalidDimensionRejected", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/mdm/scoring/trends?dimension=SQL_INJECTION", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400 Bad Request, got %d", rr.Code)
		}
	})
}

func TestHandler_HandleSimulateProfiles(t *testing.T) {
	now := time.Now()
	mockRepo := &mockScoringRepository{
		tolerances: DefaultTolerances(),
		candidates: []VendorCandidate{
			{EntityID: 1, AttributeCode: "LEI", VendorID: "BBG", NormalizedValue: "V1", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},
			{EntityID: 1, AttributeCode: "LEI", VendorID: "RFT", NormalizedValue: "V1", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},
		},
		goldenRecords: []GoldenRecord{
			{EntityID: 1, AttributeCode: "LEI", GoldenValue: "V1", WinningVendorID: "BBG", RuleApplied: "SOURCE_PRIORITY"},
		},
		costs: map[string]float64{
			"BBG": 2140000,
			"RFT": 1180000,
		},
	}
	service := NewService(mockRepo)
	handler := NewHandler(service)

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	t.Run("ValidProfilesSimulation", func(t *testing.T) {
		payload := `{
			"profile_a": {
				"name": "Quality Focus",
				"weights": {
					"weight_sufficiency": 0.30,
					"weight_coverage": 0.30,
					"weight_sla": 0.15,
					"weight_stability": 0.10,
					"weight_friction": 0.05,
					"weight_licensing": 0.10
				}
			},
			"profile_b": {
				"name": "Operations Focus",
				"weights": {
					"weight_sufficiency": 0.10,
					"weight_coverage": 0.10,
					"weight_sla": 0.35,
					"weight_stability": 0.25,
					"weight_friction": 0.10,
					"weight_licensing": 0.10
				}
			}
		}`

		req := httptest.NewRequest("POST", "/api/mdm/scoring/simulate-profiles", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tenant-ID", "tenant-test")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("POST /simulate-profiles returned %d: %s", rr.Code, rr.Body.String())
		}

		var res ProfileSimulationResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if res.ProfileA.ProfileName != "Quality Focus" {
			t.Errorf("expected Profile A name 'Quality Focus', got '%s'", res.ProfileA.ProfileName)
		}
		if res.ProfileB.ProfileName != "Operations Focus" {
			t.Errorf("expected Profile B name 'Operations Focus', got '%s'", res.ProfileB.ProfileName)
		}
		if len(res.RankShifts) == 0 {
			t.Errorf("expected non-empty rank shifts")
		}
		if res.BundleImpact.Insight == "" {
			t.Errorf("expected bundle impact insight")
		}
	})

	t.Run("InvalidWeightsProfileA", func(t *testing.T) {
		payload := `{
			"profile_a": {
				"name": "Bad Sum",
				"weights": {
					"weight_sufficiency": 0.20,
					"weight_coverage": 0.20,
					"weight_sla": 0.20,
					"weight_stability": 0.20,
					"weight_friction": 0.10,
					"weight_licensing": 0.00
				}
			},
			"profile_b": {
				"name": "Valid",
				"weights": {
					"weight_sufficiency": 0.20,
					"weight_coverage": 0.20,
					"weight_sla": 0.20,
					"weight_stability": 0.20,
					"weight_friction": 0.10,
					"weight_licensing": 0.10
				}
			}
		}`

		req := httptest.NewRequest("POST", "/api/mdm/scoring/simulate-profiles", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400 Bad Request, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("InvalidWeightsProfileB", func(t *testing.T) {
		payload := `{
			"profile_a": {
				"name": "Valid",
				"weights": {
					"weight_sufficiency": 0.20,
					"weight_coverage": 0.20,
					"weight_sla": 0.20,
					"weight_stability": 0.20,
					"weight_friction": 0.10,
					"weight_licensing": 0.10
				}
			},
			"profile_b": {
				"name": "Bad Sum",
				"weights": {
					"weight_sufficiency": 0.50,
					"weight_coverage": 0.50,
					"weight_sla": 0.50,
					"weight_stability": 0.00,
					"weight_friction": 0.00,
					"weight_licensing": 0.00
				}
			}
		}`

		req := httptest.NewRequest("POST", "/api/mdm/scoring/simulate-profiles", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400 Bad Request, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}






