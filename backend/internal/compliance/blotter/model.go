package blotter

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrProvenanceVerificationFailed is returned when snapshot lookup fails or content hash verification fails.
var ErrProvenanceVerificationFailed = errors.New("provenance verification failed: rule snapshot mismatch or cryptographic hash corruption")

// EvaluationEventRecord models a single compliance evaluation event row joined with rule metadata.
type EvaluationEventRecord struct {
	ID              uuid.UUID       `json:"id"`
	LineageID       uuid.UUID       `json:"lineage_id"`
	TenantID        uuid.UUID       `json:"tenant_id"`
	OrderID         *uuid.UUID      `json:"order_id,omitempty"`
	RuleID          uuid.UUID       `json:"rule_id"`
	RuleCode        string          `json:"rule_code"`
	RuleName        string          `json:"rule_name"`
	RuleVersion     int             `json:"rule_version"`
	RulePhase       string          `json:"rule_phase"`
	Severity        string          `json:"severity"`
	Passed          bool            `json:"passed"`
	ActionTaken     string          `json:"action_taken"`
	LatencyMicros   int64           `json:"latency_micros"`
	RuleContentHash string          `json:"rule_content_hash"`
	EvaluationHash  string          `json:"evaluation_hash"`
	InputParams     json.RawMessage `json:"input_params"`
	MetricSnapshots json.RawMessage `json:"metric_snapshots"`
	EvaluatedAt     time.Time       `json:"evaluated_at"`
	CreatedAt       time.Time       `json:"created_at"`
}

// RuleSnapshotInfo contains immutable rule definition at the time of evaluation.
type RuleSnapshotInfo struct {
	RuleID              uuid.UUID       `json:"rule_id"`
	Version             int             `json:"version"`
	ContentHash         string          `json:"content_hash"`
	ResolvedAST         json.RawMessage `json:"resolved_ast"`
	ParameterThresholds json.RawMessage `json:"parameter_thresholds"`
	Citation            string          `json:"citation"`
}

// EvaluatedMetricItem provides structured comparison between observed metric and threshold.
type EvaluatedMetricItem struct {
	MetricPath     string `json:"metric_path"`
	ObservedValue  any    `json:"observed_value"`
	ThresholdValue any    `json:"threshold_value,omitempty"`
	Operator       string `json:"operator,omitempty"`
	Breached       bool   `json:"breached"`
	Margin         string `json:"margin,omitempty"`
}

// IntegrityProof documents cryptographic validation of the decision record.
type IntegrityProof struct {
	ContentHashMatches    bool   `json:"content_hash_matches"`
	StoredContentHash     string `json:"stored_content_hash"`
	RecomputedContentHash string `json:"recomputed_content_hash"`
	EvaluationHash        string `json:"evaluation_hash"`
	Authority             string `json:"authority"`
}

// DecisionEvidenceBundle is the complete evidence artifact for cold archive, auditor review, and UI explainability.
type DecisionEvidenceBundle struct {
	Evaluation                 EvaluationEventRecord `json:"evaluation"`
	RuleSnapshot               RuleSnapshotInfo      `json:"rule_snapshot"`
	Metrics                    []EvaluatedMetricItem `json:"metrics"`
	NaturalLanguageExplanation string                `json:"natural_language_explanation"`
	IntegrityProof             IntegrityProof        `json:"integrity_proof"`
}

// ListFilter parameters for querying evaluation history.
type ListFilter struct {
	TenantID    uuid.UUID  `json:"tenant_id"`
	OrderID     *uuid.UUID `json:"order_id,omitempty"`
	LineageID   *uuid.UUID `json:"lineage_id,omitempty"`
	RuleCode    string     `json:"rule_code,omitempty"`
	ActionTaken string     `json:"action_taken,omitempty"`
	Passed      *bool      `json:"passed,omitempty"`
	From        *time.Time `json:"from,omitempty"`
	To          *time.Time `json:"to,omitempty"`
	Page        int        `json:"page"`
	PageSize    int        `json:"page_size"`
}

// PaginatedResponse wraps items with total counts and pagination details.
type PaginatedResponse[T any] struct {
	Data       []T   `json:"data"`
	TotalCount int64 `json:"total_count"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int   `json:"total_pages"`
}
