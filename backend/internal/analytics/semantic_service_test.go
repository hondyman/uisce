package analytics

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/hondyman/semlayer/backend/models"
)

// TestPersistIgnoreLocal runs against local Postgres (alpha) and verifies PersistIgnore
// Note: this test requires local Postgres reachable at postgres://postgres:postgres@localhost:5432/alpha
func TestPersistIgnoreLocal(t *testing.T) {
	dsn := os.Getenv("POSTGRES_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable"
	}

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	svc := NewSemanticService(db)
	ctx := context.Background()

	tenantID := "00000000-0000-0000-0000-000000000000"
	tenantDS := "11111111-1111-1111-1111-111111111111"
	columnNodeID := fmt.Sprintf("00000000-0000-0000-0000-%012d", time.Now().UnixNano()%999999999999)
	term := fmt.Sprintf("test-ignore-%d", time.Now().Unix())

	if err := svc.PersistIgnore(ctx, tenantID, tenantDS, columnNodeID, term); err != nil {
		t.Fatalf("PersistIgnore returned error: %v", err)
	}

	// Cleanup the inserted row
	if _, err := db.ExecContext(ctx, `DELETE FROM public.semantic_mapping_ignores WHERE tenant_datasource_id = $1 AND database_column_node_id = $2 AND ignored_term = $3`, tenantDS, columnNodeID, term); err != nil {
		t.Logf("cleanup failed: %v", err)
	}
}

func TestExecuteSemanticQuery_PreAggWrongRegion(t *testing.T) {
	svc := NewSemanticService(nil)

	q := models.SemanticQuery{
		Dimensions: []string{"customer_region"},
		Metrics:    []string{"total_revenue"},
		Limit:      100,
		Region:     "EMEA",
	}

	_, err := svc.ExecuteSemanticQuery(context.Background(), "sales_overview", q)
	if err == nil {
		t.Fatalf("expected an error due to pre-agg region mismatch, got nil")
	}

	expected := "pre-aggregation 'sales_by_region_daily' is not available in region 'EMEA'."
	if err.Error() != expected {
		t.Fatalf("unexpected error: %v", err)
	}
}
