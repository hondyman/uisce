package cash

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ValidateCashLedger performs DQ checks referencing semantic terms
// Aligns with Whitepaper §7: Rules reference semantic terms, not columns
func ValidateCashLedger(l *CashLedgerEntryRecord) []error {
	var errs []error

	// RULE CashLedger_Required (Semantic Terms: st_portfolio_id, st_currency, st_value_date, st_cash_event_type, st_amount)
	if l.PortfolioID == uuid.Nil {
		errs = append(errs, fmt.Errorf("portfolio_id is required"))
	}
	if l.Currency == "" {
		errs = append(errs, fmt.Errorf("currency is required"))
	}
	if l.ValueDate.IsZero() {
		errs = append(errs, fmt.Errorf("value_date is required"))
	}
	if l.CashEventType == "" {
		errs = append(errs, fmt.Errorf("cash_event_type is required"))
	}
	if l.Amount.IsZero() {
		errs = append(errs, fmt.Errorf("amount cannot be zero"))
	}

	// RULE CashLedger_AmountValidity
	if l.Amount.IsZero() {
		errs = append(errs, fmt.Errorf("cash ledger amount cannot be zero"))
	}

	// RULE CashLedger_SignConvention
	inflowTypes := map[string]bool{"CONTRIBUTION": true, "INCOME": true, "FX_INFLOW": true}
	outflowTypes := map[string]bool{"WITHDRAWAL": true, "FEE": true, "COMMISSION": true, "TAX": true, "FX_OUTFLOW": true}

	if inflowTypes[l.CashEventType] && l.Amount.LessThan(decimal.Zero) {
		errs = append(errs, fmt.Errorf("expected positive amount for inflow event type %s", l.CashEventType))
	}
	if outflowTypes[l.CashEventType] && l.Amount.GreaterThan(decimal.Zero) {
		errs = append(errs, fmt.Errorf("expected negative amount for outflow event type %s", l.CashEventType))
	}

	// Currency format validation
	if len(l.Currency) != 3 {
		errs = append(errs, fmt.Errorf("invalid currency format"))
	}

	return errs
}

// ValidateCashBalance performs DQ checks on CashBalanceRecord
func ValidateCashBalance(b *CashBalanceRecord) []error {
	var errs []error

	// RULE CashBalance_Required
	if b.PortfolioID == uuid.Nil {
		errs = append(errs, fmt.Errorf("portfolio_id is required"))
	}
	if b.Currency == "" {
		errs = append(errs, fmt.Errorf("currency is required"))
	}
	if b.ValuationDate.IsZero() {
		errs = append(errs, fmt.Errorf("valuation_date is required"))
	}

	// RULE CashBalance_ClosingConsistency
	if b.OpeningBalance != nil && b.CashInflows != nil && b.CashOutflows != nil &&
		b.InterestAccrual != nil && b.FXEffect != nil && b.ClosingBalance != nil {
		expected := b.OpeningBalance.Add(*b.CashInflows).Sub(*b.CashOutflows).Add(*b.InterestAccrual).Add(*b.FXEffect)
		diff := b.ClosingBalance.Sub(expected).Abs()
		if diff.GreaterThan(decimal.NewFromFloat(0.01)) {
			errs = append(errs, fmt.Errorf("closing balance inconsistent with components (diff: %s)", diff.String()))
		}
	}

	return errs
}

// ValidateTransactionCashMapping validates Transaction → Cash Ledger mappings
func ValidateTransactionCashMapping(m *TransactionCashMapping) []error {
	var errs []error

	if m.TransactionID == uuid.Nil {
		errs = append(errs, fmt.Errorf("transaction_id is required"))
	}
	if m.Amount.IsZero() {
		errs = append(errs, fmt.Errorf("mapping amount cannot be zero"))
	}
	if len(m.Currency) != 3 {
		errs = append(errs, fmt.Errorf("invalid currency format"))
	}

	validMappingTypes := map[string]bool{
		"SETTLEMENT": true, "COMMISSION": true, "FEE": true, "TAX": true, "INCOME": true,
	}
	if !validMappingTypes[m.MappingType] {
		errs = append(errs, fmt.Errorf("invalid mapping type: %s", m.MappingType))
	}

	return errs
}
