package querybuilder

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/hondyman/uisce/backend/internal/rules/vm"
)

// MetricTermGate decides whether a metric may read a term, given the field that
// term resolves to. It is the metric-path equivalent of the BO path's masking
// check (boresolver's resolveCol closure), which has always run for calc terms
// and did not exist here.
//
// field is nil when the term does not resolve to a field in the supplied BO,
// which is itself a refusal: a metric may not name a column the semantic layer
// does not know about.
type MetricTermGate func(termNodeID string, field *boresolver.BOField) error

// NewSensitivityTermGate returns the standard gate: a term is permitted only if
// every column it can reach is passthrough for this caller's role and clearance.
//
// It recurses into calc terms, which is the substance of the BO path's rule
// rather than an addition to it. The BO predicate at
// boresolver/bo_sql_generator.go:781-787 fires INSIDE a calc term's expression,
// on the sensitivity tag of the field that expression references. A gate that
// only inspected the term a metric names would therefore miss precisely the case
// the BO rule exists for: a calc term carries no PhysicalColumn and usually no
// SensitivityTag of its own, so it would pass, and the PII would be read one
// level down. Verified before it was fixed - see the ADR-024 C2 entry.
//
// The recursion carries the same two protections the BO resolver's resolveCol
// closure carries: a cycle guard (a calc term reachable from itself) and a depth
// cap, so a long or cyclic chain of calc terms cannot walk unboundedly.
//
// calcTerms maps a calc term's SemanticTermID to its compiled expression, the
// same shape as boresolver.GenerationContext.CalcTermConfigs. A nil map means no
// calc term can be inspected, and a metric naming one is then REFUSED rather than
// permitted: an uninspectable chain is not a clean chain.
func NewSensitivityTermGate(boDef *boresolver.BODefinition, userRole, clearanceLevel string, calcTerms map[string]*vm.Expression) MetricTermGate {
	var walk func(termNodeID string, seen map[string]bool, depth int) error
	walk = func(termNodeID string, seen map[string]bool, depth int) error {
		if boDef == nil {
			return fmt.Errorf("%w: no business object supplied to resolve term %q against", ErrMetricTermNotPermitted, termNodeID)
		}
		field, err := resolveTermToField(boDef, termNodeID)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrMetricTermNotPermitted, err)
		}

		if field.TermType == "calculated" {
			if seen[field.Name] {
				return fmt.Errorf("%w: cycle detected: calc term %q is reachable from itself", ErrMetricTermNotPermitted, field.Name)
			}
			if depth >= maxMetricCalcTermDepth {
				return fmt.Errorf("%w: calc term %q exceeds max chaining depth (%d)", ErrMetricTermNotPermitted, field.Name, maxMetricCalcTermDepth)
			}
			expr, ok := calcTerms[field.SemanticTermID]
			if !ok || expr == nil {
				return fmt.Errorf("%w: calc term %q has no preloaded expression to inspect", ErrMetricTermNotPermitted, field.Name)
			}
			seen[field.Name] = true
			defer delete(seen, field.Name)
			for _, ref := range collectFieldRefs(expr.Root) {
				if err := walk(ref, seen, depth+1); err != nil {
					return err
				}
			}
			return nil
		}

		if field.SensitivityTag == "" {
			return nil
		}
		tier := boresolver.DetermineMaskingTier(field.SensitivityTag, userRole, clearanceLevel)
		if tier != boresolver.MaskingTierPassthrough {
			return fmt.Errorf("%w: term %q resolves to %q tagged %q, tier %s at role %q/clearance %q",
				ErrMetricTermNotPermitted, termNodeID, field.PhysicalColumn, field.SensitivityTag, tier, userRole, clearanceLevel)
		}
		return nil
	}
	return func(termNodeID string, _ *boresolver.BOField) error {
		return walk(termNodeID, map[string]bool{}, 0)
	}
}

// maxMetricCalcTermDepth caps how many levels of calc-term chaining the gate
// will walk. It matches the BO resolver's maxCalcTermDepth so the two paths
// refuse the same chains: a metric may not reach further into calc terms than a
// calc term may.
const maxMetricCalcTermDepth = 1

