package query

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type PreflightFinOpsEstimate struct {
	HotPercentage      int     `json:"hotPercentage"`      // e.g. 80% StarRocks
	ColdPercentage     int     `json:"coldPercentage"`     // e.g. 20% Iceberg
	EstLatencyMs       int     `json:"estLatencyMs"`       // e.g. 180ms
	ScannedVolumeMB    float64 `json:"scannedVolumeMb"`    // e.g. 14.2 MB
	EstComputeUSD      float64 `json:"estComputeUsd"`      // e.g. 0.002
	ComplexityScore    int     `json:"complexityScore"`    // 0-100
	PassesBreaker      bool    `json:"passesBreaker"`
	BreakerMessage     string  `json:"breakerMessage,omitempty"`
	ExplainDAGSteps    []ExplainDAGStep `json:"explainDagSteps"`
}

type ExplainDAGStep struct {
	StepID      string `json:"stepId"`
	Name        string `json:"name"`
	Engine      string `json:"engine"` // StarRocks, Iceberg, Postgres, WASM
	Operation   string `json:"operation"` // PartitionPrune, PushdownFilter, Join, Aggregate
	Cost        float64 `json:"cost"`
	RowCount    int64  `json:"rowCount"`
	Description string `json:"description"`
}

type PreflightFinOpsService struct {
	db *sqlx.DB
}

func NewPreflightFinOpsService(db *sqlx.DB) *PreflightFinOpsService {
	return &PreflightFinOpsService{db: db}
}

// EvaluatePreflightCost calculates query complexity and estimated compute cost before execution
func (s *PreflightFinOpsService) EvaluatePreflightCost(
	ctx context.Context,
	tenantID uuid.UUID,
	boKey string,
	dimensionCount, measureCount, filterCount int,
	hasTimeRange bool,
) (*PreflightFinOpsEstimate, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	// Complexity calculation: Base + 15*Joins + 30*(unpartitioned if no time range)
	complexity := 10 + (dimensionCount * 2) + (measureCount * 3) + (filterCount * 2)
	if !hasTimeRange {
		complexity += 25
	}
	if complexity > 100 {
		complexity = 100
	}

	passesBreaker := complexity < 85
	var breakerMsg string
	if !passesBreaker {
		breakerMsg = "Query complexity exceeds 85/100 threshold. Add time partitioning filter to prevent unpartitioned full lakehouse scans."
	}

	hotPct := 85
	coldPct := 15
	if !hasTimeRange {
		hotPct = 40
		coldPct = 60
	}

	volumeMB := 12.4 + float64(dimensionCount*2) + float64(measureCount*4)
	computeUSD := volumeMB * 0.00015
	latencyMs := 120 + (complexity * 2)

	dagSteps := []ExplainDAGStep{
		{
			StepID:      "step-01",
			Name:        "Partition Pruning & Filter Pushdown",
			Engine:      "StarRocks",
			Operation:   "PartitionPrune",
			Cost:        1.2,
			RowCount:    154000,
			Description: fmt.Sprintf("Pruned to active tenant partitions with %d pushdown predicates", filterCount),
		},
		{
			StepID:      "step-02",
			Name:        "Hot/Cold Lakehouse Federation",
			Engine:      "Iceberg W_t",
			Operation:   "FederatedJoin",
			Cost:        4.5,
			RowCount:    32000,
			Description: fmt.Sprintf("Federated union across StarRocks (%d%%) and Iceberg (%d%%)", hotPct, coldPct),
		},
		{
			StepID:      "step-03",
			Name:        "Vectorized Aggregation & Semantic Mapping",
			Engine:      "WASM / Compute",
			Operation:   "Aggregate",
			Cost:        2.1,
			RowCount:    120,
			Description: fmt.Sprintf("Evaluated %d measures and %d dimensions", measureCount, dimensionCount),
		},
	}

	return &PreflightFinOpsEstimate{
		HotPercentage:   hotPct,
		ColdPercentage:  coldPct,
		EstLatencyMs:    latencyMs,
		ScannedVolumeMB: volumeMB,
		EstComputeUSD:   computeUSD,
		ComplexityScore: complexity,
		PassesBreaker:   passesBreaker,
		BreakerMessage:  breakerMsg,
		ExplainDAGSteps: dagSteps,
	}, nil
}
