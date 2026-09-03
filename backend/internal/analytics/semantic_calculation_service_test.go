package analytics

import (
	"testing"
)

// TestVectorizedExcelFormula and TestDetectVectorizedArguments used to
// cover executeExcelFormula/detectVectorizedArguments, removed 2026-09 as
// dead code (see semantic_calculation_service.go's replacement note) —
// they were never reachable from any real route and used a "nested array
// of argument sets = one calc per set" convention specific to that stub.
//
// The "excel_formula" path now goes through executeFormulaViaCalcEngine
// (see execute_formula_via_calc_engine_test.go), which evaluates one
// formula against one row series via the real boresolver/finlib engine.
// Running the same formula across many independent entities is now
// boresolver.HostRuntimeExecutor.Execute's job (tested in
// internal/boresolver/host_runtime_executor_test.go and
// sql_row_source_integration_test.go) — it batches across every entity a
// RowSource returns in one call, which is the "vectorized" replacement.

func TestExecuteCalculationRouting(t *testing.T) {
	service := &SemanticCalculationService{}

	// Test Cube routing
	cubeCalc := map[string]interface{}{
		"type":   "financial",
		"engine": "cube",
	}
	cubeAdapter := NewFinancialCalcAdapter(cubeCalc)
	cubeResult, err := service.ExecuteCalculation(cubeAdapter)
	if err != nil {
		t.Fatalf("Expected no error for cube calc, got: %v", err)
	}
	cubeResultMap, ok := cubeResult.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected cube result to be a map")
	}
	if cubeResultMap["engine"] != "cube" {
		t.Errorf("Expected engine to be 'cube', got: %v", cubeResultMap["engine"])
	}

	// Test Spark routing
	sparkCalc := map[string]interface{}{
		"type":   "financial",
		"engine": "spark",
	}
	sparkAdapter := NewFinancialCalcAdapter(sparkCalc)
	sparkResult, err := service.ExecuteCalculation(sparkAdapter)
	if err != nil {
		t.Fatalf("Expected no error for spark calc, got: %v", err)
	}
	sparkResultMap, ok := sparkResult.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected spark result to be a map")
	}
	if sparkResultMap["engine"] != "spark" {
		t.Errorf("Expected engine to be 'spark', got: %v", sparkResultMap["engine"])
	}

	// Test Internal routing (default)
	internalCalc := map[string]interface{}{
		"type":   "irr",
		"engine": "internal",
		"cash_flows": []interface{}{
			map[string]interface{}{"amount": -100.0, "period": 0.0},
			map[string]interface{}{"amount": 110.0, "period": 1.0},
		},
	}
	internalAdapter := NewFinancialCalcAdapter(internalCalc)
	internalResult, err := service.ExecuteCalculation(internalAdapter)
	if err != nil {
		t.Fatalf("Expected no error for internal calc, got: %v", err)
	}
	// Just verify it didn't error and returned something reasonable
	if internalResult == nil {
		t.Error("Expected internal result to be non-nil")
	}
}
