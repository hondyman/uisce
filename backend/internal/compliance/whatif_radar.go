package compliance

import (
	"context"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type ExecutionScope string

const (
	ScopePositionsOnly     ExecutionScope = "POS"     // Production official mode
	ScopePositionsPlusExec ExecutionScope = "POSEXEC" // Positions + Executed Orders
	ScopeWhatIfProposed    ExecutionScope = "POSORD"  // Positions + Open/Proposed Orders
)

type PortfolioHoldingSnapshot struct {
	SecurityID  string  `json:"securityId"`
	Quantity    float64 `json:"quantity"`
	MarketValue float64 `json:"marketValue"`
}

type ExceptionReopenRadarService struct {
	db *sqlx.DB
}

func NewExceptionReopenRadarService(db *sqlx.DB) *ExceptionReopenRadarService {
	return &ExceptionReopenRadarService{db: db}
}

// EvaluateWhatIfCompliance tests proposed blotter orders without writing official breach records
func (s *ExceptionReopenRadarService) EvaluateWhatIfCompliance(
	ctx context.Context,
	tenantID uuid.UUID,
	portfolioID string,
	scope ExecutionScope,
	holdings []PortfolioHoldingSnapshot,
	ruleThresholdPct float64,
) (bool, float64, error) {
	if tenantID == uuid.Nil {
		return false, 0, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	var totalAUM, maxIssuerMV float64
	for _, h := range holdings {
		totalAUM += h.MarketValue
		if h.MarketValue > maxIssuerMV {
			maxIssuerMV = h.MarketValue
		}
	}

	var concentrationPct float64
	if totalAUM > 0 {
		concentrationPct = (maxIssuerMV / totalAUM) * 100.0
	}

	isBreach := concentrationPct > ruleThresholdPct
	return isBreach, concentrationPct, nil
}

// EvaluateReopenState checks if closed alerts must automatically reopen due to market or constituent drift
func (s *ExceptionReopenRadarService) EvaluateReopenState(
	previousStatus string,
	previousMV float64,
	currentMV float64,
	previousFingerprint string,
	currentFingerprint string,
	tolerancePct float64,
	stillViolating bool,
) (bool, string) {
	if tolerancePct <= 0 {
		tolerancePct = 10.0 // Default 10% drift tolerance
	}

	// 1. CLOSED_CORRECTED check: Reopen if condition is not actually resolved
	if previousStatus == "CLOSED_CORRECTED" && stillViolating {
		return true, "Condition persisted on subsequent scan without correction"
	}

	// 2. CLOSED_NO_ACTION: Reopen if Market Value drift exceeds tolerance
	if previousStatus == "CLOSED_NO_ACTION" && stillViolating {
		var mvDriftPct float64
		if previousMV > 0 {
			mvDriftPct = math.Abs(currentMV-previousMV) / previousMV * 100.0
		}
		if mvDriftPct > tolerancePct {
			return true, fmt.Sprintf("Market value drift (%.2f%%) exceeded tolerance threshold (%.2f%%)", mvDriftPct, tolerancePct)
		}
	}

	// 3. Constituent Fingerprint Shift
	if previousFingerprint != "" && previousFingerprint != currentFingerprint && stillViolating {
		return true, "Underlying portfolio constituent security IDs shifted"
	}

	return false, ""
}
