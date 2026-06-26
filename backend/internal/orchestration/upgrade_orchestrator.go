package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/metadata"
	"github.com/hondyman/semlayer/backend/internal/testing"
	"github.com/jmoiron/sqlx"
)

// UpgradeOrchestrator manages the full upgrade pipeline with evidence bundle generation
type UpgradeOrchestrator struct {
	diffEngine      *metadata.DiffEngine
	rebaseService   *metadata.RebaseService
	testGenerator   *testing.RegressionTestGenerator
	evidenceService *metadata.EvidenceBundleService
	approvalService *metadata.ApprovalService
	db              *sqlx.DB
}

// NewUpgradeOrchestrator creates a new orchestrator with evidence bundle support
func NewUpgradeOrchestrator(db *sqlx.DB) *UpgradeOrchestrator {
	return &UpgradeOrchestrator{
		diffEngine:      metadata.NewDiffEngine(),
		rebaseService:   metadata.NewRebaseService(),
		testGenerator:   testing.NewRegressionTestGenerator(),
		evidenceService: metadata.NewEvidenceBundleService(db),
		approvalService: metadata.NewApprovalService(db),
		db:              db,
	}
}

// UpgradeRequest defines the input for the upgrade pipeline
type UpgradeRequest struct {
	ID             uuid.UUID
	OldCoreVersion string
	NewCoreVersion string
	TargetTenants  []string
	RequestedBy    string
}

// UpgradeResult contains the outcome of the upgrade with evidence bundle reference
type UpgradeResult struct {
	SuccessCount int
	FailureCount int
	ReportID     string
	SnapshotID   string
	EvidenceBundleID uuid.UUID
}

