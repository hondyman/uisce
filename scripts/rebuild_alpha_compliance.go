package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	_ "github.com/lib/pq"
)

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

func main() {
	dsn := getAlphaDSN()
	if dsn == "" {
		log.Fatal("ALPHA_DSN not configured and certs not found")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to open db: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Failed to ping db: %v", err)
	}

	log.Println("--- Step 1: Dropping compliance schema cascade on alpha ---")
	_, err = db.ExecContext(ctx, "DROP SCHEMA IF EXISTS compliance CASCADE;")
	if err != nil {
		log.Fatalf("Failed to drop schema: %v", err)
	}

	log.Println("--- Step 2: Applying canonical migrations 001 through 009 ---")
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
			log.Fatalf("Failed to read migration %s: %v", m, err)
		}
		log.Printf("Applying %s...", m)
		_, err = db.ExecContext(ctx, string(content))
		if err != nil {
			log.Fatalf("Failed to execute migration %s: %v", m, err)
		}
	}

	log.Println("--- Step 3: Seeding demo tenant via crd-bakeoff-demo ---")
	cmd := exec.Command("go", "run", "backend/cmd/crd-bakeoff-demo/main.go", "-seed=true", "-demo=true")
	cmd.Dir = "."
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to run demo seed: %v", err)
	}

	log.Println("--- Step 4: Verifying Master & Test Invariants on alpha ---")
	var masterActiveRules, masterSnapshots, masterSoftDeleted, testRulesCount int
	err = db.QueryRowContext(ctx, `
		SELECT 
			(SELECT count(*) FROM compliance.compliance_rule WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid AND valid_to IS NULL),
			(SELECT count(*) FROM compliance.compliance_rule_version WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid),
			(SELECT count(*) FROM compliance.compliance_rule WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid AND valid_to IS NOT NULL),
			(SELECT count(*) FROM compliance.compliance_rule WHERE rule_code LIKE 'TEST%' OR rule_code LIKE 'E2E%' OR name LIKE '%Test%' OR tenant_id NOT IN ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, '00000000-0000-4000-a000-000000000002'::uuid))
	`).Scan(&masterActiveRules, &masterSnapshots, &masterSoftDeleted, &testRulesCount)
	if err != nil {
		log.Fatalf("Failed to query invariants: %v", err)
	}

	fmt.Println("======================================================================")
	fmt.Printf("INVARIANTS ON ALPHA POST-REBUILD:\n")
	fmt.Printf("  master_active_rules: %d (expected 50)\n", masterActiveRules)
	fmt.Printf("  master_snapshots:    %d (expected 50)\n", masterSnapshots)
	fmt.Printf("  master_soft_deleted: %d (expected 0)\n", masterSoftDeleted)
	fmt.Printf("  test_rules_count:    %d (expected 0)\n", testRulesCount)
	fmt.Println("======================================================================")

	if masterActiveRules != 50 || masterSnapshots != 50 || masterSoftDeleted != 0 || testRulesCount != 0 {
		log.Fatalf("INVARIANT CHECK FAILED!")
	}
	log.Println("SUCCESS: Alpha rebuild and invariant check verified cleanly.")
}
