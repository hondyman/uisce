package scoring

import (
	"fmt"
	"math"
	"testing"
	"time"
)

func TestEvaluateSubstitutionMatrix(t *testing.T) {
	eval := NewEvaluator()

	tolerances := map[string]AttributeTolerance{
		"LEI": {
			AttributeCode: "LEI",
			Tier:          1,
			MatchType:     MatchExact,
			TierWeight:    0.50,
		},
		"LEGAL_NAME": {
			AttributeCode: "LEGAL_NAME",
			Tier:          2,
			MatchType:     MatchFuzzyJaro,
			ToleranceVal:  0.92,
			TierWeight:    0.30,
		},
	}

	now := time.Now()

	// Entity 1: BBG matches golden, RFT matches golden, FDS differs
	// Entity 2: BBG has valid value, RFT is absent -> BBG is SOLO
	candidates := []VendorCandidate{
		// Entity 1 LEI
		{EntityID: 1, AttributeCode: "LEI", VendorID: "BBG", NormalizedValue: "549300ABC1", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},
		{EntityID: 1, AttributeCode: "LEI", VendorID: "RFT", NormalizedValue: "549300ABC1", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},
		{EntityID: 1, AttributeCode: "LEI", VendorID: "FDS", NormalizedValue: "DIFF_VALUE", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},

		// Entity 2 LEI (RFT is absent, BBG is SOLO)
		{EntityID: 2, AttributeCode: "LEI", VendorID: "BBG", NormalizedValue: "549300ABC2", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},
	}

	goldenRecords := []GoldenRecord{
		{EntityID: 1, AttributeCode: "LEI", GoldenValue: "549300ABC1", WinningVendorID: "BBG", RuleApplied: "SOURCE_PRIORITY"},
		{EntityID: 2, AttributeCode: "LEI", GoldenValue: "549300ABC2", WinningVendorID: "BBG", RuleApplied: "SOURCE_PRIORITY"},
	}

	overrides := []ValueOverrideRecord{
		{EntityID: 1, AttributeCode: "LEI", EndorsementVendorID: "RFT"},
	}

	matrix := eval.EvaluateSubstitutionMatrix(tolerances, candidates, goldenRecords, overrides, 2)

	if len(matrix) == 0 {
		t.Fatalf("expected non-empty substitution matrix")
	}

	// Verify BBG metrics: 2 in scope, 2 available (100%), 2 valid matches (100%), 1 solo (50%)
	var bbgScore, rftScore *SubstitutionScore
	for i := range matrix {
		if matrix[i].VendorID == "BBG" && matrix[i].AttributeCode == "LEI" {
			bbgScore = &matrix[i]
		}
		if matrix[i].VendorID == "RFT" && matrix[i].AttributeCode == "LEI" {
			rftScore = &matrix[i]
		}
	}

	if bbgScore == nil {
		t.Fatalf("missing BBG LEI score")
	}
	if bbgScore.CoveragePct != 100.0 {
		t.Errorf("BBG coverage = %.1f; want 100.0", bbgScore.CoveragePct)
	}
	if bbgScore.SufficiencyRatePct != 100.0 {
		t.Errorf("BBG sufficiency = %.1f; want 100.0", bbgScore.SufficiencyRatePct)
	}
	if bbgScore.SoloRecordsCount != 1 || bbgScore.SoloRatePct != 50.0 {
		t.Errorf("BBG solo records = %d (%.1f%%); want 1 (50.0%%)", bbgScore.SoloRecordsCount, bbgScore.SoloRatePct)
	}

	// Verify RFT metrics: 2 in scope, 1 available (50%), 1 valid match (50%), 0 solo (0%)
	if rftScore == nil {
		t.Fatalf("missing RFT LEI score")
	}
	if rftScore.CoveragePct != 50.0 {
		t.Errorf("RFT coverage = %.1f; want 50.0", rftScore.CoveragePct)
	}
	if rftScore.SufficiencyRatePct != 50.0 {
		t.Errorf("RFT sufficiency = %.1f; want 50.0", rftScore.SufficiencyRatePct)
	}
	if rftScore.ConditionalSufficiency != 100.0 {
		t.Errorf("RFT conditional sufficiency = %.1f; want 100.0", rftScore.ConditionalSufficiency)
	}
	if rftScore.SoloRecordsCount != 0 {
		t.Errorf("RFT solo records = %d; want 0", rftScore.SoloRecordsCount)
	}
	if rftScore.OverrideEndorsementRate != 100.0 {
		t.Errorf("RFT OER = %.1f; want 100.0", rftScore.OverrideEndorsementRate)
	}
}

func TestSimulateVendorDisplacement(t *testing.T) {
	eval := NewEvaluator()

	tolerances := map[string]AttributeTolerance{
		"LEI": {AttributeCode: "LEI", Tier: 1, MatchType: MatchExact},
	}

	now := time.Now()

	// Entity 1: BBG and RFT both have valid value "549300ABC1"
	// Entity 2: BBG has valid value, RFT has nothing (solo to BBG)
	candidates := []VendorCandidate{
		{EntityID: 1, AttributeCode: "LEI", VendorID: "BBG", NormalizedValue: "549300ABC1", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},
		{EntityID: 1, AttributeCode: "LEI", VendorID: "RFT", NormalizedValue: "549300ABC1", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},

		{EntityID: 2, AttributeCode: "LEI", VendorID: "BBG", NormalizedValue: "549300ABC2", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},
	}

	goldenRecords := []GoldenRecord{
		{EntityID: 1, AttributeCode: "LEI", GoldenValue: "549300ABC1", WinningVendorID: "BBG"},
		{EntityID: 2, AttributeCode: "LEI", GoldenValue: "549300ABC2", WinningVendorID: "BBG"},
	}

	hierarchy := []string{"BBG", "RFT"}
	annualCost := 2000000.0

	res := eval.SimulateVendorDisplacement("BBG", "Bloomberg", annualCost, tolerances, hierarchy, candidates, goldenRecords)

	if res.DroppedVendorID != "BBG" {
		t.Errorf("dropped vendor = %s; want BBG", res.DroppedVendorID)
	}

	// Entity 1 should be UNCHANGED (RFT matches old golden)
	// Entity 2 should be NOW NULL (no other vendor has it)
	if res.TotalNullValues != 1 {
		t.Errorf("total null values = %d; want 1", res.TotalNullValues)
	}

	if len(res.TierSummaries) == 0 {
		t.Fatalf("expected tier summaries")
	}

	t1 := res.TierSummaries[0]
	if t1.UnchangedPct != 50.0 {
		t.Errorf("tier 1 unchanged = %.1f; want 50.0", t1.UnchangedPct)
	}
	if t1.NowNullPct != 50.0 {
		t.Errorf("tier 1 now null = %.1f; want 50.0", t1.NowNullPct)
	}

	// Verify net commercial benefit: $2,000,000 - (1 * $18) = $1,999,982
	expectedNet := annualCost - 18.0
	if res.NetFirstYearBenefit != expectedNet {
		t.Errorf("net benefit = %.2f; want %.2f", res.NetFirstYearBenefit, expectedNet)
	}
}

func TestComputeValueForMoneyFrontier(t *testing.T) {
	eval := NewEvaluator()

	vendors := map[string]string{
		"V1": "Vendor 1 (Cheap)",
		"V2": "Vendor 2 (Mid)",
		"V3": "Vendor 3 (Dominated)",
		"V4": "Vendor 4 (Premium)",
	}
	costs := map[string]float64{
		"V1": 100000,
		"V2": 300000,
		"V3": 400000, // Costs more than V2 but lower quality than V2
		"V4": 800000,
	}
	scores := map[string]float64{
		"V1": 70.0,
		"V2": 85.0,
		"V3": 80.0, // Dominated by V2
		"V4": 95.0,
	}

	frontier := eval.ComputeValueForMoneyFrontier(vendors, costs, scores)

	frontierMap := make(map[string]bool)
	for _, p := range frontier {
		frontierMap[p.VendorID] = p.IsOnFrontier
	}

	if !frontierMap["V1"] {
		t.Errorf("expected V1 to be on frontier")
	}
	if !frontierMap["V2"] {
		t.Errorf("expected V2 to be on frontier")
	}
	if frontierMap["V3"] {
		t.Errorf("expected V3 to be dominated (not on frontier)")
	}
	if !frontierMap["V4"] {
		t.Errorf("expected V4 to be on frontier")
	}
}

