package vm

import "sort"

// FieldRefs returns every field a rule reads - condition fields (or their
// FieldPath) and expression field references - sorted and de-duplicated.
// Callers use it to fail loud before evaluation when a referenced field is
// not in the data: Condition treats a missing field as false, which would
// otherwise make an unbound term read as an ordinary rule outcome.
func FieldRefs(node RuleNode) []string {
	set := map[string]bool{}
	collectRuleFieldRefs(node, set)
	out := make([]string, 0, len(set))
	for f := range set {
		out = append(out, f)
	}
	sort.Strings(out)
	return out
}

func collectRuleFieldRefs(node RuleNode, out map[string]bool) {
	switch node.Type {
	case NodeTypeGroup:
		if node.Group != nil {
			for _, c := range node.Group.Conditions {
				collectRuleFieldRefs(c, out)
			}
		}
	case NodeTypeCondition:
		if node.Condition != nil {
			f := node.Condition.Field
			if node.Condition.FieldPath != "" {
				f = node.Condition.FieldPath
			}
			if f != "" {
				out[f] = true
			}
		}
	case NodeTypeExpression:
		if node.Expression != nil {
			collectExprFieldRefs(node.Expression.Root, out)
		}
	}
}

func collectExprFieldRefs(n ExprNode, out map[string]bool) {
	switch t := n.(type) {
	case *BinaryExpr:
		collectExprFieldRefs(t.Left, out)
		collectExprFieldRefs(t.Right, out)
	case *FieldRef:
		out[t.Path] = true
	case *FuncCall:
		for _, a := range t.Args {
			collectExprFieldRefs(a, out)
		}
	}
}
