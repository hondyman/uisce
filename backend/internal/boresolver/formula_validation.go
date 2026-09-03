package boresolver

import (
	"fmt"
	"strings"
)

// FormulaError is one problem found in a tenant-authored formula, with the
// position (in the ORIGINAL "${term}"-syntax string the user typed) where
// it should be highlighted — the whole point of a formula-bar editor is
// pointing at the exact offending token, not just showing a generic
// message under the box.
//
// Position mapping is byte-offset based: since "${term}" reference names
// are restricted to [a-zA-Z0-9_]+ (calcTermRefRegex), byte and rune offsets
// coincide for every formula this maps correctly. A formula containing
// non-ASCII string literals could see the position drift for tokens after
// the first one — acceptable for now, worth a real rune-based remap if
// that turns out to matter in practice.
type FormulaError struct {
	Message string `json:"message"`
	Pos     int    `json:"pos"`
}

// FormulaValidation is the response shape for a formula-bar "as you type"
// validation call.
type FormulaValidation struct {
	Valid           bool           `json:"valid"`
	Tier            string         `json:"tier,omitempty"` // "pushdown" | "host_runtime"
	ReferencedTerms []string       `json:"referenced_terms,omitempty"`
	FunctionsUsed   []string       `json:"functions_used,omitempty"`
	Errors          []FormulaError `json:"errors,omitempty"`
}

// ValidateFormula parses a tenant-authored formula (Excel-style, using
// "${term}" references) and reports whether it's valid, which terms and
// functions it references, and which execution tier it would resolve to —
// using the SAME parser and ResolveTier logic CompileDeepCalculations uses
// at compile time, so what a formula-bar UI shows the user is never out of
// sync with what actually runs when the calc is saved.
//
// dialect and pref decide tier the same way a real compile would (pass the
// tenant's target dialect and the calc's ExecutionPreference); pass nil
// dialect for PostgresDialect{}. knownTerms, if non-nil, is checked against
// every referenced term so a typo'd or unpublished term name is flagged
// immediately instead of failing later at compile time.
func ValidateFormula(formula string, dialect Dialect, pref ExecutionPreference, knownTerms map[string]bool) FormulaValidation {
	rewritten, origPos := rewriteCalcFormula(formula)

	expr, err := ParseExpression(rewritten)
	if err != nil {
		pos := 0
		if pe, ok := err.(*ParseError); ok {
			pos = toOriginalPos(origPos, pe.Pos)
		}
		return FormulaValidation{
			Valid:  false,
			Errors: []FormulaError{{Message: err.Error(), Pos: pos}},
		}
	}

	if dialect == nil {
		dialect = PostgresDialect{}
	}

	var errs []FormulaError

	refs := collectTermRefs(expr)
	seenRef := make(map[string]bool, len(refs))
	var uniqueRefs []string
	for _, ref := range refs {
		if seenRef[ref] {
			continue
		}
		seenRef[ref] = true
		uniqueRefs = append(uniqueRefs, ref)
		if knownTerms != nil && !knownTerms[ref] {
			errs = append(errs, FormulaError{Message: fmt.Sprintf("unknown term: %s", ref)})
		}
	}

	tier, tierErr := ResolveTier(expr, dialect, pref)
	if tierErr != nil {
		errs = append(errs, FormulaError{Message: tierErr.Error()})
	}

	return FormulaValidation{
		Valid:           len(errs) == 0,
		Tier:            tier.String(),
		ReferencedTerms: uniqueRefs,
		FunctionsUsed:   collectFunctionNames(expr),
		Errors:          errs,
	}
}

func collectFunctionNames(expr Expr) []string {
	var out []string
	seen := make(map[string]bool)
	var walk func(Expr)
	walk = func(e Expr) {
		switch n := e.(type) {
		case *FunctionCall:
			if !seen[n.FunctionName] {
				seen[n.FunctionName] = true
				out = append(out, n.FunctionName)
			}
			for _, arg := range n.Args {
				walk(arg)
			}
		case *BinaryExpr:
			walk(n.Left)
			walk(n.Right)
		case *UnaryExpr:
			walk(n.Expr)
		}
	}
	walk(expr)
	return out
}

// rewriteCalcFormula rewrites "${term}" references to bare identifiers for
// the expression parser (same substitution ParseCalcFormula does), and
// additionally returns origPos, a rewritten-position -> original-position
// lookup table so parse-error positions can be reported against the
// formula the user actually typed rather than the internal rewritten form.
func rewriteCalcFormula(formula string) (rewritten string, origPos []int) {
	matches := calcTermRefRegex.FindAllStringSubmatchIndex(formula, -1)

	var sb strings.Builder
	appendVerbatim := func(from, to int) {
		for i := from; i < to; i++ {
			origPos = append(origPos, i)
		}
		sb.WriteString(formula[from:to])
	}

	lastOrig := 0
	for _, m := range matches {
		matchStart, matchEnd := m[0], m[1]
		nameStart, nameEnd := m[2], m[3]
		appendVerbatim(lastOrig, matchStart)
		appendVerbatim(nameStart, nameEnd)
		lastOrig = matchEnd
	}
	appendVerbatim(lastOrig, len(formula))
	origPos = append(origPos, len(formula)) // sentinel: EOF position

	return sb.String(), origPos
}

func toOriginalPos(origPos []int, rewrittenPos int) int {
	if rewrittenPos < 0 {
		return 0
	}
	if rewrittenPos >= len(origPos) {
		if len(origPos) == 0 {
			return 0
		}
		return origPos[len(origPos)-1]
	}
	return origPos[rewrittenPos]
}