func TestComputeStabilityScore(t *testing.T) {
	eval := NewEvaluator()

	tests := []struct {
		revisions int
		published int
		decayK    float64
		wantRate  float64
		wantMin   float64
		wantMax   float64
	}{
		{revisions: 0, published: 1000, decayK: 50.0, wantRate: 0.0, wantMin: 1.0, wantMax: 1.0},
		{revisions: 5, published: 1000, decayK: 50.0, wantRate: 0.005, wantMin: 0.77, wantMax: 0.79},    // e^-0.25 ≈ 0.7788
		{revisions: 20, published: 1000, decayK: 50.0, wantRate: 0.020, wantMin: 0.36, wantMax: 0.375}, // e^-1.0 ≈ 0.3679
		{revisions: 50, published: 1000, decayK: 50.0, wantRate: 0.050, wantMin: 0.08, wantMax: 0.09},   // e^-2.5 ≈ 0.0821
	}

	for _, tt := range tests {
		score, rate := eval.ComputeStabilityScore(tt.revisions, tt.published, tt.decayK)
		if math.Abs(rate-tt.wantRate) > 1e-6 {
			t.Errorf("revision rate for %d/%d = %f; want %f", tt.revisions, tt.published, rate, tt.wantRate)
		}
		if score < tt.wantMin || score > tt.wantMax {
			t.Errorf("stability score for rate %f = %f; want between %f and %f", rate, score, tt.wantMin, tt.wantMax)
		}
	}
}

func TestComputeFrictionScore(t *testing.T) {
	eval := NewEvaluator()

	// 20 hours @ $150 = $3,000 friction cost. Budget = $100,000.
	// Score = 1.0 - (3000 / 100000) = 0.970.
	score, cost := eval.ComputeFrictionScore(20.0, 150.0, 0.0, 100000.0)
	if cost != 3000.0 {
		t.Errorf("friction cost = %.2f; want 3000.0", cost)
	}
	if math.Abs(score-0.970) > 1e-4 {
		t.Errorf("friction score = %.4f; want 0.970", score)
	}

	// With $1,000 SLA credits offset: cost = $2,000 -> score = 0.980
	scoreWithCredit, costWithCredit := eval.ComputeFrictionScore(20.0, 150.0, 1000.0, 100000.0)
	if costWithCredit != 2000.0 {
		t.Errorf("friction cost with credit = %.2f; want 2000.0", costWithCredit)
	}
	if math.Abs(scoreWithCredit-0.980) > 1e-4 {
		t.Errorf("friction score with credit = %.4f; want 0.980", scoreWithCredit)
	}

	// Cost exceeding budget yields 0.0 floor
	zeroScore, zeroCost := eval.ComputeFrictionScore(1000.0, 150.0, 0.0, 100000.0)
	if zeroCost != 150000.0 {
		t.Errorf("cost = %.2f; want 150000.0", zeroCost)
	}
	if zeroScore != 0.0 {
		t.Errorf("zeroScore = %f; want 0.0", zeroScore)
	}
}

func TestComputeSLAScore(t *testing.T) {
	eval := NewEvaluator()

	// 0 breaches in 30 days -> 1.0
	if s := eval.ComputeSLAScore(0, 0, 30); s != 1.0 {
		t.Errorf("SLA score = %f; want 1.0", s)
	}

	// 1 breach in 30 days -> 1 - 1/30 ≈ 0.9667
	if s := eval.ComputeSLAScore(1, 0, 30); math.Abs(s-(29.0/30.0)) > 1e-4 {
		t.Errorf("SLA score = %f; want %f", s, 29.0/30.0)
	}

	// 1 missing delivery (3x penalty) in 30 days -> 1 - 3/30 = 0.90
	if s := eval.ComputeSLAScore(0, 1, 30); math.Abs(s-0.90) > 1e-4 {
		t.Errorf("SLA score = %f; want 0.90", s)
	}
}

func TestComputeCommercialRightsScore(t *testing.T) {
	eval := NewEvaluator()

	rights := VendorContractRights{
		DerivedDataRights:      90,
		ClientRedistribution:   80,
		ExternalWebRights:      65,
		UnbundledAPIAccess:     true,
		CancellationNoticeDays: 60,
	}
	// Formula: (0.35*90 + 0.25*80 + 0.15*65 + 0.15*100 + 0.10*70) / 100
	// = (31.5 + 20.0 + 9.75 + 15.0 + 7.0) / 100 = 83.25 / 100 = 0.8325
	score := eval.ComputeCommercialRightsScore(rights)
	if math.Abs(score-0.8325) > 1e-4 {
		t.Errorf("rights score = %f; want 0.8325", score)
	}
}

func TestComputeCompositeQuality(t *testing.T) {
	eval := NewEvaluator()

	comp := QualityComponents{
		Sufficiency: 0.95,
		Coverage:    0.98,
		SLA:         0.97,
		Stability:   0.90,
		Friction:    0.95,
		Licensing:   0.85,
	}

	profile := WeightProfile{
		WeightSuff: 0.300,
		WeightCov:  0.200,
		WeightSLA:  0.150,
		WeightStab: 0.150,
		WeightOER:  0.100,
		WeightLic:  0.100,
	}

	// 0.95*0.30 + 0.98*0.20 + 0.97*0.15 + 0.90*0.15 + 0.95*0.10 + 0.85*0.10
	// = 0.285 + 0.196 + 0.1455 + 0.135 + 0.095 + 0.085 = 0.9415
	q := eval.ComputeCompositeQuality(comp, profile)
	if math.Abs(q-0.9415) > 1e-4 {
		t.Errorf("composite quality = %f; want 0.9415", q)
	}
}

func TestSimulateVendorDisplacement_TCO(t *testing.T) {
	eval := NewEvaluator()

	tolerances := map[string]AttributeTolerance{
		"LEI": {AttributeCode: "LEI", Tier: 1, MatchType: MatchExact},
	}
	now := time.Now()
	candidates := []VendorCandidate{
		{EntityID: 1, AttributeCode: "LEI", VendorID: "BBG", NormalizedValue: "V1", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},
		{EntityID: 1, AttributeCode: "LEI", VendorID: "RFT", NormalizedValue: "V1", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},
		{EntityID: 2, AttributeCode: "LEI", VendorID: "BBG", NormalizedValue: "V2", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},
	}
	golden := []GoldenRecord{
		{EntityID: 1, AttributeCode: "LEI", GoldenValue: "V1", WinningVendorID: "BBG"},
		{EntityID: 2, AttributeCode: "LEI", GoldenValue: "V2", WinningVendorID: "BBG"},
	}

	res := eval.SimulateVendorDisplacement(
		"BBG", "Bloomberg", 2000000.0, tolerances, []string{"BBG", "RFT"}, candidates, golden,
		DisplacementTCOOptions{
			DroppedFrictionCost: 50000.0,
			DroppedSLACredits:   5000.0,
		},
	)

	// Net first year benefit: 2,000,000 - (1 * 18) = 1,999,982
	if res.NetFirstYearBenefit != 1999982.0 {
		t.Errorf("NetFirstYearBenefit = %.2f; want 1999982.00", res.NetFirstYearBenefit)
	}

	// Net TCO benefit: 1,999,982 + 50,000 (friction eliminated) - 5,000 (forfeited SLA credits) = 2,044,982
	expectedTCO := 1999982.0 + 50000.0 - 5000.0
	if res.NetTCOBenefit != expectedTCO {
		t.Errorf("NetTCOBenefit = %.2f; want %.2f", res.NetTCOBenefit, expectedTCO)
	}
	if res.FrictionSavings != 50000.0 {
		t.Errorf("FrictionSavings = %.2f; want 50000.0", res.FrictionSavings)
	}
	if res.ForfeitedSLACredits != 5000.0 {
		t.Errorf("ForfeitedSLACredits = %.2f; want 5000.0", res.ForfeitedSLACredits)
	}
}

