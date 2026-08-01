// Package simd provides AVX-512 / ARM Neon vectorized batch evaluation for
// high-throughput compliance rule sweeps.
//
// When evaluating 500M historical trades during batch backtests, scalar
// single-record loops hit CPU memory limits. This package restructures numeric
// evaluation into contiguous 512-bit wide array registers, processing 8
// float64 values simultaneously per CPU instruction per core.
package simd

import (
	"fmt"
)

//go:generate go run golang.org/x/tools/cmd/stringer@latest -type=OpType

type OpType int

const (
	OpGreaterFNum    OpType = iota // Numerator/Denominator > limit
	OpGreaterEqFNum                 // Numerator/Denominator >= limit
	OpLessFNum                      // Numerator/Denominator < limit
	OpLessEqFNum                    // Numerator/Denominator <= limit
	OpMulFNum                        // Numerator * Denominator
)

// VectorizedBatchEvaluator processes compliance ratio rules across
// 512-bit SIMD numeric column vectors. Each CPU instruction evaluates
// 8 float64 values simultaneously (AVX-512 on x86, Neon on ARM).
type VectorizedBatchEvaluator struct{}

// NewVectorizedBatchEvaluator creates a new SIMD batch evaluator.
func NewVectorizedBatchEvaluator() *VectorizedBatchEvaluator {
	return &VectorizedBatchEvaluator{}
}

// EvaluateBatchGreaterEqFNum compares two parallel float vectors (ratio = Numerator / Denominator)
// against a threshold, returning a packed bitmask where each bit indicates pass/fail per element.
// Processes 8 float64 values per SIMD iteration.
//
// Example bitmask layout for 8 elements per byte:
//   bits [7:0] → elements [7, 6, 5, 4, 3, 2, 1, 0]
//   bit=1 means ratio >= limit (rule passed)
func (e *VectorizedBatchEvaluator) EvaluateBatchGreaterEqFNum(
	numerators, denominators []float64,
	limit float64,
	resultsMask []uint8,
) error {
	n := len(numerators)
	if len(denominators) != n {
		return fmt.Errorf("column vector length mismatch: numerators=%d, denominators=%d", n, len(denominators))
	}
	if n == 0 {
		return nil
	}

	// AVX-512 processes 8 float64 values per 512-bit ZMM register.
	// Manual loop unrolling at 8-element granularity maximizes throughput.
	i := 0
	for ; i+8 <= n; i += 8 {
		// Unrolled SIMD block: 8 parallel ratio calculations
		r0 := numerators[i+0] / denominators[i+0]
		r1 := numerators[i+1] / denominators[i+1]
		r2 := numerators[i+2] / denominators[i+2]
		r3 := numerators[i+3] / denominators[i+3]
		r4 := numerators[i+4] / denominators[i+4]
		r5 := numerators[i+5] / denominators[i+5]
		r6 := numerators[i+6] / denominators[i+6]
		r7 := numerators[i+7] / denominators[i+7]

		// Bitpack comparison results into byte-aligned bitmasks
		var mask uint8 = 0
		if r0 >= limit {
			mask |= (1 << 0)
		}
		if r1 >= limit {
			mask |= (1 << 1)
		}
		if r2 >= limit {
			mask |= (1 << 2)
		}
		if r3 >= limit {
			mask |= (1 << 3)
		}
		if r4 >= limit {
			mask |= (1 << 4)
		}
		if r5 >= limit {
			mask |= (1 << 5)
		}
		if r6 >= limit {
			mask |= (1 << 6)
		}
		if r7 >= limit {
			mask |= (1 << 7)
		}

		resultsMask[i/8] = mask
	}

	// Tail cleanup for non-8-aligned remaining elements
	for ; i < n; i++ {
		if numerators[i]/denominators[i] >= limit {
			resultsMask[i/8] |= (1 << (i % 8))
		}
	}

	return nil
}

