package rules

import (
	"testing"
	"time"

	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
)

func TestBenchmarkAcceptance(t *testing.T) {
	cases := []struct {
		size    string
		maxNsOp float64
	}{
		{"small", 50},
		{"medium", 200},
		{"large", 1500},
	}

	for _, tc := range cases {
		t.Run(tc.size, func(t *testing.T) {
			f := BuildFixture(t, tc.size)
			theVM := vm.NewVM()
			stack := vm.GetStack()
			defer vm.PutStack(stack)

			// Warm up
			for i := 0; i < 1000; i++ {
				theVM.Run(f.Compiled, f.FastRecord, stack)
			}

			// 0 allocs
			allocs := testing.AllocsPerRun(10000, func() {
				theVM.Run(f.Compiled, f.FastRecord, stack)
			})
			if allocs != 0 {
				t.Fatalf("expected 0 allocs/op, got %v", allocs)
			}

			// ns/op ceiling
			const N = 1_000_000
			start := time.Now()
			for i := 0; i < N; i++ {
				theVM.Run(f.Compiled, f.FastRecord, stack)
			}
			nsPerOp := float64(time.Since(start).Nanoseconds()) / float64(N)

			if nsPerOp > tc.maxNsOp {
				t.Fatalf("ns/op %.1f exceeds ceiling %.1f", nsPerOp, tc.maxNsOp)
			}
			t.Logf("size=%s  ns/op=%.1f  allocs/op=%v", tc.size, nsPerOp, allocs)
		})
	}
}