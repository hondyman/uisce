package rules

import (
	"fmt"
	"strings"

	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
)

// CompileError provides observability into why a rule failed to compile.
// It carries the offending AST node, the operator that triggered the
// failure, and a human-readable reason — surfaced via Snapshot.Fallbacks
// metrics and emitted to logs for post-mortem analysis.
type CompileError struct {
	Node     *RuleNode
	Operator string
	Reason   string
}

func (e *CompileError) Error() string { return e.Reason }

// vmCompiler walks a RuleNode tree once and emits a flat vm.CompiledProgram.
// It is unexported because callers should use CompileVM() rather than
// constructing a vmCompiler directly — that gives the package control
// over emit order and stack-depth tracking invariants.
type vmCompiler struct {
	syms   *vm.SymbolDict
	enums  *vm.EnumDict
	out    vm.CompiledProgram
	err    *CompileError

	numDepth   uint8
	boolDepth uint8
	fNumDepth  uint8
	numMax     uint8
	boolMax    uint8
	fNumMax    uint8
}

// CompileVM is the entry point for compiling rules to VM bytecode.
//
// Pre: syms and enums must already be populated (or frozen — see CompileVM's
//      Intern→Resolve fallback below). Either intern all field paths at
//      startup (RegisterAndFreeze) or call CompileVM on each unique rule
//      before any concurrent traffic begins.
//
// Post: on success Program is fully resolved. On failure CompileResult.Unsupported
//       is set and tells the caller to use the recursive path.
func CompileVM(node *RuleNode, syms *vm.SymbolDict, enums *vm.EnumDict) vm.CompileResult {
	c := &vmCompiler{syms: syms, enums: enums}
	c.compileNode(node)

	if c.err != nil {
		return vm.CompileResult{Unsupported: c.err}
	}

	c.emit(vm.OpReturnBool, 0, 0)
	c.out.NumPeakDepth = c.numMax
	c.out.BoolPeakDepth = c.boolMax
	c.out.FNumPeakDepth = c.fNumMax

	return vm.CompileResult{Program: &c.out}
}

func (c *vmCompiler) fail(node *RuleNode, op string, reason string) {
	if c.err == nil {
		c.err = &CompileError{Node: node, Operator: op, Reason: reason}
	}
}

func (c *vmCompiler) emit(op vm.OpCode, symID uint32, aux uint16) int {
	pc := len(c.out.Insts)
	c.out.Insts = append(c.out.Insts, vm.Instruction{Op: op, SymbolID: symID, Aux: aux})
	c.updateDepth(op)
	return pc
}

func (c *vmCompiler) updateDepth(op vm.OpCode) {
	switch op {
	case vm.OpLoadSymbolNum, vm.OpLoadConstNum, vm.OpLoadSymbolStr, vm.OpLoadConstStr,
		vm.OpLoadSymbolEnum, vm.OpLoadConstEnum:
		c.numDepth++
		if c.numDepth > c.numMax {
			c.numMax = c.numDepth
		}
	case vm.OpLoadSymbolBool, vm.OpLoadConstBool, vm.OpIsNull, vm.OpIsNotNull:
		c.boolDepth++
		if c.boolDepth > c.boolMax {
			c.boolMax = c.boolDepth
		}
	case vm.OpLoadSymbolFNum, vm.OpLoadConstFNum:
		c.fNumDepth++
		if c.fNumDepth > c.fNumMax {
			c.fNumMax = c.fNumDepth
		}

	case vm.OpEqualNum, vm.OpNotEqualNum, vm.OpGreaterNum, vm.OpLessNum, vm.OpGreaterEqNum,
		vm.OpLessEqNum, vm.OpEqualEnum, vm.OpNotEqualEnum, vm.OpInNum:
		c.numDepth -= 2
		c.boolDepth++
		if c.boolDepth > c.boolMax {
			c.boolMax = c.boolDepth
		}

	case vm.OpEqualStr, vm.OpNotEqualStr:
		c.boolDepth++
		if c.boolDepth > c.boolMax {
			c.boolMax = c.boolDepth
		}

	case vm.OpEqualBool, vm.OpNotEqualBool:
		c.boolDepth--

	case vm.OpAddFNum, vm.OpSubFNum, vm.OpMulFNum, vm.OpDivFNum:
		c.fNumDepth -= 2
		c.fNumDepth++
		if c.fNumDepth > c.fNumMax {
			c.fNumMax = c.fNumDepth
		}
	case vm.OpAbsFNum:
		// net 0 (pop 1, push 1)

	case vm.OpEqualFNum, vm.OpNotEqualFNum, vm.OpGreaterFNum, vm.OpLessFNum,
		vm.OpGreaterEqFNum, vm.OpLessEqFNum:
		c.fNumDepth -= 2
		c.boolDepth++
		if c.boolDepth > c.boolMax {
			c.boolMax = c.boolDepth
		}

	case vm.OpAnd, vm.OpOr:
		c.boolDepth--
	case vm.OpNot:
		// net 0
	case vm.OpJumpIfFalse, vm.OpJumpIfTrue:
		c.boolDepth--
	case vm.OpJump, vm.OpReturnBool:
		// net 0
	}
}

