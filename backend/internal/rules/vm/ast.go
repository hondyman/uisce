package vm

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

type RuleNodeType string

const (
	NodeTypeGroup       RuleNodeType = "group"
	NodeTypeCondition  RuleNodeType = "condition"
	NodeTypeExpression RuleNodeType = "expression"
)

type RuleNode struct {
	Type       RuleNodeType
	Group      *RuleGroup
	Condition  *RuleCondition
	Expression *Expression
}

type RuleGroup struct {
	ID         string
	Operator   string
	Conditions []RuleNode
}

type RuleCondition struct {
	ID          string
	Field       string
	FieldPath   string
	Operator    string
	Value       any
	ValueType   string
	SecondValue any
}

type ExprNode interface {
	exprNode()
}

type BinaryExpr struct {
	Op    string
	Left  ExprNode
	Right ExprNode
}

type FieldRef struct {
	Path string
}

type Literal struct {
	Value float64
}

// FuncCall represents a named function applied to a list of argument
// expressions, e.g. SUM(field) or NPV(rate, cash_flows). Added to let the
// same rule/calc AST express aggregate and financial functions used by
// calculated semantic terms, not just the scalar arithmetic BinaryExpr
// already supported. Backends (VM compiler, SQL compiler, ...) that don't
// yet support a given function name should fail explicitly rather than
// silently mis-evaluate.
type FuncCall struct {
	Name string
	Args []ExprNode
}

func (*BinaryExpr) exprNode() {}
func (*FieldRef) exprNode()   {}
func (*Literal) exprNode()    {}
func (*FuncCall) exprNode()   {}

type Expression struct {
	Root ExprNode
}

// MarshalJSON produces {"root": ...}, matching Expression.UnmarshalJSON.
// Standalone Expression marshaling (e.g. calc-term rule_ast, which stores
// just {"root": {...}} directly rather than wrapping it in a RuleNode) -
// RuleNode.MarshalJSON's NodeTypeExpression case sets "root" itself rather
// than delegating here, since it needs "root" flat alongside "type", but
// this keeps a plain Expression value equally round-trippable on its own.
func (e Expression) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{"root": e.Root})
}

func (be *BinaryExpr) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{"op": be.Op, "left": be.Left, "right": be.Right})
}

func (fr *FieldRef) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{"path": fr.Path})
}

func (l *Literal) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{"value": l.Value})
}

func (fc *FuncCall) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{"func": fc.Name, "args": fc.Args})
}

// MarshalJSON flattens n into the same wire shape UnmarshalJSON reads back:
// {"type": ..., <fields of whichever of Group/Condition/Expression is set,
// as direct siblings of "type" - never nested under a "Group"/"Condition"/
// "Expression" key>}. Value receiver (not pointer) so this is picked up
// when RuleNode appears by value, e.g. RuleGroup.Conditions []RuleNode.
//
// Without this, encoding/json's default struct marshaling produces
// {"Type":"expression","Expression":{"Root":{...}}} - capitalized Go
// field names, and Expression nested under its own key - which
// UnmarshalJSON cannot read back (it looks for a flat, lowercase "root"
// sibling of "type"). That silent asymmetry was found by round-trip
// testing, not by inspection - Marshal always "succeeded" and produced
// well-formed JSON, it just wasn't JSON this package could read.
func (n RuleNode) MarshalJSON() ([]byte, error) {
	out := map[string]any{"type": n.Type}
	switch n.Type {
	case NodeTypeGroup:
		if n.Group != nil {
			out["id"] = n.Group.ID
			out["operator"] = n.Group.Operator
			out["conditions"] = n.Group.Conditions
		}
	case NodeTypeCondition:
		if n.Condition != nil {
			out["id"] = n.Condition.ID
			out["field"] = n.Condition.Field
			out["fieldPath"] = n.Condition.FieldPath
			out["operator"] = n.Condition.Operator
			out["value"] = n.Condition.Value
			out["valueType"] = n.Condition.ValueType
			out["secondValue"] = n.Condition.SecondValue
		}
	case NodeTypeExpression:
		if n.Expression != nil {
			out["root"] = n.Expression.Root
		}
	}
	return json.Marshal(out)
}

