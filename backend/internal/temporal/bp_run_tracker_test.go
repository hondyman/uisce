package temporal

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBPRunTracker_RecordStart(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	tracker := NewBPRunTracker(db, nil)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		rec := WorkflowRunRecord{
			WorkflowID:   "bp-order-flow-123",
			RunID:        "run-abc-999",
			TenantID:     "tenant-alpha",
			ProcessID:    "proc-order-validation",
			ProcessName:  "Order Validation",
			TriggerType:  "event",
			TriggerName:  "cdc.order.created",
			Entity:       "TradeOrder",
			EntityID:     "ORD-7788",
			InputPayload: json.RawMessage(`{"amount": 500000}`),
		}

		mock.ExpectExec(`INSERT INTO public\.bp_workflow_run`).
			WithArgs(
				rec.WorkflowID,
				rec.RunID,
				rec.TenantID,
				rec.ProcessID,
				rec.ProcessName,
				rec.TriggerType,
				rec.TriggerName,
				rec.Entity,
				rec.EntityID,
				[]byte(`{"amount": 500000}`),
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := tracker.RecordStart(ctx, rec)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("missing required fields", func(t *testing.T) {
		err := tracker.RecordStart(ctx, WorkflowRunRecord{
			TenantID: "tenant-1",
		})
		assert.Error(t, err)
	})
}

func TestBPRunTracker_RecordTerminal(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	tracker := NewBPRunTracker(db, nil)
	ctx := context.Background()

	t.Run("completed with output", func(t *testing.T) {
		output := []byte(`{"status": "APPROVED", "score": 98}`)
		outStr := string(output)

		mock.ExpectExec(`UPDATE public\.bp_workflow_run`).
			WithArgs("COMPLETED", &outStr, (*string)(nil), "run-123").
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := tracker.RecordTerminal(ctx, "run-123", "COMPLETED", output, "")
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("failed with error message", func(t *testing.T) {
		errMsg := "Sanctions SDN match detected"

		mock.ExpectExec(`UPDATE public\.bp_workflow_run`).
			WithArgs("FAILED", (*string)(nil), &errMsg, "run-456").
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := tracker.RecordTerminal(ctx, "run-456", "FAILED", nil, errMsg)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectExec(`UPDATE public\.bp_workflow_run`).
			WithArgs("COMPLETED", (*string)(nil), (*string)(nil), "run-nonexistent").
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := tracker.RecordTerminal(ctx, "run-nonexistent", "COMPLETED", nil, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no workflow run found")
	})
}

func TestBPRunTracker_ListRuns(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	tracker := NewBPRunTracker(db, nil)
	ctx := context.Background()

	id := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "workflow_id", "run_id", "tenant_id", "process_id", "process_name",
		"trigger_type", "trigger_name", "entity", "entity_id", "status",
		"input_payload", "output_payload", "error_message",
		"started_at", "completed_at", "duration_ms", "created_at", "updated_at",
	}).AddRow(
		id, "wf-1", "run-1", "tenant-alpha", "proc-1", "Trade Check",
		"event", "order.new", "Trade", "TR-100", "COMPLETED",
		[]byte(`{}`), []byte(`{"result":"ok"}`), sql.NullString{},
		now, &now, 1500, now, now,
	)

	mock.ExpectQuery(`SELECT (.+) FROM public\.bp_workflow_run WHERE tenant_id = \$1 AND entity = \$2 AND entity_id = \$3`).
		WithArgs("tenant-alpha", "Trade", "TR-100", 50).
		WillReturnRows(rows)

	runs, err := tracker.ListRuns(ctx, WorkflowRunFilter{
		TenantID: "tenant-alpha",
		Entity:   "Trade",
		EntityID: "TR-100",
		Limit:    50,
	})

	assert.NoError(t, err)
	require.Len(t, runs, 1)
	assert.Equal(t, "run-1", runs[0].RunID)
	assert.Equal(t, "COMPLETED", runs[0].Status)
	assert.Equal(t, "Trade", runs[0].Entity)
	assert.Equal(t, "TR-100", runs[0].EntityID)
	assert.NoError(t, mock.ExpectationsWereMet())
}
