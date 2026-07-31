package rulefabric

import (
	"encoding/json"
	"fmt"
	"strings"

	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
)

// CompileError reports why a rulefabric rule could not be lowered to VM
// bytecode. The wiring layer uses this to fall back to the recursive
// evaluator on the same goroutine — no error propagates to the caller
// unless something catastrophic happens (e.g., json.Unmarshal failure).
type CompileError struct {
	RuleID string
	Reason string
}

func (e *CompileError) Error() string {
	return fmt.Sprintf("rule %s: %s", e.RuleID, e.Reason)
}

// rfVMCompiler walks a rulefabric condition tree and emits a flat
// vm.CompiledProgram. The condition tree uses polymorphic JSON shapes
// (Conditions is []interface{} of either map[string]interface{} groups
// or leaves), so the compiler must type-switch at each node.
type rfVMCompiler struct {
	ruleID string
	syms   *vm.SymbolDict
	enums  *vm.EnumDict
	out    vm.CompiledProgram
	err    *CompileError

	numDepth  uint8
	boolDepth uint8
	numMax    uint8
	boolMax   uint8
}

// CompileRuleFabric is the entry point for compiling a rulefabric rule
// to VM bytecode. Caller must pre-populate the dictionaries (RegisterAndFreeze)
// OR rely on the Compile→Resolve fallback when the dict is already frozen.
//
// On failure, returns CompileResult with Unsupported set to a *CompileError.
// On success, returns a fully-resolved CompiledProgram.
func CompileRuleFabric(rule *RuleWithLogic, root *ConditionGroup, syms *vm.SymbolDict, enums *vm.EnumDict) vm.CompileResult {
	ruleID := rule.ID.String()
	c := &rfVMCompiler{ruleID: ruleID, syms: syms, enums: enums}

	// Scoring formulas are evaluated via CEL separately from the boolean
	// gate; we cannot inline them into the VM v1 bytecode. Surface this
	// as Unsupported so the caller can run the score after VM evaluation.
	if rule.Logic.ScoringFormula != "" {
		c.fail("rule has CEL scoring formula (unsupported in VM v1)")
		return vm.CompileResult{Unsupported: c.err}
	}

	if root == nil {
		c.fail("rule has nil ConditionGroup")
		return vm.CompileResult{Unsupported: c.err}
	}

	c.compileGroup(root)

	if c.err != nil {
		return vm.CompileResult{Unsupported: c.err}
	}

	c.emit(vm.OpReturnBool, 0, 0)
	c.out.NumPeakDepth = c.numMax
	c.out.BoolPeakDepth = c.boolMax

	return vm.CompileResult{Program: &c.out}
}

// CompileRuleFabricFromJSON compiles a rule directly from its
// JSON-serialized ConditionGroup, skipping the intermediate
// unmarshal-to-ConditionGroup round-trip the legacy path uses.
// Caller owns the resulting CompiledProgram's lifetime.
func CompileRuleFabricFromJSON(ruleID string, conditionJSON json.RawMessage, syms *vm.SymbolDict, enums *vm.EnumDict) (vm.CompileResult, error) {
	c := &rfVMCompiler{ruleID: ruleID, syms: syms, enums: enums}

	var group ConditionGroup
	if err := json.Unmarshal(conditionJSON, &group); err != nil {
		return vm.CompileResult{}, fmt.Errorf("parse condition: %w", err)
	}

	c.compileGroup(&group)
	if c.err != nil {
		return vm.CompileResult{Unsupported: c.err}, nil
	}

	c.emit(vm.OpReturnBool, 0, 0)
	c.out.NumPeakDepth = c.numMax
	c.out.BoolPeakDepth = c.boolMax

	return vm.CompileResult{Program: &c.out}, nil
}

func (c *rfVMCompiler) fail(reason string) {
	if c.err == nil {
		c.err = &CompileError{RuleID: c.ruleID, Reason: reason}
	}
}

func (c *rfVMCompiler) emit(op vm.OpCode, symID uint32, aux uint16) int {
	pc := len(c.out.Insts)
	c.out.Insts = append(c.out.Insts, vm.Instruction{Op: op, SymbolID: symID, Aux: aux})
	c.updateDepth(op)
	return pc
}

