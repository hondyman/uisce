package mdm

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/analytics"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// ExecutionTrace represents the trace of a calculation execution
type ExecutionTrace struct {
	TermID       uuid.UUID                 `json:"term_id"`
	TermName     string                    `json:"term_name"`
	Inputs       map[string]interface{}    `json:"inputs"`
	Output       interface{}               `json:"output"`
	Dependencies map[string]ExecutionTrace `json:"dependencies,omitempty"`
	Error        string                    `json:"error,omitempty"`
}

// ExecutionEngine handles recursive semantic term resolution and execution
type ExecutionEngine struct {
	graphService *analytics.SemanticGraphService
	wasmRuntime  wazero.Runtime
	moduleCache  sync.Map // Map[string]wazero.CompiledModule
	monitor      *analytics.ExecutionMonitorService
}

// NewExecutionEngine creates a new execution engine
func NewExecutionEngine(ctx context.Context, graphService *analytics.SemanticGraphService, monitor *analytics.ExecutionMonitorService) (*ExecutionEngine, error) {
	r := wazero.NewRuntime(ctx)

	// Add WASI to the runtime
	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	return &ExecutionEngine{
		graphService: graphService,
		wasmRuntime:  r,
		monitor:      monitor,
	}, nil
}

// Close closes the runtime
func (e *ExecutionEngine) Close(ctx context.Context) error {
	return e.wasmRuntime.Close(ctx)
}

// ExecuteCalculation resolves dependencies and executes a calculation term
func (e *ExecutionEngine) ExecuteCalculation(ctx context.Context, termID uuid.UUID, context map[string]interface{}) (interface{}, *ExecutionTrace, error) {
	start := time.Now()
	trace := &ExecutionTrace{TermID: termID}

	defer func() {
		// Log to operational monitor if available
		if e.monitor != nil {
			status := "success"
			if trace.Error != "" {
				status = "error"
			}

			// Extract tenant from context if available
			tenantID := uuid.Nil
			if t, ok := context["TenantID"].(uuid.UUID); ok {
				tenantID = t
			}

			// Capture metrics for Ops Cockpit
			// We cast to interface to avoid circular imports if needed, or use the service directly
			duration := time.Since(start)
			e.monitor.RecordMetric(ctx, analytics.ExecutionMetric{
				TenantID: tenantID,
				TermID:   termID,
				TermName: trace.TermName,
				Duration: duration,
				Status:   status,
				Engine:   "semantic-fabric",
			})
		}
	}()

	// 1. Get the node
	node, err := e.graphService.GetNodeByID(termID)
	if err != nil {
		trace.Error = "failed to get node"
		return nil, trace, err
	}
	if node == nil {
		trace.Error = "node not found"
		return nil, trace, fmt.Errorf("node not found: %s", termID)
	}
	trace.TermName = node.NodeName

	// 2. Resolve dependencies from outgoing edges
	edges, err := e.graphService.GetOutgoingEdges(termID)
	if err != nil {
		trace.Error = "failed to get dependencies"
		return nil, trace, err
	}

	inputs := make(map[string]interface{})
	trace.Dependencies = make(map[string]ExecutionTrace)

	for _, edge := range edges {
		// Only follow calculation dependency edges
		if edge.EdgeType == "calc_depends_on_term" || edge.EdgeType == "calc_depends_on_calc" {
			targetID := edge.TargetNodeID

			// Get target name if not in context
			targetNode, _ := e.graphService.GetNodeByID(targetID)
			targetName := ""
			if targetNode != nil {
				targetName = targetNode.NodeName
			}

			// Check if already in context (base case/leaf)
			if val, ok := context[targetName]; ok {
				inputs[targetName] = val
				trace.Dependencies[targetName] = ExecutionTrace{
					TermID:   targetID,
					TermName: targetName,
					Output:   val,
				}
				continue
			}

			// Recursive resolution
			res, depTrace, err := e.ExecuteCalculation(ctx, targetID, context)
			if err != nil {
				trace.Error = fmt.Sprintf("dependency error: %s", targetName)
				return nil, trace, err
			}
			inputs[targetName] = res
			trace.Dependencies[targetName] = *depTrace
		}
	}

	trace.Inputs = inputs

	// 3. Execution logic based on node properties
	engine, _ := node.Properties["engine"].(string)
	expression, _ := node.Properties["expression"].(string)

	var result interface{}
	switch engine {
	case "wasm":
		result, err = e.executeWASM(ctx, expression, inputs)
	case "mock":
		// Simple mock engine for testing: if expression is "sum", sum inputs
		if expression == "sum" {
			var sum float64
			for _, v := range inputs {
				if f, ok := v.(float64); ok {
					sum += f
				}
			}
			result = sum
		} else {
			result = 0.0
		}
	default:
		// Fallback to mock for now if not specified
		result = 0.0
	}

	if err != nil {
		trace.Error = err.Error()
		return nil, trace, err
	}

	trace.Output = result
	return result, trace, nil
}

// executeWASM pseudo-implementation for demo
// In a real scenario, 'expression' would be a key to a WASM module or the bytecode itself
func (e *ExecutionEngine) executeWASM(ctx context.Context, expression string, inputs map[string]interface{}) (interface{}, error) {
	// For this prototype, we'll treat the 'expression' as a simple JS-like formula
	// that we map to a "built-in" WASM module or a placeholder.
	// In a full implementation, we would load the WASM binary associated with the term.

	// Example placeholder logic:
	if strings.Contains(expression, "sum") {
		var sum float64
		for _, v := range inputs {
			switch val := v.(type) {
			case float64:
				sum += val
			case int:
				sum += float64(val)
			}
		}
		return sum, nil
	}

	return nil, fmt.Errorf("WASM engine: expression not implemented: %s", expression)
}