// collectFieldRefs returns every field path an expression references, in
// walk order. The AST is a closed set of four node types (BinaryExpr, FieldRef,
// Literal, FuncCall), so this is exhaustive by construction rather than by
// default - a fifth node type would fail to compile here rather than be
// silently skipped, which is the failure direction that matters for a security
// walk: an unvisited branch is an unchecked branch.
func collectFieldRefs(node vm.ExprNode) []string {
	switch n := node.(type) {
	case nil:
		return nil
	case *vm.FieldRef:
		return []string{n.Path}
	case *vm.BinaryExpr:
		return append(collectFieldRefs(n.Left), collectFieldRefs(n.Right)...)
	case *vm.FuncCall:
		var out []string
		for _, a := range n.Args {
			out = append(out, collectFieldRefs(a)...)
		}
		return out
	case *vm.Literal:
		return nil
	default:
		// Unreachable while the AST is the four types above; a new node type
		// must be handled here or this walk would silently under-report.
		panic(fmt.Sprintf("collectFieldRefs: unhandled vm node type %T - a security walk must not skip it", node))
	}
}

// MetricCompiler compiles metric definitions and AST expressions into safe SQL expressions and arguments.
type MetricCompiler struct {
	dialect boresolver.Dialect
	// termGate is the C2 PII gate. It is consulted for every term a metric
	// reads. A nil gate means "no classification is available for this
	// compilation", which is why CubeDDLGenerator refuses to run without one:
	// emitting physical SQL from an ungated metric is how a PII column gets
	// materialized. Compile-only callers (the golden corpus, the equivalence
	// suite) legitimately have no BO and are not affected.
	termGate MetricTermGate
}

// NewMetricCompiler creates a new metric compiler instance.
func NewMetricCompiler(dialect boresolver.Dialect) *MetricCompiler {
	if dialect == nil {
		dialect = boresolver.PostgresDialect{}
	}
	return &MetricCompiler{dialect: dialect}
}

// WithTermGate installs the PII gate and returns the compiler, so a gate cannot
// be forgotten at a call site that then looks configured.
func (mc *MetricCompiler) WithTermGate(gate MetricTermGate) *MetricCompiler {
	mc.termGate = gate
	return mc
}

// checkTerm applies the gate if one is installed. A nil gate is a no-op here by
// design; the fail-closed decision belongs to the DDL generator, which is the
// only caller that turns compiled output into a physical artifact.
func (mc *MetricCompiler) checkTerm(termNodeID string) error {
	if mc.termGate == nil {
		return nil
	}
	return mc.termGate(termNodeID, nil)
}

// CompiledMetricResult contains the SQL expression and parameterized arguments.
type CompiledMetricResult struct {
	SQLExpr     string
	Args        []interface{}
	ContentHash string
}

// CompileMetric compiles a MetricDefinition into SQL expression and parameters,
// resolving any runtime variables against provided variable bindings.
func (mc *MetricCompiler) CompileMetric(m MetricDefinition, varBindings map[string]interface{}, metricLookup map[string]MetricDefinition) (*CompiledMetricResult, error) {
	visited := make(map[string]bool)
	return mc.compileMetricWithCycleDetection(m, varBindings, metricLookup, visited)
}

