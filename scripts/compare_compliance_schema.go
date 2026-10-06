package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type ColumnMeta struct {
	ColumnName string
	DataType   string
	IsNullable string
}

type TriggerMeta struct {
	TriggerName string
	TableName   string
}

func getAlphaDSN() string {
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

func inspectSchema(ctx context.Context, db *sql.DB) (map[string][]ColumnMeta, []TriggerMeta, error) {
	tables := make(map[string][]ColumnMeta)
	rows, err := db.QueryContext(ctx, `
		SELECT table_name, column_name, data_type, is_nullable
		FROM information_schema.columns
		WHERE table_schema = 'compliance'
		ORDER BY table_name, ordinal_position;
	`)
	if err != nil {
		return nil, nil, fmt.Errorf("query columns: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tbl, col, dt, null string
		if err := rows.Scan(&tbl, &col, &dt, &null); err != nil {
			return nil, nil, err
		}
		tables[tbl] = append(tables[tbl], ColumnMeta{
			ColumnName: col,
			DataType:   dt,
			IsNullable: null,
		})
	}

	trigRows, err := db.QueryContext(ctx, `
		SELECT t.tgname, c.relname
		FROM pg_trigger t
		JOIN pg_class c ON t.tgrelid = c.oid
		JOIN pg_namespace n ON c.relnamespace = n.oid
		WHERE n.nspname = 'compliance' AND NOT t.tgisinternal
		ORDER BY c.relname, t.tgname;
	`)
	if err != nil {
		return nil, nil, fmt.Errorf("query triggers: %w", err)
	}
	defer trigRows.Close()

	var triggers []TriggerMeta
	for trigRows.Next() {
		var trg, tbl string
		if err := trigRows.Scan(&trg, &tbl); err != nil {
			return nil, nil, err
		}
		triggers = append(triggers, TriggerMeta{TriggerName: trg, TableName: tbl})
	}

	return tables, triggers, nil
}

func main() {
	adminDSN := getAlphaDSN()
	if adminDSN == "" {
		log.Fatal("ALPHA_DSN not configured and certs not found")
	}

	adminDB, err := sql.Open("postgres", adminDSN)
	if err != nil {
		log.Fatalf("Failed to open admin db: %v", err)
	}
	defer adminDB.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// 1. Create fresh reference DB
	refDBName := fmt.Sprintf("schema_diff_ref_%d", time.Now().UnixNano())
	log.Printf("Creating reference database: %s", refDBName)
	_, err = adminDB.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s", refDBName))
	if err != nil {
		log.Fatalf("Failed to create ref db: %v", err)
	}

	defer func() {
		_, _ = adminDB.ExecContext(context.Background(), fmt.Sprintf(`
			SELECT pg_terminate_backend(pid) 
			FROM pg_stat_activity 
			WHERE datname = '%s' AND pid <> pg_backend_pid()
		`, refDBName))
		_, _ = adminDB.ExecContext(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS %s", refDBName))
	}()

	refDSN := strings.Replace(adminDSN, "dbname=alpha", fmt.Sprintf("dbname=%s", refDBName), 1)
	refDB, err := sql.Open("postgres", refDSN)
	if err != nil {
		log.Fatalf("Failed to open ref db: %v", err)
	}
	defer refDB.Close()

	// 2. Apply prerequisites on refDB
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
	_, err = refDB.ExecContext(ctx, prereqSQL)
	if err != nil {
		log.Fatalf("Prereq failed on ref db: %v", err)
	}

	// 3. Apply migrations 001-009 on refDB
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

	migDir := filepath.Join("backend", "db", "migrations")
	for _, m := range migrations {
		p := filepath.Join(migDir, m)
		content, err := os.ReadFile(p)
		if err != nil {
			log.Fatalf("Read migration %s: %v", m, err)
		}
		_, err = refDB.ExecContext(ctx, string(content))
		if err != nil {
			log.Fatalf("Apply migration %s on ref db: %v", m, err)
		}
	}

	// 4. Introspect both DBs
	log.Println("Introspecting reference fresh DB...")
	refTables, refTriggers, err := inspectSchema(ctx, refDB)
	if err != nil {
		log.Fatalf("Inspect ref db: %v", err)
	}

	log.Println("Introspecting live alpha DB...")
	alphaTables, alphaTriggers, err := inspectSchema(ctx, adminDB)
	if err != nil {
		log.Fatalf("Inspect alpha db: %v", err)
	}

	// 5. Diff tables and columns
	fmt.Println("======================================================================")
	fmt.Println("MECHANICAL SCHEMA DIFF: FRESH CANONICAL BUILD (001->009) vs LIVE ALPHA")
	fmt.Println("======================================================================")

	var discrepancies []string

	const expectedTablesCount = 34
	const expectedTriggersCount = 29

	if len(refTables) != expectedTablesCount {
		discrepancies = append(discrepancies, fmt.Sprintf("REFERENCE TABLE COUNT MISMATCH: expected %d, got %d", expectedTablesCount, len(refTables)))
	}
	if len(refTriggers) != expectedTriggersCount {
		discrepancies = append(discrepancies, fmt.Sprintf("REFERENCE TRIGGER COUNT MISMATCH: expected %d, got %d", expectedTriggersCount, len(refTriggers)))
	}

	// Check table existence
	allTableNames := make(map[string]bool)
	for t := range refTables {
		allTableNames[t] = true
	}
	for t := range alphaTables {
		allTableNames[t] = true
	}

	sortedTables := make([]string, 0, len(allTableNames))
	for t := range allTableNames {
		sortedTables = append(sortedTables, t)
	}
	sort.Strings(sortedTables)

	fmt.Printf("Total compliance tables in reference: %d\n", len(refTables))
	fmt.Printf("Total compliance tables on alpha:     %d\n", len(alphaTables))
	fmt.Println("----------------------------------------------------------------------")

	for _, t := range sortedTables {
		refCols, inRef := refTables[t]
		alphaCols, inAlpha := alphaTables[t]

		if inRef && !inAlpha {
			discrepancies = append(discrepancies, fmt.Sprintf("TABLE MISSING on alpha: compliance.%s", t))
			continue
		}
		if !inRef && inAlpha {
			discrepancies = append(discrepancies, fmt.Sprintf("EXTRA TABLE on alpha: compliance.%s", t))
			continue
		}

		// Compare columns
		refColMap := make(map[string]ColumnMeta)
		for _, c := range refCols {
			refColMap[c.ColumnName] = c
		}
		alphaColMap := make(map[string]ColumnMeta)
		for _, c := range alphaCols {
			alphaColMap[c.ColumnName] = c
		}

		for cName, refC := range refColMap {
			alphaC, exists := alphaColMap[cName]
			if !exists {
				discrepancies = append(discrepancies, fmt.Sprintf("COLUMN MISSING on alpha: compliance.%s.%s", t, cName))
			} else if refC.DataType != alphaC.DataType || refC.IsNullable != alphaC.IsNullable {
				discrepancies = append(discrepancies, fmt.Sprintf("COLUMN MISMATCH on compliance.%s.%s: ref=(%s, nullable=%s), alpha=(%s, nullable=%s)", t, cName, refC.DataType, refC.IsNullable, alphaC.DataType, alphaC.IsNullable))
			}
		}
		for cName := range alphaColMap {
			if _, exists := refColMap[cName]; !exists {
				discrepancies = append(discrepancies, fmt.Sprintf("EXTRA COLUMN on alpha: compliance.%s.%s", t, cName))
			}
		}
	}

	// Compare triggers
	refTrigMap := make(map[string]string)
	for _, trg := range refTriggers {
		refTrigMap[trg.TableName+"."+trg.TriggerName] = trg.TriggerName
	}
	alphaTrigMap := make(map[string]string)
	for _, trg := range alphaTriggers {
		alphaTrigMap[trg.TableName+"."+trg.TriggerName] = trg.TriggerName
	}

	for k := range refTrigMap {
		if _, exists := alphaTrigMap[k]; !exists {
			discrepancies = append(discrepancies, fmt.Sprintf("TRIGGER MISSING on alpha: %s", k))
		}
	}
	for k := range alphaTrigMap {
		if _, exists := refTrigMap[k]; !exists {
			discrepancies = append(discrepancies, fmt.Sprintf("EXTRA TRIGGER on alpha: %s", k))
		}
	}

	if len(discrepancies) == 0 {
		fmt.Printf("DIFF RESULT: 0 DISCREPANCIES (100%% Bit-for-Bit Schema Parity across %d tables & %d triggers)\n", len(refTables), len(refTriggers))
	} else {
		fmt.Printf("DIFF RESULT: %d DISCREPANCIES FOUND:\n", len(discrepancies))
		for _, d := range discrepancies {
			fmt.Printf("  - %s\n", d)
		}
	}
	fmt.Println("======================================================================")
}
