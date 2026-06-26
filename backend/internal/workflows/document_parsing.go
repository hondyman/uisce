package workflows

import (
	"time"

	"github.com/hondyman/semlayer/backend/internal/activities"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// Signal definition
const SignalReviewComplete = "SignalReviewComplete"

// Workflow Input
type DocumentWorkflowInput struct {
	DocumentID string
	StorageURI string
	SchemaDef  string
}

// Signal Payload
type ReviewSignal struct {
	Action        string // "APPROVE" or "REJECT"
	CorrectedJSON string
}

func DocumentParsingWorkflow(ctx workflow.Context, input DocumentWorkflowInput) error {
	// Configure Activity Options
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 5, // Gemini can be slow
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Step 1: Extract Data using Gemini
	var rawJSON string
	err := workflow.ExecuteActivity(ctx, activities.GeminiExtractActivity, input).Get(ctx, &rawJSON)
	if err != nil {
		return err
	}

	// Step 2: Validate Data
	var valResult activities.ValidationResult
	err = workflow.ExecuteActivity(ctx, activities.ValidateSchemaActivity, rawJSON, input.SchemaDef).Get(ctx, &valResult)
	if err != nil {
		return err
	}

	// Step 3: Check Validity
	if valResult.IsValid {
		// Happy Path: Persist
		return workflow.ExecuteActivity(ctx, activities.PersistDataActivity, valResult.ResultJSON).Get(ctx, nil)
	}

	// Step 4: Human-in-the-Loop (HITL) Path
	// Trigger notification to human analysts
	err = workflow.ExecuteActivity(ctx, activities.NotifyHumanReviewActivity, input.DocumentID, valResult.Errors).Get(ctx, nil)
	if err != nil {
		return err
	}

	// Initialize Selector for Signal Handling
	var signalVal ReviewSignal
	signalChan := workflow.GetSignalChannel(ctx, SignalReviewComplete)
	selector := workflow.NewSelector(ctx)

	// Register the signal receiver
	selector.AddReceive(signalChan, func(c workflow.ReceiveChannel, more bool) {
		c.Receive(ctx, &signalVal)
	})

	// Register a timeout (e.g., 7 days for human review)
	// If the human doesn't respond, we fail the workflow or auto-reject
	timerFuture := workflow.NewTimer(ctx, time.Hour*24*7)
	selector.AddFuture(timerFuture, func(f workflow.Future) {
		signalVal.Action = "TIMEOUT"
	})

	// Wait for Signal or Timeout
	// This blocks the workflow routine but does not consume resources on the worker
	selector.Select(ctx)

	// Process the Human Decision
	switch signalVal.Action {
	case "APPROVE":
		// User provided corrected JSON, persist it
		return workflow.ExecuteActivity(ctx, activities.PersistDataActivity, signalVal.CorrectedJSON).Get(ctx, nil)
	case "REJECT":
		// Mark document as rejected in system
		return workflow.ExecuteActivity(ctx, activities.MarkDocumentRejectedActivity, input.DocumentID).Get(ctx, nil)
	case "TIMEOUT":
		return temporal.NewApplicationError("Human review timed out", "REVIEW_TIMEOUT", nil)
	default:
		return temporal.NewApplicationError("Unknown review action", "UNKNOWN_ACTION", nil)
	}
}
