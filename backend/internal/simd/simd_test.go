package simd_test

import (
	"fmt"
	"testing"

	"github.com/hondyman/uisce/backend/internal/simd"
)

func TestEvaluateBatchGreaterEqFNum(t *testing.T) {
	evaluator := simd.NewVectorizedBatchEvaluator()

	tests := []struct {
		name       string
		numerators []float64
		denoms     []float64
		limit      float64
		expected   []bool
	}{
		{
			name:       "all pass",
			numerators: []float64{25, 30, 35, 40},
			denoms:     []float64{100, 100, 100, 100},
			limit:      0.25,
			expected:   []bool{true, true, true, true},
		},
		{
			name:       "all fail",
			numerators: []float64{10, 20, 15, 5},
			denoms:     []float64{100, 100, 100, 100},
			limit:      0.25,
			expected:   []bool{false, false, false, false},
		},
		{
			name:       "mixed",
			numerators: []float64{10, 30, 24, 50},
			denoms:     []float64{100, 100, 100, 100},
			limit:      0.25,
			expected:   []bool{false, true, false, true},
		},
		{
			name:       "boundary exactly equal",
			numerators: []float64{25},
			denoms:     []float64{100},
			limit:      0.25,
			expected:   []bool{true},
		},
		{
			name:       "boundary just below",
			numerators: []float64{24},
			denoms:     []float64{100},
			limit:      0.25,
			expected:   []bool{false},
		},
		{
			name:       "with zeros in denominator",
			numerators: []float64{30, 0, 30},
			denoms:     []float64{100, 100, 0},
			limit:      0.25,
			expected:   []bool{true, false, true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maskLen := (len(tt.numerators) + 7) / 8
			mask := make([]uint8, maskLen)

			err := evaluator.EvaluateBatchGreaterEqFNum(tt.numerators, tt.denoms, tt.limit, mask)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			result := simd.MaskToBoolSlice(mask, len(tt.numerators))
			if len(result) != len(tt.expected) {
				t.Fatalf("length mismatch: got %d, want %d", len(result), len(tt.expected))
			}

			for i, want := range tt.expected {
				if result[i] != want {
					t.Errorf("element %d: got %v, want %v (ratio=%.6f, limit=%.2f)",
						i, result[i], want, tt.numerators[i]/tt.denoms[i], tt.limit)
				}
			}
		})
	}
}

func TestEvaluateBatchGreaterFNum(t *testing.T) {
	evaluator := simd.NewVectorizedBatchEvaluator()

	numerators := []float64{26, 25, 24}
	denoms := []float64{100, 100, 100}
	limit := 0.25
	mask := make([]uint8, (len(numerators)+7)/8)

	err := evaluator.EvaluateBatchGreaterFNum(numerators, denoms, limit, mask)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := simd.MaskToBoolSlice(mask, len(numerators))
	expected := []bool{true, false, false}

	for i, want := range expected {
		if result[i] != want {
			t.Errorf("element %d: got %v, want %v", i, result[i], want)
		}
	}
}

func TestEvaluateBatchLessFNum(t *testing.T) {
	evaluator := simd.NewVectorizedBatchEvaluator()

	numerators := []float64{10, 30, 24}
	denoms := []float64{100, 100, 100}
	limit := 0.25
	mask := make([]uint8, (len(numerators)+7)/8)

	err := evaluator.EvaluateBatchLessFNum(numerators, denoms, limit, mask)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := simd.MaskToBoolSlice(mask, len(numerators))
	expected := []bool{true, false, true}

	for i, want := range expected {
		if result[i] != want {
			t.Errorf("element %d: got %v, want %v", i, result[i], want)
		}
	}
}

func TestEvaluateBatchLessEqFNum(t *testing.T) {
	evaluator := simd.NewVectorizedBatchEvaluator()

	numerators := []float64{25, 24, 26}
	denoms := []float64{100, 100, 100}
	limit := 0.25
	mask := make([]uint8, (len(numerators)+7)/8)

	err := evaluator.EvaluateBatchLessEqFNum(numerators, denoms, limit, mask)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := simd.MaskToBoolSlice(mask, len(numerators))
	expected := []bool{true, true, false}

	for i, want := range expected {
		if result[i] != want {
			t.Errorf("element %d: got %v, want %v", i, result[i], want)
		}
	}
}

func TestEvaluateOpDispatch(t *testing.T) {
	evaluator := simd.NewVectorizedBatchEvaluator()

	numerators := []float64{30, 20}
	denoms := []float64{100, 100}
	limit := 0.25

	opTests := []struct {
		op      simd.OpType
		want    []bool
	}{
		{simd.OpGreaterFNum, []bool{true, false}},
		{simd.OpGreaterEqFNum, []bool{true, false}},
		{simd.OpLessFNum, []bool{false, true}},
		{simd.OpLessEqFNum, []bool{false, true}},
	}

	for _, tt := range opTests {
		t.Run(fmt.Sprintf("%v", tt.op), func(t *testing.T) {
			m := make([]uint8, 1)
			err := evaluator.EvaluateOp(tt.op, numerators, denoms, limit, m)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			result := simd.MaskToBoolSlice(m, len(numerators))
			for i, want := range tt.want {
				if result[i] != want {
					t.Errorf("op=%v element %d: got %v, want %v", tt.op, i, result[i], want)
				}
			}
		})
	}
}

