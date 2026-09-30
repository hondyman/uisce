package mdm

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getLivePostgresDB(t *testing.T) *sql.DB {
	if os.Getenv("BP_LIVE_SMOKE") != "1" {
		t.Skip("skipping live postgres RLS smoke test: set BP_LIVE_SMOKE=1 to run against alpha")
	}

	home, err := os.UserHomeDir()
	require.NoError(t, err)

	certDir := filepath.Join(home, ".uisce", "certs")
	caCertPath := filepath.Join(certDir, "ca.crt")
	clientCertPath := filepath.Join(certDir, "postgres-client.crt")
	clientKeyPath := filepath.Join(certDir, "postgres-client.key")

	caCert, err := os.ReadFile(caCertPath)
	require.NoError(t, err, "failed to read ca.crt")

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	cert, err := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
	require.NoError(t, err, "failed to load postgres client key pair")

	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}

	connStr := fmt.Sprintf("postgres://postgres:postgres@100.84.50.65:5432/alpha?sslmode=verify-full")
	config, err := pgx.ParseConfig(connStr)
	require.NoError(t, err)
	config.TLSConfig = tlsConfig

	db := stdlib.OpenDB(*config)
	return db
}

func TestLiveRuleResolver_RLS_Enforcement(t *testing.T) {
	db := getLivePostgresDB(t)
	defer db.Close()

	ctx := context.Background()

	// 1. Identify Core Tenant ID live
	var coreTenantID uuid.UUID
	err := db.QueryRowContext(ctx, "SELECT public.get_core_tenant_id()").Scan(&coreTenantID)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, coreTenantID, "core tenant ID must exist")

	tenantA := uuid.New()
	tenantB := uuid.New()
	dummyTermID := uuid.New()

	// We run inside a transaction with rollback to keep the live DB pristine
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback() //nolint:errcheck

	// Seed one rule for Core, one for Tenant A, and one for Tenant B (as admin/postgres)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO public.semantic_survivorship_rules (
			id, tenant_id, entity_type, semantic_term_id, strategy, priority_order, is_active
		) VALUES 
			(gen_random_uuid(), $1, 'ACCOUNT', $2, 'SOURCE_PRIORITY', ARRAY['CORE_SYSTEM'], true),
			(gen_random_uuid(), $3, 'ACCOUNT', $2, 'SOURCE_PRIORITY', ARRAY['TENANT_A_FEED'], true),
			(gen_random_uuid(), $4, 'ACCOUNT', $2, 'SOURCE_PRIORITY', ARRAY['TENANT_B_FEED'], true)
	`, coreTenantID, dummyTermID, tenantA, tenantB)
	require.NoError(t, err)

	// Switch session role to app_user (non-superuser) so PostgreSQL RLS is actively enforced
	_, err = tx.ExecContext(ctx, "SET ROLE app_user")
	require.NoError(t, err)

	// Step 1: SET LOCAL app.current_tenant to Tenant A
	_, err = tx.ExecContext(ctx, fmt.Sprintf("SET LOCAL app.current_tenant = '%s'", tenantA))
	require.NoError(t, err)

	// Step 2: Tenant A should see its own row and the core tenant's row
	var countVisible int
	err = tx.QueryRowContext(ctx, `
		SELECT count(*) FROM public.semantic_survivorship_rules 
		WHERE semantic_term_id = $1
	`, dummyTermID).Scan(&countVisible)
	require.NoError(t, err)
	assert.Equal(t, 2, countVisible, "Tenant A should see exactly 2 rows (own + core)")

	// Step 3: Direct query for Tenant B's rule under Tenant A context MUST return 0 rows
	var tenantBVisible int
	err = tx.QueryRowContext(ctx, `
		SELECT count(*) FROM public.semantic_survivorship_rules 
		WHERE tenant_id = $1 AND semantic_term_id = $2
	`, tenantB, dummyTermID).Scan(&tenantBVisible)
	require.NoError(t, err)
	assert.Equal(t, 0, tenantBVisible, "Tenant A MUST NOT see Tenant B rows (blocked by RLS)")

	// Step 4: Write attempt to Tenant B under Tenant A context MUST be rejected by RLS
	_, err = tx.ExecContext(ctx, `
		INSERT INTO public.semantic_survivorship_rules (
			id, tenant_id, entity_type, semantic_term_id, strategy, is_active
		) VALUES (gen_random_uuid(), $1, 'ACCOUNT', gen_random_uuid(), 'MOST_RECENT', true)
	`, tenantB)
	require.Error(t, err, "Write to Tenant B from Tenant A session must violate RLS")
	assert.Contains(t, err.Error(), "row-level security policy", "Error must be RLS violation")

	// Rollback dirty tx and start clean tx for unscoped test
	_ = tx.Rollback()

	tx2, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx2.Rollback() //nolint:errcheck

	_, err = tx2.ExecContext(ctx, "SET ROLE app_user")
	require.NoError(t, err)

	// Step 5: With NO tenant setting (unscoped batch resolver context), only core rows are visible
	_, err = tx2.ExecContext(ctx, "SET LOCAL app.current_tenant = ''")
	require.NoError(t, err)

	var unscopedCount int
	err = tx2.QueryRowContext(ctx, `
		SELECT count(*) FROM public.semantic_survivorship_rules 
		WHERE tenant_id = $1
	`, coreTenantID).Scan(&unscopedCount)
	require.NoError(t, err)
	assert.True(t, unscopedCount > 0, "Unscoped context should see core rows via get_core_tenant_id()")

	// Step 6: Malformed / garbage tenant setting fails cleanly with invalid uuid syntax
	_, err = tx2.ExecContext(ctx, "SET LOCAL app.current_tenant = 'malformed-not-a-uuid'")
	require.NoError(t, err)

	var garbageCount int
	err = tx2.QueryRowContext(ctx, "SELECT count(*) FROM public.semantic_survivorship_rules").Scan(&garbageCount)
	require.Error(t, err, "Malformed tenant UUID string must fail cleanly on uuid cast")
	assert.Contains(t, err.Error(), "invalid input syntax for type uuid")
}

func TestLiveApplyPendingMigrations(t *testing.T) {
	db := getLivePostgresDB(t)
	defer db.Close()

	ctx := context.Background()
	migration004, err := os.ReadFile("../../db/migrations/20261130_004_create_bp_workflow_run.up.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(migration004))
	require.NoError(t, err, "Migration 20261130_004 should apply cleanly to alpha")

	migration006, err := os.ReadFile("../../db/migrations/20261130_006_finalize_mdm_and_workflow_extensions.up.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(migration006))
	require.NoError(t, err, "Migration 20261130_006 should apply cleanly to alpha")

	migration007, err := os.ReadFile("../../db/migrations/20261130_007_bp_process_definition_extensions_and_rls.up.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(migration007))
	require.NoError(t, err, "Migration 20261130_007 should apply cleanly to alpha")

	migration008, err := os.ReadFile("../../db/migrations/20261130_008_create_bp_trigger_subscriptions.up.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(migration008))
	require.NoError(t, err, "Migration 20261130_008 should apply cleanly to alpha")

	// Verify bp_trigger_subscriptions table exists
	var trigTableExists bool
	err = db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_name = 'bp_trigger_subscriptions'
		)
	`).Scan(&trigTableExists)
	require.NoError(t, err)
	assert.True(t, trigTableExists, "bp_trigger_subscriptions table must exist")

	// Verify columns exist on bp_process_definition
	var colCount int
	err = db.QueryRowContext(ctx, `
		SELECT count(*) FROM information_schema.columns 
		WHERE table_name = 'bp_process_definition'
		  AND column_name IN ('source_type', 'base_definition_id', 'base_version', 'extensions_json')
	`).Scan(&colCount)
	require.NoError(t, err)
	assert.Equal(t, 4, colCount, "All 4 extension columns must exist on bp_process_definition")

	// Verify RLS is enabled on bp_process_definition
	var rlsEnabled bool
	err = db.QueryRowContext(ctx, `
		SELECT rowsecurity FROM pg_tables 
		WHERE schemaname = 'public' AND tablename = 'bp_process_definition';
	`).Scan(&rlsEnabled)
	require.NoError(t, err)
	assert.True(t, rlsEnabled, "RLS must be enabled on bp_process_definition")

	policyRows, err := db.QueryContext(ctx, `
		SELECT policyname, cmd, qual, with_check 
		FROM pg_policies 
		WHERE schemaname = 'public' AND tablename = 'bp_process_definition';
	`)
	require.NoError(t, err)
	defer policyRows.Close()

	for policyRows.Next() {
		var pname, pcmd string
		var qual, wcheck sql.NullString
		require.NoError(t, policyRows.Scan(&pname, &pcmd, &qual, &wcheck))
		t.Logf("Existing Policy: %s (%s) QUAL: %s WITH_CHECK: %s", pname, pcmd, qual.String, wcheck.String)
	}
}

