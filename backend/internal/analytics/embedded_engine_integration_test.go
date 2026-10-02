//go:build integration

package analytics

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// TestEmbeddedEngineNotifyInvalidation verifies that the LISTEN/NOTIFY invalidation
// channel detects changes inside the catalog without waiting for the poll interval.
func TestEmbeddedEngineNotifyInvalidation(t *testing.T) {
	dsn := os.Getenv("UISCE_TEST_DSN")
	if dsn == "" {
		t.Skip("UISCE_TEST_DSN not set")
	}
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := NewValidationRuleService(db)
	e := NewEmbeddedEngine(db, "00000000-0000-0000-0000-000000000001", svc, WithDSN(dsn), WithPollInterval(time.Hour))
	e.Start(context.Background())
	defer e.Close()

	v0, err := e.SnapshotVersion(context.Background(), "order", "", "")
	if err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		v1, err := e.SnapshotVersion(context.Background(), "order", "", "")
		if err == nil && v1 != v0 {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}
