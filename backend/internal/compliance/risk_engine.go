package compliance

import (
	"strings"
)

// ComplianceRiskEngine calculates risk scores for jobs and DAGs
type ComplianceRiskEngine struct {
	// Future: Add configurable weights
}

// NewComplianceRiskEngine creates a new risk engine
func NewComplianceRiskEngine() *ComplianceRiskEngine {
	return &ComplianceRiskEngine{}
}

// ComplianceContext defines the input parameters for risk calculation
type ComplianceContext struct {
	PII                  bool
	Residency            string // e.g., "EU", "US", "GLOBAL"
	Sensitivity          string // e.g., "LOW", "MEDIUM", "HIGH"
	SemanticCount        int
	AffectedTenants      int
	HistoricalViolations int
	SLOCritical          bool
}

// Score calculates the risk score (0-1) and risk level (LOW/MEDIUM/HIGH)
func (e *ComplianceRiskEngine) Score(ctx ComplianceContext) (float64, string) {
	score := 0.0

	// Base risk from PII
	if ctx.PII {
		score += 0.3
	}

	// Sensitivity Impact
	switch strings.ToUpper(ctx.Sensitivity) {
	case "MEDIUM":
		score += 0.2
	case "HIGH":
		score += 0.4
	}

	// Residency Constraints (EU often implies stricter GDPR controls)
	if strings.ToUpper(ctx.Residency) == "EU" {
		score += 0.1
	}

	// Blast Radius / Complexity
	if ctx.SemanticCount > 10 {
		score += 0.1
	}
	if ctx.AffectedTenants > 3 {
		score += 0.1
	}

	// Historical Context
	if ctx.HistoricalViolations > 0 {
		score += 0.2
	}

	// Operational Criticality
	if ctx.SLOCritical {
		score += 0.1
	}

	// Cap score at 1.0
	if score > 1.0 {
		score = 1.0
	}

	// Determine Level
	var level string
	if score < 0.3 {
		level = "LOW"
	} else if score < 0.6 {
		level = "MEDIUM"
	} else {
		level = "HIGH"
	}

	return score, level
}
