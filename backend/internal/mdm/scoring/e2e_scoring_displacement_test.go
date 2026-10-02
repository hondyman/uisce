package scoring

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

// TestE2E_MultiVendorDisplacement_Workflow validates the full end-to-end multi-vendor
// displacement lifecycle: TCO recalculation, residual gap isolation, sub-license pricing,
// safety cap enforcement (60% ceiling), and negotiation leverage computation.
func TestE2E_MultiVendorDisplacement_Workflow(t *testing.T) {
	ctx := context.Background()
	costs := map[string]float64{
		"BBG": 2140000,
		"RFT": 1180000,
		"FDS": 720000,
		"ICE": 540000,
		"SPG": 610000,
	}

	repo := &mockScoringRepository{
		tolerances: DefaultTolerances(),
		costs:      costs,
	}
	service := NewService(repo)

	// Dropping BBG ($2.14M) and RFT ($1.18M), replacing with ICE ($540k) and FDS ($720k)
	req := MultiVendorDisplacementRequest{
		DroppedVendorIDs:     []string{"BBG", "RFT"},
		ReplacementVendorIDs: []string{"ICE", "FDS"},
		TargetT1Coverage:     0.995,
		TargetT2Coverage:     0.950,
		TargetT3Coverage:     0.850,
		UniverseSize:         42000,
	}

	result, err := service.EvaluateMultiDisplacement(ctx, req)
	if err != nil {
		t.Fatalf("EvaluateMultiDisplacement failed: %v", err)
	}

	// 1. TCO & Savings Verification
	expectedLicenseSavings := costs["BBG"] + costs["RFT"] // $3,320,000
	expectedReplacementCost := costs["ICE"] + costs["FDS"] // $1,260,000
	expectedNetSavings := expectedLicenseSavings - expectedReplacementCost // $2,060,000

	if math.Abs(result.CombinedTCO.LicenseSavingsTotal-expectedLicenseSavings) > 0.01 {
		t.Errorf("LicenseSavingsTotal = %f, want %f", result.CombinedTCO.LicenseSavingsTotal, expectedLicenseSavings)
	}
	if math.Abs(result.CombinedTCO.ReplacementCostDelta-expectedReplacementCost) > 0.01 {
		t.Errorf("ReplacementCostDelta = %f, want %f", result.CombinedTCO.ReplacementCostDelta, expectedReplacementCost)
	}
	// Net Annual TCO benefit accounts for gross savings ($2.06M) minus operational friction & remediation drag
	if result.CombinedTCO.NetAnnualTCOBenefit <= 0 {
		t.Errorf("expected positive NetAnnualTCOBenefit, got %f", result.CombinedTCO.NetAnnualTCOBenefit)
	}
	if result.CombinedTCO.NetAnnualTCOBenefit > expectedNetSavings {
		t.Errorf("NetAnnualTCOBenefit (%f) cannot exceed gross net savings (%f)", result.CombinedTCO.NetAnnualTCOBenefit, expectedNetSavings)
	}

	// 2. Residual Gaps & Sub-license Proposals
	sublicenseCost := result.GapReport.EstimatedGapRemediation
	leverage := result.GapReport.NetNegotiationLeverage

	// Sublicense cost must be substantially smaller than dropped license spend
	if sublicenseCost >= expectedLicenseSavings {
		t.Errorf("sublicenseCost (%f) must be less than dropped savings (%f)", sublicenseCost, expectedLicenseSavings)
	}

	// Leverage must be positive and strong (> 90%)
	if leverage < 0.90 || leverage > 1.00 {
		t.Errorf("expected negotiation leverage between 0.90 and 1.00, got %f", leverage)
	}

	// Verify no single sub-license proposal exceeds 60% of dropped vendor spend
	for _, prop := range result.GapReport.RecommendedSubLicenses {
		vendorCost := costs[prop.VendorID]
		maxAllowedCap := vendorCost * 0.60
		if prop.EstimatedAnnualCost > maxAllowedCap {
			t.Errorf("sub-license proposal for vendor %s ($%f) exceeds 60%% cap ($%f)", prop.VendorID, prop.EstimatedAnnualCost, maxAllowedCap)
		}
	}
}

