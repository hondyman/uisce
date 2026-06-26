package benchmark

import (
"time"
)

// BenchmarkType defines the type of benchmark
type BenchmarkType string

const (
MarketIndexBenchmark   BenchmarkType = "market_index"
BlendedBenchmark       BenchmarkType = "blended"
CustomBenchmark        BenchmarkType = "custom"
PeerGroupBenchmark     BenchmarkType = "peer_group"
AbsoluteReturnBenchmark BenchmarkType = "absolute_return"
LiabilityDrivenBenchmark BenchmarkType = "liability_driven"
)

// Benchmark represents a benchmark definition
type Benchmark struct {
	ID              string
	Name            string
	Description     string
	Type            BenchmarkType
	Currency        string
	InceptionDate   time.Time
	IsActive        bool
	
	// For blended benchmarks
	Components      []BenchmarkComponent
	RebalanceFreq   RebalanceFrequency
	
	// For custom benchmarks
	CustomRules     *CustomBenchmarkRules
	
	// For absolute return
	TargetReturn    float64
	
	// Metadata
	CreatedAt       time.Time
	UpdatedAt       time.Time
	CreatedBy       string
}

// BenchmarkComponent represents a component in a blended benchmark
type BenchmarkComponent struct {
	BenchmarkID     string
	BenchmarkName   string
	Weight          float64
	AssetClass      string
	Sector          string
	Region          string
	EffectiveDate   time.Time
	ExpirationDate  time.Time
}

// RebalanceFrequency defines when blended benchmarks rebalance
type RebalanceFrequency string

const (
RebalanceDaily     RebalanceFrequency = "daily"
RebalanceMonthly   RebalanceFrequency = "monthly"
RebalanceQuarterly RebalanceFrequency = "quarterly"
RebalanceAnnual    RebalanceFrequency = "annual"
RebalanceNone      RebalanceFrequency = "none" // Buy and hold
)

// CustomBenchmarkRules defines rules for custom benchmark construction
type CustomBenchmarkRules struct {
	Universe          []UniverseRule
	WeightingScheme   WeightingScheme
	Constraints       []BenchmarkConstraint
	ScreeningCriteria []ScreeningCriterion
	ReconstitutionFreq RebalanceFrequency
}

// UniverseRule defines how to select the benchmark universe
type UniverseRule struct {
	RuleType    string // "asset_class", "sector", "market_cap", "region", "custom"
	Operator    string // "equals", "in", "not_in", "greater_than", "less_than"
	Value       interface{}
	Priority    int
}

// WeightingScheme defines how securities are weighted
type WeightingScheme string

const (
MarketCapWeight    WeightingScheme = "market_cap"
EqualWeight        WeightingScheme = "equal"
FundamentalWeight  WeightingScheme = "fundamental"
RiskParityWeight   WeightingScheme = "risk_parity"
MinVarianceWeight  WeightingScheme = "min_variance"
MaxDiversification WeightingScheme = "max_diversification"
CustomWeight       WeightingScheme = "custom"
)

// BenchmarkConstraint defines constraints on benchmark construction
type BenchmarkConstraint struct {
	ConstraintType string  // "max_weight", "min_weight", "sector_cap", "country_cap"
	Target         string  // What the constraint applies to
	Value          float64
}

// ScreeningCriterion defines inclusion/exclusion criteria
type ScreeningCriterion struct {
	Field      string // "market_cap", "volume", "esg_score", "dividend_yield"
	Operator   string // "greater_than", "less_than", "equals", "between"
	Value      interface{}
	IsExclude  bool
}

// BenchmarkReturn represents benchmark returns for a period
type BenchmarkReturn struct {
	BenchmarkID   string
	Date          time.Time
	ReturnValue   float64
	ReturnType    ReturnType
	CumulativeReturn float64
	AnnualizedReturn float64
}

// ReturnType defines the type of return
type ReturnType string

const (
TotalReturn      ReturnType = "total"
PriceReturn      ReturnType = "price"
IncomeReturn     ReturnType = "income"
GrossReturn      ReturnType = "gross"
NetReturn        ReturnType = "net"
)

// BenchmarkComparison holds comparison between portfolio and benchmark
type BenchmarkComparison struct {
	PortfolioID       string
	BenchmarkID       string
	BenchmarkName     string
	AsOfDate          time.Time
	Period            string
	
	// Returns
	PortfolioReturn   float64
	BenchmarkReturn   float64
	ActiveReturn      float64 // Portfolio - Benchmark
	
	// Risk metrics
	PortfolioVol      float64
	BenchmarkVol      float64
	TrackingError     float64
	InformationRatio  float64
	Beta              float64
	Alpha             float64
	RSqaured          float64
	
	// Return periods
	ReturnPeriods     []PeriodComparison
	
	// Holdings comparison
	ActiveWeights     []ActiveWeight
	SectorDeviation   map[string]float64
}

// PeriodComparison compares returns over standard periods
type PeriodComparison struct {
	Period            string // "1D", "1W", "1M", "3M", "6M", "YTD", "1Y", "3Y", "5Y", "10Y", "ITD"
	PortfolioReturn   float64
	BenchmarkReturn   float64
	ActiveReturn      float64
	TrackingError     float64
}

// ActiveWeight shows over/underweight vs benchmark
type ActiveWeight struct {
	SecurityID       string
	SecurityName     string
	AssetClass       string
	Sector           string
	PortfolioWeight  float64
	BenchmarkWeight  float64
	ActiveWeight     float64
	ContributionToTE float64 // Contribution to tracking error
}

// BenchmarkHolding represents a security in a benchmark
type BenchmarkHolding struct {
	BenchmarkID    string
	SecurityID     string
	SecurityName   string
	AssetClass     string
	Sector         string
	Country        string
	Currency       string
	Weight         float64
	MarketCap      float64
	EffectiveDate  time.Time
}

// BenchmarkStatistics contains benchmark analytics
type BenchmarkStatistics struct {
	BenchmarkID          string
	AsOfDate             time.Time
	
	// Returns
	DailyReturn          float64
	MTDReturn            float64
	QTDReturn            float64
	YTDReturn            float64
	OneYearReturn        float64
	ThreeYearReturn      float64
	FiveYearReturn       float64
	TenYearReturn        float64
	SinceInceptionReturn float64
	
	// Risk
	Volatility           float64
	SharpeRatio          float64
	SortinoRatio         float64
	MaxDrawdown          float64
	CalmarRatio          float64
	
	// Characteristics
	NumberOfHoldings     int
	AverageMarketCap     float64
	MedianMarketCap      float64
	DividendYield        float64
	PERatio              float64
	PBRatio              float64
	
	// Sector allocation
	SectorWeights        map[string]float64
	CountryWeights       map[string]float64
	CurrencyWeights      map[string]float64
}
