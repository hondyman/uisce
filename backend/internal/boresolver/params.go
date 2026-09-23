package boresolver

import (
	"crypto/rand"
	"encoding/hex"
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
//
// Sentinels are unforgeable by construction, not just by convention. As of
// this package's main generation path (GenerateSQL), nothing else
// concatenates raw text into the string renumberParams scans - calculated-
// field expressions resolve through ResolvePolymorphicField's
// TransformationSQL template, not through this file, and the one thing
// that does inline literals via QuoteLiteral (Resolver.ToSQL in sqlgen.go,
// reached only via bo_sql_resolver.go's ResolveExpression) is a separate,
// standalone fragment compiler whose output is never passed to
// renumberParams. So there is no live path today by which attacker-
// influenced text reaches this scanner. But "no live path today" is a
// property of the current call graph, not of this function, and this
// function has no way to tell a sentinel it emitted from one forged by
// whatever gets wired in next. So each *generation* (one GenerationContext,
// or one querybuilder pending-args slice) gets its own random nonce
// (paramNonce), and the sentinel format embeds it: forging a sentinel that
// renumberParams will honor requires guessing that nonce, not just knowing
// the byte pattern.

const (
	sentinelPrefix = "\x00p"
	sentinelSuffix = "\x00"
)

// newParamNonce returns a fresh random nonce for one generation's
// sentinels. It never derives from, or is influenced by, any value that
// could originate outside this package - see the package-level comment
// above on why that matters.
func newParamNonce() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand failing is a host-level problem this package cannot
		// recover from; a predictable fallback would defeat the point of
		// the nonce, so fail loudly instead of silently degrading to it.
		panic(fmt.Sprintf("boresolver: crypto/rand unavailable: %v", err))
	}
	return hex.EncodeToString(b)
}

// NewParamNonce is the exported form of newParamNonce, for packages (e.g.
// querybuilder's multi-BO generator) that manage their own pending-args
// slice outside a *GenerationContext and so must mint their own nonce.
func NewParamNonce() string { return newParamNonce() }

// EnsureParamNonce is the exported form of ensureParamNonce, for packages
// (e.g. querybuilder's multi-BO generator) that share a *GenerationContext
// with this package's own sentinel-emitting helpers (CompileFilterPredicate
// et al.) but also mint sentinels of their own - both must resolve to the
// same nonce regardless of call order, which is exactly what this and
// ensureParamNonce guarantee by storing it on ctx itself.
func EnsureParamNonce(ctx *GenerationContext) string { return ensureParamNonce(ctx) }

// ensureParamNonce returns ctx's nonce, minting one on first use. Every
// call site that emits a sentinel for this ctx must go through this (not
// newParamNonce directly), or different call sites within the same
// generation would scope their sentinels to different nonces and
// renumberParams - which only knows one nonce per call - would silently
// treat the others as ordinary text instead of parameters.
func ensureParamNonce(ctx *GenerationContext) string {
	if ctx.ParamNonce == "" {
		ctx.ParamNonce = newParamNonce()
	}
	return ctx.ParamNonce
}

// ParamSentinel is the exported form of paramSentinel, for other packages
// that build parameterized SQL against a shared pending-args slice but
// assemble their own final query text outside GenerateSQL, so must do
// their own final renumbering pass with RenumberParams below - using the
// SAME nonce for every sentinel in one generation.
func ParamSentinel(nonce string, pendingIdx int) string { return paramSentinel(nonce, pendingIdx) }

// RenumberParams is the exported form of renumberParams.
func RenumberParams(sql string, dialect Dialect, pending []interface{}, nonce string) (string, []interface{}, error) {
	return renumberParams(sql, dialect, pending, nonce)
}

// paramSentinel returns the opaque placeholder standing in for the
// pendingIdx'th value recorded in one generation's pending args (0-based,
// i.e. its index at the time it was appended), scoped to nonce so only
// renumberParams called with the same nonce will recognize it. It must
// never be embedded in SQL as a dialect token directly.
func paramSentinel(nonce string, pendingIdx int) string {
	return sentinelPrefix + nonce + ":" + strconv.FormatInt(int64(pendingIdx), 36) + sentinelSuffix
}

// renumberParams scans sql left-to-right for sentinels matching nonce,
// replaces each occurrence with its dialect-correct token numbered by
// TEXTUAL order of appearance, and returns the args slice built to match -
// one entry per occurrence, in the order a driver would read the
// placeholders off the returned string. pending holds values recorded in
// creation order, indexed by the number encoded in each sentinel; the same
// pending value may be referenced by more than one occurrence (e.g. a
// bitemporal AS-OF bound used twice in one predicate), in which case it is
// bound once per occurrence rather than reused, which is correct for both
// self-indexing and positional dialects.
//
// F13 (fail loud, never silently mis-bind): every sentinel this function
// consumes is counted before the scan; after the scan, that count must
// equal len(args) and zero recognizable sentinels (this nonce's prefix)
// may remain in the output. Both are internal invariant violations, not
// caller errors, and both panic rather than return a plausible-looking but
// wrong query - a mismatch here means either this function's own loop is
// broken, or something outside the sentinel mechanism forged a nonce
// collision, and either way the only safe response is to refuse to produce
// SQL at all, not to guess. Text carrying a *different* nonce (or none) is
// left untouched, exactly like any other SQL text - it was never this
// generation's parameter.
func renumberParams(sql string, dialect Dialect, pending []interface{}, nonce string) (string, []interface{}, error) {
	prefix := sentinelPrefix + nonce + ":"
	expected := strings.Count(sql, prefix)
	if expected == 0 {
		return sql, nil, nil
	}

	var out strings.Builder
	var args []interface{}
	rest := sql
	for {
		start := strings.Index(rest, prefix)
		if start < 0 {
			break
		}
		out.WriteString(rest[:start])
		rest = rest[start+len(prefix):]

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

	// F13 guard. Both conditions should be unreachable given the loop
	// above always appends exactly one arg per matched occurrence - which
	// is exactly why a violation here means the invariant broke somewhere
	// this function can't see, and a wrong-but-plausible query is the one
	// outcome that must never happen. Returned as an error, like every
	// other internal-invariant case in this function, rather than a
	// panic: GenerateSQL's callers (HTTP handlers) already treat a
	// non-nil error as "refuse to run this query," which is exactly the
	// response this needs - a panic would depend on recovery middleware
	// this package doesn't control to get the same outcome.
	if len(args) != expected {
		return "", nil, fmt.Errorf("internal error: parameter sentinel count mismatch: found %d occurrences, bound %d args", expected, len(args))
	}
	if strings.Contains(final, prefix) {
		return "", nil, fmt.Errorf("internal error: parameter sentinel survived renumbering")
	}

	return final, args, nil
}