func TestMaskBitCount(t *testing.T) {
	tests := []struct {
		mask   []uint8
		expect int
	}{
		{[]uint8{0b00000000}, 0},
		{[]uint8{0b11111111}, 8},
		{[]uint8{0b10101010}, 4},
		{[]uint8{0b10000001, 0b00000001}, 3},
		{[]uint8{0b11111111, 0b11111111}, 16},
	}

	for _, tt := range tests {
		got := simd.MaskBitCount(tt.mask)
		if got != tt.expect {
			t.Errorf("MaskBitCount(%08b) = %d, want %d", tt.mask[0], got, tt.expect)
		}
	}
}

func TestMaskToBoolSlice(t *testing.T) {
	mask := []uint8{0b10110011}
	result := simd.MaskToBoolSlice(mask, 8)
	expected := []bool{true, true, false, false, true, true, false, true}

	for i, want := range expected {
		if result[i] != want {
			t.Errorf("element %d: got %v, want %v", i, result[i], want)
		}
	}
}

func TestUnalignedLength(t *testing.T) {
	evaluator := simd.NewVectorizedBatchEvaluator()

	// Test with length that is not a multiple of 8
	numerators := []float64{30, 20, 40, 15, 35}
	denoms := []float64{100, 100, 100, 100, 100}
	limit := 0.25
	mask := make([]uint8, (len(numerators)+7)/8)

	err := evaluator.EvaluateBatchGreaterEqFNum(numerators, denoms, limit, mask)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := simd.MaskToBoolSlice(mask, len(numerators))
	expected := []bool{true, false, true, false, true}

	for i, want := range expected {
		if result[i] != want {
			t.Errorf("element %d: got %v, want %v", i, result[i], want)
		}
	}

	// Verify bit count
	count := simd.MaskBitCount(mask)
	if count != 3 {
		t.Errorf("MaskBitCount = %d, want 3", count)
	}
}

func BenchmarkAVX512_VectorizedBatchSweep(b *testing.B) {
	const count = 1000000 // 1 Million Trades

	numerators := make([]float64, count)
	denominators := make([]float64, count)
	results := make([]uint8, count/8)

	for i := 0; i < count; i++ {
		numerators[i] = float64(i * 100)
		denominators[i] = 1000000.0
	}

	evaluator := simd.NewVectorizedBatchEvaluator()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = evaluator.EvaluateBatchGreaterEqFNum(numerators, denominators, 0.25, results)
	}
}

func BenchmarkAVX512_VectorizedBatchSweep_1Million(b *testing.B) {
	const count = 1_000_000

	numerators := make([]float64, count)
	denominators := make([]float64, count)
	results := make([]uint8, count/8)

	for i := 0; i < count; i++ {
		numerators[i] = float64(i%10000) * 100.0
		denominators[i] = 10000.0
	}

	evaluator := simd.NewVectorizedBatchEvaluator()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = evaluator.EvaluateBatchGreaterEqFNum(numerators, denominators, 0.25, results)
	}

	passCount := simd.MaskBitCount(results)
	b.Logf("Pass count: %d / %d", passCount, count)
}

func BenchmarkAVX512_VectorizedBatchSweep_500Million(b *testing.B) {
	b.Skip("Long-running benchmark — run manually with: go test -bench=BenchmarkAVX512_VectorizedBatchSweep_500Million -benchtime=1x")

	const count = 500_000_000

	numerators := make([]float64, count)
	denominators := make([]float64, count)
	results := make([]uint8, count/8)

	for i := 0; i < count; i++ {
		numerators[i] = float64(i%100000) * 10.0
		denominators[i] = 100000.0
	}

	evaluator := simd.NewVectorizedBatchEvaluator()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = evaluator.EvaluateBatchGreaterEqFNum(numerators, denominators, 0.25, results)
	}

	passCount := simd.MaskBitCount(results)
	nsPerTrade := b.Elapsed().Nanoseconds() / count
	b.Logf("500M trades evaluated in %v (%.1f ns/trade, pass=%.1f%%)",
		b.Elapsed(), float64(nsPerTrade), float64(passCount)*100.0/float64(count))
}

func BenchmarkScalarVsSIMD_Correctness(b *testing.B) {
	const count = 64

	numerators := make([]float64, count)
	denominators := make([]float64, count)

	for i := 0; i < count; i++ {
		numerators[i] = float64(i*50) / 100.0
		denominators[i] = 100.0
	}

	evaluator := simd.NewVectorizedBatchEvaluator()
	mask := make([]uint8, count/8)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = evaluator.EvaluateBatchGreaterEqFNum(numerators, denominators, 0.25, mask)
	}

	result := simd.MaskToBoolSlice(mask, count)
	passCount := simd.MaskBitCount(mask)

	fmt.Printf("SIMD 64-element batch: pass=%d/%d\n", passCount, count)
	for i, v := range result {
		scalarResult := numerators[i]/denominators[i] >= 0.25
		if v != scalarResult {
			b.Errorf("mismatch at %d: SIMD=%v scalar=%v", i, v, scalarResult)
		}
	}
}
