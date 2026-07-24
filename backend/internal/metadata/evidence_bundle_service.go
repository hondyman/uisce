package metadata

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// EvidenceBundleService manages evidence collection throughout upgrade lifecycle
type EvidenceBundleService struct {
	db *sqlx.DB
}

// NewEvidenceBundleService creates a new evidence bundle service
func NewEvidenceBundleService(db *sqlx.DB) *EvidenceBundleService {
	return &EvidenceBundleService{db: db}
}

// CreateBundle initializes a new evidence bundle for an upgrade request
func (s *EvidenceBundleService) CreateBundle(ctx context.Context, upgradeRequestID uuid.UUID, oldVersion, newVersion string) (*EvidenceBundle, error) {
	bundle := &EvidenceBundle{
		ID:               uuid.New(),
		UpgradeRequestID: upgradeRequestID,
		OldVersion:       oldVersion,
		NewVersion:       newVersion,
		Status:           BundleStatusInProgress,
		CreatedAt:        time.Now(),
	}

	query := `
		INSERT INTO metadata.evidence_bundles (id, upgrade_request_id, old_version, new_version, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := s.db.ExecContext(ctx, query, bundle.ID, bundle.UpgradeRequestID, bundle.OldVersion, bundle.NewVersion, bundle.Status, bundle.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create evidence bundle: %w", err)
	}

	return bundle, nil
}

// RecordStage appends evidence for a specific upgrade stage
func (s *EvidenceBundleService) RecordStage(ctx context.Context, bundleID uuid.UUID, stageName StageName, artifacts []Artifact, actorID string) error {
	stageEvidence := StageEvidence{
		ID:        uuid.New(),
		BundleID:  bundleID,
		StageName: stageName,
		Status:    StageStatusRunning,
		Artifacts: artifacts,
		StartedAt: time.Now(),
		ActorID:   actorID,
	}

	artifactsJSON, err := json.Marshal(stageEvidence.Artifacts)
	if err != nil {
		return fmt.Errorf("failed to marshal artifacts: %w", err)
	}

	query := `
		INSERT INTO metadata.stage_evidence (id, bundle_id, stage_name, status, started_at, actor_id, artifacts)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err = s.db.ExecContext(ctx, query, stageEvidence.ID, stageEvidence.BundleID, stageEvidence.StageName, stageEvidence.Status, stageEvidence.StartedAt, stageEvidence.ActorID, artifactsJSON)
	if err != nil {
		return fmt.Errorf("failed to record stage evidence: %w", err)
	}

	return nil
}

// UpdateStageStatus updates the status and completion time of a stage
func (s *EvidenceBundleService) UpdateStageStatus(ctx context.Context, bundleID uuid.UUID, stageName StageName, status StageStatus) error {
	completedAt := time.Now()
	query := `
		UPDATE metadata.stage_evidence
		SET status = $1, completed_at = $2
		WHERE bundle_id = $3 AND stage_name = $4
	`
	_, err := s.db.ExecContext(ctx, query, status, completedAt, bundleID, stageName)
	if err != nil {
		return fmt.Errorf("failed to update stage status: %w", err)
	}

	return nil
}

// GetBundle retrieves a complete evidence bundle with all stages
func (s *EvidenceBundleService) GetBundle(ctx context.Context, bundleID uuid.UUID) (*EvidenceBundle, error) {
	var bundle EvidenceBundle
	query := `
		SELECT id, upgrade_request_id, old_version, new_version, status, created_at, completed_at
		FROM metadata.evidence_bundles
		WHERE id = $1
	`
	err := s.db.GetContext(ctx, &bundle, query, bundleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get evidence bundle: %w", err)
	}

	// Load stages
	stages, err := s.GetStages(ctx, bundleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get stages: %w", err)
	}
	bundle.Stages = stages

	return &bundle, nil
}

// GetStages retrieves all stage evidence for a bundle
func (s *EvidenceBundleService) GetStages(ctx context.Context, bundleID uuid.UUID) ([]StageEvidence, error) {
	var stages []struct {
		StageEvidence
		ArtifactsJSON json.RawMessage `db:"artifacts"`
	}

	query := `
		SELECT id, bundle_id, stage_name, status, started_at, completed_at, actor_id, artifacts
		FROM metadata.stage_evidence
		WHERE bundle_id = $1
		ORDER BY started_at ASC
	`
	err := s.db.SelectContext(ctx, &stages, query, bundleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get stages: %w", err)
	}

	result := make([]StageEvidence, len(stages))
	for i, stage := range stages {
		result[i] = stage.StageEvidence
		if err := json.Unmarshal(stage.ArtifactsJSON, &result[i].Artifacts); err != nil {
			return nil, fmt.Errorf("failed to unmarshal artifacts: %w", err)
		}
	}

	return result, nil
}

// CompleteBundle marks a bundle as completed or failed
func (s *EvidenceBundleService) CompleteBundle(ctx context.Context, bundleID uuid.UUID, status BundleStatus) error {
	completedAt := time.Now()
	query := `
		UPDATE metadata.evidence_bundles
		SET status = $1, completed_at = $2
		WHERE id = $3
	`
	_, err := s.db.ExecContext(ctx, query, status, completedAt, bundleID)
	if err != nil {
		return fmt.Errorf("failed to complete bundle: %w", err)
	}

	return nil
}

