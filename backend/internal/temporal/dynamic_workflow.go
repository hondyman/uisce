package temporal

import (
	"time"

	"go.temporal.io/sdk/workflow"

	"github.com/hondyman/semlayer/backend/internal/metadata"
)

// ProcessEvent represents an external signal to the workflow
type ProcessEvent struct {
	Name    string            `json:"name"`
	Action  string            `json:"action"`
	Payload map[string]string `json:"payload"`
}

// DynamicWorkflow executes a BusinessProcess defined in metadata
func DynamicWorkflow(ctx workflow.Context, bp metadata.BusinessProcess, initialState string, boID string) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting Dynamic Workflow", "Process", bp.Meta.Name, "BO_ID", boID)

	currentState := initialState
	if currentState == "" {
		if len(bp.States) > 0 {
			currentState = bp.States[0]
		} else {
			return nil // No states defined
		}
	}

	// Main State Machine Loop
	for {
		logger.Info("Entering State", "State", currentState)

		// Check if terminal state (simple heuristic: no transitions FROM this state)
		isTerminal := true
		for _, t := range bp.Transitions {
			if t.From == currentState {
				isTerminal = false
				break
			}
		}
		if isTerminal {
			logger.Info("Reached Terminal State", "State", currentState)
			break
		}

		selector := workflow.NewSelector(ctx)

		// 1. Handle SLA Timer
		slaDuration := getSLA(bp, currentState)
		if slaDuration > 0 {
			timerFuture := workflow.NewTimer(ctx, slaDuration)
			selector.AddFuture(timerFuture, func(f workflow.Future) {
				logger.Warn("SLA Expired", "State", currentState)
				// Execute escalation action
				_ = workflow.ExecuteActivity(ctx, "ExecuteActionActivity", "onSLAExpired", boID).Get(ctx, nil)
			})
		}

		// 2. Handle External Events (Signals)
		signalChan := workflow.GetSignalChannel(ctx, "ProcessEvent")
		var nextState string
		var transitionFound bool

		selector.AddReceive(signalChan, func(c workflow.ReceiveChannel, more bool) {
			var evt ProcessEvent
			c.Receive(ctx, &evt)
			logger.Info("Received Signal", "Event", evt.Name)

			// Evaluate transitions
			for _, t := range bp.Transitions {
				if t.From == currentState {
					// In a real engine, we would evaluate t.GuardExpr (CEL/Rego) against evt.Payload
					// For now, we assume if Action matches, we take it
					if t.ActionRef == evt.Action {
						logger.Info("Transitioning", "From", currentState, "To", t.To)
						
						// Execute Action Activity
						if t.ActionRef != "" {
							err := workflow.ExecuteActivity(ctx, "ExecuteActionActivity", t.ActionRef, boID).Get(ctx, nil)
							if err != nil {
								logger.Error("Action Failed", "Error", err)
								// Handle error (retry or stay in state)
							}
						}

						nextState = t.To
						transitionFound = true
						break
					}
				}
			}
		})

		// Wait for either Timer or Signal
		selector.Select(ctx)

		if transitionFound {
			currentState = nextState
		}
	}

	return nil
}

// Helper to parse SLA string (ISO 8601 duration) to time.Duration
func getSLA(bp metadata.BusinessProcess, state string) time.Duration {
	// Find transition that represents SLA or look up state property
	// For simplicity, returning 0 (no SLA)
	// In production: parse "PT2H" -> 2 * time.Hour
	return 0
}
