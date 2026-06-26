package workflows

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// MigrationWorkflowInput represents the input to start a migration
type MigrationWorkflowInput struct {
	JobID          string `json:"jobId"`
	SourceCode     string `json:"sourceCode"`
	SourceLanguage string `json:"sourceLanguage"`
}

// MigrationWorkflowOutput represents the result of a migration
type MigrationWorkflowOutput struct {
	Status          string                 `json:"status"` // EXTRACTED, GENERATED, REVIEW, FAILED
	ExtractedIntent map[string]interface{} `json:"extractedIntent,omitempty"`
	GeneratedDAG    map[string]interface{} `json:"generatedDag,omitempty"`
	GeneratedRego   string                 `json:"generatedRego,omitempty"`
	Error           string                 `json:"error,omitempty"`
}

// MigrationWorkflow orchestrates the two-stage AI migration pipeline
func MigrationWorkflow(ctx workflow.Context, input MigrationWorkflowInput) (*MigrationWorkflowOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting MigrationWorkflow", "jobId", input.JobID)

	// Configure activity options with retries
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var output MigrationWorkflowOutput

	// Stage 1: Code-to-Intent (Business Logic Extraction)
	logger.Info("Stage 1: Extracting business intent from code")

	stage1Config := map[string]interface{}{
		"jobId": input.JobID,
	}
	stage1State := map[string]interface{}{
		"sourceCode": input.SourceCode,
		"language":   input.SourceLanguage,
	}

	var extractResult map[string]interface{}
	err := workflow.ExecuteActivity(ctx, "ActivityAnnotateCode", stage1Config, stage1State).Get(ctx, &extractResult)
	if err != nil {
		output.Status = "FAILED"
		output.Error = "Stage 1 failed: " + err.Error()
		return &output, nil // Return result, not error, to allow DB update
	}

	// Update status to EXTRACTED
	output.Status = "EXTRACTED"
	if intent, ok := extractResult["extracted_intent"].(map[string]interface{}); ok {
		output.ExtractedIntent = intent
	}

	// Stage 2: Intent-to-Config (RAG-powered generation)
	logger.Info("Stage 2: Generating Titan configuration")

	stage2State := map[string]interface{}{
		"extracted_intent": extractResult["extracted_intent"],
		"jobId":            input.JobID,
	}

	var generateResult map[string]interface{}
	err = workflow.ExecuteActivity(ctx, "ActivityGenerateConfig", stage1Config, stage2State).Get(ctx, &generateResult)
	if err != nil {
		output.Status = "FAILED"
		output.Error = "Stage 2 failed: " + err.Error()
		return &output, nil
	}

	// Update status to GENERATED -> REVIEW
	output.Status = "REVIEW"
	if dag, ok := generateResult["generated_dag"].(map[string]interface{}); ok {
		output.GeneratedDAG = dag
	}
	if rego, ok := generateResult["generated_rego"].(string); ok {
		output.GeneratedRego = rego
	}

	logger.Info("MigrationWorkflow completed, ready for HITL review", "jobId", input.JobID)

	return &output, nil
}
