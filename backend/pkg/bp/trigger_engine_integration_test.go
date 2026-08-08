package bp

import (
	"context"
	"io"
	"log"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

type mockInitiator struct{}

func (m *mockInitiator) StartBPWorkflow(ctx context.Context, bpID string, data map[string]interface{}) (string, error) {
	return "workflow-id", nil
}

func TestTriggerEngineStartStop(t *testing.T) {
	dbSQL, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	sqlxDB := sqlx.NewDb(dbSQL, "sqlmock")
	defer sqlxDB.Close()

	initiator := &mockInitiator{}
	logger := log.New(io.Discard, "", 0)

	te := NewTriggerEngine(sqlxDB, initiator, "tenant-1", logger)

	ctx := context.Background()
	err = te.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	te.Stop()
}
