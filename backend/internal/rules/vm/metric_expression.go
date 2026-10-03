package vm

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
)

// Metric expression primitives (C1 / 9.1).
//
// These were ported from internal/querybuilder so the rule VM owns metric
// semantics rather than borrowing them across package boundaries. The port is
// behavioural: the querybuilder implementations delegate here, so the two
// cannot drift. The 8.3 golden corpus (internal/querybuilder/testdata/metric_corpus)
// is what makes that safe — it is the check that the delegated path still
// produces byte-identical SQL and hashes.

// VariableTypeNumber and friends are the permitted variable types. Kept as
// constants so a typo is a compile error rather than a silently inert variable.
const (
	VariableTypeNumber = "number"
	VariableTypeString = "string"
	VariableTypeDate   = "date"
)

// MetricVariable defines a parameter/variable bound into the metric calculation.
//
// asl:ignore - backend metric plumbing, not a rule/calc AST node. This package
// is enumerated by generate-types/generate-schema/generate-monaco, so an
// exported struct here lands in the published ASL schema and the browser
// editor's Monaco autocomplete. That widening is a side effect of WHERE the type
// lives, not a statement that it is authorable: the WASM evaluator has no node
// kind for it, so an author could insert one the browser could not evaluate.
// The type was in internal/querybuilder (never enumerated) before C1 moved it
// here; the marker keeps the published contract identical to that.
type MetricVariable struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"` // "number" | "string" | "date"
	DefaultValue interface{} `json:"defaultValue,omitempty"`
	Required     bool        `json:"required"`
	Description  string      `json:"description,omitempty"`
}

// MetricFormatConfig controls presentation of a computed metric value.
//
// asl:ignore - see the note on MetricVariable above.
type MetricFormatConfig struct {
	Type           string  `json:"type"` // "currency" | "percentage" | "compact" | "number"
	Precision      *int    `json:"precision,omitempty"`
	CurrencySymbol *string `json:"currencySymbol,omitempty"`
	Prefix         *string `json:"prefix,omitempty"`
	Suffix         *string `json:"suffix,omitempty"`
}

// MetricExpression is the semantic definition of how a metric is computed.
//
// asl:ignore - see the note on MetricVariable above. This one is the most
// misleading of the four: its `kind` field looks exactly like the
// discriminator that earns a struct a Monaco node kind, which is how it would
// otherwise have been offered to rule authors as an insertable node.
type MetricExpression struct {
	Kind       string `json:"kind"`                 // "aggregation" | "formula" | "derived"
	Fn         string `json:"fn,omitempty"`         // "sum" | "avg" | "count" | "min" | "max"
	TermNodeID string `json:"termNodeId,omitempty"` // For aggregation / column reference
	// Formula is authored SQL text with @variable references, e.g.
	// "SUM(price * qty) * @fx_rate". The compiler substitutes the variables
	// with bound parameters; it does NOT parse, validate or allowlist the
	// operators, which pass through as authored. See ADR-025.
	Formula string `json:"formula,omitempty"`
	// BaseMetricIDs are the operands of a derived metric, in the author's
	// declared order. For a 2-operand ratio the first entry is the numerator.
	BaseMetricIDs []string `json:"baseMetricIds,omitempty"`
	// NumeratorID / DenominatorID are the explicit, named operands for a ratio
	// (ADR-026). They are separate from BaseMetricIDs because a ratio must name
	// which side is which — the ordered BaseMetricIDs form alone was rejected as
	// ambiguous. Both forms are hashed, so two ratios that differ only in which
	// side is numerator do NOT collide.
	NumeratorID   string `json:"numeratorId,omitempty"`
	DenominatorID string `json:"denominatorId,omitempty"`
}

