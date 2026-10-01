package mdm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/analytics"
	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
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

// calcGraph is the slice of the semantic graph the engine reads: a term and
// its calc_depends_on_* edges. *analytics.SemanticGraphService satisfies it.
type calcGraph interface {
	GetNodeByID(nodeID uuid.UUID) (*analytics.SemanticNode, error)
	GetOutgoingEdges(nodeID uuid.UUID) ([]analytics.SemanticEdge, error)
}

// ExecutionEngine resolves a calculation term's dependencies recursively and
// evaluates it with the platform's single rule engine, internal/rules/vm.
type ExecutionEngine struct {
	graphService calcGraph
	monitor      *analytics.ExecutionMonitorService
}

// NewExecutionEngine creates a new execution engine
func NewExecutionEngine(ctx context.Context, graphService *analytics.SemanticGraphService, monitor *analytics.ExecutionMonitorService) (*ExecutionEngine, error) {
	return &ExecutionEngine{
		graphService: graphService,
		monitor:      monitor,
	}, nil
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

	// 3. Evaluate with internal/rules/vm
	result, err := evaluateTerm(node, inputs)
	if err != nil {
		trace.Error = err.Error()
		return nil, trace, err
	}

	trace.Output = result
	return result, trace, nil
}

// evaluateTerm computes a calculation term with internal/rules/vm, the same
// engine and function library calc terms use everywhere else
// (analytics.CalcTermService, SQL pushdown, the browser WASM build). The
// expression comes from, in order: config.rule_ast (an Expression AST - the
// calc-term storage convention), config.expression, or the legacy
// properties.expression. inputs are the resolved dependencies, by term name.
//
// There is no fallback value: a term with no evaluable expression is an
// error, never 0 - the engine this replaces returned 0.0 for anything it
// did not recognise.
func evaluateTerm(node *analytics.SemanticNode, inputs map[string]interface{}) (float64, error) {
	expr, err := termExpression(node)
	if err != nil {
		return 0, fmt.Errorf("term %q: %w", node.NodeName, err)
	}
	data := make(map[string]interface{}, len(inputs))
	for k, v := range inputs {
		data[k] = numericInput(v)
	}
	res, err := vm.NewAdvancedEvaluator().EvaluateNumeric(vm.RuleNode{Type: vm.NodeTypeExpression, Expression: expr}, data)
	if err != nil {
		return 0, fmt.Errorf("term %q: %w", node.NodeName, err)
	}
	return res, nil
}

func termExpression(node *analytics.SemanticNode) (*vm.Expression, error) {
	if raw, ok := node.Config["rule_ast"]; ok && raw != nil {
		b, err := json.Marshal(raw)
		if err != nil {
			return nil, fmt.Errorf("rule_ast: %w", err)
		}
		var expr vm.Expression
		if err := json.Unmarshal(b, &expr); err == nil && expr.Root != nil {
			return &expr, nil
		}
		var rn vm.RuleNode
		if err := json.Unmarshal(b, &rn); err == nil && rn.Type == vm.NodeTypeExpression && rn.Expression != nil {
			return rn.Expression, nil
		}
		return nil, fmt.Errorf("rule_ast is not an expression")
	}
	src, _ := node.Config["expression"].(string)
	if strings.TrimSpace(src) == "" {
		src, _ = node.Properties["expression"].(string)
	}
	src = stripAssignment(src)
	if strings.TrimSpace(src) == "" {
		return nil, fmt.Errorf("no expression (config.rule_ast, config.expression or properties.expression) and no value supplied in the calculation context")
	}
	return vm.ParseExpression(src)
}

// stripAssignment drops a legacy "NAME = " prefix ("NAV = sum(PositionValue)"),
// which the vm expression grammar does not accept. "==" is left alone.
func stripAssignment(src string) string {
	i := strings.Index(src, "=")
	if i <= 0 || (i+1 < len(src) && src[i+1] == '=') {
		return src
	}
	name := strings.TrimSpace(src[:i])
	for j, r := range name {
		if !(r == '_' || unicode.IsLetter(r) || (j > 0 && unicode.IsDigit(r))) {
			return src
		}
	}
	if name == "" {
		return src
	}
	return strings.TrimSpace(src[i+1:])
}

func numericInput(v interface{}) interface{} {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int32:
		return float64(n)
	case int64:
		return float64(n)
	case float32:
		return float64(n)
	}
	return v
}
