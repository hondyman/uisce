package boresolver

import (
	"strings"
	"testing"
)

// TestRenumberParams_TextualOrder is the direct regression test for the bug
// this file fixes: sentinels created out of textual order must come back
// numbered by where they actually land in the string, not by the order
// the code that produced them ran in.
func TestRenumberParams_TextualOrder(t *testing.T) {
	// Simulate exactly what GenerateSQL does: the value for what will be
	// the SECOND placeholder in the text ("b") is recorded FIRST (mirrors
	// filters being compiled before tenant scoping), and the value for the
	// FIRST placeholder in the text ("a") is recorded SECOND (mirrors the
	// tenant predicate being spliced in front of the filter clause).
	pending := []interface{}{"creation-order-first-value-b", "creation-order-second-value-a"}
	nonce := newParamNonce()
	sentinelForB := paramSentinel(nonce, 0)
	sentinelForA := paramSentinel(nonce, 1)

	sql := "WHERE a = " + sentinelForA + " AND b = " + sentinelForB

	gotSQL, gotArgs, err := renumberParams(sql, PostgresDialect{}, pending, nonce)
	if err != nil {
		t.Fatalf("renumberParams failed: %v", err)
	}
	if strings.ContainsRune(gotSQL, 0) {
		t.Fatalf("sentinel survived into final SQL: %q", gotSQL)
	}
	if gotSQL != "WHERE a = $1 AND b = $2" {
		t.Fatalf("expected textual-order numbering, got: %q", gotSQL)
	}
	if len(gotArgs) != 2 || gotArgs[0] != "creation-order-second-value-a" || gotArgs[1] != "creation-order-first-value-b" {
		t.Fatalf("expected args reordered to match text, got: %#v", gotArgs)
	}
}

// TestRenumberParams_QuestionMarkDialect_NoSwap is the scenario that made
// this fix necessary: a positional ("?") dialect has no index in the token
// itself, so if Args isn't rebuilt in textual order, the driver silently
// binds the wrong value to the wrong "?" - here, a tenant-id predicate
// would receive a filter's value instead of the tenant ID.
func TestRenumberParams_QuestionMarkDialect_NoSwap(t *testing.T) {
	// pending[0] = filter value, recorded first (as ConvertFilters would);
	// pending[1] = tenant id, recorded second (as InjectTenantScopingToGraph
	// would) - but the tenant predicate is spliced to appear FIRST in the
	// text, exactly as GenerateSQL's WHERE assembly does.
	pending := []interface{}{"filter-value", "tenant-id-value"}
	nonce := newParamNonce()
	filterSentinel := paramSentinel(nonce, 0)
	tenantSentinel := paramSentinel(nonce, 1)

	sql := "WHERE t0.tenant_id = " + tenantSentinel + " AND t0.name = " + filterSentinel

	gotSQL, gotArgs, err := renumberParams(sql, SnowflakeDialect{}, pending, nonce)
	if err != nil {
		t.Fatalf("renumberParams failed: %v", err)
	}
	if gotSQL != "WHERE t0.tenant_id = ? AND t0.name = ?" {
		t.Fatalf("unexpected SQL: %q", gotSQL)
	}
	// The FIRST "?" a driver reads must bind to the tenant id, the SECOND
	// to the filter value - matching the order the placeholders actually
	// appear in the string, not the order they were allocated in code.
	if len(gotArgs) != 2 || gotArgs[0] != "tenant-id-value" || gotArgs[1] != "filter-value" {
		t.Fatalf("tenant id and filter value swapped: got %#v", gotArgs)
	}
}

// TestRenumberParams_ForgedSentinelWrongNonce_FailsLoud is the F13
// regression test: text that is byte-for-byte a well-formed sentinel
// EXCEPT for the nonce must never be recognized as THIS call's parameter
// placeholder - but it must also never silently ride through into the
// final SQL either, because it still contains a raw NUL byte from a
// source this function cannot identify (a sentinel from a different
// generation, or a genuine forgery - indistinguishable from here). An
// earlier version of this test accepted the wrong-nonce sentinel surviving
// untouched as "plain text" and asserted only that it wasn't misbound;
// that's the wrong bar. Per this file's own fail-loud discipline (verified
// in this same review round: the stray-NUL check that would have caught
// this was accidentally dropped when the nonce was added, then restored),
// an unexplained NUL reaching the output is refused outright, not passed
// through - the safe response to "I can't tell what this is" is to
// refuse, not to shrug and forward it to the driver.
func TestRenumberParams_ForgedSentinelWrongNonce_FailsLoud(t *testing.T) {
	realNonce := newParamNonce()
	forgedNonce := newParamNonce()
	if realNonce == forgedNonce {
		t.Fatal("test setup: nonces collided, cannot demonstrate isolation")
	}

	pending := []interface{}{"real-value"}
	real := paramSentinel(realNonce, 0)
	// Forged: same shape (prefix, a valid base-36 index, suffix) but under
	// a nonce this call to renumberParams does not know.
	forged := paramSentinel(forgedNonce, 0)

	sql := "WHERE forged = " + forged + " AND real = " + real

	_, _, err := renumberParams(sql, PostgresDialect{}, pending, realNonce)
	if err == nil {
		t.Fatal("expected renumberParams to refuse to produce SQL when a foreign NUL-bearing sequence survives - it must not silently forward an unrecognized sentinel's raw bytes to the driver")
	}
}

