package position

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	zero           = decimal.Zero
	reconThreshold = decimal.NewFromFloat(0.01)
)

// validLotMethods enumerates accepted lot matching methods.
var validLotMethods = map[string]bool{
	"FIFO": true, "LIFO": true, "HIFO": true, "Specific": true, "AverageCost": true,
}

// validSides enumerates valid position sides.
var validSides = map[string]bool{
	"Long": true, "Short": true, "Net": true,
}

// DQResult captures the outcome of a single DQ rule.
type DQResult struct {
	RuleName string
	Passed   bool
	Error    string
}

// ─────────────────────────────────────────────────────────────────────────────
// ValidatePosition validates a parsed PositionMasterRecord.
// Returns (results, hardFail) — hardFail=true means do not upsert.
// ─────────────────────────────────────────────────────────────────────────────
func ValidatePosition(p *PositionMasterRecord) ([]DQResult, bool) {
	var results []DQResult
	hardFail := false

	requireUUID := func(id uuid.UUID, field, rule string) {
		if id == uuid.Nil {
			results = append(results, DQResult{rule, false, field + " is required"})
			hardFail = true
		} else {
			results = append(results, DQResult{rule, true, ""})
		}
	}
	requireStr := func(v, field, rule string) {
		if v == "" {
			results = append(results, DQResult{rule, false, field + " is required"})
			hardFail = true
		} else {
			results = append(results, DQResult{rule, true, ""})
		}
	}

	requireUUID(p.PortfolioID, "portfolio_id", "Position_Required_PortfolioID")
	requireUUID(p.SecurityID, "security_id", "Position_Required_SecurityID")
	requireStr(p.PositionCurrency, "position_currency", "Position_Required_Currency")
	requireStr(p.PositionSource, "position_source", "Position_Required_Source")

	if p.PositionDate.IsZero() {
		results = append(results, DQResult{"Position_Required_PositionDate", false, "position_date is required"})
		hardFail = true
	} else {
		results = append(results, DQResult{"Position_Required_PositionDate", true, ""})
	}

	if p.PositionQuantity.IsZero() {
		results = append(results, DQResult{"Position_Required_Quantity", false, "position_quantity must not be zero"})
		hardFail = true
	} else {
		results = append(results, DQResult{"Position_Required_Quantity", true, ""})
	}

	// Side validation + sign consistency (non-fatal)
	side := p.PositionSide
	if side == "" {
		side = "Long"
	}
	if !validSides[side] {
		results = append(results, DQResult{"Position_SideValidity", false,
			fmt.Sprintf("position_side '%s' must be Long, Short, or Net", side)})
	} else {
		results = append(results, DQResult{"Position_SideValidity", true, ""})
	}

	if side == "Long" && p.PositionQuantity.LessThan(zero) {
		results = append(results, DQResult{"Position_SideSignConsistency", false,
			"Long position cannot have negative quantity"})
	} else if side == "Short" && p.PositionQuantity.GreaterThan(zero) {
		results = append(results, DQResult{"Position_SideSignConsistency", false,
			"Short position cannot have positive quantity"})
	} else {
		results = append(results, DQResult{"Position_SideSignConsistency", true, ""})
	}

	// Reconciliation alert (soft — never triggers hardFail)
	if p.ReconciliationDiff != nil && p.ReconciliationDiff.Abs().GreaterThan(reconThreshold) {
		results = append(results, DQResult{"Position_ReconciliationAlert", false,
			"reconciliation_diff " + p.ReconciliationDiff.String() + " exceeds threshold 0.01"})
	}

	return results, hardFail
}

