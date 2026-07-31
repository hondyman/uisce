package vm

import (
	"testing"
	"unsafe"
)

// TestInstructionSize asserts the 8-byte packed layout required for
// cache-line density (8 instructions per 64-byte line).
func TestInstructionSize(t *testing.T) {
	if got := unsafe.Sizeof(Instruction{}); got != 8 {
		t.Fatalf("Instruction size = %d, want 8", got)
	}
}

// TestBCE_CleanVMHotLoop is designed to force the compiler to analyze the
// VM.Run switch statement.
//
// To verify bounds checks are eliminated, run:
//   go test -gcflags="-d=ssa/check_bce/debug=1" \
//       -run TestBCE_CleanVMHotLoop -v ./backend/internal/rules/vm/...
//
// Note: Go's BCE pass is conservative for runtime indices over slices,
// so a handful of bounds checks on `inst.SymbolID`/`inst.Aux`-indexed
// accesses are expected. The script accepts that count; the real
// binding contracts (0 allocs/op, <50 ns/op) are enforced by
// TestBenchmarkAcceptance.
func TestBCE_CleanVMHotLoop(t *testing.T) {
	syms := NewSymbolDict()
	_, _ = syms.Intern("a")
	syms.Freeze()

	// Push NumVals[0]=20 (LHS), then NumConsts[0]=10 (RHS).
	// Stack=[20, 10]. OpGreaterNum pops RHS (10) first as v2,
	// then LHS (20) as v1, evaluates v1 > v2 = 20 > 10 = true.
	prog := &CompiledProgram{
		Insts: []Instruction{
			{Op: OpLoadSymbolNum, SymbolID: 0},
			{Op: OpLoadConstNum, Aux: 0},
			{Op: OpGreaterNum},
			{Op: OpReturnBool},
		},
		NumConsts: []int64{10},
	}

	rec := &FastRecord{
		NumVals:  []int64{20},
		BoolVals: []bool{false},
		StrVals:  []string{""},
		EnumVals: []uint32{0},
		Present:  []uint8{HasNum},
	}

	stack := GetStack()
	defer PutStack(stack)

	vm := NewVM()

	for i := 0; i < 1000; i++ {
		if !vm.Run(prog, rec, stack) {
			t.Fatal("expected true (20 > 10)")
		}
	}
}