// TestRenumberParams_NoSentinelsAtAll_PassesThroughUnchanged is the
// legitimate case the stray-NUL guard must NOT catch: ordinary SQL text
// with no sentinels and no NUL bytes of any kind must return unchanged,
// with a nil args slice - proving the guard is scoped to actual NUL
// bytes, not to "any call where expected == 0."
func TestRenumberParams_NoSentinelsAtAll_PassesThroughUnchanged(t *testing.T) {
	sql := "SELECT t0.id FROM orders AS t0 LIMIT 10"
	gotSQL, gotArgs, err := renumberParams(sql, PostgresDialect{}, nil, newParamNonce())
	if err != nil {
		t.Fatalf("renumberParams failed on plain SQL with no sentinels: %v", err)
	}
	if gotSQL != sql {
		t.Fatalf("expected unchanged SQL, got: %q", gotSQL)
	}
	if gotArgs != nil {
		t.Fatalf("expected nil args, got: %#v", gotArgs)
	}
}

// TestGenerateSQL_SnowflakeDialect_TenantAndFilterDoNotSwap is the
// end-to-end version: GenerateSQL itself, with a "?"-style dialect and a
// filter, must not swap the tenant predicate's value with the filter's.
// Before params.go, ConvertFilters allocated its parameter before
// InjectTenantScopingToGraph did, yet the tenant predicate is spliced to
// appear textually before the filter clause - so under a self-indexing
// dialect ($1/@p1) this was invisible, but under "?" it silently bound
// the tenant check to the filter's value instead of the tenant id.
func TestGenerateSQL_SnowflakeDialect_TenantAndFilterDoNotSwap(t *testing.T) {
	repo := &MockBORepository{
		BODefinitions: map[string]*BODefinition{
			"bo_orders": {
				ID:           "bo_orders",
				DrivingTable: "orders",
				Fields: []BOField{
					{ID: "f_id", Name: "id", PhysicalColumn: "orders.id"},
					{ID: "f_total", Name: "total", PhysicalColumn: "orders.total_amount"},
				},
			},
		},
	}

	generator, err := NewBOSQLGenerator(repo, "snowflake")
	if err != nil {
		t.Fatalf("NewBOSQLGenerator failed: %v", err)
	}

	req := SQLGenerationRequest{
		BusinessObjectID: "bo_orders",
		SelectedFields:   []string{"id"},
		TenantID:         "tenant-alpha",
		Filters: []FilterClause{{
			FieldID:  "total",
			Operator: ">",
			Value:    100,
		}},
	}

	sql, args, err := generator.GenerateSQL(req)
	if err != nil {
		t.Fatalf("GenerateSQL failed: %v", err)
	}
	if strings.ContainsRune(sql, 0) {
		t.Fatalf("sentinel leaked into final SQL: %q", sql)
	}

	where := extractWhere(sql)
	tenantPos := strings.Index(where, "t0.tenant_id = ?")
	filterPos := strings.Index(where, "t0.total_amount > ?")
	if tenantPos < 0 || filterPos < 0 {
		t.Fatalf("expected two '?' placeholders (tenant + filter), got: %q", sql)
	}
	if tenantPos > filterPos {
		t.Fatalf("expected tenant predicate before filter clause in text, got: %q", where)
	}

	// The tenant predicate's "?" is the FIRST one in the text, so it must
	// bind to args[0]; the filter's "?" is second, so args[1].
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d: %#v", len(args), args)
	}
	if args[0] != "tenant-alpha" {
		t.Fatalf("tenant predicate bound to wrong value: args[0] = %#v, want \"tenant-alpha\" (tenant isolation bug)", args[0])
	}
	if args[1] != 100 {
		t.Fatalf("filter predicate bound to wrong value: args[1] = %#v, want 100", args[1])
	}
}

