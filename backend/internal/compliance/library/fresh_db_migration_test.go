package library

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestCoreLibrary_FreshDatabaseMigrationChain(t *testing.T) {
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

	if dsn == "" {
		t.Skip("ALPHA_DSN not configured and certs not found; skipping fresh DB migration chain test")
		return
	}

	adminDB, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("Cannot open admin DB connection: %v", err)
		return
	}
	defer adminDB.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := adminDB.PingContext(ctx); err != nil {
		t.Skipf("Cannot ping alpha database: %v", err)
		return
	}

	// 1. Create temporary database
	tempDBName := fmt.Sprintf("fresh_mig_test_%d", time.Now().UnixNano())
	_, err = adminDB.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s", tempDBName))
	require.NoError(t, err)

	defer func() {
		// Terminate any active connections and drop the temporary database
		_, _ = adminDB.ExecContext(context.Background(), fmt.Sprintf(`
			SELECT pg_terminate_backend(pid) 
			FROM pg_stat_activity 
			WHERE datname = '%s' AND pid <> pg_backend_pid()
		`, tempDBName))
		_, _ = adminDB.ExecContext(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS %s", tempDBName))
	}()

	// 2. Connect to the fresh temporary database
	var freshDSN string
	if strings.Contains(dsn, "dbname=alpha") {
		freshDSN = strings.Replace(dsn, "dbname=alpha", fmt.Sprintf("dbname=%s", tempDBName), 1)
	} else if strings.Contains(dsn, "/alpha?") {
		freshDSN = strings.Replace(dsn, "/alpha?", fmt.Sprintf("/%s?", tempDBName), 1)
	} else {
		freshDSN = dsn + " dbname=" + tempDBName
	}

	freshDB, err := sql.Open("postgres", freshDSN)
	require.NoError(t, err)
	defer freshDB.Close()

	// 3. Locate migration directory
	migDir := filepath.Join("..", "..", "..", "db", "migrations")
	if _, err := os.Stat(migDir); err != nil {
		// Fallback search
		migDir = filepath.Join("backend", "db", "migrations")
	}

	// 4. Create prerequisites in public schema (tenants and catalog_node_types)
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
	_, err = freshDB.ExecContext(ctx, prereqSQL)
	require.NoError(t, err)

	// 5. Apply migrations 001 -> 009 UP in order
	upMigrations := []string{
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

	for _, migFile := range upMigrations {
		content, err := os.ReadFile(filepath.Join(migDir, migFile))
		require.NoError(t, err, "Migration file %s must exist", migFile)

		_, err = freshDB.ExecContext(ctx, string(content))
		require.NoError(t, err, "Migration %s UP must succeed on fresh database", migFile)
		t.Logf("Migration %s UP applied successfully on fresh database", migFile)
	}

	// 6. Assert Full Library State
	var ruleCount int
	err = freshDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM compliance.compliance_rule").Scan(&ruleCount)
	require.NoError(t, err)
	require.Equal(t, 50, ruleCount, "Fresh database migration must yield exactly 50 core rules")

	var versionCount int
	err = freshDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM compliance.compliance_rule_version").Scan(&versionCount)
	require.NoError(t, err)
	require.Equal(t, 50, versionCount, "Fresh database migration must yield exactly 50 rule version snapshots")

	var rulesetCount int
	err = freshDB.QueryRowContext(ctx, "SELECT COUNT(DISTINCT ruleset_code) FROM compliance.compliance_ruleset_membership").Scan(&rulesetCount)
	require.NoError(t, err)
	require.Equal(t, 3, rulesetCount, "Must seed 3 standard licensable rulesets")

	var membershipCount int
	err = freshDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM compliance.compliance_ruleset_membership").Scan(&membershipCount)
	require.NoError(t, err)
	require.Equal(t, 50, membershipCount, "All 50 rules must be mapped to ruleset memberships")

	var regCaseTableCount int
	err = freshDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'compliance' AND table_name IN ('regulatory_change_case', 'regulatory_case_event', 'compliance_notification', 'regulatory_draft_rule')").Scan(&regCaseTableCount)
	require.NoError(t, err)
	require.Equal(t, 4, regCaseTableCount, "Migration 007 and 008 tables must exist")

	var survTableCount int
	err = freshDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'compliance' AND table_name IN ('compliance_surveillance_finding', 'compliance_surveillance_event')").Scan(&survTableCount)
	require.NoError(t, err)
	require.Equal(t, 2, survTableCount, "Migration 009 surveillance tables must exist")

	t.Logf("Fresh DB Integrity Assertions Passed: Rules=%d, Versions=%d, Rulesets=%d, Memberships=%d, RegTables=%d, SurvTables=%d",
		ruleCount, versionCount, rulesetCount, membershipCount, regCaseTableCount, survTableCount)

	// 7. Test Structural Mutation Guard Trigger
	var sampleRuleID string
	err = freshDB.QueryRowContext(ctx, "SELECT id FROM compliance.compliance_rule WHERE rule_code = 'UCITS_ISSUER_5'").Scan(&sampleRuleID)
	require.NoError(t, err)

	// Unauthorized update without version snapshot must fail
	_, err = freshDB.ExecContext(ctx, "UPDATE compliance.compliance_rule SET citation = 'Unauthorized Mutation' WHERE id = $1", sampleRuleID)
	require.Error(t, err, "Mutation without snapshot must be rejected by trigger")
	require.Contains(t, err.Error(), "Audit Violation: Mutation of compliance_rule")

	t.Logf("Structural Mutation Guard Trigger verified: unauthorized mutation correctly blocked!")

	// 8. Test Surveillance Finding State Machine & Append-Only Event Guard
	goldTenantUUID := "99e99e99-99e9-49e9-89e9-99e99e99e999"
	var findingID string
	err = freshDB.QueryRowContext(ctx, `
		INSERT INTO compliance.compliance_surveillance_finding (
			tenant_id, detector_type, severity, status, dedup_key, title, description, entity_type,
			activity_window_start, activity_window_end
		) VALUES (
			$1, 'WASH_SALE', 'HIGH', 'OPEN', 'dedup_test_001', 'Wash sale detected', 'Simulated wash sale', 'BENEFICIAL_OWNER',
			now() - interval '30 days', now()
		) RETURNING id
	`, goldTenantUUID).Scan(&findingID)
	require.NoError(t, err)

	// Valid transition OPEN -> IN_REVIEW
	_, err = freshDB.ExecContext(ctx, "UPDATE compliance.compliance_surveillance_finding SET status = 'IN_REVIEW' WHERE id = $1", findingID)
	require.NoError(t, err)

	// Invalid transition IN_REVIEW -> DISMISSED without resolution notes must fail
	_, err = freshDB.ExecContext(ctx, "UPDATE compliance.compliance_surveillance_finding SET status = 'DISMISSED' WHERE id = $1", findingID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "requires resolution_notes")

	// Valid transition with resolution notes
	_, err = freshDB.ExecContext(ctx, `
		UPDATE compliance.compliance_surveillance_finding 
		SET status = 'DISMISSED', resolution_notes = 'False positive non-substantially identical instrument', resolved_by = 'officer_1'
		WHERE id = $1
	`, findingID)
	require.NoError(t, err)

	// Test append-only event insertion
	var eventID string
	err = freshDB.QueryRowContext(ctx, `
		INSERT INTO compliance.compliance_surveillance_event (
			finding_id, tenant_id, event_type, actor, payload
		) VALUES (
			$1, $2, 'DETECTED', 'streaming_consumer', '{"qty": 5000}'::jsonb
		) RETURNING id
	`, findingID, goldTenantUUID).Scan(&eventID)
	require.NoError(t, err)

	// Mutation of event must fail
	_, err = freshDB.ExecContext(ctx, "DELETE FROM compliance.compliance_surveillance_event WHERE id = $1", eventID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "compliance.compliance_surveillance_event is strictly append-only")

	t.Logf("Surveillance Finding State Machine & Append-Only Event Guard verified!")

	// 9. Rollback Cycle: 009 -> 001 DOWN
	downMigrations := []string{
		"20261220_009_compliance_surveillance_findings.down.sql",
		"20261219_008_trigger_refactor_and_draft_guard.down.sql",
		"20261218_007_regulatory_change_workflow.down.sql",
		"20261218_006_rule_version_snapshots.down.sql",
		"20261218_005_core_rule_library_seed.down.sql",
		"20261218_004_core_rule_library.down.sql",
		"20261218_003_compliance_ingest_lsn.down.sql",
		"20261218_002_governance_audit_and_privileges.down.sql",
		"20261218_001_compliance_engine_core_tables.down.sql",
	}

	for _, migFile := range downMigrations {
		content, err := os.ReadFile(filepath.Join(migDir, migFile))
		require.NoError(t, err, "Down migration file %s must exist", migFile)

		_, err = freshDB.ExecContext(ctx, string(content))
		require.NoError(t, err, "Migration %s DOWN must succeed on fresh database", migFile)
		t.Logf("Migration %s DOWN executed cleanly", migFile)
	}

	t.Logf("FRESH DATABASE MIGRATION CHAIN & ROLLBACK CYCLE 001 <-> 009 FULLY VERIFIED!")
}
