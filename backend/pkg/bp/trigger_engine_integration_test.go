package bp

import (
	"context"
	"io"
	"log"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

type mockInitiator struct{}

func (m *mockInitiator) StartBPWorkflow(ctx context.Context, bpID string, data map[string]interface{}) (string, error) {
	return "workflow-id", nil
}

// unreachableDSN points at a closed local port, so the LISTEN connection can
// never be established. This is the case that used to hang Stop forever:
// pq.Listener.Listen blocks until connected.
const unreachableDSN = "postgres://u:p@127.0.0.1:1/none?sslmode=disable"

func newTestEngine(t *testing.T) *TriggerEngine {
	t.Helper()
	dbSQL, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	sqlxDB := sqlx.NewDb(dbSQL, "sqlmock")
	t.Cleanup(func() { sqlxDB.Close() })
	return NewTriggerEngine(sqlxDB, &mockInitiator{}, "tenant-1", log.New(io.Discard, "", 0))
}

// returnsWithin fails the test if fn does not return in d.
func returnsWithin(t *testing.T, d time.Duration, what string, fn func()) {
	t.Helper()
	done := make(chan struct{})
	go func() { fn(); close(done) }()
	select {
	case <-done:
	case <-time.After(d):
		t.Fatalf("%s did not return within %s (hang)", what, d)
	}
}

func TestTriggerEngineStartStop(t *testing.T) {
	te := newTestEngine(t)
	if err := te.Start(context.Background(), unreachableDSN); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	returnsWithin(t, 5*time.Second, "Stop with an unreachable database", te.Stop)
}

func TestTriggerEngineStopIsIdempotent(t *testing.T) {
	te := newTestEngine(t)
	if err := te.Start(context.Background(), unreachableDSN); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	returnsWithin(t, 5*time.Second, "first Stop", te.Stop)
	returnsWithin(t, 5*time.Second, "second Stop", te.Stop)
}

func TestTriggerEngineContextCancelUnblocksListener(t *testing.T) {
	te := newTestEngine(t)
	ctx, cancel := context.WithCancel(context.Background())
	if err := te.Start(ctx, unreachableDSN); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	cancel()
	// The listener goroutine must exit on cancel alone, so Stop's wait is instant.
	returnsWithin(t, 5*time.Second, "Stop after context cancel", te.Stop)
}

func TestTriggerEngineStartRequiresDSN(t *testing.T) {
	te := newTestEngine(t)
	if err := te.Start(context.Background(), ""); err == nil {
		t.Fatal("expected an error when pgURL is empty")
	}
}
