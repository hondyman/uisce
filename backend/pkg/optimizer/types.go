package optimizer

import "time"

// DriftReport represents the current state of the portfolio versus its target
type DriftReport struct {
	PortfolioID   string
	Timestamp     time.Time
	DriftPercent  float64
	TrackingError float64
	Exposures     []Exposure // current vs target per symbol/sector
}

// Exposure represents the weight of a single asset or sector
type Exposure struct {
	Symbol      string
	CurrentWgt  float64
	TargetWgt   float64
	MarketValue float64
	Sector      string
}

// Lot represents a tax lot
type Lot struct {
	LotID         string
	Symbol        string
	Quantity      float64
	CostBasis     float64
	MarketPrice   float64
	AcquiredAt    time.Time
	AccountType   string // taxable, retirement
	UnrealizedPNL float64
	Term          string // short, long
}

// TaxRules defines the constraints and parameters for tax-aware optimization
type TaxRules struct {
	WashSaleDays            int     // e.g., 30
	ShortTermPenaltyWeight  float64 // penalize realizing short-term gains
	TransactionCostPerShare float64
	HarvestBudgetUSD        float64 // max allowed harvested losses
	MinTradeUSD             float64
	AllowedReplacementMap   map[string][]string // symbol -> allowed replacements (to avoid wash sale)
}

// CandidateTrade represents a proposed trade
type CandidateTrade struct {
	Side     string // BUY/SELL
	Symbol   string
	Qty      float64
	LotIDs   []string
	EstValue float64
	Reason   string
}

// Plan represents the output of the optimizer
type Plan struct {
	ID         string           `json:"plan_id"`
	Trades     []CandidateTrade `json:"trades"`
	TEAfter    float64          `json:"te_after"`
	TaxImpact  float64          `json:"tax_impact"` // negative = tax benefit
	TransCost  float64          `json:"trans_cost"`
	Confidence float64          `json:"confidence"`
	Citations  []string         `json:"citations"` // provenance ids/snapshots
	MonteCarlo MonteCarloSummary `json:"monte_carlo"`
}

// ScoreWeights defines the objective function weights
type ScoreWeights struct {
	TEWeight         float64 // higher reduces TE
	TaxAlphaWeight   float64 // higher rewards tax benefit
	TransCostWeight  float64 // penalize cost
	ShortTermPenalty float64 // additional penalty factor
}
