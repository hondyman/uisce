package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
)

// ============================================================================
// Portfolio Handler - Risk & Compliance Console
// ============================================================================

type PortfolioHandler struct {
	db *sqlx.DB
}

func NewPortfolioHandler(db *sqlx.DB) *PortfolioHandler {
	return &PortfolioHandler{
		db: db,
	}
}

// RegisterRoutes registers all portfolio routes
func (h *PortfolioHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/portfolios", func(r chi.Router) {
		r.Get("/{portfolioId}/overview", h.GetPortfolioOverview)
		r.Get("/{portfolioId}/holdings", h.GetHoldings)
		r.Get("/{portfolioId}/risk", h.GetPortfolioRisk)
		r.Get("/{portfolioId}/compliance", h.GetPortfolioCompliance)
		r.Get("/{portfolioId}/scenarios", h.GetScenarios)
	})
}

// ============================================================================
// Request/Response Types
// ============================================================================

// PortfolioOverview contains portfolio summary metrics
type PortfolioMetrics struct {
	TotalValue        float64 `json:"totalValue"`
	DayChangeAmt      float64 `json:"dayChangeAmt"`
	DayChangePercent  float64 `json:"dayChangePercent"`
	YTDReturnPercent  float64 `json:"ytdReturnPercent"`
	OneYearReturn     float64 `json:"oneYearReturn"`
	IncepToDateReturn float64 `json:"incepToDateReturn"`
}

type PortfolioPerformance struct {
	BenchmarkName   string  `json:"benchmarkName"`
	PortfolioReturn float64 `json:"portfolioReturn"`
	BenchmarkReturn float64 `json:"benchmarkReturn"`
	Outperformance  float64 `json:"outperformance"`
	Inception       string  `json:"inception"`
}

type PortfolioOverviewResponse struct {
	PortfolioID   string               `json:"portfolioId"`
	PortfolioName string               `json:"portfolioName"`
	Manager       string               `json:"manager"`
	Status        string               `json:"status"` // "Active" | "Closed"
	CreatedDate   string               `json:"createdDate"`
	ValuationDate string               `json:"valuationDate"`
	Metrics       PortfolioMetrics     `json:"metrics"`
	Performance   PortfolioPerformance `json:"performance"`
	Timestamp     string               `json:"timestamp"`
}

// Holding represents a single portfolio position
type Holding struct {
	InstrumentID  string  `json:"instrumentId"`
	Symbol        string  `json:"symbol"`
	Name          string  `json:"name"`
	AssetClass    string  `json:"assetClass"` // "Equity" | "Fixed Income" | "Commodity" | etc
	Quantity      float64 `json:"quantity"`
	UnitPrice     float64 `json:"unitPrice"`
	PositionValue float64 `json:"positionValue"`
	WeightPercent float64 `json:"weightPercent"`
	DayChange     float64 `json:"dayChange"`
	YTDReturn     float64 `json:"ytdReturn"`
	CountryCode   string  `json:"countryCode"`
	SectorCode    string  `json:"sectorCode"`
}

type SectorWeight struct {
	SectorName    string  `json:"sectorName"`
	WeightPercent float64 `json:"weightPercent"`
	ValueAmt      float64 `json:"valueAmt"`
}

type HoldingsResponse struct {
	PortfolioID     string         `json:"portfolioId"`
	ValuationDate   string         `json:"valuationDate"`
	TotalHoldings   int64          `json:"totalHoldings"`
	TopHoldings     []Holding      `json:"topHoldings"`
	SectorWeights   []SectorWeight `json:"sectorWeights"`
	AssetAllocation []SectorWeight `json:"assetAllocation"` // Alternative name for asset class weighting
	CashPosition    float64        `json:"cashPosition"`
	Timestamp       string         `json:"timestamp"`
}

// PortfolioRiskMetric represents a risk factor or metric
type PortfolioRiskMetric struct {
	FactorName   string  `json:"factorName"`
	Exposure     float64 `json:"exposure"`
	Beta         float64 `json:"beta"`
	Contribution float64 `json:"contribution"` // % contribution to total risk
}

