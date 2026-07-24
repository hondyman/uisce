package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/workflow"
)

// ==================================================================================
// GLOBAL MATERIALIZATION WORKFLOW (Fan-Out Pattern)
// ==================================================================================

// GlobalMaterializationInput defines input for global materialization
type GlobalMaterializationInput struct {
	FeatureID       string
	FeatureName     string
	StartTime       time.Time
	EndTime         time.Time
	Priority        string // low, normal, high
	NotifyOnSuccess bool
}

// GlobalMaterializationOutput defines output from global materialization
type GlobalMaterializationOutput struct {
	FeatureID       string
	TotalRegions    int
	SuccessRegions  int
	FailedRegions   int
	PartialFailure  bool
	ExecutionID     string
	StartTime       time.Time
	CompletionTime  time.Time
	TotalDurationMs int64
	RegionResults   map[string]RegionMaterializationResult
}

// RegionMaterializationResult tracks per-region materialization result
type RegionMaterializationResult struct {
	RegionCode   string
	Status       string // success, failed, timeout
	StartTime    time.Time
	DurationMs   int64
	RowCount     int64
	ErrorMessage string
}

// GlobalMaterializationWorkflow orchestrates feature materialization across all regions
func GlobalMaterializationWorkflow(ctx workflow.Context, input GlobalMaterializationInput) (*GlobalMaterializationOutput, error) {
	logger := workflow.GetLogger(ctx)
	executionID := workflow.GetInfo(ctx).WorkflowExecution.ID

	logger.Info("Starting global materialization", "feature_id", input.FeatureID, "feature_name", input.FeatureName)

	// 1. Load list of active regions
	var activeRegions []string
	aerr := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: 2 * time.Minute,
		}),
		GetActiveRegionsActivity,
	).Get(ctx, &activeRegions)

	if aerr != nil {
		logger.Error("Failed to load active regions", "error", aerr)
		return nil, aerr
	}

	logger.Info("Loaded active regions", "count", len(activeRegions), "regions", activeRegions)

	// 2. Record global workflow execution
	werr := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: 1 * time.Minute,
		}),
		RecordGlobalWorkflowActivity,
		executionID,
		"GlobalMaterialization",
		input.FeatureID,
		activeRegions,
		"running",
	).Get(ctx, nil)

	if werr != nil {
		logger.Warn("Failed to record workflow execution", "error", werr)
	}

	// 3. Fan out to region workflows
	regionMaterializationFutures := make(map[string]workflow.Future)

	for _, region := range activeRegions {
		regionInput := RegionMaterializationInput{
			FeatureID:   input.FeatureID,
			FeatureName: input.FeatureName,
			Region:      region,
			StartTime:   input.StartTime,
			EndTime:     input.EndTime,
			Priority:    input.Priority,
		}

		// Execute child workflow for each region
		childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			TaskQueue: "materialization_" + region,
		})

		future := workflow.ExecuteChildWorkflow(childCtx, RegionMaterializationWorkflow, regionInput)
		regionMaterializationFutures[region] = future
	}

	logger.Info("Fanned out to region workflows", "count", len(regionMaterializationFutures))

	// 4. Wait for all region workflows to complete
	output := &GlobalMaterializationOutput{
		FeatureID:     input.FeatureID,
		TotalRegions:  len(activeRegions),
		ExecutionID:   executionID,
		StartTime:     time.Now(),
		RegionResults: make(map[string]RegionMaterializationResult),
	}

	successCount := 0
	failCount := 0

	for region, future := range regionMaterializationFutures {
		var regionOutput RegionMaterializationResult
		if rerr := future.Get(ctx, &regionOutput); rerr != nil {
			logger.Error("Region materialization failed", "region", region, "error", rerr)
			regionOutput.Status = "failed"
			regionOutput.ErrorMessage = rerr.Error()
			failCount++
		} else {
			if regionOutput.Status == "success" {
				successCount++
			} else {
				failCount++
			}
		}

		output.RegionResults[region] = regionOutput
	}

	output.SuccessRegions = successCount
	output.FailedRegions = failCount
	output.CompletionTime = time.Now()
	output.TotalDurationMs = output.CompletionTime.Sub(output.StartTime).Milliseconds()

	// 5. Determine final status
	finalStatus := "success"
	if failCount > 0 {
		output.PartialFailure = true
		finalStatus = "partial_failure"
		if successCount == 0 {
			finalStatus = "failed"
		}
	}

	// 6. Update global workflow execution record
	uerr := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: 1 * time.Minute,
		}),
		UpdateGlobalWorkflowActivity,
		executionID,
		finalStatus,
		successCount,
		failCount,
		"",
	).Get(ctx, nil)

	if uerr != nil {
		logger.Warn("Failed to update workflow execution", "error", uerr)
	}

	// 7. Send completion notification if requested
	if input.NotifyOnSuccess && finalStatus == "success" {
		workflow.ExecuteActivity(
			workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
				StartToCloseTimeout: 1 * time.Minute,
			}),
			SendNotificationActivity,
			fmt.Sprintf("Feature %s materialization completed successfully across %d regions", input.FeatureName, successCount),
		)
	}

	logger.Info("Global materialization completed", "final_status", finalStatus, "success_count", successCount, "failed_count", failCount)

	return output, nil
}

