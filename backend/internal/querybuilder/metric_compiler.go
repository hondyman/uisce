package querybuilder

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/hondyman/uisce/backend/internal/boresolver"
)

// MetricCompiler compiles metric definitions and AST expressions into safe SQL expressions and arguments.
type MetricCompiler struct {
	dialect boresolver.Dialect
}

// NewMetricCompiler creates a new metric compiler instance.
func NewMetricCompiler(dialect boresolver.Dialect) *MetricCompiler {
	if dialect == nil {
		dialect = boresolver.PostgresDialect{}
	}
	return &MetricCompiler{dialect: dialect}
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
		// Derived metric: combine base metrics.
		if len(m.Expression.BaseMetricIDs) == 0 {
			return nil, fmt.Errorf("%w: derived metric requires baseMetricIds", ErrInvalidMetricFormula)
		}

		// Operand order. ADR-025: a derived metric's meaning must not depend on
		// how its base metric IDs happen to be spelled. An explicit
		// numeratorId/denominatorId wins; otherwise the author's declared order
		// is the order. Only the N-operand sum - where operand order cannot
		// change the value - is sorted, which keeps its output deterministic.
		ids := make([]string, len(m.Expression.BaseMetricIDs))
		copy(ids, m.Expression.BaseMetricIDs)

		if num, den := m.Expression.NumeratorID, m.Expression.DenominatorID; num != "" || den != "" {
			if num == "" || den == "" {
				return nil, fmt.Errorf("%w: derived ratio needs both numeratorId and denominatorId", ErrInvalidMetricFormula)
			}
			if num == den {
				return nil, fmt.Errorf("%w: derived ratio numeratorId and denominatorId are the same metric", ErrInvalidMetricFormula)
			}
			if len(ids) != 2 {
				return nil, fmt.Errorf("%w: numeratorId/denominatorId describe a 2-metric ratio, but baseMetricIds has %d entries", ErrInvalidMetricFormula, len(ids))
			}
			hasNum, hasDen := false, false
			for _, id := range ids {
				if id == num {
					hasNum = true
				}
				if id == den {
					hasDen = true
				}
			}
			if !hasNum || !hasDen {
				return nil, fmt.Errorf("%w: numeratorId and denominatorId must both appear in baseMetricIds", ErrInvalidMetricFormula)
			}
			ids = []string{num, den}
		} else if len(ids) > 2 {
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
