package rules

import (
	"testing"

	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
)

type Fixture struct {
	AST        *RuleNode
	Compiled   *vm.CompiledProgram
	Syms       *vm.SymbolDict
	Enums      *vm.EnumDict
	MapInput   map[string]any
	FastRecord *vm.FastRecord
}

func BuildFixture(tb testing.TB, size string) *Fixture {
	syms := vm.NewSymbolDict()
	enums := vm.NewEnumDict()

	var ast *RuleNode
	input := map[string]any{
		"customer": map[string]any{
			"tier":    "GOLD",
			"balance": float64(15000),
		},
	}

	switch size {
	case "small":
		ast = &RuleNode{
			Type:      NodeTypeCondition,
			Condition: &RuleCondition{FieldPath: "customer.balance", Operator: ">", Value: float64(10000), ValueType: "number"},
		}
	case "medium":
		ast = &RuleNode{
			Type: NodeTypeGroup,
			Group: &RuleGroup{
				Operator: "AND",
				Conditions: []RuleNode{
					{Type: NodeTypeCondition, Condition: &RuleCondition{FieldPath: "customer.tier", Operator: "==", Value: "GOLD", ValueType: "string"}},
					{Type: NodeTypeCondition, Condition: &RuleCondition{FieldPath: "customer.balance", Operator: ">", Value: float64(10000), ValueType: "number"}},
				},
			},
		}
	case "large":
		group := &RuleGroup{Operator: "AND", Conditions: make([]RuleNode, 0, 50)}
		for i := 0; i < 50; i++ {
			group.Conditions = append(group.Conditions, RuleNode{
				Type:      NodeTypeCondition,
				Condition: &RuleCondition{FieldPath: "customer.balance", Operator: ">", Value: float64(i), ValueType: "number"},
			})
		}
		ast = &RuleNode{Type: NodeTypeGroup, Group: group}
	default:
		tb.Fatalf("unknown size %s", size)
	}

	// Compile FIRST — this populates the dicts. Then freeze for hot-path lookups.
	res := CompileVM(ast, syms, enums)
	if res.Unsupported != nil {
		tb.Fatalf("compile failed: %v", res.Unsupported)
	}
	syms.Freeze()
	enums.Freeze()

	return &Fixture{
		AST:        ast,
		Compiled:   res.Program,
		Syms:       syms,
		Enums:      enums,
		MapInput:   input,
		FastRecord: vm.Project(input, syms, enums),
	}
}