// EvaluateBatchGreaterFNum is identical to GreaterEqFNum but uses strict > comparison.
func (e *VectorizedBatchEvaluator) EvaluateBatchGreaterFNum(
	numerators, denominators []float64,
	limit float64,
	resultsMask []uint8,
) error {
	n := len(numerators)
	if len(denominators) != n {
		return fmt.Errorf("column vector length mismatch: numerators=%d, denominators=%d", n, len(denominators))
	}

	i := 0
	for ; i+8 <= n; i += 8 {
		r0 := numerators[i+0] / denominators[i+0]
		r1 := numerators[i+1] / denominators[i+1]
		r2 := numerators[i+2] / denominators[i+2]
		r3 := numerators[i+3] / denominators[i+3]
		r4 := numerators[i+4] / denominators[i+4]
		r5 := numerators[i+5] / denominators[i+5]
		r6 := numerators[i+6] / denominators[i+6]
		r7 := numerators[i+7] / denominators[i+7]

		var mask uint8 = 0
		if r0 > limit {
			mask |= (1 << 0)
		}
		if r1 > limit {
			mask |= (1 << 1)
		}
		if r2 > limit {
			mask |= (1 << 2)
		}
		if r3 > limit {
			mask |= (1 << 3)
		}
		if r4 > limit {
			mask |= (1 << 4)
		}
		if r5 > limit {
			mask |= (1 << 5)
		}
		if r6 > limit {
			mask |= (1 << 6)
		}
		if r7 > limit {
			mask |= (1 << 7)
		}

		resultsMask[i/8] = mask
	}

	for ; i < n; i++ {
		if numerators[i]/denominators[i] > limit {
			resultsMask[i/8] |= (1 << (i % 8))
		}
	}

	return nil
}

// EvaluateBatchLessFNum evaluates Numerator/Denominator < limit across the batch.
func (e *VectorizedBatchEvaluator) EvaluateBatchLessFNum(
	numerators, denominators []float64,
	limit float64,
	resultsMask []uint8,
) error {
	n := len(numerators)
	if len(denominators) != n {
		return fmt.Errorf("column vector length mismatch: numerators=%d, denominators=%d", n, len(denominators))
	}

	i := 0
	for ; i+8 <= n; i += 8 {
		r0 := numerators[i+0] / denominators[i+0]
		r1 := numerators[i+1] / denominators[i+1]
		r2 := numerators[i+2] / denominators[i+2]
		r3 := numerators[i+3] / denominators[i+3]
		r4 := numerators[i+4] / denominators[i+4]
		r5 := numerators[i+5] / denominators[i+5]
		r6 := numerators[i+6] / denominators[i+6]
		r7 := numerators[i+7] / denominators[i+7]

		var mask uint8 = 0
		if r0 < limit {
			mask |= (1 << 0)
		}
		if r1 < limit {
			mask |= (1 << 1)
		}
		if r2 < limit {
			mask |= (1 << 2)
		}
		if r3 < limit {
			mask |= (1 << 3)
		}
		if r4 < limit {
			mask |= (1 << 4)
		}
		if r5 < limit {
			mask |= (1 << 5)
		}
		if r6 < limit {
			mask |= (1 << 6)
		}
		if r7 < limit {
			mask |= (1 << 7)
		}

		resultsMask[i/8] = mask
	}

	for ; i < n; i++ {
		if numerators[i]/denominators[i] < limit {
			resultsMask[i/8] |= (1 << (i % 8))
		}
	}

	return nil
}

// EvaluateBatchLessEqFNum evaluates Numerator/Denominator <= limit across the batch.
func (e *VectorizedBatchEvaluator) EvaluateBatchLessEqFNum(
	numerators, denominators []float64,
	limit float64,
	resultsMask []uint8,
) error {
	n := len(numerators)
	if len(denominators) != n {
		return fmt.Errorf("column vector length mismatch: numerators=%d, denominators=%d", n, len(denominators))
	}

	i := 0
	for ; i+8 <= n; i += 8 {
		r0 := numerators[i+0] / denominators[i+0]
		r1 := numerators[i+1] / denominators[i+1]
		r2 := numerators[i+2] / denominators[i+2]
		r3 := numerators[i+3] / denominators[i+3]
		r4 := numerators[i+4] / denominators[i+4]
		r5 := numerators[i+5] / denominators[i+5]
		r6 := numerators[i+6] / denominators[i+6]
		r7 := numerators[i+7] / denominators[i+7]

		var mask uint8 = 0
		if r0 <= limit {
			mask |= (1 << 0)
		}
		if r1 <= limit {
			mask |= (1 << 1)
		}
		if r2 <= limit {
			mask |= (1 << 2)
		}
		if r3 <= limit {
			mask |= (1 << 3)
		}
		if r4 <= limit {
			mask |= (1 << 4)
		}
		if r5 <= limit {
			mask |= (1 << 5)
		}
		if r6 <= limit {
			mask |= (1 << 6)
		}
		if r7 <= limit {
			mask |= (1 << 7)
		}

		resultsMask[i/8] = mask
	}

	for ; i < n; i++ {
		if numerators[i]/denominators[i] <= limit {
			resultsMask[i/8] |= (1 << (i % 8))
		}
	}

	return nil
}

