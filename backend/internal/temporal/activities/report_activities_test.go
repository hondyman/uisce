package activities_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hondyman/uisce/backend/internal/temporal/activities"
)

func TestWriter4_CompletedEvent_AtomicityRollback(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	executionID := uuid.New().String()
	tenantID := uuid.New().String()
	input := activities.GenerateArtifactInput{
		ExecutionID: executionID,
		Template: activities.HydratedReportTemplate{
			ID:       uuid.New().String(),
			TenantID: tenantID,
		},
	}
	result := activities.ArtifactResult{
		OutputURL:        "s3://reports/test.pdf",
		OutputSizeBytes: 1024,
		RowsProcessed:    100,
		ExecutionTimeMS:  500,
		CompletedAt:      time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC),
		Engine:           "duckdb",
	}

	act := activities.NewReportActivities(db)

	// UPDATE succeeds, event insert fails → ROLLBACK
	sqlMock.ExpectBegin()
	sqlMock.ExpectExec("SELECT set_config").
		WillReturnResult(sqlmock.NewResult(0, 0))
	sqlMock.ExpectExec("UPDATE public.report_executions").
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectExec("INSERT INTO public.report_execution_events").
		WillReturnError(errors.New("event insert failed"))
	sqlMock.ExpectRollback()

	err = act.StoreExecutionResultActivity(context.Background(), input, result)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "event insert failed")
	require.NoError(t, sqlMock.ExpectationsWereMet(), "all mock expectations must be met")
}

func TestWriter4_CompletedEvent_ActorIsSystemActivity(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	executionID := uuid.New().String()
	tenantID := uuid.New().String()
	input := activities.GenerateArtifactInput{
		ExecutionID: executionID,
		Template: activities.HydratedReportTemplate{
			ID:       uuid.New().String(),
			TenantID: tenantID,
		},
	}
	result := activities.ArtifactResult{
		OutputURL:        "s3://reports/test.pdf",
		OutputSizeBytes: 1024,
		RowsProcessed:    100,
		ExecutionTimeMS:  500,
		CompletedAt:      time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC),
		Engine:           "duckdb",
	}

	act := activities.NewReportActivities(db)

	sqlMock.ExpectBegin()
	sqlMock.ExpectExec("SELECT set_config").
		WillReturnResult(sqlmock.NewResult(0, 0))
	sqlMock.ExpectExec("UPDATE public.report_executions").
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectExec("INSERT INTO public.report_execution_events").
		WithArgs(sqlmock.AnyArg(), tenantID, "COMPLETED", "running", "completed", "system:activity", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectCommit()

	err = act.StoreExecutionResultActivity(context.Background(), input, result)

	require.NoError(t, err)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestWriter4_CompletedEvent_DetailContainsOutputMetrics(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	executionID := uuid.New().String()
	tenantID := uuid.New().String()
	input := activities.GenerateArtifactInput{
		ExecutionID: executionID,
		Template: activities.HydratedReportTemplate{
			ID:       uuid.New().String(),
			TenantID: tenantID,
		},
	}
	result := activities.ArtifactResult{
		OutputURL:        "s3://reports/test.pdf",
		OutputSizeBytes: 1024,
		RowsProcessed:    100,
		ExecutionTimeMS:  500,
		CompletedAt:      time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC),
		Engine:           "duckdb",
	}

	act := activities.NewReportActivities(db)

	sqlMock.ExpectBegin()
	sqlMock.ExpectExec("SELECT set_config").
		WillReturnResult(sqlmock.NewResult(0, 0))
	sqlMock.ExpectExec("UPDATE public.report_executions").
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectExec("INSERT INTO public.report_execution_events").
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectCommit()

	err = act.StoreExecutionResultActivity(context.Background(), input, result)

	require.NoError(t, err)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}
