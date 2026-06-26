package position_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/position"
	"github.com/shopspring/decimal"
)

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func mustDec(s string) decimal.Decimal {
	d, _ := decimal.NewFromString(s)
	return d
}

func validPosition() *position.PositionMasterRecord {
	return &position.PositionMasterRecord{
		TenantID:         uuid.New(),
		PortfolioID:      uuid.New(),
		SecurityID:       uuid.New(),
		PositionDate:     time.Date(2026, 2, 22, 0, 0, 0, 0, time.UTC),
		PositionQuantity: mustDec("1000"),
		PositionSide:     "Long",
		PositionCurrency: "USD",
		PositionSource:   "Custodian",
		IsReconciled:     true,
	}
}

func validLot() *position.PositionLotRecord {
	return &position.PositionLotRecord{
		ID:              uuid.New(),
		TenantID:        uuid.New(),
		PositionID:      uuid.New(),
		AcquisitionDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		LotQuantity:     mustDec("500"),
		CostPerUnit:     mustDec("150.00"),
		TotalCostBasis:  mustDec("75000.00"),
		LotMethod:       "FIFO",
		IsClosed:        false,
	}
}

func validCash() *position.CashPositionRecord {
	return &position.CashPositionRecord{
		ID:            uuid.New(),
		TenantID:      uuid.New(),
		PortfolioID:   uuid.New(),
		CashCurrency:  "USD",
		ValueDate:     time.Date(2026, 2, 22, 0, 0, 0, 0, time.UTC),
		BalanceAmount: mustDec("125000.00"),
		CashSource:    "Custodian",
	}
}

// assertRule checks that a specific rule name has the expected pass/fail state.
func assertRule(t *testing.T, results []position.DQResult, ruleName string, wantPassed bool) {
	t.Helper()
	for _, r := range results {
		if r.RuleName == ruleName {
			if r.Passed != wantPassed {
				if wantPassed {
					t.Errorf("rule %q: expected PASS, got FAIL (error: %s)", ruleName, r.Error)
				} else {
					t.Errorf("rule %q: expected FAIL, got PASS", ruleName)
				}
			}
			return
		}
	}
	t.Errorf("rule %q not found in results", ruleName)
}

// ─────────────────────────────────────────────────────────────────────────────
// ValidatePosition tests
// ─────────────────────────────────────────────────────────────────────────────

func TestValidatePosition_Valid(t *testing.T) {
	pos := validPosition()
	results, hardFail := position.ValidatePosition(pos)
	if hardFail {
		t.Fatal("expected no hard fail for valid position")
	}
	for _, r := range results {
		if !r.Passed {
			t.Errorf("unexpected DQ failure: rule=%s error=%s", r.RuleName, r.Error)
		}
	}
}

func TestValidatePosition_RequiredFields(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(*position.PositionMasterRecord)
		ruleName string
	}{
		{
			name:     "missing portfolio_id",
			mutate:   func(p *position.PositionMasterRecord) { p.PortfolioID = uuid.Nil },
			ruleName: "Position_Required_PortfolioID",
		},
		{
			name:     "missing security_id",
			mutate:   func(p *position.PositionMasterRecord) { p.SecurityID = uuid.Nil },
			ruleName: "Position_Required_SecurityID",
		},
		{
			name:     "missing position_date",
			mutate:   func(p *position.PositionMasterRecord) { p.PositionDate = time.Time{} },
			ruleName: "Position_Required_PositionDate",
		},
		{
			name:     "zero quantity",
			mutate:   func(p *position.PositionMasterRecord) { p.PositionQuantity = decimal.Zero },
			ruleName: "Position_Required_Quantity",
		},
		{
			name:     "missing currency",
			mutate:   func(p *position.PositionMasterRecord) { p.PositionCurrency = "" },
			ruleName: "Position_Required_Currency",
		},
		{
			name:     "missing source",
			mutate:   func(p *position.PositionMasterRecord) { p.PositionSource = "" },
			ruleName: "Position_Required_Source",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pos := validPosition()
			tc.mutate(pos)
			results, hardFail := position.ValidatePosition(pos)
			if !hardFail {
				t.Error("expected hard fail")
			}
			assertRule(t, results, tc.ruleName, false)
		})
	}
}

