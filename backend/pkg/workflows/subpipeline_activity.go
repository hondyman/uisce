package workflows

import (
	"encoding/json"
	"fmt"
	"strings"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/workflow"
)

// ============================================================================
// Sub-Pipeline (Composable Workflows) - Execute child pipelines
// ============================================================================

// SubPipelineConfig defines the configuration for a subPipeline node
// This follows the Composable Workflows pattern for building reusable business processes
type SubPipelineConfig struct {
	PipelineID     string            `json:"pipeline_id"`     // ID of sub-pipeline to execute
	InputMapping   map[string]string `json:"input_mapping"`   // Map of child input key -> JSONPath from parent context
	OutputVariable string            `json:"output_variable"` // Variable name to store result in parent context
}

// SubPipelineResult is the outcome of executing a sub-pipeline
type SubPipelineResult struct {
	SubPipelineID string                 `json:"sub_pipeline_id"`
	WorkflowRunID string                 `json:"workflow_run_id"`
	Status        string                 `json:"status"`
	Output        map[string]interface{} `json:"output"`
}

// ============================================================================
// Execute Sub-Pipeline Node (called from InterpreterWorkflow)
// ============================================================================

// ExecuteSubPipelineNode is called from within the InterpreterWorkflow
// to execute a sub-pipeline using Temporal's Child Workflow feature.
//
// This creates a parent-child relationship where:
// - The child workflow is automatically retried on failure
// - The child is terminated if the parent is cancelled
// - Both workflows share the same Temporal namespace
func ExecuteSubPipelineNode(
	ctx workflow.Context,
	config SubPipelineConfig,
	parentContext map[string]interface{},
) (*SubPipelineResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Executing sub-pipeline",
		"pipelineId", config.PipelineID,
		"outputVariable", config.OutputVariable)

	// 1. Resolve input data using the input_mapping from the parent context
	subPipelineInput := resolveInputMapping(config.InputMapping, parentContext)

	// 2. Configure child workflow options
	parentInfo := workflow.GetInfo(ctx)
	childWorkflowID := fmt.Sprintf("%s-%s", config.PipelineID, parentInfo.WorkflowExecution.RunID)

	cwo := workflow.ChildWorkflowOptions{
		WorkflowID: childWorkflowID,
		// CRITICAL: Terminate child if parent is cancelled/terminated
		// This ensures proper cleanup and prevents orphaned workflows
		ParentClosePolicy: enums.PARENT_CLOSE_POLICY_TERMINATE,
	}
	ctx = workflow.WithChildOptions(ctx, cwo)

	// 3. Execute the generic InterpreterWorkflow as a child
	// ALL pipelines (parent or child) use the same InterpreterWorkflow
	var childResult WorkflowResult
	childWorkflowFuture := workflow.ExecuteChildWorkflow(ctx, "InterpreterWorkflow",
		InterpreterInput{
			WorkflowID:  config.PipelineID,
			InitialData: subPipelineInput,
		})

	// 4. Wait for the child workflow to complete and get its result
	err := childWorkflowFuture.Get(ctx, &childResult)
	if err != nil {
		logger.Error("Sub-pipeline execution failed",
			"PipelineID", config.PipelineID,
			"Error", err)
		return nil, fmt.Errorf("sub-pipeline '%s' failed: %w", config.PipelineID, err)
	}

	logger.Info("Sub-pipeline execution completed successfully",
		"PipelineID", config.PipelineID,
		"Status", childResult.Status)

	return &SubPipelineResult{
		SubPipelineID: config.PipelineID,
		WorkflowRunID: childWorkflowID,
		Status:        childResult.Status,
		Output:        childResult.FinalState,
	}, nil
}

// resolveInputMapping transforms parent context data into sub-pipeline input
// using the input_mapping configuration
func resolveInputMapping(mapping map[string]string, parentContext map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for targetKey, sourcePath := range mapping {
		value, err := resolveDataPath(sourcePath, parentContext)
		if err != nil {
			// Log but continue - allow partial mapping
			continue
		}
		result[targetKey] = value
	}

	return result
}

// resolveDataPath extracts a value from the parent context using dot notation
// Supports paths like: $.input.customer_id, $.nodes.get_data.output.value
func resolveDataPath(path string, context map[string]interface{}) (interface{}, error) {
	if path == "" {
		return nil, fmt.Errorf("empty path")
	}

	// Handle special values
	if path == "$.now" {
		return "{{NOW}}", nil // Placeholder, resolved at runtime
	}

	// Remove leading $. if present
	path = strings.TrimPrefix(path, "$.")

	// Split by dots and traverse
	parts := strings.Split(path, ".")
	current := interface{}(context)

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

// ============================================================================
// Config Parsing
// ============================================================================

// ParseSubPipelineConfig extracts SubPipelineConfig from node config map
func ParseSubPipelineConfig(config map[string]interface{}) (*SubPipelineConfig, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var cfg SubPipelineConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse sub-pipeline config: %w", err)
	}

	if cfg.PipelineID == "" {
		return nil, fmt.Errorf("pipeline_id is required for subPipeline node")
	}

	return &cfg, nil
}

// ============================================================================
// Backward Compatibility: Support both CHILD_PIPELINE and subPipeline
// ============================================================================

// IsSubPipelineNode checks if a node type is a sub-pipeline call
func IsSubPipelineNode(nodeType string) bool {
	return nodeType == "subPipeline" || nodeType == "CHILD_PIPELINE" || nodeType == "child_pipeline"
}
