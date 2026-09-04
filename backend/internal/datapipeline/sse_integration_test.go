package datapipeline

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func TestSSEHandler_BusListenDeliversEvents(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
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
		t.Fatalf("bus.Start failed: %v", err)
	}
	defer bus.Stop()

	engine := NewPipelineEngine(db, nil, nil)
	engine.AttachTelemetryBus(bus)

	runID := uuid.New()

	ch, cleanup := bus.Listen(runID.String())
	defer cleanup()

	run := PipelineExecutionRun{
		RunID: runID,
		StepTelemetry: map[string]StepMetrics{
			"node-a": {
				NodeID:       "node-a",
				NodeLabel:    "Source",
				NodeType:     "source",
				Status:       "completed",
				RecordsIn:    0,
				RecordsOut:   2,
				RecordsError: 0,
				Duration:     10 * time.Millisecond,
				RowsPerSec:   200.0,
			},
		},
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		bus.NotifyStep(ctx, run, "node-a")
	}()

	select {
	case n := <-ch:
		if n == nil {
			t.Fatal("received nil notification")
		}
		if n.RunID != runID.String() {
			t.Errorf("RunID: expected %s, got %s", runID.String(), n.RunID)
		}
		if n.Run != nil {
			t.Errorf("step notification should not include Run")
		}
		if n.Step == nil {
			t.Error("Step should not be nil for step notification")
		}
		if nodeID, ok := n.Step["node_id"].(string); !ok || nodeID != "node-a" {
			t.Errorf("node_id: expected node-a, got %v", n.Step["node_id"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("NotifyStep was not received within 2s")
	}
}

func TestSSEHandler_503WhenBusNotConfigured(t *testing.T) {
	db := &sqlx.DB{}
	engine := NewPipelineEngine(nil, nil, nil)
	h := &DataPipelineHandler{db: db, engine: engine, bus: nil}

	req := httptest.NewRequest("GET", "/api/v1/data-pipelines/runs/test-run/telemetry", nil)
	w := httptest.NewRecorder()

	h.StreamTelemetrySSE(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestTelemetryBus_NotifyCompletionIncludesFullRun(t *testing.T) {
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
		t.Fatalf("bus.Start failed: %v", err)
	}
	defer bus.Stop()

	runID := uuid.New()
	ch, cleanup := bus.Listen(runID.String())
	defer cleanup()

	run := PipelineExecutionRun{
		RunID:          runID,
		Status:         "completed",
		TotalRecordsIn:  10,
		TotalRecordsOut: 10,
	}

	if err := bus.NotifyCompletion(ctx, run); err != nil {
		t.Fatalf("NotifyCompletion failed: %v", err)
	}

	select {
	case n := <-ch:
		if n == nil {
			t.Fatal("received nil notification")
		}
		if n.RunID != runID.String() {
			t.Errorf("RunID: expected %s, got %s", runID.String(), n.RunID)
		}
		if n.Run == nil {
			t.Fatal("completion notification should include Run")
		}
		if n.Run.Status != "completed" {
			t.Errorf("status: expected completed, got %s", n.Run.Status)
		}
		if n.Step != nil {
			t.Errorf("completion notification should not include Step")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("NotifyCompletion was not received within 2s")
	}
}

func TestTelemetryBus_NotifyStep_JsonFloatsNotInt64(t *testing.T) {
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
		t.Fatalf("bus.Start failed: %v", err)
	}
	defer bus.Stop()

	runID := uuid.New()
	ch, cleanup := bus.Listen(runID.String())
	defer cleanup()

	run := PipelineExecutionRun{
		RunID: runID,
		StepTelemetry: map[string]StepMetrics{
			"node-a": {
				NodeID:       "node-a",
				NodeLabel:    "Source",
				NodeType:     "source",
				Status:       "completed",
				RecordsIn:    5,
				RecordsOut:   5,
				RecordsError: 0,
				Duration:     10 * time.Millisecond,
				RowsPerSec:   500.0,
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
		if v, ok := n.Step["records_out"].(float64); !ok || int64(v) != 5 {
			t.Errorf("records_out: expected 5 (float64), got %v", n.Step["records_out"])
		}
		if v, ok := n.Step["rows_per_sec"].(float64); !ok || v != 500.0 {
			t.Errorf("rows_per_sec: expected 500.0, got %v", n.Step["rows_per_sec"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("NotifyStep was not received within 2s")
	}
}
