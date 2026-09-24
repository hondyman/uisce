package fix

import (
	"context"
	"database/sql"
	"os"
	"testing"
)

// captureSinkForTest satisfies InboundSink for adapter tests.
type captureSinkForTest struct{}

func (captureSinkForTest) Emit(_ context.Context, _ InboundRecord) error { return nil }

// openAdapterTestDB is a small helper for adapter tests that need
// real Postgres. Skips when FIX_TEST_DATABASE_URL is unset so unit-only
// CI doesn't fail.
func openAdapterTestDB(t *testing.T) *sql.DB {
	t.Helper()
	url := os.Getenv("FIX_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("FIX_TEST_DATABASE_URL not set; skipping adapter test")
	}
	db, err := sql.Open("postgres", url)
	if err != nil {
		t.Fatalf("open test DB: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("ping test DB: %v", err)
	}
	return db
}
