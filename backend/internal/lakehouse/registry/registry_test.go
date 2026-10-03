package registry_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	dbpkg "github.com/hondyman/uisce/backend/internal/db"
	"github.com/hondyman/uisce/backend/internal/lakehouse/registry"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
)

// These tests run against a real Postgres. Set LAKEHOUSE_REGISTRY_TEST_DSN to a
// database that has migrations 20261206_001 and 20261206_002 applied, a
// public.tenants(id uuid primary key, name text, code text) table, and whose
// connecting role is neither a superuser nor BYPASSRLS: row-level security is what
// the isolation tests exercise, and a role that bypasses it would make them
// vacuous, so they refuse to run as one.
func openDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("LAKEHOUSE_REGISTRY_TEST_DSN")
	if dsn == "" {
		t.Skip("LAKEHOUSE_REGISTRY_TEST_DSN not set")
	}
	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	var bypass bool
	require.NoError(t, db.QueryRow(
		`SELECT rolsuper OR rolbypassrls FROM pg_roles WHERE rolname = current_user`).Scan(&bypass))
	require.False(t, bypass, "the test role bypasses row-level security; use an ordinary role")
	return db
}

func newTenant(t *testing.T, db *sql.DB, name string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := db.Exec(`INSERT INTO public.tenants (id, name, code) VALUES ($1, $2, $3)`,
		id, name, "code-"+id.String()[:8])
	require.NoError(t, err)
	return id
}

var actor = registry.Actor{ID: "admin-1", Role: "global_admin"}

func days(c *registry.Config) int {
	if c.AuditRetentionDays == nil {
		return -1
	}
	return *c.AuditRetentionDays
}

func actions(t *testing.T, s *registry.Store, id uuid.UUID) []string {
	t.Helper()
	entries, err := s.Audit(context.Background(), id, 100)
	require.NoError(t, err)
	out := make([]string, 0, len(entries))
	for i := len(entries) - 1; i >= 0; i-- { // oldest first
		out = append(out, entries[i].Action)
	}
	return out
}

func TestGet_UnconfiguredAndUnknown(t *testing.T) {
	db := openDB(t)
	s := registry.NewStore(db)
	ctx := context.Background()

	id := newTenant(t, db, "Never Configured")
	c, err := s.Get(ctx, id)
	require.NoError(t, err)
	require.False(t, c.Configured)
	require.Equal(t, "unconfigured", c.LifecycleState)
	require.Nil(t, c.AuditRetentionDays)
	require.Equal(t, "Never Configured", c.TenantName)

	_, err = s.Get(ctx, uuid.New())
	require.ErrorIs(t, err, registry.ErrTenantNotFound)
}

func TestSetRetention_CreatesExtendsAndRefusesToLower(t *testing.T) {
	db := openDB(t)
	s := registry.NewStore(db)
	ctx := context.Background()
	id := newTenant(t, db, "Retention")

	c, err := s.SetRetention(ctx, id, 365, actor)
	require.NoError(t, err)
	require.True(t, c.Configured)
	require.Equal(t, 365, days(c))
	want := "ivy-t-" + strings.ReplaceAll(id.String(), "-", "")
	require.Equal(t, want, c.WarehouseName, "names are derived from the tenant id")
	require.Equal(t, want, c.Bucket)
	require.False(t, c.Provisioned)
	require.Equal(t, []string{"configured"}, actions(t, s, id))

	// Same value: nothing changes and nothing is audited.
	again, err := s.SetRetention(ctx, id, 365, actor)
	require.NoError(t, err)
	require.Equal(t, c.Version, again.Version)
	require.Equal(t, []string{"configured"}, actions(t, s, id))

	// Extending.
	ext, err := s.SetRetention(ctx, id, 2555, actor)
	require.NoError(t, err)
	require.Equal(t, 2555, days(ext))
	require.Equal(t, c.Version+1, ext.Version)
	require.Equal(t, []string{"configured", "retention_extended"}, actions(t, s, id))

	// Lowering is refused, and leaves the value and the audit trail alone.
	_, err = s.SetRetention(ctx, id, 30, actor)
	require.ErrorIs(t, err, registry.ErrRetentionLowered)
	still, err := s.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, 2555, days(still))
	require.Equal(t, []string{"configured", "retention_extended"}, actions(t, s, id))

	// Out of range.
	for _, bad := range []int{0, -5, registry.MaxRetentionDays + 1} {
		_, err = s.SetRetention(ctx, id, bad, actor)
		require.ErrorIs(t, err, registry.ErrInvalidRetention, "%d", bad)
	}

	// An unknown tenant never gets a row.
	_, err = s.SetRetention(ctx, uuid.New(), 365, actor)
	require.ErrorIs(t, err, registry.ErrTenantNotFound)
}

