package interpreter

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/workflow"
)

// WorkflowDSL represents the JSON structure of a dynamic workflow
type WorkflowDSL struct {
	ID      string              `json:"id"`
	Version string              `json:"version"`
	StartAt string              `json:"startAt"`
	States  map[string]StateDef `json:"states"`
}

// StateDef represents a single state in the state machine
type StateDef struct {
	Type     string                 `json:"type"` // task, choice, wait_signal, succeed, fail
	Activity string                 `json:"activity,omitempty"`
	Args     map[string]interface{} `json:"args,omitempty"`
	Next     string                 `json:"next,omitempty"`
	End      bool                   `json:"end,omitempty"`
	Signal   string                 `json:"signal,omitempty"`  // For wait_signal
	Choices  []Choice               `json:"choices,omitempty"` // For choice
	Default  string                 `json:"default,omitempty"` // For choice
}

type Choice struct {
	Variable string      `json:"variable"`
	Operator string      `json:"operator"` // eq, neq
	Value    interface{} `json:"value"`
	Next     string      `json:"next"`
}

// ExecuteDynamicWorkflow interprets the DSL and executes the workflow
func ExecuteDynamicWorkflow(ctx workflow.Context, dsl WorkflowDSL, input interface{}) (interface{}, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting Dynamic Workflow", "ID", dsl.ID, "Version", dsl.Version)

	currentStateName := dsl.StartAt
	stateData := input

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 10,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	for {
		stateDef, ok := dsl.States[currentStateName]
		if !ok {
			return nil, fmt.Errorf("state not found: %s", currentStateName)
		}

		logger.Info("Entering State", "State", currentStateName, "Type", stateDef.Type)

		switch stateDef.Type {
		case "task":
			var result interface{}
			// Resolve arguments (in a real system, we'd support variable interpolation like ${input.foo})
			// For now, we pass the static args + the stateData
			activityInput := stateDef.Args
			if activityInput == nil {
				activityInput = make(map[string]interface{})
			}
			activityInput["_input"] = stateData

			// Execute the activity by string name
			// Note: The activity must be registered with the worker using this name
			err := workflow.ExecuteActivity(ctx, stateDef.Activity, activityInput).Get(ctx, &result)
			if err != nil {
				return nil, err
			}

			stateData = result // Update state data with activity output

			if stateDef.End {
				return stateData, nil
			}
			currentStateName = stateDef.Next

		case "wait_signal":
			logger.Info("Waiting for signal", "Signal", stateDef.Signal)
			var signalData interface{}
			signalChan := workflow.GetSignalChannel(ctx, stateDef.Signal)
			
			// Block until signal received
			signalChan.Receive(ctx, &signalData)
			
			logger.Info("Signal received", "Data", signalData)
			// Merge signal data into state data (simplified)
			stateData = signalData 
			
			if stateDef.End {
				return stateData, nil
			}
			currentStateName = stateDef.Next

		case "choice":
			// Simplified choice logic
			nextState := stateDef.Default
			
			// In a real implementation, we would evaluate stateDef.Choices against stateData
			// For this MVP, we just take the default
			
			if nextState == "" {
				return nil, fmt.Errorf("no matching choice and no default in state: %s", currentStateName)
			}
			currentStateName = nextState

		case "succeed":
			return stateData, nil

		case "fail":
			return nil, fmt.Errorf("workflow failed in state: %s", currentStateName)

		default:
			return nil, fmt.Errorf("unknown state type: %s", stateDef.Type)
		}
	}
}
