package rules

import (
	"fmt"

	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
)

type AdvancedEvaluator struct {
	baseEvaluator *ConditionEvaluator
}

func NewAdvancedEvaluator() *AdvancedEvaluator {
	return &AdvancedEvaluator{
		baseEvaluator: NewConditionEvaluator(),
	}
}

func (ae *AdvancedEvaluator) Evaluate(node RuleNode, data map[string]interface{}) (bool, error) {
	switch node.Type {
	case NodeTypeGroup:
		if node.Group == nil {
			return false, fmt.Errorf("group node is nil")
		}
		return ae.evaluateGroup(node.Group, data)
	case NodeTypeCondition:
		if node.Condition == nil {
			return false, fmt.Errorf("condition node is nil")
		}
		return ae.evaluateCondition(node.Condition, data)
	case vm.NodeTypeExpression:
		if node.Expression == nil {
			return false, fmt.Errorf("expression node is nil")
		}
		return ae.evaluateExpression(node.Expression, data)
	default:
		return false, fmt.Errorf("unknown node type: %s", node.Type)
	}
}

func (ae *AdvancedEvaluator) evaluateGroup(group *RuleGroup, data map[string]interface{}) (bool, error) {
	if len(group.Conditions) == 0 {
		return true, nil
	}

	switch group.Operator {
	case "AND":
		for _, child := range group.Conditions {
			result, err := ae.Evaluate(child, data)
			if err != nil {
				return false, err
			}
			if !result {
				return false, nil
			}
		}
		return true, nil

	case "OR":
		for _, child := range group.Conditions {
			result, err := ae.Evaluate(child, data)
			if err != nil {
				return false, err
			}
			if result {
				return true, nil
			}
		}
		return false, nil

	case "NOT":
		for _, child := range group.Conditions {
			result, err := ae.Evaluate(child, data)
			if err != nil {
				return false, err
			}
			if !result {
				return true, nil
			}
		}
		return false, nil

	default:
		return false, fmt.Errorf("unknown group operator: %s", group.Operator)
	}
}

func (ae *AdvancedEvaluator) evaluateCondition(cond *RuleCondition, data map[string]interface{}) (bool, error) {
	field := cond.Field
	if cond.FieldPath != "" {
		field = cond.FieldPath
	}
	conditionMap := map[string]interface{}{
		"type":     "simple",
		"field":    field,
		"operator": cond.Operator,
		"value":    cond.Value,
	}
	return ae.baseEvaluator.EvaluateWithHierarchy(conditionMap, data)
}

func (ae *AdvancedEvaluator) evaluateExpression(expr *vm.Expression, data map[string]interface{}) (bool, error) {
	if expr == nil || expr.Root == nil {
		return false, fmt.Errorf("nil expression")
	}
	result, err := ae.evalExprNode(expr.Root, data)
	if err != nil {
		return false, err
	}
	if b, ok := result.(bool); ok {
		return b, nil
	}
	return false, fmt.Errorf("expression did not evaluate to bool: %T", result)
}

func (ae *AdvancedEvaluator) evalExprNode(node vm.ExprNode, data map[string]interface{}) (any, error) {
	switch n := node.(type) {
	case *vm.BinaryExpr:
		return ae.evalBinaryExpr(n, data)
	case *vm.FieldRef:
		return ae.evalFieldRef(n, data)
	case *vm.Literal:
		return n.Value, nil
	default:
		return nil, fmt.Errorf("unknown ExprNode type: %T", node)
	}
}

func (ae *AdvancedEvaluator) evalBinaryExpr(be *vm.BinaryExpr, data map[string]interface{}) (any, error) {
	lVal, err := ae.evalExprNode(be.Left, data)
	if err != nil {
		return nil, err
	}
	rVal, err := ae.evalExprNode(be.Right, data)
	if err != nil {
		return nil, err
	}

	a, b, ok := toFloat64(lVal, rVal)
	if !ok {
		return nil, fmt.Errorf("expression operands not numeric: %T, %T", lVal, rVal)
	}

	switch be.Op {
	case "+":
		return a + b, nil
	case "-":
		return a - b, nil
	case "*":
		return a * b, nil
	case "/":
		if b == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return a / b, nil
	case "==":
		return a == b, nil
	case "!=":
		return a != b, nil
	case ">":
		return a > b, nil
	case "<":
		return a < b, nil
	case ">=":
		return a >= b, nil
	case "<=":
		return a <= b, nil
	default:
		return nil, fmt.Errorf("unsupported expression operator: %s", be.Op)
	}
}

func (ae *AdvancedEvaluator) evalFieldRef(fr *vm.FieldRef, data map[string]interface{}) (any, error) {
	val, found := ae.baseEvaluator.GetFieldValue(fr.Path, data)
	if !found {
		return nil, fmt.Errorf("field not found: %s", fr.Path)
	}
	return val, nil
}

func toFloat64(a, b any) (float64, float64, bool) {
	var af, bf float64
	ok := true
	switch va := a.(type) {
	case float64:
		af = va
	case int:
		af = float64(va)
	case int64:
		af = float64(va)
	default:
		ok = false
	}
	switch vb := b.(type) {
	case float64:
		bf = vb
	case int:
		bf = float64(vb)
	case int64:
		bf = float64(vb)
	default:
		ok = false
	}
	if !ok {
		return 0, 0, false
	}
	return af, bf, true
}