// UpgradePipeline orchestrates the full metadata upgrade process with evidence generation
func (o *UpgradeOrchestrator) UpgradePipeline(ctx context.Context, req UpgradeRequest) (*UpgradeResult, error) {
	// 0. Create evidence bundle
	bundle, err := o.evidenceService.CreateBundle(ctx, req.ID, req.OldCoreVersion, req.NewCoreVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to create evidence bundle: %w", err)
	}

	// 1. Load old and new core metadata
	oldCore, err := o.loadCoreMetadata(req.OldCoreVersion)
	if err != nil {
		o.evidenceService.CompleteBundle(ctx, bundle.ID, metadata.BundleStatusFailed)
		return nil, fmt.Errorf("failed to load old core: %w", err)
	}

	newCore, err := o.loadCoreMetadata(req.NewCoreVersion)
	if err != nil {
		o.evidenceService.CompleteBundle(ctx, bundle.ID, metadata.BundleStatusFailed)
		return nil, fmt.Errorf("failed to load new core: %w", err)
	}

	// 2. Diff core versions and generate evidence
	diffs := o.diffCore(oldCore, newCore)
	diffReport := metadata.GenerateDiffReport(oldCore[0], newCore[0])
	
	diffArtifacts, err := o.createDiffArtifacts(diffReport)
	if err != nil {
		return nil, fmt.Errorf("failed to create diff artifacts: %w", err)
	}
	
	if err := o.evidenceService.RecordStage(ctx, bundle.ID, metadata.StageDiff, diffArtifacts, "system"); err != nil {
		return nil, fmt.Errorf("failed to record diff stage: %w", err)
	}
	o.evidenceService.UpdateStageStatus(ctx, bundle.ID, metadata.StageDiff, metadata.StageStatusSuccess)

	o.logUAR("DiffComputed", "system", map[string]interface{}{"diffs": len(diffs), "bundle_id": bundle.ID})

	// 3. Load tenant overlays
	overlays, err := o.loadTenantOverlays(req.TargetTenants)
	if err != nil {
		o.evidenceService.CompleteBundle(ctx, bundle.ID, metadata.BundleStatusFailed)
		return nil, fmt.Errorf("failed to load overlays: %w", err)
	}

	// 4. Rebase overlays and generate evidence
	var rebasedOverlays []metadata.RebaseResult
	var allConflicts []metadata.Conflict
	
	for _, overlay := range overlays {
		rebased := o.rebaseService.RebaseBusinessObject(oldCore[0], newCore[0], overlay)
		rebasedOverlays = append(rebasedOverlays, rebased)
		allConflicts = append(allConflicts, rebased.Conflicts...)
		o.logUAR("OverlayRebased", overlay.Meta.TenantID, rebased)
	}

	rebaseArtifacts, err := o.createRebaseArtifacts(allConflicts)
	if err != nil {
		return nil, fmt.Errorf("failed to create rebase artifacts: %w", err)
	}
	
	if err := o.evidenceService.RecordStage(ctx, bundle.ID, metadata.StageRebase, rebaseArtifacts, "system"); err != nil {
		return nil, fmt.Errorf("failed to record rebase stage: %w", err)
	}
	o.evidenceService.UpdateStageStatus(ctx, bundle.ID, metadata.StageRebase, metadata.StageStatusSuccess)

	// 5. Deploy to sandbox and run regression tests
	successCount := 0
	failureCount := 0
	var allTestResults []testing.TestResult

	for _, rebased := range rebasedOverlays {
		// Deploy to sandbox
		if err := o.deploySandbox(rebased); err != nil {
			o.logUAR("SandboxDeployFailed", rebased.RebasedBO.Meta.TenantID, err)
			failureCount++
			continue
		}

		// Generate and run tests
		tests := o.testGenerator.GenerateTests([]metadata.BusinessObject{*rebased.RebasedBO})
		results, err := o.testGenerator.RunSuite(tests)
		if err != nil {
			o.logUAR("RegressionTestFailed", rebased.RebasedBO.Meta.TenantID, err)
			failureCount++
			continue
		}

		allTestResults = append(allTestResults, results...)

		// Check for failures
		hasFailures := false
		for _, result := range results {
			if result.Status == "FAIL" {
				hasFailures = true
				break
			}
		}

		if hasFailures || len(rebased.Conflicts) > 0 {
			o.logUAR("UpgradeBlocked", rebased.RebasedBO.Meta.TenantID, results)
			failureCount++
			continue
		}

		successCount++
	}

	// Generate test evidence
	testArtifacts, err := o.createTestArtifacts(allTestResults)
	if err != nil {
		return nil, fmt.Errorf("failed to create test artifacts: %w", err)
	}
	
	if err := o.evidenceService.RecordStage(ctx, bundle.ID, metadata.StageTest, testArtifacts, "system"); err != nil {
		return nil, fmt.Errorf("failed to record test stage: %w", err)
	}
	o.evidenceService.UpdateStageStatus(ctx, bundle.ID, metadata.StageTest, metadata.StageStatusSuccess)

	// 6. Request approval for breaking changes
	if diffReport.Summary.BreakingChanges > 0 {
		approvalReq, err := o.approvalService.RequestApproval(ctx, bundle.ID, req.RequestedBy, "data_steward")
		if err != nil {
			return nil, fmt.Errorf("failed to request approval: %w", err)
		}

		approvalArtifacts := []metadata.Artifact{
			{
				Type:        metadata.ArtifactTypeApprovalRecord,
				StoragePath: fmt.Sprintf("s3://evidence/%s/approval-%s.json", bundle.ID, approvalReq.ID),
				Metadata:    []byte(fmt.Sprintf(`{"request_id":"%s","breaking_changes":%d}`, approvalReq.ID, diffReport.Summary.BreakingChanges)),
				CreatedAt:   time.Now(),
			},
		}
		
		if err := o.evidenceService.RecordStage(ctx, bundle.ID, metadata.StageApproval, approvalArtifacts, req.RequestedBy); err != nil {
			return nil, fmt.Errorf("failed to record approval stage: %w", err)
		}

		// In production: wait for approval decision
		// For now, we'll mark as pending
		o.evidenceService.UpdateStageStatus(ctx, bundle.ID, metadata.StageApproval, metadata.StageStatusPending)
		o.logUAR("ApprovalRequested", req.RequestedBy, approvalReq)
	} else {
		// Auto-approve for non-breaking changes
		o.evidenceService.UpdateStageStatus(ctx, bundle.ID, metadata.StageApproval, metadata.StageStatusSkipped)
	}

	// 7. Deploy to production
	for _, rebased := range rebasedOverlays {
		if err := o.deployProduction(rebased); err != nil {
			o.logUAR("ProductionDeployFailed", rebased.RebasedBO.Meta.TenantID, err)
			continue
		}
		o.logUAR("ProductionDeployed", rebased.RebasedBO.Meta.TenantID, rebased)
	}

	deploymentArtifacts := o.createDeploymentArtifacts(req.TargetTenants, successCount, failureCount)
	if err := o.evidenceService.RecordStage(ctx, bundle.ID, metadata.StageDeploy, deploymentArtifacts, "system"); err != nil {
		return nil, fmt.Errorf("failed to record deploy stage: %w", err)
	}
	o.evidenceService.UpdateStageStatus(ctx, bundle.ID, metadata.StageDeploy, metadata.StageStatusSuccess)

	// 8. Create Iceberg snapshot
	snapshotID := o.exportSnapshotToIceberg(newCore)
	snapshotArtifact := metadata.Artifact{
		Type:        metadata.ArtifactTypeIcebergSnapshot,
		StoragePath: fmt.Sprintf("iceberg://metadata/snapshots/%s", snapshotID),
		Metadata:    []byte(fmt.Sprintf(`{"snapshot_id":"%s","version":"%s"}`, snapshotID, req.NewCoreVersion)),
		CreatedAt:   time.Now(),
	}
	
	if err := o.evidenceService.RecordStage(ctx, bundle.ID, metadata.StageAudit, []metadata.Artifact{snapshotArtifact}, "system"); err != nil {
		return nil, fmt.Errorf("failed to record audit stage: %w", err)
	}
	o.evidenceService.UpdateStageStatus(ctx, bundle.ID, metadata.StageAudit, metadata.StageStatusSuccess)
	o.logUAR("SnapshotExported", "system", snapshotID)

	// Complete evidence bundle
	bundleStatus := metadata.BundleStatusCompleted
	if failureCount > 0 {
		bundleStatus = metadata.BundleStatusFailed
	}
	o.evidenceService.CompleteBundle(ctx, bundle.ID, bundleStatus)

	reportID := fmt.Sprintf("report-%s-%d", req.NewCoreVersion, time.Now().Unix())

	return &UpgradeResult{
		SuccessCount:     successCount,
		FailureCount:     failureCount,
		ReportID:         reportID,
		SnapshotID:       snapshotID,
		EvidenceBundleID: bundle.ID,
	}, nil
}