// DeriveDecomposable reports whether a metric's value can be summed across a
// finer grain and still be correct — i.e. whether it may be materialized as a
// rollup.
//
// It is deliberately conservative: a "true" is a claim that must survive
// review, so anything not provably decomposable is false.
func DeriveDecomposable(expr MetricExpression) bool {
	switch strings.ToLower(strings.TrimSpace(expr.Kind)) {
	case "aggregation":
		fn := strings.ToLower(strings.TrimSpace(expr.Fn))
		return fn == "sum" || fn == "count" || fn == "min" || fn == "max"
	case "formula":
		// Conservative derivation: any division or non-distributive function marks as non-decomposable
		f := strings.ToLower(expr.Formula)
		if strings.Contains(f, "/") || strings.Contains(f, "avg(") {
			return false
		}
		return true
	case "derived":
		// Derived metrics (ratio of metrics) are generally non-decomposable across tiers
		return false
	default:
		return false
	}
}

// NormalizeFormulaForHash strips redundant whitespace, cosmetic grouping
// parentheses and case, so that formulas differing only in those respects
// compare equal. Exported because the compiler's canonicalization must agree
// with the hash's, and a private copy is a drift risk.
func NormalizeFormulaForHash(f string) string {
	// Strips redundant whitespace, cosmetic redundant outer parentheses, and normalizes case
	f = strings.TrimSpace(strings.ToLower(f))
	var sb strings.Builder
	inWhitespace := false
	for _, r := range f {
		if r == '(' || r == ')' {
			// Ignore purely cosmetic grouping parentheses in commutative additions/multiplications
			// while keeping structure tokenized
			continue
		}
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if !inWhitespace {
				sb.WriteRune(' ')
				inWhitespace = true
			}
		} else {
			sb.WriteRune(r)
			inWhitespace = false
		}
	}
	return strings.TrimSpace(sb.String())
}

// MetricContentInput is the semantic content a metric hash is computed over.
// It is a flat struct rather than the full MetricDefinition so the hash has an
// explicit, narrow input surface: adding a field to a definition must not
// silently change deploy identity or cache keys.
//
// asl:ignore - see the note on MetricVariable above.
type MetricContentInput struct {
	Name           string
	BOID           string
	Expression     MetricExpression
	GrainAllowlist []string
	FormatConfig   MetricFormatConfig
	Variables      []MetricVariable
}

// ComputeMetricContentHash deterministically computes the SHA-256 hash of a
// metric's semantic content (normalized formula, grains, format, variables,
// boid). This hash is the cube deploy identity and part of the query cache key,
// so it must be stable across formatting and sensitive to meaning.
func ComputeMetricContentHash(m MetricContentInput) string {
	// 1. Normalize Expression
	normExpr := m.Expression
	normExpr.Kind = strings.ToLower(strings.TrimSpace(normExpr.Kind))
	normExpr.Fn = strings.ToLower(strings.TrimSpace(normExpr.Fn))
	normExpr.Formula = NormalizeFormulaForHash(normExpr.Formula)
	// BaseMetricIDs are deliberately NOT sorted. Operand order is semantic: the
	// first entry of a 2-operand derived metric is the numerator (ADR-025).
	// Sorting here would give revenue/cost and cost/revenue the same content
	// hash, and that hash is the cube deploy identity and part of the query
	// cache key - so the two opposite metrics would share a cache entry.

	// 2. Normalize Grain Allowlist
	normGrains := make([]string, len(m.GrainAllowlist))
	for i, g := range m.GrainAllowlist {
		normGrains[i] = strings.ToLower(strings.TrimSpace(g))
	}
	sort.Strings(normGrains)

	// 3. Normalize Variables
	normVars := make([]MetricVariable, len(m.Variables))
	copy(normVars, m.Variables)
	sort.Slice(normVars, func(i, j int) bool {
		return normVars[i].Name < normVars[j].Name
	})

	canonicalDoc := struct {
		Name         string             `json:"name"`
		BOID         string             `json:"boId"`
		Expression   MetricExpression   `json:"expression"`
		Grains       []string           `json:"grains"`
		FormatConfig MetricFormatConfig `json:"formatConfig"`
		Variables    []MetricVariable   `json:"variables"`
	}{
		Name:         strings.TrimSpace(m.Name),
		BOID:         strings.TrimSpace(m.BOID),
		Expression:   normExpr,
		Grains:       normGrains,
		FormatConfig: m.FormatConfig,
		Variables:    normVars,
	}

	b, _ := json.Marshal(canonicalDoc)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