func (mc *MetricCompiler) compileMetricWithCycleDetection(
	m MetricDefinition,
	varBindings map[string]interface{},
	metricLookup map[string]MetricDefinition,
	visited map[string]bool,
) (*CompiledMetricResult, error) {
	if visited[m.ID] {
		return nil, fmt.Errorf("%w: metric %s (%s)", ErrCyclicMetricRef, m.ID, m.Name)
	}
	visited[m.ID] = true
	defer delete(visited, m.ID)

	var args []interface{}
	var sqlExpr string

	switch strings.ToLower(strings.TrimSpace(m.Expression.Kind)) {
	case "aggregation":
		fn := strings.ToUpper(strings.TrimSpace(m.Expression.Fn))
		if fn != "SUM" && fn != "AVG" && fn != "COUNT" && fn != "MIN" && fn != "MAX" {
			return nil, fmt.Errorf("%w: unsupported aggregation function %q", ErrInvalidMetricFormula, m.Expression.Fn)
		}
		termID := m.Expression.TermNodeID
		if termID == "" {
			return nil, fmt.Errorf("%w: aggregation metric requires termNodeId", ErrInvalidMetricFormula)
		}
		// C2 PII gate. This is the only point in the metric compiler where an
		// authored term becomes a physical column name, so it is the only point
		// at which the term has to be cleared for reading. It runs before the
		// identifier is sanitized into SQL rather than after, so a refused term
		// never reaches the emitted string at all.
		if err := mc.checkTerm(termID); err != nil {
			return nil, fmt.Errorf("metric %s (%s): %w", m.ID, m.Name, err)
		}
		// Render aggregation column: e.g. SUM(t0.price)
		colRef := fmt.Sprintf("t0.%s", sanitizeIdentifier(termID))
		sqlExpr = fmt.Sprintf("%s(%s)", fn, colRef)

	case "formula":
		// Formula compilation: @var references are substituted with bound
		// parameters. Operators are NOT parsed, validated or allowlisted - they
		// pass through to SQL as authored. See ADR-025.
		compiled, formArgs, err := mc.compileFormula(m.Expression.Formula, m.Variables, varBindings)
		if err != nil {
			return nil, err
		}
		sqlExpr = compiled
		args = append(args, formArgs...)

	case "derived":
		// One validator, shared with the cube validation path, so the compiler
		// and the authoring path can never disagree about what a derived metric
		// is allowed to be. See ADR-026.
		if err := ValidateMetricExpression(m); err != nil {
			return nil, err
		}

		// Operand order. A 2-operand derived metric is a ratio and MUST name its
		// numerator and denominator: reading the direction off positional order
		// is what inverted revenue/cost, and guessing at it is how a named
		// business metric silently became its reciprocal. Only the N-operand sum
		// - where order cannot change the value - is sorted, which keeps its
		// output deterministic.
		ids := make([]string, len(m.Expression.BaseMetricIDs))
		copy(ids, m.Expression.BaseMetricIDs)
		if len(ids) == 2 {
			ids = []string{m.Expression.NumeratorID, m.Expression.DenominatorID}
		} else {
			sort.Strings(ids)
		}

		var baseSubExprs []string
		for _, baseID := range ids {
			baseMetric, exists := metricLookup[baseID]
			if !exists {
				return nil, fmt.Errorf("%w: base metric %s not found", ErrMissingMetricDep, baseID)
			}
			compiledBase, err := mc.compileMetricWithCycleDetection(baseMetric, varBindings, metricLookup, visited)
			if err != nil {
				return nil, err
			}
			baseSubExprs = append(baseSubExprs, fmt.Sprintf("(%s)", compiledBase.SQLExpr))
			args = append(args, compiledBase.Args...)
		}

		if len(baseSubExprs) == 2 {
			// Explicitly a ratio: baseSubExprs[0] is the numerator by
			// construction now, not by alphabetical accident.
			sqlExpr = fmt.Sprintf("%s / NULLIF(%s, 0)", baseSubExprs[0], baseSubExprs[1])
		} else {
			sqlExpr = strings.Join(baseSubExprs, " + ")
		}

	default:
		return nil, fmt.Errorf("%w: unsupported expression kind %q", ErrInvalidMetricFormula, m.Expression.Kind)
	}

	contentHash := m.ContentHash
	if contentHash == "" {
		contentHash = ComputeMetricContentHash(m)
	}

	return &CompiledMetricResult{
		SQLExpr:     sqlExpr,
		Args:        args,
		ContentHash: contentHash,
	}, nil
}