type PortfolioRiskResponse struct {
	PortfolioID       string                `json:"portfolioId"`
	ValuationDate     string                `json:"valuationDate"`
	Volatility        float64               `json:"volatility"`
	VaR95             float64               `json:"var95"`
	VaR99             float64               `json:"var99"`
	ExpectedShortfall float64               `json:"expectedShortfall"`
	SharpeRatio       float64               `json:"sharpeRatio"`
	Factors           []PortfolioRiskMetric `json:"factors"`
	Timestamp         string                `json:"timestamp"`
}

// ComplianceBreachDetail represents a compliance rule violation
type ComplianceBreachDetail struct {
	RuleID        string  `json:"ruleId"`
	RuleName      string  `json:"ruleName"`
	Status        string  `json:"status"` // "Breach" | "Warning" | "Pass"
	CurrentValue  float64 `json:"currentValue"`
	LimitValue    float64 `json:"limitValue"`
	Severity      string  `json:"severity"` // "Critical" | "Warning" | "Info"
	Description   string  `json:"description"`
	RemediationBy *string `json:"remediationBy,omitempty"`
}

type PortfolioComplianceResponse struct {
	PortfolioID   string                   `json:"portfolioId"`
	ValuationDate string                   `json:"valuationDate"`
	TotalRules    int64                    `json:"totalRules"`
	PassingRules  int64                    `json:"passingRules"`
	BreachCount   int64                    `json:"breachCount"`
	WarningCount  int64                    `json:"warningCount"`
	BreachDetails []ComplianceBreachDetail `json:"breachDetails"`
	Timestamp     string                   `json:"timestamp"`
}

// ScenarioAnalysis represents a what-if scenario PnL
type ScenarioAnalysis struct {
	ScenarioID     string  `json:"scenarioId"`
	ScenarioName   string  `json:"scenarioName"`
	Description    string  `json:"description"`
	BasedOnDate    string  `json:"basedOnDate"`
	BaselineValue  float64 `json:"baselineValue"`
	SimulatedValue float64 `json:"simulatedValue"`
	PnLChange      float64 `json:"pnlChange"`
	PercentChange  float64 `json:"percentChange"`
	BreachCount    int64   `json:"breachCount"` // Compliance breaches triggered
	RiskMetrics    struct {
		VolatilityChange float64 `json:"volatilityChange"`
		VaRChange        float64 `json:"varChange"`
	} `json:"riskMetrics"`
}

type ScenariosResponse struct {
	PortfolioID   string             `json:"portfolioId"`
	ValuationDate string             `json:"valuationDate"`
	Scenarios     []ScenarioAnalysis `json:"scenarios"`
	Timestamp     string             `json:"timestamp"`
}

// ============================================================================
// Handler Methods
// ============================================================================

