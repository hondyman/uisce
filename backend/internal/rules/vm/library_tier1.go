package vm

// Tier 1 additions to the function library: the primitives needed to
// complete PE-standard performance reporting (TVPI = DPI + RVPI; XNPV/
// XIRR as the irregular-cash-flow pair; SUMPRODUCT for weighted
// averages like avg_price). Same registration discipline as the rest of
// library.go - one FunctionSpec, native + (where it has a real closed
// form) a SQL emitter.
//
// Dates here are accepted as strings (YYYY-MM-DD or RFC3339), matching
// the existing IS_DATE/IS_DATETIME predicates and real BO field values -
// not day-offset numbers like XIRR's `dates` arg, which was a
// deliberately simplified proof-script convention (see irr_excel_fixtures_test.go).
// Real calc-term/rule authors have date-typed columns, not pre-computed
// offsets, so XNPV/YEARFRAC parse them directly.

import (
	"fmt"
	"math"
	"time"
)

func init() {
	reg := func(s *FunctionSpec) { Library[s.Name] = s }

	reg(&FunctionSpec{
		Name: "YEARFRAC", Signature: "(start date, end date, basis string) -> number",
		Category:    "date",
		Description: "Year fraction between two dates under a day-count convention: \"ACT/365\" (default, actual days / 365 - matches XIRR's convention), \"ACT/360\" (actual days / 360, common for money-market instruments), or \"30/360\" (US bond basis: each month counted as 30 days). The day-count primitive XIRR/XNPV and hurdle-rate annualization all reduce to.",
		Native: func(args []any) (any, error) {
			start, end, basis, err := yearfracArgs(args)
			if err != nil {
				return nil, err
			}
			return yearFrac(start, end, basis)
		},
	})

	reg(&FunctionSpec{
		Name: "XNPV", Signature: "(rate number, cash_flows number[], dates date[]) -> number",
		Category:    "financial",
		Description: "Net present value for irregularly-dated cash flows: SUM(cf_i / (1+rate)^((date_i - date_0)/365)) - XIRR's companion the way NPV is IRR's, using the same ACT/365 day-count as XIRR. No SQL pushdown, for the same reason XIRR has none: this is here for the day-count logic, not because it needs a solver, but its date-string parsing isn't (yet) expressible as a single StarRocks expression.",
		Native: func(args []any) (any, error) {
			if len(args) != 3 {
				return nil, fmt.Errorf("XNPV expects 3 args (rate, cash_flows, dates), got %d", len(args))
			}
			rate, ok := args[0].(float64)
			if !ok {
				return nil, fmt.Errorf("XNPV rate must be numeric, got %T", args[0])
			}
			cashFlows, err := requireFloatSlice(args[1:2])
			if err != nil {
				return nil, fmt.Errorf("XNPV cash_flows: %w", err)
			}
			dates, err := requireDateSlice(args[2:3])
			if err != nil {
				return nil, fmt.Errorf("XNPV dates: %w", err)
			}
			if len(dates) != len(cashFlows) {
				return nil, fmt.Errorf("XNPV cash_flows and dates must be the same length (%d vs %d)", len(cashFlows), len(dates))
			}
			base := dates[0]
			npv := 0.0
			for i, cf := range cashFlows {
				years := end365(base, dates[i])
				npv += cf / math.Pow(1+rate, years)
			}
			return npv, nil
		},
	})

	reg(&FunctionSpec{
		Name: "SUMPRODUCT", Signature: "(a number[], b number[]) -> number",
		Category:    "aggregation",
		Description: "Sum of pairwise products: SUM(a_i * b_i). Used for weighted aggregates over grouped rows, e.g. avg_price = SUMPRODUCT(qty, price) / SUM(qty).",
		Native: func(args []any) (any, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("SUMPRODUCT expects 2 args, got %d", len(args))
			}
			a, err := requireFloatSlice(args[0:1])
			if err != nil {
				return nil, fmt.Errorf("SUMPRODUCT first arg: %w", err)
			}
			b, err := requireFloatSlice(args[1:2])
			if err != nil {
				return nil, fmt.Errorf("SUMPRODUCT second arg: %w", err)
			}
			if len(a) != len(b) {
				return nil, fmt.Errorf("SUMPRODUCT args must be the same length (%d vs %d)", len(a), len(b))
			}
			sum := 0.0
			for i := range a {
				sum += a[i] * b[i]
			}
			return sum, nil
		},
		SQLEmit: map[Dialect]SQLEmitter{
			DialectStarRocks: func(args []string) (string, error) {
				if len(args) != 2 {
					return "", fmt.Errorf("SUMPRODUCT expects 2 args, got %d", len(args))
				}
				return fmt.Sprintf("SUM(%s * %s)", args[0], args[1]), nil
			},
		},
	})

	reg(&FunctionSpec{
		Name: "LN", Signature: "(x number) -> number", Category: "math",
		Description: "Natural logarithm - used by annualization ((1+r)^(365/days)-1), geometric linking, and lnPME.",
		Native: func(args []any) (any, error) {
			x, err := require1Float(args, "LN")
			if err != nil {
				return nil, err
			}
			if x <= 0 {
				return nil, fmt.Errorf("LN: argument must be positive, got %v", x)
			}
			return math.Log(x), nil
		},
		SQLEmit: map[Dialect]SQLEmitter{DialectStarRocks: math1ArgSQL("LN")},
	})

	reg(&FunctionSpec{
		Name: "EXP", Signature: "(x number) -> number", Category: "math",
		Description: "e^x - the inverse of LN, used alongside it in geometric linking and annualization.",
		Native: func(args []any) (any, error) {
			x, err := require1Float(args, "EXP")
			if err != nil {
				return nil, err
			}
			return math.Exp(x), nil
		},
		SQLEmit: map[Dialect]SQLEmitter{DialectStarRocks: math1ArgSQL("EXP")},
	})

	reg(&FunctionSpec{
		Name: "SQRT", Signature: "(x number) -> number", Category: "math",
		Description: "Square root - used by standard-deviation-style calculations.",
		Native: func(args []any) (any, error) {
			x, err := require1Float(args, "SQRT")
			if err != nil {
				return nil, err
			}
			if x < 0 {
				return nil, fmt.Errorf("SQRT: argument must be non-negative, got %v", x)
			}
			return math.Sqrt(x), nil
		},
		SQLEmit: map[Dialect]SQLEmitter{DialectStarRocks: math1ArgSQL("SQRT")},
	})
}

