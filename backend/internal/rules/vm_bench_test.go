package rules

import (
	"testing"

	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
)

func BenchmarkVM_FastRecord(b *testing.B) {
	sizes := []string{"small", "medium", "large"}
	for _, size := range sizes {
		b.Run(size, func(b *testing.B) {
			f := BuildFixture(b, size)
			theVM := vm.NewVM()
			stack := vm.GetStack()
			defer vm.PutStack(stack)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				theVM.Run(f.Compiled, f.FastRecord, stack)
			}
		})
	}
}

func BenchmarkVM_Parallel(b *testing.B) {
	f := BuildFixture(b, "medium")
	theVM := vm.NewVM()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		stack := vm.GetStack()
		defer vm.PutStack(stack)
		rec := f.FastRecord // Read-only, safe for concurrent reads

		for pb.Next() {
			theVM.Run(f.Compiled, rec, stack)
		}
	})
}