func (c *vmCompiler) compileNode(n *RuleNode) {
	if c.err != nil {
		return
	}
	switch n.Type {
	case NodeTypeGroup:
		c.compileGroup(n.Group)
	case NodeTypeCondition:
		c.compileCondition(n.Condition)
	case vm.NodeTypeExpression:
		c.compileExpression(n.Expression)
	default:
		c.fail(n, "", fmt.Sprintf("unknown rule node type: %s", n.Type))
	}
}

func (c *vmCompiler) compileGroup(g *RuleGroup) {
	if g == nil {
		c.fail(nil, "", "group node has nil Group field")
		return
	}
	op := strings.ToUpper(g.Operator)

	if len(g.Conditions) == 0 {
		if op == "AND" {
			idx := c.addBoolConst(true)
			c.emit(vm.OpLoadConstBool, 0, idx)
		} else if op == "OR" {
			idx := c.addBoolConst(false)
			c.emit(vm.OpLoadConstBool, 0, idx)
		} else {
			c.fail(nil, op, "empty group without AND/OR operator")
		}
		return
	}

	switch op {
	case "AND":
		failJumps := []int{}
		for i := range g.Conditions {
			c.compileNode(&g.Conditions[i])
			jmpPC := c.emit(vm.OpJumpIfFalse, 0, 0)
			failJumps = append(failJumps, jmpPC)
		}

		trueIdx := c.addBoolConst(true)
		c.emit(vm.OpLoadConstBool, 0, trueIdx)
		endJmpPC := c.emit(vm.OpJump, 0, 0)

		falsePC := len(c.out.Insts)
		falseIdx := c.addBoolConst(false)
		c.emit(vm.OpLoadConstBool, 0, falseIdx)

		for _, pc := range failJumps {
			c.out.Insts[pc].Aux = uint16(int16(falsePC - pc - 1))
		}
		c.out.Insts[endJmpPC].Aux = uint16(int16(len(c.out.Insts) - endJmpPC - 1))

	case "OR":
		successJumps := []int{}
		for i := range g.Conditions {
			c.compileNode(&g.Conditions[i])
			jmpPC := c.emit(vm.OpJumpIfTrue, 0, 0)
			successJumps = append(successJumps, jmpPC)
		}

		falseIdx := c.addBoolConst(false)
		c.emit(vm.OpLoadConstBool, 0, falseIdx)
		endJmpPC := c.emit(vm.OpJump, 0, 0)

		successPC := len(c.out.Insts)
		trueIdx := c.addBoolConst(true)
		c.emit(vm.OpLoadConstBool, 0, trueIdx)

		for _, pc := range successJumps {
			c.out.Insts[pc].Aux = uint16(int16(successPC - pc - 1))
		}
		c.out.Insts[endJmpPC].Aux = uint16(int16(len(c.out.Insts) - endJmpPC - 1))

	case "NOT":
		c.compileNode(&g.Conditions[0])
		c.emit(vm.OpNot, 0, 0)

	default:
		c.fail(nil, op, fmt.Sprintf("unsupported group operator: %s", op))
	}
}