// GetPortfolioOverview returns portfolio summary data
// GET /api/portfolios/{portfolioId}/overview?tenant_id=xxx&valuation_date=yyyy-mm-dd
func (h *PortfolioHandler) GetPortfolioOverview(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolioId")
	tenantID := r.URL.Query().Get("tenant_id")

	if tenantID == "" {
		http.Error(w, "tenant_id query parameter is required", http.StatusBadRequest)
		return
	}

	if portfolioID == "" {
		http.Error(w, "portfolioId path parameter is required", http.StatusBadRequest)
		return
	}

	valuationDate := r.URL.Query().Get("valuation_date")
	if valuationDate == "" {
		valuationDate = time.Now().Format("2006-01-02")
	}

	// TODO: Query database for portfolio data
	// Verify portfolio belongs to tenant
	response := PortfolioOverviewResponse{
		PortfolioID:   portfolioID,
		PortfolioName: "Growth Equity Fund",
		Manager:       "Patrick Chen",
		Status:        "Active",
		CreatedDate:   "2023-01-15",
		ValuationDate: valuationDate,
		Metrics: PortfolioMetrics{
			TotalValue:        12500000.0,
			DayChangeAmt:      85620.0,
			DayChangePercent:  0.68,
			YTDReturnPercent:  12.35,
			OneYearReturn:     18.42,
			IncepToDateReturn: 42.18,
		},
		Performance: PortfolioPerformance{
			BenchmarkName:   "Russell 2000",
			PortfolioReturn: 18.42,
			BenchmarkReturn: 16.25,
			Outperformance:  2.17,
			Inception:       "2023-01-15",
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetHoldings returns top holdings and sector breakdown
// GET /api/portfolios/{portfolioId}/holdings?tenant_id=xxx&valuation_date=yyyy-mm-dd
func (h *PortfolioHandler) GetHoldings(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolioId")
	tenantID := r.URL.Query().Get("tenant_id")

	if tenantID == "" {
		http.Error(w, "tenant_id query parameter is required", http.StatusBadRequest)
		return
	}

	if portfolioID == "" {
		http.Error(w, "portfolioId path parameter is required", http.StatusBadRequest)
		return
	}

	valuationDate := r.URL.Query().Get("valuation_date")
	if valuationDate == "" {
		valuationDate = time.Now().Format("2006-01-02")
	}

	response := HoldingsResponse{
		PortfolioID:   portfolioID,
		ValuationDate: valuationDate,
		TotalHoldings: 145,
		TopHoldings: []Holding{
			{
				InstrumentID:  "INSTR-001",
				Symbol:        "AAPL",
				Name:          "Apple Inc.",
				AssetClass:    "Equity",
				Quantity:      5000,
				UnitPrice:     195.50,
				PositionValue: 977500.0,
				WeightPercent: 7.82,
				DayChange:     1.25,
				YTDReturn:     28.30,
				CountryCode:   "US",
				SectorCode:    "Tech",
			},
			{
				InstrumentID:  "INSTR-002",
				Symbol:        "MSFT",
				Name:          "Microsoft Corporation",
				AssetClass:    "Equity",
				Quantity:      3200,
				UnitPrice:     380.45,
				PositionValue: 1217440.0,
				WeightPercent: 9.74,
				DayChange:     0.85,
				YTDReturn:     31.20,
				CountryCode:   "US",
				SectorCode:    "Tech",
			},
			{
				InstrumentID:  "INSTR-003",
				Symbol:        "NVDA",
				Name:          "NVIDIA Corporation",
				AssetClass:    "Equity",
				Quantity:      1500,
				UnitPrice:     875.30,
				PositionValue: 1312950.0,
				WeightPercent: 10.51,
				DayChange:     2.15,
				YTDReturn:     45.60,
				CountryCode:   "US",
				SectorCode:    "Tech",
			},
		},
		SectorWeights: []SectorWeight{
			{SectorName: "Technology", WeightPercent: 32.5, ValueAmt: 4062500.0},
			{SectorName: "Healthcare", WeightPercent: 18.2, ValueAmt: 2275000.0},
			{SectorName: "Financials", WeightPercent: 15.8, ValueAmt: 1975000.0},
			{SectorName: "Consumer Discretionary", WeightPercent: 12.1, ValueAmt: 1512500.0},
			{SectorName: "Industrials", WeightPercent: 10.5, ValueAmt: 1312500.0},
			{SectorName: "Other", WeightPercent: 10.9, ValueAmt: 1362500.0},
		},
		AssetAllocation: []SectorWeight{
			{SectorName: "Equities", WeightPercent: 92.0, ValueAmt: 11500000.0},
			{SectorName: "Fixed Income", WeightPercent: 6.0, ValueAmt: 750000.0},
			{SectorName: "Cash", WeightPercent: 2.0, ValueAmt: 250000.0},
		},
		CashPosition: 250000.0,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetPortfolioRisk returns portfolio risk metrics
// GET /api/portfolios/{portfolioId}/risk?tenant_id=xxx&valuation_date=yyyy-mm-dd
func (h *PortfolioHandler) GetPortfolioRisk(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolioId")
	tenantID := r.URL.Query().Get("tenant_id")

	if tenantID == "" {
		http.Error(w, "tenant_id query parameter is required", http.StatusBadRequest)
		return
	}

	if portfolioID == "" {
		http.Error(w, "portfolioId path parameter is required", http.StatusBadRequest)
		return
	}

	valuationDate := r.URL.Query().Get("valuation_date")
	if valuationDate == "" {
		valuationDate = time.Now().Format("2006-01-02")
	}

	response := PortfolioRiskResponse{
		PortfolioID:       portfolioID,
		ValuationDate:     valuationDate,
		Volatility:        8.34,
		VaR95:             2350000.0,
		VaR99:             3125000.0,
		ExpectedShortfall: 3500000.0,
		SharpeRatio:       1.45,
		Factors: []PortfolioRiskMetric{
			{
				FactorName:   "Market Risk",
				Exposure:     1.2,
				Beta:         1.15,
				Contribution: 45.3,
			},
			{
				FactorName:   "Size (SMB)",
				Exposure:     0.35,
				Beta:         0.42,
				Contribution: 12.1,
			},
			{
				FactorName:   "Value (HML)",
				Exposure:     -0.15,
				Beta:         -0.18,
				Contribution: -5.2,
			},
			{
				FactorName:   "Momentum",
				Exposure:     0.65,
				Beta:         0.58,
				Contribution: 18.4,
			},
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetPortfolioCompliance returns portfolio compliance status
// GET /api/portfolios/{portfolioId}/compliance?tenant_id=xxx&valuation_date=yyyy-mm-dd
func (h *PortfolioHandler) GetPortfolioCompliance(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolioId")
	tenantID := r.URL.Query().Get("tenant_id")

	if tenantID == "" {
		http.Error(w, "tenant_id query parameter is required", http.StatusBadRequest)
		return
	}

	if portfolioID == "" {
		http.Error(w, "portfolioId path parameter is required", http.StatusBadRequest)
		return
	}

	valuationDate := r.URL.Query().Get("valuation_date")
	if valuationDate == "" {
		valuationDate = time.Now().Format("2006-01-02")
	}

	response := PortfolioComplianceResponse{
		PortfolioID:   portfolioID,
		ValuationDate: valuationDate,
		TotalRules:    24,
		PassingRules:  21,
		BreachCount:   1,
		WarningCount:  2,
		BreachDetails: []ComplianceBreachDetail{
			{
				RuleID:        "rule-c-001",
				RuleName:      "Sector Concentration - Technology",
				Status:        "Breach",
				CurrentValue:  32.5,
				LimitValue:    30.0,
				Severity:      "Critical",
				Description:   "Technology sector concentration exceeds 30% policy limit",
				RemediationBy: ptrString("2026-02-28"),
			},
			{
				RuleID:        "rule-c-002",
				RuleName:      "Single Position Limit",
				Status:        "Warning",
				CurrentValue:  10.51,
				LimitValue:    10.0,
				Severity:      "Warning",
				Description:   "Single position (NVDA) exceeds 10% limit by 51 bps",
				RemediationBy: ptrString("2026-02-25"),
			},
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetScenarios returns scenario analysis results
// GET /api/portfolios/{portfolioId}/scenarios?tenant_id=xxx&valuation_date=yyyy-mm-dd
func (h *PortfolioHandler) GetScenarios(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolioId")
	tenantID := r.URL.Query().Get("tenant_id")

	if tenantID == "" {
		http.Error(w, "tenant_id query parameter is required", http.StatusBadRequest)
		return
	}

	if portfolioID == "" {
		http.Error(w, "portfolioId path parameter is required", http.StatusBadRequest)
		return
	}

	valuationDate := r.URL.Query().Get("valuation_date")
	if valuationDate == "" {
		valuationDate = time.Now().Format("2006-01-02")
	}

	scenario1 := ScenarioAnalysis{
		ScenarioID:     "scen-001",
		ScenarioName:   "Rate Hike +100bps",
		Description:    "Fed raises rates by 100 basis points in next quarter",
		BasedOnDate:    valuationDate,
		BaselineValue:  12500000.0,
		SimulatedValue: 12125000.0,
		PnLChange:      -375000.0,
		PercentChange:  -3.0,
		BreachCount:    1,
	}
	scenario1.RiskMetrics.VolatilityChange = 1.2
	scenario1.RiskMetrics.VaRChange = 425000.0

	scenario2 := ScenarioAnalysis{
		ScenarioID:     "scen-002",
		ScenarioName:   "Market Correction -20%",
		Description:    "Broad market correction of 20% across all sectors",
		BasedOnDate:    valuationDate,
		BaselineValue:  12500000.0,
		SimulatedValue: 10000000.0,
		PnLChange:      -2500000.0,
		PercentChange:  -20.0,
		BreachCount:    3,
	}
	scenario2.RiskMetrics.VolatilityChange = 3.5
	scenario2.RiskMetrics.VaRChange = 1250000.0

	response := ScenariosResponse{
		PortfolioID:   portfolioID,
		ValuationDate: valuationDate,
		Scenarios:     []ScenarioAnalysis{scenario1, scenario2},
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ============================================================================
// Helper Functions
// ============================================================================
// Note: ptrString helper is defined in client_onboarding_handlers.go
