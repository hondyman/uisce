package risk

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// RiskFactor represents edm.risk_factor
type RiskFactor struct {
	FactorID   uuid.UUID `json:"factor_id" db:"factor_id"`
	FactorCode string    `json:"factor_code" db:"factor_code"`
	FactorName string    `json:"factor_name" db:"factor_name"`
	Category   *string   `json:"category" db:"category"`
	FactorType *string   `json:"factor_type" db:"factor_type"`
	Unit       *string   `json:"unit" db:"unit"`
	TenantID   uuid.UUID `json:"tenant_id" db:"tenant_id"`
}

// SecurityFactorExposure represents edm.security_factor_exposure
type SecurityFactorExposure struct {
	ExposureID uuid.UUID        `json:"exposure_id" db:"exposure_id"`
	SecurityID uuid.UUID        `json:"security_id" db:"security_id"`
	FactorID   uuid.UUID        `json:"factor_id" db:"factor_id"`
	AsOfDate   time.Time        `json:"as_of_date" db:"as_of_date"`
	Exposure   *decimal.Decimal `json:"exposure" db:"exposure"`
	Confidence *decimal.Decimal `json:"confidence" db:"confidence"`
	TenantID   uuid.UUID        `json:"tenant_id" db:"tenant_id"`
}

// PortfolioRisk represents edm.portfolio_risk
type PortfolioRisk struct {
	PortfolioRiskID     uuid.UUID        `json:"portfolio_risk_id" db:"portfolio_risk_id"`
	PortfolioID         uuid.UUID        `json:"portfolio_id" db:"portfolio_id"`
	ValuationDate       time.Time        `json:"valuation_date" db:"valuation_date"`
	TotalVolatility     *decimal.Decimal `json:"total_volatility" db:"total_volatility"`
	TrackingError       *decimal.Decimal `json:"tracking_error" db:"tracking_error"`
	VaR95               *decimal.Decimal `json:"var_95" db:"var_95"`
	VaR99               *decimal.Decimal `json:"var_99" db:"var_99"`
	ExpectedShortfall   *decimal.Decimal `json:"expected_shortfall" db:"expected_shortfall"`
	FactorContributions json.RawMessage  `json:"factor_contributions" db:"factor_contributions"`
	Methodology         *string          `json:"methodology" db:"methodology"`
	AuditTrace          json.RawMessage  `json:"audit_trace" db:"audit_trace"`
	TenantID            uuid.UUID        `json:"tenant_id" db:"tenant_id"`
}

// RiskScenario represents edm.risk_scenario
type RiskScenario struct {
	ScenarioID   uuid.UUID       `json:"scenario_id" db:"scenario_id"`
	ScenarioCode string          `json:"scenario_code" db:"scenario_code"`
	ScenarioName string          `json:"scenario_name" db:"scenario_name"`
	Description  *string         `json:"description" db:"description"`
	ScenarioType *string         `json:"scenario_type" db:"scenario_type"`
	Shocks       json.RawMessage `json:"shocks" db:"shocks"`
	Status       string          `json:"status" db:"status"`
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at" db:"updated_at"`
	TenantID     uuid.UUID       `json:"tenant_id" db:"tenant_id"`
}

// RiskScenarioResult represents edm.risk_scenario_result
type RiskScenarioResult struct {
	ScenarioResultID uuid.UUID        `json:"scenario_result_id" db:"scenario_result_id"`
	ScenarioID       uuid.UUID        `json:"scenario_id" db:"scenario_id"`
	PortfolioID      uuid.UUID        `json:"portfolio_id" db:"portfolio_id"`
	ValuationDate    time.Time        `json:"valuation_date" db:"valuation_date"`
	PnL              *decimal.Decimal `json:"pnl" db:"pnl"`
	PnLPercent       *decimal.Decimal `json:"pnl_percent" db:"pnl_percent"`
	Details          json.RawMessage  `json:"details" db:"details"`
	RunAt            time.Time        `json:"run_at" db:"run_at"`
	TenantID         uuid.UUID        `json:"tenant_id" db:"tenant_id"`
}