// TestE2E_ShadowValidation_Decomposition validates the mathematical guarantees of the shadow comparator:
// Total Delta = Model Delta + Input Delta, neighbor gaps, and degenerate old ranking handling.
func TestE2E_ShadowValidation_Decomposition(t *testing.T) {
	eval := NewEvaluator()
	vendorNames := map[string]string{
		"FDS": "FactSet",
		"ICE": "ICE Data",
		"BBG": "Bloomberg",
		"RFT": "Refinitiv",
	}

	// 1. Normal Drift Decomposition
	oldScores := map[string]float64{
		"FDS": 70.0,
		"ICE": 75.0,
		"BBG": 90.0,
		"RFT": 80.0,
	}

	profiles := []VendorDimensionProfile{
		{VendorID: "FDS", VendorName: "FactSet", CompositeQuality: 0.785, Components: QualityComponents{Sufficiency: 0.85, Coverage: 0.88, SLA: 0.95, Stability: 0.90, Friction: 0.95, Licensing: 0.80}},
		{VendorID: "ICE", VendorName: "ICE Data", CompositeQuality: 0.720, Components: QualityComponents{Sufficiency: 0.80, Coverage: 0.82, SLA: 0.90, Stability: 0.88, Friction: 0.92, Licensing: 0.75}},
		{VendorID: "BBG", VendorName: "Bloomberg", CompositeQuality: 0.880, Components: QualityComponents{Sufficiency: 0.98, Coverage: 0.99, SLA: 0.99, Stability: 0.95, Friction: 0.98, Licensing: 0.95}},
		{VendorID: "RFT", VendorName: "Refinitiv", CompositeQuality: 0.820, Components: QualityComponents{Sufficiency: 0.92, Coverage: 0.94, SLA: 0.92, Stability: 0.91, Friction: 0.94, Licensing: 0.85}},
	}

	report := eval.RunShadowValidation("tenant-e2e", "Balanced Profile", oldScores, profiles, vendorNames)

	if len(report.Comparisons) != 4 {
		t.Fatalf("expected 4 vendor comparisons, got %d", len(report.Comparisons))
	}

	for _, comp := range report.Comparisons {
		sum := comp.ModelEffect + comp.InputEffect
		if math.Abs(sum-comp.ScoreDelta) > 1e-4 {
			t.Errorf("Vendor %s: decomposition violation: %f + %f = %f != %f",
				comp.VendorID, comp.ModelEffect, comp.InputEffect, sum, comp.ScoreDelta)
		}
		if comp.RankDelta == nil {
			t.Errorf("Vendor %s: rank delta should not be nil for valid spread", comp.VendorID)
		}
	}

	// 2. Degenerate Ranking Check (zero spread)
	degenerateOldScores := map[string]float64{
		"FDS": 50.0,
		"ICE": 50.0,
		"BBG": 50.0,
		"RFT": 50.0,
	}

	degenReport := eval.RunShadowValidation("tenant-e2e", "Balanced Profile", degenerateOldScores, profiles, vendorNames)
	if degenReport.Basis.OldRanking != "DEGENERATE" {
		t.Errorf("expected DEGENERATE old ranking, got %s", degenReport.Basis.OldRanking)
	}

	for _, comp := range degenReport.Comparisons {
		if comp.RankDelta != nil {
			t.Errorf("Vendor %s: rank delta must be nil when old ranking is degenerate", comp.VendorID)
		}
	}
}

// TestE2E_HTTP_DisplacementAndScorecard tests the HTTP layer end-to-end with full request/response cycle.
func TestE2E_HTTP_DisplacementAndScorecard(t *testing.T) {
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
		costs: map[string]float64{
			"BBG": 2140000,
			"RFT": 1180000,
			"FDS": 720000,
			"ICE": 540000,
			"SPG": 610000,
		},
	}
	service := NewService(mockRepo)
	handler := NewHandler(service)

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	// 1. POST /api/mdm/scoring/displacement/multi
	multiReq := MultiVendorDisplacementRequest{
		DroppedVendorIDs:     []string{"BBG"},
		ReplacementVendorIDs: []string{"ICE"},
		TargetT1Coverage:     0.995,
		TargetT2Coverage:     0.950,
		TargetT3Coverage:     0.850,
		UniverseSize:         42000,
	}
	body, _ := json.Marshal(multiReq)
	req := httptest.NewRequest("POST", "/api/mdm/scoring/displacement/multi", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("POST /displacement/multi returned %d: %s", rr.Code, rr.Body.String())
	}

	var multiResult MultiVendorDisplacementResult
	if err := json.Unmarshal(rr.Body.Bytes(), &multiResult); err != nil {
		t.Fatalf("failed to decode displacement/multi response: %v", err)
	}
	if len(multiResult.DroppedVendorIDs) != 1 || multiResult.DroppedVendorIDs[0] != "BBG" {
		t.Errorf("unexpected dropped vendor in response: %v", multiResult.DroppedVendorIDs)
	}
	if multiResult.CombinedTCO.LicenseSavingsTotal <= 0 {
		t.Errorf("expected positive license savings total")
	}

	// 2. GET /api/mdm/scoring/shadow-eval
	reqShadow := httptest.NewRequest("GET", "/api/mdm/scoring/shadow-eval", nil)
	reqShadow.Header.Set("X-Tenant-ID", "tenant-e2e")
	rrShadow := httptest.NewRecorder()
	r.ServeHTTP(rrShadow, reqShadow)

	if rrShadow.Code != http.StatusOK {
		t.Fatalf("GET /shadow-eval returned %d: %s", rrShadow.Code, rrShadow.Body.String())
	}

	var shadowResult ShadowValidationReport
	if err := json.Unmarshal(rrShadow.Body.Bytes(), &shadowResult); err != nil {
		t.Fatalf("failed to decode shadow-eval response: %v", err)
	}
	if shadowResult.TenantID != "tenant-e2e" {
		t.Errorf("expected tenant-e2e, got %s", shadowResult.TenantID)
	}
	if len(shadowResult.Comparisons) == 0 {
		t.Errorf("expected comparisons in shadow evaluation")
	}

	// 3. GET /api/mdm/scoring/scorecard
	reqScorecard := httptest.NewRequest("GET", "/api/mdm/scoring/scorecard", nil)
	rrScorecard := httptest.NewRecorder()
	r.ServeHTTP(rrScorecard, reqScorecard)

	if rrScorecard.Code != http.StatusOK {
		t.Fatalf("GET /scorecard returned %d: %s", rrScorecard.Code, rrScorecard.Body.String())
	}

	var scorecard VendorScorecardReport
	if err := json.Unmarshal(rrScorecard.Body.Bytes(), &scorecard); err != nil {
		t.Fatalf("failed to decode scorecard response: %v", err)
	}
	if len(scorecard.SubstitutionMatrix) == 0 {
		t.Errorf("expected populated substitution matrix in scorecard")
	}
	if len(scorecard.DisplacementScenarios) == 0 {
		t.Errorf("expected populated displacement scenarios in scorecard")
	}
}
