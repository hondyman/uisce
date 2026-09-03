// Package finlib implements host-runtime financial functions — calculations
// that cannot be expressed as a pushdown SQL expression (iterative solvers,
// cashflow-series inputs) and therefore run in Go against materialized rows
// rather than compiling through boresolver.Dialect.
package finlib

import (
	"errors"
	"math"
	"sort"
	"time"
)

// Cashflow is a single dated cash flow used by XIRR.
type Cashflow struct {
	Date   time.Time
	Amount float64
}

const (
	maxNewtonIterations    = 100
	newtonTolerance        = 1e-9
	maxBisectionIterations = 200
	bisectionTolerance     = 1e-9
	daysPerYear            = 365.0
)

var (
	ErrInsufficientCashflows = errors.New("finlib: at least two cash flows are required")
	ErrNoSignChange          = errors.New("finlib: cash flows must include at least one positive and one negative value")
	ErrNoConvergence         = errors.New("finlib: solver did not converge")
)

// XIRR computes the annualized internal rate of return for a schedule of
// irregularly dated cash flows, matching Excel's XIRR semantics (Actual/365).
//
// XIRR's Newton-Raphson step is well known to diverge on adversarial cash
// flow patterns (e.g. late large sign-reversals), so this falls back to
// bisection over a wide bracket when Newton fails to converge.
func XIRR(flows []Cashflow, guess float64) (float64, error) {
	if len(flows) < 2 {
		return 0, ErrInsufficientCashflows
	}

	sorted := make([]Cashflow, len(flows))
	copy(sorted, flows)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Date.Before(sorted[j].Date) })

	if !hasSignChange(amounts(sorted)) {
		return 0, ErrNoSignChange
	}

	t0 := sorted[0].Date
	years := make([]float64, len(sorted))
	for i, cf := range sorted {
		years[i] = cf.Date.Sub(t0).Hours() / 24 / daysPerYear
	}

	npv := func(rate float64) float64 {
		sum := 0.0
		for i, cf := range sorted {
			sum += cf.Amount / math.Pow(1+rate, years[i])
		}
		return sum
	}
	dnpv := func(rate float64) float64 {
		sum := 0.0
		for i, cf := range sorted {
			if years[i] == 0 {
				continue
			}
			sum -= years[i] * cf.Amount / math.Pow(1+rate, years[i]+1)
		}
		return sum
	}

	if rate, ok := newtonSolve(npv, dnpv, guess); ok {
		return rate, nil
	}
	return bisectionSolve(npv)
}

// IRR computes the periodic internal rate of return for a series of
// equally-spaced cash flows (flows[0] is period 0, flows[i] is period i).
func IRR(flows []float64, guess float64) (float64, error) {
	if len(flows) < 2 {
		return 0, ErrInsufficientCashflows
	}
	if !hasSignChange(flows) {
		return 0, ErrNoSignChange
	}

	npv := func(rate float64) float64 {
		sum := 0.0
		for i, v := range flows {
			sum += v / math.Pow(1+rate, float64(i))
		}
		return sum
	}
	dnpv := func(rate float64) float64 {
		sum := 0.0
		for i, v := range flows {
			if i == 0 {
				continue
			}
			sum -= float64(i) * v / math.Pow(1+rate, float64(i+1))
		}
		return sum
	}

	if rate, ok := newtonSolve(npv, dnpv, guess); ok {
		return rate, nil
	}
	return bisectionSolve(npv)
}

func newtonSolve(npv, dnpv func(float64) float64, guess float64) (float64, bool) {
	rate := guess
	if rate <= -1 {
		rate = 0.1
	}
	for i := 0; i < maxNewtonIterations; i++ {
		f := npv(rate)
		if math.Abs(f) < newtonTolerance {
			return rate, true
		}
		d := dnpv(rate)
		if d == 0 || math.IsNaN(d) {
			return 0, false
		}
		next := rate - f/d
		if next <= -1 {
			// Pull back toward the valid domain (rate > -1) instead of
			// stepping into an undefined (1+rate)^t.
			next = (rate - 1) / 2
		}
		if math.IsNaN(next) || math.IsInf(next, 0) {
			return 0, false
		}
		if math.Abs(next-rate) < newtonTolerance {
			return next, true
		}
		rate = next
	}
	return 0, false
}

// bisectionSolve brackets a root of npv on (-0.9999, hi], expanding hi if
// needed, then bisects. Used only when Newton's method fails to converge.
func bisectionSolve(npv func(float64) float64) (float64, error) {
	lo, hi := -0.9999, 10.0
	fLo, fHi := npv(lo), npv(hi)

	if math.IsNaN(fLo) {
		return 0, ErrNoConvergence
	}
	if fLo*fHi > 0 {
		hi = 100.0
		fHi = npv(hi)
		if fLo*fHi > 0 {
			return 0, ErrNoConvergence
		}
	}

	for i := 0; i < maxBisectionIterations; i++ {
		mid := (lo + hi) / 2
		fMid := npv(mid)
		if math.Abs(fMid) < bisectionTolerance {
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

func hasSignChange(values []float64) bool {
	hasPos, hasNeg := false, false
	for _, v := range values {
		if v > 0 {
			hasPos = true
		} else if v < 0 {
			hasNeg = true
		}
	}
	return hasPos && hasNeg
}

func amounts(flows []Cashflow) []float64 {
	out := make([]float64, len(flows))
	for i, cf := range flows {
		out[i] = cf.Amount
	}
	return out
}
