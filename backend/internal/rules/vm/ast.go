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

func (*BinaryExpr) exprNode() {}
func (*FieldRef) exprNode()   {}
func (*Literal) exprNode()    {}

type Expression struct {
	Root ExprNode
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
