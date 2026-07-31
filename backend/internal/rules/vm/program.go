package vm

// CompiledProgram is the output of Compile(). It is immutable and
// safe for concurrent reads — no internal locking required.
type CompiledProgram struct {
	Insts         []Instruction
	NumConsts     []int64
	StrConsts     []string
	BoolConsts    []bool
	EnumConsts    []uint32
	NumInSet      [][]int64

	NumPeakDepth  uint8
	BoolPeakDepth uint8
}

// CompileResult tells the caller whether the VM path is usable.
// If Unsupported is non-nil, the engine must use the recursive fallback.
//
// Unsupported is a plain error to keep vm package dependency-free.
// The richer *CompileError (with Node + Operator fields) is defined in
// the rules package where the AST types live.
type CompileResult struct {
	Program     *CompiledProgram
	Unsupported error
}