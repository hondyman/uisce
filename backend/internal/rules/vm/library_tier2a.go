package vm

// Tier 2a additions to the function library: plain statistical aggregates
// (STDEV.S/.P, VAR.S/.P, CORREL, COVARIANCE.S, SLOPE, MEDIAN, PERCENTILE) -
// per docs/unified-rule-engine-handoff.md's Tier 2a, these are mechanical:
// mostly StarRocks emitters and fixtures, no new AST shapes, no window
// functions (that's Tier 2c - see the deferred TWRR/drawdown note in the
// handoff). Naming follows the existing MAX_LENGTH/NOT_EMPTY convention
// (underscore, not dot) rather than Excel's STDEV.S/VAR.S spelling, since
// the parser's isIdentPart already treats '.' as part of a dotted field
// path - STDEV.S would be ambiguous with field-path syntax if followed by
// "(".
//
// Sample-vs-population is the convention landmine this whole engagement
// keeps flagging: STDEV_S/VAR_S use Bessel's correction (n-1, "this is a
// sample of a larger population" - the common case for realized returns
// over a finite observation window); STDEV_P/VAR_P use n (the data *is*
// the whole population). Each Description states this explicitly, not
// just the doc comment here.

import (
	"fmt"
	"math"
	"sort"
)

func init() {
	reg := func(s *FunctionSpec) { Library[s.Name] = s }

	reg(&FunctionSpec{
		Name: "STDEV_S", Signature: "(values number[]) -> number", Category: "statistics",
		Description: "Sample standard deviation (n-1 denominator, Bessel's correction). Convention: sample, not population - see STDEV_P for the population form.",
		Native: func(args []any) (any, error) {
			v, err := variance(args, "STDEV_S", true)
			if err != nil {
				return nil, err
			}
			return math.Sqrt(v), nil
		},
		SQLEmit: map[Dialect]SQLEmitter{DialectStarRocks: aggPassthrough("STDDEV_SAMP")},
	})
	reg(&FunctionSpec{
		Name: "STDEV_P", Signature: "(values number[]) -> number", Category: "statistics",
		Description: "Population standard deviation (n denominator) - use when the values ARE the entire population being measured, not a sample of a larger one.",
		Native: func(args []any) (any, error) {
			v, err := variance(args, "STDEV_P", false)
			if err != nil {
				return nil, err
			}
			return math.Sqrt(v), nil
		},
		SQLEmit: map[Dialect]SQLEmitter{DialectStarRocks: aggPassthrough("STDDEV_POP")},
	})
	reg(&FunctionSpec{
		Name: "VAR_S", Signature: "(values number[]) -> number", Category: "statistics",
		Description: "Sample variance (n-1 denominator, Bessel's correction). Convention: sample, not population - see VAR_P.",
		Native: func(args []any) (any, error) {
			return variance(args, "VAR_S", true)
		},
		SQLEmit: map[Dialect]SQLEmitter{DialectStarRocks: aggPassthrough("VAR_SAMP")},
	})
	reg(&FunctionSpec{
		Name: "VAR_P", Signature: "(values number[]) -> number", Category: "statistics",
		Description: "Population variance (n denominator) - use when the values ARE the entire population being measured, not a sample of a larger one.",
		Native: func(args []any) (any, error) {
			return variance(args, "VAR_P", false)
		},
		SQLEmit: map[Dialect]SQLEmitter{DialectStarRocks: aggPassthrough("VAR_POP")},
	})
	reg(&FunctionSpec{
		Name: "COVARIANCE_S", Signature: "(x number[], y number[]) -> number", Category: "statistics",
		Description: "Sample covariance between two series (n-1 denominator). Convention: sample, not population.",
		Native: func(args []any) (any, error) {
			x, y, err := requirePairedSlices(args, "COVARIANCE_S")
			if err != nil {
				return nil, err
			}
			if len(x) < 2 {
				return nil, fmt.Errorf("COVARIANCE_S needs at least 2 paired values, got %d", len(x))
			}
			mx, my := mean(x), mean(y)
			sum := 0.0
			for i := range x {
				sum += (x[i] - mx) * (y[i] - my)
			}
			return sum / float64(len(x)-1), nil
		},
		SQLEmit: map[Dialect]SQLEmitter{DialectStarRocks: agg2Passthrough("COVAR_SAMP")},
	})
	reg(&FunctionSpec{
		Name: "CORREL", Signature: "(x number[], y number[]) -> number", Category: "statistics",
		Description: "Pearson correlation coefficient between two series, in [-1, 1]. Scale-invariant - sample vs. population normalization cancels, so this needs no such convention note.",
		Native: func(args []any) (any, error) {
			x, y, err := requirePairedSlices(args, "CORREL")
			if err != nil {
				return nil, err
			}
			r, err := correl(x, y)
			if err != nil {
				return nil, fmt.Errorf("CORREL: %w", err)
			}
			return r, nil
		},
		SQLEmit: map[Dialect]SQLEmitter{DialectStarRocks: agg2Passthrough("CORR")},
	})
	reg(&FunctionSpec{
		Name: "SLOPE", Signature: "(known_ys number[], known_xs number[]) -> number", Category: "statistics",
		Description: "Slope of the linear least-squares regression line through the given points - Cov(y,x)/Var(x). Argument order matches Excel's SLOPE(known_y's, known_x's): y first, x second.",
		Native: func(args []any) (any, error) {
			y, x, err := requirePairedSlices(args, "SLOPE")
			if err != nil {
				return nil, err
			}
			s, err := slope(y, x)
			if err != nil {
				return nil, fmt.Errorf("SLOPE: %w", err)
			}
			return s, nil
		},
		SQLEmit: map[Dialect]SQLEmitter{
			DialectStarRocks: func(args []string) (string, error) {
				if len(args) != 2 {
					return "", fmt.Errorf("SLOPE expects 2 args (known_ys, known_xs), got %d", len(args))
				}
				y, x := args[0], args[1]
				return fmt.Sprintf("(COVAR_SAMP(%s, %s) / VAR_SAMP(%s))", y, x, x), nil
			},
		},
	})
	reg(&FunctionSpec{
		Name: "MEDIAN", Signature: "(values number[]) -> number", Category: "statistics",
		Description: "Median (50th percentile) of a set of values - the average of the two middle values when the count is even.",
		Native: func(args []any) (any, error) {
			vals, err := requireFloatSlice(args)
			if err != nil {
				return nil, err
			}
			if len(vals) == 0 {
				return nil, fmt.Errorf("MEDIAN: expects at least 1 value, got 0")
			}
			return medianOf(vals), nil
		},
		SQLEmit: map[Dialect]SQLEmitter{
			DialectStarRocks: func(args []string) (string, error) {
				if len(args) != 1 {
					return "", fmt.Errorf("MEDIAN expects 1 arg, got %d", len(args))
				}
				return fmt.Sprintf("PERCENTILE_CONT(%s, 0.5)", args[0]), nil
			},
		},
	})
	reg(&FunctionSpec{
		Name: "PERCENTILE", Signature: "(values number[], k number) -> number", Category: "statistics",
		Description: "The k-th percentile of a set of values (0 <= k <= 1), linearly interpolated between the two nearest values - matches StarRocks PERCENTILE_CONT and Excel's PERCENTILE.INC (inclusive), NOT PERCENTILE.EXC.",
		Native: func(args []any) (any, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("PERCENTILE expects 2 args (values, k), got %d", len(args))
			}
			vals, err := requireFloatSlice(args[0:1])
			if err != nil {
				return nil, fmt.Errorf("PERCENTILE values: %w", err)
			}
			k, ok := args[1].(float64)
			if !ok {
				return nil, fmt.Errorf("PERCENTILE k must be numeric, got %T", args[1])
			}
			return percentileOf(vals, k)
		},
		SQLEmit: map[Dialect]SQLEmitter{DialectStarRocks: agg2Passthrough("PERCENTILE_CONT")},
	})
}

