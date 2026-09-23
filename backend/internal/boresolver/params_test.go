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

// TestRenumberParams_ForgedSentinelWrongNonce_NotTreatedAsParam is the F13
// regression test: text that is byte-for-byte a well-formed sentinel EXCEPT
// for the nonce must not be recognized as a parameter placeholder. Without
// the nonce, any code path that concatenates attacker-influenced text
// containing "\x00p<n>\x00" into the assembled SQL (a NUL byte is trivial
// to put in a Go string) could forge a placeholder and desynchronize Args
// from a later, real sentinel - this is what makes the nonce load-bearing
// rather than decorative.
func TestRenumberParams_ForgedSentinelWrongNonce_NotTreatedAsParam(t *testing.T) {
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

	gotSQL, gotArgs, err := renumberParams(sql, PostgresDialect{}, pending, realNonce)
	if err != nil {
		t.Fatalf("renumberParams failed: %v", err)
	}
	// Exactly one real placeholder was consumed; the forged one passes
	// through as ordinary (if odd) text, never as a second $N stealing a
	// value that doesn't exist for it.
	if len(gotArgs) != 1 || gotArgs[0] != "real-value" {
		t.Fatalf("expected exactly the one real arg, got: %#v", gotArgs)
	}
	if !strings.Contains(gotSQL, forged) {
		t.Fatalf("expected the wrong-nonce sentinel to survive untouched as plain text, got: %q", gotSQL)
	}
	if !strings.Contains(gotSQL, "real = $1") {
		t.Fatalf("expected the real sentinel to become $1, got: %q", gotSQL)
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