func (n *RuleNode) UnmarshalJSON(data []byte) error {
	var temp struct {
		Type RuleNodeType `json:"type"`
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	n.Type = temp.Type

	switch n.Type {
	case NodeTypeGroup:
		var g RuleGroup
		if err := json.Unmarshal(data, &g); err != nil {
			return err
		}
		n.Group = &g
	case NodeTypeCondition:
		var c RuleCondition
		if err := json.Unmarshal(data, &c); err != nil {
			return err
		}
		n.Condition = &c
	case NodeTypeExpression:
		var e Expression
		if err := json.Unmarshal(data, &e); err != nil {
			return err
		}
		n.Expression = &e
	default:
		return fmt.Errorf("unknown rule node type: %s", n.Type)
	}
	return nil
}

func (e *Expression) UnmarshalJSON(data []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if root, ok := raw["root"]; ok {
		rootData, err := json.Marshal(root)
		if err != nil {
			return err
		}
		node, err := unmarshalExprNode(rootData)
		if err != nil {
			return err
		}
		e.Root = node
	}
	return nil
}

func unmarshalExprNode(data []byte) (ExprNode, error) {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if _, ok := m["op"].(string); ok {
		rootData, err := json.Marshal(m)
		if err != nil {
			return nil, err
		}
		var be BinaryExpr
		if err := json.Unmarshal(rootData, &be); err != nil {
			return nil, err
		}
		return &be, nil
	}
	if name, ok := m["func"].(string); ok {
		rawArgs, _ := m["args"].([]any)
		args := make([]ExprNode, 0, len(rawArgs))
		for _, ra := range rawArgs {
			argData, err := json.Marshal(ra)
			if err != nil {
				return nil, err
			}
			argNode, err := unmarshalExprNode(argData)
			if err != nil {
				return nil, err
			}
			args = append(args, argNode)
		}
		return &FuncCall{Name: name, Args: args}, nil
	}
	if path, ok := m["path"].(string); ok {
		return &FieldRef{Path: path}, nil
	}
	if val, ok := m["value"].(float64); ok {
		return &Literal{Value: val}, nil
	}
	return nil, fmt.Errorf("unknown ExprNode shape: %s", string(data))
}

func (be *BinaryExpr) UnmarshalJSON(data []byte) error {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	be.Op = m["op"].(string)
	if leftData, err := json.Marshal(m["left"]); err != nil {
		return err
	} else if leftNode, err := unmarshalExprNode(leftData); err != nil {
		return err
	} else {
		be.Left = leftNode
	}
	if rightData, err := json.Marshal(m["right"]); err != nil {
		return err
	} else if rightNode, err := unmarshalExprNode(rightData); err != nil {
		return err
	} else {
		be.Right = rightNode
	}
	return nil
}

type RuleRecord struct {
	ID                 uuid.UUID
	TenantID           uuid.UUID
	TargetEntityID     uuid.UUID
	Name               string
	Description        string
	RuleType           string
	CompiledSQL        string
	CompiledWASM       []byte
	CompiledCUE        string
	ExecuteServerSide  bool
	ExecuteClientSide  bool
	RunOnSubmit        bool
	Severity           string
	RemediationHint    string
	EvaluationOrder    int
	IsActive           bool
	CoreRuleID         *uuid.UUID
	DatasourceID       *uuid.UUID
}

func (n *RuleNode) ID() string {
	if n == nil {
		return ""
	}
	if n.Condition != nil && n.Condition.ID != "" {
		return "cond:" + n.Condition.ID
	}
	if n.Group != nil && n.Group.ID != "" {
		return "grp:" + n.Group.ID
	}
	return ""
}
