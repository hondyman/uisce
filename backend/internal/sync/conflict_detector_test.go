package sync

import (
	"testing"
)

// TestResolveConflict is currently a placeholder.
// The GoogleSyncRepo has been migrated from HasuraClient to direct sqlx.DB.
// These tests should be rewritten with a test database or pgx mock.
func TestResolveConflict(t *testing.T) {
	t.Skip("Skipping: GoogleSyncRepo migrated from Hasura to direct SQL — test needs sqlx mock setup")
}