func TestAudit_RecordsActorAndVerifies(t *testing.T) {
	db := openDB(t)
	s := registry.NewStore(db)
	ctx := context.Background()
	id := newTenant(t, db, "Audited")

	_, err := s.SetRetention(ctx, id, 365, registry.Actor{ID: "alice", Role: "global_admin"})
	require.NoError(t, err)
	_, err = s.SetRetention(ctx, id, 730, registry.Actor{ID: "bob", Role: "global_ops"})
	require.NoError(t, err)
	require.NoError(t, s.Record(ctx, id, registry.Actor{ID: "bob", Role: "global_ops"}, "provision_requested", nil, map[string]int{"audit_retention_days": 730}))

	entries, err := s.Audit(ctx, id, 100)
	require.NoError(t, err)
	require.Len(t, entries, 3)
	require.Equal(t, "provision_requested", entries[0].Action, "newest first")
	require.Equal(t, "bob", entries[0].ActorID)
	require.Equal(t, "alice", entries[2].ActorID)
	require.JSONEq(t, `{"audit_retention_days":365}`, string(entries[1].Before))
	require.JSONEq(t, `{"audit_retention_days":730}`, string(entries[1].After))
	require.Equal(t, entries[2].Hash, entries[1].PrevHash, "each entry links to the one before")
	require.Equal(t, strings.Repeat("0", 64), entries[2].PrevHash)

	broken, err := s.VerifyAudit(ctx, id)
	require.NoError(t, err)
	require.Nil(t, broken, "an untouched chain verifies")
}

func TestIsolation_AnotherTenantAndNoTenantSeeNothing(t *testing.T) {
	db := openDB(t)
	s := registry.NewStore(db)
	ctx := context.Background()
	a := newTenant(t, db, "Tenant A")
	b := newTenant(t, db, "Tenant B")
	_, err := s.SetRetention(ctx, a, 365, actor)
	require.NoError(t, err)

	count := func(tx *sql.Tx, table string) int {
		var n int
		require.NoError(t, tx.QueryRow(`SELECT count(*) FROM public.`+table).Scan(&n))
		return n
	}
	for _, table := range []string{"tenant_lakehouse", "tenant_lakehouse_audit"} {
		// Tenant B's context sees none of A's rows.
		require.NoError(t, dbpkg.WithTenantTransaction(ctx, db, b.String(), func(tx *sql.Tx) error {
			require.Zero(t, count(tx, table), table+" leaked to another tenant")
			return nil
		}))
		// A transaction with no tenant set sees nothing at all.
		tx, err := db.BeginTx(ctx, nil)
		require.NoError(t, err)
		require.Zero(t, count(tx, table), table+" visible with no tenant set")
		require.NoError(t, tx.Rollback())
	}

	// B cannot write A's row either: B's own context rejects a row for A.
	err = dbpkg.WithTenantTransaction(ctx, db, b.String(), func(tx *sql.Tx) error {
		_, e := tx.Exec(`INSERT INTO public.tenant_lakehouse_audit (tenant_id, actor_id, action) VALUES ($1, 'x', 'configured')`, a)
		return e
	})
	require.Error(t, err, "a tenant context must not be able to write another tenant's audit")
}

