package audit

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func TestBackfillSnapshotsIntegration(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") == "" {
		t.Skip("integration test skipped; set INTEGRATION_TEST=1 to run")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Fatal("DATABASE_URL must be set for integration test")
	}

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		t.Fatalf("failed to connect to db: %v", err)
	}
	defer db.Close()

	// Prepare the environment
	_, err = db.Exec(`CREATE SCHEMA IF NOT EXISTS iceberg.audit;`)
	if err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS iceberg.audit.semantic_snapshots (snapshot_id VARCHAR PRIMARY KEY, tenant_id VARCHAR, definition VARCHAR, region VARCHAR);`)
	if err != nil {
		t.Fatalf("failed to ensure snapshot table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS public.catalog_node (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), properties JSONB);`)
	if err != nil {
		t.Fatalf("failed to ensure catalog_node table: %v", err)
	}

	// Clean up any existing test rows
	_, _ = db.Exec(`DELETE FROM iceberg.audit.semantic_snapshots WHERE snapshot_id = 'test-snap-01'`)
	_, _ = db.Exec(`DELETE FROM public.catalog_node WHERE properties->>'snapshot_id' = 'test-snap-01'`)

	// Insert a snapshot with NULL region
	_, err = db.Exec(`INSERT INTO iceberg.audit.semantic_snapshots (snapshot_id, tenant_id, definition, region) VALUES ('test-snap-01', 't1', '{}', NULL)`)
	if err != nil {
		t.Fatalf("failed to insert snapshot row: %v", err)
	}

	// Insert a catalog node with region in properties
	_, err = db.Exec(`INSERT INTO public.catalog_node (properties) VALUES ($1)`, `{"snapshot_id":"test-snap-01","region":"eu-west"}`)
	if err != nil {
		t.Fatalf("failed to insert catalog node: %v", err)
	}

	// Run backfill SQL
	_, err = db.Exec(`UPDATE iceberg.audit.semantic_snapshots ss SET region = coalesce((SELECT n.properties->>'region' FROM public.catalog_node n WHERE n.properties->>'snapshot_id' = ss.snapshot_id LIMIT 1), ss.region) WHERE ss.snapshot_id = 'test-snap-01'`)
	if err != nil {
		t.Fatalf("backfill SQL failed: %v", err)
	}

	// Verify
	var region sql.NullString
	err = db.Get(&region, `SELECT region FROM iceberg.audit.semantic_snapshots WHERE snapshot_id = 'test-snap-01'`)
	if err != nil {
		t.Fatalf("failed to query snapshot: %v", err)
	}

	if !region.Valid || region.String == "" {
		t.Fatalf("expected region to be populated, got: %v", region)
	}

	if region.String != "eu-west" {
		t.Fatalf("expected region 'eu-west', got '%s'", region.String)
	}

	fmt.Println("Backfill integration test succeeded: region updated to", region.String)
}
