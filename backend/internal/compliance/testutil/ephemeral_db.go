package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

var (
	templateInitOnce sync.Once
	templateInitErr  error
	templateDBName   = "template_compliance_pristine"
)

func GetAdminDSN() string {
	dsn := os.Getenv("ALPHA_DSN")
	if dsn == "" {
		home, _ := os.UserHomeDir()
		caPath := filepath.Join(home, ".uisce/certs/ca.crt")
		certPath := filepath.Join(home, ".uisce/certs/postgres-client.crt")
		keyPath := filepath.Join(home, ".uisce/certs/postgres-client.key")

		if _, err := os.Stat(caPath); err == nil {
			dsn = fmt.Sprintf("host=100.84.50.65 port=5432 user=postgres password=postgres dbname=alpha sslmode=verify-full sslrootcert=%s sslcert=%s sslkey=%s", caPath, certPath, keyPath)
		}
	}
	return dsn
}

func ensureTemplateDatabase(adminDB *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var exists bool
	err := adminDB.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", templateDBName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check template db existence: %w", err)
	}

	if exists {
		return nil
	}

	// Create template database
	_, err = adminDB.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s", templateDBName))
	if err != nil {
		return fmt.Errorf("create template database: %w", err)
	}

	adminDSN := GetAdminDSN()
	templateDSN := strings.Replace(adminDSN, "dbname=alpha", fmt.Sprintf("dbname=%s", templateDBName), 1)
	if !strings.Contains(adminDSN, "dbname=alpha") {
		templateDSN = adminDSN + " dbname=" + templateDBName
	}

	tplDB, err := sql.Open("postgres", templateDSN)
	if err != nil {
		return fmt.Errorf("open template db: %w", err)
	}
	defer tplDB.Close()

	// Apply prerequisites
	prereqSQL := `
		CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
		CREATE EXTENSION IF NOT EXISTS "pgcrypto";

		CREATE TABLE IF NOT EXISTS public.tenants (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL,
			display_name VARCHAR(255) NOT NULL,
			description TEXT,
			gold_copy BOOLEAN DEFAULT false,
			is_active BOOLEAN DEFAULT true,
			status VARCHAR(50) DEFAULT 'active',
			plan VARCHAR(50) DEFAULT 'enterprise',
			is_suspended BOOLEAN DEFAULT false,
			is_deleted BOOLEAN DEFAULT false,
			created_at TIMESTAMPTZ DEFAULT now(),
			updated_at TIMESTAMPTZ DEFAULT now()
		);

		CREATE UNIQUE INDEX IF NOT EXISTS unq_tenants_gold_copy 
			ON public.tenants (gold_copy) 
			WHERE gold_copy = true;

		CREATE TABLE IF NOT EXISTS public.catalog_node_types (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID,
			catalog_type_name VARCHAR(100) NOT NULL,
			description TEXT,
			is_active BOOLEAN DEFAULT true,
			config JSONB DEFAULT '{}'::jsonb,
			created_at TIMESTAMPTZ DEFAULT now(),
			updated_at TIMESTAMPTZ DEFAULT now()
		);

		CREATE TABLE IF NOT EXISTS public.catalog_nodes (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL,
			node_type_id UUID,
			parent_id UUID,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			metadata JSONB DEFAULT '{}'::jsonb,
			created_at TIMESTAMPTZ DEFAULT now(),
			updated_at TIMESTAMPTZ DEFAULT now()
		);

		DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_user') THEN
				CREATE ROLE app_user;
			END IF;
		END $$;

		CREATE OR REPLACE FUNCTION public.uisce_gold_copy_tenant_id() RETURNS UUID AS $$
		BEGIN
			RETURN '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid;
		END;
		$$ LANGUAGE plpgsql IMMUTABLE;

		INSERT INTO public.tenants (id, name, display_name, gold_copy, is_active, status, plan, is_suspended, is_deleted)
		VALUES ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'northwind', 'Northwind Traders', true, true, 'active', 'enterprise', false, false)
		ON CONFLICT (id) DO NOTHING;
	`
	_, err = tplDB.ExecContext(ctx, prereqSQL)
	if err != nil {
		return fmt.Errorf("apply prereqs on template db: %w", err)
	}

	// Apply migrations 001 -> 009
	migrations := []string{
		"20261218_001_compliance_engine_core_tables.up.sql",
		"20261218_002_governance_audit_and_privileges.up.sql",
		"20261218_003_compliance_ingest_lsn.up.sql",
		"20261218_004_core_rule_library.up.sql",
		"20261218_005_core_rule_library_seed.up.sql",
		"20261218_006_rule_version_snapshots.up.sql",
		"20261218_007_regulatory_change_workflow.up.sql",
		"20261219_008_trigger_refactor_and_draft_guard.up.sql",
		"20261220_009_compliance_surveillance_findings.up.sql",
	}

	// Search for migration directory
	migDir := filepath.Join("..", "..", "..", "db", "migrations")
	if _, err := os.Stat(migDir); err != nil {
		migDir = filepath.Join("..", "..", "db", "migrations")
		if _, err := os.Stat(migDir); err != nil {
			migDir = filepath.Join("backend", "db", "migrations")
		}
	}

	for _, migFile := range migrations {
		content, err := os.ReadFile(filepath.Join(migDir, migFile))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", migFile, err)
		}
		_, err = tplDB.ExecContext(ctx, string(content))
		if err != nil {
			return fmt.Errorf("apply migration %s on template: %w", migFile, err)
		}
	}

	return nil
}

