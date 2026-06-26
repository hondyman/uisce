package simulation

import (
	"context"
	"fmt"
	"time"

	"github.com/hondyman/semlayer/backend/pkg/workflows"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

type SimulationResult struct {
	Success      bool                   `json:"success"`
	Events       []string               `json:"events"`
	Output       map[string]interface{} `json:"output,omitempty"`
	ErrorMessage string                 `json:"errorMessage,omitempty"`
}

type SimulationRunner struct {
	suite testsuite.WorkflowTestSuite
}

func NewSimulationRunner() *SimulationRunner {
	return &SimulationRunner{
		suite: testsuite.WorkflowTestSuite{},
	}
}

// RunSimulation executes the pipeline in a virtual time environment with mocked activities
func (r *SimulationRunner) RunSimulation(ctx context.Context, dsl workflows.WorkflowDefinition) (*SimulationResult, error) {
	env := r.suite.NewTestWorkflowEnvironment()

	// Register the workflow
	env.RegisterWorkflow(workflows.InterpreterWorkflow)

	// Register Mocks for all known activities
	// We iterate over the Central Registry (conceptually - for now simpler approach)
	// In a real implementation we would iterate GlobalRegistry.
	// We'll define a generic mock handler.

	// Create a mock object - (Not strictly needed if we use generic closures, but good for future ext)

	// We need to register "generic" mocks for any activity the pipeline might call.
	// Since we can't easily iterate all potential string names in the test env registration
	// without the registry exposing keys, we assume the Registry has a Keys() method or similar.
	// For this MVP, we will rely on the fact that the Interpreter passes the string name.
	// But TestEnv requires explicit registration of what it expects.

	// Strategy: Only mock the activities present in the DSL nodes!
	mockedNames := make(map[string]bool)
	for _, node := range dsl.Nodes {
		if node.Type == "ACTIVITY" {
			name, _ := node.Config["activityName"].(string)
			if name != "" && !mockedNames[name] {
				// Register a mock for this activity name
				// note: function signature must match what the workflow expects.
				// The interpreter expects: func(ctx context.Context, input map[string]interface{}, state map[string]interface{}) (map[string]interface{}, error)

				// We create a dynamically generated function or just use a generic one and register it UNDER THAT NAME.
				env.RegisterActivityWithOptions(
					func(ctx context.Context, input map[string]interface{}, state map[string]interface{}) (map[string]interface{}, error) {
						// Record event
						// In a real sim we'd capture this in a list. TestEnv has GetHistory?
						return map[string]interface{}{
							"simulated_output": fmt.Sprintf("Mock result from %s", name),
							"timestamp":        time.Now().String(),
						}, nil
					},
					activity.RegisterOptions{Name: name},
				)
				mockedNames[name] = true
			}
		}
	}

	// Also register Ledger activities if used
	env.RegisterActivityWithOptions(
		func(ctx context.Context, record interface{}) (string, error) {
			return "simulated-hash-chain", nil
		},
		activity.RegisterOptions{Name: "DurableLedgerWrite"},
	)

	// Execute
	env.ExecuteWorkflow(workflows.InterpreterWorkflow, dsl)

	// Collect results
	result := &SimulationResult{
		Success: env.IsWorkflowCompleted(),
		Events:  []string{}, // TODO: Parse env.GetHistory() if available or use an Interceptor
	}

	if env.IsWorkflowCompleted() {
		err := env.GetWorkflowError()
		if err != nil {
			result.Success = false
			result.ErrorMessage = err.Error()
		} else {
			var output map[string]interface{}
			_ = env.GetWorkflowResult(&output)
			result.Output = output
		}
	}

	// Capture "events" - simpler approximation for MVP:
	// We can't easily get the full event history strings from TestWorkflowEnvironment output directly in this SDK version
	// without iterating the history events.
	// For now, valid result or error is good.

	return result, nil
}
