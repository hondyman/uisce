package reviewer

import (
	"context"

	"github.com/google/uuid"
)

type RiskFlag struct {
	Type        string `json:"type"`     // semantic_drift, breaking_change, pii_exposure, slo_regression
	Severity    string `json:"severity"` // critical, high, medium, low
	Description string `json:"description"`
}

type ReviewerSuggestion struct {
	ReviewerName string  `json:"reviewer_name"`
	Reason       string  `json:"reason"`
	Confidence   float64 `json:"confidence"`
}

type AIReviewReport struct {
	ChangeSetID        uuid.UUID            `json:"changeset_id"`
	Summary            string               `json:"summary"`
	RiskFlags          []RiskFlag           `json:"risk_flags"`
	SuggestedReviewers []ReviewerSuggestion `json:"suggested_reviewers"`
	ApprovalLikelihood float64              `json:"approval_likelihood"` // 0.0-1.0
	ComplianceNotes    string               `json:"compliance_notes"`
}

type AIReviewer struct {
	// LLM integration, lineage engine, SLO metrics
}

func NewAIReviewer() *AIReviewer {
	return &AIReviewer{}
}

func (r *AIReviewer) Review(ctx context.Context, changesetID uuid.UUID) (*AIReviewReport, error) {
	// Mock: Generate AI review
	// Real: Analyze changeset diff, lineage, SLOs, semantic quality, tenant overlays

	report := &AIReviewReport{
		ChangeSetID: changesetID,
		Summary:     "This ChangeSet modifies 3 BOs (Position, Account, Trade), 2 APIs (positions_api, accounts_api), and 1 page (Positions Dashboard). No breaking changes detected. Low risk.",
		RiskFlags: []RiskFlag{
			{
				Type:        "slo_regression",
				Severity:    "medium",
				Description: "Positions Dashboard p95 render time may increase by 50ms",
			},
		},
		SuggestedReviewers: []ReviewerSuggestion{
			{
				ReviewerName: "Sarah Chen",
				Reason:       "Owns Positions domain and has reviewed 15 similar changes",
				Confidence:   0.92,
			},
			{
				ReviewerName: "Michael Torres",
				Reason:       "Subject matter expert in trading workflows",
				Confidence:   0.78,
			},
		},
		ApprovalLikelihood: 0.92,
		ComplianceNotes:    "No PII exposure detected. Residency rules unchanged. All data policies compliant. No regulatory impact.",
	}

	return report, nil
}