func TestSolveOptimalVendorBundle_Bitmask(t *testing.T) {
	eval := NewEvaluator()

	tolerances := map[string]AttributeTolerance{
		"ISIN": {AttributeCode: "ISIN", Tier: 1, MatchType: MatchExact},
		"NAME": {AttributeCode: "NAME", Tier: 2, MatchType: MatchExact},
		"INFO": {AttributeCode: "INFO", Tier: 3, MatchType: MatchExact},
	}

	// 100 entities:
	// V_EXPENSIVE covers 100% of entities in T1, T2, T3 (Cost $1,000,000)
	// V_CHEAP1 covers 1-60 in T1, T2, T3 (Cost $200,000)
	// V_CHEAP2 covers 50-100 in T1, T2, T3 (Cost $200,000)
	// Bundling CHEAP1 + CHEAP2 gives 100% coverage at $400,000 (Saving $600,000 vs V_EXPENSIVE)
	now := time.Now()
	var candidates []VendorCandidate

	for id := int64(1); id <= 100; id++ {
		// V_EXPENSIVE has all
		candidates = append(candidates,
			VendorCandidate{EntityID: id, AttributeCode: "ISIN", VendorID: "V_EXP", NormalizedValue: "V", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},
			VendorCandidate{EntityID: id, AttributeCode: "NAME", VendorID: "V_EXP", NormalizedValue: "V", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},
			VendorCandidate{EntityID: id, AttributeCode: "INFO", VendorID: "V_EXP", NormalizedValue: "V", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},
		)

		if id <= 60 {
			candidates = append(candidates,
				VendorCandidate{EntityID: id, AttributeCode: "ISIN", VendorID: "V_C1", NormalizedValue: "V", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},
				VendorCandidate{EntityID: id, AttributeCode: "NAME", VendorID: "V_C1", NormalizedValue: "V", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},
				VendorCandidate{EntityID: id, AttributeCode: "INFO", VendorID: "V_C1", NormalizedValue: "V", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},
			)
		}
		if id >= 50 {
			candidates = append(candidates,
				VendorCandidate{EntityID: id, AttributeCode: "ISIN", VendorID: "V_C2", NormalizedValue: "V", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},
				VendorCandidate{EntityID: id, AttributeCode: "NAME", VendorID: "V_C2", NormalizedValue: "V", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},
				VendorCandidate{EntityID: id, AttributeCode: "INFO", VendorID: "V_C2", NormalizedValue: "V", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},
			)
		}
	}

	allVendors := []string{"V_EXP", "V_C1", "V_C2"}
	vendorNames := map[string]string{
		"V_EXP": "Expensive Mega Feed",
		"V_C1":  "Cheap Regional Feed 1",
		"V_C2":  "Cheap Regional Feed 2",
	}
	costs := map[string]float64{
		"V_EXP": 1000000.0,
		"V_C1":  200000.0,
		"V_C2":  200000.0,
	}

	req := OptimalBundleRequest{
		TargetT1Coverage: 0.99,
		TargetT2Coverage: 0.95,
		TargetT3Coverage: 0.85,
		UniverseSize:     100,
	}

	res := eval.SolveOptimalVendorBundle(req, allVendors, vendorNames, costs, candidates, tolerances, 100)

	// Optimal bundle should pick V_C1 + V_C2 for $400,000, achieving 100% coverage
	if res.TotalAnnualCost != 400000.0 {
		t.Errorf("TotalAnnualCost = %.2f; want 400000.00", res.TotalAnnualCost)
	}
	if len(res.SelectedVendors) != 2 {
		t.Errorf("len(SelectedVendors) = %d; want 2", len(res.SelectedVendors))
	}
	if res.AnnualSavings != 1000000.0 {
		t.Errorf("AnnualSavings = %.2f; want 1000000.00", res.AnnualSavings)
	}
	if res.T1CoverageAchieved != 1.0 {
		t.Errorf("T1CoverageAchieved = %f; want 1.0", res.T1CoverageAchieved)
	}
	if res.WasRelaxed {
		t.Errorf("expected bundle to be feasible without relaxation")
	}

	// Test infeasibility relaxation
	unachievableReq := OptimalBundleRequest{
		TargetT1Coverage: 1.0,
		TargetT2Coverage: 1.0,
		TargetT3Coverage: 1.0,
		ExcludedVendors:  []string{"V_EXP", "V_C2"}, // Only V_C1 allowed (only has 60% coverage)
		UniverseSize:     100,
	}
	relaxedRes := eval.SolveOptimalVendorBundle(unachievableReq, allVendors, vendorNames, costs, candidates, tolerances, 100)
	if !relaxedRes.WasRelaxed {
		t.Errorf("expected WasRelaxed = true for impossible constraint")
	}
	if len(relaxedRes.ResidualGaps) == 0 {
		t.Errorf("expected ResidualGaps to be populated")
	}
	if relaxedRes.T1CoverageAchieved != 0.60 {
		t.Errorf("Achieved coverage = %f; want 0.60", relaxedRes.T1CoverageAchieved)
	}
}

func setupSyntheticV20Universe(universeSize int) (
	allVendors []string,
	vendorNames map[string]string,
	costs map[string]float64,
	candidates []VendorCandidate,
	tolerances map[string]AttributeTolerance,
) {
	tolerances = map[string]AttributeTolerance{
		"ISIN": {AttributeCode: "ISIN", Tier: 1, MatchType: MatchExact},
		"NAME": {AttributeCode: "NAME", Tier: 2, MatchType: MatchExact},
		"INFO": {AttributeCode: "INFO", Tier: 3, MatchType: MatchExact},
	}

	V := 20
	vendorNames = make(map[string]string)
	costs = make(map[string]float64)

	for i := 1; i <= V; i++ {
		vID := fmt.Sprintf("V_%02d", i)
		allVendors = append(allVendors, vID)
		vendorNames[vID] = fmt.Sprintf("Vendor %d", i)
		costs[vID] = float64(100000 + i*50000)
	}

	now := time.Now()
	for id := int64(1); id <= int64(universeSize); id++ {
		// V_01 and V_02 provide high core market coverage
		if id%50 != 0 {
			candidates = append(candidates,
				VendorCandidate{EntityID: id, AttributeCode: "ISIN", VendorID: "V_01", NormalizedValue: "VAL", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},
				VendorCandidate{EntityID: id, AttributeCode: "NAME", VendorID: "V_01", NormalizedValue: "VAL", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},
				VendorCandidate{EntityID: id, AttributeCode: "INFO", VendorID: "V_01", NormalizedValue: "VAL", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},
			)
		}
		if id%30 != 0 {
			candidates = append(candidates,
				VendorCandidate{EntityID: id, AttributeCode: "ISIN", VendorID: "V_02", NormalizedValue: "VAL", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},
				VendorCandidate{EntityID: id, AttributeCode: "NAME", VendorID: "V_02", NormalizedValue: "VAL", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},
			)
		}
		// Specialized vendors (V_03 through V_20) cover segment slices
		for v := 3; v <= V; v++ {
			if int(id)%v == 0 {
				candidates = append(candidates,
					VendorCandidate{EntityID: id, AttributeCode: "ISIN", VendorID: fmt.Sprintf("V_%02d", v), NormalizedValue: "VAL", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},
				)
			}
		}
	}
	return
}

