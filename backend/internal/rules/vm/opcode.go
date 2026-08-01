package vm

// OpCode is 1 byte; total Instruction size is 8 bytes (verified by TestInstructionSize).
type OpCode uint8

const (
	// Loads (push 1 value to operand stack)
	OpLoadSymbolNum OpCode = iota // NumVals[SymID]
	OpLoadSymbolStr               // StrVals[SymID]
	OpLoadSymbolBool              // BoolVals[SymID]
	OpLoadSymbolEnum              // EnumVals[SymID]   (push uint32 onto num stack)
	OpLoadConstNum                // NumConsts[Aux]
	OpLoadConstStr                // StrConsts[Aux]
	OpLoadConstBool               // BoolConsts[Aux]
	OpLoadConstEnum               // EnumConsts[Aux]   (push uint32 onto num stack)

	// Presence tests (push 1 bool)
	OpIsNull    // Present[SymID] == 0
	OpIsNotNull // Present[SymID] != 0

	// Numeric / enum compares (pop 2 nums, push 1 bool)
	OpEqualNum
	OpNotEqualNum
	OpGreaterNum
	OpLessNum
	OpGreaterEqNum
	OpLessEqNum
	OpEqualEnum
	OpNotEqualEnum

	// String compares (pop 2 strs, push 1 bool)
	OpEqualStr
	OpNotEqualStr

	// Bool compares (pop 2 bools, push 1 bool)
	OpEqualBool
	OpNotEqualBool

	// Set membership: pop 1 num, scan int64 slice in Aux, push bool
	OpInNum

	// Logical (pop bools, push bool)
	OpAnd
	OpOr
	OpNot

	// Short-circuit jumps (pop 1 bool; if condition met, i += int16(Aux))
	OpJumpIfFalse
	OpJumpIfTrue
	OpJump // unconditional relative jump (pops nothing)

	// Float loads (push 1 float64 to float operand stack)
	OpLoadSymbolFNum // FNumVals[SymID]
	OpLoadConstFNum  // FNumConsts[Aux]

	// Float arithmetic (pop 2 floats, push 1 float result)
	OpAddFNum // push a + b
	OpSubFNum // push a - b
	OpMulFNum // push a * b
	OpDivFNum // push a / b
	OpAbsFNum // pop 1, push |a|

	// Float comparisons (pop 2 floats, push 1 bool)
	OpEqualFNum
	OpNotEqualFNum
	OpGreaterFNum
	OpLessFNum
	OpGreaterEqFNum
	OpLessEqFNum

	// Result terminator
	OpReturnBool // pop 1 bool; halt VM with that result
)

// Instruction is exactly 8 bytes, packed for one 64-byte cache line
// (8 instructions per line). Field order chosen so Go does NOT add
// trailing alignment padding:
//
//	SymbolID uint32  // 4 bytes — offsets 0-3 (4-byte aligned)
//	Aux      uint16  // 2 bytes — offsets 4-5
//	Op       OpCode  // 1 byte  — offset 6
//	_pad     [1]byte // 1 byte  — offset 7 (compiler would otherwise pad here)
//
// Aux is uint16 in memory; compiler emits signed jump offsets via
// uint16(int16(x)), VM reads them back via int(int16(inst.Aux)).
type Instruction struct {
	SymbolID uint32 // 4 bytes — index into FastRecord slices
	Aux      uint16 // 2 bytes — dual-purpose: ConstIdx OR JumpDst
	Op       OpCode // 1 byte  — which operation
	_pad     [1]byte
}