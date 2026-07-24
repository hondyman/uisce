package transaction

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ValidateTransaction performs DQ checks referencing semantic terms
func ValidateTransaction(t *TransactionMasterRecord) []error {
	var errs []error

	// RULE Tx_Required (Semantic Terms: st_transaction_id, st_portfolio_id, st_trade_date, st_transaction_type)
	if t.TransactionID == uuid.Nil {
		errs = append(errs, fmt.Errorf("transaction_id is required"))
	}
	if t.PortfolioID == uuid.Nil {
		errs = append(errs, fmt.Errorf("portfolio_id is required"))
	}
	if t.TradeDate.IsZero() {
		errs = append(errs, fmt.Errorf("trade_date is required"))
	}
	if t.TransactionType == "" {
		errs = append(errs, fmt.Errorf("transaction_type is required"))
	}

	// RULE Tx_QuantityValidity (Semantic Terms: st_tx_quantity, st_transaction_type)
	tradeTypes := map[string]bool{"BUY": true, "SELL": true, "SHORT": true, "COVER": true}
	if tradeTypes[t.TransactionType] {
		if t.Quantity == nil || t.Quantity.IsZero() {
			errs = append(errs, fmt.Errorf("trade transaction must have non-zero quantity"))
		}
		if t.Price == nil || t.Price.LessThanOrEqual(decimal.Zero) {
			errs = append(errs, fmt.Errorf("trade transaction must have positive price"))
		}
	}

	// RULE Tx_GrossAmountConsistency (Semantic Terms: st_gross_amount, st_tx_quantity, st_tx_price)
	if t.Quantity != nil && t.Price != nil && t.GrossAmount != nil {
		expected := t.Quantity.Mul(*t.Price)
		diff := t.GrossAmount.Sub(expected).Abs()
		if diff.GreaterThan(decimal.NewFromFloat(0.01)) {
			errs = append(errs, fmt.Errorf("gross amount inconsistent with quantity * price (diff: %s)", diff.String()))
		}
	}

	// RULE Tx_CurrencyValidity (Semantic Terms: st_transaction_currency)
	if len(t.TransactionCurrency) != 3 {
		errs = append(errs, fmt.Errorf("invalid transaction currency format"))
	}

	return errs
}