func TestSolveOptimalVendorBundle_V20_Scale(t *testing.T) {
	eval := NewEvaluator()
	universeSize := 42000
	allVendors, vendorNames, costs, candidates, tolerances := setupSyntheticV20Universe(universeSize)

	req := OptimalBundleRequest{
		TargetT1Coverage: 0.95,
		TargetT2Coverage: 0.90,
		TargetT3Coverage: 0.80,
		UniverseSize:     universeSize,
	}

	start := time.Now()
	res := eval.SolveOptimalVendorBundle(req, allVendors, vendorNames, costs, candidates, tolerances, universeSize)
	elapsed := time.Since(start)

	if len(res.SelectedVendors) == 0 {
		t.Fatalf("expected non-empty selected vendors")
	}
	if res.SolverStrategy != "BITMASK_BRANCH_AND_BOUND" {
		t.Errorf("solver strategy = %s; want BITMASK_BRANCH_AND_BOUND", res.SolverStrategy)
	}

	t.Logf("V=20 Solver Execution Time: %v (solver reported: %.2f ms), selected %d vendors: %v, cost: $%.2f",
		elapsed, res.SolverExecutionMs, len(res.SelectedVendors), res.SelectedVendors, res.TotalAnnualCost)
}

func BenchmarkSolveOptimalVendorBundle_V20(b *testing.B) {
	eval := NewEvaluator()
	universeSize := 42000
	allVendors, vendorNames, costs, candidates, tolerances := setupSyntheticV20Universe(universeSize)

	req := OptimalBundleRequest{
		TargetT1Coverage: 0.95,
		TargetT2Coverage: 0.90,
		TargetT3Coverage: 0.80,
		UniverseSize:     universeSize,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res := eval.SolveOptimalVendorBundle(req, allVendors, vendorNames, costs, candidates, tolerances, universeSize)
		if len(res.SelectedVendors) == 0 {
			b.Fatalf("failed to find bundle")
		}
	}
}

func TestEvaluateMultiDisplacement(t *testing.T) {
	eval := NewEvaluator()
	tolerances := map[string]AttributeTolerance{
		"ISIN": {
			AttributeCode: "ISIN",
			Tier:          1,
			MatchType:     MatchExact,
			TierWeight:    0.50,
		},
		"LEGAL_NAME": {
			AttributeCode: "LEGAL_NAME",
			Tier:          2,
			MatchType:     MatchFuzzyJaro,
			ToleranceVal:  0.92,
			TierWeight:    0.30,
		},
		"YEAR_FOUNDED": {
			AttributeCode: "YEAR_FOUNDED",
			Tier:          3,
			MatchType:     MatchExact,
			TierWeight:    0.20,
		},
	}

	hierarchy := []string{"BBG", "RFT", "FDS", "ICE", "SPG"}
	costs := map[string]float64{
		"BBG": 2140000,
		"RFT": 1180000,
		"FDS": 720000,
		"ICE": 540000,
		"SPG": 610000,
	}
	vendorNames := map[string]string{
		"BBG": "Bloomberg",
		"RFT": "Refinitiv (LSEG)",
		"FDS": "FactSet",
		"ICE": "ICE Data Services",
		"SPG": "S&P Global MI",
	}

	now := time.Now()
	var candidates []VendorCandidate
	for i := 1; i <= 100; i++ {
		candidates = append(candidates,
			VendorCandidate{EntityID: int64(i), AttributeCode: "ISIN", VendorID: "BBG", NormalizedValue: fmt.Sprintf("US%010d", i), FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},
			VendorCandidate{EntityID: int64(i), AttributeCode: "ISIN", VendorID: "RFT", NormalizedValue: fmt.Sprintf("US%010d", i), FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},
			VendorCandidate{EntityID: int64(i), AttributeCode: "ISIN", VendorID: "ICE", NormalizedValue: fmt.Sprintf("US%010d", i), FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},
			VendorCandidate{EntityID: int64(i), AttributeCode: "LEGAL_NAME", VendorID: "BBG", NormalizedValue: "Acme Corp", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},
			VendorCandidate{EntityID: int64(i), AttributeCode: "LEGAL_NAME", VendorID: "RFT", NormalizedValue: "Acme Corp", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},
		)
	}

	var goldenRecords []GoldenRecord
	for i := 1; i <= 100; i++ {
		goldenRecords = append(goldenRecords,
			GoldenRecord{EntityID: int64(i), AttributeCode: "ISIN", GoldenValue: fmt.Sprintf("US%010d", i), WinningVendorID: "BBG", RuleApplied: "SOURCE_PRIORITY"},
			GoldenRecord{EntityID: int64(i), AttributeCode: "LEGAL_NAME", GoldenValue: "Acme Corp", WinningVendorID: "BBG", RuleApplied: "SOURCE_PRIORITY"},
		)
	}

	frictions := []VendorOperationalFriction{
		{VendorID: "BBG", DefectTicketsCount: 8, InvestigationHours: 30, CalculatedFrictionCost: 4500, ContractSLACredits: 15000},
		{VendorID: "FDS", DefectTicketsCount: 15, InvestigationHours: 40, CalculatedFrictionCost: 6000, ContractSLACredits: 8000},
	}

	t.Run("MultiDropAndReplace", func(t *testing.T) {
		req := MultiVendorDisplacementRequest{
			DroppedVendorIDs:     []string{"BBG", "FDS"},
			ReplacementVendorIDs: []string{"ICE", "SPG"},
			TargetT1Coverage:     0.995,
			TargetT2Coverage:     0.950,
			TargetT3Coverage:     0.850,
			UniverseSize:         100,
		}

		res := eval.EvaluateMultiDisplacement(req, tolerances, hierarchy, costs, vendorNames, candidates, goldenRecords, frictions, 100)

		if len(res.DroppedVendorIDs) != 2 {
			t.Errorf("expected 2 dropped vendors, got %d", len(res.DroppedVendorIDs))
		}
		if len(res.ReplacementVendorIDs) != 2 {
			t.Errorf("expected 2 replacement vendors, got %d", len(res.ReplacementVendorIDs))
		}

		expectedSavings := costs["BBG"] + costs["FDS"]
		if res.CombinedTCO.LicenseSavingsTotal != expectedSavings {
			t.Errorf("expected license savings %.2f, got %.2f", expectedSavings, res.CombinedTCO.LicenseSavingsTotal)
		}

		expectedReplCost := costs["ICE"] + costs["SPG"]
		if res.CombinedTCO.ReplacementCostDelta != expectedReplCost {
			t.Errorf("expected replacement cost %.2f, got %.2f", expectedReplCost, res.CombinedTCO.ReplacementCostDelta)
		}

		if len(res.PerVendorBreakdown) != 2 {
			t.Errorf("expected 2 per-vendor breakdown rows, got %d", len(res.PerVendorBreakdown))
		}

		if len(res.TierCoverageDeltas) != 3 {
			t.Errorf("expected 3 tier coverage deltas, got %d", len(res.TierCoverageDeltas))
		}

		if res.CombinedTCO.PaybackMonths < 0 {
			t.Errorf("expected positive payback months, got %.2f", res.CombinedTCO.PaybackMonths)
		}
	})

	t.Run("SingleDropNoReplacement", func(t *testing.T) {
		req := MultiVendorDisplacementRequest{
			DroppedVendorIDs: []string{"BBG"},
			UniverseSize:     100,
		}

		res := eval.EvaluateMultiDisplacement(req, tolerances, hierarchy, costs, vendorNames, candidates, goldenRecords, frictions, 100)
		if len(res.DroppedVendorIDs) != 1 {
			t.Errorf("expected 1 dropped vendor, got %d", len(res.DroppedVendorIDs))
		}
		if res.CombinedTCO.ReplacementCostDelta != 0 {
			t.Errorf("expected 0 replacement cost, got %.2f", res.CombinedTCO.ReplacementCostDelta)
		}
		if res.CombinedTCO.LicenseSavingsTotal != costs["BBG"] {
			t.Errorf("expected license savings %.2f, got %.2f", costs["BBG"], res.CombinedTCO.LicenseSavingsTotal)
		}
		if res.GapReport.NetNegotiationLeverage <= 0 || res.GapReport.NetNegotiationLeverage > 1.0 {
			t.Errorf("unexpected negotiation leverage: %f", res.GapReport.NetNegotiationLeverage)
		}
	})
}