// Helper methods for artifact creation

func (o *UpgradeOrchestrator) createDiffArtifacts(diffReport *metadata.DiffReport) ([]metadata.Artifact, error) {
	diffJSON, err := json.Marshal(diffReport)
	if err != nil {
		return nil, err
	}

	metadataJSON, _ := json.Marshal(map[string]interface{}{
		"breaking_changes": diffReport.Summary.BreakingChanges,
		"additive_changes": diffReport.Summary.AdditiveChanges,
		"safe_changes":     diffReport.Summary.SafeChanges,
	})

	artifact := metadata.Artifact{
		Type:        metadata.ArtifactTypeDiffReport,
		StoragePath: fmt.Sprintf("s3://evidence/diff-%d.json", time.Now().Unix()),
		Checksum:    metadata.ComputeChecksum(diffJSON),
		Metadata:    metadataJSON,
		CreatedAt:   time.Now(),
		SizeBytes:   int64(len(diffJSON)),
	}

	return []metadata.Artifact{artifact}, nil
}

func (o *UpgradeOrchestrator) createRebaseArtifacts(conflicts []metadata.Conflict) ([]metadata.Artifact, error) {
	conflictJSON, err := json.Marshal(conflicts)
	if err != nil {
		return nil, err
	}

	metadataJSON, _ := json.Marshal(map[string]interface{}{
		"conflict_count": len(conflicts),
	})

	artifact := metadata.Artifact{
		Type:        metadata.ArtifactTypeConflictReport,
		StoragePath: fmt.Sprintf("s3://evidence/conflicts-%d.json", time.Now().Unix()),
		Checksum:    metadata.ComputeChecksum(conflictJSON),
		Metadata:    metadataJSON,
		CreatedAt:   time.Now(),
		SizeBytes:   int64(len(conflictJSON)),
	}

	return []metadata.Artifact{artifact}, nil
}

