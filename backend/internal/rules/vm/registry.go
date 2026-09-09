package vm

// FunctionCapability describes one FuncCall the registry knows about -
// what it does and which compile targets actually support it. Built for
// the editor's capability badges and autocomplete (parked as an explicit
// follow-up - see docs/unified-rule-engine-handoff.md - this is the data
// layer they'd read from, not the UI itself) and, more immediately, as a
// single place that states what this file's own comments have had to
// repeat inline until now: which functions are pushdownable to
// StarRocks SQL, which only run at tree-walking speed (native/WASM),
// and why.
//
// Deliberately NOT a place to register TVPI/DPI/MOIC or similar ratio
// terms - those compose from SUM (already in this registry) via plain
// division (BinaryExpr "/"), which every compile target already
// supports. Adding them here would suggest they need dedicated function
// support they don't - see registry_test.go for the composition proof.
type FunctionCapability struct {
	Name         string
	Description  string
	Pushdownable bool // has a starrocksFuncs entry (sql_compiler.go)
	Native       bool // has a nativeFuncs entry (advanced_evaluator.go) - tree-walking, server-side
	WASM         bool // reachable from the browser build (same source as Native - true whenever Native is)
	NoClosedForm bool // true for functions with no algebraic solution (numerically solved) - explains why Pushdownable is false despite being a real, supported function
}

// Registry is the capability list itself - built by hand to match what
// sql_compiler.go's starrocksFuncs and advanced_evaluator.go's
// nativeFuncs actually contain (registry_test.go cross-checks this
// against both maps directly, so the two can't silently drift apart).
var Registry = []FunctionCapability{
	{Name: "SUM", Description: "Sum of a numeric field across grouped rows.", Pushdownable: true, Native: true, WASM: true},
	{Name: "AVG", Description: "Average of a numeric field across grouped rows.", Pushdownable: true, Native: true, WASM: true},
	{Name: "MIN", Description: "Minimum of a numeric field across grouped rows.", Pushdownable: true, Native: true, WASM: true},
	{Name: "MAX", Description: "Maximum of a numeric field across grouped rows.", Pushdownable: true, Native: true, WASM: true},
	{Name: "NPV", Description: "Net present value: SUM(cf_i / (1+rate)^i). No native StarRocks function - pushed down as a hand-expanded SQL expression.", Pushdownable: true, Native: true, WASM: true},
	{Name: "IRR", Description: "Internal rate of return - the rate that makes NPV zero, over regularly-spaced (e.g. annual) cash flows. Solved numerically (Newton-Raphson, bisection fallback) - no algebraic solution exists.", Pushdownable: false, Native: true, WASM: true, NoClosedForm: true},
	{Name: "XIRR", Description: "IRR for irregularly-dated cash flows (actual/365 day-count, matching Excel's XIRR). Same numerical solver as IRR, different period calculation.", Pushdownable: false, Native: true, WASM: true, NoClosedForm: true},
	{Name: "NOT_EMPTY", Description: "True if the field is present and non-empty.", Pushdownable: false, Native: true, WASM: true},
	{Name: "IS_INTEGER", Description: "Format predicate: true if the field's value is an integer.", Pushdownable: false, Native: true, WASM: true},
	{Name: "IS_NUMBER", Description: "Format predicate: true if the field's value is numeric.", Pushdownable: false, Native: true, WASM: true},
	{Name: "IS_BOOLEAN", Description: "Format predicate: true if the field's value is a boolean.", Pushdownable: false, Native: true, WASM: true},
	{Name: "IS_UUID", Description: "Format predicate: true if the field's value is a UUID.", Pushdownable: false, Native: true, WASM: true},
	{Name: "IS_DATE", Description: "Format predicate: true if the field's value is a date.", Pushdownable: false, Native: true, WASM: true},
	{Name: "IS_DATETIME", Description: "Format predicate: true if the field's value is a datetime.", Pushdownable: false, Native: true, WASM: true},
	{Name: "IS_JSON", Description: "Format predicate: true if the field's value is valid JSON.", Pushdownable: false, Native: true, WASM: true},
	{Name: "MAX_LENGTH", Description: "Format predicate: true if the field's string length is within a maximum.", Pushdownable: false, Native: true, WASM: true},
}

// LookupCapability returns the registry entry for name (case-sensitive,
// matching FuncCall.Name's convention), or false if the function isn't
// registered - e.g. because it's a composition (TVPI, DPI, MOIC) rather
// than a function at all.
func LookupCapability(name string) (FunctionCapability, bool) {
	for _, c := range Registry {
		if c.Name == name {
			return c, true
		}
	}
	return FunctionCapability{}, false
}
