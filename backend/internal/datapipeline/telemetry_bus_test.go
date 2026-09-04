package datapipeline

import (
	"testing"
	"time"

	"github.com/google/uuid"
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