func TestValidatePosition_SideSignConsistency(t *testing.T) {
	tests := []struct {
		name   string
		side   string
		qty    string
		wantOk bool
	}{
		{"long positive qty", "Long", "500", true},
		{"long negative qty", "Long", "-500", false},
		{"short positive qty", "Short", "500", false},
		{"short negative qty", "Short", "-500", true},
		{"net positive qty", "Net", "100", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pos := validPosition()
			pos.PositionSide = tc.side
			pos.PositionQuantity = mustDec(tc.qty)
			results, _ := position.ValidatePosition(pos)
			assertRule(t, results, "Position_SideSignConsistency", tc.wantOk)
		})
	}
}

func TestValidatePosition_InvalidSide(t *testing.T) {
	pos := validPosition()
	pos.PositionSide = "BULLISH"
	results, hardFail := position.ValidatePosition(pos)
	if hardFail {
		t.Error("invalid side should be soft, not hard fail")
	}
	assertRule(t, results, "Position_SideValidity", false)
}

func TestValidatePosition_ReconciliationAlert_AboveThreshold(t *testing.T) {
	pos := validPosition()
	diff := mustDec("0.05") // above 0.01 threshold
	pos.ReconciliationDiff = &diff
	results, hardFail := position.ValidatePosition(pos)
	if hardFail {
		t.Error("reconciliation alert should be soft, not hard fail")
	}
	assertRule(t, results, "Position_ReconciliationAlert", false)
}

func TestValidatePosition_ReconciliationAlert_BelowThreshold(t *testing.T) {
	pos := validPosition()
	diff := mustDec("0.005") // below 0.01 threshold — should NOT alert
	pos.ReconciliationDiff = &diff
	results, _ := position.ValidatePosition(pos)
	// Rule should not appear when diff is within tolerance
	for _, r := range results {
		if r.RuleName == "Position_ReconciliationAlert" && !r.Passed {
			t.Error("reconciliation alert should not fire for diff < 0.01")
		}
	}
}

func TestValidatePosition_NilReconciliationDiff(t *testing.T) {
	pos := validPosition()
	pos.ReconciliationDiff = nil
	_, hardFail := position.ValidatePosition(pos)
	if hardFail {
		t.Error("nil reconciliation diff should not cause hard fail")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// ValidateCashPosition tests
// ─────────────────────────────────────────────────────────────────────────────

func TestValidateCashPosition_Valid(t *testing.T) {
	cash := validCash()
	results, hardFail := position.ValidateCashPosition(cash)
	if hardFail {
		t.Fatal("expected no hard fail for valid cash position")
	}
	for _, r := range results {
		if !r.Passed {
			t.Errorf("unexpected DQ failure: rule=%s error=%s", r.RuleName, r.Error)
		}
	}
}

func TestValidateCashPosition_RequiredFields(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(*position.CashPositionRecord)
		ruleName string
	}{
		{
			name:     "missing portfolio_id",
			mutate:   func(c *position.CashPositionRecord) { c.PortfolioID = uuid.Nil },
			ruleName: "Cash_Required_PortfolioID",
		},
		{
			name:     "missing currency",
			mutate:   func(c *position.CashPositionRecord) { c.CashCurrency = "" },
			ruleName: "Cash_Required_Currency",
		},
		{
			name:     "missing value_date",
			mutate:   func(c *position.CashPositionRecord) { c.ValueDate = time.Time{} },
			ruleName: "Cash_Required_ValueDate",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cash := validCash()
			tc.mutate(cash)
			results, hardFail := position.ValidateCashPosition(cash)
			if !hardFail {
				t.Error("expected hard fail")
			}
			assertRule(t, results, tc.ruleName, false)
		})
	}
}

func TestValidateCashPosition_NegativeBalance_NonMargin(t *testing.T) {
	cash := validCash()
	cash.BalanceAmount = mustDec("-5000.00")
	cash.CashCurrency = "USD"
	results, hardFail := position.ValidateCashPosition(cash)
	if hardFail {
		t.Error("negative balance should be soft, not hard fail")
	}
	assertRule(t, results, "Cash_NegativeBalanceAlert", false)
}

func TestValidateCashPosition_NegativeBalance_MarginAllowed(t *testing.T) {
	cash := validCash()
	cash.BalanceAmount = mustDec("-5000.00")
	cash.CashCurrency = "MARGIN"
	results, _ := position.ValidateCashPosition(cash)
	assertRule(t, results, "Cash_NegativeBalanceAlert", true) // should pass for MARGIN
}

