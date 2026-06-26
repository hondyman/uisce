package review

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/semantic"
)

// ChangeReview represents the complete review artifact for a ChangeSet
type ChangeReview struct {
	ID                 uuid.UUID       `json:"id" db:"id"`
	ChangeSetID        uuid.UUID       `json:"change_set_id" db:"change_set_id"`
	LineageImpact      json.RawMessage `json:"lineage_impact" db:"lineage_impact"`
	SemanticDiff       json.RawMessage `json:"semantic_diff" db:"semantic_diff"`
	TestResults        json.RawMessage `json:"test_results" db:"test_results"`
	ASOImpact          json.RawMessage `json:"aso_impact" db:"aso_impact"`
	ApiBreakingChanges json.RawMessage `json:"api_breaking_changes" db:"api_breaking_changes"`
	Reviewer           *string         `json:"reviewer" db:"reviewer"`
	Status             string          `json:"status" db:"status"` // pending | approved | rejected
	CreatedAt          time.Time       `json:"created_at" db:"created_at"`

	// AI Enrichment (Hydrated dynamically)
	AISummary   string  `json:"ai_summary" db:"-"`
	AIRiskScore float64 `json:"ai_risk_score" db:"-"`
	AIRiskLevel string  `json:"ai_risk_level" db:"-"`
}

// ImpactReportDTO is a helper struct for unmarshalling LineageImpact
type ImpactReportDTO struct {
	AffectedBOs              []string `json:"affected_bos"`
	AffectedPreAggs          []string `json:"affected_preaggs"`
	AffectedEntitlements     []string `json:"affected_entitlements"`
	AffectedASOOptimizations []string `json:"affected_aso_optimizations"`
	AffectedTenants          []string `json:"affected_tenants"`
}

// TestResultsDTO represents the unmarshalled test results
type TestResultsDTO struct {
	Tests []semantic.TestResult `json:"tests"`
}

// ASOImpactDTO represents the unmarshalled ASO impact
type ASOImpactDTO struct {
	StaleOptimizations []string `json:"stale_optimizations"`
	NeedsRecompute     []string `json:"needs_recompute"`
	InvalidPreAggs     []string `json:"invalid_preaggs"`
	PolicyTriggers     []string `json:"policy_triggers"`
}