// ==================================================================================
// REGION MATERIALIZATION WORKFLOW
// ==================================================================================

type RegionMaterializationInput struct {
	FeatureID   string
	FeatureName string
	Region      string
	StartTime   time.Time
	EndTime     time.Time
	Priority    string
}

// RegionMaterializationWorkflow orchestrates materialization within a single region
func RegionMaterializationWorkflow(ctx workflow.Context, input RegionMaterializationInput) (*RegionMaterializationResult, error) {
	logger := workflow.GetLogger(ctx)

	logger.Info("Starting region materialization", "feature_id", input.FeatureID, "region", input.Region)

	result := &RegionMaterializationResult{
		RegionCode: input.Region,
		StartTime:  time.Now(),
	}

	// Execute materialization activity for this region
	// Build region-specific activity options
	regionActivityOptions := workflow.ActivityOptions{
		TaskQueue:           "activity_" + input.Region,
		StartToCloseTimeout: 30 * time.Minute,
	}
	aerr := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, regionActivityOptions),
		MaterializeFeatureActivity,
		input,
	).Get(ctx, result)

	if aerr != nil {
		logger.Error("Materialization activity failed", "region", input.Region, "error", aerr)
		result.Status = "failed"
		result.ErrorMessage = aerr.Error()
		return result, aerr
	}

	result.DurationMs = time.Since(result.StartTime).Milliseconds()

	// Record region execution
	workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: 1 * time.Minute,
		}),
		RecordRegionWorkflowActivity,
		workflow.GetInfo(ctx).WorkflowExecution.ID,
		input.Region,
		result.Status,
		result.DurationMs,
		result.ErrorMessage,
	)

	logger.Info("Region materialization completed", "region", input.Region, "status", result.Status, "duration_ms", result.DurationMs)

	return result, nil
}

// ==================================================================================
// GLOBAL DRIFT DETECTION WORKFLOW
// ==================================================================================

type GlobalDriftDetectionInput struct {
	FeatureIDs   []string
	BaselineWin  string   // e.g., "7d"
	EvalWindow   string   // e.g., "1d"
	Methods      []string // ks_test, js_distance, wasserstein
	Threshold    float64  // 0.05 for p-value
	AlertOnDrift bool
}

type GlobalDriftDetectionOutput struct {
	TotalFeatures int
	DriftedCount  int
	RegionResults map[string]int // region -> count of drifted features
	ExecutionTime time.Time
}