func TestComputeResidualGaps(t *testing.T) {
	eval := NewEvaluator()
	tolerances := map[string]AttributeTolerance{
		"T1_SOLE": {AttributeCode: "T1_SOLE", Tier: 1},
		"T2_SOLE": {AttributeCode: "T2_SOLE", Tier: 2},
		"T3_SOLE": {AttributeCode: "T3_SOLE", Tier: 3},
		"T2_SUB":  {AttributeCode: "T2_SUB", Tier: 2},
	}
	hierarchy := []string{"V_DROP", "V_KEEP"}
	costs := map[string]float64{
		"V_DROP": 1000000.0,
		"V_KEEP": 500000.0,
	}
	vendorNames := map[string]string{
		"V_DROP": "Drop Vendor Inc",
		"V_KEEP": "Keep Vendor Inc",
	}

	now := time.Now()
	var candidates []VendorCandidate

	// 1000 universe size
	// V_DROP provides T1_SOLE for 100 entities (10% of universe)
	// V_DROP provides T2_SOLE for 200 entities (20% of universe)
	// V_DROP provides T3_SOLE for 300 entities (30% of universe)
	// V_DROP and V_KEEP provide T2_SUB for entity 1..50; V_DROP provides T2_SUB for entity 51..100
	for id := int64(1); id <= 100; id++ {
		candidates = append(candidates, VendorCandidate{
			EntityID: id, AttributeCode: "T1_SOLE", VendorID: "V_DROP", NormalizedValue: "VAL", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now,
		})
	}
	for id := int64(1); id <= 200; id++ {
		candidates = append(candidates, VendorCandidate{
			EntityID: id, AttributeCode: "T2_SOLE", VendorID: "V_DROP", NormalizedValue: "VAL", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now,
		})
	}
	for id := int64(1); id <= 300; id++ {
		candidates = append(candidates, VendorCandidate{
			EntityID: id, AttributeCode: "T3_SOLE", VendorID: "V_DROP", NormalizedValue: "VAL", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now,
		})
	}
	for id := int64(1); id <= 100; id++ {
		candidates = append(candidates, VendorCandidate{
			EntityID: id, AttributeCode: "T2_SUB", VendorID: "V_DROP", NormalizedValue: "VAL", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now,
		})
		if id <= 50 {
			candidates = append(candidates, VendorCandidate{
				EntityID: id, AttributeCode: "T2_SUB", VendorID: "V_KEEP", NormalizedValue: "VAL", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now,
			})
		}
	}

	t.Run("GranularOrphanDetectionAndPricing", func(t *testing.T) {
		report := eval.ComputeResidualGaps(
			[]string{"V_DROP"},
			nil,
			hierarchy,
			candidates,
			tolerances,
			costs,
			vendorNames,
			1000,
		)

		if len(report.Tier1Gaps) != 1 {
			t.Fatalf("expected 1 T1 gap, got %d", len(report.Tier1Gaps))
		}
		t1 := report.Tier1Gaps[0]
		if t1.AttributeCode != "T1_SOLE" || t1.EntitiesAffected != 100 || !t1.IsTier1Critical {
			t.Errorf("unexpected T1 gap: %+v", t1)
		}
		// T1: Cost($1M) * (100/1000) * 1.5 = $150,000
		if t1.EstimatedSubLicenseCost != 150000.0 {
			t.Errorf("T1 cost = %.2f; want 150000.00", t1.EstimatedSubLicenseCost)
		}

		// Check T2_SUB has substitute
		foundSub := false
		for _, g := range report.Tier2Gaps {
			if g.AttributeCode == "T2_SUB" {
				foundSub = true
				if g.SuggestedSubstituteID != "V_KEEP" {
					t.Errorf("T2_SUB substitute = %s; want V_KEEP", g.SuggestedSubstituteID)
				}
				if g.EstimatedSubLicenseCost != 0.0 {
					t.Errorf("T2_SUB cost = %.2f; want 0.00", g.EstimatedSubLicenseCost)
				}
			}
		}
		if !foundSub {
			t.Errorf("expected T2_SUB in Tier2Gaps")
		}

		// Check RecommendedSubLicenses
		if len(report.RecommendedSubLicenses) != 1 {
			t.Fatalf("expected 1 sub-license proposal, got %d", len(report.RecommendedSubLicenses))
		}
		proposal := report.RecommendedSubLicenses[0]
		if proposal.VendorID != "V_DROP" {
			t.Errorf("proposal vendor = %s; want V_DROP", proposal.VendorID)
		}
		if proposal.TierPriority != 1 {
			t.Errorf("proposal tier priority = %d; want 1", proposal.TierPriority)
		}
		if proposal.EntitiesCovered != 600 {
			t.Errorf("proposal entities covered = %d; want 600", proposal.EntitiesCovered)
		}

		// Leverage = 1 - (EstimatedGapRemediation / 1,000,000)
		expectedLeverage := 1.0 - (report.EstimatedGapRemediation / 1000000.0)
		if math.Abs(report.NetNegotiationLeverage-expectedLeverage) > 0.0001 {
			t.Errorf("leverage = %f; want %f", report.NetNegotiationLeverage, expectedLeverage)
		}
	})

	t.Run("CapEnforcementAt60Percent", func(t *testing.T) {
		// When orphan entities are 900/1000, unconstrained cost would exceed 60% cap
		var bigCandidates []VendorCandidate
		for id := int64(1); id <= 900; id++ {
			bigCandidates = append(bigCandidates, VendorCandidate{
				EntityID: id, AttributeCode: "T1_SOLE", VendorID: "V_DROP", NormalizedValue: "VAL", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now,
			})
		}

		report := eval.ComputeResidualGaps(
			[]string{"V_DROP"},
			nil,
			hierarchy,
			bigCandidates,
			tolerances,
			costs,
			vendorNames,
			1000,
		)

		maxCap := costs["V_DROP"] * 0.60 // $600,000
		if report.EstimatedGapRemediation > maxCap {
			t.Errorf("total remediation = %.2f; exceeds 60%% cap %.2f", report.EstimatedGapRemediation, maxCap)
		}
		if report.NetNegotiationLeverage < 0.40 {
			t.Errorf("leverage = %f; expected >= 0.40 with 60%% cap", report.NetNegotiationLeverage)
		}
	})
}