func TestList_PagesSearchesAndEscapes(t *testing.T) {
	db := openDB(t)
	s := registry.NewStore(db)
	ctx := context.Background()

	marker := "zzlist" + uuid.New().String()[:6]
	var ids []uuid.UUID
	for _, n := range []string{"a", "b", "c"} {
		ids = append(ids, newTenant(t, db, marker+"-"+n))
	}
	_, err := s.SetRetention(ctx, ids[1], 365, actor)
	require.NoError(t, err)

	items, total, err := s.List(ctx, marker, 2, 0)
	require.NoError(t, err)
	require.Equal(t, 3, total)
	require.Len(t, items, 2)
	require.Equal(t, marker+"-a", items[0].TenantName)
	require.False(t, items[0].Configured)

	items, _, err = s.List(ctx, marker, 2, 2)
	require.NoError(t, err)
	require.Len(t, items, 1)

	items, _, err = s.List(ctx, marker+"-b", 10, 0)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.True(t, items[0].Configured)
	require.Equal(t, 365, days(&items[0]))

	// A literal % or _ in the search is text, not a wildcard.
	_, total, err = s.List(ctx, "%", 10, 0)
	require.NoError(t, err)
	var literal int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM public.tenants WHERE lower(name) LIKE '%\%%' OR lower(COALESCE(code,'')) LIKE '%\%%'`).Scan(&literal))
	require.Equal(t, literal, total, "searching % must match names containing a literal %, not everything")
}

// Several first-time writers race on the same tenant. All must resolve cleanly (a
// write that loses to a larger value is a refused lowering, nothing else), the final
// value is the largest that was accepted, there is exactly one "configured" entry,
// and the audit chain is intact.
func TestSetRetention_ConcurrentWritersLeaveAConsistentChain(t *testing.T) {
	db := openDB(t)
	db.SetMaxOpenConns(12)
	s := registry.NewStore(db)
	ctx := context.Background()
	id := newTenant(t, db, "Race")

	const n = 10
	results := make([]error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, results[i] = s.SetRetention(ctx, id, 100+i, actor)
		}(i)
	}
	wg.Wait()

	for i, err := range results {
		if err != nil && !errors.Is(err, registry.ErrRetentionLowered) {
			t.Fatalf("writer %d: unexpected error %v", i, err)
		}
	}
	final, err := s.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, 100+n-1, days(final), "the largest requested value wins; none can lower it")

	acts := actions(t, s, id)
	configured := 0
	for _, a := range acts {
		if a == "configured" {
			configured++
		}
	}
	require.Equal(t, 1, configured, "exactly one writer created the row: %v", acts)

	broken, err := s.VerifyAudit(ctx, id)
	require.NoError(t, err)
	require.Nil(t, broken, "the chain must survive concurrent writers")
}

func TestMarkProvisioned(t *testing.T) {
	db := openDB(t)
	s := registry.NewStore(db)
	ctx := context.Background()
	wh := uuid.New()

	t.Run("refuses a tenant that was never configured", func(t *testing.T) {
		id := newTenant(t, db, "Unconfigured")
		require.ErrorIs(t, s.MarkProvisioned(ctx, id, wh, "key-"+id.String(), actor), registry.ErrNotConfigured)
	})

	t.Run("binds the warehouse, goes active, and audits it", func(t *testing.T) {
		id := newTenant(t, db, "Provision")
		_, err := s.SetRetention(ctx, id, 2555, actor)
		require.NoError(t, err)
		before, err := s.Get(ctx, id)
		require.NoError(t, err)

		require.NoError(t, s.MarkProvisioned(ctx, id, uuid.New(), "key-"+id.String(), actor))
		got, err := s.Get(ctx, id)
		require.NoError(t, err)
		require.True(t, got.Provisioned)
		require.Equal(t, "active", got.LifecycleState)
		require.Equal(t, before.Version+1, got.Version)
		require.Equal(t, []string{"configured", "provisioned"}, actions(t, s, id))

		entries, err := s.Audit(ctx, id, 10)
		require.NoError(t, err)
		require.JSONEq(t, `{"audit_retention_days":2555,"bucket":"`+got.Bucket+`","warehouse_id":"`+
			extractWarehouse(t, entries[0].After)+`"}`, string(entries[0].After))
		broken, err := s.VerifyAudit(ctx, id)
		require.NoError(t, err)
		require.Nil(t, broken)
	})

	t.Run("is idempotent for the same warehouse and refuses a different one", func(t *testing.T) {
		id := newTenant(t, db, "Idempotent")
		_, err := s.SetRetention(ctx, id, 365, actor)
		require.NoError(t, err)
		same, key := uuid.New(), "key-"+id.String()
		require.NoError(t, s.MarkProvisioned(ctx, id, same, key, actor))
		require.NoError(t, s.MarkProvisioned(ctx, id, same, key, actor), "a retry of the same binding is a no-op")
		require.Equal(t, []string{"configured", "provisioned"}, actions(t, s, id), "and writes no second audit entry")

		require.ErrorIs(t, s.MarkProvisioned(ctx, id, uuid.New(), key, actor), registry.ErrWarehouseMismatch,
			"a tenant's one warehouse is never silently replaced")
		got, err := s.Get(ctx, id)
		require.NoError(t, err)
		require.True(t, got.Provisioned)
	})

	t.Run("refuses a tenant that is not in the provisioning state", func(t *testing.T) {
		id := newTenant(t, db, "Suspended")
		_, err := s.SetRetention(ctx, id, 365, actor)
		require.NoError(t, err)
		require.NoError(t, dbpkg.WithTenantTransaction(ctx, db, id.String(), func(tx *sql.Tx) error {
			_, e := tx.Exec(`UPDATE public.tenant_lakehouse SET lifecycle_state = 'suspended' WHERE tenant_id = $1`, id)
			return e
		}))
		require.ErrorIs(t, s.MarkProvisioned(ctx, id, uuid.New(), "key-"+id.String(), actor), registry.ErrInvalidState)
	})

	t.Run("two tenants can never share a warehouse", func(t *testing.T) {
		a, b := newTenant(t, db, "Share A"), newTenant(t, db, "Share B")
		for _, id := range []uuid.UUID{a, b} {
			_, err := s.SetRetention(ctx, id, 365, actor)
			require.NoError(t, err)
		}
		shared := uuid.New()
		require.NoError(t, s.MarkProvisioned(ctx, a, shared, "key-"+a.String(), actor))
		require.Error(t, s.MarkProvisioned(ctx, b, shared, "key-"+b.String(), actor), "the unique index must refuse it")
		got, err := s.Get(ctx, b)
		require.NoError(t, err)
		require.False(t, got.Provisioned, "and the failed attempt must leave tenant B untouched")
	})

	t.Run("requires a warehouse id and a key", func(t *testing.T) {
		id := newTenant(t, db, "Blank")
		_, err := s.SetRetention(ctx, id, 365, actor)
		require.NoError(t, err)
		require.Error(t, s.MarkProvisioned(ctx, id, uuid.Nil, "k", actor))
		require.Error(t, s.MarkProvisioned(ctx, id, uuid.New(), "", actor))
	})
}