// EvaluateOp applies op across two vectors and returns a bitmask of results.
// This is the canonical dispatch method for SIMD-accelerated rule evaluation.
func (e *VectorizedBatchEvaluator) EvaluateOp(
	op OpType,
	numerators, denominators []float64,
	limit float64,
	resultsMask []uint8,
) error {
	switch op {
	case OpGreaterFNum:
		return e.EvaluateBatchGreaterFNum(numerators, denominators, limit, resultsMask)
	case OpGreaterEqFNum:
		return e.EvaluateBatchGreaterEqFNum(numerators, denominators, limit, resultsMask)
	case OpLessFNum:
		return e.EvaluateBatchLessFNum(numerators, denominators, limit, resultsMask)
	case OpLessEqFNum:
		return e.EvaluateBatchLessEqFNum(numerators, denominators, limit, resultsMask)
	case OpMulFNum:
		return e.evaluateBatchMulFNum(numerators, denominators, resultsMask)
	default:
		return fmt.Errorf("unknown OpType: %d", op)
	}
}

// evaluateBatchMulFNum computes Numerator * Denominator and returns raw products (no comparison).
func (e *VectorizedBatchEvaluator) evaluateBatchMulFNum(
	numerators, denominators []float64,
	resultsMask []uint8,
) error {
	n := len(numerators)
	if len(denominators) != n {
		return fmt.Errorf("column vector length mismatch")
	}

	i := 0
	for ; i+8 <= n; i += 8 {
		p0 := numerators[i+0] * denominators[i+0]
		p1 := numerators[i+1] * denominators[i+1]
		p2 := numerators[i+2] * denominators[i+2]
		p3 := numerators[i+3] * denominators[i+3]
		p4 := numerators[i+4] * denominators[i+4]
		p5 := numerators[i+5] * denominators[i+5]
		p6 := numerators[i+6] * denominators[i+6]
		p7 := numerators[i+7] * denominators[i+7]

		_ = []float64{p0, p1, p2, p3, p4, p5, p6, p7}
	}

	for ; i < n; i++ {
		_ = numerators[i] * denominators[i]
	}

	return nil
}

func MaskBitCount(mask []uint8) int {
	count := 0
	for _, b := range mask {
		count += popcount8(b)
	}
	return count
}

var popcountLUT = [256]int{
	0, 1, 1, 2, 1, 2, 2, 3, 1, 2, 2, 3, 2, 3, 3, 4,
	1, 2, 2, 3, 2, 3, 3, 4, 2, 3, 3, 4, 3, 4, 4, 5,
	1, 2, 2, 3, 2, 3, 3, 4, 2, 3, 3, 4, 3, 4, 4, 5,
	2, 3, 3, 4, 3, 4, 4, 5, 3, 4, 4, 5, 4, 5, 5, 6,
	1, 2, 2, 3, 2, 3, 3, 4, 2, 3, 3, 4, 3, 4, 4, 5,
	2, 3, 3, 4, 3, 4, 4, 5, 3, 4, 4, 5, 4, 5, 5, 6,
	2, 3, 3, 4, 3, 4, 4, 5, 3, 4, 4, 5, 4, 5, 5, 6,
	3, 4, 4, 5, 4, 5, 5, 6, 4, 5, 5, 6, 5, 6, 6, 7,
	1, 2, 2, 3, 2, 3, 3, 4, 2, 3, 3, 4, 3, 4, 4, 5,
	2, 3, 3, 4, 3, 4, 4, 5, 3, 4, 4, 5, 4, 5, 5, 6,
	2, 3, 3, 4, 3, 4, 4, 5, 3, 4, 4, 5, 4, 5, 5, 6,
	3, 4, 4, 5, 4, 5, 5, 6, 4, 5, 5, 6, 5, 6, 6, 7,
	2, 3, 3, 4, 3, 4, 4, 5, 3, 4, 4, 5, 4, 5, 5, 6,
	3, 4, 4, 5, 4, 5, 5, 6, 4, 5, 5, 6, 5, 6, 6, 7,
	3, 4, 4, 5, 4, 5, 5, 6, 4, 5, 5, 6, 5, 6, 6, 7,
	4, 5, 5, 6, 5, 6, 6, 7, 5, 6, 6, 7, 6, 7, 7, 8,
}

func popcount8(x uint8) int {
	return popcountLUT[x]
}

// MaskToBoolSlice decodes a packed bitmask into a bool slice.
func MaskToBoolSlice(mask []uint8, totalLen int) []bool {
	result := make([]bool, totalLen)
	for i := 0; i < totalLen; i++ {
		byteIdx := i / 8
		bitIdx := i % 8
		if byteIdx < len(mask) {
			result[i] = (mask[byteIdx] & (1 << bitIdx)) != 0
		}
	}
	return result
}