// ExportComplianceReport generates a regulator-facing compliance report
func (s *EvidenceBundleService) ExportComplianceReport(ctx context.Context, bundleID uuid.UUID) (*ComplianceReport, error) {
	bundle, err := s.GetBundle(ctx, bundleID)
	if err != nil {
		return nil, err
	}

	report := &ComplianceReport{
		BundleID:    bundleID,
		GeneratedAt: time.Now(),
	}

	// Build executive summary
	report.ExecutiveSummary = s.buildExecutiveSummary(bundle)

	// Collect change inventory from diff stage
	report.ChangeInventory, err = s.extractChangeInventory(bundle)
	if err != nil {
		return nil, fmt.Errorf("failed to extract change inventory: %w", err)
	}

	// Collect test summary from test stage
	report.TestSummary, err = s.extractTestSummary(bundle)
	if err != nil {
		return nil, fmt.Errorf("failed to extract test summary: %w", err)
	}

	// Collect approval chain
	report.ApprovalChain, err = s.getApprovalChain(ctx, bundleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get approval chain: %w", err)
	}

	// Collect deployment log from deploy stage
	report.DeploymentLog, err = s.extractDeploymentSummary(bundle)
	if err != nil {
		return nil, fmt.Errorf("failed to extract deployment summary: %w", err)
	}

	// Collect all artifacts
	report.Artifacts = s.collectAllArtifacts(bundle)

	return report, nil
}

// Helper methods

func (s *EvidenceBundleService) buildExecutiveSummary(bundle *EvidenceBundle) ExecutiveSummary {
	summary := ExecutiveSummary{
		Status:            string(bundle.Status),
		RiskLevel:         "LOW",
		DeploymentSuccess: bundle.Status == BundleStatusCompleted,
	}

	// Calculate metrics from stages
	for _, stage := range bundle.Stages {
		if stage.StageName == StageDiff {
			// Extract breaking/additive changes from diff artifacts
			for _, artifact := range stage.Artifacts {
				if artifact.Type == ArtifactTypeDiffReport {
					// Parse metadata to get change counts
					var diffMeta struct {
						BreakingChanges int `json:"breaking_changes"`
						AdditiveChanges int `json:"additive_changes"`
					}
					if err := json.Unmarshal(artifact.Metadata, &diffMeta); err == nil {
						summary.BreakingChanges = diffMeta.BreakingChanges
						summary.AdditiveChanges = diffMeta.AdditiveChanges
					}
				}
			}
		}

		if stage.StageName == StageTest {
			// Extract test pass rate
			for _, artifact := range stage.Artifacts {
				if artifact.Type == ArtifactTypeTestResults {
					var testMeta struct {
						PassRate float64 `json:"pass_rate"`
					}
					if err := json.Unmarshal(artifact.Metadata, &testMeta); err == nil {
						summary.TestPassRate = testMeta.PassRate
					}
				}
			}
		}
	}

	// Determine risk level
	if summary.BreakingChanges > 0 {
		summary.RiskLevel = "HIGH"
	} else if summary.AdditiveChanges > 10 {
		summary.RiskLevel = "MEDIUM"
	}

	return summary
}

func (s *EvidenceBundleService) extractChangeInventory(bundle *EvidenceBundle) ([]ChangeRecord, error) {
	for _, stage := range bundle.Stages {
		if stage.StageName == StageDiff {
			for _, artifact := range stage.Artifacts {
				if artifact.Type == ArtifactTypeDiffReport {
					var changes []ChangeRecord
					if err := json.Unmarshal(artifact.Metadata, &changes); err != nil {
						return nil, err
					}
					return changes, nil
				}
			}
		}
	}
	return []ChangeRecord{}, nil
}

func (s *EvidenceBundleService) extractTestSummary(bundle *EvidenceBundle) (TestSummary, error) {
	for _, stage := range bundle.Stages {
		if stage.StageName == StageTest {
			for _, artifact := range stage.Artifacts {
				if artifact.Type == ArtifactTypeTestResults {
					var summary TestSummary
					if err := json.Unmarshal(artifact.Metadata, &summary); err != nil {
						return TestSummary{}, err
					}
					return summary, nil
				}
			}
		}
	}
	return TestSummary{}, nil
}

func (s *EvidenceBundleService) extractDeploymentSummary(bundle *EvidenceBundle) (DeploymentSummary, error) {
	for _, stage := range bundle.Stages {
		if stage.StageName == StageDeploy {
			for _, artifact := range stage.Artifacts {
				if artifact.Type == ArtifactTypeDeploymentLog {
					var summary DeploymentSummary
					if err := json.Unmarshal(artifact.Metadata, &summary); err != nil {
						return DeploymentSummary{}, err
					}
					return summary, nil
				}
			}
		}
	}
	return DeploymentSummary{}, nil
}

func (s *EvidenceBundleService) getApprovalChain(ctx context.Context, bundleID uuid.UUID) ([]ApprovalDecision, error) {
	var decisions []ApprovalDecision
	query := `
		SELECT approver_id, decision, justification, decided_at
		FROM metadata.approval_requests
		WHERE bundle_id = $1 AND decision IS NOT NULL
		ORDER BY decided_at ASC
	`
	err := s.db.SelectContext(ctx, &decisions, query, bundleID)
	if err != nil {
		return nil, err
	}
	return decisions, nil
}

func (s *EvidenceBundleService) collectAllArtifacts(bundle *EvidenceBundle) []Artifact {
	var allArtifacts []Artifact
	for _, stage := range bundle.Stages {
		allArtifacts = append(allArtifacts, stage.Artifacts...)
	}
	return allArtifacts
}

// ComputeChecksum generates a SHA256 checksum for artifact tamper detection
func ComputeChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
