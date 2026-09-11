package reports_test

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/mocks"

	"github.com/hondyman/uisce/backend/internal/reports"
)

func TestWriter1_CreatedEvent_AtomicityRollback(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	tenantID := uuid.New().String()
	tmpl := &reports.ReportTemplate{
		ID:           uuid.New(),
		TenantID:     uuid.MustParse(tenantID),
		TemplateName: "test-template",
		CreatedByID:  ptrStr("user-owner-001"),
	}
	params := map[string]interface{}{
		"triggered_by": "user-trigger-001",
		"schedule_id":  "sched-001",
	}

	mockTemporal := &mocks.Client{}
	mockTemporal.On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&mocks.WorkflowRun{}, nil)
	mockTemporal.On("Close").Return(nil)

	sqlMock.ExpectBegin()
	sqlMock.ExpectExec("SELECT set_config").
		WillReturnResult(sqlmock.NewResult(0, 0))
	sqlMock.ExpectExec("INSERT INTO public.report_executions").
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectExec("INSERT INTO public.report_execution_events").
		WillReturnError(errors.New("foreign key violation — execution_id not found"))
	sqlMock.ExpectRollback()

	executor := reports.NewTemporalReportExecutor(db, mockTemporal)
	_, err = executor.ExecuteReport(context.Background(), tmpl, params)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "foreign key")
	require.NoError(t, sqlMock.ExpectationsWereMet(), "all mock expectations must be met")
}

