package workflows

import (
	"encoding/json"
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ============================================================================
// Event & Error Handling Nodes: Wait_For_Event, Try_Catch, Compensate
// ============================================================================

// ========================= WAIT FOR EVENT (SIGNALS) =========================

// WaitForEventConfig defines configuration for waiting on a Temporal signal
type WaitForEventConfig struct {
	SignalName     string `json:"signal_name"`     // Name of signal to wait for
	Timeout        string `json:"timeout"`         // Max wait time (e.g., "1h", "24h")
	OutputVariable string `json:"output_variable"` // Where to store signal payload
	TimeoutAction  string `json:"timeout_action"`  // "continue" or "fail"
}

// WaitForEventResult holds the result of waiting for an event
type WaitForEventResult struct {
	Received bool                   `json:"received"`
	Payload  map[string]interface{} `json:"payload"`
	TimedOut bool                   `json:"timed_out"`
}

// ExecuteWaitForEventNode waits for a Temporal signal
func ExecuteWaitForEventNode(ctx workflow.Context, config WaitForEventConfig) (*WaitForEventResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Waiting for event", "signalName", config.SignalName, "timeout", config.Timeout)

	// Create signal channel
	signalChannel := workflow.GetSignalChannel(ctx, config.SignalName)

	// Parse timeout
	var timeoutDuration time.Duration
	if config.Timeout != "" {
		var err error
		timeoutDuration, err = time.ParseDuration(config.Timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid timeout format '%s': %w", config.Timeout, err)
		}
	} else {
		// Default 24 hour timeout
		timeoutDuration = 24 * time.Hour
	}

	// Create selector for signal or timeout
	var signalPayload map[string]interface{}
	received := false
	timedOut := false

	selector := workflow.NewSelector(ctx)

	// Add signal receive
	selector.AddReceive(signalChannel, func(c workflow.ReceiveChannel, more bool) {
		c.Receive(ctx, &signalPayload)
		received = true
		logger.Info("Signal received", "signalName", config.SignalName)
	})

	// Add timeout
	selector.AddFuture(workflow.NewTimer(ctx, timeoutDuration), func(f workflow.Future) {
		timedOut = true
		logger.Warn("Signal wait timed out", "signalName", config.SignalName)
	})

	// Wait for either
	selector.Select(ctx)

	if timedOut && config.TimeoutAction == "fail" {
		return nil, temporal.NewApplicationError(
			fmt.Sprintf("timeout waiting for signal '%s'", config.SignalName),
			"SIGNAL_TIMEOUT",
		)
	}

	return &WaitForEventResult{
		Received: received,
		Payload:  signalPayload,
		TimedOut: timedOut,
	}, nil
}

// ParseWaitForEventConfig extracts config from node
func ParseWaitForEventConfig(config map[string]interface{}) (*WaitForEventConfig, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var cfg WaitForEventConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse wait for event config: %w", err)
	}

	if cfg.SignalName == "" {
		return nil, fmt.Errorf("signal_name is required for waitForEvent node")
	}
	if cfg.TimeoutAction == "" {
		cfg.TimeoutAction = "continue" // Default: continue on timeout
	}

	return &cfg, nil
}

// ========================= TRY CATCH ========================================

// TryCatchConfig defines configuration for error handling
type TryCatchConfig struct {
	TryNodeIDs     []string `json:"try_node_ids"`     // Nodes to execute in try block
	CatchNodeIDs   []string `json:"catch_node_ids"`   // Error handler nodes
	FinallyNodeIDs []string `json:"finally_node_ids"` // Always execute
	ErrorVariable  string   `json:"error_variable"`   // Store error info
}

// TryCatchResult holds the result of a try/catch execution
type TryCatchResult struct {
	TrySucceeded    bool                   `json:"try_succeeded"`
	CatchExecuted   bool                   `json:"catch_executed"`
	FinallyExecuted bool                   `json:"finally_executed"`
	Error           string                 `json:"error,omitempty"`
	TryResult       map[string]interface{} `json:"try_result,omitempty"`
	CatchResult     map[string]interface{} `json:"catch_result,omitempty"`
	FinallyResult   map[string]interface{} `json:"finally_result,omitempty"`
}

