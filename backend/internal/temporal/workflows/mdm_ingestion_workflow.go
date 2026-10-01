package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// MDMIngestScoringInput defines inputs for the full-cycle MDM Ingestion,
// Iceberg Lakehouse Export, Centralized Validation, Mastering, and Vendor Quality Scoring workflow.
type MDMIngestScoringInput struct {
	TenantID      string `json:"tenant_id"`
	Entity        string `json:"entity"`          // e.g. "SECURITY"
	PipelineRunID string `json:"pipeline_run_id"`  // ID of the data pipeline run
	StagingTable  string `json:"staging_table"`   // e.g. "staging.security_data"
	AsOfDate      string `json:"as_of_date"`       // YYYY-MM-DD
	SyncStarRocks bool   `json:"sync_starrocks"`   // sync to StarRocks mdm_analytics
}

// MDMIngestScoringResult encapsulates execution metrics across all stages.
type MDMIngestScoringResult struct {
	PipelineRunID   string  `json:"pipeline_run_id"`
	MasteringRunID  string  `json:"mastering_run_id"`
	GoldenPublished int     `json:"golden_published"`
	Exceptions      int     `json:"exceptions"`
	ScoredVendors   int     `json:"scored_vendors"`
	StarRocksSynced bool    `json:"starrocks_synced"`
	DurationSeconds float64 `json:"duration_seconds"`
}

// MDMIngestAndScoringWorkflow coordinates:
// 1. Data Pipeline Ingest (File -> Iceberg Parquet Lakehouse + Centralized Validation Rules + Staging Table)
// 2. MDM Entity Mastering (Match -> Survivorship -> Publish Golden Records)
// 3. Vendor Quality & Substitution Scoring (Tolerances -> Sufficiency Matrix -> Pareto Frontier)
// 4. Hot StarRocks Analytical Mart Synchronization
func MDMIngestAndScoringWorkflow(ctx workflow.Context, input MDMIngestScoringInput) (*MDMIngestScoringResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("MDMIngestAndScoringWorkflow initiated",
		"tenantID", input.TenantID,
		"entity", input.Entity,
		"pipelineRunID", input.PipelineRunID,
		"stagingTable", input.StagingTable,
	)

	start := workflow.Now(ctx)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second * 2,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Step 1: Execute Data Pipeline (File -> Iceberg + RuleCheck + Staging DB)
	var pipelineOutput struct {
		RecordsStaged int    `json:"records_staged"`
		LoadRunID     string `json:"load_run_id"`
		Status        string `json:"status"`
	}
	err := workflow.ExecuteActivity(ctx, "ExecuteMDMPipelineActivity", input).Get(ctx, &pipelineOutput)
	if err != nil {
		logger.Error("Step 1 failed: Data Pipeline ingestion", "error", err)
		return nil, fmt.Errorf("data pipeline ingestion failed: %w", err)
	}
	logger.Info("Step 1 completed: Data Pipeline ingestion", "loadRunID", pipelineOutput.LoadRunID, "recordsStaged", pipelineOutput.RecordsStaged)

	// Step 2: Execute MDM Mastering (Canonicalize -> Match -> Survive -> Publish)
	var masterOutput struct {
		RunID         string `json:"run_id"`
		Published     int    `json:"published"`
		HeldForReview int    `json:"held_for_review"`
		Exceptions    int    `json:"exceptions"`
		Status        string `json:"status"`
	}
	masterReq := map[string]any{
		"tenant_id":       input.TenantID,
		"entity":          input.Entity,
		"staging_table":   input.StagingTable,
		"load_run_id":     pipelineOutput.LoadRunID,
		"pipeline_run_id": input.PipelineRunID,
	}
	err = workflow.ExecuteActivity(ctx, "ExecuteMDMMasteringActivity", masterReq).Get(ctx, &masterOutput)
	if err != nil {
		logger.Error("Step 2 failed: MDM Mastering", "error", err)
		return nil, fmt.Errorf("mdm mastering failed: %w", err)
	}
	logger.Info("Step 2 completed: MDM Mastering", "masterRunID", masterOutput.RunID, "published", masterOutput.Published)

	// Step 3: Execute Vendor Quality & Displacement Scoring
	var scoringOutput struct {
		ScoredVendors int     `json:"scored_vendors"`
		T1Sufficiency float64 `json:"t1_sufficiency"`
		Status        string  `json:"status"`
	}
	scoringReq := map[string]any{
		"tenant_id": input.TenantID,
		"entity":    input.Entity,
		"as_of":     input.AsOfDate,
	}
	err = workflow.ExecuteActivity(ctx, "ExecuteVendorScoringActivity", scoringReq).Get(ctx, &scoringOutput)
	if err != nil {
		logger.Error("Step 3 failed: Vendor Quality Scoring", "error", err)
		return nil, fmt.Errorf("vendor quality scoring failed: %w", err)
	}
	logger.Info("Step 3 completed: Vendor Quality Scoring", "scoredVendors", scoringOutput.ScoredVendors)

	// Step 4: Optional Hot StarRocks Analytical Mart Sync
	starrocksSynced := false
	if input.SyncStarRocks {
		var srOutput string
		err = workflow.ExecuteActivity(ctx, "ExecuteStarRocksMartSyncActivity", input).Get(ctx, &srOutput)
		if err != nil {
			logger.Warn("Step 4 warning: StarRocks Mart sync encountered error (non-fatal)", "error", err)
		} else {
			starrocksSynced = true
			logger.Info("Step 4 completed: StarRocks Hot Mart synced")
		}
	}

	duration := workflow.Now(ctx).Sub(start).Seconds()

	result := &MDMIngestScoringResult{
		PipelineRunID:   input.PipelineRunID,
		MasteringRunID:  masterOutput.RunID,
		GoldenPublished: masterOutput.Published,
		Exceptions:      masterOutput.Exceptions,
		ScoredVendors:   scoringOutput.ScoredVendors,
		StarRocksSynced: starrocksSynced,
		DurationSeconds: duration,
	}

	logger.Info("MDMIngestAndScoringWorkflow completed successfully",
		"durationSeconds", duration,
		"published", result.GoldenPublished,
	)

	return result, nil
}
