package marketdata

import (
	"context"
	"math/rand"
	"time"
)

// MockProvider generates synthetic market data
type MockProvider struct {
	basePrices map[string]float64
}

func NewMockProvider() *MockProvider {
	return &MockProvider{
		basePrices: map[string]float64{
			"AAPL": 150.0,
			"GOOG": 2800.0,
			"MSFT": 300.0,
			"TSLA": 900.0,
			"SPY":  450.0,
		},
	}
}

func (p *MockProvider) GetLatestPrice(ctx context.Context, symbol string) (*MarketQuote, error) {
	base, ok := p.basePrices[symbol]
	if !ok {
		base = 100.0 // Default for unknown symbols
	}

	// Random walk: Price +/- 1%
	change := (rand.Float64() - 0.5) * 0.02
	price := base * (1 + change)

	return &MarketQuote{
		Symbol:    symbol,
		Price:     price,
		Timestamp: time.Now(),
		Source:    "MOCK",
	}, nil
}

func (p *MockProvider) GetHistory(ctx context.Context, symbol string, start, end time.Time) ([]MarketQuote, error) {
	var quotes []MarketQuote
	base, ok := p.basePrices[symbol]
	if !ok {
		base = 100.0
	}

	currentPrice := base
	currentDate := start

	for currentDate.Before(end) {
		// Daily random walk
		change := (rand.Float64() - 0.5) * 0.02
		currentPrice = currentPrice * (1 + change)

		quotes = append(quotes, MarketQuote{
			Symbol:    symbol,
			Price:     currentPrice,
			Timestamp: currentDate,
			Source:    "MOCK",
		})

		currentDate = currentDate.AddDate(0, 0, 1)
	}

	return quotes, nil
}
