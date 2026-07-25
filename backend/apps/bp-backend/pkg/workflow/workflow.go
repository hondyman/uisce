package workflow

import (
	"encoding/json"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// WorkflowDefinition is the struct representation of the DSL.
// This is passed as an argument from the client.
type WorkflowDefinition struct {
	RootNodeID string
	Nodes      map[string]WorkflowNode
	Edges      map[string]WorkflowEdge
}

// WorkflowNode represents a single step in the DSL.
type WorkflowNode struct {
	ID                string
	Type              string // e.g., "NODE_TYPE_ACTIVITY", "NODE_TYPE_BRANCH"
	Name              string
	ConfigJSON        string  // JSON string with step-specific config
	DefaultNextNodeID *string // For simple sequential steps
}

// WorkflowEdge represents a transition between nodes.
type WorkflowEdge struct {
	ID                  string
	SourceNodeID        string
	TargetNodeID        string
	ConditionExpression *string // e.g., "result.status == 'approved'"
}

// InterpreterWorkflow is the single, generic workflow that executes the DSL.
func InterpreterWorkflow(ctx workflow.Context, dsl WorkflowDefinition) (string, error) {
	logger := workflow.GetLogger(ctx)

	// 'executionState' holds any results from activities, to be checked by conditions.
	executionState := make(map[string]interface{})

	// Get the starting node.
	currentNode, ok := dsl.Nodes[dsl.RootNodeID]
	if !ok {
		return "", temporal.NewApplicationError("invalid DSL: root node not found", "DSL_ERROR")
	}

	// The main state machine loop.
	for {
		logger.Info("Interpreter executing node", "NodeID", currentNode.ID, "NodeType", currentNode.Type)

		switch currentNode.Type {
		case "NODE_TYPE_ACTIVITY":
			// 1. Parse the node's configuration.
			// (In a real implementation, this would be unmarshaled into a struct)
			var nodeConfig struct {
				ActivityName string        `json:"activityName"`
				TimeoutSec   int           `json:"timeout"`
				InputArgs    []interface{} `json:"inputArgs"`
			}
			// This parsing logic MUST be deterministic.
			if err := json.Unmarshal([]byte(currentNode.ConfigJSON), &nodeConfig); err != nil {
				return "", temporal.NewApplicationError("invalid node config", "DSL_ERROR", err.Error())
			}

			// 2. Dynamically build ActivityOptions from the DSL.
			// This allows users to configure retries/timeouts from the UI.
			ao := workflow.ActivityOptions{
				StartToCloseTimeout: time.Second * time.Duration(nodeConfig.TimeoutSec),
				// Other options (RetryPolicy, etc.) can also be set from DSL.
			}
			ctx = workflow.WithActivityOptions(ctx, ao)

			// 3. Execute the Activity by its string name.
			var result interface{}
			future := workflow.ExecuteActivity(ctx, nodeConfig.ActivityName, nodeConfig.InputArgs...)
			if err := future.Get(ctx, &result); err != nil {
				logger.Warn("Activity execution failed", "NodeID", currentNode.ID, "Error", err)

				// Check if the user defined a custom "on-fail" path in the DSL.
				// onFailNodeID := findOnFailNode(dsl, currentNode.ID) // Helper function

				// if onFailNodeID != nil {
				// 	// Transition to the failure-handling node.
				// 	nextNode, _ := dsl.Nodes[*onFailNodeID]
				// 	currentNode = nextNode
				// 	continue // Continue the main 'for' loop from the new node.
				// }

				// If no custom path, fail the workflow.
				return "", temporal.NewApplicationError("activity failed with no compensation path", "ACTIVITY_FAILURE", err.Error())
			}

			// 4. Store the result to be used in future branching logic.
			executionState[currentNode.ID] = result

		case "NODE_TYPE_BRANCH":
			//... (Logic to evaluate edges based on executionState)...

		case "NODE_TYPE_HITL":
			//... (See Section V.C)...

		case "NODE_TYPE_END":
			logger.Info("Interpreter reached END node.")
			return "workflow completed", nil

		default:
			return "", temporal.NewApplicationError("unknown node type", "DSL_ERROR", currentNode.Type)
		}

		// After a step, determine the next node.
		// This logic would be complex, evaluating edges and conditions.
		// For this example, we assume a simple 'DefaultNextNodeID'.
		if currentNode.DefaultNextNodeID == nil {
			return "", temporal.NewApplicationError("node has no next step", "DSL_ERROR", currentNode.ID)
		}

		nextNode, ok := dsl.Nodes[*currentNode.DefaultNextNodeID]
		if !ok {
			return "", temporal.NewApplicationError("invalid DSL: next node not found", "DSL_ERROR")
		}
		currentNode = nextNode
	}
}