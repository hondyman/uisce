//go:build integration

// Run with: go test -tags integration ./internal/analytics -run TestExportRulesIntegration
// Requires UISCE_TEST_DSN pointing at a database with the uisce schema and
// at least one validation rule row.

package analytics

import (
	"context"
	"os"
	"testing"

	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func TestExportRulesIntegration(t *testing.T) {
	dsn := os.Getenv("UISCE_TEST_DSN")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("UISCE_TEST_DSN / DATABASE_URL not set")
	}
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer db.Close()

	porter := NewValidationRulePorter(db)
	ctx := context.Background()

	// Resolve a real tenant for the smoke export.
	var tenantID string
	if err := db.GetContext(ctx, &tenantID, `SELECT id::text FROM tenants LIMIT 1`); err != nil {
		t.Skipf("no tenant rows: %v", err)
	}

	bundle, err := porter.ExportRules(ctx, tenantID, "", "", "")
	if err != nil {
		t.Fatalf("ExportRules: %v", err)
	}
	if bundle.Checksum == "" {
		t.Fatal("checksum not computed")
	}

	// Re-export must produce an identical checksum (determinism).
	again, err := porter.ExportRules(ctx, tenantID, "", "", "")
	if err != nil {
		t.Fatalf("ExportRules second pass: %v", err)
	}
	if bundle.Checksum != again.Checksum {
		t.Fatalf("export not deterministic:\n %s\n %s", bundle.Checksum, again.Checksum)
	}
}

func TestImportRoundTripChecksumStability(t *testing.T) {
	dsn := os.Getenv("UISCE_TEST_DSN")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("UISCE_TEST_DSN / DATABASE_URL not set")
	}
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	var sourceTenant, targetTenant string
	var tenants []string
	if err := db.SelectContext(ctx, &tenants, `SELECT id::text FROM tenants ORDER BY created_at LIMIT 2`); err != nil || len(tenants) < 2 {
		t.Skipf("need at least 2 tenants for roundtrip test: %v (found %d)", err, len(tenants))
	}
	sourceTenant = tenants[0]
	targetTenant = tenants[1]

	porter := NewValidationRulePorter(db)

	bundle, err := porter.ExportRules(ctx, sourceTenant, "", "", "")
	if err != nil {
		t.Fatalf("export source: %v", err)
	}
	if len(bundle.Rules) == 0 {
		t.Skip("source tenant has no rules")
	}

	// Rewrite tenant scope for the target and re-checksum.
	bundle.TenantScope = targetTenant
	if err := bundle.ComputeChecksum(); err != nil {
		t.Fatal(err)
	}

	req := models.RuleImportRequest{
		Bundle:          *bundle,
		TargetTenantID:  targetTenant,
		OverwritePolicy: models.ImportOverwriteOverwrite,
	}
	report, err := porter.ImportRules(ctx, req, false, "test-roundtrip-1")
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if len(report.Errors) > 0 {
		t.Fatalf("import errors: %+v", report.Errors)
	}

	reexported, err := porter.ExportRules(ctx, targetTenant, "", "", "")
	if err != nil {
		t.Fatalf("re-export: %v", err)
	}
	if reexported.Checksum != bundle.Checksum {
		t.Fatalf("round-trip checksum drift:\n imported =%s\n reexported=%s", bundle.Checksum, reexported.Checksum)
	}

	// Idempotent replay: same key must return the cached report, no new mutations.
	replay, err := porter.ImportRules(ctx, req, false, "test-roundtrip-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(replay.Created) != 0 || len(replay.Updated) != 0 {
		t.Fatalf("idempotent replay re-executed mutations: created=%v updated=%v", replay.Created, replay.Updated)
	}
}

