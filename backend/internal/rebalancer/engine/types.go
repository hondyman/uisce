package engine

// DriftReport represents the analysis of portfolio drift
type DriftReport struct {
	PortfolioID   string
	DriftPercent  float64
	TrackingError float64
	Exposures     []Exposure
}

// Exposure represents a factor or sector exposure
type Exposure struct {
	Symbol      string
	CurrentWgt  float64
	TargetWgt   float64
	MarketValue float64
}

// Plan represents the proposed rebalancing actions
type Plan struct {
	ID          string
	PortfolioID string
	Explanation string
	TEBefore    float64
	TEAfter     float64
	TaxImpact   float64
	Trades      []Trade
	Citations   []Citation
	MonteCarlo  MonteCarloSummary
	Confidence  float64
}

// Trade represents a single buy or sell order
type Trade struct {
	Side        string // "BUY" or "SELL"
	Symbol      string
	Qty         float64
	EstValueUSD float64
	Reason      string
}

// Citation represents a reference to a data snapshot
type Citation struct {
	ID         string
	Source     string
	SnapshotID string
	Excerpt    string
}

// MonteCarloSummary holds the results of the tax impact simulation
type MonteCarloSummary struct {
	MeanTaxImpact   float64
	MedianTaxImpact float64
	Pct05           float64
	Pct95           float64
	Confidence80Min float64
	Confidence80Max float64
	Runs            int
}
