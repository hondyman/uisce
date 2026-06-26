package audit

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"

	_ "github.com/trinodb/trino-go-client/trino"
)

// TestTrinoSnapshotBackfillIntegration verifies the Iceberg table backfill on Trino.
// Behavior:
// - If TRINO_TEST_SNAP_IDS and TRINO_EXPECTED_REGIONS are set, it checks those snapshot ids have the expected regions.
// - Otherwise it asserts no NULL regions exist in semantic_snapshots.
// Environment variables:
// TRINO_HOST, TRINO_PORT, TRINO_USER, TRINO_PASSWORD, TRINO_CATALOG, TRINO_SCHEMA
// Optional: TRINO_TEST_SNAP_IDS (comma list), TRINO_EXPECTED_REGIONS (comma list matching ids)
func TestTrinoSnapshotBackfillIntegration(t *testing.T) {
	if os.Getenv("TRINO_HOST") == "" {
		t.Skip("TRINO_HOST not set; skipping Trino integration test")
	}

	host := os.Getenv("TRINO_HOST")
	port := os.Getenv("TRINO_PORT")
	if port == "" {
		port = "8080"
	}
	user := os.Getenv("TRINO_USER")
	password := os.Getenv("TRINO_PASSWORD")
	catalog := os.Getenv("TRINO_CATALOG")
	if catalog == "" {
		catalog = "iceberg"
	}
	schema := os.Getenv("TRINO_SCHEMA")
	if schema == "" {
		schema = "audit"
	}

	dsn := fmt.Sprintf("http://%s:%s@%s:%s?catalog=%s&schema=%s", user, password, host, port, catalog, schema)
	db, err := sql.Open("trino", dsn)
	if err != nil {
		t.Fatalf("failed to open trino connection: %v", err)
	}
	defer db.Close()

	// If specific snapshot IDs are provided, validate expected regions
	snapIDs := os.Getenv("TRINO_TEST_SNAP_IDS")
	expected := os.Getenv("TRINO_EXPECTED_REGIONS")

	if snapIDs != "" && expected != "" {
		ids := strings.Split(snapIDs, ",")
		exps := strings.Split(expected, ",")
		if len(ids) != len(exps) {
			t.Fatalf("TRINO_TEST_SNAP_IDS and TRINO_EXPECTED_REGIONS must have same length")
		}

		for i, id := range ids {
			var region sql.NullString
			query := fmt.Sprintf("SELECT region FROM semantic_snapshots WHERE snapshot_id = '%s'", strings.TrimSpace(id))
			if err := db.QueryRow(query).Scan(&region); err != nil {
				t.Fatalf("failed to query snapshot %s: %v", id, err)
			}
			if !region.Valid || region.String == "" {
				t.Fatalf("expected region for snapshot %s to be populated, got: %v", id, region)
			}
			if region.String != strings.TrimSpace(exps[i]) {
				t.Fatalf("expected region '%s' for snapshot %s, got '%s'", strings.TrimSpace(exps[i]), id, region.String)
			}
		}

		// Idempotency: run a null count twice and ensure same
		var c1, c2 int64
		if err := db.QueryRow("SELECT count(*) FROM semantic_snapshots WHERE region IS NULL").Scan(&c1); err != nil {
			t.Fatalf("failed null count query: %v", err)
		}
		if err := db.QueryRow("SELECT count(*) FROM semantic_snapshots WHERE region IS NULL").Scan(&c2); err != nil {
			t.Fatalf("failed null count query 2: %v", err)
		}
		if c1 != c2 {
			t.Fatalf("idempotency violation: null counts differ %d != %d", c1, c2)
		}
		return
	}

	// Default: assert zero NULL regions
	var count int64
	if err := db.QueryRow("SELECT count(*) FROM semantic_snapshots WHERE region IS NULL").Scan(&count); err != nil {
		t.Fatalf("failed to query semantic_snapshots via trino: %v", err)
	}
	if count != 0 {
		t.Fatalf("unexpected %d snapshot rows with NULL region in Trino Iceberg table", count)
	}
}
