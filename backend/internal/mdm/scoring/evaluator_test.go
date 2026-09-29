package scoring

import (
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
