package boresolver

import (
	"fmt"
	"strconv"
	"strings"
)

// Parameter values are bound by TEXTUAL position for placeholder-style
// dialects (Snowflake's "?" today, via paramToken; any future MySQL-wire
// dialect such as a real StarRocks implementation would be the same) - the
// driver binds Args[0] to the first "?" it reads left to right, Args[1] to
// the second, and so on. The token carries no index of its own.
//
// This generator assembles SQL out of textual order: filters are compiled
// (and therefore allocate parameters) in GenerateSQL step 5, before tenant
// scoping and bitemporal scoping are injected in steps 6, yet the tenant/
// bitemporal predicates are spliced to appear textually BEFORE the filter
// clause in the final WHERE (ctx.RootTenantPredicate + " AND " +
// whereClause). For self-indexing dialects ($1, @p1) that's harmless - the
// index is embedded in the token, so binding is correct no matter where the
// token lands in the text. For "?" it is not: the tenant-id value and a
// filter value silently swap. That is not just a wrong-answer bug, it is a
// tenant-isolation bug.
//
// The fix: every call site that needs a parameter records its value in
// creation order and emits an opaque sentinel - never a dialect token -
// standing in for it. Once the full SQL string is assembled, one final pass
// (renumberParams) scans left to right, finds the sentinels in their
// TEXTUAL order, and only then assigns dialect tokens and builds Args - so
// Args[i] always matches the i-th placeholder a driver would read off the
// string, regardless of the order the code that built the string ran in.

const (
	sentinelPrefix = "\x00p"
	sentinelSuffix = "\x00"
)

// paramSentinel returns the opaque placeholder standing in for the
// pendingIdx'th value recorded in a GenerationContext's pending args
// (0-based, i.e. its index at the time it was appended). It must never be
// embedded in SQL as a dialect token directly - only renumberParams may
// turn it into one.
func paramSentinel(pendingIdx int) string {
	return sentinelPrefix + strconv.FormatInt(int64(pendingIdx), 36) + sentinelSuffix
}

// ParamSentinel is the exported form of paramSentinel, for other packages
// (e.g. querybuilder's multi-BO generator) that build parameterized SQL
// against a shared pending-args slice but assemble their own final query
// text outside GenerateSQL, so must do their own final renumbering pass
// with RenumberParams below.
func ParamSentinel(pendingIdx int) string { return paramSentinel(pendingIdx) }

// RenumberParams is the exported form of renumberParams.
func RenumberParams(sql string, dialect Dialect, pending []interface{}) (string, []interface{}, error) {
	return renumberParams(sql, dialect, pending)
}

// renumberParams scans sql left-to-right for sentinels, replaces each
// occurrence with its dialect-correct token numbered by TEXTUAL order of
// appearance, and returns the args slice built to match - one entry per
// occurrence, in the order a driver would read the placeholders off the
// returned string. pending holds values recorded in creation order,
// indexed by the number encoded in each sentinel; the same pending value
// may be referenced by more than one occurrence (e.g. a bitemporal AS-OF
// bound used twice in one predicate), in which case it is bound once per
// occurrence rather than reused, which is correct for both self-indexing
// and positional dialects.
//
// A sentinel decoding to an out-of-range index, an unterminated sentinel,
// or a stray NUL byte surviving the pass are internal invariant violations,
// not caller errors - user-supplied values never reach this function
// directly (see expression_builder.go, CompileFilterPredicate), so a NUL in
// the input can only come from this package's own SQL assembly.
func renumberParams(sql string, dialect Dialect, pending []interface{}) (string, []interface{}, error) {
	if !strings.Contains(sql, sentinelPrefix) {
		if strings.ContainsRune(sql, 0) {
			return "", nil, fmt.Errorf("internal error: stray NUL byte in generated SQL with no recognizable parameter sentinel")
		}
		return sql, nil, nil
	}

	var out strings.Builder
	var args []interface{}
	rest := sql
	for {
		start := strings.Index(rest, sentinelPrefix)
		if start < 0 {
			break
		}
		out.WriteString(rest[:start])
		rest = rest[start+len(sentinelPrefix):]

		end := strings.IndexByte(rest, 0)
		if end < 0 {
			return "", nil, fmt.Errorf("internal error: unterminated parameter sentinel in generated SQL")
		}
		idxStr := rest[:end]
		rest = rest[end+len(sentinelSuffix):]

		idx, err := strconv.ParseInt(idxStr, 36, 64)
		if err != nil || idx < 0 || int(idx) >= len(pending) {
			return "", nil, fmt.Errorf("internal error: parameter sentinel decoded to invalid index %q", idxStr)
		}

		args = append(args, pending[idx])
		out.WriteString(paramToken(dialect, len(args)))
	}
	out.WriteString(rest)

	final := out.String()
	if strings.ContainsRune(final, 0) {
		return "", nil, fmt.Errorf("internal error: stray NUL byte survived parameter renumbering")
	}
	return final, args, nil
}
