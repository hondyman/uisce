package monitoring

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

func TestDefaultPartitionMonitor_ZeroIncidents(t *testing.T) {
	dsn := os.Getenv("ALPHA_DSN")
	if dsn == "" {
		// Local mock or skip
		t.Skip("ALPHA_DSN not set, skipping live database test")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("Failed to connect to alpha: %v", err)
	}
	defer db.Close()

	alertCalled := false
	monitor := NewDefaultPartitionMonitor(db, func(count int64) {
		alertCalled = true
	}, 1*time.Minute)

	count, err := monitor.CheckIncident(context.Background())
	if err != nil {
		t.Fatalf("CheckIncident error: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 incidents in default partition, got %d", count)
	}
	if alertCalled {
		t.Errorf("Expected no alert called when count is 0")
	}
}
