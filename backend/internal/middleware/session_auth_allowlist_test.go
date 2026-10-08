package middleware

import (
	"database/sql"
	"os"
	"reflect"
	"testing"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Characterization test. It pins how enforcement selects the addresses for a
// tenant: only entries assigned to that tenant. If that behavior changes, this
// test fails, and the change is then a deliberate, reviewed diff rather than drift.
//
// Runs only when TENANT_MW_TEST_DSN is set, against a disposable database.
func allowlistTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TENANT_MW_TEST_DSN")
	if dsn == "" {
		t.Skip("TENANT_MW_TEST_DSN not set; run against a disposable database")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func seedAllowlistTenant(t *testing.T, db *sql.DB) string {
	t.Helper()
	id := uuid.New().String()
	if _, err := db.Exec(`INSERT INTO public.tenants (id, name, display_name, gold_copy) VALUES ($1, $2, $2, false)`,
		id, "mw-allowlist-"+id[:8]); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM tenant_ip_whitelist_assignments WHERE tenant_id = $1`, id)
		_, _ = db.Exec(`DELETE FROM tenant_ip_whitelist_entries WHERE tenant_id = $1`, id)
		_, _ = db.Exec(`DELETE FROM tenants WHERE id = $1`, id)
	})
	return id
}

// addEntry writes an entry for a tenant and returns its ID. The assignment is
// optional: passing assign=false leaves the entry unassigned.
func addEntry(t *testing.T, db *sql.DB, owner, cidr string, assign bool, assignee string) string {
	t.Helper()
	var id string
	if err := db.QueryRow(`INSERT INTO tenant_ip_whitelist_entries (tenant_id, ip_address, created_at, updated_at)
		VALUES ($1, $2, now(), now()) RETURNING id::text`, owner, cidr).Scan(&id); err != nil {
		t.Fatalf("insert entry: %v", err)
	}
	if assign {
		if _, err := db.Exec(`INSERT INTO tenant_ip_whitelist_assignments (whitelist_id, tenant_id, created_at) VALUES ($1::uuid, $2::uuid, now())`,
			id, assignee); err != nil {
			t.Fatalf("assign: %v", err)
		}
	}
	return id
}

func TestEnforcementSelectsOnlyEntriesAssignedToTheTenant(t *testing.T) {
	db := allowlistTestDB(t)
	tenant := seedAllowlistTenant(t, db)
	addEntry(t, db, tenant, "203.0.113.10/32", true, tenant)

	got, err := loadTenantAllowlist(db, tenant)
	if err != nil {
		t.Fatalf("loadTenantAllowlist: %v", err)
	}
	if !reflect.DeepEqual(got, []string{"203.0.113.10/32"}) {
		t.Fatalf("enforced = %v, want the assigned entry", got)
	}
}

// The behavior the admin API's "enforced globally" label contradicted: an entry
// with no assignment is not enforced for any tenant. Pinned here so the question
// of making it global becomes a deliberate change.
func TestUnassignedEntryIsNeverEnforced(t *testing.T) {
	db := allowlistTestDB(t)
	tenant := seedAllowlistTenant(t, db)
	addEntry(t, db, tenant, "203.0.113.20/32", false, "")

	got, err := loadTenantAllowlist(db, tenant)
	if err != nil {
		t.Fatalf("loadTenantAllowlist: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("unassigned entry enforced for its own tenant: %v", got)
	}
}

// An entry assigned to another tenant is not enforced for this one.
func TestAnotherTenantsEntryIsNotEnforced(t *testing.T) {
	db := allowlistTestDB(t)
	mine := seedAllowlistTenant(t, db)
	other := seedAllowlistTenant(t, db)
	addEntry(t, db, other, "203.0.113.30/32", true, other)

	got, err := loadTenantAllowlist(db, mine)
	if err != nil {
		t.Fatalf("loadTenantAllowlist: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("another tenant's entry enforced here: %v", got)
	}
}

// A tenant with no entries gets an empty list. The middleware reads that as
// "no restriction", which is the fail-open behavior the allowlist was decided on.
func TestTenantWithNoEntriesGetsAnEmptyList(t *testing.T) {
	db := allowlistTestDB(t)
	tenant := seedAllowlistTenant(t, db)
	got, err := loadTenantAllowlist(db, tenant)
	if err != nil {
		t.Fatalf("loadTenantAllowlist: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no restriction, got %v", got)
	}
}

// A failed load must surface as an error, never as an empty list. The middleware
// answers an error with 500 and stops, so the request is refused rather than
// served without the tenant's restrictions. This pins that fail-closed path: a
// database outage must not strip a restricted tenant's allowlist.
func TestLoadFailureIsAnErrorNotAnEmptyAllowlist(t *testing.T) {
	db := allowlistTestDB(t)
	tenant := seedAllowlistTenant(t, db)
	addEntry(t, db, tenant, "203.0.113.60/32", true, tenant)
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	got, err := loadTenantAllowlist(db, tenant)
	if err == nil {
		t.Fatalf("load on a closed database returned %v with no error; a restricted tenant would be allowed through", got)
	}
	if err.Error() != "Failed to query IP whitelist" {
		t.Fatalf("error text = %q, the middleware's 500 body depends on it", err.Error())
	}
	if got != nil {
		t.Fatalf("returned a partial list with the error: %v", got)
	}
}