// GlobalDriftDetectionWorkflow detects drift across all regions
func GlobalDriftDetectionWorkflow(ctx workflow.Context, input GlobalDriftDetectionInput) (*GlobalDriftDetectionOutput, error) {
	logger := workflow.GetLogger(ctx)

	// Load active regions
	var activeRegions []string
	if err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: 2 * time.Minute,
		}),
		GetActiveRegionsActivity,
	).Get(ctx, &activeRegions); err != nil {
		logger.Error("Failed to load active regions", "error", err)
		return nil, err
	}

	// Fan out detect drift to each region
	output := &GlobalDriftDetectionOutput{
		TotalFeatures: len(input.FeatureIDs),
		RegionResults: make(map[string]int),
		ExecutionTime: time.Now(),
	}

	regionFutures := make(map[string]workflow.Future)

	for _, region := range activeRegions {
		regionInput := RegionDriftDetectionInput{
			FeatureIDs:  input.FeatureIDs,
			Region:      region,
			BaselineWin: input.BaselineWin,
			EvalWindow:  input.EvalWindow,
			Methods:     input.Methods,
			Threshold:   input.Threshold,
		}

		childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			TaskQueue: "drift_" + region,
		})

		regionFutures[region] = workflow.ExecuteChildWorkflow(childCtx, RegionDriftDetectionWorkflow, regionInput)
	}

	// Wait for all regions
	totalDrifted := 0
	for region, future := range regionFutures {
		var regionOutput RegionDriftDetectionOutput
		if err := future.Get(ctx, &regionOutput); err != nil {
			logger.Warn("Region drift detection failed", "region", region, "error", err)
			output.RegionResults[region] = 0
		} else {
			output.RegionResults[region] = regionOutput.DriftedCount
			totalDrifted += regionOutput.DriftedCount
		}
	}

	output.DriftedCount = totalDrifted

	// Send alert if threshold breached
	if input.AlertOnDrift && totalDrifted > 0 {
		workflow.ExecuteActivity(
			workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
				StartToCloseTimeout: 1 * time.Minute,
			}),
			SendAlertActivity,
			fmt.Sprintf("Drift detected in %d features across %.0f%% of regions", totalDrifted,
				float64(len(output.RegionResults))/float64(len(activeRegions))*100.0),
		)
	}

	logger.Info("Global drift detection completed", "total_drifted", totalDrifted, "regions", len(activeRegions))

	return output, nil
}

// ==================================================================================
// REGION DRIFT DETECTION WORKFLOW
// ==================================================================================

type RegionDriftDetectionInput struct {
	FeatureIDs  []string
	Region      string
	BaselineWin string
	EvalWindow  string
	Methods     []string
	Threshold   float64
}

type RegionDriftDetectionOutput struct {
	Region        string
	TotalFeatures int
	DriftedCount  int
}

// RegionDriftDetectionWorkflow detects drift within a single region
func RegionDriftDetectionWorkflow(ctx workflow.Context, input RegionDriftDetectionInput) (*RegionDriftDetectionOutput, error) {
	logger := workflow.GetLogger(ctx)

	output := &RegionDriftDetectionOutput{
		Region:        input.Region,
		TotalFeatures: len(input.FeatureIDs),
	}

	// Execute drift detection activity
	// Build region-specific activity options
	regionActivityOptions := workflow.ActivityOptions{
		TaskQueue:           "activity_" + input.Region,
		StartToCloseTimeout: 30 * time.Minute,
	}
	if err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, regionActivityOptions),
		DetectDriftActivity,
		input,
	).Get(ctx, &output.DriftedCount); err != nil {
		logger.Error("Drift detection activity failed", "region", input.Region, "error", err)
		output.DriftedCount = 0
		return output, err
	}

	logger.Info("Region drift detection completed", "region", input.Region, "drifted_count", output.DriftedCount)

	return output, nil
}

// ==================================================================================
// GLOBAL DISCOVERY WORKFLOW
// ==================================================================================

