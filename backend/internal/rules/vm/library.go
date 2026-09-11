package vm

// The function library: one declarative FunctionSpec per function,
// everything else - native evaluation, SQL pushdown, capability
// metadata for the editor - derived from it. Before this file, the same
// information lived in two independently-maintained maps (nativeFuncs in
// this package, starrocksFuncs in sql_compiler.go) plus a third,
// separately-maintained registry describing them - three hand-kept
// sources of truth for facts that should only ever be stated once. This
// is the same dedup principle this whole engagement has applied
// elsewhere (the asl.d.ts/schema/monaco codegen pipeline, the MAPS_TO
// resolver shared by DDL generation and rule evaluation) applied to
// functions.
//
// A new function is registration, not surgery: one FunctionSpec, no
// edits to evalFuncCall, compileNodeToSQL, or any generator. Native is
// mandatory (or the function can't run at all - there's no target
// where "native" doesn't mean something, since WASM *is* this same Go
// code cross-compiled); SQLEmit is optional and dialect-keyed, because
// pushdown isn't StarRocks-forever - the binding layer already
// enumerates POSTGRES/STARROCKS/SNOWFLAKE/ICEBERG/CRIMS as backend
// types, and a function's SQL expansion is genuinely dialect-specific
// (NPV's StarRocks expansion uses ROW_NUMBER() OVER (...), which reads
// differently in other dialects).

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Dialect identifies a SQL pushdown target. Only StarRocks exists today
// (the only binding backend_type this engine actually emits SQL for);
// adding a dialect is adding a value here plus SQLEmit entries on the
// functions that support it - never a new codepath.
type Dialect string

const DialectStarRocks Dialect = "starrocks"

// NativeImpl is a function's tree-walking implementation - the same
// code native server-side evaluation and the WASM browser build both
// run, since WASM is this package cross-compiled, not a separate
// implementation.
type NativeImpl func(args []any) (any, error)

// SQLEmitter renders a FuncCall's already-compiled argument SQL into a
// full SQL expression for one dialect.
type SQLEmitter func(args []string) (string, error)

// FunctionSpec is the one declaration a function needs. Category and
// Description exist for editor autocomplete/capability badges (see
// LibraryEntries and cmd/generate-monaco) - they're read, not decorative.
type FunctionSpec struct {
	Name string
	// Signature is a human-readable argument/return description, e.g.
	// "(rate number, cash_flows number[]) -> number" - not parsed by
	// anything, purely for editor autocomplete detail text and docs.
	Signature   string
	Category    string
	Description string
	// NoClosedForm marks a function with no algebraic solution (solved
	// numerically) - explains why Pushdownable is false for it even
	// though it's a real, fully-supported function, as opposed to a
	// function whose SQL emitter simply hasn't been written yet.
	NoClosedForm bool
	Native       NativeImpl
	SQLEmit      map[Dialect]SQLEmitter
}

// Pushdownable reports whether this function has an SQL emitter for the
// given dialect.
func (s *FunctionSpec) Pushdownable(d Dialect) bool {
	return s.SQLEmit != nil && s.SQLEmit[d] != nil
}

// Library is the one source of truth every consumer derives from:
// evalFuncCall (native/WASM evaluation), compileNodeToSQL (SQL
// pushdown), and - once built - the Monaco autocomplete generator and
// capability badges. Keyed by uppercase name, matching FuncCall.Name's
// existing case-insensitive convention.
var Library = buildLibrary()

// LibraryEntries returns the registered functions in a stable (name)
// order - for consumers that enumerate rather than look up by name
// (autocomplete, documentation, the registry self-consistency test).
func LibraryEntries() []*FunctionSpec {
	names := make([]string, 0, len(Library))
	for name := range Library {
		names = append(names, name)
	}
	// simple insertion sort - the library is small, and avoiding a
	// sort.Strings import keeps this file's import list to exactly what
	// the specs below need.
	for i := 1; i < len(names); i++ {
		for j := i; j > 0 && names[j-1] > names[j]; j-- {
			names[j-1], names[j] = names[j], names[j-1]
		}
	}
	out := make([]*FunctionSpec, len(names))
	for i, n := range names {
		out[i] = Library[n]
	}
	return out
}