func TestLiveBPProcessDefinition_RLS_Enforcement(t *testing.T) {
	db := getLivePostgresDB(t)
	defer db.Close()

	ctx := context.Background()

	// 1. Core Tenant ID
	var coreTenantID uuid.UUID
	err := db.QueryRowContext(ctx, "SELECT public.get_core_tenant_id()").Scan(&coreTenantID)
	require.NoError(t, err)

	tenantA := uuid.New().String()
	tenantB := uuid.New().String()

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback() //nolint:errcheck

	// Seed Core, Tenant A, and Tenant B process definitions as postgres admin
	_, err = tx.ExecContext(ctx, `
		INSERT INTO public.bp_process_definition (
			id, process_id, tenant_id, version, name, steps_json, graph_json, source_type
		) VALUES 
			(gen_random_uuid(), 'proc-core-mdm', $1, 1, 'Core MDM Process', '[]'::jsonb, '{}'::jsonb, 'CORE'),
			(gen_random_uuid(), 'proc-tenant-a', $2, 1, 'Tenant A Custom Process', '[]'::jsonb, '{}'::jsonb, 'EXTENDED'),
			(gen_random_uuid(), 'proc-tenant-b', $3, 1, 'Tenant B Custom Process', '[]'::jsonb, '{}'::jsonb, 'EXTENDED');
	`, coreTenantID.String(), tenantA, tenantB)
	require.NoError(t, err)

	// Switch session role to app_user (non-superuser)
	_, err = tx.ExecContext(ctx, "SET ROLE app_user")
	require.NoError(t, err)

	// Step 1: Set context to Tenant A
	_, err = tx.ExecContext(ctx, fmt.Sprintf("SET LOCAL app.current_tenant = '%s'", tenantA))
	require.NoError(t, err)

	// Tenant A sees own process + Core process (2 processes)
	var countVisible int
	err = tx.QueryRowContext(ctx, `
		SELECT count(*) FROM public.bp_process_definition 
		WHERE process_id IN ('proc-core-mdm', 'proc-tenant-a', 'proc-tenant-b')
	`).Scan(&countVisible)
	require.NoError(t, err)
	assert.Equal(t, 2, countVisible, "Tenant A must see exactly 2 processes (own + Core)")

	// Step 2: Tenant A MUST NOT see Tenant B's process
	var tenantBVisible int
	err = tx.QueryRowContext(ctx, `
		SELECT count(*) FROM public.bp_process_definition 
		WHERE process_id = 'proc-tenant-b'
	`).Scan(&tenantBVisible)
	require.NoError(t, err)
	assert.Equal(t, 0, tenantBVisible, "Tenant A cannot see Tenant B process (blocked by RLS)")

	// Step 3: Tenant A write to Tenant B rejected by RLS
	_, err = tx.ExecContext(ctx, `
		INSERT INTO public.bp_process_definition (
			id, process_id, tenant_id, version, name, steps_json, graph_json
		) VALUES (gen_random_uuid(), 'proc-hacked', $1, 1, 'Illegal Write', '[]'::jsonb, '{}'::jsonb)
	`, tenantB)
	require.Error(t, err, "Cross-tenant write to bp_process_definition must violate RLS")
	assert.Contains(t, err.Error(), "row-level security policy")
}

func TestLiveWorkerRuleResolver_RLS_Regression(t *testing.T) {
	db := getLivePostgresDB(t)
	defer db.Close()

	ctx := context.Background()
	testTenantID := uuid.New()

	// Set database session role to app_user (non-superuser with RLS active)
	// Clear any session-level GUC to simulate un-initialized worker pool connection
	_, err := db.ExecContext(ctx, "SET ROLE app_user; SET app.current_tenant = '';")
	require.NoError(t, err)

	// Test RuleResolver under un-initialized worker connection
	resolver := NewRuleResolver(db)
	rules, err := resolver.ResolveEntityRules(ctx, testTenantID, "ACCOUNT")
	require.NoError(t, err, "RuleResolver must resolve rules under non-superuser app_user without pre-set GUC")
	assert.NotEmpty(t, rules, "Rules must be returned via internal SET LOCAL scoping")
}