// agg2Passthrough is agg2Passthrough(sqlName) -> "sqlName(a, b)", the
// 2-argument sibling of library.go's 1-argument aggPassthrough.
func agg2Passthrough(sqlName string) SQLEmitter {
	return func(args []string) (string, error) {
		if len(args) != 2 {
			return "", fmt.Errorf("%s expects 2 args, got %d", sqlName, len(args))
		}
		return fmt.Sprintf("%s(%s, %s)", sqlName, args[0], args[1]), nil
	}
}

func mean(vals []float64) float64 {
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

// variance computes sample (n-1) or population (n) variance, erroring on
// too few values for the requested form (sample variance is undefined for
// n<2; population variance is defined for n>=1).
func variance(args []any, fnName string, sample bool) (float64, error) {
	vals, err := requireFloatSlice(args)
	if err != nil {
		return 0, err
	}
	minN := 1
	if sample {
		minN = 2
	}
	if len(vals) < minN {
		return 0, fmt.Errorf("%s needs at least %d value(s), got %d", fnName, minN, len(vals))
	}
	m := mean(vals)
	sumSq := 0.0
	for _, v := range vals {
		d := v - m
		sumSq += d * d
	}
	denom := float64(len(vals))
	if sample {
		denom = float64(len(vals) - 1)
	}
	return sumSq / denom, nil
}

// requirePairedSlices flattens two array arguments into equal-length
// []float64 pairs, erroring if their lengths disagree.
func requirePairedSlices(args []any, fnName string) (x, y []float64, err error) {
	if len(args) != 2 {
		return nil, nil, fmt.Errorf("%s expects 2 args, got %d", fnName, len(args))
	}
	x, err = requireFloatSlice(args[0:1])
	if err != nil {
		return nil, nil, fmt.Errorf("%s first arg: %w", fnName, err)
	}
	y, err = requireFloatSlice(args[1:2])
	if err != nil {
		return nil, nil, fmt.Errorf("%s second arg: %w", fnName, err)
	}
	if len(x) != len(y) {
		return nil, nil, fmt.Errorf("%s args must be the same length (%d vs %d)", fnName, len(x), len(y))
	}
	return x, y, nil
}

// correl computes the Pearson correlation coefficient directly from the
// centered sums (rather than via covariance/stdev), so it needs no
// sample-vs-population choice at all - the n (or n-1) normalization on
// numerator and denominator cancels identically either way.
func correl(x, y []float64) (float64, error) {
	if len(x) < 2 {
		return 0, fmt.Errorf("needs at least 2 paired values, got %d", len(x))
	}
	mx, my := mean(x), mean(y)
	var sxy, sxx, syy float64
	for i := range x {
		dx, dy := x[i]-mx, y[i]-my
		sxy += dx * dy
		sxx += dx * dx
		syy += dy * dy
	}
	denom := math.Sqrt(sxx * syy)
	if denom == 0 {
		return 0, fmt.Errorf("undefined: one or both series has zero variance")
	}
	return sxy / denom, nil
}

// slope is Excel's SLOPE(known_ys, known_xs): the OLS regression
// coefficient Sxy/Sxx, sharing correl's centered-sum approach so the
// sample-vs-population normalization question doesn't arise here either.
func slope(y, x []float64) (float64, error) {
	if len(x) < 2 {
		return 0, fmt.Errorf("needs at least 2 paired values, got %d", len(x))
	}
	mx, my := mean(x), mean(y)
	var sxy, sxx float64
	for i := range x {
		dx, dy := x[i]-mx, y[i]-my
		sxy += dx * dy
		sxx += dx * dx
	}
	if sxx == 0 {
		return 0, fmt.Errorf("undefined: known_xs has zero variance")
	}
	return sxy / sxx, nil
}

func medianOf(vals []float64) float64 {
	sorted := append([]float64(nil), vals...)
	sort.Float64s(sorted)
	n := len(sorted)
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}

// percentileOf implements the same linear-interpolation convention as
// StarRocks PERCENTILE_CONT and Excel PERCENTILE.INC: rank = k*(n-1),
// interpolating between the values at floor(rank) and ceil(rank).
func percentileOf(vals []float64, k float64) (float64, error) {
	if len(vals) == 0 {
		return 0, fmt.Errorf("PERCENTILE: expects at least 1 value, got 0")
	}
	if k < 0 || k > 1 {
		return 0, fmt.Errorf("PERCENTILE: k must be in [0, 1], got %v", k)
	}
	sorted := append([]float64(nil), vals...)
	sort.Float64s(sorted)
	n := len(sorted)
	if n == 1 {
		return sorted[0], nil
	}
	rank := k * float64(n-1)
	lo := int(math.Floor(rank))
	hi := int(math.Ceil(rank))
	if lo == hi {
		return sorted[lo], nil
	}
	frac := rank - float64(lo)
	return sorted[lo] + frac*(sorted[hi]-sorted[lo]), nil
}
