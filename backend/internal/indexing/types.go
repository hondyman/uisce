package indexing

import (
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/values"
)

// BenchmarkIndex represents a market index (e.g., S&P 500)
type BenchmarkIndex struct {
	ID           string             `json:"id"`
	Name         string             `json:"name"`
	Symbol       string             `json:"symbol"`
	Constituents []IndexConstituent `json:"constituents"`
	LastUpdated  time.Time          `json:"last_updated"`
}

// IndexConstituent represents a single stock in an index
type IndexConstituent struct {
	Ticker string  `json:"ticker"`
	Name   string  `json:"name"`
	Weight float64 `json:"weight"` // Percentage (0.0 to 1.0)
	Sector string  `json:"sector"`
	Region string  `json:"region"`
}

// Portfolio represents a client's actual holdings
type Portfolio struct {
	ID              uuid.UUID `json:"id"`
	ClientID        string    `json:"client_id"`
	BenchmarkID     string    `json:"benchmark_id"`
	ValuesProfileID uuid.UUID `json:"values_profile_id"`
	Holdings        []Holding `json:"holdings"`
	Cash            float64   `json:"cash"`
	LastRebalanced  time.Time `json:"last_rebalanced"`
}

// Holding represents a specific asset held in a portfolio
type Holding struct {
	Ticker string  `json:"ticker"`
	Shares float64 `json:"shares"`
	Value  float64 `json:"value"`
	Weight float64 `json:"weight"` // Percentage of total portfolio value
}

// OrderType represents Buy or Sell
type OrderType string

const (
	OrderTypeBuy  OrderType = "BUY"
	OrderTypeSell OrderType = "SELL"
)

// Order represents a trade to be executed
type Order struct {
	ID          uuid.UUID `json:"id"`
	PortfolioID uuid.UUID `json:"portfolio_id"`
	Ticker      string    `json:"ticker"`
	Type        OrderType `json:"type"`
	Quantity    float64   `json:"quantity"` // Number of shares
	Amount      float64   `json:"amount"`   // Estimated dollar amount
	Reason      string    `json:"reason"`   // e.g., "Rebalance", "Values Violation"
	Status      string    `json:"status"`   // PENDING, EXECUTED, FAILED
	CreatedAt   time.Time `json:"created_at"`
}

// RebalanceRequest encapsulates the input for a rebalance operation
type RebalanceRequest struct {
	PortfolioID   uuid.UUID
	Benchmark     BenchmarkIndex
	ValuesProfile values.ClientValuesProfile
	Constraints   []values.Constraint
	Signals       []values.ValueSignal // Relevant signals for the universe
}
