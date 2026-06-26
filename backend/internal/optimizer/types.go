package optimizer

import (
	"time"
)

// DriftReport represents the detected drift in a portfolio.
type DriftReport struct {
	PortfolioID   string     `json:"portfolio_id"`
	TenantID      string     `json:"tenant_id"`
	DriftPercent  float64    `json:"drift_percent"`
	TrackingError float64    `json:"tracking_error"`
	Exposures     []Exposure `json:"exposures"`
	ComputedAt    time.Time  `json:"computed_at"`
}

// Exposure represents an individual position's deviation from target.
type Exposure struct {
	Symbol       string  `json:"symbol"`
	CurrentWgt   float64 `json:"current_wgt"`
	TargetWgt    float64 `json:"target_wgt"`
	MarketValue  float64 `json:"market_value"`
	DriftPercent float64 `json:"drift_percent"`
}

// Lot represents a tax lot for a position.
type Lot struct {
	LotID           string    `json:"lot_id"`
	Symbol          string    `json:"symbol"`
	Quantity        float64   `json:"quantity"`
	CostBasis       float64   `json:"cost_basis"`
	PurchaseDate    time.Time `json:"purchase_date"`
	UnrealizedPnL   float64   `json:"unrealized_pnl"`
	Term            string    `json:"term"` // "short" or "long"
	WashSaleDisable bool      `json:"wash_sale_disable"`
}

// TaxRules contains the tax policy configuration.
type TaxRules struct {
	SnapshotID        string    `json:"snapshot_id"`
	WashSaleDays      int       `json:"wash_sale_days"`
	ShortTermRate     float64   `json:"short_term_rate"`
	LongTermRate      float64   `json:"long_term_rate"`
	PreferLongTerm    bool      `json:"prefer_long_term"`
	HarvestThreshold  float64   `json:"harvest_threshold_usd"`
	TransactionCostBp int       `json:"transaction_cost_bp"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// ScoreWeights configures the optimizer's objective function weights.
type ScoreWeights struct {
	TEWeight        float64 `json:"te_weight"`
	TaxAlphaWeight  float64 `json:"tax_alpha_weight"`
	TransCostWeight float64 `json:"trans_cost_weight"`
}

// Inputs bundles all inputs for the optimizer.
type Inputs struct {
	Drift   DriftReport        `json:"drift"`
	Lots    []Lot              `json:"lots"`
	Prices  map[string]float64 `json:"prices"`
	Rules   TaxRules           `json:"rules"`
	Weights ScoreWeights       `json:"weights"`
}

// CandidateTrade represents a proposed trade in the plan.
type CandidateTrade struct {
	Side          string  `json:"side"` // "BUY" or "SELL"
	Symbol        string  `json:"symbol"`
	Qty           float64 `json:"qty"`
	EstValue      float64 `json:"est_value_usd"`
	Reason        string  `json:"reason"`
	LotID         string  `json:"lot_id,omitempty"`
	Term          string  `json:"term,omitempty"`
	UnrealizedPnL float64 `json:"unrealized_pnl,omitempty"`
}

// MonteCarloSummary contains simulation distribution statistics.
type MonteCarloSummary struct {
	MeanTaxImpact   float64 `json:"mean"`
	MedianTaxImpact float64 `json:"median"`
	Pct05           float64 `json:"pct05"`
	Pct95           float64 `json:"pct95"`
	Confidence80Min float64 `json:"confidence80_min"`
	Confidence80Max float64 `json:"confidence80_max"`
	Runs            int     `json:"runs"`
	Seed            int64   `json:"seed,omitempty"`
}

// Citation represents a data source used in generating the proposal.
type Citation struct {
	ID         string `json:"id"`
	Source     string `json:"source"`
	SnapshotID string `json:"snapshot_id"`
	Excerpt    string `json:"excerpt"`
}

// FactorSimilarity contains factor comparison data for UI visualization.
type FactorSimilarity struct {
	Target       FactorVector   `json:"target"`
	Replacements []FactorVector `json:"replacements"`
}

// FactorVector represents factor loadings for a symbol.
type FactorVector struct {
	Symbol  string    `json:"symbol"`
	Factors []float64 `json:"factors"` // [Size, Value, Momentum, Quality, Volatility]
}

// Plan represents a complete rebalancing proposal.
type Plan struct {
	ID               string            `json:"proposal_id"`
	PortfolioID      string            `json:"portfolio_id"`
	TenantID         string            `json:"tenant_id"`
	GeneratedAt      time.Time         `json:"generated_at"`
	Trades           []CandidateTrade  `json:"trades"`
	Explanation      string            `json:"explanation"`
	TEBefore         float64           `json:"tracking_error_before"`
	TEAfter          float64           `json:"tracking_error_after"`
	TaxImpact        float64           `json:"tax_impact_usd"`
	MonteCarlo       MonteCarloSummary `json:"monte_carlo"`
	Confidence       float64           `json:"confidence"`
	FactorSimilarity *FactorSimilarity `json:"factor_similarity,omitempty"`
	Citations        []Citation        `json:"citations"`
	Disclosures      []string          `json:"disclosures"`
}

// PolicyResult represents the outcome of a policy check.
type PolicyResult struct {
	OK      bool     `json:"ok"`
	Reasons []string `json:"reasons,omitempty"`
}

// AdvisorDecision represents an advisor's action on a proposal.
type AdvisorDecision struct {
	Action    string    `json:"action"` // "approve", "reject", "clarify"
	AdvisorID string    `json:"advisor_id"`
	Rationale string    `json:"rationale,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}
