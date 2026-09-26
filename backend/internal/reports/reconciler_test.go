package reports_test

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/hondyman/uisce/backend/internal/reports"
)

func TestWriter5_SweepStaleExecutions_SweepProducesSWEEPEvents(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	tenantID := uuid.New().String()
	execID := uuid.New()
	cutoff := 15 * time.Minute

	sqlMock.ExpectBegin()
	sqlMock.ExpectQuery("UPDATE public.report_executions").
		WithArgs(cutoff.Seconds()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "error_message"}).
			AddRow(execID.String(), tenantID, "timeout error"))

	sqlMock.ExpectExec("INSERT INTO public.report_execution_events").
		WithArgs(execID.String(), tenantID, "SWEEP_RECONCILED", "running", "failed", "system:sweep", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	sqlMock.ExpectQuery("SELECT e.id, e.tenant_id, e.status, e.created_at").
		WithArgs(reports.Phase1GoLiveDate).
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "status", "created_at"}))

	sqlMock.ExpectCommit()

	swept, broken, err := reports.SweepStaleExecutions(context.Background(), db, cutoff)
	require.NoError(t, err)
	require.Equal(t, int64(1), swept)
	require.Equal(t, int64(0), broken)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestWriter5_SweepStaleExecutions_AtomicityRollback(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	execID := uuid.New()
	cutoff := 15 * time.Minute

	sqlMock.ExpectBegin()
	sqlMock.ExpectQuery("UPDATE public.report_executions").
		WithArgs(cutoff.Seconds()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "error_message"}).
			AddRow(execID.String(), uuid.New().String(), nil))

	sqlMock.ExpectExec("INSERT INTO public.report_execution_events").
		WillReturnError(errFake)

	sqlMock.ExpectRollback()

	_, _, err = reports.SweepStaleExecutions(context.Background(), db, cutoff)
	require.Error(t, err)
	require.Contains(t, err.Error(), "event insert failed")
}

func TestWriter5_SweepStaleExecutions_BrokenChainDetection(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	tenantID := uuid.New().String()
	execID := uuid.New()
	cutoff := 15 * time.Minute

	sqlMock.ExpectBegin()
	sqlMock.ExpectQuery("UPDATE public.report_executions").
		WithArgs(cutoff.Seconds()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "error_message"}))

	sqlMock.ExpectQuery("SELECT e.id, e.tenant_id, e.status, e.created_at").
		WithArgs(reports.Phase1GoLiveDate).
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "status", "created_at"}).
			AddRow(execID.String(), tenantID, "failed", time.Now()))

	sqlMock.ExpectCommit()

	_, broken, err := reports.SweepStaleExecutions(context.Background(), db, cutoff)
	require.NoError(t, err)
	require.Equal(t, int64(1), broken, "broken-chain count should be 1 for terminal-status row missing terminal event")
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestWriter5_SweepStaleExecutions_CutoffExcludesPreInstrumentationRows(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	cutoff := 15 * time.Minute

	sqlMock.ExpectBegin()
	sqlMock.ExpectQuery("UPDATE public.report_executions").
		WithArgs(cutoff.Seconds()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "error_message"}))

	sqlMock.ExpectQuery("SELECT e.id, e.tenant_id, e.status, e.created_at").
		WithArgs(reports.Phase1GoLiveDate).
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "status", "created_at"}))

	sqlMock.ExpectCommit()

	_, broken, err := reports.SweepStaleExecutions(context.Background(), db, cutoff)
	require.NoError(t, err)
	require.Equal(t, int64(0), broken, "no broken chains should be reported when none exist")
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

var errFake = context.DeadlineExceeded
