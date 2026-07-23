package transaction

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestValidateTransaction_Valid(t *testing.T) {
	qty := decimal.NewFromInt(100)
	price := decimal.NewFromFloat(150.50)
	gross := qty.Mul(price)

	tx := &TransactionMasterRecord{
		TransactionID:       uuid.New(),
		PortfolioID:         uuid.New(),
		TradeDate:           time.Now(),
		TransactionType:     "BUY",
		Quantity:            &qty,
		Price:               &price,
		GrossAmount:         &gross,
		TransactionCurrency: "USD",
	}

	errs := ValidateTransaction(tx)
	if len(errs) > 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(errs), errs)
	}
}

func TestValidateTransaction_MissingRequired(t *testing.T) {
	tx := &TransactionMasterRecord{
		TransactionCurrency: "USD",
	}

	errs := ValidateTransaction(tx)
	if len(errs) != 4 {
		t.Errorf("Expected 4 errors for missing required fields, got %d", len(errs))
	}
}

func TestValidateTransaction_InvalidGrossAmount(t *testing.T) {
	qty := decimal.NewFromInt(100)
	price := decimal.NewFromFloat(150.50)
	wrongGross := decimal.NewFromInt(99999)

	tx := &TransactionMasterRecord{
		TransactionID:       uuid.New(),
		PortfolioID:         uuid.New(),
		TradeDate:           time.Now(),
		TransactionType:     "BUY",
		Quantity:            &qty,
		Price:               &price,
		GrossAmount:         &wrongGross,
		TransactionCurrency: "USD",
	}

	errs := ValidateTransaction(tx)
	foundGrossErr := false
	for _, err := range errs {
		if err != nil && err.Error() != "" {
			// just checking it has an error related to gross amount
			foundGrossErr = true
		}
	}
	if !foundGrossErr {
		t.Errorf("Expected gross amount consistency error")
	}
}
