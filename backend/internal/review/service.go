package review

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/lineage"
	"github.com/hondyman/semlayer/backend/internal/semantic"
	"github.com/jmoiron/sqlx"
	"go.temporal.io/sdk/client"
)

// LineageAnalyzer defines interface for impact analysis
type LineageAnalyzer interface {
	ImpactOfNode(ctx context.Context, nodeID string, depth int) (*lineage.ImpactReport, error)
	InvalidateASO(ctx context.Context, nodeID string, invalidator lineage.ASOInvalidator) error
	RebuildForObject(ctx context.Context, objType, objID string, payload []byte) error
}

// VersionManager defines interface for versioning
type VersionManager interface {
	Diff(ctx context.Context, id string, from, to int) (*semantic.SemanticDiffDTO, error)
	GetVersion(ctx context.Context, id string, version int) (*semantic.SemanticObject, error)
	SaveObject(ctx context.Context, obj semantic.SemanticObject, actor string) error
	Rollback(ctx context.Context, id string, targetVersion int, actor string) error
}

// TestRunner defines interface for running tests
type TestRunner interface {
	RunTest(ctx context.Context, test semantic.SemanticTest) (*semantic.TestResult, error)
}

// ChangeReviewService orchestrates the review process
type ChangeReviewService struct {
	db             *sqlx.DB
	lineage        LineageAnalyzer
	versions       VersionManager
	runner         TestRunner
	asoInvalidator lineage.ASOInvalidator
	temporal       client.Client
}

// NewChangeReviewService creates a new review service
func NewChangeReviewService(db *sqlx.DB, lineage LineageAnalyzer, versions VersionManager, runner TestRunner, asoInv lineage.ASOInvalidator, temporal client.Client) *ChangeReviewService {
	return &ChangeReviewService{
		db:             db,
		lineage:        lineage,
		versions:       versions,
		runner:         runner,
		asoInvalidator: asoInv,
		temporal:       temporal,
	}
}

// GetReviewForChangeSet retrieves the latest review for a changeset
func (s *ChangeReviewService) GetReviewForChangeSet(ctx context.Context, changeSetID uuid.UUID) (*ChangeReview, error) {
	var review ChangeReview
	err := s.db.GetContext(ctx, &review, `
		SELECT * FROM semantic.change_reviews 
		WHERE change_set_id=$1 
		ORDER BY created_at DESC LIMIT 1
	`, changeSetID)
	if err != nil {
		return nil, err
	}

	// Dynamic AI Enrichment: Fetch latest AI Summary
	// We look for a successful AI request associated with this ChangeSet
	// This assumes the payload stores "change_set_id" as a string
	type AIResult struct {
		Output json.RawMessage `db:"output"`
	}
	var aiReq AIResult
	// JSON path query might be database-specific (PostgreSQL syntax used here)
	err = s.db.GetContext(ctx, &aiReq, `
		SELECT output FROM ai_requests 
		WHERE payload->>'change_set_id' = $1 
		AND type = 'changeset'
		AND status = 'SUCCESS'
		ORDER BY created_at DESC LIMIT 1
	`, changeSetID.String())

	if err == nil {
		// Parse output
		var output struct {
			Summary   string  `json:"summary"`
			RiskScore float64 `json:"risk_score"`
			RiskLevel string  `json:"risk_level"`
		}
		if json.Unmarshal(aiReq.Output, &output) == nil {
			review.AISummary = output.Summary
			review.AIRiskScore = output.RiskScore
			review.AIRiskLevel = output.RiskLevel
		}
	} else {
		// Log debug? Ignore validation missing?
		// Just leave fields empty if no AI result yet
	}

	return &review, nil
}

