import type { ConditionNode } from '../components/ExpressionBuilder/AdvancedConditionBuilder';

// Converts the condition builder's ConditionNode shape into the wire format
// internal/rules/vm.RuleNode.UnmarshalJSON expects (flat "type" + sibling
// fields, not nested under a "Condition"/"Group" key - see
// backend/internal/rules/vm/ast.go). Structural discrimination
// ("conditions" in node) rather than trusting node.type, since Condition
// nodes from the builder don't always set an explicit type.
//
// Every evaluation of a builder tree goes through this and the rule engine
// (evaluateRuleWasm) - there is no second evaluator in the frontend.
export function toRuleNode(node: ConditionNode): unknown {
  if ('conditions' in node) {
    return {
      type: 'group',
      id: node.id,
      operator: node.operator,
      conditions: node.conditions.map(toRuleNode),
    };
  }
  return {
    type: 'condition',
    id: node.id,
    field: node.fieldPath || node.field,
    operator: node.operator,
    value: node.value,
    ...(node.secondValue !== undefined ? { secondValue: node.secondValue } : {}),
  };
}
