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

	var (
		masterActiveRules, masterSnapshots, masterSoftDeleted, totalRules int
		masterLibraryActive, masterLibraryProvisional                     int
		foreignRules, foreignVersions, foreignActivations                 int
		foreignEvaluations, foreignAuditEvents, foreignFindings           int
		foreignSurvEvents, foreignPortSnapshots, foreignPostFindings      int
		foreignOverrides, foreignDraftRules, foreignCases                 int
		foreignCaseEvents, foreignNotifications                           int
	)

	err = db.QueryRowContext(ctx, `
		SELECT 
			-- 1. Master gold-copy rule library invariants
			(SELECT count(*) FROM compliance.compliance_rule WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid AND valid_to IS NULL),
			(SELECT count(*) FROM compliance.compliance_rule_version WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid),
			(SELECT count(*) FROM compliance.compliance_rule WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid AND valid_to IS NOT NULL),
			(SELECT count(*) FROM compliance.compliance_rule),
			(SELECT count(*) FROM compliance.compliance_rule WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid AND library_status = 'ACTIVE' AND valid_to IS NULL),
			(SELECT count(*) FROM compliance.compliance_rule WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid AND library_status = 'PROVISIONAL' AND valid_to IS NULL),

			-- 2. Foreign / Test row probes across all tables (must be 0 outside gold master and demo tenant)
			(SELECT count(*) FROM compliance.compliance_rule WHERE tenant_id NOT IN ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, '00000000-0000-4000-a000-000000000002'::uuid)),
			(SELECT count(*) FROM compliance.compliance_rule_version WHERE tenant_id NOT IN ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, '00000000-0000-4000-a000-000000000002'::uuid)),
			(SELECT count(*) FROM compliance.tenant_rule_activation WHERE tenant_id NOT IN ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, '00000000-0000-4000-a000-000000000002'::uuid)),
			(SELECT count(*) FROM compliance.compliance_evaluation_event WHERE tenant_id NOT IN ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, '00000000-0000-4000-a000-000000000002'::uuid)),
			(SELECT count(*) FROM compliance.governance_audit_event WHERE tenant_id NOT IN ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, '00000000-0000-4000-a000-000000000002'::uuid)),
			(SELECT count(*) FROM compliance.compliance_surveillance_finding WHERE tenant_id NOT IN ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, '00000000-0000-4000-a000-000000000002'::uuid)),
			(SELECT count(*) FROM compliance.compliance_surveillance_event WHERE tenant_id NOT IN ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, '00000000-0000-4000-a000-000000000002'::uuid)),
			(SELECT count(*) FROM compliance.compliance_portfolio_snapshot WHERE tenant_id NOT IN ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, '00000000-0000-4000-a000-000000000002'::uuid)),
			(SELECT count(*) FROM compliance.compliance_finding WHERE tenant_id NOT IN ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, '00000000-0000-4000-a000-000000000002'::uuid)),
			(SELECT count(*) FROM master.tenant_classification_override WHERE tenant_id NOT IN ('99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, '00000000-0000-4000-a000-000000000002'::uuid)),
			(SELECT count(*) FROM compliance.regulatory_draft_rule),
			(SELECT count(*) FROM compliance.regulatory_change_case),
			(SELECT count(*) FROM compliance.regulatory_case_event),
			(SELECT count(*) FROM compliance.compliance_notification)
	`).Scan(
		&masterActiveRules, &masterSnapshots, &masterSoftDeleted, &totalRules,
		&masterLibraryActive, &masterLibraryProvisional,
		&foreignRules, &foreignVersions, &foreignActivations,
		&foreignEvaluations, &foreignAuditEvents, &foreignFindings,
		&foreignSurvEvents, &foreignPortSnapshots, &foreignPostFindings,
		&foreignOverrides, &foreignDraftRules, &foreignCases,
		&foreignCaseEvents, &foreignNotifications,
	)
	if err != nil {
		log.Fatalf("Failed to query invariants: %v", err)
	}

	fmt.Println("======================================================================")
	fmt.Printf("POST-SUITE LIVE INVARIANTS & STRUCTURAL POLLUTION PROBE (LIVE ALPHA):\n")
	fmt.Printf("  master_active_rules:         %d (expected 86)\n", masterActiveRules)
	fmt.Printf("  master_library_active:       %d (expected 86)\n", masterLibraryActive)
	fmt.Printf("  master_library_provisional:  %d (expected 0)\n", masterLibraryProvisional)
	fmt.Printf("  master_snapshots:            %d (expected 86)\n", masterSnapshots)
	fmt.Printf("  master_soft_deleted:         %d (expected 0)\n", masterSoftDeleted)
	fmt.Printf("  total_rules:                 %d (expected 86)\n", totalRules)
	fmt.Println("----------------------------------------------------------------------")
	fmt.Printf("NON-MASTER / NON-DEMO ROW COUNTS (STRICT ZERO REQUIRED):\n")
	fmt.Printf("  compliance_rule:                 %d (expected 0)\n", foreignRules)
	fmt.Printf("  compliance_rule_version:         %d (expected 0)\n", foreignVersions)
	fmt.Printf("  tenant_rule_activation:          %d (expected 0)\n", foreignActivations)
	fmt.Printf("  compliance_evaluation_event:     %d (expected 0)\n", foreignEvaluations)
	fmt.Printf("  governance_audit_event:          %d (expected 0)\n", foreignAuditEvents)
	fmt.Printf("  compliance_surveillance_finding: %d (expected 0)\n", foreignFindings)
	fmt.Printf("  compliance_surveillance_event:   %d (expected 0)\n", foreignSurvEvents)
	fmt.Printf("  compliance_portfolio_snapshot:   %d (expected 0)\n", foreignPortSnapshots)
	fmt.Printf("  compliance_finding:              %d (expected 0)\n", foreignPostFindings)
	fmt.Printf("  tenant_classification_override:  %d (expected 0)\n", foreignOverrides)
	fmt.Printf("  regulatory_draft_rule:           %d (expected 0)\n", foreignDraftRules)
	fmt.Printf("  regulatory_change_case:          %d (expected 0)\n", foreignCases)
	fmt.Printf("  regulatory_case_event:           %d (expected 0)\n", foreignCaseEvents)
	fmt.Printf("  compliance_notification:         %d (expected 0)\n", foreignNotifications)
	fmt.Println("======================================================================")

	if masterActiveRules != 86 || masterLibraryActive != 86 || masterLibraryProvisional != 0 ||
		masterSnapshots != 86 || masterSoftDeleted != 0 || totalRules != 86 ||
		foreignRules != 0 || foreignVersions != 0 || foreignActivations != 0 ||
		foreignEvaluations != 0 || foreignAuditEvents != 0 || foreignFindings != 0 ||
		foreignSurvEvents != 0 || foreignPortSnapshots != 0 || foreignPostFindings != 0 ||
		foreignOverrides != 0 || foreignDraftRules != 0 || foreignCases != 0 ||
		foreignCaseEvents != 0 || foreignNotifications != 0 {
		log.Fatalf("INVARIANT / STRUCTURAL POLLUTION CHECK FAILED!")
	}
	fmt.Println("STATUS: ZERO TEST POLLUTION CONFIRMED ACROSS ALL COMPLIANCE TABLES.")
}
