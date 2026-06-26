package wealth

import (
	"time"

	"github.com/hondyman/semlayer/backend/internal/models"
)

// PrivateMarketsService handles private markets data operations
type PrivateMarketsService struct{}

// NewPrivateMarketsService creates a new private markets service
func NewPrivateMarketsService() *PrivateMarketsService {
	return &PrivateMarketsService{}
}

// GetUser returns user information by ID
func (s *PrivateMarketsService) GetUser(userID string, role string) *models.User {
	if role == "" {
		role = "lp" // default
	}

	return &models.User{
		ID:           userID,
		Name:         "John Doe",
		Role:         role,
		Organization: "Sample Organization",
		Permissions:  []string{"read", "write", "admin"},
	}
}

// GetBundles returns available bundles for a specific audience
func (s *PrivateMarketsService) GetBundles(audience string) []*models.Bundle {
	if audience == "" {
		audience = "lp"
	}

	switch audience {
	case "lp":
		return []*models.Bundle{
			{
				ID:       "lp_private_markets_bundle",
				Name:     "LP Private Markets Bundle",
				Audience: "lp",
				Version:  "1.0.0",
				Modules: []models.BundleModule{
					{ID: "fund-selector", Name: "Fund Selector", Type: "selector", Config: map[string]interface{}{"multiSelect": true}},
					{ID: "irr-curve", Name: "IRR Curve Chart", Type: "chart", Config: map[string]interface{}{"timeRange": "5y"}},
					{ID: "j-curve", Name: "J-Curve Plot", Type: "chart", Config: map[string]interface{}{"showBenchmark": true}},
					{ID: "benchmark-comparison", Name: "Benchmark Comparison", Type: "comparison", Config: map[string]interface{}{"indices": []string{"S&P 500", "NASDAQ"}}},
					{ID: "liquidity-panel", Name: "Liquidity Panel", Type: "panel", Config: map[string]interface{}{"showProjections": true}},
				},
				Metrics: []models.BundleMetric{
					{ID: "tvpi", Name: "TVPI", Type: "ratio", Formula: "(distributions + residual_value) / paid_in_capital"},
					{ID: "irr", Name: "IRR", Type: "percentage", Formula: "XIRR(cash_flows, dates)"},
					{ID: "pme", Name: "PME", Type: "ratio", Formula: "PME(cash_flows, benchmark)"},
				},
				Governance: models.BundleGovernance{
					Status:       "active",
					StewardGroup: "data-stewards",
					SchemaHash:   "abc123",
					SLA:          models.BundleSLA{RefreshFrequency: "daily", MaxLatency: "4h"},
				},
			},
		}
	case "gp":
		return []*models.Bundle{
			{
				ID:       "gp_private_markets_bundle",
				Name:     "GP Private Markets Bundle",
				Audience: "gp",
				Version:  "1.0.0",
				Modules: []models.BundleModule{
					{ID: "deployment-pacing", Name: "Deployment Pacing Chart", Type: "chart", Config: map[string]interface{}{"targetPacing": "24months"}},
					{ID: "irr-nav-tracking", Name: "IRR/NAV Tracking", Type: "tracking", Config: map[string]interface{}{"frequency": "quarterly"}},
					{ID: "fee-analysis", Name: "Fee Analysis", Type: "analysis", Config: map[string]interface{}{"feeTypes": []string{"management", "performance"}}},
					{ID: "value-attribution", Name: "Value Attribution", Type: "attribution", Config: map[string]interface{}{"methodology": "brinson"}},
					{ID: "exit-analysis", Name: "Exit Analysis", Type: "analysis", Config: map[string]interface{}{"exitTypes": []string{"ipo", "merger", "sale"}}},
				},
				Metrics: []models.BundleMetric{
					{ID: "dpi", Name: "DPI", Type: "ratio", Formula: "distributions / paid_in_capital"},
					{ID: "rvpi", Name: "RVPI", Type: "ratio", Formula: "residual_value / paid_in_capital"},
					{ID: "tvpi", Name: "TVPI", Type: "ratio", Formula: "dpi + rvpi"},
				},
				Governance: models.BundleGovernance{
					Status:       "active",
					StewardGroup: "gp-stewards",
					SchemaHash:   "def456",
					SLA:          models.BundleSLA{RefreshFrequency: "weekly", MaxLatency: "24h"},
				},
			},
		}
	case "fof":
		return []*models.Bundle{
			{
				ID:       "fof_private_markets_bundle",
				Name:     "FoF Private Markets Bundle",
				Audience: "fof",
				Version:  "1.0.0",
				Modules: []models.BundleModule{
					{ID: "portfolio-overview", Name: "Portfolio Overview", Type: "overview", Config: map[string]interface{}{"groupBy": "strategy"}},
					{ID: "manager-performance", Name: "Manager Performance", Type: "performance", Config: map[string]interface{}{"benchmark": true}},
					{ID: "allocation-analysis", Name: "Allocation Analysis", Type: "analysis", Config: map[string]interface{}{"dimensions": []string{"geography", "vintage"}}},
					{ID: "risk-attribution", Name: "Risk Attribution", Type: "attribution", Config: map[string]interface{}{"method": "factor"}},
				},
				Metrics: []models.BundleMetric{
					{ID: "portfolio-irr", Name: "Portfolio IRR", Type: "percentage", Formula: "weighted_average(irr)"},
					{ID: "diversification", Name: "Diversification Score", Type: "score", Formula: "1 - concentration_ratio"},
					{ID: "alpha", Name: "Alpha vs Benchmark", Type: "percentage", Formula: "irr - benchmark_irr"},
				},
				Governance: models.BundleGovernance{
					Status:       "active",
					StewardGroup: "fof-stewards",
					SchemaHash:   "ghi789",
					SLA:          models.BundleSLA{RefreshFrequency: "monthly", MaxLatency: "48h"},
				},
			},
		}
	default:
		return []*models.Bundle{}
	}
}

