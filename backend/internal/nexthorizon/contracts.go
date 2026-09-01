package nexthorizon

import (
	"context"

	"github.com/google/uuid"
)

type RuleConstraint struct {
	DimensionKey string  `json:"dimensionKey"`
	Operator    string  `json:"operator"`
	ValuePct    float64 `json:"valuePct"`
}

type SMTResult struct {
	IsSatisfiable     bool     `json:"isSatisfiable"`
	ConflictDetected  bool     `json:"conflictDetected"`
	DiagnosticMessage string   `json:"diagnosticMessage"`
	CounterExample    []string `json:"counterExample,omitempty"`
}

type AdaptivePlan struct {
	OriginalStrategy     string `json:"originalStrategy"`
	AdaptedStrategy      string `json:"adaptedStrategy"`
	EstimatedRows        int64  `json:"estimatedRows"`
	ActualFilteredRows   int64  `json:"actualFilteredRows"`
	BroadcastThreshold   int64  `json:"broadcastThreshold"`
	PrunedS3Splits       int    `json:"prunedS3Splits"`
	TotalS3Splits        int    `json:"totalS3Splits"`
	DynamicPruningActive bool   `json:"dynamicPruningActive"`
	PlanNotes            string `json:"planNotes"`
}

type PrivacyResult struct {
	MetricName     string  `json:"metricName"`
	NoisyValue     float64 `json:"noisyValue"`
	EpsilonUsed    float64 `json:"epsilonUsed"`
	Sensitivity    float64 `json:"sensitivity"`
	ZKPassportHash string  `json:"zkPassportHash"`
}

type MandateSMTVerifierPort interface {
	VerifyMandateConsistency(ctx context.Context, tenantID uuid.UUID, rules []RuleConstraint) (*SMTResult, error)
}

type AdaptiveExecutionPort interface {
	AdaptPlanAtRuntime(ctx context.Context, tenantID uuid.UUID, initialRows, actualRows int64, totalSplits int) (*AdaptivePlan, error)
}

type DifferentialPrivacyPort interface {
	ApplyLaplaceNoise(ctx context.Context, tenantID uuid.UUID, metric string, rawVal, sensitivity, epsilon float64) (*PrivacyResult, error)
}

type ArrowFlightStreamPort interface {
	StreamRecordBatches(ctx context.Context, tenantID uuid.UUID, ticket []byte) error
}
