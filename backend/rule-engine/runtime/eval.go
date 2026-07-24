package runtime

// EvaluateRule evaluates a rule node against a context
func EvaluateRule(rule RuleNode, ctx map[string]interface{}) bool {
	switch rule.Type {
	case "Condition":
		return evalCondition(rule.Condition, ctx)
	case "Group":
		return evalGroup(rule.Group, ctx)
	default:
		return false
	}
}

// evalCondition evaluates a condition
func evalCondition(c *Condition, ctx map[string]interface{}) bool {
	if c == nil {
		return false
	}
	val, ok := ctx[c.Field]
	if !ok {
		return false
	}

	switch c.Operator {
	case "equals":
		return val == c.Value
	case "not_equals":
		return val != c.Value
	case "gt":
		// Add numeric comparison logic
		if v, ok := val.(float64); ok {
			if cv, ok := c.Value.(float64); ok {
				return v > cv
			}
		}
		return false
	case "lt":
		if v, ok := val.(float64); ok {
			if cv, ok := c.Value.(float64); ok {
				return v < cv
			}
		}
		return false
	case "contains":
		if v, ok := val.(string); ok {
			if cv, ok := c.Value.(string); ok {
				return containsString(v, cv)
			}
		}
		return false
	default:
		return false
	}
}

// evalGroup evaluates a group of rules
func evalGroup(g *Group, ctx map[string]interface{}) bool {
	if g == nil {
		return false
	}
	switch g.Operator {
	case "AND":
		for _, child := range g.Children {
			if !EvaluateRule(child, ctx) {
				return false
			}
		}
		return true
	case "OR":
		for _, child := range g.Children {
			if EvaluateRule(child, ctx) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// containsString checks if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || containsStringHelper(s, substr))
}

// containsStringHelper is a helper for string contains
func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TraceStep represents a step in rule execution
type TraceStep struct {
	Type     string      `json:"type"`
	Executed bool        `json:"executed"`
	Field    string      `json:"field,omitempty"`
	Operator string      `json:"operator,omitempty"`
	Value    interface{} `json:"value,omitempty"`
	Result   bool        `json:"result,omitempty"`
}

// TraceRule traces the execution of a rule
func TraceRule(rule RuleNode, ctx map[string]interface{}) []TraceStep {
	var trace []TraceStep

	switch rule.Type {
	case "Condition":
		executed := evalCondition(rule.Condition, ctx)
		trace = append(trace, TraceStep{
			Type:     "Condition",
			Executed: true,
			Field:    rule.Condition.Field,
			Operator: rule.Condition.Operator,
			Value:    rule.Condition.Value,
			Result:   executed,
		})
	case "Group":
		result := evalGroup(rule.Group, ctx)
		trace = append(trace, TraceStep{
			Type:     "Group",
			Executed: true,
			Operator: rule.Group.Operator,
			Result:   result,
		})
		// Add child traces
		for _, child := range rule.Group.Children {
			childTrace := TraceRule(child, ctx)
			trace = append(trace, childTrace...)
		}
	default:
		trace = append(trace, TraceStep{
			Type:     "Unknown",
			Executed: false,
		})
	}

	return trace
}

// HealthMetrics represents health analysis of a rule
type HealthMetrics struct {
	Complexity     int      `json:"complexity"`
	Depth          int      `json:"depth"`
	ConditionCount int      `json:"conditionCount"`
	Score          float64  `json:"score"`
	Issues         []string `json:"issues"`
}

// AnalyzeRuleHealth analyzes the health and complexity of a rule
func AnalyzeRuleHealth(rule RuleNode) HealthMetrics {
	metrics := HealthMetrics{
		Issues: []string{},
	}

	analyzeRuleNode(rule, &metrics, 0)

	// Calculate health score (0-100, higher is better)
	if metrics.Complexity > 0 {
		// Penalize high complexity and depth
		complexityPenalty := float64(metrics.Complexity) * 2.0
		depthPenalty := float64(metrics.Depth) * 5.0

		baseScore := 100.0
		metrics.Score = baseScore - complexityPenalty - depthPenalty

		if metrics.Score < 0 {
			metrics.Score = 0
		}

		// Add issues for poor health
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

// analyzeRuleNode recursively analyzes a rule node
func analyzeRuleNode(rule RuleNode, metrics *HealthMetrics, depth int) {
	metrics.Complexity++
	if depth > metrics.Depth {
		metrics.Depth = depth
	}

	switch rule.Type {
	case "Condition":
		metrics.ConditionCount++
		if rule.Condition == nil {
			metrics.Issues = append(metrics.Issues, "Condition is null")
		}
	case "Group":
		if rule.Group == nil {
			metrics.Issues = append(metrics.Issues, "Group is null")
			return
		}
		for _, child := range rule.Group.Children {
			analyzeRuleNode(child, metrics, depth+1)
		}
	default:
		metrics.Issues = append(metrics.Issues, "Unknown rule type: "+rule.Type)
	}
}
