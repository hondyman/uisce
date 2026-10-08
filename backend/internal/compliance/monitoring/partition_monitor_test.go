package monitoring

import (
	"context"
	"testing"
	"time"

	"github.com/hondyman/uisce/backend/internal/compliance/testutil"
	_ "github.com/lib/pq"
)

func TestDefaultPartitionMonitor_ZeroIncidents(t *testing.T) {
	db := testutil.GetEphemeralTestDB(t)
	if db == nil {
		return
	}

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