type GlobalDiscoveryInput struct {
	ScanInterval  int    // hours
	UseCase       string // forecasting, classification, etc.
	AllowMutation bool
}

type GlobalDiscoveryOutput struct {
	TotalCandidates    int
	ApprovedCandidates int
	ExecutionDuration  time.Duration
	RegionResults      map[string]int
}

// GlobalDiscoveryWorkflow orchestrates feature discovery across all regions
func GlobalDiscoveryWorkflow(ctx workflow.Context, input GlobalDiscoveryInput) (*GlobalDiscoveryOutput, error) {
	logger := workflow.GetLogger(ctx)

	startTime := time.Now()

	// Load active regions
	var activeRegions []string
	if err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: 2 * time.Minute,
		}),
		GetActiveRegionsActivity,
	).Get(ctx, &activeRegions); err != nil {
		logger.Error("Failed to load active regions", "error", err)
		return nil, err
	}

	output := &GlobalDiscoveryOutput{
		RegionResults: make(map[string]int),
	}

	// Fan out discovery to each region
	regionFutures := make(map[string]workflow.Future)

	for _, region := range activeRegions {
		discoveryInput := RegionDiscoveryInput{
			Region:        region,
			ScanInterval:  input.ScanInterval,
			UseCase:       input.UseCase,
			AllowMutation: input.AllowMutation,
		}

		childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			TaskQueue: "discovery_" + region,
		})

		regionFutures[region] = workflow.ExecuteChildWorkflow(childCtx, RegionDiscoveryWorkflow, discoveryInput)
	}

	// Collect results
	totalCandidates := 0
	for region, future := range regionFutures {
		var count int
		if err := future.Get(ctx, &count); err != nil {
			logger.Warn("Region discovery failed", "region", region, "error", err)
			output.RegionResults[region] = 0
		} else {
			output.RegionResults[region] = count
			totalCandidates += count
		}
	}

	output.TotalCandidates = totalCandidates
	output.ExecutionDuration = time.Since(startTime)

	logger.Info("Global discovery completed", "total_candidates", totalCandidates, "duration", output.ExecutionDuration)

	return output, nil
}

// ==================================================================================
// REGION DISCOVERY WORKFLOW
// ==================================================================================

type RegionDiscoveryInput struct {
	Region        string
	ScanInterval  int
	UseCase       string
	AllowMutation bool
}

// RegionDiscoveryWorkflow executes discovery in a single region
func RegionDiscoveryWorkflow(ctx workflow.Context, input RegionDiscoveryInput) (int, error) {
	logger := workflow.GetLogger(ctx)

	var candidateCount int

	// Build region-specific activity options
	regionActivityOptions := workflow.ActivityOptions{
		TaskQueue:           "activity_" + input.Region,
		StartToCloseTimeout: 30 * time.Minute,
	}
	if err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, regionActivityOptions),
		DiscoverFeaturesActivity,
		input,
	).Get(ctx, &candidateCount); err != nil {
		logger.Error("Discovery activity failed", "region", input.Region, "error", err)
		return 0, err
	}

	logger.Info("Region discovery completed", "region", input.Region, "candidates", candidateCount)

	return candidateCount, nil
}

// ==================================================================================
// ACTIVITY DEFINITIONS (Stubs - implement in activities package)
// ==================================================================================

var (
	GetActiveRegionsActivity     = "GetActiveRegions"
	RecordGlobalWorkflowActivity = "RecordGlobalWorkflow"
	UpdateGlobalWorkflowActivity = "UpdateGlobalWorkflow"
	RecordRegionWorkflowActivity = "RecordRegionWorkflow"
	MaterializeFeatureActivity   = "MaterializeFeature"
	DetectDriftActivity          = "DetectDrift"
	DiscoverFeaturesActivity     = "DiscoverFeatures"
	SendNotificationActivity     = "SendNotification"
	SendAlertActivity            = "SendAlert"
)