// GetEphemeralTestDB creates a fast, isolated ephemeral PostgreSQL database cloned from the canonical template.
// When the test or benchmark completes, tb.Cleanup drops the ephemeral database.
func GetEphemeralTestDB(tb testing.TB) *sql.DB {
	if tb != nil {
		tb.Helper()
	}

	adminDSN := GetAdminDSN()
	if adminDSN == "" {
		if tb != nil {
			tb.Skip("ALPHA_DSN not set and mTLS certificates not found; skipping integration test")
		}
		return nil
	}

	adminDB, err := sql.Open("postgres", adminDSN)
	if err != nil {
		if tb != nil {
			tb.Skipf("Cannot open admin db: %v", err)
		}
		return nil
	}

	// Ensure template DB is initialized once
	templateInitOnce.Do(func() {
		templateInitErr = ensureTemplateDatabase(adminDB)
	})
	if templateInitErr != nil {
		if tb != nil {
			tb.Fatalf("Failed to initialize template database: %v", templateInitErr)
		}
		return nil
	}

	tempDBName := fmt.Sprintf("test_comp_%d", time.Now().UnixNano()%100000000)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, err = adminDB.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s TEMPLATE %s", tempDBName, templateDBName))
	if err != nil {
		if tb != nil {
			tb.Fatalf("Failed to create ephemeral database %s: %v", tempDBName, err)
		}
		return nil
	}

	tempDSN := strings.Replace(adminDSN, "dbname=alpha", fmt.Sprintf("dbname=%s", tempDBName), 1)
	if !strings.Contains(adminDSN, "dbname=alpha") {
		tempDSN = adminDSN + " dbname=" + tempDBName
	}

	testDB, err := sql.Open("postgres", tempDSN)
	if err != nil {
		if tb != nil {
			tb.Fatalf("Failed to open connection to ephemeral database %s: %v", tempDBName, err)
		}
		return nil
	}

	cleanup := func() {
		_ = testDB.Close()
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()

		_, _ = adminDB.ExecContext(cleanupCtx, fmt.Sprintf(`
			SELECT pg_terminate_backend(pid) 
			FROM pg_stat_activity 
			WHERE datname = '%s' AND pid <> pg_backend_pid()
		`, tempDBName))
		_, _ = adminDB.ExecContext(cleanupCtx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", tempDBName))
		_ = adminDB.Close()
	}

	if tb != nil {
		tb.Cleanup(cleanup)
	}

	return testDB
}
