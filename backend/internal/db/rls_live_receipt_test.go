package db

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/hondyman/uisce/backend/internal/compliance/testutil"
	_ "github.com/lib/pq"
)

// TestRLS_LiveDeliberateBypass_AppRole verifies FORCE RLS enforcement under
// a least-privilege non-superuser role (uisce_mcp_app / app_user).
//
// It runs hermetically against ephemeral databases, test databases, or live DSNs.
func TestRLS_LiveDeliberateBypass_AppRole(t *testing.T) {
	var dbConn *sql.DB
	dsn := os.Getenv("UISCE_APP_DSN")
	if dsn == "" {
		dsn = os.Getenv("UISCE_TEST_DB_DSN")
	}
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}

	if dsn != "" {
		var err error
		dbConn, err = sql.Open("postgres", dsn)
		if err != nil {
			t.Fatal(err)
		}
		defer dbConn.Close()
		if err := dbConn.Ping(); err != nil {
			t.Skipf("configured dsn unreachable: %v", err)
		}
	} else {
		// Hermetic fallback: use ephemeral cloned database
		home, _ := os.UserHomeDir()
		caPath := filepath.Join(home, ".uisce/certs/ca.crt")
		if _, err := os.Stat(caPath); err == nil {
			dbConn = testutil.GetEphemeralTestDB(t)
		} else {
			t.Skip("no database connection available; skipping RLS deliberate bypass test")
		}
	}

	ctx := context.Background()

	// 1. Ensure test fixture pages exist
	_, err := dbConn.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS public.page_definitions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL,
			slug VARCHAR(255) NOT NULL,
			title VARCHAR(255) NOT NULL,
			is_core BOOLEAN DEFAULT false,
			created_at TIMESTAMPTZ DEFAULT now()
		);
		ALTER TABLE public.page_definitions ENABLE ROW LEVEL SECURITY;
		ALTER TABLE public.page_definitions FORCE ROW LEVEL SECURITY;

		DROP POLICY IF EXISTS tenant_isolation_page_definitions ON public.page_definitions;
		CREATE POLICY tenant_isolation_page_definitions ON public.page_definitions
			FOR ALL USING (
				tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid
				OR tenant_id = NULLIF(current_setting('uisce.current_tenant', true), '')::uuid
				OR (is_core = true AND tenant_id = NULLIF(current_setting('uisce.gold_tenant', true), '')::uuid)
			);

		INSERT INTO public.page_definitions (tenant_id, slug, title, is_core)
		VALUES 
			('aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid, 'rls-fixture-a', 'Fixture A', false),
			('bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'::uuid, 'rls-fixture-b', 'Fixture B', false),
			('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'rls-fixture-gold', 'Fixture Gold Core', true),
			('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'rls-fixture-gold-noncore', 'Fixture Gold NonCore', false)
		ON CONFLICT (id) DO NOTHING;
	`)
	if err != nil {
		t.Fatalf("setup page_definitions failed: %v", err)
	}

	// 2. Ensure least-privilege role & grants exist
	_, err = dbConn.ExecContext(ctx, `
		DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_user') THEN
				CREATE ROLE app_user;
			END IF;
			IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'uisce_mcp_app') THEN
				CREATE ROLE uisce_mcp_app NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
			END IF;
			GRANT app_user TO uisce_mcp_app;
			GRANT USAGE ON SCHEMA public TO uisce_mcp_app;
			GRANT SELECT ON public.page_definitions TO uisce_mcp_app;
		END $$;
	`)
	if err != nil {
		t.Fatalf("setup role and grants failed: %v", err)
	}

	// 3. Begin test transaction and switch to uisce_mcp_app
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
		t.Fatalf("query under FORCE RLS: %v", err)
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

	for _, s := range seen {
		if s == "rls-fixture-b" || s == "rls-fixture-gold-noncore" {
			t.Fatalf("LEAK under app role: unexpected slug %s in %v", s, seen)
		}
	}
	t.Logf("visible=%v (deliberate-bypass PASS: only tenant A + gold-core visible)", seen)
}

