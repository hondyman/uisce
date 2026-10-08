package tenantidentity

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// The issuer store test needs a real Postgres with the tenants table. It runs only
// when TENANT_IDENTITY_TEST_DSN is set, and must point at a disposable database.
func issuerTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TENANT_IDENTITY_TEST_DSN")
	if dsn == "" {
		t.Skip("TENANT_IDENTITY_TEST_DSN not set; run against a disposable database")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func seedIssuerTenant(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	id := uuid.New().String()
	if _, err := pool.Exec(context.Background(), `INSERT INTO public.tenants (id, name, display_name, gold_copy) VALUES ($1, $2, $2, false)`,
		id, "issuer-test-"+id[:8]); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id = $1`, id) })
	return id
}

func readIssuer(t *testing.T, pool *pgxpool.Pool, id string) string {
	t.Helper()
	var v *string
	if err := pool.QueryRow(context.Background(),
		`SELECT configuration->>'`+IssuerKey+`' FROM tenants WHERE id = $1`, id).Scan(&v); err != nil {
		t.Fatalf("read issuer: %v", err)
	}
	if v == nil {
		return ""
	}
	return *v
}

func TestIssuerStoreCreatesThenOverwritesOnUpdate(t *testing.T) {
	pool := issuerTestPool(t)
	id := seedIssuerTenant(t, pool)
	st := &PgIssuerStore{Pool: pool}
	ctx := context.Background()

	if err := st.SetIssuer(ctx, id, "https://kc.example.internal/realms/first"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if got := readIssuer(t, pool, id); got != "https://kc.example.internal/realms/first" {
		t.Fatalf("after create = %q", got)
	}
	// A retry or a corrected realm overwrites the value; it never fails.
	if err := st.SetIssuer(ctx, id, "https://kc.example.internal/realms/second"); err != nil {
		t.Fatalf("update: %v", err)
	}
	if got := readIssuer(t, pool, id); got != "https://kc.example.internal/realms/second" {
		t.Fatalf("after update = %q", got)
	}
}

func TestIssuerStoreKeepsOtherConfiguration(t *testing.T) {
	pool := issuerTestPool(t)
	id := seedIssuerTenant(t, pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `UPDATE tenants SET configuration = '{"keep":"me"}'::jsonb WHERE id = $1`, id); err != nil {
		t.Fatalf("seed config: %v", err)
	}
	if err := (&PgIssuerStore{Pool: pool}).SetIssuer(ctx, id, "https://kc.example.internal/realms/x"); err != nil {
		t.Fatalf("SetIssuer: %v", err)
	}
	var keep string
	if err := pool.QueryRow(ctx, `SELECT configuration->>'keep' FROM tenants WHERE id = $1`, id).Scan(&keep); err != nil || keep != "me" {
		t.Fatalf("existing configuration lost: keep=%q err=%v", keep, err)
	}
}

func TestIssuerStoreUnknownTenantIsAnError(t *testing.T) {
	pool := issuerTestPool(t)
	if err := (&PgIssuerStore{Pool: pool}).SetIssuer(context.Background(), uuid.New().String(), "https://kc.example.internal/realms/x"); err == nil {
		t.Fatal("wrote an issuer for a tenant that does not exist")
	}
}