// ExecuteTryCatchNode executes nodes with try/catch/finally semantics
func ExecuteTryCatchNode(
	ctx workflow.Context,
	config TryCatchConfig,
	currentState map[string]interface{},
	executeNodesFn func(ctx workflow.Context, nodeIDs []string, state map[string]interface{}) (map[string]interface{}, error),
) (*TryCatchResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Executing try/catch block")

	result := &TryCatchResult{}

	// Execute TRY block
	tryState := copyState(currentState)
	tryResult, tryErr := executeNodesFn(ctx, config.TryNodeIDs, tryState)

	if tryErr != nil {
		// TRY failed - execute CATCH
		logger.Warn("Try block failed, executing catch", "error", tryErr)
		result.TrySucceeded = false
		result.Error = tryErr.Error()

		// Store error in state for catch block to access
		catchState := copyState(currentState)
		catchState["_error"] = map[string]interface{}{
			"message": tryErr.Error(),
			"type":    "workflow_error",
		}
		if config.ErrorVariable != "" {
			catchState[config.ErrorVariable] = catchState["_error"]
		}

		if len(config.CatchNodeIDs) > 0 {
			catchResult, catchErr := executeNodesFn(ctx, config.CatchNodeIDs, catchState)
			result.CatchExecuted = true
			if catchErr != nil {
				logger.Error("Catch block also failed", "error", catchErr)
				// Don't fail - continue to finally
			}
			result.CatchResult = catchResult
		}
	} else {
		// TRY succeeded
		result.TrySucceeded = true
		result.TryResult = tryResult
	}

	// Execute FINALLY block (always)
	if len(config.FinallyNodeIDs) > 0 {
		finallyState := copyState(currentState)
		finallyState["_try_succeeded"] = result.TrySucceeded
		finallyState["_error"] = result.Error

		finallyResult, finallyErr := executeNodesFn(ctx, config.FinallyNodeIDs, finallyState)
		result.FinallyExecuted = true
		if finallyErr != nil {
			logger.Error("Finally block failed", "error", finallyErr)
		}
		result.FinallyResult = finallyResult
	}

	logger.Info("Try/catch completed", "succeeded", result.TrySucceeded)
	return result, nil
}

// ParseTryCatchConfig extracts config from node
func ParseTryCatchConfig(config map[string]interface{}) (*TryCatchConfig, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var cfg TryCatchConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse try/catch config: %w", err)
	}

	if len(cfg.TryNodeIDs) == 0 {
		return nil, fmt.Errorf("try_node_ids is required for tryCatch node")
	}

	return &cfg, nil
}

// ========================= COMPENSATE (SAGA) ================================

// CompensateConfig defines configuration for saga compensation
type CompensateConfig struct {
	CompensationNodeIDs []string `json:"compensation_node_ids"` // Rollback nodes
	TriggerOnFailure    bool     `json:"trigger_on_failure"`    // Auto-trigger on workflow failure
	CompensationName    string   `json:"compensation_name"`     // Identifier for this compensation
}

// SagaState tracks registered compensations for a workflow
type SagaState struct {
	Compensations []RegisteredCompensation `json:"compensations"`
}

// RegisteredCompensation represents a registered compensation action
type RegisteredCompensation struct {
	Name    string   `json:"name"`
	NodeIDs []string `json:"node_ids"`
}

// RegisterCompensation adds a compensation to the saga state
func RegisterCompensation(state map[string]interface{}, config CompensateConfig) {
	// Get or create saga state
	var saga SagaState
	if existing, ok := state["_saga"].(map[string]interface{}); ok {
		data, _ := json.Marshal(existing)
		json.Unmarshal(data, &saga)
	}

	// Add new compensation
	saga.Compensations = append(saga.Compensations, RegisteredCompensation{
		Name:    config.CompensationName,
		NodeIDs: config.CompensationNodeIDs,
	})

	// Store back
	sagaMap := make(map[string]interface{})
	data, _ := json.Marshal(saga)
	json.Unmarshal(data, &sagaMap)
	state["_saga"] = sagaMap
}

// ExecuteCompensations runs all registered compensations in reverse order
func ExecuteCompensations(
	ctx workflow.Context,
	state map[string]interface{},
	executeNodesFn func(ctx workflow.Context, nodeIDs []string, state map[string]interface{}) (map[string]interface{}, error),
) error {
	logger := workflow.GetLogger(ctx)

	// Get saga state
	var saga SagaState
	if existing, ok := state["_saga"].(map[string]interface{}); ok {
		data, _ := json.Marshal(existing)
		json.Unmarshal(data, &saga)
	}

	if len(saga.Compensations) == 0 {
		logger.Info("No compensations registered")
		return nil
	}

	logger.Info("Executing compensations", "count", len(saga.Compensations))

	// Execute in reverse order (LIFO)
	for i := len(saga.Compensations) - 1; i >= 0; i-- {
		comp := saga.Compensations[i]
		logger.Info("Running compensation", "name", comp.Name)

		_, err := executeNodesFn(ctx, comp.NodeIDs, state)
		if err != nil {
			logger.Error("Compensation failed", "name", comp.Name, "error", err)
			// Continue with other compensations even if one fails
		}
	}

	return nil
}

// ParseCompensateConfig extracts config from node
func ParseCompensateConfig(config map[string]interface{}) (*CompensateConfig, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var cfg CompensateConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse compensate config: %w", err)
	}

	if len(cfg.CompensationNodeIDs) == 0 {
		return nil, fmt.Errorf("compensation_node_ids is required for compensate node")
	}

	if cfg.CompensationName == "" {
		cfg.CompensationName = "compensation"
	}

	return &cfg, nil
}

// ========================= HELPERS ==========================================

// IsEventErrorNode checks if a node type is an event/error handling node
func IsEventErrorNode(nodeType string) bool {
	switch nodeType {
	case "waitForEvent", "wait_for_event", "signal":
		return true
	case "tryCatch", "try_catch":
		return true
	case "compensate", "saga":
		return true
	default:
		return false
	}
}