// LookupFunction returns the spec for name (case-insensitive), or false
// if nothing is registered under it - e.g. because it's a composition
// (TVPI, DPI, MOIC) rather than a function at all; see library_test.go
// for the proof that compositions need no registry entry.
func LookupFunction(name string) (*FunctionSpec, bool) {
	s, ok := Library[strings.ToUpper(name)]
	return s, ok
}

func buildLibrary() map[string]*FunctionSpec {
	m := make(map[string]*FunctionSpec)
	reg := func(s *FunctionSpec) { m[s.Name] = s }

	reg(&FunctionSpec{
		Name: "SUM", Signature: "(values number[]) -> number", Category: "aggregation",
		Description: "Sum of a numeric field across grouped rows.",
		Native: func(args []any) (any, error) {
			return aggFold(args, 0, func(acc, v float64) float64 { return acc + v })
		},
		SQLEmit: map[Dialect]SQLEmitter{DialectStarRocks: aggPassthrough("SUM")},
	})
	reg(&FunctionSpec{
		Name: "AVG", Signature: "(values number[]) -> number", Category: "aggregation",
		Description: "Average of a numeric field across grouped rows.",
		Native: func(args []any) (any, error) {
			vals, err := requireFloatSlice(args)
			if err != nil {
				return nil, err
			}
			if len(vals) == 0 {
				return 0.0, nil
			}
			sum := 0.0
			for _, v := range vals {
				sum += v
			}
			return sum / float64(len(vals)), nil
		},
		SQLEmit: map[Dialect]SQLEmitter{DialectStarRocks: aggPassthrough("AVG")},
	})
	reg(&FunctionSpec{
		Name: "MIN", Signature: "(values number[]) -> number", Category: "aggregation",
		Description: "Minimum of a numeric field across grouped rows.",
		Native: func(args []any) (any, error) {
			vals, err := requireFloatSlice(args)
			if err != nil || len(vals) == 0 {
				return nil, err
			}
			m := vals[0]
			for _, v := range vals[1:] {
				if v < m {
					m = v
				}
			}
			return m, nil
		},
		SQLEmit: map[Dialect]SQLEmitter{DialectStarRocks: aggPassthrough("MIN")},
	})
	reg(&FunctionSpec{
		Name: "MAX", Signature: "(values number[]) -> number", Category: "aggregation",
		Description: "Maximum of a numeric field across grouped rows.",
		Native: func(args []any) (any, error) {
			vals, err := requireFloatSlice(args)
			if err != nil || len(vals) == 0 {
				return nil, err
			}
			m := vals[0]
			for _, v := range vals[1:] {
				if v > m {
					m = v
				}
			}
			return m, nil
		},
		SQLEmit: map[Dialect]SQLEmitter{DialectStarRocks: aggPassthrough("MAX")},
	})
	reg(&FunctionSpec{
		Name: "NPV", Signature: "(rate number, cash_flows number[]) -> number", Category: "financial",
		Description: "Net present value: SUM(cf_i / (1+rate)^i). No native StarRocks function - pushed down as a hand-expanded SQL expression using ROW_NUMBER().",
		Native: func(args []any) (any, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("NPV expects 2 args (rate, cash_flows), got %d", len(args))
			}
			rate, ok := args[0].(float64)
			if !ok {
				return nil, fmt.Errorf("NPV rate must be numeric, got %T", args[0])
			}
			cashFlows, err := requireFloatSlice(args[1:])
			if err != nil {
				return nil, err
			}
			npv := 0.0
			for i, cf := range cashFlows {
				npv += cf / math.Pow(1+rate, float64(i))
			}
			return npv, nil
		},
		SQLEmit: map[Dialect]SQLEmitter{
			DialectStarRocks: func(args []string) (string, error) {
				if len(args) != 2 {
					return "", fmt.Errorf("NPV expects 2 args (rate, cash_flow), got %d", len(args))
				}
				rate, cashFlow := args[0], args[1]
				return fmt.Sprintf(
					"SUM(%s / POWER(1 + %s, ROW_NUMBER() OVER (ORDER BY %s) - 1))",
					cashFlow, rate, cashFlow,
				), nil
			},
		},
	})
	reg(&FunctionSpec{
		Name: "IRR", Signature: "(cash_flows number[]) -> number", Category: "financial", NoClosedForm: true,
		Description: "Internal rate of return - the rate that makes NPV zero, over regularly-spaced (e.g. annual) cash flows. Solved numerically (Newton-Raphson, bisection fallback) - no algebraic solution exists, so this has no SQL pushdown.",
		Native: func(args []any) (any, error) {
			cashFlows, err := requireFloatSlice(args)
			if err != nil {
				return nil, err
			}
			return solveIRR(cashFlows, integerPeriods(len(cashFlows)))
		},
	})
	reg(&FunctionSpec{
		Name: "XIRR", Signature: "(cash_flows number[], dates number[]) -> number", Category: "financial", NoClosedForm: true,
		Description: "IRR for irregularly-dated cash flows (actual/365 day-count, matching Excel's XIRR). Same numerical solver as IRR, different period calculation. No SQL pushdown, for the same reason as IRR.",
		Native: func(args []any) (any, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("XIRR expects 2 args (cash_flows, dates), got %d", len(args))
			}
			cashFlows, err := requireFloatSlice(args[0:1])
			if err != nil {
				return nil, fmt.Errorf("XIRR cash_flows: %w", err)
			}
			days, err := requireFloatSlice(args[1:2])
			if err != nil {
				return nil, fmt.Errorf("XIRR dates: %w", err)
			}
			if len(days) != len(cashFlows) {
				return nil, fmt.Errorf("XIRR cash_flows and dates must be the same length (%d vs %d)", len(cashFlows), len(days))
			}
			return solveIRR(cashFlows, dayPeriods(days))
		},
	})
	reg(&FunctionSpec{
		Name: "MIRR", Signature: "(cash_flows number[], finance_rate number, reinvest_rate number) -> number", Category: "financial",
		Description: "Modified IRR: (FV(positive cash flows, reinvest_rate) / -PV(negative cash flows, finance_rate))^(1/n) - 1. Unlike IRR/XIRR this IS closed-form (no iterative solver) - Pushdownable is still false here, but only because the SQL emitter (a conditional FV/PV expansion over grouped rows, more involved than NPV's) hasn't been written and verified against a live table yet, not because it's impossible. See docs/unified-rule-engine-handoff.md.",
		Native: func(args []any) (any, error) {
			if len(args) != 3 {
				return nil, fmt.Errorf("MIRR expects 3 args (cash_flows, finance_rate, reinvest_rate), got %d", len(args))
			}
			cashFlows, err := requireFloatSlice(args[0:1])
			if err != nil {
				return nil, fmt.Errorf("MIRR cash_flows: %w", err)
			}
			financeRate, ok := args[1].(float64)
			if !ok {
				return nil, fmt.Errorf("MIRR finance_rate must be numeric, got %T", args[1])
			}
			reinvestRate, ok := args[2].(float64)
			if !ok {
				return nil, fmt.Errorf("MIRR reinvest_rate must be numeric, got %T", args[2])
			}
			return solveMIRR(cashFlows, financeRate, reinvestRate)
		},
	})

	// Field-format predicates, added for the catalog_validation_rules ->
	// rule_ast migration (backend/cmd/migrate_validation_rules). None
	// are pushdownable - format/type validation doesn't map to a
	// StarRocks SQL expression the way an aggregate does. All treat a
	// present-but-JSON-null field value as failing the predicate (false,
	// no error) rather than a type error - a genuinely absent field is a
	// separate case, already an error from evalFieldRef before these
	// ever run.
	reg(&FunctionSpec{
		Name: "NOT_EMPTY", Signature: "(value any) -> boolean", Category: "format",
		Description: "True if the field is present and non-empty.",
		Native: func(args []any) (any, error) {
			v, err := require1(args, "NOT_EMPTY")
			if err != nil {
				return nil, err
			}
			if v == nil {
				return false, nil
			}
			s, ok := v.(string)
			return !ok || s != "", nil
		},
	})
	reg(&FunctionSpec{
		Name: "IS_INTEGER", Signature: "(value any) -> boolean", Category: "format",
		Description: "True if the field's value is an integer.",
		Native: func(args []any) (any, error) {
			v, err := require1(args, "IS_INTEGER")
			if err != nil {
				return nil, err
			}
			switch n := v.(type) {
			case int, int32, int64:
				return true, nil
			case float64:
				return n == math.Trunc(n), nil
			case string:
				_, err := strconv.ParseInt(n, 10, 64)
				return err == nil, nil
			default:
				return false, nil
			}
		},
	})
	reg(&FunctionSpec{
		Name: "IS_NUMBER", Signature: "(value any) -> boolean", Category: "format",
		Description: "True if the field's value is numeric.",
		Native: func(args []any) (any, error) {
			v, err := require1(args, "IS_NUMBER")
			if err != nil {
				return nil, err
			}
			switch n := v.(type) {
			case int, int32, int64, float32, float64:
				return true, nil
			case string:
				_, err := strconv.ParseFloat(n, 64)
				return err == nil, nil
			default:
				return false, nil
			}
		},
	})
	reg(&FunctionSpec{
		Name: "IS_BOOLEAN", Signature: "(value any) -> boolean", Category: "format",
		Description: "True if the field's value is a boolean.",
		Native: func(args []any) (any, error) {
			v, err := require1(args, "IS_BOOLEAN")
			if err != nil {
				return nil, err
			}
			switch b := v.(type) {
			case bool:
				return true, nil
			case string:
				return b == "true" || b == "false", nil
			default:
				return false, nil
			}
		},
	})
	reg(&FunctionSpec{
		Name: "IS_UUID", Signature: "(value any) -> boolean", Category: "format",
		Description: "True if the field's value is a UUID.",
		Native: func(args []any) (any, error) {
			v, err := require1(args, "IS_UUID")
			if err != nil {
				return nil, err
			}
			s, ok := v.(string)
			if !ok {
				return false, nil
			}
			return uuidPattern.MatchString(s), nil
		},
	})
	reg(&FunctionSpec{
		Name: "IS_DATE", Signature: "(value any) -> boolean", Category: "format",
		Description: "True if the field's value is a date (YYYY-MM-DD).",
		Native: func(args []any) (any, error) {
			v, err := require1(args, "IS_DATE")
			if err != nil {
				return nil, err
			}
			s, ok := v.(string)
			if !ok {
				return false, nil
			}
			_, parseErr := time.Parse("2006-01-02", s)
			return parseErr == nil, nil
		},
	})
	reg(&FunctionSpec{
		Name: "IS_DATETIME", Signature: "(value any) -> boolean", Category: "format",
		Description: "True if the field's value is an RFC3339 datetime.",
		Native: func(args []any) (any, error) {
			v, err := require1(args, "IS_DATETIME")
			if err != nil {
				return nil, err
			}
			s, ok := v.(string)
			if !ok {
				return false, nil
			}
			_, parseErr := time.Parse(time.RFC3339, s)
			return parseErr == nil, nil
		},
	})
	reg(&FunctionSpec{
		Name: "IS_JSON", Signature: "(value any) -> boolean", Category: "format",
		Description: "True if the field's value is valid JSON.",
		Native: func(args []any) (any, error) {
			v, err := require1(args, "IS_JSON")
			if err != nil {
				return nil, err
			}
			s, ok := v.(string)
			if !ok {
				return false, nil
			}
			return json.Valid([]byte(s)), nil
		},
	})
	reg(&FunctionSpec{
		Name: "MAX_LENGTH", Signature: "(value any, max number) -> boolean", Category: "format",
		Description: "True if the field's string length is within a maximum.",
		Native: func(args []any) (any, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("MAX_LENGTH expects 2 args (field, max), got %d", len(args))
			}
			max, ok := args[1].(float64)
			if !ok {
				return nil, fmt.Errorf("MAX_LENGTH's second arg must be numeric, got %T", args[1])
			}
			if args[0] == nil {
				return true, nil
			}
			s, ok := args[0].(string)
			if !ok {
				return false, nil
			}
			return float64(len(s)) <= max, nil
		},
	})

	return m
}

// require1 validates a predicate received exactly one argument and returns
// it, nil-safe (a resolved FieldRef for an absent/null field comes through
// as a nil any, which is a valid input to these predicates, not an error).
func require1(args []any, fnName string) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("%s expects 1 arg, got %d", fnName, len(args))
	}
	return args[0], nil
}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func aggPassthrough(sqlName string) SQLEmitter {
	return func(args []string) (string, error) {
		if len(args) != 1 {
			return "", fmt.Errorf("%s expects 1 arg, got %d", sqlName, len(args))
		}
		return fmt.Sprintf("%s(%s)", sqlName, args[0]), nil
	}
}
