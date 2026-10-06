package drift

import (
	"testing"
)

func TestASTSemanticDiff_StructureAware(t *testing.T) {
	oldAST := map[string]interface{}{
		"operator": "AND",
		"conditions": []interface{}{
			map[string]interface{}{
				"field":    "position.single_issuer_weight",
				"operator": "LTE",
				"value":    0.10,
			},
			map[string]interface{}{
				"field":    "account.jurisdiction",
				"operator": "NEQ",
				"value":    "LUX",
			},
		},
	}

	newAST := map[string]interface{}{
		"operator": "AND",
		"conditions": []interface{}{
			map[string]interface{}{
				"field":    "position.single_issuer_weight",
				"operator": "LTE",
				"value":    0.08, // Tightened threshold
			},
			map[string]interface{}{
				"field":    "account.jurisdiction",
				"operator": "NEQ",
				"value":    "LUX",
			},
			map[string]interface{}{
				"field":    "security.esg_score", // New condition branch
				"operator": "GTE",
				"value":    60,
			},
		},
	}

	diff := CompareAST(oldAST, newAST)

	if len(diff.DiffEntries) != 2 {
		t.Fatalf("Expected 2 diff entries, got %d:\n%s", len(diff.DiffEntries), diff.SummaryText)
	}

	// Verify Threshold Change detected
	if diff.DiffEntries[0].Type != DiffThresholdChange {
		t.Errorf("Diff 0 expected THRESHOLD_CHANGED, got %s", diff.DiffEntries[0].Type)
	}

	// Verify Node Added detected
	if diff.DiffEntries[1].Type != DiffNodeAdded {
		t.Errorf("Diff 1 expected NODE_ADDED, got %s", diff.DiffEntries[1].Type)
	}

	if !diff.HasBreakingChanges {
		t.Errorf("Expected HasBreakingChanges=true due to new condition branch")
	}

	t.Logf("Generated AST Diff Summary:\n%s", diff.SummaryText)
}

func TestExecuteTestCorpus_GoldenVectorAssertions(t *testing.T) {
	service := &RepinService{}

	testCases := []ScenarioTestCase{
		{
			CaseID:      "TC-01-PASS",
			Description: "Weight within 5% limit",
			MetricSnapshots: map[string]interface{}{
				"proposedWeight":   0.045,
				"maxAllowedWeight": 0.050,
			},
			ExpectedPassed: true,
		},
		{
			CaseID:      "TC-02-BREACH",
			Description: "Weight breaches 5% limit",
			MetricSnapshots: map[string]interface{}{
				"proposedWeight":   0.055,
				"maxAllowedWeight": 0.050,
			},
			ExpectedPassed: false,
		},
	}

	res := service.ExecuteTestCorpus(testCases)
	if !res.AllPassed {
		t.Fatalf("Expected all test cases to pass assertions, failed: %v", res.FailureNotes)
	}
	if res.PassedCases != 2 || res.FailedCases != 0 {
		t.Errorf("Mismatch in case counts: passed=%d, failed=%d", res.PassedCases, res.FailedCases)
	}
}
