package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
)

// ============================================================================
// Child Pipeline Activity - Executes a Sub-Pipeline
// ============================================================================

// ChildPipelineConfig defines the configuration for a CHILD_PIPELINE node
type ChildPipelineConfig struct {
	PipelineID   string            `json:"pipeline_id"`   // ID of child pipeline to execute
	InputMapping map[string]string `json:"input_mapping"` // Map of child input key -> JSONPath from parent state
	ResultPath   string            `json:"result_path"`   // Where to store result in parent state
}

// ChildPipelineInput is passed to the activity
type ChildPipelineInput struct {
	ParentPipelineID string                 `json:"parent_pipeline_id"`
	ParentRunID      string                 `json:"parent_run_id"`
	Config           ChildPipelineConfig    `json:"config"`
	ParentState      map[string]interface{} `json:"parent_state"`
}

// ChildPipelineResult is returned from the activity
type ChildPipelineResult struct {
	ChildRunID string                 `json:"child_run_id"`
	Status     string                 `json:"status"`
	Result     map[string]interface{} `json:"result"`
}

// ChildPipelineActivities holds dependencies for child pipeline execution
type ChildPipelineActivities struct {
	// In production, this would hold a reference to pipeline service
	// for loading pipeline definitions from the database
}

// NewChildPipelineActivities creates a new instance
func NewChildPipelineActivities() *ChildPipelineActivities {
	return &ChildPipelineActivities{}
}

// ActivityPrepareChildInput resolves the input mapping from parent state
// This is a lightweight activity that can be used inline
func (a *ChildPipelineActivities) ActivityPrepareChildInput(
	ctx context.Context,
	mapping map[string]string,
	parentState map[string]interface{},
) (map[string]interface{}, error) {
	activity.RecordHeartbeat(ctx, "Preparing child pipeline input...")

	childInput := make(map[string]interface{})

	for targetKey, sourcePath := range mapping {
		value, err := resolveJSONPath(sourcePath, parentState)
		if err != nil {
			// Log warning but continue - allow partial input
			activity.GetLogger(ctx).Warn("Failed to resolve input mapping",
				"targetKey", targetKey,
				"sourcePath", sourcePath,
				"error", err)
			continue
		}
		childInput[targetKey] = value
	}

	return childInput, nil
}

// resolveJSONPath extracts a value from state using a simple JSONPath expression
// Supports: $.field, $.nested.field, $.input.field
func resolveJSONPath(path string, state map[string]interface{}) (interface{}, error) {
	if path == "" {
		return nil, fmt.Errorf("empty path")
	}

	// Handle special values
	if path == "$.now" {
		return "{{NOW}}", nil // Placeholder for workflow time
	}

	// Remove leading $. if present
	path = strings.TrimPrefix(path, "$.")

	// Split by dots and traverse
	parts := strings.Split(path, ".")
	current := interface{}(state)

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			val, exists := v[part]
			if !exists {
				return nil, fmt.Errorf("key not found: %s", part)
			}
			current = val
		default:
			return nil, fmt.Errorf("cannot traverse into non-object at: %s", part)
		}
	}

	return current, nil
}

// setInStateByPath places a value in state at the specified path
// Supports: $.results.field, $.output.child_result
func setInStateByPath(state map[string]interface{}, path string, value interface{}) error {
	if path == "" {
		return fmt.Errorf("empty result path")
	}

	// Remove leading $. if present
	path = strings.TrimPrefix(path, "$.")

	// Split by dots
	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return fmt.Errorf("invalid path")
	}

	// Navigate/create nested structure
	current := state
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		if next, exists := current[part]; exists {
			if nextMap, ok := next.(map[string]interface{}); ok {
				current = nextMap
			} else {
				// Overwrite non-object
				newMap := make(map[string]interface{})
				current[part] = newMap
				current = newMap
			}
		} else {
			// Create new nested object
			newMap := make(map[string]interface{})
			current[part] = newMap
			current = newMap
		}
	}

	// Set the final value
	current[parts[len(parts)-1]] = value
	return nil
}

// ============================================================================
// Execute Child Pipeline as Workflow (called from Interpreter)
// ============================================================================

// ExecuteChildPipeline is designed to be called from within a workflow context
// It uses Temporal's ExecuteChildWorkflow for durable execution
func ExecuteChildPipeline(
	ctx workflow.Context,
	config ChildPipelineConfig,
	parentState map[string]interface{},
) (*ChildPipelineResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Executing child pipeline", "pipelineId", config.PipelineID)

	// 1. Prepare child input from mapping
	childInput := make(map[string]interface{})
	for targetKey, sourcePath := range config.InputMapping {
		value, err := resolveJSONPath(sourcePath, parentState)
		if err != nil {
			logger.Warn("Failed to resolve input mapping",
				"targetKey", targetKey,
				"sourcePath", sourcePath)
			continue
		}
		childInput[targetKey] = value
	}

	// 2. Configure child workflow options
	childOptions := workflow.ChildWorkflowOptions{
		WorkflowID: fmt.Sprintf("child_%s_%s", config.PipelineID, workflow.GetInfo(ctx).WorkflowExecution.ID),
		// IMPORTANT: Terminate child if parent is terminated
		ParentClosePolicy: enums.PARENT_CLOSE_POLICY_TERMINATE,
	}
	childCtx := workflow.WithChildOptions(ctx, childOptions)

	// 3. Execute child workflow (RunStoredWorkflow with the child pipeline ID)
	var childResult WorkflowResult
	childFuture := workflow.ExecuteChildWorkflow(childCtx, RunStoredWorkflow, InterpreterInput{
		WorkflowID:  config.PipelineID,
		InitialData: childInput,
	})

	// 4. Wait for child completion
	err := childFuture.Get(childCtx, &childResult)
	if err != nil {
		logger.Error("Child pipeline failed", "pipelineId", config.PipelineID, "error", err)
		return nil, fmt.Errorf("child pipeline '%s' failed: %w", config.PipelineID, err)
	}

	logger.Info("Child pipeline completed",
		"pipelineId", config.PipelineID,
		"status", childResult.Status)

	return &ChildPipelineResult{
		ChildRunID: childOptions.WorkflowID,
		Status:     childResult.Status,
		Result:     childResult.FinalState,
	}, nil
}

// ============================================================================
// Utility: Serialize config from node
// ============================================================================

// ParseChildPipelineConfig extracts ChildPipelineConfig from node config map
func ParseChildPipelineConfig(config map[string]interface{}) (*ChildPipelineConfig, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var cfg ChildPipelineConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse child pipeline config: %w", err)
	}

	return &cfg, nil
}