// CreateReview generates a review artifact for a ChangeSet
func (s *ChangeReviewService) CreateReview(ctx context.Context, changeSetID uuid.UUID) (*ChangeReview, error) {
	// Fetch Change Set Items
	var items []semantic.ChangeSetItem
	err := s.db.SelectContext(ctx, &items, `SELECT * FROM semantic.change_set_items WHERE change_set_id=$1`, changeSetID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch change set items: %w", err)
	}

	review := &ChangeReview{
		ID:          uuid.New(),
		ChangeSetID: changeSetID,
		Status:      "pending",
		CreatedAt:   time.Now(),
	}

	// 1. Analyze Semantics Diff
	diffs := make(map[string]semantic.SemanticDiffDTO)
	for _, item := range items {
		diff, err := s.versions.Diff(ctx, item.ObjectID, item.OldVersion, item.NewVersion)
		if err != nil {
			return nil, fmt.Errorf("diff failed for %s: %w", item.ObjectID, err)
		}
		if diff != nil {
			diffs[item.ObjectID] = *diff
		}
	}
	diffJSON, _ := json.Marshal(diffs)
	review.SemanticDiff = diffJSON

	// 2. Analyze Lineage Impact
	impacts := make(map[string]*lineage.ImpactReport)
	for _, item := range items {
		report, err := s.lineage.ImpactOfNode(ctx, item.ObjectID, 5)
		if err != nil {
			return nil, fmt.Errorf("impact analysis failed for %s: %w", item.ObjectID, err)
		}
		impacts[item.ObjectID] = report
	}
	impactJSON, _ := json.Marshal(impacts)
	review.LineageImpact = impactJSON

	// 3. Run Semantic Tests (Regression)
	// Fetch relevant tests based on affected nodes
	var results []semantic.TestResult
	// For each affected BO, find tests
	for _, report := range impacts {
		for _, bo := range report.AffectedBOs {
			tests, err := s.fetchTestsForScope(ctx, "bo", bo.ID)
			if err != nil {
				continue
			}
			for _, test := range tests {
				res, err := s.runner.RunTest(ctx, test)
				if err != nil {
					// Log error but continue? Or fail review?
					continue
				}
				results = append(results, *res)
			}
		}
	}
	resultsJSON, _ := json.Marshal(results)
	review.TestResults = resultsJSON

	// Save Review
	_, err = s.db.NamedExecContext(ctx, `
		INSERT INTO semantic.change_reviews (id, change_set_id, lineage_impact, semantic_diff, test_results, status, created_at)
		VALUES (:id, :change_set_id, :lineage_impact, :semantic_diff, :test_results, :status, :created_at)
	`, review)
	if err != nil {
		return nil, err
	}

	return review, nil
}

// Promote triggers the promotion workflow
func (s *ChangeReviewService) Promote(ctx context.Context, changeSetID uuid.UUID, actor string) error {
	var review ChangeReview
	// Check approval status
	err := s.db.GetContext(ctx, &review, `
		SELECT * FROM semantic.change_reviews 
		WHERE change_set_id=$1 
		ORDER BY created_at DESC LIMIT 1
	`, changeSetID)
	if err != nil {
		return err
	}

	if review.Status != "approved" {
		return fmt.Errorf("review not approved")
	}

	if s.temporal == nil {
		return fmt.Errorf("temporal client not configured")
	}

	// Trigger Workflow
	workflowOptions := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("promote-%s", changeSetID),
		TaskQueue: "semantic-worker", // Todo: config
	}

	_, err = s.temporal.ExecuteWorkflow(ctx, workflowOptions, PromoteChangeSetWorkflow, changeSetID)
	return err
}

// Rollback rolls back an object to a specific version
func (s *ChangeReviewService) Rollback(ctx context.Context, objectID string, targetVersion int, actor string) error {
	if err := s.versions.Rollback(ctx, objectID, targetVersion, actor); err != nil {
		return err
	}

	// Rebuild lineage immediately for rollback (or trigger a workflow)
	obj, err := s.versions.GetVersion(ctx, objectID, -1) // get new head
	if err != nil {
		return err
	}

	// We assume obj type is available or derived
	if err := s.lineage.RebuildForObject(ctx, obj.Type, obj.ID, obj.Payload); err != nil {
		// Log error but rollback succeeded
		fmt.Printf("Warning: failed to rebuild lineage during rollback: %v\n", err)
	}

	if s.asoInvalidator != nil {
		_ = s.lineage.InvalidateASO(ctx, objectID, s.asoInvalidator)
	}

	return nil
}

func (s *ChangeReviewService) fetchTestsForScope(ctx context.Context, scopeType, scopeID string) ([]semantic.SemanticTest, error) {
	var tests []semantic.SemanticTest
	err := s.db.SelectContext(ctx, &tests, `
		SELECT * FROM semantic.tests WHERE scope_type=$1 AND scope_id=$2 AND enabled=true
	`, scopeType, scopeID)
	return tests, err
}
