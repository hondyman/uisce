package financial

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSuperpowersService_ResolveSymbology(t *testing.T) {
	svc := NewSuperpowersService(nil)

	res, err := svc.ResolveSymbology(context.Background(), SymbologyResolveRequest{
		IdentifierType:  "ISIN",
		IdentifierValue: "US0378331005",
	})

	assert.NoError(t, err)
	assert.Equal(t, "AAPL", res.PrimaryTicker)
	assert.Equal(t, "US0378331005", res.ISIN)
	assert.NotEmpty(t, res.FeedSurvivorshipRules)
}

func TestSuperpowersService_EvaluatePreTradeCompliance(t *testing.T) {
	svc := NewSuperpowersService(nil)

	// Clean order
	resClean, err := svc.EvaluatePreTradeCompliance(context.Background(), TradeOrder{
		PortfolioID:     "PORT-1",
		Symbol:          "AAPL",
		OrderType:       "BUY",
		Quantity:        100,
		LimitPrice:      150.0,
		EstimatedAmount: 15000.0,
	})

	assert.NoError(t, err)
	assert.True(t, resClean.Passed)
	assert.False(t, resClean.Blocked)

	// Restricted Order
	resRestricted, err := svc.EvaluatePreTradeCompliance(context.Background(), TradeOrder{
		PortfolioID:     "PORT-1",
		Symbol:          "RESTRICTED_XYZ",
		OrderType:       "BUY",
		Quantity:        100,
		LimitPrice:      150.0,
		EstimatedAmount: 15000.0,
	})

	assert.NoError(t, err)
	assert.False(t, resRestricted.Passed)
	assert.True(t, resRestricted.Blocked)
}

func TestSuperpowersService_PostTransaction(t *testing.T) {
	svc := NewSuperpowersService(nil)

	res, err := svc.PostTransaction(context.Background(), TradePostingRequest{
		PortfolioID: "PORT-1",
		Symbol:      "AAPL",
		Quantity:    100,
		Price:       200.0,
	})

	assert.NoError(t, err)
	assert.Len(t, res.IBORPostings, 2)
	assert.Len(t, res.ABORPostings, 3)
	assert.Equal(t, -20000.0, res.CashMovement)
}

func TestSuperpowersService_OptimizeHouseholdHarvesting(t *testing.T) {
	svc := NewSuperpowersService(nil)

	res, err := svc.OptimizeHouseholdHarvesting(context.Background(), "HH-SMITH-FAMILY")

	assert.NoError(t, err)
	assert.Len(t, res, 2)
	assert.Equal(t, "BND", res[0].TargetSymbol)
	assert.Equal(t, "AGG", res[0].SubstituteSymbol)
}
