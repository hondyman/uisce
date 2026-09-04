package datapipeline

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func TestTelemetryBus_ListenAndCleanup(t *testing.T) {
	bus := NewTelemetryBus(nil)

	ch, cleanup := bus.Listen("run-123")
	if ch == nil {
		t.Fatal("Listen returned nil channel")
	}
	if cleanup == nil {
		t.Fatal("Listen returned nil cleanup")
	}

	cleanup()

	select {
	case _, ok := <-ch:
		if ok {
			t.Error("channel should be closed after cleanup")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("channel not closed within timeout")
	}
}

func TestTelemetryBus_NotifyStep_NopWhenNoDB(t *testing.T) {
	bus := NewTelemetryBus(nil)

	run := PipelineExecutionRun{
		RunID: uuid.New(),
	}
	err := bus.NotifyStep(nil, run, "node-a")
	if err != nil {
		t.Errorf("NotifyStep with nil DB should be no-op, got: %v", err)
	}
}

func TestTelemetryBus_Stop_NopWhenNoListener(t *testing.T) {
	bus := NewTelemetryBus(nil)
	err := bus.Stop()
	if err != nil {
		t.Errorf("Stop with nil listener should be no-op, got: %v", err)
	}
}

func TestTelemetryBus_Roundtrip_NotifyStepDelivered(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := sqlx.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("DB ping failed: %v", err)
	}

	bus := NewTelemetryBus(db)
	if err := bus.Start(ctx, dbURL); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer bus.Stop()

	runID := uuid.New()
	ch, cleanup := bus.Listen(runID.String())
	defer cleanup()

	run := PipelineExecutionRun{
		RunID: runID,
		StepTelemetry: map[string]StepMetrics{
			"node-a": {
				NodeID:      "node-a",
				NodeLabel:   "Source",
				NodeType:    "source",
				Status:      "completed",
				RecordsIn:    0,
				RecordsOut:  2,
				RecordsError: 0,
				Duration:    100 * time.Millisecond,
				RowsPerSec:  20.0,
			},
		},
	}

	if err := bus.NotifyStep(ctx, run, "node-a"); err != nil {
		t.Fatalf("NotifyStep failed: %v", err)
	}

	select {
	case n := <-ch:
		if n == nil {
			t.Fatal("received nil notification")
		}
		if n.RunID != runID.String() {
			t.Errorf("RunID: expected %s, got %s", runID.String(), n.RunID)
		}
		if n.Run != nil {
			t.Errorf("step notification should not include Run; got Run=%v", n.Run)
		}
		if n.Step == nil {
			t.Fatal("Step should not be nil for step notification")
		}
		if n.Step["node_id"] != "node-a" {
			t.Errorf("node_id: expected node-a, got %v", n.Step["node_id"])
		}
		if v, ok := n.Step["records_out"].(float64); !ok || int64(v) != 2 {
			t.Errorf("records_out: expected 2, got %v", n.Step["records_out"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("NotifyStep was not received within 2s")
	}
}
