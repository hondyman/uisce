package main

import "time"

type Portfolio struct {
	ID                string                 `json:"id"`
	TenantID          string                 `json:"tenant_id"`
	Name              string                 `json:"name"`
	AUM               float64                `json:"aum"`
	Drift             float64                `json:"drift"`
	RiskScore         float64                `json:"risk_score,omitempty"`
	Alpha             float64                `json:"alpha,omitempty"`
	SectorAttribution map[string]interface{} `json:"sector_attribution,omitempty"`
	MitigationAction  string                 `json:"mitigation_action,omitempty"`
	Holdings          []Holding              `json:"holdings"`
	TargetModel       map[string]float64     `json:"target_model"`
	Constraints       RebalanceConstraints   `json:"constraints"`
	LastRebalance     time.Time              `json:"last_rebalance"`
	PolicyDocument    string                 `json:"policy_document,omitempty"`
}

type Holding struct {
	Symbol       string    `json:"symbol"`
	Shares       float64   `json:"shares"`
	CurrentPrice float64   `json:"current_price"`
	CostBasis    float64   `json:"cost_basis"`
	PurchaseDate time.Time `json:"purchase_date"`
	TaxLotID     string    `json:"tax_lot_id"`
	Sector       string    `json:"sector"`
}

type RebalanceConstraints struct {
	MaxTradeSize     float64  `json:"max_trade_size"`
	MinTradeSize     float64  `json:"min_trade_size"`
	MaxTurnover      float64  `json:"max_turnover"`
	TaxBudget        float64  `json:"tax_budget"`
	DriftTolerance   float64  `json:"drift_tolerance"`
	RestrictedList   []string `json:"restricted_list"`
	ESGPreference    string   `json:"esg_preference,omitempty"`
	RiskAppetite     string   `json:"risk_appetite,omitempty"`
	ForbiddenSectors []string `json:"forbidden_sectors,omitempty"`
}

type RebalancePlan struct {
	ID             string      `json:"id,omitempty"`
	PortfolioID    string      `json:"portfolio_id"`
	Timestamp      time.Time   `json:"timestamp"`
	CurrentDrift   float64     `json:"current_drift"`
	ExpectedDrift  float64     `json:"expected_drift"`
	ProposedTrades []Trade     `json:"proposed_trades"`
	TaxImpact      TaxAnalysis `json:"tax_impact"`
	Rationale      string      `json:"rationale"`
	Confidence     float64     `json:"confidence"`
	Status         string      `json:"status"`
	Summary        string      `json:"summary,omitempty"`
}

type Trade struct {
	Symbol         string  `json:"symbol"`
	Action         string  `json:"action"`
	Shares         float64 `json:"shares"`
	EstimatedPrice float64 `json:"estimated_price"`
	Notional       float64 `json:"notional"`
	TaxLotID       string  `json:"tax_lot_id"`
	Reason         string  `json:"reason"`
}

type TaxAnalysis struct {
	ShortTermGains float64 `json:"short_term_gains"`

	LongTermGains float64 `json:"long_term_gains"`

	TotalTax float64 `json:"total_tax"`

	TaxSavingsVsRandom float64 `json:"tax_savings_vs_random"`

	Strategy string `json:"strategy"`
}

// Simulation Engine Models

type SimulationParameters struct {
	PortfolioID string `json:"portfolio_id"`

	StartDate time.Time `json:"start_date"`

	EndDate time.Time `json:"end_date"`

	RebalanceFrequency string `json:"rebalance_frequency"` // e.g., "quarterly", "annually"

}

type SimulationResult struct {
	FinalPortfolioValue float64 `json:"final_portfolio_value"`

	BenchmarkValue float64 `json:"benchmark_value"`

	Trades []Trade `json:"trades"`

	PerformanceChart []map[string]float64 `json:"performance_chart"` // e.g., [{"date": 1672531200, "portfolio": 10000, "benchmark": 10000}]

}

type HistoricalPrice struct {
	Date time.Time `json:"date"`

	Price float64 `json:"price"`
}
