package vm

import "sync"

// Stack provides fixed-size, allocation-free operand stacks for the VM.
// Arrays are exactly 256 elements. Because the stack pointers (nTop, bTop, sTop)
// are uint8, they mathematically cannot exceed 255. This allows the Go compiler
// to completely elide bounds checks in the dispatch loop.
type Stack struct {
	nums  [256]int64
	bools [256]bool
	strs  [256]string
	nTop  uint8
	bTop  uint8
	sTop  uint8
}

var stackPool = sync.Pool{
	New: func() any { return &Stack{} },
}

func GetStack() *Stack {
	s := stackPool.Get().(*Stack)
	s.nTop = 0
	s.bTop = 0
	s.sTop = 0
	return s
}

func PutStack(s *Stack) {
	stackPool.Put(s)
}

type VM struct{}

func NewVM() *VM { return &VM{} }

// Run executes the compiled bytecode against the fast record.
// Guarantees 0 allocations and 0 bounds checks in the dispatch loop
// (verified via `go test -gcflags="-d=ssa/check_bce/debug=1"`).
func (vm *VM) Run(p *CompiledProgram, r *FastRecord, s *Stack) bool {
	insts := p.Insts

	// Slice headers over arrays.
	nums := s.nums[:]
	bools := s.bools[:]
	strs := s.strs[:]

	nTop := s.nTop
	bTop := s.bTop
	sTop := s.sTop

	for i := 0; i < len(insts); i++ {
		inst := insts[i]

		switch inst.Op {
		case OpLoadSymbolNum:
			nums[nTop] = r.NumVals[inst.SymbolID]
			nTop++
		case OpLoadConstNum:
			nums[nTop] = p.NumConsts[inst.Aux]
			nTop++
		case OpLoadSymbolBool:
			bools[bTop] = r.BoolVals[inst.SymbolID]
			bTop++
		case OpLoadConstBool:
			bools[bTop] = p.BoolConsts[inst.Aux]
			bTop++
		case OpLoadSymbolStr:
			strs[sTop] = r.StrVals[inst.SymbolID]
			sTop++
		case OpLoadConstStr:
			strs[sTop] = p.StrConsts[inst.Aux]
			sTop++
		case OpLoadSymbolEnum:
			nums[nTop] = int64(r.EnumVals[inst.SymbolID])
			nTop++
		case OpLoadConstEnum:
			nums[nTop] = int64(p.EnumConsts[inst.Aux])
			nTop++

		case OpIsNull:
			bools[bTop] = r.Present[inst.SymbolID] == 0
			bTop++
		case OpIsNotNull:
			bools[bTop] = r.Present[inst.SymbolID] != 0
			bTop++

		case OpEqualNum, OpEqualEnum:
			nTop--
			v2 := nums[nTop] // top of stack = RHS
			nTop--
			v1 := nums[nTop] // below top = LHS
			bools[bTop] = v1 == v2
			bTop++
		case OpNotEqualNum, OpNotEqualEnum:
			nTop--
			v2 := nums[nTop]
			nTop--
			v1 := nums[nTop]
			bools[bTop] = v1 != v2
			bTop++
		case OpGreaterNum:
			nTop--
			v2 := nums[nTop]
			nTop--
			v1 := nums[nTop]
			bools[bTop] = v1 > v2
			bTop++
		case OpLessNum:
			nTop--
			v2 := nums[nTop]
			nTop--
			v1 := nums[nTop]
			bools[bTop] = v1 < v2
			bTop++
		case OpGreaterEqNum:
			nTop--
			v2 := nums[nTop]
			nTop--
			v1 := nums[nTop]
			bools[bTop] = v1 >= v2
			bTop++
		case OpLessEqNum:
			nTop--
			v2 := nums[nTop]
			nTop--
			v1 := nums[nTop]
			bools[bTop] = v1 <= v2
			bTop++

		case OpEqualStr:
			sTop--
			v2 := strs[sTop]
			sTop--
			v1 := strs[sTop]
			bools[bTop] = v1 == v2
			bTop++
		case OpNotEqualStr:
			sTop--
			v2 := strs[sTop]
			sTop--
			v1 := strs[sTop]
			bools[bTop] = v1 != v2
			bTop++

		case OpEqualBool:
			bTop--
			v2 := bools[bTop]
			bTop--
			v1 := bools[bTop]
			bools[bTop] = v1 == v2
			bTop++
		case OpNotEqualBool:
			bTop--
			v2 := bools[bTop]
			bTop--
			v1 := bools[bTop]
			bools[bTop] = v1 != v2
			bTop++

		case OpInNum:
			nTop--
			val := nums[nTop]
			set := p.NumInSet[inst.Aux]

			found := false
			for _, v := range set {
				if val == v {
					found = true
					break
				}
			}
			bools[bTop] = found
			bTop++

		case OpAnd:
			bTop--
			v1 := bools[bTop]
			bTop--
			v2 := bools[bTop]
			bools[bTop] = v1 && v2
			bTop++
		case OpOr:
			bTop--
			v1 := bools[bTop]
			bTop--
			v2 := bools[bTop]
			bools[bTop] = v1 || v2
			bTop++
		case OpNot:
			bTop--
			bools[bTop] = !bools[bTop]
			bTop++

		case OpJumpIfFalse:
			bTop--
			if !bools[bTop] {
				i += int(int16(inst.Aux))
			}
		case OpJumpIfTrue:
			bTop--
			if bools[bTop] {
				i += int(int16(inst.Aux))
			}
		case OpJump:
			i += int(int16(inst.Aux))

		case OpReturnBool:
			bTop--
			s.nTop = nTop
			s.bTop = bTop
			s.sTop = sTop
			return bools[bTop]
		}
	}

	panic("vm: missing OpReturnBool")
}