func (c *rfVMCompiler) updateDepth(op vm.OpCode) {
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

	// Numeric/enum compare: pop 2 nums, push 1 bool
	case vm.OpEqualNum, vm.OpNotEqualNum, vm.OpGreaterNum, vm.OpLessNum, vm.OpGreaterEqNum,
		vm.OpLessEqNum, vm.OpEqualEnum, vm.OpNotEqualEnum, vm.OpInNum:
		c.numDepth -= 2
		c.boolDepth++
		if c.boolDepth > c.boolMax {
			c.boolMax = c.boolDepth
		}

	// String compare: pop 2 strs, push 1 bool
	case vm.OpEqualStr, vm.OpNotEqualStr:
		c.boolDepth++
		if c.boolDepth > c.boolMax {
			c.boolMax = c.boolDepth
		}

	// Bool compare: pop 2 bools, push 1 bool
	case vm.OpEqualBool, vm.OpNotEqualBool:
		c.boolDepth--

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

// compileGroup processes a ConditionGroup (AND/OR/NOT).
// Conditions is []interface{} where each element is either a
// map[string]interface{} (sub-group or leaf) — we type-switch.
func (c *rfVMCompiler) compileGroup(g *ConditionGroup) {
	if g == nil {
		return
	}
	op := strings.ToUpper(g.Operator)

	if len(g.Conditions) == 0 {
		// Empty group semantics: AND -> true, OR -> false.
		switch op {
		case "AND":
			c.emit(vm.OpLoadConstBool, 0, c.addBoolConst(true))
		case "OR":
			c.emit(vm.OpLoadConstBool, 0, c.addBoolConst(false))
		default:
			// Unknown operator with no children — surface as error.
			c.fail("empty group with unsupported operator: " + op)
		}
		return
	}

	switch op {
	case "AND":
		failJumps := []int{}
		for i := range g.Conditions {
			c.compileNode(g.Conditions[i])
			failJumps = append(failJumps, c.emit(vm.OpJumpIfFalse, 0, 0))
		}

		trueIdx := c.addBoolConst(true)
		c.emit(vm.OpLoadConstBool, 0, trueIdx)
		endJmp := c.emit(vm.OpJump, 0, 0)

		falsePC := len(c.out.Insts)
		c.emit(vm.OpLoadConstBool, 0, c.addBoolConst(false))

		for _, pc := range failJumps {
			c.out.Insts[pc].Aux = uint16(int16(falsePC - pc - 1))
		}
		c.out.Insts[endJmp].Aux = uint16(int16(len(c.out.Insts) - endJmp - 1))

	case "OR":
		successJumps := []int{}
		for i := range g.Conditions {
			c.compileNode(g.Conditions[i])
			successJumps = append(successJumps, c.emit(vm.OpJumpIfTrue, 0, 0))
		}

		c.emit(vm.OpLoadConstBool, 0, c.addBoolConst(false))
		endJmp := c.emit(vm.OpJump, 0, 0)

		successPC := len(c.out.Insts)
		c.emit(vm.OpLoadConstBool, 0, c.addBoolConst(true))

		for _, pc := range successJumps {
			c.out.Insts[pc].Aux = uint16(int16(successPC - pc - 1))
		}
		c.out.Insts[endJmp].Aux = uint16(int16(len(c.out.Insts) - endJmp - 1))

	case "NOT":
		// NOT applies to the first child. If the group has more than
		// one child, the legacy evaluator treats them as an implicit
		// AND — match that semantic.
		if len(g.Conditions) == 0 {
			c.fail("NOT group has no children")
			return
		}
		if len(g.Conditions) == 1 {
			c.compileNode(g.Conditions[0])
			c.emit(vm.OpNot, 0, 0)
			return
		}
		// Implicit AND over multiple children, then NOT.
		failJumps := []int{}
		for i := range g.Conditions {
			c.compileNode(g.Conditions[i])
			failJumps = append(failJumps, c.emit(vm.OpJumpIfFalse, 0, 0))
		}
		c.emit(vm.OpLoadConstBool, 0, c.addBoolConst(true))
		endJmp := c.emit(vm.OpJump, 0, 0)
		falsePC := len(c.out.Insts)
		c.emit(vm.OpLoadConstBool, 0, c.addBoolConst(false))
		for _, pc := range failJumps {
			c.out.Insts[pc].Aux = uint16(int16(falsePC - pc - 1))
		}
		c.out.Insts[endJmp].Aux = uint16(int16(len(c.out.Insts) - endJmp - 1))
		c.emit(vm.OpNot, 0, 0)

	default:
		c.fail("unsupported group operator: " + op)
	}
}

// compileNode dispatches on the runtime type of an element of
// g.Conditions. Each element is map[string]interface{} (from JSON
// unmarshal); the "type" field tells us whether it's a sub-group
// or a leaf condition.
func (c *rfVMCompiler) compileNode(elem interface{}) {
	if c.err != nil {
		return
	}
	m, ok := elem.(map[string]interface{})
	if !ok {
		c.fail("condition element is not a JSON object")
		return
	}

	condType, _ := m["type"].(string)
	switch condType {
	case "group":
		var sub ConditionGroup
		// Cheap re-encode/decode; cheaper than walking the map directly
		// for nested conditions because it avoids 12 type assertions.
		buf, _ := json.Marshal(m)
		if err := json.Unmarshal(buf, &sub); err != nil {
			c.fail("parse sub-group: " + err.Error())
			return
		}
		c.compileGroup(&sub)

	case "condition":
		var cond Condition
		buf, _ := json.Marshal(m)
		if err := json.Unmarshal(buf, &cond); err != nil {
			c.fail("parse condition: " + err.Error())
			return
		}
		c.compileCondition(&cond)

	default:
		c.fail("unknown condition node type: " + condType)
	}
}

// compileCondition lowers a single leaf condition.
//
// Operator mapping per the v1 subset:
//
//	eq/ne/gt/lt/gte/lte     → numeric compares
//	in                     → numeric set membership (≤256 elements)
//	eq/ne (enum string)     → enum compares
//	eq/ne (plain string)    → string compares (Go == is memcmp, zero-alloc)
//	is_null / is_not_null   → presence checks
//
// Everything else (regex, contains, between, date_*, starts_with,
// ends_with, etc.) is unsupported → CompileError → fallback.
func (c *rfVMCompiler) compileCondition(cond *Condition) {
	if cond == nil {
		c.fail("nil condition")
		return
	}

	// FieldPath can be cross-entity (e.g., "data.customer.tier" or
	// "related.order.amount"). v1 supports flat field paths; we use
	// cond.Field directly. If EntityPath is set and Field references
	// a related entity, the caller's FastRecord must include the
	// "related." prefix — see rulefabric Pre-populateRelated for
	// the projection contract.
	path := cond.Field
	if cond.EntityPath != nil && cond.EntityPath.Field != "" {
		path = cond.EntityPath.Field
	}
	if path == "" {
		c.fail("condition has empty field path")
		return
	}

	symID, err := c.syms.Intern(path)
	if err != nil {
		// Dict may be frozen (RegisterAndFreeze). Fall back to Resolve.
		if id, ok := c.syms.Resolve(path); ok {
			symID = id
		} else {
			c.fail(fmt.Sprintf("field %q not registered", path))
			return
		}
	}

	op := strings.ToLower(cond.Operator)

	// Presence checks first (no value comparison).
	switch op {
	case "is_null", "isnull":
		c.emit(vm.OpIsNull, symID, 0)
		return
	case "is_not_null", "isnotnull":
		c.emit(vm.OpIsNotNull, symID, 0)
		return
	}

	// Unsupported operators — fail closed (caller falls back to recursive).
	switch op {
	case "regex", "matches_regex":
		c.fail("regex operator unsupported in VM v1")
		return
	case "contains", "not_contains":
		c.fail("contains operator unsupported in VM v1")
		return
	case "starts_with", "ends_with":
		c.fail(op + " operator unsupported in VM v1 (Phase 9)")
		return
	case "between":
		c.fail("between operator unsupported in VM v1")
		return
	case "date_before", "date_after", "days_ago_less_than":
		c.fail("date operator unsupported in VM v1 (Phase 9)")
		return
	}

	// Type-aware comparison dispatch.
	switch v := cond.Value.(type) {
	case float64, int, int64:
		c.compileNumCondition(op, symID, cond)
	case string:
		c.compileStrCondition(op, symID, cond)
	case bool:
		c.compileBoolCondition(op, symID, cond)
	default:
		c.fail(fmt.Sprintf("unsupported value type %T for field %q", v, path))
	}
}

func (c *rfVMCompiler) compileNumCondition(op string, symID uint32, cond *Condition) {
	c.emit(vm.OpLoadSymbolNum, symID, 0)

	if op == "in" {
		arr, ok := cond.Value.([]interface{})
		if !ok {
			c.fail("'in' value must be an array")
			return
		}
		if len(arr) > 256 {
			c.fail("'in' array exceeds 256 elements")
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
				c.fail(fmt.Sprintf("'in' element type %T not supported", v))
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
		c.fail(fmt.Sprintf("invalid number type %T", cond.Value))
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
	case ">=", "gte", "greater_than_or_equals", "greater_equal":
		c.emit(vm.OpGreaterEqNum, 0, 0)
	case "<=", "lte", "less_than_or_equals", "less_equal":
		c.emit(vm.OpLessEqNum, 0, 0)
	default:
		c.fail("unsupported numeric operator: " + op)
	}
}

func (c *rfVMCompiler) compileStrCondition(op string, symID uint32, cond *Condition) {
	strVal, ok := cond.Value.(string)
	if !ok {
		c.fail(fmt.Sprintf("expected string value, got %T", cond.Value))
		return
	}

	// Try enum compare first (interned categorical string).
	if enumID, ok := c.enums.ID(strVal); ok {
		c.emit(vm.OpLoadSymbolEnum, symID, 0)
		c.emit(vm.OpLoadConstEnum, 0, c.addEnumConst(enumID))

		switch op {
		case "==", "eq", "equals":
			c.emit(vm.OpEqualEnum, 0, 0)
		case "!=", "ne", "not_equals":
			c.emit(vm.OpNotEqualEnum, 0, 0)
		default:
			c.fail("unsupported enum string operator: " + op)
		}
		return
	}

	// Plain string compare (Go == is length+pointer+memcmp; zero-alloc).
	c.emit(vm.OpLoadSymbolStr, symID, 0)
	c.emit(vm.OpLoadConstStr, 0, c.addStrConst(strVal))

	switch op {
	case "==", "eq", "equals":
		c.emit(vm.OpEqualStr, 0, 0)
	case "!=", "ne", "not_equals":
		c.emit(vm.OpNotEqualStr, 0, 0)
	default:
		c.fail("unsupported string operator: " + op)
	}
}

func (c *rfVMCompiler) compileBoolCondition(op string, symID uint32, cond *Condition) {
	boolVal, ok := cond.Value.(bool)
	if !ok {
		c.fail(fmt.Sprintf("expected bool value, got %T", cond.Value))
		return
	}

	c.emit(vm.OpLoadSymbolBool, symID, 0)
	c.emit(vm.OpLoadConstBool, 0, c.addBoolConst(boolVal))

	switch op {
	case "==", "eq", "equals":
		c.emit(vm.OpEqualBool, 0, 0)
	case "!=", "ne", "not_equals":
		c.emit(vm.OpNotEqualBool, 0, 0)
	default:
		c.fail("unsupported bool operator: " + op)
	}
}

func (c *rfVMCompiler) addNumConst(val int64) uint16 {
	idx := uint16(len(c.out.NumConsts))
	c.out.NumConsts = append(c.out.NumConsts, val)
	return idx
}

func (c *rfVMCompiler) addStrConst(val string) uint16 {
	idx := uint16(len(c.out.StrConsts))
	c.out.StrConsts = append(c.out.StrConsts, val)
	return idx
}

func (c *rfVMCompiler) addBoolConst(val bool) uint16 {
	idx := uint16(len(c.out.BoolConsts))
	c.out.BoolConsts = append(c.out.BoolConsts, val)
	return idx
}

func (c *rfVMCompiler) addEnumConst(val uint32) uint16 {
	idx := uint16(len(c.out.EnumConsts))
	c.out.EnumConsts = append(c.out.EnumConsts, val)
	return idx
}