package workflows

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.temporal.io/sdk/workflow"
)

// ============================================================================
// Control Flow Nodes: Parallel, ForEach, Wait
// ============================================================================

// =========================== PARALLEL =======================================

// ParallelConfig defines configuration for parallel branch execution
type ParallelConfig struct {
	Branches       []ParallelBranch `json:"branches"`        // List of branches to execute
	OutputVariable string           `json:"output_variable"` // Where to store combined results
}

// ParallelBranch defines a single branch in parallel execution
type ParallelBranch struct {
	Name    string   `json:"name"`     // Branch identifier
	NodeIDs []string `json:"node_ids"` // Node IDs to execute in sequence within this branch
}

// ParallelResult holds the results from all parallel branches
type ParallelResult struct {
	BranchResults map[string]interface{} `json:"branch_results"`
	AllSucceeded  bool                   `json:"all_succeeded"`
}

// ExecuteParallelNode executes multiple branches concurrently
// Uses Temporal's workflow.Go for fan-out and channel-based collection for fan-in
func ExecuteParallelNode(
	ctx workflow.Context,
	config ParallelConfig,
	currentState map[string]interface{},
	executeNodesFn func(ctx workflow.Context, nodeIDs []string, state map[string]interface{}) (map[string]interface{}, error),
) (*ParallelResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Executing parallel node", "branchCount", len(config.Branches))

	// Create channels for results and errors
	resultChan := workflow.NewChannel(ctx)
	errChan := workflow.NewChannel(ctx)

	// Launch each branch as a goroutine
	for _, branch := range config.Branches {
		branchCopy := branch // Capture for closure
		workflow.Go(ctx, func(gCtx workflow.Context) {
			// Copy state for this branch to avoid race conditions
			branchState := copyState(currentState)

			// Execute the nodes in this branch
			result, err := executeNodesFn(gCtx, branchCopy.NodeIDs, branchState)
			if err != nil {
				errChan.Send(gCtx, fmt.Errorf("branch '%s' failed: %w", branchCopy.Name, err))
				return
			}

			// Send result back
			resultChan.Send(gCtx, map[string]interface{}{
				"name":   branchCopy.Name,
				"result": result,
			})
		})
	}

	// Collect results from all branches
	branchResults := make(map[string]interface{})
	var firstError error
	completed := 0

	for completed < len(config.Branches) {
		selector := workflow.NewSelector(ctx)

		selector.AddReceive(resultChan, func(c workflow.ReceiveChannel, more bool) {
			var res map[string]interface{}
			c.Receive(ctx, &res)
			name := res["name"].(string)
			branchResults[name] = res["result"]
			completed++
		})

		selector.AddReceive(errChan, func(c workflow.ReceiveChannel, more bool) {
			var err error
			c.Receive(ctx, &err)
			if firstError == nil {
				firstError = err
			}
			completed++
		})

		selector.Select(ctx)
	}

	if firstError != nil {
		logger.Error("Parallel execution had failures", "error", firstError)
		return &ParallelResult{
			BranchResults: branchResults,
			AllSucceeded:  false,
		}, firstError
	}

	logger.Info("Parallel execution completed", "branchCount", len(config.Branches))
	return &ParallelResult{
		BranchResults: branchResults,
		AllSucceeded:  true,
	}, nil
}

// ParseParallelConfig extracts ParallelConfig from node config
func ParseParallelConfig(config map[string]interface{}) (*ParallelConfig, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var cfg ParallelConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse parallel config: %w", err)
	}

	if len(cfg.Branches) == 0 {
		return nil, fmt.Errorf("parallel node requires at least one branch")
	}

	return &cfg, nil
}

// =========================== FOR EACH =======================================

// ForEachConfig defines configuration for iterating over a collection
type ForEachConfig struct {
	Collection     string   `json:"collection"`      // JSONPath to array (e.g., "$.items")
	ItemVariable   string   `json:"item_variable"`   // Variable name for current item
	IndexVariable  string   `json:"index_variable"`  // Variable name for current index
	Mode           string   `json:"mode"`            // "sequential" or "parallel"
	MaxConcurrency int      `json:"max_concurrency"` // Max concurrent iterations (parallel mode)
	BodyNodeIDs    []string `json:"body_node_ids"`   // Nodes to execute per iteration
	OutputVariable string   `json:"output_variable"` // Where to store collected results
}

// ForEachResult holds the results from all iterations
type ForEachResult struct {
	ItemResults []interface{} `json:"item_results"`
	TotalItems  int           `json:"total_items"`
	Succeeded   int           `json:"succeeded"`
	Failed      int           `json:"failed"`
}

