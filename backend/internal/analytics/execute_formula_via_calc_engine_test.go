package analytics

import "testing"

// TestExecuteFormulaViaCalcEngine_XIRR proves the replacement for the
// dead evaluateXIRR stub produces the correct, finlib-backed result (the
// old stub fell back to periodic IRR on irregular dates, which is wrong;
// this must match finlib's Actual/365 date-weighted solve).
func TestExecuteFormulaViaCalcEngine_XIRR(t *testing.T) {
	svc := &SemanticCalculationService{}

	calc := NewFinancialCalcAdapter(map[string]interface{}{
		"formula": "xirr(${cash_flows}, ${dates})",
		"arguments": map[string]interface{}{
			"cash_flows": []interface{}{-10000.0, 2750.0, 4250.0, 3250.0, 2750.0},
			"dates":      []interface{}{"2008-01-01", "2008-03-01", "2008-10-30", "2009-02-15", "2009-04-01"},
		},
	})

	result, err := svc.executeFormulaViaCalcEngine(calc)
	if err != nil {
		t.Fatalf("executeFormulaViaCalcEngine failed: %v", err)
	}

	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map[string]interface{}, got %T", result)
	}
	value, ok := m["result"].(float64)
	if !ok {
		t.Fatalf("expected result to be float64, got %T (%v)", m["result"], m["result"])
	}

	const expected = 0.373362535
	if diff := value - expected; diff > 1e-6 || diff < -1e-6 {
		t.Errorf("XIRR = %.9f, want %.9f", value, expected)
	}
}

// TestExecuteFormulaViaCalcEngine_ScalarBroadcast proves scalar arguments
// (not arrays) broadcast correctly alongside array arguments in the same
// formula.
func TestExecuteFormulaViaCalcEngine_ScalarBroadcast(t *testing.T) {
	svc := &SemanticCalculationService{}

	calc := NewFinancialCalcAdapter(map[string]interface{}{
		"formula": "${amount} * ${multiplier}",
		"arguments": map[string]interface{}{
			"amount":     100.0,
			"multiplier": 1.5,
		},
	})

	result, err := svc.executeFormulaViaCalcEngine(calc)
	if err != nil {
		t.Fatalf("executeFormulaViaCalcEngine failed: %v", err)
	}
	m := result.(map[string]interface{})
	value := m["result"].(float64)
	if value != 150.0 {
		t.Errorf("expected 150.0, got %v", value)
	}
}

func TestExecuteFormulaViaCalcEngine_RequiresFormula(t *testing.T) {
	svc := &SemanticCalculationService{}
	calc := NewFinancialCalcAdapter(map[string]interface{}{
		"arguments": map[string]interface{}{},
	})
	if _, err := svc.executeFormulaViaCalcEngine(calc); err == nil {
		t.Error("expected an error when formula is empty")
	}
}

func TestArgumentsToCalcRows_MismatchedLengthsRejected(t *testing.T) {
	_, err := argumentsToCalcRows(map[string]interface{}{
		"a": []interface{}{1.0, 2.0},
		"b": []interface{}{1.0, 2.0, 3.0},
	})
	if err == nil {
		t.Error("expected an error for mismatched array argument lengths")
	}
}