func math1ArgSQL(sqlName string) SQLEmitter {
	return func(args []string) (string, error) {
		if len(args) != 1 {
			return "", fmt.Errorf("%s expects 1 arg, got %d", sqlName, len(args))
		}
		return fmt.Sprintf("%s(%s)", sqlName, args[0]), nil
	}
}

func require1Float(args []any, fnName string) (float64, error) {
	v, err := require1(args, fnName)
	if err != nil {
		return 0, err
	}
	f, ok := v.(float64)
	if !ok {
		return 0, fmt.Errorf("%s expects a numeric argument, got %T", fnName, v)
	}
	return f, nil
}

// parseDate accepts either a bare date (YYYY-MM-DD) or an RFC3339
// datetime, matching the two formats IS_DATE/IS_DATETIME already
// recognize.
func parseDate(v any) (time.Time, error) {
	s, ok := v.(string)
	if !ok {
		return time.Time{}, fmt.Errorf("expected a date string, got %T", v)
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("%q is not a recognized date (want YYYY-MM-DD or RFC3339)", s)
}

func requireDateSlice(args []any) ([]time.Time, error) {
	var out []time.Time
	for _, a := range args {
		switch v := a.(type) {
		case []any:
			for _, e := range v {
				t, err := parseDate(e)
				if err != nil {
					return nil, err
				}
				out = append(out, t)
			}
		default:
			t, err := parseDate(v)
			if err != nil {
				return nil, err
			}
			out = append(out, t)
		}
	}
	return out, nil
}

func yearfracArgs(args []any) (start, end time.Time, basis string, err error) {
	if len(args) != 3 {
		return time.Time{}, time.Time{}, "", fmt.Errorf("YEARFRAC expects 3 args (start, end, basis), got %d", len(args))
	}
	start, err = parseDate(args[0])
	if err != nil {
		return time.Time{}, time.Time{}, "", fmt.Errorf("YEARFRAC start: %w", err)
	}
	end, err = parseDate(args[1])
	if err != nil {
		return time.Time{}, time.Time{}, "", fmt.Errorf("YEARFRAC end: %w", err)
	}
	basisStr, ok := args[2].(string)
	if !ok {
		return time.Time{}, time.Time{}, "", fmt.Errorf("YEARFRAC basis must be a string, got %T", args[2])
	}
	return start, end, basisStr, nil
}

// end365 is XIRR/XNPV's day-count convention: actual days between base
// and t, divided by 365 - matching irr.go's dayPeriods but operating on
// real dates instead of pre-computed day-offsets.
func end365(base, t time.Time) float64 {
	return t.Sub(base).Hours() / 24 / 365.0
}

// yearFrac computes the year fraction between start and end under the
// given day-count basis.
func yearFrac(start, end time.Time, basis string) (float64, error) {
	switch basis {
	case "ACT/365", "":
		return end.Sub(start).Hours() / 24 / 365.0, nil
	case "ACT/360":
		return end.Sub(start).Hours() / 24 / 360.0, nil
	case "30/360":
		return thirtyThreeSixty(start, end), nil
	default:
		return 0, fmt.Errorf("YEARFRAC: unsupported basis %q (want ACT/365, ACT/360, or 30/360)", basis)
	}
}

// thirtyThreeSixty implements the US (NASD) 30/360 convention: each
// month counted as 30 days, with the standard end-of-month adjustment
// (day 31 clamped to 30 on both ends before differencing).
func thirtyThreeSixty(start, end time.Time) float64 {
	d1, d2 := start.Day(), end.Day()
	if d1 == 31 {
		d1 = 30
	}
	if d2 == 31 && d1 == 30 {
		d2 = 30
	}
	days := (end.Year()-start.Year())*360 + (int(end.Month())-int(start.Month()))*30 + (d2 - d1)
	return float64(days) / 360.0
}