func TestRunShadowValidation(t *testing.T) {
	eval := NewEvaluator()
	vendorNames := map[string]string{
		"V1": "Vendor One",
		"V2": "Vendor Two",
		"V3": "Vendor Three",
		"V4": "Vendor Four",
	}

	t.Run("StableRanking_DeltaZero", func(t *testing.T) {
		oldScores := map[string]float64{
			"V1": 95.0,
			"V2": 85.0,
			"V3": 75.0,
			"V4": 65.0,
		}
		newProfiles := []VendorDimensionProfile{
			{VendorID: "V1", CompositeQuality: 0.94},
			{VendorID: "V2", CompositeQuality: 0.86},
			{VendorID: "V3", CompositeQuality: 0.74},
			{VendorID: "V4", CompositeQuality: 0.66},
		}

		res := eval.RunShadowValidation("tenant-1", "Default Procurement", oldScores, newProfiles, vendorNames)

		if !res.IsStable {
			t.Errorf("expected IsStable = true for identical ranking")
		}
		if res.MaxRankDelta != 0 {
			t.Errorf("expected MaxRankDelta = 0, got %d", res.MaxRankDelta)
		}
		if res.PairwiseAgreement != 100.0 {
			t.Errorf("expected 100%% agreement, got %.1f%%", res.PairwiseAgreement)
		}
		if res.RankInversionRate != 0.0 {
			t.Errorf("expected 0%% inversion, got %.1f%%", res.RankInversionRate)
		}
	})

	t.Run("ShiftedRanking_DeltaThreeUnstable", func(t *testing.T) {
		oldScores := map[string]float64{
			"V1": 95.0, // Rank 1
			"V2": 85.0, // Rank 2
			"V3": 75.0, // Rank 3
			"V4": 65.0, // Rank 4
		}
		// V1 plunges to bottom due to high friction and restatements
		newProfiles := []VendorDimensionProfile{
			{VendorID: "V2", CompositeQuality: 0.90}, // New Rank 1 (old 2)
			{VendorID: "V3", CompositeQuality: 0.80}, // New Rank 2 (old 3)
			{VendorID: "V4", CompositeQuality: 0.70}, // New Rank 3 (old 4)
			{VendorID: "V1", CompositeQuality: 0.50}, // New Rank 4 (old 1) -> Delta = -3
		}

		res := eval.RunShadowValidation("tenant-1", "Default Procurement", oldScores, newProfiles, vendorNames)

		if res.IsStable {
			t.Errorf("expected IsStable = false for 3-rank displacement")
		}
		if res.MaxRankDelta != 3 {
			t.Errorf("expected MaxRankDelta = 3, got %d", res.MaxRankDelta)
		}
		if res.RankInversionRate <= 0.0 {
			t.Errorf("expected positive inversion rate, got %.1f%%", res.RankInversionRate)
		}
	})

	t.Run("TieStableDamping_MicroShift", func(t *testing.T) {
		oldScores := map[string]float64{
			"V1": 90.0, // Rank 1
			"V2": 89.9, // Rank 2
		}
		// V2 slightly nudges ahead by 0.002 (0.2 points), within 0.005 damping margin
		newProfiles := []VendorDimensionProfile{
			{VendorID: "V2", CompositeQuality: 0.892}, // 89.2
			{VendorID: "V1", CompositeQuality: 0.890}, // 89.0 -> diff is 0.2 points (< 0.5)
		}

		res := eval.RunShadowValidation("tenant-1", "Default Procurement", oldScores, newProfiles, vendorNames)

		if !res.IsStable {
			t.Errorf("expected IsStable = true under tie-break damping")
		}
		for _, comp := range res.Comparisons {
			if comp.Stability != ShadowTieBound {
				t.Errorf("expected vendor %s to be tie-bound, got %s", comp.VendorID, comp.Stability)
			}
			if comp.NeighborGap == nil || math.Abs(*comp.NeighborGap-0.2) > 0.01 {
				t.Errorf("expected vendor %s NeighborGap = 0.2, got %v", comp.VendorID, comp.NeighborGap)
			}
		}
	})

	t.Run("SafetyAlert_ZeroLicensingRights", func(t *testing.T) {
		oldScores := map[string]float64{"V1": 80.0, "V2": 70.0}
		newProfiles := []VendorDimensionProfile{
			{VendorID: "V1", CompositeQuality: 0.85, Components: QualityComponents{Licensing: 0.0, SLA: 1.0}},
			{VendorID: "V2", CompositeQuality: 0.75, Components: QualityComponents{Licensing: 1.0, SLA: 1.0}},
		}

		res := eval.RunShadowValidation("tenant-1", "Default Procurement", oldScores, newProfiles, vendorNames)

		foundAlert := false
		for _, comp := range res.Comparisons {
			if comp.VendorID == "V1" && comp.SafetyAlert != "" {
				foundAlert = true
			}
		}
		if !foundAlert {
			t.Errorf("expected safety alert for vendor with zero licensing rights promoted to top tier")
		}
	})
}

func TestDecomposeEffect_ExactSum(t *testing.T) {
	oldW := map[string]float64{"D1": 0.4, "D2": 0.3, "D3": 0.3}
	newW := map[string]float64{"D1": 0.2, "D2": 0.5, "D3": 0.3}
	oldS := map[string]float64{"D1": 80.0, "D2": 60.0, "D3": 90.0}
	newS := map[string]float64{"D1": 85.0, "D2": 75.0, "D3": 70.0}

	modelEff, inputEff := decomposeEffect(oldW, newW, oldS, newS)

	oldTotal := 0.4*80.0 + 0.3*60.0 + 0.3*90.0
	newTotal := 0.2*85.0 + 0.5*75.0 + 0.3*70.0
	deltaTotal := newTotal - oldTotal

	sum := modelEff + inputEff
	if math.Abs(sum-deltaTotal) > 1e-9 {
		t.Fatalf("exact sum failed: modelEff (%.4f) + inputEff (%.4f) = %.4f, want %.4f",
			modelEff, inputEff, sum, deltaTotal)
	}
}

func TestDecomposeEffect_PureModelChange(t *testing.T) {
	// Scores remain exactly identical; only weights shift
	oldW := map[string]float64{"SUFF": 0.8, "COV": 0.2}
	newW := map[string]float64{"SUFF": 0.3, "COV": 0.7}
	oldS := map[string]float64{"SUFF": 90.0, "COV": 60.0}
	newS := map[string]float64{"SUFF": 90.0, "COV": 60.0}

	modelEff, inputEff := decomposeEffect(oldW, newW, oldS, newS)

	if math.Abs(inputEff) > 1e-9 {
		t.Errorf("expected inputEff == 0 for identical inputs, got %.4f", inputEff)
	}

	oldTotal := 0.8*90.0 + 0.2*60.0 // 84.0
	newTotal := 0.3*90.0 + 0.7*60.0 // 69.0
	expectedDelta := newTotal - oldTotal // -15.0

	if math.Abs(modelEff-expectedDelta) > 1e-9 {
		t.Errorf("expected modelEff == %.2f, got %.2f", expectedDelta, modelEff)
	}
}

func TestDecomposeEffect_PureInputDrift(t *testing.T) {
	// Weights remain identical; only scores drift
	weights := map[string]float64{"SUFF": 0.5, "COV": 0.5}
	oldS := map[string]float64{"SUFF": 80.0, "COV": 70.0}
	newS := map[string]float64{"SUFF": 90.0, "COV": 85.0}

	modelEff, inputEff := decomposeEffect(weights, weights, oldS, newS)

	if math.Abs(modelEff) > 1e-9 {
		t.Errorf("expected modelEff == 0 for identical weights, got %.4f", modelEff)
	}

	oldTotal := 0.5*80.0 + 0.5*70.0 // 75.0
	newTotal := 0.5*90.0 + 0.5*85.0 // 87.5
	expectedDelta := newTotal - oldTotal // +12.5

	if math.Abs(inputEff-expectedDelta) > 1e-9 {
		t.Errorf("expected inputEff == %.2f, got %.2f", expectedDelta, inputEff)
	}
}

func TestDecomposeEffect_DimensionAdded(t *testing.T) {
	// New dimension added to new weights that was absent in old weights
	oldW := map[string]float64{"SUFF": 1.0}
	newW := map[string]float64{"SUFF": 0.7, "LICENSING": 0.3}
	oldS := map[string]float64{"SUFF": 95.0}
	newS := map[string]float64{"SUFF": 95.0, "LICENSING": 50.0}

	modelEff, inputEff := decomposeEffect(oldW, newW, oldS, newS)

	oldTotal := 95.0
	newTotal := 0.7*95.0 + 0.3*50.0 // 66.5 + 15 = 81.5
	deltaTotal := newTotal - oldTotal // -13.5

	sum := modelEff + inputEff
	if math.Abs(sum-deltaTotal) > 1e-9 {
		t.Fatalf("exact sum failed with added dimension: model (%.4f) + input (%.4f) = %.4f, want %.4f",
			modelEff, inputEff, sum, deltaTotal)
	}
}