func extractWarehouse(t *testing.T, after []byte) string {
	t.Helper()
	var v struct {
		WarehouseID string `json:"warehouse_id"`
	}
	require.NoError(t, json.Unmarshal(after, &v))
	return v.WarehouseID
}

func TestRecordProvisionFailure(t *testing.T) {
	db := openDB(t)
	s := registry.NewStore(db)
	ctx := context.Background()
	id := newTenant(t, db, "Failing")
	_, err := s.SetRetention(ctx, id, 365, actor)
	require.NoError(t, err)

	require.NoError(t, s.RecordProvisionFailure(ctx, id, actor, "EnsureLakehouseBucket", strings.Repeat("x", 900)))
	entries, err := s.Audit(ctx, id, 10)
	require.NoError(t, err)
	require.Equal(t, "provision_failed", entries[0].Action)
	var after map[string]string
	require.NoError(t, json.Unmarshal(entries[0].After, &after))
	require.Equal(t, "EnsureLakehouseBucket", after["step"])
	require.Len(t, after["error"], 500, "a long reason is truncated, not stored whole")

	got, err := s.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "provisioning", got.LifecycleState, "a failure leaves the tenant waiting, not broken")
	broken, err := s.VerifyAudit(ctx, id)
	require.NoError(t, err)
	require.Nil(t, broken)
}

func TestMarkCredentialIssued(t *testing.T) {
	db := openDB(t)
	s := registry.NewStore(db)
	ctx := context.Background()

	t.Run("is false until marked, and marking is idempotent", func(t *testing.T) {
		id := newTenant(t, db, "Cred")
		_, err := s.SetRetention(ctx, id, 365, actor)
		require.NoError(t, err)
		got, err := s.Get(ctx, id)
		require.NoError(t, err)
		require.False(t, got.CredentialIssued)

		require.NoError(t, s.MarkCredentialIssued(ctx, id))
		got, err = s.Get(ctx, id)
		require.NoError(t, err)
		require.True(t, got.CredentialIssued)
		firstVersion := got.Version

		require.NoError(t, s.MarkCredentialIssued(ctx, id), "a second mark is a no-op, not an error")
		again, err := s.Get(ctx, id)
		require.NoError(t, err)
		require.True(t, again.CredentialIssued)
		require.Equal(t, firstVersion, again.Version, "and it does not churn the version")
	})

	t.Run("refuses a tenant with no registry row", func(t *testing.T) {
		id := newTenant(t, db, "NoRow")
		require.ErrorIs(t, s.MarkCredentialIssued(ctx, id), registry.ErrNotConfigured)
	})

	t.Run("does not survive into another tenant", func(t *testing.T) {
		a, b := newTenant(t, db, "CredA"), newTenant(t, db, "CredB")
		for _, id := range []uuid.UUID{a, b} {
			_, err := s.SetRetention(ctx, id, 365, actor)
			require.NoError(t, err)
		}
		require.NoError(t, s.MarkCredentialIssued(ctx, a))
		got, err := s.Get(ctx, b)
		require.NoError(t, err)
		require.False(t, got.CredentialIssued)
	})
}