func (c *vmCompiler) compileCondition(cond *RuleCondition) {
	if cond == nil {
		c.fail(nil, "", "condition node has nil Condition field")
		return
	}

	path := cond.FieldPath
	if path == "" {
		path = cond.Field
	}

	// Try Intern first (cheap on unfrozen dict). If the dict is frozen
	// (e.g., warmed via VMManager.RegisterAndFreeze), fall back to Resolve.
	// A path that's both frozen AND unknown means the rule references a
	// field that wasn't in the warming corpus — surface as Unsupported.
	symID, err := c.syms.Intern(path)
	if err != nil {
		if id, ok := c.syms.Resolve(path); ok {
			symID = id
		} else {
			c.fail(nil, cond.Operator, fmt.Sprintf("symbol not registered for %q: %v", path, err))
			return
		}
	}

	op := strings.ToLower(cond.Operator)

	switch op {
	case "is_null", "isnull", "null":
		c.emit(vm.OpIsNull, symID, 0)
		return
	case "is_not_null", "isnotnull", "notnull":
		c.emit(vm.OpIsNotNull, symID, 0)
		return
	}

	switch cond.ValueType {
	case "number", "int", "float", "integer":
		c.compileNumCondition(op, symID, cond)
	case "string", "text":
		c.compileStrCondition(op, symID, cond)
	case "boolean", "bool":
		c.compileBoolCondition(op, symID, cond)
	default:
		switch cond.Value.(type) {
		case float64, int, int64:
			c.compileNumCondition(op, symID, cond)
		case string:
			c.compileStrCondition(op, symID, cond)
		case bool:
			c.compileBoolCondition(op, symID, cond)
		default:
			c.fail(nil, op, fmt.Sprintf("unsupported value type: %s", cond.ValueType))
		}
	}
}

func (c *vmCompiler) compileNumCondition(op string, symID uint32, cond *RuleCondition) {
	c.emit(vm.OpLoadSymbolNum, symID, 0)

	if op == "in" {
		arr, ok := cond.Value.([]any)
		if !ok {
			c.fail(nil, op, "'in' operator value is not an array")
			return
		}
		if len(arr) > 256 {
			c.fail(nil, op, "'in' operator array exceeds 256 elements")
			return
		}
		set := make([]int64, len(arr))
		for i, v := range arr {
			switch n := v.(type) {
			case float64:
				set[i] = int64(n)
			case int:
				set[i] = int64(n)
			case int64:
				set[i] = n
			default:
				c.fail(nil, op, fmt.Sprintf("unsupported number type in 'in' array: %T", v))
				return
			}
		}
		setIdx := uint16(len(c.out.NumInSet))
		c.out.NumInSet = append(c.out.NumInSet, set)
		c.emit(vm.OpInNum, 0, setIdx)
		return
	}

	var numVal int64
	switch n := cond.Value.(type) {
	case float64:
		numVal = int64(n)
	case int:
		numVal = int64(n)
	case int64:
		numVal = n
	default:
		c.fail(nil, op, fmt.Sprintf("invalid number value type: %T", cond.Value))
		return
	}

	constIdx := c.addNumConst(numVal)
	c.emit(vm.OpLoadConstNum, 0, constIdx)

	switch op {
	case "==", "eq", "equals":
		c.emit(vm.OpEqualNum, 0, 0)
	case "!=", "ne", "not_equals":
		c.emit(vm.OpNotEqualNum, 0, 0)
	case ">", "gt", "greater_than":
		c.emit(vm.OpGreaterNum, 0, 0)
	case "<", "lt", "less_than":
		c.emit(vm.OpLessNum, 0, 0)
	case ">=", "gte", "greater_equal":
		c.emit(vm.OpGreaterEqNum, 0, 0)
	case "<=", "lte", "less_equal":
		c.emit(vm.OpLessEqNum, 0, 0)
	default:
		c.fail(nil, op, fmt.Sprintf("unsupported numeric operator: %s", op))
	}
}

func (c *vmCompiler) compileStrCondition(op string, symID uint32, cond *RuleCondition) {
	strVal, ok := cond.Value.(string)
	if !ok {
		c.fail(nil, op, fmt.Sprintf("expected string value, got %T", cond.Value))
		return
	}

	if enumID, ok := c.enums.ID(strVal); ok {
		c.emit(vm.OpLoadSymbolEnum, symID, 0)
		constIdx := c.addEnumConst(enumID)
		c.emit(vm.OpLoadConstEnum, 0, constIdx)

		switch op {
		case "==", "eq", "equals":
			c.emit(vm.OpEqualEnum, 0, 0)
		case "!=", "ne", "not_equals":
			c.emit(vm.OpNotEqualEnum, 0, 0)
		default:
			c.fail(nil, op, fmt.Sprintf("unsupported enum string operator: %s", op))
		}
		return
	}

	c.emit(vm.OpLoadSymbolStr, symID, 0)
	constIdx := c.addStrConst(strVal)
	c.emit(vm.OpLoadConstStr, 0, constIdx)

	switch op {
	case "==", "eq", "equals":
		c.emit(vm.OpEqualStr, 0, 0)
	case "!=", "ne", "not_equals":
		c.emit(vm.OpNotEqualStr, 0, 0)
	default:
		c.fail(nil, op, fmt.Sprintf("unsupported string operator: %s", op))
	}
}

