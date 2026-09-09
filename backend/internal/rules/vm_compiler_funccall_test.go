package rules

import (
	"strings"
	"testing"

	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
)

// TestCompileVM_FuncCall_RejectsLoudly proves that FuncCall (added this
// session for calc-term SQL pushdown and browser/native evaluation) is
// rejected by the bytecode compiler with a specific, named error - not
// silently compiled into an empty/partial program - until step 6
// (FuncCall bytecode dispatch) lands. This is the exact failure mode that
// bit the browser WASM path before this session's fix; asserting it here
// keeps it from silently reappearing one layer up.
func TestCompileVM_FuncCall_RejectsLoudly(t *testing.T) {
	syms := vm.NewSymbolDict()
	enums := vm.NewEnumDict()
	syms.Intern("cash_flows")
	syms.Freeze()

	node := &RuleNode{
		Type: vm.NodeTypeExpression,
		Expression: &vm.Expression{
			Root: &vm.FuncCall{
				Name: "SUM",
				Args: []vm.ExprNode{&vm.FieldRef{Path: "cash_flows"}},
			},
		},
	}

	res := CompileVM(node, syms, enums)

	if res.Unsupported == nil {
		t.Fatalf("expected Unsupported to be set for FuncCall, got nil (res.Program=%+v)", res.Program)
	}
	if res.Program != nil {
		t.Errorf("expected no Program to be emitted alongside Unsupported, got %+v", res.Program)
	}
	if !strings.Contains(res.Unsupported.Error(), "FuncCall") {
		t.Errorf("expected error to name FuncCall specifically, got: %s", res.Unsupported.Error())
	}
}