func TestValidateCashPosition_NegativeBalance_OverdraftAllowed(t *testing.T) {
	cash := validCash()
	cash.BalanceAmount = mustDec("-1000.00")
	cash.CashCurrency = "OVERDRAFT"
	results, _ := position.ValidateCashPosition(cash)
	assertRule(t, results, "Cash_NegativeBalanceAlert", true)
}

// ─────────────────────────────────────────────────────────────────────────────
// ValidatePositionLot tests
// ─────────────────────────────────────────────────────────────────────────────

func TestValidatePositionLot_Valid(t *testing.T) {
	lot := validLot()
	results, hardFail := position.ValidatePositionLot(lot)
	if hardFail {
		t.Fatal("expected no hard fail for valid lot")
	}
	for _, r := range results {
		if !r.Passed {
			t.Errorf("unexpected DQ failure: rule=%s error=%s", r.RuleName, r.Error)
		}
	}
}

func TestValidatePositionLot_InvalidMethod(t *testing.T) {
	methods := []struct {
		method string
		wantOk bool
	}{
		{"FIFO", true},
		{"LIFO", true},
		{"HIFO", true},
		{"Specific", true},
		{"AverageCost", true},
		{"RANDOM", false},
		{"fifo", false}, // case-sensitive
		{"", false},
	}

	for _, tc := range methods {
		t.Run("method="+tc.method, func(t *testing.T) {
			lot := validLot()
			lot.LotMethod = tc.method
			results, hardFail := position.ValidatePositionLot(lot)
			assertRule(t, results, "Lot_MethodValidity", tc.wantOk)
			if !tc.wantOk && !hardFail {
				t.Error("invalid lot method should trigger hard fail")
			}
		})
	}
}

func TestValidatePositionLot_ZeroQuantity(t *testing.T) {
	lot := validLot()
	lot.LotQuantity = decimal.Zero
	results, hardFail := position.ValidatePositionLot(lot)
	if !hardFail {
		t.Error("zero lot_quantity should trigger hard fail")
	}
	assertRule(t, results, "Lot_PositiveQuantity", false)
}

func TestValidatePositionLot_NegativeCostPerUnit(t *testing.T) {
	lot := validLot()
	lot.CostPerUnit = mustDec("-1.00")
	results, hardFail := position.ValidatePositionLot(lot)
	if !hardFail {
		t.Error("negative cost_per_unit should trigger hard fail")
	}
	assertRule(t, results, "Lot_CostPerUnitNonNegative", false)
}

func TestValidatePositionLot_CostBasisConsistency(t *testing.T) {
	tests := []struct {
		name           string
		qty            string
		costPerUnit    string
		totalCostBasis string
		wantOk         bool
	}{
		{
			name:           "exact match",
			qty:            "500",
			costPerUnit:    "150",
			totalCostBasis: "75000",
			wantOk:         true,
		},
		{
			name:           "within 0.001 tolerance",
			qty:            "333",
			costPerUnit:    "3",
			totalCostBasis: "999.0005", // 333*3=999, diff=0.0005 < 0.001
			wantOk:         true,
		},
		{
			name:           "exceeds tolerance",
			qty:            "100",
			costPerUnit:    "10",
			totalCostBasis: "1005.00", // 100*10=1000, diff=5 > 0.001
			wantOk:         false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lot := validLot()
			lot.LotQuantity = mustDec(tc.qty)
			lot.CostPerUnit = mustDec(tc.costPerUnit)
			lot.TotalCostBasis = mustDec(tc.totalCostBasis)
			results, _ := position.ValidatePositionLot(lot)
			assertRule(t, results, "Lot_CostBasisConsistency", tc.wantOk)
		})
	}
}

func TestValidatePositionLot_ClosedDateRequired(t *testing.T) {
	lot := validLot()
	lot.IsClosed = true
	lot.ClosedDate = nil // missing!
	results, _ := position.ValidatePositionLot(lot)
	assertRule(t, results, "Lot_ClosedDateRequired", false)
}