// TestRenumberParams_MixedNonce_ForeignNULFailsLoud_PostLoop is the case
// TestRenumberParams_ForgedSentinelWrongNonce_FailsLoud does NOT cover:
// that test has zero sentinels matching the call's own nonce, so it only
// exercises the PRE-loop guard (expected == 0). This test mixes one
// real sentinel (under ctx's nonce) with one foreign one (under a fresh,
// different nonce) in the same string, so expected > 0, the loop runs,
// consumes only the real one, and the foreign sentinel's raw NUL bytes
// are what the POST-loop guard (`strings.ContainsRune(final, 0)`) has to
// catch after the loop exits. This is also the shape of the half-mis-
// wired call site the exported ParamSentinel/EnsureParamNonce footgun
// (tracked separately) would actually produce: one call site using the
// ctx's nonce correctly, another minting its own fresh one by mistake.
func TestRenumberParams_MixedNonce_ForeignNULFailsLoud_PostLoop(t *testing.T) {
	realNonce := newParamNonce()
	foreignNonce := newParamNonce()
	if realNonce == foreignNonce {
		t.Fatal("test setup: nonces collided, cannot demonstrate isolation")
	}

	pending := []interface{}{"real-value"}
	real := paramSentinel(realNonce, 0)
	foreign := paramSentinel(foreignNonce, 0)

	sql := "WHERE real = " + real + " AND foreign = " + foreign

	_, _, err := renumberParams(sql, PostgresDialect{}, pending, realNonce)
	if err == nil {
		t.Fatal("expected renumberParams to refuse to produce SQL: the foreign sentinel's NUL bytes survive the loop (which only consumes matches for realNonce) and must be caught by the POST-loop guard, not silently forwarded")
	}
}

// TestRenumberParams_HighIndex_Base36RoundTrip pins the encode/decode base
// agreement across paramSentinel (encodes with strconv.FormatInt(idx, 36))
// and renumberParams (decodes with strconv.ParseInt(idxStr, 36, 64)). A
// low-index fixture (idx < 10) can't distinguish base 10 from base 36 -
// "9" means the same value in both. idx=11 does: base 36 "b" only round-
// trips correctly if both sides agree it's base 36; misread as base 10 it
// would fail to parse at all ("b" isn't a decimal digit), and a mismatch
// in the other direction (encode decimal, decode base 36) would silently
// land on the wrong pending[] slot once indices reach two digits. This
// test uses 12 pending values so idx=11 is real, in-bounds data, not an
// out-of-range probe.
func TestRenumberParams_HighIndex_Base36RoundTrip(t *testing.T) {
	nonce := newParamNonce()
	pending := make([]interface{}, 12)
	for i := range pending {
		pending[i] = i
	}
	sentinelForIdx11 := paramSentinel(nonce, 11)

	sql := "WHERE x = " + sentinelForIdx11

	gotSQL, gotArgs, err := renumberParams(sql, PostgresDialect{}, pending, nonce)
	if err != nil {
		t.Fatalf("renumberParams failed: %v", err)
	}
	if gotSQL != "WHERE x = $1" {
		t.Fatalf("unexpected SQL: %q", gotSQL)
	}
	if len(gotArgs) != 1 || gotArgs[0] != 11 {
		t.Fatalf("expected pending[11] (=11) bound, got: %#v (base mismatch between encode and decode)", gotArgs)
	}
}

// TestRenumberParams_MalformedIndex_NeverReachesPendingLookup is the F13
// invariant's other half: strconv.ParseInt failing on a corrupted index
// must short-circuit the `||` before `pending[idx]` is ever evaluated -
// not fall through with idx left at its zero value and silently bind
// pending[0]. Proven by using a pending slice whose index 0 holds a
// value distinguishable from "no real parameter here" - if the bug this
// test guards against were reintroduced, this test would observe
// "sentinel-value-at-index-0" bound to a placeholder that was never a
// real parameter at all, instead of the expected error.
func TestRenumberParams_MalformedIndex_NeverReachesPendingLookup(t *testing.T) {
	nonce := newParamNonce()
	pending := []interface{}{"sentinel-value-at-index-0"}
	// A hand-built sentinel with a non-base36 index ("!!" is not valid
	// base-36) - strconv.ParseInt must fail on this, not silently decode
	// to 0.
	malformed := sentinelPrefix + nonce + ":!!" + sentinelSuffix
	sql := "WHERE x = " + malformed

	_, _, err := renumberParams(sql, PostgresDialect{}, pending, nonce)
	if err == nil {
		t.Fatal("expected renumberParams to refuse a malformed index rather than silently decode it to 0 and bind pending[0]")
	}
	if !strings.Contains(err.Error(), "invalid index") {
		t.Fatalf("expected the invalid-index error, got: %v", err)
	}
}
