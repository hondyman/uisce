package cbo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// GatingResult captures the pre-flight complexity evaluation and circuit breaker status.
type GatingResult struct {
	QueryHash        string         `json:"query_hash"`
	ComplexityScore  float64        `json:"complexity_score"`
	Status           string         `json:"status"` // "ALLOWED" | "WARNING" | "FORBIDDEN"
	Blocked          bool           `json:"blocked"`
	Reason           string         `json:"reason,omitempty"`
	JoinCount        int            `json:"join_count"`
	MissingPartition bool           `json:"missing_partition"`
	CrossTierFederated bool         `json:"cross_tier_federated"`
	AuditLogID       uuid.UUID      `json:"audit_log_id"`
	EvaluatedAt      time.Time      `json:"evaluated_at"`
	EvaluationTimeUS int64          `json:"evaluation_time_us"`
}

// CostGovernor enforces pre-flight AST complexity scoring and Rule 8 circuit breakers.
type CostGovernor struct {
	maxAllowedScore  float64
	warningScore     float64
	logger           *zap.Logger
}

// NewCostGovernor creates a new CostGovernor instance.
func NewCostGovernor(maxScore, warnScore float64, logger *zap.Logger) *CostGovernor {
	if maxScore <= 0 {
		maxScore = 85.0 // Rule 8: Circuit breaker threshold
	}
	if warnScore <= 0 {
		warnScore = 65.0
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &CostGovernor{
		maxAllowedScore: maxScore,
		warningScore:    warnScore,
		logger:          logger,
	}
}

// EvaluatePreFlightComplexity calculates the AST complexity score:
// Score = BaseCost + (JoinCount * 15) + (MissingPartition * 30) + (CrossTierFederation * 40)
func (g *CostGovernor) EvaluatePreFlightComplexity(
	ctx context.Context,
	tenantID uuid.UUID,
	querySQL string,
	joinCount int,
	missingPartition bool,
	crossTierFederated bool,
) *GatingResult {
	start := time.Now()

	// 1. SHA-256 Non-leaking Query Fingerprint
	hasher := sha256.New()
	hasher.Write([]byte(querySQL))
	queryHash := hex.EncodeToString(hasher.Sum(nil))

	// 2. Derive Complexity Score
	baseCost := 10.0
	score := baseCost + float64(joinCount*15)
	if missingPartition {
		score += 30.0
	}
	if crossTierFederated {
		score += 40.0
	}

	result := &GatingResult{
		QueryHash:          queryHash,
		ComplexityScore:    score,
		JoinCount:          joinCount,
		MissingPartition:   missingPartition,
		CrossTierFederated: crossTierFederated,
		AuditLogID:         uuid.New(),
		EvaluatedAt:        time.Now().UTC(),
		EvaluationTimeUS:   time.Since(start).Microseconds(),
	}

	// 3. Rule 8 Circuit Breaker Enforcement
	if score >= g.maxAllowedScore {
		result.Status = "FORBIDDEN"
		result.Blocked = true
		result.Reason = fmt.Sprintf("Circuit breaker triggered: Query complexity score %.1f exceeds threshold %.1f (Joins: %d, MissingPartition: %t, CrossFederation: %t)",
			score, g.maxAllowedScore, joinCount, missingPartition, crossTierFederated)
		g.logger.Warn("Rule 8 Cost Governor blocked runaway query",
			zap.String("tenant_id", tenantID.String()),
			zap.String("query_hash", queryHash),
			zap.Float64("score", score),
		)
	} else if score >= g.warningScore {
		result.Status = "WARNING"
		result.Blocked = false
		result.Reason = fmt.Sprintf("High complexity query warning: score %.1f (Joins: %d)", score, joinCount)
	} else {
		result.Status = "ALLOWED"
		result.Blocked = false
	}

	return result
}
