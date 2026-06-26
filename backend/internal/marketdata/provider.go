package marketdata

import (
	"context"
	"time"
)

// MarketQuote represents a price update
type MarketQuote struct {
	Symbol    string
	Price     float64
	Timestamp time.Time
	Source    string
}

// Provider defines the interface for fetching market data
type Provider interface {
	// GetLatestPrice returns the most recent price for a symbol
	GetLatestPrice(ctx context.Context, symbol string) (*MarketQuote, error)
	
	// GetHistory returns historical prices
	GetHistory(ctx context.Context, symbol string, start, end time.Time) ([]MarketQuote, error)
}
