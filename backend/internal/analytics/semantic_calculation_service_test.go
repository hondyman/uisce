package analytics

import (
	"testing"
)

func TestVectorizedExcelFormula(t *testing.T) {
	service := &SemanticCalculationService{}

	// Test vectorized XIRR calculation
	calc := map[string]interface{}{
		"type":    "excel_formula",
		"formula": "=XIRR({cash_flows}, {dates})",
		"arguments": map[string]interface{}{
			"cash_flows": []interface{}{
				[]interface{}{-1000.0, 200.0, 300.0, 400.0, 500.0},  // Portfolio 1
				[]interface{}{-2000.0, 400.0, 600.0, 800.0, 1000.0}, // Portfolio 2
			},
			"dates": []interface{}{
				[]interface{}{1.0, 2.0, 3.0, 4.0, 5.0}, // Dates for portfolio 1
				[]interface{}{1.0, 2.0, 3.0, 4.0, 5.0}, // Dates for portfolio 2
			},
		},
	}

	adapter := NewFinancialCalcAdapter(calc)
	result, err := service.executeExcelFormula(adapter)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected result to be a map, got: %T", result)
	}

	if resultMap["calculation_type"] != "excel_formula_vectorized" {
		t.Errorf("Expected calculation_type to be 'excel_formula_vectorized', got: %v", resultMap["calculation_type"])
	}

	results, ok := resultMap["results"].([]interface{})
	if !ok {
		t.Fatalf("Expected results to be an array, got: %T", resultMap["results"])
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got: %d", len(results))
	}

	// Check that each result is a number (XIRR calculation result)
	for i, res := range results {
		if _, ok := res.(float64); !ok {
			t.Errorf("Expected result %d to be a float64, got: %T", i, res)
		}
	}
}

func TestDetectVectorizedArguments(t *testing.T) {
	service := &SemanticCalculationService{}

	// Test detection of vectorized arguments
	args := map[string]interface{}{
		"cash_flows": []interface{}{
			[]interface{}{-1000.0, 200.0},
			[]interface{}{-2000.0, 400.0},
		},
		"rate": 0.1, // Scalar value
	}

	vectorizedArgs, isVectorized := service.detectVectorizedArguments(args)

	if !isVectorized {
		t.Error("Expected arguments to be detected as vectorized")
	}

	if len(vectorizedArgs) != 2 {
		t.Errorf("Expected 2 vectorized argument sets, got: %d", len(vectorizedArgs))
	}

	// Check first argument set
	if len(vectorizedArgs[0]["cash_flows"].([]interface{})) != 2 {
		t.Error("First argument set cash_flows should have 2 elements")
	}

	// Check that scalar values are replicated
	if vectorizedArgs[0]["rate"] != 0.1 || vectorizedArgs[1]["rate"] != 0.1 {
		t.Error("Scalar rate should be replicated to all argument sets")
	}
}

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
