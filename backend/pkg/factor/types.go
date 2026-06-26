package factor

// FactorVector represents a vector of factor loadings (e.g., [size, value, momentum, quality, vol])
type FactorVector []float64

// InstrumentMeta contains metadata and factor loadings for a financial instrument
type InstrumentMeta struct {
	Symbol            string
	Sector            string
	Industry          string
	LiquidityScore    float64 // 0..1 (higher is more liquid)
	TransCostPerShare float64
	Factor            FactorVector
}

// Universe holds the data required for replacement selection
type Universe struct {
	BySymbol    map[string]InstrumentMeta
	Correlation map[string]map[string]float64 // rho[s1][s2] in [-1,1]
	Prices      map[string]float64
}

// Constraints defines the limitations for replacement selection
type Constraints struct {
	Disallow             map[string]bool // symbols disallowed due to wash-sale or policy
	MinTradeUSD          float64
	MaxReplacements      int
	MaxPerReplacementUSD float64 // cap sizing for diversification
}

// ExposureTarget defines the desired exposure to preserve
type ExposureTarget struct {
	Symbol        string
	DesiredUSD    float64      // desired exposure to preserve (for replacement basket)
	SectorWeight  float64      // optional sector target
	FactorWeights FactorVector // target factor profile (optional)
}

// ReplacementTrade represents a selected replacement trade
type ReplacementTrade struct {
	Symbol string
	Qty    float64
	USD    float64
	Score  float64
	Reason string
}