func (o *UpgradeOrchestrator) createTestArtifacts(results []testing.TestResult) ([]metadata.Artifact, error) {
	// Calculate test summary
	passed := 0
	failed := 0
	for _, result := range results {
		if result.Status == "PASS" {
			passed++
		} else {
			failed++
		}
	}

	passRate := 0.0
	if len(results) > 0 {
		passRate = float64(passed) / float64(len(results)) * 100.0
	}

	metadataJSON, _ := json.Marshal(map[string]interface{}{
		"pass_rate":    passRate,
		"total_tests":  len(results),
		"passed_tests": passed,
		"failed_tests": failed,
	})

	resultsJSON, err := json.Marshal(results)
	if err != nil {
		return nil, err
	}

	artifact := metadata.Artifact{
		Type:        metadata.ArtifactTypeTestResults,
		StoragePath: fmt.Sprintf("s3://evidence/test-results-%d.json", time.Now().Unix()),
		Checksum:    metadata.ComputeChecksum(resultsJSON),
		Metadata:    metadataJSON,
		CreatedAt:   time.Now(),
		SizeBytes:   int64(len(resultsJSON)),
	}

	return []metadata.Artifact{artifact}, nil
}

func (o *UpgradeOrchestrator) createDeploymentArtifacts(tenants []string, successCount, failureCount int) []metadata.Artifact {
	metadataJSON, _ := json.Marshal(map[string]interface{}{
		"target_tenants":     tenants,
		"successful_deploys": successCount,
		"failed_deploys":     failureCount,
	})

	return []metadata.Artifact{
		{
			Type:        metadata.ArtifactTypeDeploymentLog,
			StoragePath: fmt.Sprintf("s3://evidence/deployment-%d.json", time.Now().Unix()),
			Metadata:    metadataJSON,
			CreatedAt:   time.Now(),
		},
	}
}

// Existing helper methods

func (o *UpgradeOrchestrator) loadCoreMetadata(version string) ([]metadata.BusinessObject, error) {
	// In production: query meta_objects table
	return []metadata.BusinessObject{}, nil
}

func (o *UpgradeOrchestrator) loadTenantOverlays(tenantIDs []string) ([]metadata.BusinessObject, error) {
	// In production: query meta_objects with tenant_id
	return []metadata.BusinessObject{}, nil
}

func (o *UpgradeOrchestrator) diffCore(oldCore, newCore []metadata.BusinessObject) []metadata.Diff {
	if len(oldCore) == 0 || len(newCore) == 0 {
		return []metadata.Diff{}
	}
	return o.diffEngine.DiffBusinessObjects(oldCore[0], newCore[0])
}

func (o *UpgradeOrchestrator) deploySandbox(rebased metadata.RebaseResult) error {
	// In production: deploy to sandbox DB partition
	fmt.Printf("Deploying to sandbox for tenant %s\n", rebased.RebasedBO.Meta.TenantID)
	return nil
}

func (o *UpgradeOrchestrator) deployProduction(rebased metadata.RebaseResult) error {
	// In production: deploy to production DB
	// Execute update
	return nil
}

func (o *UpgradeOrchestrator) exportSnapshotToIceberg(core []metadata.BusinessObject) string {
	// In production: export to Iceberg/Parquet
	snapshotID := fmt.Sprintf("iceberg-snap-%d", time.Now().Unix())
	fmt.Printf("Exporting snapshot %s\n", snapshotID)
	return snapshotID
}

func (o *UpgradeOrchestrator) logUAR(action, actorID string, data interface{}) {
	// Log to audit_records table
	fmt.Printf("UAR Log: %s by %s\n", action, actorID)
}
