package tenantplatform

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// The store and probe tests need a real Postgres with the tenant_instance table.
// They run only when TENANT_PLATFORM_TEST_DSN is set, and must point at a
// disposable database.
func platformTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TENANT_PLATFORM_TEST_DSN")
	if dsn == "" {
		t.Skip("TENANT_PLATFORM_TEST_DSN not set; run against a disposable database")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// newProvisioningInstance creates a tenant and one instance in the given status.
func newProvisioningInstance(t *testing.T, pool *pgxpool.Pool, status string) string {
	t.Helper()
	ctx := context.Background()
	tenant := uuid.New().String()
	instance := uuid.New().String()
	if _, err := pool.Exec(ctx, `INSERT INTO public.tenants (id, name, display_name, gold_copy) VALUES ($1, $2, $2, false)`,
		tenant, "platform-test-"+tenant[:8]); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO tenant_instance (id, tenant_id, instance_name, display_name, config, status)
		VALUES ($1, $2, 'dev', 'dev', '{}', $3)`, instance, tenant, status); err != nil {
		t.Fatalf("seed instance: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM tenant_instance WHERE id = $1`, instance)
		_, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id = $1`, tenant)
	})
	return instance
}

func instanceStatus(t *testing.T, pool *pgxpool.Pool, id string) string {
	t.Helper()
	var s string
	if err := pool.QueryRow(context.Background(), `SELECT status FROM tenant_instance WHERE id = $1`, id).Scan(&s); err != nil {
		t.Fatalf("read status: %v", err)
	}
	return s
}

var storeEndpoints = Endpoints{
	PostgresHost: "pg.us-east-1.internal", PostgresPort: 5432,
	DatabaseName: "tenant_acme", Issuer: "https://keycloak.example.internal/realms/acme",
}

func TestMarkRunningIsOneGuardedWriteWithEndpoints(t *testing.T) {
	pool := platformTestPool(t)
	id := newProvisioningInstance(t, pool, "provisioning")
	st := &PgInstanceStore{Pool: pool}

	if err := st.MarkRunning(context.Background(), id, storeEndpoints); err != nil {
		t.Fatalf("MarkRunning: %v", err)
	}
	if got := instanceStatus(t, pool, id); got != "running" {
		t.Fatalf("status = %q, want running", got)
	}
	var host string
	if err := pool.QueryRow(context.Background(), `SELECT config->'endpoints'->>'host' FROM tenant_instance WHERE id = $1`, id).Scan(&host); err != nil || host != storeEndpoints.PostgresHost {
		t.Fatalf("endpoints not recorded: host=%q err=%v", host, err)
	}
}

// A retry after success finds zero rows. With the same endpoints it must succeed,
// so the workflow does not fail on its own retry.
func TestMarkRunningRetryAfterSuccessSucceeds(t *testing.T) {
	pool := platformTestPool(t)
	id := newProvisioningInstance(t, pool, "provisioning")
	st := &PgInstanceStore{Pool: pool}
	if err := st.MarkRunning(context.Background(), id, storeEndpoints); err != nil {
		t.Fatalf("first: %v", err)
	}
	if err := st.MarkRunning(context.Background(), id, storeEndpoints); err != nil {
		t.Fatalf("retry after success failed: %v", err)
	}
}

// A running instance with different endpoints is a conflict, not a retry.
func TestMarkRunningDifferentEndpointsIsRefused(t *testing.T) {
	pool := platformTestPool(t)
	id := newProvisioningInstance(t, pool, "provisioning")
	st := &PgInstanceStore{Pool: pool}
	if err := st.MarkRunning(context.Background(), id, storeEndpoints); err != nil {
		t.Fatalf("first: %v", err)
	}
	other := storeEndpoints
	other.PostgresHost = "pg.eu-west-1.internal"
	if err := st.MarkRunning(context.Background(), id, other); err == nil {
		t.Fatal("re-ran with different endpoints")
	}
}

// A paused or retired instance must not be reactivated by a stale run. The guard
// is the status, so the row must stay as it was.
func TestMarkRunningRefusesAPausedInstance(t *testing.T) {
	pool := platformTestPool(t)
	for _, status := range []string{"paused", "retired", "failed"} {
		id := newProvisioningInstance(t, pool, status)
		st := &PgInstanceStore{Pool: pool}
		if err := st.MarkRunning(context.Background(), id, storeEndpoints); err == nil {
			t.Fatalf("reactivated a %s instance", status)
		}
		if got := instanceStatus(t, pool, id); got != status {
			t.Fatalf("status changed from %s to %s", status, got)
		}
	}
}

func TestMarkRunningMissingInstanceIsAnError(t *testing.T) {
	pool := platformTestPool(t)
	st := &PgInstanceStore{Pool: pool}
	if err := st.MarkRunning(context.Background(), uuid.New().String(), storeEndpoints); err == nil {
		t.Fatal("marked a missing instance running")
	}
}

// --- DatabaseProbe ---

type fakeSecretReader struct {
	values map[string]map[string]string
	err    error
}

func (f *fakeSecretReader) GetMap(_ context.Context, key string) (map[string]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.values[key], nil
}

// The probe is tested against the disposable database with the superuser URL
// that the test environment provides, standing in for the seeded tenant URL.
func TestDatabaseProbeAnswersSelectOneAgainstTheSeededURL(t *testing.T) {
	dsn := os.Getenv("TENANT_PLATFORM_TEST_DSN")
	if dsn == "" {
		t.Skip("TENANT_PLATFORM_TEST_DSN not set")
	}
	p := &PgDatabaseProbe{Secrets: &fakeSecretReader{values: map[string]map[string]string{
		"tenants/acme/dev": {"database_url": dsn},
	}}}
	if err := p.Ping(context.Background(), "tenants/acme/dev"); err != nil {
		t.Fatalf("probe failed against a live database: %v", err)
	}
}

// Failures must never carry the URL. The URL here has a password that must not
// appear in any error.
func TestDatabaseProbeErrorsNeverCarryTheURL(t *testing.T) {
	const secretURL = "postgres://app_user:Sup3r-S3cret@127.0.0.1:1/tenant_acme?sslmode=require"
	cases := map[string]*PgDatabaseProbe{
		"unreachable": {Secrets: &fakeSecretReader{values: map[string]map[string]string{"p": {"database_url": secretURL}}}},
		"malformed":   {Secrets: &fakeSecretReader{values: map[string]map[string]string{"p": {"database_url": "postgres://app_user:Sup3r-S3cret@h\n/db"}}}},
		"missing":     {Secrets: &fakeSecretReader{values: map[string]map[string]string{}}},
		"store down":  {Secrets: &fakeSecretReader{err: errString("store echoed Sup3r-S3cret")}},
	}
	for name, p := range cases {
		err := p.Ping(context.Background(), "p")
		if err == nil {
			t.Errorf("%s: probe succeeded", name)
			continue
		}
		if strings.Contains(err.Error(), "Sup3r-S3cret") || strings.Contains(err.Error(), "app_user") {
			t.Errorf("%s: error leaked credentials: %v", name, err)
		}
	}
}

type errString string

func (e errString) Error() string { return string(e) }