// compileFormula substitutes @variable references in a formula string (e.g.
// "(SUM(price * quantity)) * @fx_rate") with positional parameters and returns
// the remaining text unchanged.
//
// It does not parse, validate or allowlist arithmetic, despite what this
// function's comment used to claim. A formula is authored, governed input: its
// operators are the author's responsibility, and the only allowlist in this
// compiler is the one on aggregation functions in CompileMetric. See ADR-025.
func (mc *MetricCompiler) compileFormula(formula string, variables []MetricVariable, bindings map[string]interface{}) (string, []interface{}, error) {
	formula = strings.TrimSpace(formula)
	if formula == "" {
		return "", nil, fmt.Errorf("%w: empty formula", ErrInvalidMetricFormula)
	}

	var args []interface{}
	varMap := make(map[string]MetricVariable)
	for _, v := range variables {
		varMap[strings.TrimPrefix(v.Name, "@")] = v
	}

	// Tokenize formula and replace @variable with placeholders $N
	tokens := strings.Fields(formula)
	var processedTokens []string

	for _, token := range tokens {
		// Substitute every @var occurrence inside the token, not just when the
		// token is entirely a variable. Previously "@a*@b" was one token, was
		// looked up as a variable named "a*@b", missed, and fell through to the
		// neutral 1.0 multiplier - silently dropping the multiplication.
		//
		// Replacements are made in place and the surrounding text is kept
		// verbatim, so no whitespace is inserted: "@a*@b" becomes "$1*$2", and
		// casts and JSON operators ("@a::numeric", "@a->>'k'") survive intact.
		// Inserting spaces around the substitution point would corrupt them.
		rest := token
		var expanded []string
		for len(rest) > 0 {
			at := strings.IndexByte(rest, '@')
			if at < 0 {
				expanded = append(expanded, rest)
				break
			}
			// Consume the identifier characters after '@'. Done inline rather
			// than via a helper because the C0 freeze pins this file's func
			// surface (ADR-025).
			end := at + 1
			for end < len(rest) {
				c := rest[end]
				if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
					end++
					continue
				}
				break
			}
			if end == at+1 {
				// A bare '@' is not a variable reference; pass it through.
				expanded = append(expanded, rest[:at+1])
				rest = rest[at+1:]
				continue
			}
			if at > 0 {
				expanded = append(expanded, rest[:at])
			}
			varName := rest[at+1 : end]
			varDef, exists := varMap[varName]
			var val interface{}
			if bindings != nil && bindings[varName] != nil {
				val = bindings[varName]
			} else if exists && varDef.DefaultValue != nil {
				val = varDef.DefaultValue
			} else if exists && varDef.Required {
				return "", nil, fmt.Errorf("required metric variable @%s not provided", varName)
			} else {
				val = 1.0 // Neutral multiplier fallback
			}
			args = append(args, val)
			expanded = append(expanded, fmt.Sprintf("$%d", len(args)))
			rest = rest[end:]
		}
		// Join with "" so the segments of one original token are re-assembled
		// verbatim. Joining with " " would insert whitespace at every
		// substitution point and corrupt casts and JSON operators.
		processedTokens = append(processedTokens, strings.Join(expanded, ""))
	}

	compiled := strings.Join(processedTokens, " ")
	return compiled, args, nil
}

func sanitizeIdentifier(s string) string {
	var sb strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// ComputeQueryAndMetricsCacheKey computes the combined cache key incorporating
// the query content hash, sorted metric content hashes, and canonical parameter hashes.
func ComputeQueryAndMetricsCacheKey(tenantID, queryContentHash string, referencedMetrics []MetricDefinition, canonicalParamsHash, abacContextHash, routeTier, boSchemaVersion string) string {
	return ComputeQueryAndMetricsAndCubeCacheKey(tenantID, queryContentHash, referencedMetrics, "", canonicalParamsHash, abacContextHash, routeTier, boSchemaVersion)
}

// ComputeQueryAndMetricsAndCubeCacheKey composes the full cache key, including
// the cube content hash when a cube is in play.
//
// cubeContentHash participates because a cube deploy or edit changes which
// materialization serves a query. Including it means a cache entry minted
// against one cube version cannot be served after that cube changes, and an
// undeployed cube can never leave a stale cubeHit claim behind (ADR-016). An
// empty hash contributes a constant, so queries with no cube produce exactly
// the same key as before this term existed.
func ComputeQueryAndMetricsAndCubeCacheKey(tenantID, queryContentHash string, referencedMetrics []MetricDefinition, cubeContentHash, canonicalParamsHash, abacContextHash, routeTier, boSchemaVersion string) string {
	metricHashes := make([]string, len(referencedMetrics))
	for i, m := range referencedMetrics {
		h := m.ContentHash
		if h == "" {
			h = ComputeMetricContentHash(m)
		}
		metricHashes[i] = h
	}
	sort.Strings(metricHashes)
	concatenatedMetricHashes := strings.Join(metricHashes, ":")

	raw := fmt.Sprintf("%s:%s:%s:%s:%s:%s:%s:%s",
		tenantID,
		queryContentHash,
		concatenatedMetricHashes,
		cubeContentHash,
		canonicalParamsHash,
		abacContextHash,
		routeTier,
		boSchemaVersion,
	)
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
