package library

import (
	"context"
	"fmt"
	"testing"

	"github.com/hondyman/uisce/backend/internal/compliance/testutil"
)

func TestSchemaParity_ComplianceTablesAndConstraints(t *testing.T) {
	db := testutil.GetEphemeralTestDB(t)
	ctx := context.Background()

	// 1. Verify all expected compliance tables exist in ephemeral database
	expectedTables := []string{
		"compliance_rule",
		"compliance_rule_version",
		"compliance_ruleset_membership",
		"tenant_rule_activation",
		"compliance_evaluation_event",
		"compliance_archive_manifest",
		"compliance_watermark_checkpoint",
		"compliance_orphan_object",
		"governance_audit_event",
		"regulatory_change_case",
		"regulatory_draft_rule",
		"regulatory_case_event",
		"compliance_notification",
		"compliance_surveillance_finding",
		"compliance_surveillance_event",
		"compliance_portfolio_snapshot",
		"compliance_finding",
	}

	for _, table := range expectedTables {
		var exists bool
		err := db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.tables 
				WHERE table_schema = 'compliance' AND table_name = $1
			)
		`, table).Scan(&exists)
		if err != nil {
			t.Fatalf("failed to query table %s: %v", table, err)
		}
		if !exists {
			t.Errorf("missing expected compliance table: compliance.%s", table)
		}
	}

	// 2. Verify all append-only and version-snapshot trigger guards exist
	expectedTriggers := []struct {
		table   string
		trigger string
	}{
		{"compliance_rule_version", "trg_prevent_rule_version_mutation"},
		{"compliance_rule_version", "trg_prevent_rule_version_truncate"},
		{"compliance_rule", "trg_enforce_rule_version_snapshot"},
		{"regulatory_change_case", "trg_case_transition"},
		{"regulatory_case_event", "trg_prevent_case_event_mutation"},
		{"regulatory_case_event", "trg_prevent_case_event_truncate"},
		{"regulatory_draft_rule", "trg_guard_regulatory_draft_mutation"},
		{"compliance_surveillance_finding", "trg_validate_surveillance_finding_transition"},
		{"compliance_surveillance_event", "trg_guard_surveillance_event_append_only"},
		{"compliance_finding", "trg_validate_compliance_finding_transition"},
	}

	for _, trg := range expectedTriggers {
		var exists bool
		err := db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_trigger t
				JOIN pg_class c ON t.tgrelid = c.oid
				JOIN pg_namespace n ON c.relnamespace = n.oid
				WHERE n.nspname = 'compliance' AND c.relname = $1 AND t.tgname = $2
			)
		`, trg.table, trg.trigger).Scan(&exists)
		if err != nil {
			t.Fatalf("failed to query trigger %s on %s: %v", trg.trigger, trg.table, err)
		}
		if !exists {
			t.Errorf("missing expected trigger: %s on compliance.%s", trg.trigger, trg.table)
		}
	}

	// 3. Verify Master Tenant Gold Copy Invariant
	var masterCount, snapshotCount int
	err := db.QueryRowContext(ctx, `
		SELECT 
			(SELECT count(*) FROM compliance.compliance_rule WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid AND valid_to IS NULL),
			(SELECT count(*) FROM compliance.compliance_rule_version WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid)
	`).Scan(&masterCount, &snapshotCount)
	if err != nil {
		t.Fatalf("failed to query master tenant counts: %v", err)
	}

	if masterCount != 86 {
		t.Errorf("expected exactly 86 gold-copy rules, got %d", masterCount)
	}
	if snapshotCount != 86 {
		t.Errorf("expected exactly 86 v1 snapshots for gold-copy rules, got %d", snapshotCount)
	}

	// 4. Verify master schema tables
	var relTableExists bool
	err = db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_schema = 'master' AND table_name = 'entity_relationship_snapshot'
		)
	`).Scan(&relTableExists)
	if err != nil || !relTableExists {
		t.Errorf("expected master.entity_relationship_snapshot table to exist")
	}

	fmt.Printf("[Schema Diff Test] Verified 86 master rules, 86 v1 snapshots, 17 compliance tables, master hierarchy table, and all triggers.\n")
}
