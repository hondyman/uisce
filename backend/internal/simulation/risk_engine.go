package simulation

import (
	"context"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

// RiskEngine defines the interface for risk calculations
type RiskEngine interface {
	ComputeRisk(ctx context.Context, req RiskRequest) (*RiskResponse, error)
}

// CalculationEngine defines the interface for basic financial metrics
type CalculationEngine interface {
	ComputeMetrics(ctx context.Context, req MetricRequest) (*MetricResponse, error)
}

// PositionInput defines an asset holding for risk calculation
type PositionInput struct {
	AssetID  string  `json:"assetId"`
	Quantity float64 `json:"quantity"`
}

// MarketSnapshot defines the market environment for risk calculation
type MarketSnapshot struct {
	SpotPrices   map[string]float64 `json:"spotPrices,omitempty"`
	YieldCurves  map[string]any     `json:"yieldCurves,omitempty"`
	VolSurfaces  map[string]any     `json:"volSurfaces,omitempty"`
	ScenarioDate time.Time          `json:"scenarioDate"`
}

// RiskRequest encapsulates parameters for a risk calculation
type RiskRequest struct {
	TenantID    string
	PortfolioID string
	HorizonDays int
	AsOf        time.Time
	Positions   []PositionInput
	MarketData  MarketSnapshot
	ModelConfig map[string]any
	Shocks      *MarketShock `json:"shocks,omitempty"`
}

// RiskResponse contains the calculated risk metrics
type RiskResponse struct {
	Metrics []SimulationMetric
}

// MetricRequest encapsulates parameters for basic metric calculation (NAV, Exposure)
type MetricRequest struct {
	TenantID  string
	AsOf      time.Time
	Positions map[string]float64
	Prices    map[string]float64 // Optional overrides
	Shocks    *MarketShock       `json:"shocks,omitempty"`
}

// MetricResponse contains calculated basic metrics
type MetricResponse struct {
	Metrics []SimulationMetric
}

// ----------------------------------------------------------------------------
// Mock Implementations (for development/demo)
// ----------------------------------------------------------------------------

type MockRiskEngine struct{}

func NewMockRiskEngine() *MockRiskEngine {
	return &MockRiskEngine{}
}

func (m *MockRiskEngine) ComputeRisk(ctx context.Context, req RiskRequest) (*RiskResponse, error) {
	// Simulate computation time
	// time.Sleep(100 * time.Millisecond)

	var metrics []SimulationMetric

	// Mock VaR calculation
	baseVaR := 500000.0 // Baseline
	if req.Shocks != nil {
		if req.Shocks.VolShockPct > 0 {
			baseVaR *= (1 + req.Shocks.VolShockPct) // Higher vol = higher VaR
		}
		if req.Shocks.EquityShockPct < 0 {
			baseVaR *= 1.15 // Equity crash increases correlation/risk
		}
		if req.Shocks.ParallelShiftBps > 0 {
			baseVaR *= 1.05 // Rate hike increases risk slightly
		}
	}

	metrics = append(metrics, SimulationMetric{
		ID:             uuid.NewString(),
		MetricName:     "VaR_95",
		SimulatedValue: baseVaR,
		Unit:           "USD",
	})

	// Mock Stress Test Result
	metrics = append(metrics, SimulationMetric{
		ID:             uuid.NewString(),
		MetricName:     "Stress_Loss_2008",
		SimulatedValue: rand.Float64()*1000000 + 2000000,
		Unit:           "USD",
	})

	return &RiskResponse{Metrics: metrics}, nil
}

type MockCalculationEngine struct{}

func NewMockCalculationEngine() *MockCalculationEngine {
	return &MockCalculationEngine{}
}

func (m *MockCalculationEngine) ComputeMetrics(ctx context.Context, req MetricRequest) (*MetricResponse, error) {
	var metrics []SimulationMetric

	var totalNAV float64
	for _, qty := range req.Positions {
		// Mock price if not known, assuming $100 avg price
		price := 100.0
		totalNAV += qty * price
	}

	// Apply Market Shocks to Values
	if req.Shocks != nil {
		if req.Shocks.EquityShockPct != 0 {
			// Naive: assume 100% equity correlation for MVP
			totalNAV *= (1 + req.Shocks.EquityShockPct)
		}
		if req.Shocks.ParallelShiftBps > 0 {
			// Bond proxy: -duration * shift. Assume Duration=5.
			// 50bps => 0.5% => -2.5% impact
			impact := -5.0 * (req.Shocks.ParallelShiftBps / 10000.0)
			totalNAV *= (1 + impact)
		}
	}

	metrics = append(metrics, SimulationMetric{
		ID:             uuid.NewString(),
		MetricName:     "NAV",
		SimulatedValue: totalNAV,
		Unit:           "USD",
	})

	metrics = append(metrics, SimulationMetric{
		ID:             uuid.NewString(),
		MetricName:     "Gross_Exposure",
		SimulatedValue: totalNAV * 1.5, // Mock leverage
		Unit:           "USD",
	})

	return &MetricResponse{Metrics: metrics}, nil
}
