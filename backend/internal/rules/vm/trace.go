package vm

// TraceStep represents one step in a rule's execution trace. Used by the
// Monaco editor's live-preview panel to show why a rule passed or failed.
type TraceStep struct {
	Type     string      `json:"type"`
	Executed bool        `json:"executed"`
	Field    string      `json:"field,omitempty"`
	Operator string      `json:"operator,omitempty"`
	Value    interface{} `json:"value,omitempty"`
	Result   bool        `json:"result,omitempty"`
}

// TraceRule evaluates a rule and records a step-by-step trace of every
// condition and group it visited, in evaluation order.
func TraceRule(rule RuleNode, ctx map[string]interface{}) []TraceStep {
	ae := NewAdvancedEvaluator()
	var trace []TraceStep
	traceNode(ae, rule, ctx, &trace)
	return trace
}

func traceNode(ae *AdvancedEvaluator, rule RuleNode, ctx map[string]interface{}, trace *[]TraceStep) {
	switch rule.Type {
	case NodeTypeCondition:
		if rule.Condition == nil {
			*trace = append(*trace, TraceStep{Type: "Condition", Executed: false})
			return
		}
		result, _ := ae.Evaluate(rule, ctx)
		*trace = append(*trace, TraceStep{
			Type:     "Condition",
			Executed: true,
			Field:    rule.Condition.Field,
			Operator: rule.Condition.Operator,
			Value:    rule.Condition.Value,
			Result:   result,
		})
	case NodeTypeGroup:
		if rule.Group == nil {
			*trace = append(*trace, TraceStep{Type: "Group", Executed: false})
			return
		}
		result, _ := ae.Evaluate(rule, ctx)
		*trace = append(*trace, TraceStep{
			Type:     "Group",
			Executed: true,
			Operator: rule.Group.Operator,
			Result:   result,
		})
		for _, child := range rule.Group.Conditions {
			traceNode(ae, child, ctx, trace)
		}
	case NodeTypeExpression:
		result, err := ae.EvaluateNumeric(rule, ctx)
		*trace = append(*trace, TraceStep{
			Type:     "Expression",
			Executed: err == nil,
			Value:    result,
		})
	default:
		*trace = append(*trace, TraceStep{Type: "Unknown", Executed: false})
	}
}

// HealthMetrics summarizes the complexity/depth of a rule, surfaced in the
// Monaco editor as authoring guidance.
type HealthMetrics struct {
	Complexity     int      `json:"complexity"`
	Depth          int      `json:"depth"`
	ConditionCount int      `json:"conditionCount"`
	Score          float64  `json:"score"`
	Issues         []string `json:"issues"`
}

// AnalyzeRuleHealth analyzes the complexity and depth of a rule.
func AnalyzeRuleHealth(rule RuleNode) HealthMetrics {
	metrics := HealthMetrics{Issues: []string{}}
	analyzeRuleNode(rule, &metrics, 0)

	if metrics.Complexity > 0 {
		complexityPenalty := float64(metrics.Complexity) * 2.0
		depthPenalty := float64(metrics.Depth) * 5.0

		metrics.Score = 100.0 - complexityPenalty - depthPenalty
		if metrics.Score < 0 {
			metrics.Score = 0
		}

		if metrics.Complexity > 10 {
			metrics.Issues = append(metrics.Issues, "High complexity - consider simplifying the rule")
		}
		if metrics.Depth > 3 {
			metrics.Issues = append(metrics.Issues, "Deep nesting - consider flattening the rule structure")
		}
		if metrics.ConditionCount == 0 {
			metrics.Issues = append(metrics.Issues, "No conditions found - rule may not be meaningful")
		}
	} else {
		metrics.Score = 0
		metrics.Issues = append(metrics.Issues, "Invalid rule structure")
	}

	return metrics
}

func analyzeRuleNode(rule RuleNode, metrics *HealthMetrics, depth int) {
	metrics.Complexity++
	if depth > metrics.Depth {
		metrics.Depth = depth
	}

	switch rule.Type {
	case NodeTypeCondition:
		metrics.ConditionCount++
		if rule.Condition == nil {
			metrics.Issues = append(metrics.Issues, "Condition is null")
		}
	case NodeTypeGroup:
		if rule.Group == nil {
			metrics.Issues = append(metrics.Issues, "Group is null")
			return
		}
		for _, child := range rule.Group.Conditions {
			analyzeRuleNode(child, metrics, depth+1)
		}
	case NodeTypeExpression:
		if rule.Expression == nil {
			metrics.Issues = append(metrics.Issues, "Expression is null")
		}
	default:
		metrics.Issues = append(metrics.Issues, "Unknown rule type: "+string(rule.Type))
	}
}