// ─────────────────────────────────────────────────────────────────────────────
// ValidateCashPosition validates a cash position record.
// ─────────────────────────────────────────────────────────────────────────────
func ValidateCashPosition(c *CashPositionRecord) ([]DQResult, bool) {
	var results []DQResult
	hardFail := false

	if c.PortfolioID == uuid.Nil {
		results = append(results, DQResult{"Cash_Required_PortfolioID", false, "portfolio_id is required"})
		hardFail = true
	} else {
		results = append(results, DQResult{"Cash_Required_PortfolioID", true, ""})
	}
	if c.CashCurrency == "" {
		results = append(results, DQResult{"Cash_Required_Currency", false, "cash_currency is required"})
		hardFail = true
	} else {
		results = append(results, DQResult{"Cash_Required_Currency", true, ""})
	}
	if c.ValueDate.IsZero() {
		results = append(results, DQResult{"Cash_Required_ValueDate", false, "value_date is required"})
		hardFail = true
	} else {
		results = append(results, DQResult{"Cash_Required_ValueDate", true, ""})
	}

	// Negative balance alert — soft, only fails for non-margin currencies
	negative := c.BalanceAmount.LessThan(zero)
	allowedNeg := c.CashCurrency == "MARGIN" || c.CashCurrency == "OVERDRAFT"
	if negative && !allowedNeg {
		results = append(results, DQResult{"Cash_NegativeBalanceAlert", false,
			"negative cash balance " + c.BalanceAmount.String() + " for currency " + c.CashCurrency})
	} else {
		results = append(results, DQResult{"Cash_NegativeBalanceAlert", true, ""})
	}

	return results, hardFail
}

// ─────────────────────────────────────────────────────────────────────────────
// ValidatePositionLot validates a tax lot record.
// ─────────────────────────────────────────────────────────────────────────────
func ValidatePositionLot(l *PositionLotRecord) ([]DQResult, bool) {
	var results []DQResult
	hardFail := false

	if !validLotMethods[l.LotMethod] {
		results = append(results, DQResult{"Lot_MethodValidity", false,
			fmt.Sprintf("lot_method '%s' must be FIFO, LIFO, HIFO, Specific, or AverageCost", l.LotMethod)})
		hardFail = true
	} else {
		results = append(results, DQResult{"Lot_MethodValidity", true, ""})
	}

	if !l.LotQuantity.GreaterThan(zero) {
		results = append(results, DQResult{"Lot_PositiveQuantity", false, "lot_quantity must be positive"})
		hardFail = true
	} else {
		results = append(results, DQResult{"Lot_PositiveQuantity", true, ""})
	}

	if l.CostPerUnit.LessThan(zero) {
		results = append(results, DQResult{"Lot_CostPerUnitNonNegative", false, "cost_per_unit cannot be negative"})
		hardFail = true
	} else {
		results = append(results, DQResult{"Lot_CostPerUnitNonNegative", true, ""})
	}

	// Cost basis consistency: total_cost_basis ≈ quantity × cost_per_unit
	expected := l.LotQuantity.Mul(l.CostPerUnit)
	diff := l.TotalCostBasis.Sub(expected).Abs()
	if diff.GreaterThan(decimal.NewFromFloat(0.001)) {
		results = append(results, DQResult{"Lot_CostBasisConsistency", false,
			"total_cost_basis " + l.TotalCostBasis.String() + " != lot_quantity × cost_per_unit " + expected.String()})
	} else {
		results = append(results, DQResult{"Lot_CostBasisConsistency", true, ""})
	}

	// Lifecycle consistency
	if l.IsClosed && l.ClosedDate == nil {
		results = append(results, DQResult{"Lot_ClosedDateRequired", false, "closed_date is required when is_closed=true"})
	}

	return results, hardFail
}

// DQSummary extracts passed/failed rule names and error messages from DQ results.
func DQSummary(results []DQResult) (passed, failed, errors []string) {
	for _, r := range results {
		if r.Passed {
			passed = append(passed, r.RuleName)
		} else {
			failed = append(failed, r.RuleName)
			if r.Error != "" {
				errors = append(errors, r.Error)
			}
		}
	}
	return
}