// GetFunds returns a list of available funds
func (s *PrivateMarketsService) GetFunds() []*models.Fund {
	return []*models.Fund{
		{
			ID:        "fund-1",
			Name:      "Tech Growth Fund III",
			Vintage:   2020,
			Manager:   "TechVentures Capital",
			Strategy:  "Venture Capital",
			Geography: "North America",
			Status:    "active",
			CreatedAt: time.Now().AddDate(-3, 0, 0),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "fund-2",
			Name:      "Infrastructure Partners II",
			Vintage:   2019,
			Manager:   "InfraCapital",
			Strategy:  "Infrastructure",
			Geography: "Europe",
			Status:    "active",
			CreatedAt: time.Now().AddDate(-4, 0, 0),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "fund-3",
			Name:      "Real Estate Fund IV",
			Vintage:   2021,
			Manager:   "PropertyPartners",
			Strategy:  "Real Estate",
			Geography: "Asia Pacific",
			Status:    "active",
			CreatedAt: time.Now().AddDate(-2, 0, 0),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "fund-4",
			Name:      "Healthcare Innovation Fund",
			Vintage:   2022,
			Manager:   "MedTech Ventures",
			Strategy:  "Healthcare",
			Geography: "Global",
			Status:    "active",
			CreatedAt: time.Now().AddDate(-1, 0, 0),
			UpdatedAt: time.Now(),
		},
	}
}

// GetFundMetrics returns performance metrics for a specific fund
func (s *PrivateMarketsService) GetFundMetrics(fundID string) *models.FundMetrics {
	// Mock different metrics based on fund ID
	baseMetrics := models.FundMetrics{
		FundID:        fundID,
		PaidInCapital: 100000000,
		Distributions: 85000000,
		ResidualValue: 123000000,
		AsOfDate:      time.Now(),
	}

	switch fundID {
	case "fund-1": // Tech Growth Fund
		baseMetrics.TVPI = 1.85
		baseMetrics.RVPI = 1.23
		baseMetrics.IRR = 0.156
		baseMetrics.XIRR = 0.142
		baseMetrics.PME = 1.12
	case "fund-2": // Infrastructure
		baseMetrics.TVPI = 1.65
		baseMetrics.RVPI = 1.45
		baseMetrics.IRR = 0.123
		baseMetrics.XIRR = 0.118
		baseMetrics.PME = 1.08
	case "fund-3": // Real Estate
		baseMetrics.TVPI = 1.92
		baseMetrics.RVPI = 1.67
		baseMetrics.IRR = 0.145
		baseMetrics.XIRR = 0.138
		baseMetrics.PME = 1.15
	case "fund-4": // Healthcare
		baseMetrics.TVPI = 2.05
		baseMetrics.RVPI = 1.89
		baseMetrics.IRR = 0.178
		baseMetrics.XIRR = 0.165
		baseMetrics.PME = 1.22
	default:
		baseMetrics.TVPI = 1.75
		baseMetrics.RVPI = 1.35
		baseMetrics.IRR = 0.135
		baseMetrics.XIRR = 0.128
		baseMetrics.PME = 1.10
	}

	return &baseMetrics
}