func TestWriter1_CreatedEvent_ActorVocabulary_TriggeredBySet(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	tenantID := uuid.New().String()
	tmpl := &reports.ReportTemplate{
		ID:           uuid.New(),
		TenantID:     uuid.MustParse(tenantID),
		TemplateName: "test-template",
		CreatedByID:  ptrStr("owner-001"),
	}
	params := map[string]interface{}{
		"triggered_by": "user-trigger-001",
	}

	mockTemporal := &mocks.Client{}
	workflowRun := &mocks.WorkflowRun{}
	workflowRun.On("GetID").Return("wf-001")
	workflowRun.On("GetRunID").Return("run-001")
	mockTemporal.On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(workflowRun, nil)
	mockTemporal.On("Close").Return(nil)

	// Writer 1: CREATED transaction
	sqlMock.ExpectBegin()
	sqlMock.ExpectExec("SELECT set_config").
		WillReturnResult(sqlmock.NewResult(0, 0))
	sqlMock.ExpectExec("INSERT INTO public.report_executions").
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectExec("INSERT INTO public.report_execution_events").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "user-trigger-001", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectCommit()

	// Writer 2: STARTED transaction (needed to satisfy the full ExecuteReport flow)
	sqlMock.ExpectBegin()
	sqlMock.ExpectExec("SELECT set_config").
		WillReturnResult(sqlmock.NewResult(0, 0))
	sqlMock.ExpectExec("UPDATE public.report_executions").
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectExec("INSERT INTO public.report_execution_events").
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectCommit()

	executor := reports.NewTemporalReportExecutor(db, mockTemporal)
	_, err = executor.ExecuteReport(context.Background(), tmpl, params)

	require.NoError(t, err)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestWriter1_CreatedEvent_ActorVocabulary_SystemSchedulerFallback(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	tenantID := uuid.New().String()
	tmpl := &reports.ReportTemplate{
		ID:           uuid.New(),
		TenantID:     uuid.MustParse(tenantID),
		TemplateName: "test-template",
		CreatedByID:  ptrStr("owner-001"),
	}
	params := map[string]interface{}{}

	mockTemporal := &mocks.Client{}
	workflowRun := &mocks.WorkflowRun{}
	workflowRun.On("GetID").Return("wf-001")
	workflowRun.On("GetRunID").Return("run-001")
	mockTemporal.On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(workflowRun, nil)
	mockTemporal.On("Close").Return(nil)

	// Writer 1: CREATED transaction
	sqlMock.ExpectBegin()
	sqlMock.ExpectExec("SELECT set_config").
		WillReturnResult(sqlmock.NewResult(0, 0))
	sqlMock.ExpectExec("INSERT INTO public.report_executions").
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectExec("INSERT INTO public.report_execution_events").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "system:scheduler", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectCommit()

	// Writer 2: STARTED transaction
	sqlMock.ExpectBegin()
	sqlMock.ExpectExec("SELECT set_config").
		WillReturnResult(sqlmock.NewResult(0, 0))
	sqlMock.ExpectExec("UPDATE public.report_executions").
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectExec("INSERT INTO public.report_execution_events").
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectCommit()

	executor := reports.NewTemporalReportExecutor(db, mockTemporal)
	_, err = executor.ExecuteReport(context.Background(), tmpl, params)

	require.NoError(t, err)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestWriter1_CreatedEvent_DetailWithScheduleID(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	tenantID := uuid.New().String()
	tmpl := &reports.ReportTemplate{
		ID:           uuid.New(),
		TenantID:     uuid.MustParse(tenantID),
		TemplateName: "test-template",
		CreatedByID:  ptrStr("owner-001"),
	}
	params := map[string]interface{}{
		"triggered_by": "user-001",
		"schedule_id":  "sched-001",
	}

	mockTemporal := &mocks.Client{}
	workflowRun := &mocks.WorkflowRun{}
	workflowRun.On("GetID").Return("wf-001")
	workflowRun.On("GetRunID").Return("run-001")
	mockTemporal.On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(workflowRun, nil)
	mockTemporal.On("Close").Return(nil)

	// Writer 1: CREATED transaction
	sqlMock.ExpectBegin()
	sqlMock.ExpectExec("SELECT set_config").
		WillReturnResult(sqlmock.NewResult(0, 0))
	sqlMock.ExpectExec("INSERT INTO public.report_executions").
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectExec("INSERT INTO public.report_execution_events").
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectCommit()

	// Writer 2: STARTED transaction
	sqlMock.ExpectBegin()
	sqlMock.ExpectExec("SELECT set_config").
		WillReturnResult(sqlmock.NewResult(0, 0))
	sqlMock.ExpectExec("UPDATE public.report_executions").
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectExec("INSERT INTO public.report_execution_events").
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectCommit()

	executor := reports.NewTemporalReportExecutor(db, mockTemporal)
	_, err = executor.ExecuteReport(context.Background(), tmpl, params)

	require.NoError(t, err)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func ptrStr(s string) *string { return &s }

// Writer 2 — STARTED event atomicity rollback test
// Proves: if the STARTED event insert fails, the running-status UPDATE rolls back.
func TestWriter2_StartedEvent_AtomicityRollback(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	tenantID := uuid.New().String()
	tmpl := &reports.ReportTemplate{
		ID:           uuid.New(),
		TenantID:     uuid.MustParse(tenantID),
		TemplateName:  "test-template",
		CreatedByID:  ptrStr("owner-001"),
	}
	params := map[string]interface{}{}

	mockTemporal := &mocks.Client{}
	workflowRun := &mocks.WorkflowRun{}
	workflowRun.On("GetID").Return("wf-001")
	workflowRun.On("GetRunID").Return("run-001")
	mockTemporal.On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(workflowRun, nil)
	mockTemporal.On("Close").Return(nil)

	// Writer 1: CREATED transaction (succeeds)
	sqlMock.ExpectBegin()
	sqlMock.ExpectExec("SELECT set_config").WillReturnResult(sqlmock.NewResult(0, 0))
	sqlMock.ExpectExec("INSERT INTO public.report_executions").
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectExec("INSERT INTO public.report_execution_events").
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectCommit()

	// Writer 2: STARTED transaction — UPDATE succeeds, event insert fails → ROLLBACK
	sqlMock.ExpectBegin()
	sqlMock.ExpectExec("SELECT set_config").WillReturnResult(sqlmock.NewResult(0, 0))
	sqlMock.ExpectExec("UPDATE public.report_executions").
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectExec("INSERT INTO public.report_execution_events").
		WillReturnError(errors.New("event insert failed"))
	sqlMock.ExpectRollback()

	executor := reports.NewTemporalReportExecutor(db, mockTemporal)
	_, err = executor.ExecuteReport(context.Background(), tmpl, params)

	assert.Error(t, err)
	require.NoError(t, sqlMock.ExpectationsWereMet(), "all mock expectations must be met")
}

// Writer 3 — FAILED event atomicity rollback test
// Proves: if the FAILED event insert fails, the failed-status UPDATE rolls back.
func TestWriter3_FailedEvent_AtomicityRollback(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	tenantID := uuid.New().String()
	execID := uuid.New()

	mockTemporal := &mocks.Client{}
	mockTemporal.On("Close").Return(nil)

	// markExecutionFailed: UPDATE succeeds, event insert fails → ROLLBACK
	sqlMock.ExpectBegin()
	sqlMock.ExpectExec("SELECT set_config").WillReturnResult(sqlmock.NewResult(0, 0))
	sqlMock.ExpectExec("UPDATE public.report_executions").
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectExec("INSERT INTO public.report_execution_events").
		WillReturnError(errors.New("event insert failed"))
	sqlMock.ExpectRollback()

	executor := reports.NewTemporalReportExecutor(db, mockTemporal)
	executor.MarkExecutionFailed(context.Background(), tenantID, execID, "test error")

	require.NoError(t, sqlMock.ExpectationsWereMet(), "all mock expectations must be met")
}

// Writer 3 — actor vocabulary: system:executor
func TestWriter3_FailedEvent_ActorIsSystemExecutor(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	tenantID := uuid.New().String()
	execID := uuid.New()

	mockTemporal := &mocks.Client{}
	mockTemporal.On("Close").Return(nil)

	sqlMock.ExpectBegin()
	sqlMock.ExpectExec("SELECT set_config").WillReturnResult(sqlmock.NewResult(0, 0))
	sqlMock.ExpectExec("UPDATE public.report_executions").
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectExec("INSERT INTO public.report_execution_events").
		WithArgs(execID, tenantID, "FAILED", "running", "failed", "system:executor", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectCommit()

	executor := reports.NewTemporalReportExecutor(db, mockTemporal)
	executor.MarkExecutionFailed(context.Background(), tenantID, execID, "test error")

	require.NoError(t, sqlMock.ExpectationsWereMet())

	require.NoError(t, sqlMock.ExpectationsWereMet())
}
