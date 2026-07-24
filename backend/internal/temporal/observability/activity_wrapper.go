package observability

import (
	"context"
	"fmt"

	"go.temporal.io/sdk/workflow"
)

// TracedActivity wraps a Temporal activity with OpenTelemetry tracing.
// This creates a span that shows up in Jaeger with the BUSINESS NAME of the step,
// not just "Activity" or "ExecuteLocalActivity".
//
// Usage in workflow:
//
//	result, err := observability.TracedActivity(ctx, "ExtractTrainingData", func(ctx context.Context) (interface{}, error) {
//	    return acts.ExtractTrainingDataActivity(ctx, params)
//	})
func TracedActivity(
	ctx workflow.Context,
	stepName string,
	fn func(context.Context) (interface{}, error),
) (interface{}, error) {
	// Use LocalActivity to avoid non-determinism issues with OTel SDK
	ao := workflow.LocalActivityOptions{
		StartToCloseTimeout: workflow.GetInfo(ctx).WorkflowExecutionTimeout,
	}
	ctx = workflow.WithLocalActivityOptions(ctx, ao)

	var result interface{}
	err := workflow.ExecuteLocalActivity(ctx, func(ctx context.Context) (interface{}, error) {
		return runWithTrace(ctx, stepName, fn)
	}).Get(ctx, &result)

	return result, err
}

// runWithTrace executes the function inside an OTel span.
// This is the "interpreter span injection" pattern from the document.
func runWithTrace(ctx context.Context, stepName string, fn func(context.Context) (interface{}, error)) (interface{}, error) {
	tracer := GetTracer("dsl-interpreter")

	// START CUSTOM SPAN
	// This makes the trace readable: "DSL Step: ExtractTrainingData"
	// instead of generic "Activity"
	ctx, span := tracer.Start(ctx, fmt.Sprintf("DSL Step: %s", stepName))
	defer span.End()

	// Execute the actual business logic
	result, err := fn(ctx)

	// Record errors in the span
	if err != nil {
		span.RecordError(err)
	}

	return result, err
}

// TracedWorkflowStep is a simpler wrapper for non-activity workflow steps.
// Use this for in-workflow logic that doesn't call activities.
//
// Example:
//
//	observability.TracedWorkflowStep(ctx, "ValidateInput", func() error {
//	    // validation logic
//	    return nil
//	})
func TracedWorkflowStep(ctx workflow.Context, stepName string, fn func() error) error {
	// Note: Workflow code itself is deterministic, so we can't directly emit spans.
	// This function is a placeholder for when you want to trace synchronous workflow logic.
	// In practice, most observable work should be in activities.

	// For now, just execute the function.
	// TODO: Implement deterministic event logging to workflow history.
	return fn()
}

// TracedActivityWithMetadata is the most powerful variant.
// It accepts both the step name AND business metadata for semantic tracing.
//
// Example:
//
//	result, err := observability.TracedActivityWithMetadata(
//	    ctx,
//	    "CheckCompliance",
//	    map[string]string{
//	        "entity_id": entityID,
//	        "check_type": "kyc",
//	   asset_class": "private_equity",
//	    },
//	    func(ctx context.Context) (interface{}, error) {
//	        return performComplianceCheck(ctx, entityID)
//	    },
//	)
func TracedActivityWithMetadata(
	ctx workflow.Context,
	stepName string,
	metadata map[string]string,
	fn func(context.Context) (interface{}, error),
) (interface{}, error) {
	ao := workflow.LocalActivityOptions{
		StartToCloseTimeout: workflow.GetInfo(ctx).WorkflowExecutionTimeout,
	}
	ctx = workflow.WithLocalActivityOptions(ctx, ao)

	var result interface{}
	err := workflow.ExecuteLocalActivity(ctx, func(ctx context.Context) (interface{}, error) {
		tracer := GetTracer("dsl-interpreter")
		ctx, span := tracer.Start(ctx, fmt.Sprintf("DSL Step: %s", stepName))
		defer span.End()

		// INJECT SEMANTIC METADATA
		InjectBusinessAttributes(span, metadata)

		result, err := fn(ctx)
		if err != nil {
			span.RecordError(err)
		}
		return result, err
	}).Get(ctx, &result)

	return result, err
}
