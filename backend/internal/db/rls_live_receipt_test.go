package db

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

// TestRLS_LiveDeliberateBypass_AppRole documents and optionally runs the live
// receipt: connect as a LOGIN role that is NOT superuser / NOT BYPASSRLS
// (prefer UISCE_APP_DSN). If the DSN user is postgres/superuser, SET ROLE to
// uisce_mcp_app so FORCE RLS actually binds.
//
// Skips when no DSN is configured. Direct uisce_mcp_app TCP login may fail
// pg_hba from developer hosts; SET ROLE from DATABASE_URL remains valid proof.
func TestRLS_LiveDeliberateBypass_AppRole(t *testing.T) {
	dsn := os.Getenv("UISCE_APP_DSN")
	if dsn == "" {
		dsn = os.Getenv("UISCE_TEST_DB_DSN")
	}
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("no UISCE_APP_DSN / UISCE_TEST_DB_DSN / DATABASE_URL")
	}

	dbConn, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer dbConn.Close()
	if err := dbConn.Ping(); err != nil {
		t.Skipf("db unreachable: %v", err)
	}

	ctx := context.Background()
	tx, err := dbConn.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	var user string
	var super, bypass bool
	if err := tx.QueryRowContext(ctx, `
		SELECT current_user,
		       (SELECT rolsuper FROM pg_roles WHERE rolname = current_user),
		       (SELECT rolbypassrls FROM pg_roles WHERE rolname = current_user)`).Scan(&user, &super, &bypass); err != nil {
		t.Fatal(err)
	}
	if super || bypass || user == "postgres" {
		if _, err := tx.ExecContext(ctx, `SET LOCAL ROLE uisce_mcp_app`); err != nil {
			t.Fatalf("need non-superuser session; SET ROLE uisce_mcp_app failed: %v", err)
		}
		if err := tx.QueryRowContext(ctx, `
			SELECT current_user,
			       (SELECT rolsuper FROM pg_roles WHERE rolname = current_user),
			       (SELECT rolbypassrls FROM pg_roles WHERE rolname = current_user)`).Scan(&user, &super, &bypass); err != nil {
			t.Fatal(err)
		}
	}
	if super || bypass {
		t.Fatalf("effective role %s still super=%v bypass=%v — receipt invalid", user, super, bypass)
	}
	t.Logf("effective_role=%s super=%v bypass=%v", user, super, bypass)

	tenantA := "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	gold := "99e99e99-99e9-49e9-89e9-99e99e99e999"
	if err := ApplyTenantGUCs(ctx, tx, tenantA, gold); err != nil {
		t.Fatal(err)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT slug FROM public.page_definitions
		WHERE slug LIKE 'rls-fixture-%'
		ORDER BY slug`)
	if err != nil {
		t.Fatalf("query under FORCE RLS: %v (seed fixtures if missing)", err)
	}
	defer rows.Close()
	var seen []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		seen = append(seen, s)
	}
	if len(seen) == 0 {
		t.Skip("no rls-fixture-% pages seeded; run live SQL seed then re-run")
	}
	for _, s := range seen {
		if s == "rls-fixture-b" || s == "rls-fixture-gold-noncore" {
			t.Fatalf("LEAK under app role: unexpected slug %s in %v", s, seen)
		}
	}
	t.Logf("visible=%v (deliberate-bypass PASS if only a + gold-core)", seen)
}