func TestNeighborGap_Edges(t *testing.T) {
	// Case 1: Single vendor -> gap is nil
	single := []float64{85.0}
	if gap := neighborGap(single, 0); gap != nil {
		t.Errorf("expected nil gap for single vendor, got %v", *gap)
	}

	// Case 2: Rank 1 vendor (index 0) in descending slice
	scores := []float64{95.0, 92.0, 80.0}
	gap0 := neighborGap(scores, 0)
	if gap0 == nil || math.Abs(*gap0-3.0) > 1e-9 {
		t.Errorf("rank 1 gap want 3.0, got %v", gap0)
	}

	// Case 3: Last rank vendor (index 2)
	gap2 := neighborGap(scores, 2)
	if gap2 == nil || math.Abs(*gap2-12.0) > 1e-9 {
		t.Errorf("last rank gap want 12.0, got %v", gap2)
	}

	// Case 4: Middle vendor (index 1) -> min(gap above (3.0), gap below (12.0)) = 3.0
	gap1 := neighborGap(scores, 1)
	if gap1 == nil || math.Abs(*gap1-3.0) > 1e-9 {
		t.Errorf("middle rank gap want 3.0, got %v", gap1)
	}

	// Case 5: Exact tie -> gap is 0.0
	tied := []float64{90.0, 90.0}
	gapTied := neighborGap(tied, 0)
	if gapTied == nil || *gapTied != 0.0 {
		t.Errorf("tied rank gap want 0.0, got %v", gapTied)
	}
}

func TestClassifyStability(t *testing.T) {
	gap02 := 0.2
	gap10 := 1.0

	// 1. STABLE: rankDelta == 0 and gap >= 0.5
	if s := classifyStability(0, &gap10); s != ShadowStable {
		t.Errorf("expected STABLE, got %s", s)
	}

	// 2. TIE_BOUND: gap < 0.5 regardless of rankDelta (even if rankDelta == 0)
	if s := classifyStability(0, &gap02); s != ShadowTieBound {
		t.Errorf("expected TIE_BOUND for delta 0 and gap 0.2, got %s", s)
	}

	// 3. TIE_BOUND: gap < 0.5 when rankDelta != 0
	if s := classifyStability(1, &gap02); s != ShadowTieBound {
		t.Errorf("expected TIE_BOUND for delta 1 and gap 0.2, got %s", s)
	}

	// 4. MOVED: rankDelta != 0 and gap >= 0.5
	if s := classifyStability(2, &gap10); s != ShadowMoved {
		t.Errorf("expected MOVED for delta 2 and gap 1.0, got %s", s)
	}

	// 5. Single vendor (gap == nil)
	if s := classifyStability(0, nil); s != ShadowStable {
		t.Errorf("expected STABLE for single vendor with delta 0, got %s", s)
	}
	if s := classifyStability(1, nil); s != ShadowMoved {
		t.Errorf("expected MOVED for single vendor with delta 1, got %s", s)
	}
}

func TestRunShadowValidation_NoOldBasis(t *testing.T) {
	eval := NewEvaluator()
	oldScores := map[string]float64{"BBG": 85.0}
	newProfiles := []VendorDimensionProfile{
		{VendorID: "BBG", CompositeQuality: 0.90},
	}
	vendorNames := map[string]string{"BBG": "Bloomberg"}

	// When called without options (or Decomposable == false)
	res := eval.RunShadowValidation("tenant-1", "Default", oldScores, newProfiles, vendorNames)

	if res.Basis.Decomposition != "UNAVAILABLE" {
		t.Errorf("expected Basis.Decomposition = UNAVAILABLE, got %s", res.Basis.Decomposition)
	}
	if len(res.Comparisons) != 1 {
		t.Fatalf("expected 1 comparison, got %d", len(res.Comparisons))
	}
	cmp := res.Comparisons[0]
	if cmp.Attribution != AttributionUnavailable {
		t.Errorf("expected Attribution = UNAVAILABLE, got %s", cmp.Attribution)
	}
	if cmp.ModelEffect != 0.0 {
		t.Errorf("expected ModelEffect = 0.0, got %.2f", cmp.ModelEffect)
	}
	if cmp.InputEffect != cmp.ScoreDelta {
		t.Errorf("expected InputEffect == ScoreDelta, got InputEffect=%.2f, ScoreDelta=%.2f", cmp.InputEffect, cmp.ScoreDelta)
	}
}

func TestShadowScaleNormalization(t *testing.T) {
	eval := NewEvaluator()
	oldScores := map[string]float64{"V1": 20.0} // e.g. on 0-25 scale
	newProfiles := []VendorDimensionProfile{
		{VendorID: "V1", CompositeQuality: 0.85}, // 85.0 on 0-100 scale
	}
	vendorNames := map[string]string{"V1": "Vendor 1"}

	opts := ShadowValidationOptions{
		ScaleFactor:  4.0, // Normalize 20.0 * 4 = 80.0
		Decomposable: false,
	}

	res := eval.RunShadowValidation("tenant-1", "Default", oldScores, newProfiles, vendorNames, opts)

	if len(res.Comparisons) != 1 {
		t.Fatalf("expected 1 comparison, got %d", len(res.Comparisons))
	}
	cmp := res.Comparisons[0]
	if cmp.OldScore != 20.0 {
		t.Errorf("expected OldScore = 20.0, got %.2f", cmp.OldScore)
	}
	if cmp.OldScoreNormalized != 80.0 {
		t.Errorf("expected OldScoreNormalized = 80.0, got %.2f", cmp.OldScoreNormalized)
	}
	if cmp.NewScoreNormalized != 85.0 {
		t.Errorf("expected NewScoreNormalized = 85.0, got %.2f", cmp.NewScoreNormalized)
	}
	if math.Abs(cmp.ScoreDelta-5.0) > 0.01 {
		t.Errorf("expected ScoreDelta = 5.0, got %.2f", cmp.ScoreDelta)
	}
	if cmp.Attribution != AttributionScaled {
		t.Errorf("expected Attribution = SCALED, got %s", cmp.Attribution)
	}
}

func TestRunShadowValidation_DegenerateOldRanking(t *testing.T) {
	eval := NewEvaluator()
	// Old scores all within 0.1 pts (spread = 0.09 < 0.5)
	oldScores := map[string]float64{
		"BBG": 0.24,
		"RFT": 0.23,
		"FDS": 0.21,
		"SPG": 0.19,
		"ICE": 0.15,
	}
	newProfiles := []VendorDimensionProfile{
		{VendorID: "FDS", CompositeQuality: 0.475},
		{VendorID: "ICE", CompositeQuality: 0.466},
		{VendorID: "BBG", CompositeQuality: 0.432},
		{VendorID: "RFT", CompositeQuality: 0.429},
		{VendorID: "SPG", CompositeQuality: 0.332},
	}
	vendorNames := map[string]string{
		"BBG": "Bloomberg", "RFT": "Refinitiv", "FDS": "FactSet", "ICE": "ICE", "SPG": "S&P",
	}

	res := eval.RunShadowValidation("tenant-1", "Default Procurement", oldScores, newProfiles, vendorNames)

	if res.Basis.OldRanking != "DEGENERATE" {
		t.Errorf("expected OldRanking = DEGENERATE, got %s", res.Basis.OldRanking)
	}
	if res.Basis.OldSpread >= 0.5 {
		t.Errorf("expected OldSpread < 0.5, got %.2f", res.Basis.OldSpread)
	}
	if !res.IsStable {
		t.Errorf("expected IsStable = true under degenerate old ranking")
	}

	for _, comp := range res.Comparisons {
		if comp.RankDelta != nil {
			t.Errorf("vendor %s: expected RankDelta = nil when old ranking is DEGENERATE, got %v",
				comp.VendorID, *comp.RankDelta)
		}
	}
}