func (c *vmCompiler) compileBoolCondition(op string, symID uint32, cond *RuleCondition) {
	boolVal, ok := cond.Value.(bool)
	if !ok {
		c.fail(nil, op, fmt.Sprintf("expected bool value, got %T", cond.Value))
		return
	}

	c.emit(vm.OpLoadSymbolBool, symID, 0)
	constIdx := c.addBoolConst(boolVal)
	c.emit(vm.OpLoadConstBool, 0, constIdx)

	switch op {
	case "==", "eq", "equals":
		c.emit(vm.OpEqualBool, 0, 0)
	case "!=", "ne", "not_equals":
		c.emit(vm.OpNotEqualBool, 0, 0)
	default:
		c.fail(nil, op, fmt.Sprintf("unsupported bool operator: %s", op))
	}
}

func (c *vmCompiler) addNumConst(val int64) uint16 {
	idx := uint16(len(c.out.NumConsts))
	c.out.NumConsts = append(c.out.NumConsts, val)
	return idx
}

func (c *vmCompiler) addStrConst(val string) uint16 {
	idx := uint16(len(c.out.StrConsts))
	c.out.StrConsts = append(c.out.StrConsts, val)
	return idx
}

func (c *vmCompiler) addBoolConst(val bool) uint16 {
	idx := uint16(len(c.out.BoolConsts))
	c.out.BoolConsts = append(c.out.BoolConsts, val)
	return idx
}

func (c *vmCompiler) addEnumConst(val uint32) uint16 {
	idx := uint16(len(c.out.EnumConsts))
	c.out.EnumConsts = append(c.out.EnumConsts, val)
	return idx
}

func (c *vmCompiler) addFNumConst(val float64) uint16 {
	idx := uint16(len(c.out.FNumConsts))
	c.out.FNumConsts = append(c.out.FNumConsts, val)
	return idx
}

func (c *vmCompiler) compileExpression(expr *vm.Expression) {
	if expr == nil || expr.Root == nil {
		c.fail(nil, "expression", "nil expression")
		return
	}
	c.compileExprNode(expr.Root)
}

func (c *vmCompiler) compileExprNode(node vm.ExprNode) {
	switch n := node.(type) {
	case *vm.BinaryExpr:
		c.compileExprNode(n.Left)
		c.compileExprNode(n.Right)
		switch n.Op {
		case "+":
			c.emit(vm.OpAddFNum, 0, 0)
		case "-":
			c.emit(vm.OpSubFNum, 0, 0)
		case "*":
			c.emit(vm.OpMulFNum, 0, 0)
		case "/":
			c.emit(vm.OpDivFNum, 0, 0)
		case "==":
			c.emit(vm.OpEqualFNum, 0, 0)
		case "!=":
			c.emit(vm.OpNotEqualFNum, 0, 0)
		case ">":
			c.emit(vm.OpGreaterFNum, 0, 0)
		case "<":
			c.emit(vm.OpLessFNum, 0, 0)
		case ">=":
			c.emit(vm.OpGreaterEqFNum, 0, 0)
		case "<=":
			c.emit(vm.OpLessEqFNum, 0, 0)
		default:
			c.fail(nil, n.Op, fmt.Sprintf("unsupported expression operator: %s", n.Op))
		}
	case *vm.FieldRef:
		path := n.Path
		symID, err := c.syms.Intern(path)
		if err != nil {
			if id, ok := c.syms.Resolve(path); ok {
				symID = id
			} else {
				c.fail(nil, "field_ref", fmt.Sprintf("symbol not registered for field %q: %v", path, err))
				return
			}
		}
		c.emit(vm.OpLoadSymbolFNum, symID, 0)
	case *vm.Literal:
		idx := c.addFNumConst(n.Value)
		c.emit(vm.OpLoadConstFNum, 0, idx)
	default:
		c.fail(nil, "", fmt.Sprintf("unknown ExprNode type: %T", node))
	}
}