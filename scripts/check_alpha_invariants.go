package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var masterActiveRules, masterSnapshots, masterSoftDeleted, testRulesCount, totalRules int
	err = db.QueryRowContext(ctx, `
		SELECT 
			(SELECT count(*) FROM compliance.compliance_rule WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid AND valid_to IS NULL) AS master_active_rules,
			(SELECT count(*) FROM compliance.compliance_rule_version WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid) AS master_snapshots,
			(SELECT count(*) FROM compliance.compliance_rule WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid AND valid_to IS NOT NULL) AS master_soft_deleted,
			(SELECT count(*) FROM compliance.compliance_rule WHERE rule_code LIKE 'TEST%' OR rule_code LIKE 'E2E%' OR name LIKE '%Test%' OR tenant_id NOT IN ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, '00000000-0000-4000-a000-000000000002'::uuid)) AS test_rules_count,
			(SELECT count(*) FROM compliance.compliance_rule) AS total_rules
	`).Scan(&masterActiveRules, &masterSnapshots, &masterSoftDeleted, &testRulesCount, &totalRules)
	if err != nil {
		log.Fatalf("Failed to query invariants: %v", err)
	}

	fmt.Println("======================================================================")
	fmt.Printf("POST-SUITE LIVE INVARIANTS ASSERTION (POST-FULL-SUITE RUN):\n")
	fmt.Printf("  master_active_rules: %d (expected 50)\n", masterActiveRules)
	fmt.Printf("  master_snapshots:    %d (expected 50)\n", masterSnapshots)
	fmt.Printf("  master_soft_deleted: %d (expected 0)\n", masterSoftDeleted)
	fmt.Printf("  test_rules_count:    %d (expected 0)\n", testRulesCount)
	fmt.Printf("  total_rules:         %d (expected 50)\n", totalRules)
	fmt.Println("======================================================================")

	if masterActiveRules != 50 || masterSnapshots != 50 || masterSoftDeleted != 0 || testRulesCount != 0 {
		log.Fatalf("INVARIANT CHECK FAILED!")
	}
	fmt.Println("STATUS: ZERO TEST POLLUTION CONFIRMED ON ALPHA.")
}
