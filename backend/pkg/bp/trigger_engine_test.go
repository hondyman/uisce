package bp

import (
	"context"
	"io"
	"log"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

type mockWorkflowInitiator struct{}

func (m *mockWorkflowInitiator) StartBPWorkflow(ctx context.Context, bpID string, data map[string]interface{}) (string, error) {
	return "workflow-id-123", nil
}

func TestNewTriggerEngine(t *testing.T) {
	dbSQL, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	sqlxDB := sqlx.NewDb(dbSQL, "sqlmock")
	defer sqlxDB.Close()

	initiator := &mockWorkflowInitiator{}
	logger := log.New(io.Discard, "", 0)

	te := NewTriggerEngine(sqlxDB, initiator, "tenant-1", logger)

	if te == nil {
		t.Fatal("expected non-nil TriggerEngine")
	}
	if te.tenantID != "tenant-1" {
		t.Fatalf("expected tenant-1, got %s", te.tenantID)
	}
}