func TestRunShadowValidation_DroppedFixtureVendors(t *testing.T) {
	eval := NewEvaluator()
	oldScores := map[string]float64{
		"BBG":                85.0,
		"TEST_VENDOR_MOCK":   10.0,
		"FIXTURE_SYNTH_01":   5.0,
	}
	newProfiles := []VendorDimensionProfile{
		{VendorID: "BBG", CompositeQuality: 0.88},
		{VendorID: "TEST_VENDOR_MOCK", CompositeQuality: 0.12},
		{VendorID: "FIXTURE_SYNTH_01", CompositeQuality: 0.05},
	}
	vendorNames := map[string]string{
		"BBG": "Bloomberg", "TEST_VENDOR_MOCK": "Test Mock", "FIXTURE_SYNTH_01": "Fixture",
	}

	res := eval.RunShadowValidation("tenant-1", "Default", oldScores, newProfiles, vendorNames)

	if len(res.Comparisons) != 1 {
		t.Fatalf("expected 1 valid comparison (BBG), got %d", len(res.Comparisons))
	}
	if res.Comparisons[0].VendorID != "BBG" {
		t.Errorf("expected comparison vendor to be BBG, got %s", res.Comparisons[0].VendorID)
	}

	if len(res.DroppedVendors) != 2 {
		t.Fatalf("expected 2 dropped vendors, got %d", len(res.DroppedVendors))
	}
	for _, d := range res.DroppedVendors {
		if d.Reason != "EXCLUDED_TEST_FIXTURE" {
			t.Errorf("expected Reason = EXCLUDED_TEST_FIXTURE, got %s", d.Reason)
		}
	}
}

func TestEvaluateVendorRanking(t *testing.T) {
	eval := NewEvaluator()

	radarScores := map[string]RadarScores{
		"V1": {SufficiencyRate: 90.0, CoverageRate: 95.0, SLAComplianceRate: 60.0, StabilityScore: 50.0, StewardFrictionCost: 40.0, RightsScore: 30.0},
		"V2": {SufficiencyRate: 50.0, CoverageRate: 60.0, SLAComplianceRate: 95.0, StabilityScore: 90.0, StewardFrictionCost: 85.0, RightsScore: 90.0},
	}
	vendorNames := map[string]string{"V1": "Vendor One", "V2": "Vendor Two"}
	costs := map[string]float64{"V1": 1000000.0, "V2": 800000.0}

	// Profile favoring Sufficiency & Coverage -> V1 should be Rank 1
	profA := WeightProfile{
		ProfileName: "Data Heavy",
		WeightSuff:  0.40,
		WeightCov:   0.40,
		WeightSLA:   0.05,
		WeightStab:  0.05,
		WeightOER:   0.05,
		WeightLic:   0.05,
	}

	rankingA := eval.EvaluateVendorRanking(radarScores, costs, vendorNames, profA)
	if len(rankingA) != 2 {
		t.Fatalf("expected 2 rankings, got %d", len(rankingA))
	}
	if rankingA[0].VendorID != "V1" {
		t.Errorf("expected V1 to be rank 1 under data-heavy profile, got %s", rankingA[0].VendorID)
	}
	if rankingA[0].Rank != 1 || rankingA[1].Rank != 2 {
		t.Errorf("expected ranks 1 and 2, got %d and %d", rankingA[0].Rank, rankingA[1].Rank)
	}

	// Profile favoring SLA & Stability & Licensing -> V2 should be Rank 1
	profB := WeightProfile{
		ProfileName: "Operations Heavy",
		WeightSuff:  0.05,
		WeightCov:   0.05,
		WeightSLA:   0.35,
		WeightStab:  0.25,
		WeightOER:   0.10,
		WeightLic:   0.20,
	}

	rankingB := eval.EvaluateVendorRanking(radarScores, costs, vendorNames, profB)
	if len(rankingB) != 2 {
		t.Fatalf("expected 2 rankings, got %d", len(rankingB))
	}
	if rankingB[0].VendorID != "V2" {
		t.Errorf("expected V2 to be rank 1 under operations-heavy profile, got %s", rankingB[0].VendorID)
	}
}

func TestComputeRankShifts(t *testing.T) {
	eval := NewEvaluator()

	rankingA := []VendorRankingEntry{
		{Rank: 1, VendorID: "V1", VendorName: "Vendor One", CompositeQualityScore: 85.0},
		{Rank: 2, VendorID: "V2", VendorName: "Vendor Two", CompositeQualityScore: 65.0},
	}
	rankingB := []VendorRankingEntry{
		{Rank: 1, VendorID: "V2", VendorName: "Vendor Two", CompositeQualityScore: 80.0},
		{Rank: 2, VendorID: "V1", VendorName: "Vendor One", CompositeQualityScore: 60.0},
	}

	shifts := eval.ComputeRankShifts(rankingA, rankingB)
	if len(shifts) != 2 {
		t.Fatalf("expected 2 shifts, got %d", len(shifts))
	}

	// V1 went from rank 1 to rank 2 -> delta = 1 - 2 = -1 (dropped)
	// V2 went from rank 2 to rank 1 -> delta = 2 - 1 = +1 (improved)
	var shiftV1, shiftV2 *VendorRankShift
	for i := range shifts {
		if shifts[i].VendorID == "V1" {
			shiftV1 = &shifts[i]
		} else if shifts[i].VendorID == "V2" {
			shiftV2 = &shifts[i]
		}
	}

	if shiftV1 == nil || shiftV2 == nil {
		t.Fatalf("missing shifts for V1 or V2")
	}
	if shiftV1.RankDelta != -1 {
		t.Errorf("expected V1 rank delta = -1, got %d", shiftV1.RankDelta)
	}
	if shiftV1.ScoreDelta != -25.0 {
		t.Errorf("expected V1 score delta = -25.0, got %.1f", shiftV1.ScoreDelta)
	}
	if shiftV2.RankDelta != 1 {
		t.Errorf("expected V2 rank delta = 1, got %d", shiftV2.RankDelta)
	}
	if shiftV2.ScoreDelta != 15.0 {
		t.Errorf("expected V2 score delta = 15.0, got %.1f", shiftV2.ScoreDelta)
	}
}

func TestEvaluateBundleImpact(t *testing.T) {
	eval := NewEvaluator()

	allVendors := []string{"BBG", "RFT"}
	vendorNames := map[string]string{"BBG": "Bloomberg", "RFT": "Refinitiv"}
	costs := map[string]float64{"BBG": 2140000.0, "RFT": 1180000.0}

	now := time.Now()
	candidates := []VendorCandidate{
		{EntityID: 1, AttributeCode: "LEI", VendorID: "BBG", NormalizedValue: "V1", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},
		{EntityID: 1, AttributeCode: "LEI", VendorID: "RFT", NormalizedValue: "V1", FormatOK: true, RangeOK: true, RefIntegrityOK: true, AsOfDate: now},
	}
	tolerances := map[string]AttributeTolerance{
		"LEI": {AttributeCode: "LEI", Tier: 1, MatchType: MatchExact, TierWeight: 1.0},
	}

	rankingsA := []VendorRankingEntry{
		{Rank: 1, VendorID: "RFT", CompositeQualityScore: 70.0},
		{Rank: 2, VendorID: "BBG", CompositeQualityScore: 65.0},
	}
	rankingsB := []VendorRankingEntry{
		{Rank: 1, VendorID: "BBG", CompositeQualityScore: 90.0},
		{Rank: 2, VendorID: "RFT", CompositeQualityScore: 60.0},
	}

	profA := WeightProfile{ProfileName: "Cost Balanced", WeightSuff: 0.166, WeightCov: 0.166, WeightSLA: 0.167, WeightStab: 0.167, WeightOER: 0.167, WeightLic: 0.167}
	profB := WeightProfile{ProfileName: "SLA Premium", WeightSuff: 0.10, WeightCov: 0.10, WeightSLA: 0.40, WeightStab: 0.20, WeightOER: 0.10, WeightLic: 0.10}

	impact := eval.EvaluateBundleImpact(allVendors, vendorNames, costs, candidates, tolerances, 1, rankingsA, rankingsB, profA, profB)

	if impact.Insight == "" {
		t.Errorf("expected non-empty insight in bundle impact")
	}
	if impact.ProfileAOptimal.Cost <= 0 {
		t.Errorf("expected positive cost for Profile A optimal bundle")
	}
}