// ExecuteForEachNode iterates over a collection and executes body nodes for each item
func ExecuteForEachNode(
	ctx workflow.Context,
	config ForEachConfig,
	currentState map[string]interface{},
	executeNodesFn func(ctx workflow.Context, nodeIDs []string, state map[string]interface{}) (map[string]interface{}, error),
) (*ForEachResult, error) {
	logger := workflow.GetLogger(ctx)

	// Resolve collection from state
	collection, err := resolveCollection(config.Collection, currentState)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve collection '%s': %w", config.Collection, err)
	}

	logger.Info("Executing forEach node", "itemCount", len(collection), "mode", config.Mode)

	result := &ForEachResult{
		ItemResults: make([]interface{}, len(collection)),
		TotalItems:  len(collection),
	}

	if config.Mode == "parallel" {
		// Parallel mode with optional concurrency limit
		maxConcurrency := config.MaxConcurrency
		if maxConcurrency <= 0 {
			maxConcurrency = len(collection) // No limit
		}

		// Use semaphore pattern for concurrency control
		semaphore := make(chan struct{}, maxConcurrency)
		resultChan := workflow.NewChannel(ctx)

		for idx, item := range collection {
			idxCopy := idx
			itemCopy := item

			workflow.Go(ctx, func(gCtx workflow.Context) {
				// Acquire semaphore
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				// Create state for this iteration
				iterState := copyState(currentState)
				iterState[config.ItemVariable] = itemCopy
				if config.IndexVariable != "" {
					iterState[config.IndexVariable] = idxCopy
				}

				// Execute body nodes
				iterResult, err := executeNodesFn(gCtx, config.BodyNodeIDs, iterState)
				resultChan.Send(gCtx, map[string]interface{}{
					"index":  idxCopy,
					"result": iterResult,
					"error":  err,
				})
			})
		}

		// Collect all results
		for i := 0; i < len(collection); i++ {
			var res map[string]interface{}
			resultChan.Receive(ctx, &res)

			idx := res["index"].(int)
			if res["error"] != nil {
				result.Failed++
				result.ItemResults[idx] = map[string]interface{}{"error": res["error"]}
			} else {
				result.Succeeded++
				result.ItemResults[idx] = res["result"]
			}
		}
	} else {
		// Sequential mode (default)
		for idx, item := range collection {
			// Create state for this iteration
			iterState := copyState(currentState)
			iterState[config.ItemVariable] = item
			if config.IndexVariable != "" {
				iterState[config.IndexVariable] = idx
			}

			// Execute body nodes
			iterResult, err := executeNodesFn(ctx, config.BodyNodeIDs, iterState)
			if err != nil {
				result.Failed++
				result.ItemResults[idx] = map[string]interface{}{"error": err.Error()}
			} else {
				result.Succeeded++
				result.ItemResults[idx] = iterResult
			}
		}
	}

	logger.Info("ForEach completed", "total", result.TotalItems, "succeeded", result.Succeeded, "failed", result.Failed)
	return result, nil
}

// resolveCollection extracts an array from state using JSONPath
func resolveCollection(path string, state map[string]interface{}) ([]interface{}, error) {
	// Remove leading $. if present
	path = strings.TrimPrefix(path, "$.")

	// Navigate to the collection
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

	// Ensure result is an array
	switch v := current.(type) {
	case []interface{}:
		return v, nil
	case []map[string]interface{}:
		result := make([]interface{}, len(v))
		for i, m := range v {
			result[i] = m
		}
		return result, nil
	default:
		return nil, fmt.Errorf("path '%s' does not resolve to an array", path)
	}
}

// ParseForEachConfig extracts ForEachConfig from node config
func ParseForEachConfig(config map[string]interface{}) (*ForEachConfig, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var cfg ForEachConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse forEach config: %w", err)
	}

	if cfg.Collection == "" {
		return nil, fmt.Errorf("forEach node requires a collection path")
	}
	if cfg.ItemVariable == "" {
		cfg.ItemVariable = "item" // Default
	}
	if cfg.Mode == "" {
		cfg.Mode = "sequential" // Default
	}

	return &cfg, nil
}

// =========================== WAIT ===========================================

// WaitConfig defines configuration for wait/timer node
type WaitConfig struct {
	Duration  string `json:"duration"`   // e.g., "5m", "1h", "24h"
	UntilTime string `json:"until_time"` // ISO8601 datetime
	WaitType  string `json:"wait_type"`  // "duration" or "until"
}

// ExecuteWaitNode pauses workflow execution for specified duration
func ExecuteWaitNode(ctx workflow.Context, config WaitConfig) error {
	logger := workflow.GetLogger(ctx)

	var sleepDuration time.Duration

	if config.WaitType == "until" && config.UntilTime != "" {
		// Parse target time
		targetTime, err := time.Parse(time.RFC3339, config.UntilTime)
		if err != nil {
			return fmt.Errorf("invalid until_time format: %w", err)
		}

		// Calculate duration until target
		now := workflow.Now(ctx)
		sleepDuration = targetTime.Sub(now)
		if sleepDuration < 0 {
			logger.Warn("Wait until time is in the past, skipping")
			return nil
		}
	} else {
		// Parse duration string
		var err error
		sleepDuration, err = time.ParseDuration(config.Duration)
		if err != nil {
			return fmt.Errorf("invalid duration format '%s': %w", config.Duration, err)
		}
	}

	logger.Info("Executing wait node", "duration", sleepDuration.String())

	// Use Temporal's durable sleep
	err := workflow.Sleep(ctx, sleepDuration)
	if err != nil {
		return fmt.Errorf("wait interrupted: %w", err)
	}

	logger.Info("Wait completed")
	return nil
}

// ParseWaitConfig extracts WaitConfig from node config
func ParseWaitConfig(config map[string]interface{}) (*WaitConfig, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var cfg WaitConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse wait config: %w", err)
	}

	if cfg.Duration == "" && cfg.UntilTime == "" {
		return nil, fmt.Errorf("wait node requires either duration or until_time")
	}

	if cfg.WaitType == "" {
		if cfg.Duration != "" {
			cfg.WaitType = "duration"
		} else {
			cfg.WaitType = "until"
		}
	}

	return &cfg, nil
}

// =========================== HELPERS ========================================

// copyState creates a shallow copy of state for branch isolation
func copyState(state map[string]interface{}) map[string]interface{} {
	copy := make(map[string]interface{})
	for k, v := range state {
		copy[k] = v
	}
	return copy
}

// IsControlFlowNode checks if a node type is a control flow node
func IsControlFlowNode(nodeType string) bool {
	switch strings.ToLower(nodeType) {
	case "parallel", "foreach", "for_each", "wait", "timer":
		return true
	default:
		return false
	}
}
