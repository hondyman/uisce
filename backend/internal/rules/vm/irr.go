package vm

// IRR/XIRR - Newton-Raphson with a bisection fallback, implemented once
// in Go and shared by every compile target this evaluator has (native
// server-side, and the WASM build the browser preview uses) - the
// payoff of having a single unified engine instead of separate per-
// target implementations. Deliberately native/WASM-only: unlike
// SUM/AVG/MIN/MAX/NPV, IRR/XIRR have no StarRocks-native function and no
// closed-form SQL expansion (they're the root of a polynomial with no
// general algebraic solution), so they aren't in starrocksFuncs
// (sql_compiler.go) and aren't pushdownable - a rule or calc term using
// them runs at tree-walking speed, same as any FuncCall the bytecode
// compiler doesn't support yet (see vm_compiler_funccall_test.go).
//
// Golden-tested against real Excel-computed IRR/XIRR values in
// irr_test.go - numerical convergence bugs don't announce themselves,
// they just quietly return a plausible-looking wrong number.

import (
	"fmt"
	"math"
)

const (
	irrMaxNewtonIter  = 50
	irrMaxBisectIter  = 200
	irrTolerance      = 1e-9
	irrBisectLow      = -0.9999 // rates below this make (1+r) non-positive
	irrBisectHigh     = 10.0    // 1000% - generous upper bound for a root search
	irrBisectStepScan = 0.01    // bracket-search step size
)

// npvAt evaluates sum(cf_i / (1+r)^t_i) for irr (t_i = i, integer periods)
// or xirr (t_i = days_i/365, actual/365 day-count, matching Excel's XIRR).
func npvAt(rate float64, cashFlows []float64, periods []float64) float64 {
	sum := 0.0
	for i, cf := range cashFlows {
		sum += cf / math.Pow(1+rate, periods[i])
	}
	return sum
}

// npvDerivativeAt is d/dr of npvAt, for Newton's method.
func npvDerivativeAt(rate float64, cashFlows []float64, periods []float64) float64 {
	sum := 0.0
	for i, cf := range cashFlows {
		t := periods[i]
		sum += -t * cf / math.Pow(1+rate, t+1)
	}
	return sum
}

// solveIRR finds r such that npvAt(r, cashFlows, periods) == 0. Tries
// Newton-Raphson from a few starting points first (fast, usually
// converges in single digits of iterations for well-behaved cash flow
// series); falls back to bisection over a bracket found by scanning for
// a sign change (slower but never diverges) if Newton doesn't converge
// or leaves the valid domain (1+r > 0).
func solveIRR(cashFlows []float64, periods []float64) (float64, error) {
	if len(cashFlows) < 2 {
		return 0, fmt.Errorf("IRR/XIRR requires at least 2 cash flows, got %d", len(cashFlows))
	}
	hasPositive, hasNegative := false, false
	for _, cf := range cashFlows {
		if cf > 0 {
			hasPositive = true
		}
		if cf < 0 {
			hasNegative = true
		}
	}
	if !hasPositive || !hasNegative {
		return 0, fmt.Errorf("IRR/XIRR requires at least one positive and one negative cash flow")
	}

	for _, guess := range []float64{0.1, 0.0, 0.5, -0.5, 1.0} {
		if r, ok := newtonIRR(guess, cashFlows, periods); ok {
			return r, nil
		}
	}

	return bisectIRR(cashFlows, periods)
}

func newtonIRR(guess float64, cashFlows []float64, periods []float64) (float64, bool) {
	r := guess
	for i := 0; i < irrMaxNewtonIter; i++ {
		if r <= irrBisectLow {
			return 0, false
		}
		f := npvAt(r, cashFlows, periods)
		if math.Abs(f) < irrTolerance {
			return r, true
		}
		d := npvDerivativeAt(r, cashFlows, periods)
		if d == 0 || math.IsNaN(d) || math.IsInf(d, 0) {
			return 0, false
		}
		next := r - f/d
		if math.IsNaN(next) || math.IsInf(next, 0) {
			return 0, false
		}
		r = next
	}
	return 0, false
}

func bisectIRR(cashFlows []float64, periods []float64) (float64, error) {
	// Scan for a bracket [lo, hi] where npvAt changes sign.
	prevR := irrBisectLow
	prevF := npvAt(prevR, cashFlows, periods)
	for r := irrBisectLow + irrBisectStepScan; r <= irrBisectHigh; r += irrBisectStepScan {
		f := npvAt(r, cashFlows, periods)
		if (prevF < 0 && f >= 0) || (prevF > 0 && f <= 0) {
			return bisect(prevR, r, cashFlows, periods)
		}
		prevR, prevF = r, f
	}
	return 0, fmt.Errorf("IRR/XIRR did not converge: no sign change found for rate in [%.2f, %.2f]", irrBisectLow, irrBisectHigh)
}

func bisect(lo, hi float64, cashFlows []float64, periods []float64) (float64, error) {
	fLo := npvAt(lo, cashFlows, periods)
	for i := 0; i < irrMaxBisectIter; i++ {
		mid := (lo + hi) / 2
		fMid := npvAt(mid, cashFlows, periods)
		if math.Abs(fMid) < irrTolerance || (hi-lo) < irrTolerance {
			return mid, nil
		}
		if (fLo < 0) == (fMid < 0) {
			lo, fLo = mid, fMid
		} else {
			hi = mid
		}
	}
	return (lo + hi) / 2, nil
}

// integerPeriods returns [0, 1, 2, ...] for IRR's assumption of regular
// (e.g. annual) cash flow spacing.
func integerPeriods(n int) []float64 {
	p := make([]float64, n)
	for i := range p {
		p[i] = float64(i)
	}
	return p
}

// dayPeriods converts a slice of day-offsets (days since an epoch,
// consistent across all entries - e.g. Excel serial dates, or days since
// the first cash flow) into actual/365 year fractions relative to the
// first entry, matching Excel's XIRR day-count convention.
func dayPeriods(days []float64) []float64 {
	if len(days) == 0 {
		return nil
	}
	p := make([]float64, len(days))
	base := days[0]
	for i, d := range days {
		p[i] = (d - base) / 365.0
	}
	return p
}