func TestValidatePositionLot_ClosedWithDate(t *testing.T) {
	lot := validLot()
	d := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	lot.IsClosed = true
	lot.ClosedDate = &d
	results, hardFail := position.ValidatePositionLot(lot)
	if hardFail {
		t.Error("closed lot with valid closed_date should not hard fail")
	}
	for _, r := range results {
		if r.RuleName == "Lot_ClosedDateRequired" && !r.Passed {
			t.Error("Lot_ClosedDateRequired should pass when closed_date is set")
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// DQSummary tests
// ─────────────────────────────────────────────────────────────────────────────

func TestDQSummary_SplitsCorrectly(t *testing.T) {
	results := []position.DQResult{
		{RuleName: "rule_a", Passed: true, Error: ""},
		{RuleName: "rule_b", Passed: false, Error: "field b is missing"},
		{RuleName: "rule_c", Passed: true, Error: ""},
		{RuleName: "rule_d", Passed: false, Error: "invalid value"},
	}

	passed, failed, errors := position.DQSummary(results)

	if len(passed) != 2 {
		t.Errorf("expected 2 passed, got %d", len(passed))
	}
	if len(failed) != 2 {
		t.Errorf("expected 2 failed, got %d", len(failed))
	}
	if len(errors) != 2 {
		t.Errorf("expected 2 errors, got %d", len(errors))
	}

	// Check specific values are in right buckets
	passSet := make(map[string]bool)
	for _, p := range passed {
		passSet[p] = true
	}
	if !passSet["rule_a"] || !passSet["rule_c"] {
		t.Error("expected rule_a and rule_c in passed")
	}

	failSet := make(map[string]bool)
	for _, f := range failed {
		failSet[f] = true
	}
	if !failSet["rule_b"] || !failSet["rule_d"] {
		t.Error("expected rule_b and rule_d in failed")
	}
}

func TestDQSummary_AllPassed(t *testing.T) {
	results := []position.DQResult{
		{RuleName: "r1", Passed: true},
		{RuleName: "r2", Passed: true},
	}
	_, failed, errors := position.DQSummary(results)
	if len(failed) != 0 {
		t.Errorf("expected 0 failed, got %d", len(failed))
	}
	if len(errors) != 0 {
		t.Errorf("expected 0 errors, got %d", len(errors))
	}
}

func TestDQSummary_AllFailed(t *testing.T) {
	results := []position.DQResult{
		{RuleName: "r1", Passed: false, Error: "err1"},
		{RuleName: "r2", Passed: false, Error: "err2"},
	}
	passed, failed, errors := position.DQSummary(results)
	if len(passed) != 0 {
		t.Errorf("expected 0 passed, got %d", len(passed))
	}
	if len(failed) != 2 {
		t.Errorf("expected 2 failed, got %d", len(failed))
	}
	if len(errors) != 2 {
		t.Errorf("expected 2 errors, got %d", len(errors))
	}
}

func TestDQSummary_Empty(t *testing.T) {
	passed, failed, errors := position.DQSummary(nil)
	if passed != nil || failed != nil || errors != nil {
		t.Error("expected nil slices for empty input")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Edge cases
// ─────────────────────────────────────────────────────────────────────────────

func TestValidatePosition_MultipleHardFailsReportedAll(t *testing.T) {
	pos := &position.PositionMasterRecord{} // all zeroed = all required fields missing
	results, hardFail := position.ValidatePosition(pos)
	if !hardFail {
		t.Fatal("expected hard fail when all required fields missing")
	}

	expectedRules := []string{
		"Position_Required_PortfolioID",
		"Position_Required_SecurityID",
		"Position_Required_PositionDate",
		"Position_Required_Quantity",
		"Position_Required_Currency",
		"Position_Required_Source",
	}

	failedSet := make(map[string]bool)
	for _, r := range results {
		if !r.Passed {
			failedSet[r.RuleName] = true
		}
	}
	for _, expected := range expectedRules {
		if !failedSet[expected] {
			t.Errorf("expected rule %q to fail", expected)
		}
	}
}

func TestValidatePosition_ShortPositionNegativeQty_IsValid(t *testing.T) {
	pos := validPosition()
	pos.PositionSide = "Short"
	pos.PositionQuantity = mustDec("-250")
	_, hardFail := position.ValidatePosition(pos)
	if hardFail {
		t.Error("SHORT with negative qty should be valid")
	}
}

func TestValidatePositionLot_PositiveQuantityEdge(t *testing.T) {
	lot := validLot()
	lot.LotQuantity = mustDec("0.0001") // tiny but positive
	results, hardFail := position.ValidatePositionLot(lot)
	if hardFail {
		t.Error("tiny positive quantity should not hard fail")
	}
	assertRule(t, results, "Lot_PositiveQuantity", true)
}
