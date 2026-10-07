package limits

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// LimitDataPoint represents a historical evaluation point for sparklines.
type LimitDataPoint struct {
	Timestamp      string  `json:"timestamp"`
	Value          float64 `json:"value"`
	UtilizationPct float64 `json:"utilization_pct"`
	Status         string  `json:"status"` // SAFE, CAUTION, WARNING, BREACHED
}

// LimitUtilizationRecord represents the real-time threshold proximity and headroom for a rule.
type LimitUtilizationRecord struct {
	ID             uuid.UUID        `json:"id"`
	TenantID       uuid.UUID        `json:"tenant_id"`
	AccountID      uuid.UUID        `json:"account_id"`
	AccountName    string           `json:"account_name"`
	RuleID         string           `json:"rule_id"`
	RuleCode       string           `json:"rule_code"`
	RuleName       string           `json:"rule_name"`
	RulePack       string           `json:"rule_pack"` // SEC, UCITS, 1940_ACT, MANDATE, RISK
	Category       string           `json:"category"`  // CONCENTRATION, LEVERAGE, LIQUIDITY, ISSUER, SECTOR, COUNTERPARTY
	ThresholdLimit decimal.Decimal  `json:"threshold_limit"`
	CurrentValue   decimal.Decimal  `json:"current_value"`
	UtilizationPct decimal.Decimal  `json:"utilization_pct"` // (CurrentValue / ThresholdLimit) * 100
	HeadroomPct    decimal.Decimal  `json:"headroom_pct"`    // 100 - UtilizationPct (or threshold - current)
	HeadroomAmount decimal.Decimal  `json:"headroom_amount"` // Dollar amount buffer remaining before breach
	Currency       string           `json:"currency"`
	Status         string           `json:"status"` // SAFE (<70%), CAUTION (70-85%), WARNING (85-99%), BREACHED (>=100%)
	Trend          string           `json:"trend"`  // INCREASING, DECREASING, STABLE
	EvaluatedAt    time.Time        `json:"evaluated_at"`
	History        []LimitDataPoint `json:"history,omitempty"`
}

// LimitFilter defines query parameters for utilization metrics.
type LimitFilter struct {
	TenantID  uuid.UUID
	AccountID *uuid.UUID
	RulePack  string
	Category  string
	Status    string
}
