package rules

import (
	"fmt"
	"strings"
)

func EvaluateRecursiveWithDiagnostics(evaluator *ConditionEvaluator, node *RuleNode, data map[string]any) (bool, []RuleViolation, error) {
	if node == nil {
		return true, nil, nil
	}

	if node.Condition != nil {
		passed, leftVal, rightVal, err := evaluator.EvaluateConditionWithValues(node.Condition, data)
		if err != nil {
			return false, nil, err
		}

		if !passed {
			path := node.Condition.FieldPath
			if path == "" {
				path = node.Condition.Field
			}
			violations := []RuleViolation{{
				ConditionID:    node.Condition.ID,
				FieldPath:      path,
				Operator:       node.Condition.Operator,
				EvaluatedVal:   leftVal,
				ThresholdLimit: rightVal,
				Message:        fmt.Sprintf("Compliance limit breached: %s %s %v", path, node.Condition.Operator, rightVal),
			}}
			return false, violations, nil
		}
		return true, nil, nil
	}

	if node.Group != nil && len(node.Group.Conditions) > 0 {
		switch strings.ToUpper(node.Group.Operator) {
		case "AND":
			allPassed := true
			var allViolations []RuleViolation
			for i := range node.Group.Conditions {
				p, v, err := EvaluateRecursiveWithDiagnostics(evaluator, &node.Group.Conditions[i], data)
				if err != nil {
					return false, nil, err
				}
				if !p {
					allPassed = false
					allViolations = append(allViolations, v...)
				}
			}
			return allPassed, allViolations, nil

		case "OR":
			for i := range node.Group.Conditions {
				p, _, err := EvaluateRecursiveWithDiagnostics(evaluator, &node.Group.Conditions[i], data)
				if err != nil {
					return false, nil, err
				}
				if p {
					return true, nil, nil
				}
			}
			return false, []RuleViolation{{
				Message: "None of the OR conditions were satisfied",
			}}, nil
		}
	}

	return true, nil, nil
}
