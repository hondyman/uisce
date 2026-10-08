package tenantnetwork

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// The store tests need a real Postgres on the live schema. They run only when
// TENANT_NET_TEST_DSN is set, and must point at a disposable database.
func testStore(t *testing.T) (*PgAllowlistStore, *pgxpool.Pool) {
	t.Helper()
	dsn := os.Getenv("TENANT_NET_TEST_DSN")
	if dsn == "" {
		t.Skip("TENANT_NET_TEST_DSN not set; run against a disposable database")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return &PgAllowlistStore{Pool: pool}, pool
}

func newTestTenant(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	id := uuid.New().String()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO public.tenants (id, name, display_name, gold_copy) VALUES ($1, $2, $2, false)`,
		id, "allowlist-test-"+id[:8]); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, `DELETE FROM tenant_ip_whitelist_assignments WHERE tenant_id = $1`, id)
		_, _ = pool.Exec(ctx, `DELETE FROM tenant_ip_whitelist_entries WHERE tenant_id = $1`, id)
		_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id = $1`, id)
	})
	return id
}

func assignedCIDRs(t *testing.T, pool *pgxpool.Pool, tenantID string) []string {
	t.Helper()
	rows, err := pool.Query(context.Background(), `
		SELECT e.ip_address FROM tenant_ip_whitelist_assignments a
		JOIN tenant_ip_whitelist_entries e ON e.id = a.whitelist_id
		WHERE a.tenant_id = $1 ORDER BY e.ip_address`, tenantID)
	if err != nil {
		t.Fatalf("read assignments: %v", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out = append(out, s)
	}
	return out
}

// The invariant: every entry a tenant owns has a matching assignment to that
// same tenant. An entry without one is inert, but it is still a broken write.
func assertNoUnassignedEntries(t *testing.T, pool *pgxpool.Pool, tenantID string) {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM tenant_ip_whitelist_entries e
		WHERE e.tenant_id = $1 AND NOT EXISTS (
			SELECT 1 FROM tenant_ip_whitelist_assignments a
			WHERE a.whitelist_id = e.id AND a.tenant_id = e.tenant_id)`, tenantID).Scan(&n); err != nil {
		t.Fatalf("count unassigned: %v", err)
	}
	if n != 0 {
		t.Fatalf("%d entries for tenant %s have no assignment", n, tenantID)
	}
}

func TestStoreWritesEveryEntryWithItsAssignment(t *testing.T) {
	store, pool := testStore(t)
	tenant := newTestTenant(t, pool)
	ctx := context.Background()

	if err := store.ReplaceTenantAllowlist(ctx, tenant, []string{"203.0.113.10/32", "203.0.113.0/24"}); err != nil {
		t.Fatalf("first replace: %v", err)
	}
	assertNoUnassignedEntries(t, pool, tenant)
	if got := assignedCIDRs(t, pool, tenant); len(got) != 2 {
		t.Fatalf("assigned = %v", got)
	}
	if err := store.ReplaceTenantAllowlist(ctx, tenant, []string{"203.0.113.20/32"}); err != nil {
		t.Fatalf("second replace: %v", err)
	}
	assertNoUnassignedEntries(t, pool, tenant)
	if got := assignedCIDRs(t, pool, tenant); len(got) != 1 || got[0] != "203.0.113.20/32" {
		t.Fatalf("after replace assigned = %v", got)
	}
	if err := store.ReplaceTenantAllowlist(ctx, tenant, nil); err != nil {
		t.Fatalf("clear: %v", err)
	}
	assertNoUnassignedEntries(t, pool, tenant)
}

// One tenant's replace must not change another tenant's list, even when both
// use the same address.
func TestReplaceDoesNotTouchAnotherTenant(t *testing.T) {
	store, pool := testStore(t)
	a := newTestTenant(t, pool)
	b := newTestTenant(t, pool)
	ctx := context.Background()

	if err := store.ReplaceTenantAllowlist(ctx, a, []string{"203.0.113.30/32"}); err != nil {
		t.Fatalf("a: %v", err)
	}
	if err := store.ReplaceTenantAllowlist(ctx, b, []string{"203.0.113.30/32"}); err != nil {
		t.Fatalf("b: %v", err)
	}
	if err := store.ReplaceTenantAllowlist(ctx, a, nil); err != nil {
		t.Fatalf("a clear: %v", err)
	}
	if got := assignedCIDRs(t, pool, b); len(got) != 1 || got[0] != "203.0.113.30/32" {
		t.Fatalf("tenant b lost its entry: %v", got)
	}
}

// A failed write must leave the previous list in place. The failure here is a
// tenant ID with no tenant row, which breaks the assignment's foreign key.
func TestFailedReplaceRollsBackToPreviousList(t *testing.T) {
	store, pool := testStore(t)
	tenant := newTestTenant(t, pool)
	ctx := context.Background()

	if err := store.ReplaceTenantAllowlist(ctx, tenant, []string{"203.0.113.40/32"}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	ghost := uuid.New().String()
	if err := store.ReplaceTenantAllowlist(ctx, ghost, []string{"203.0.113.41/32"}); err == nil {
		t.Fatal("write for an unknown tenant succeeded")
	}
	if got := assignedCIDRs(t, pool, tenant); len(got) != 1 || got[0] != "203.0.113.40/32" {
		t.Fatalf("previous list changed after a failed write: %v", got)
	}
	var stray int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM tenant_ip_whitelist_entries WHERE tenant_id = $1::uuid`, ghost).Scan(&stray); err != nil {
		t.Fatalf("count: %v", err)
	}
	if stray != 0 {
		t.Fatal("failed write left an entry behind")
	}
}

func TestStoreErrorsDoNotEchoInputs(t *testing.T) {
	store, _ := testStore(t)
	ghost := uuid.New().String()
	err := store.ReplaceTenantAllowlist(context.Background(), ghost, []string{"203.0.113.50/32"})
	if err == nil {
		t.Fatal("expected failure for unknown tenant")
	}
	if strings.Contains(err.Error(), ghost) || strings.Contains(err.Error(), "203.0.113.50") {
		t.Fatalf("error echoed input: %v", err)
	}
}
