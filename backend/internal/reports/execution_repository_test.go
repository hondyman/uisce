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

func TestCursor_EncodeDecode_RoundTrip(t *testing.T) {
	c := reports.Cursor{
		Version:   1,
		CreatedAt: time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC),
		ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
	}
	encoded, err := reports.EncodeCursor(c)
	require.NoError(t, err)
	require.NotEmpty(t, encoded)

	decoded, err := reports.DecodeCursor(encoded)
	require.NoError(t, err)
	require.Equal(t, c.Version, decoded.Version)
	require.True(t, c.CreatedAt.Equal(decoded.CreatedAt))
	require.Equal(t, c.ID, decoded.ID)
}

func TestCursor_UnknownVersion_ReturnsErr(t *testing.T) {
	encoded := "eyJ2IjoyLCJjcmVhdGVkX2F0IjoiMjAyNi0wOS0xNVQxMjowMDowMFoiLCJpZCI6IjU1MGU4NDAwLWUyOWItNDFkNC1hNzE2LTQ0NjY1NTQ0MDAwMCJ9"
	_, err := reports.DecodeCursor(encoded)
	require.ErrorIs(t, err, reports.ErrInvalidCursor)
}

func TestCursor_InvalidBase64_ReturnsErr(t *testing.T) {
	_, err := reports.DecodeCursor("not-valid-base64!!!")
	require.ErrorIs(t, err, reports.ErrInvalidCursor)
}

func TestCursor_EmptyString_ReturnsErr(t *testing.T) {
	_, err := reports.DecodeCursor("")
	require.ErrorIs(t, err, reports.ErrInvalidCursor)
}

func TestCursor_ZeroUUID_ReturnsErr(t *testing.T) {
	c := reports.Cursor{
		Version:   1,
		CreatedAt: time.Now(),
		ID:        uuid.Nil,
	}
	encoded, err := reports.EncodeCursor(c)
	require.NoError(t, err)

	_, err = reports.DecodeCursor(encoded)
	require.ErrorIs(t, err, reports.ErrInvalidCursor)
}

func TestCursor_ZeroTime_ReturnsErr(t *testing.T) {
	c := reports.Cursor{
		Version:   1,
		CreatedAt: time.Time{},
		ID:        uuid.New(),
	}
	encoded, err := reports.EncodeCursor(c)
	require.NoError(t, err)

	_, err = reports.DecodeCursor(encoded)
	require.ErrorIs(t, err, reports.ErrInvalidCursor)
}

func TestListExecutions_FirstPageNoCursor(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	tenantID := uuid.New()
	userID := "test-user"
	execID := uuid.New()
	now := time.Now()

	sqlMock.ExpectQuery("SELECT e.id, e.tenant_id, e.template_id, e.schedule_id, e.report_key, e.status").
		WithArgs(nil, nil, tenantID, userID, false, 50).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "tenant_id", "template_id", "schedule_id", "report_key", "status",
			"parameters", "output_url", "output_size_bytes", "rows_processed",
			"execution_time_ms", "error_message", "workflow_id", "run_id",
			"requested_by", "triggered_by", "metadata", "created_at", "completed_at",
			"is_personal", "created_by_id",
		}).AddRow(
			execID, tenantID, uuid.New(), nil, "test-report", "completed",
			[]byte(`{}`), "", 1024, 100, 50, nil, "wf-1", "run-1",
			"user", "", []byte(`{}`), now, nil, false, "",
		))

	repo := reports.NewExecutionRepository(db)
	execs, err := repo.ListExecutions(context.Background(), tenantID, userID, false, nil, 50)
	require.NoError(t, err)
	require.Len(t, execs, 1)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestGetExecution_NotFound_ReturnsErrNotFound(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	tenantID := uuid.New()
	userID := "test-user"
	execID := uuid.New()

	sqlMock.ExpectQuery("SELECT e.id, e.tenant_id, e.template_id, e.schedule_id").
		WithArgs(execID, tenantID, userID, false).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "tenant_id", "template_id", "schedule_id", "report_key", "status",
			"parameters", "output_url", "output_size_bytes", "rows_processed",
			"execution_time_ms", "error_message", "workflow_id", "run_id",
			"requested_by", "triggered_by", "metadata", "created_at", "completed_at",
			"is_personal", "created_by_id",
		}))

	repo := reports.NewExecutionRepository(db)
	_, err = repo.GetExecution(context.Background(), execID, tenantID, userID, false)
	require.ErrorIs(t, err, reports.ErrNotFound)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestListScheduleExecutions_ScheduleOwnership_MissingSchedule(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	tenantID := uuid.New()
	scheduleID := uuid.New()
	userID := "test-user"

	sqlMock.ExpectQuery("SELECT e.id, e.tenant_id, e.template_id, e.schedule_id, e.report_key, e.status").
		WithArgs(scheduleID, tenantID, nil, nil, userID, false, 50).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "tenant_id", "template_id", "schedule_id", "report_key", "status",
			"parameters", "output_url", "output_size_bytes", "rows_processed",
			"execution_time_ms", "error_message", "workflow_id", "run_id",
			"requested_by", "triggered_by", "metadata", "created_at", "completed_at",
			"is_personal", "created_by_id",
		}))

	repo := reports.NewExecutionRepository(db)
	execs, err := repo.ListScheduleExecutions(context.Background(), scheduleID, tenantID, userID, false, nil, 50)
	require.NoError(t, err)
	require.Len(t, execs, 0)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestListExecutionEvents_TwoStep_TenantSwitch(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	execID := uuid.New()
	callerTenantID := uuid.New()
	execTenantID := uuid.MustParse("99e99e99-99e9-49e9-89e9-99e99e99e999")
	userID := "test-user"
	eventID := uuid.New()
	now := time.Now()

	sqlMock.ExpectQuery("SELECT e.tenant_id").
		WithArgs(execID, callerTenantID, userID, false).
		WillReturnRows(sqlmock.NewRows([]string{"tenant_id"}).AddRow(execTenantID))

	sqlMock.ExpectBegin()
	sqlMock.ExpectExec("SELECT set_config").
		WithArgs(execTenantID.String()).
		WillReturnResult(sqlmock.NewResult(0, 0))
	sqlMock.ExpectQuery("SELECT id, execution_id, event, from_status, to_status, actor_id, detail, created_at").
		WithArgs(execID, nil, nil, 1000).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "execution_id", "event", "from_status", "to_status", "actor_id", "detail", "created_at",
		}).AddRow(
			eventID, execID, "CREATED", nil, "pending", userID, []byte(`{}`), now,
		).AddRow(
			uuid.New(), execID, "STARTED", "pending", "running", "system:executor", []byte(`{}`), now,
		).AddRow(
			uuid.New(), execID, "COMPLETED", "running", "completed", "system:activity", []byte(`{}`), now,
		))
	sqlMock.ExpectCommit()

	repo := reports.NewExecutionRepository(db)
	events, truncated, err := repo.ListExecutionEvents(context.Background(), execID, callerTenantID, userID, false, nil, 1000)
	require.NoError(t, err)
	require.Len(t, events, 3)
	require.False(t, truncated)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestListExecutionEvents_NotFound_ReturnsErrNotFound(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	execID := uuid.New()
	callerTenantID := uuid.New()
	userID := "test-user"

	sqlMock.ExpectQuery("SELECT e.tenant_id").
		WithArgs(execID, callerTenantID, userID, false).
		WillReturnRows(sqlmock.NewRows([]string{"tenant_id"}))

	repo := reports.NewExecutionRepository(db)
	_, _, err = repo.ListExecutionEvents(context.Background(), execID, callerTenantID, userID, false, nil, 1000)
	require.ErrorIs(t, err, reports.ErrNotFound)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestListExecutionEvents_CapLimit_1000(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	execID := uuid.New()
	callerTenantID := uuid.New()
	execTenantID := uuid.MustParse("99e99e99-99e9-49e9-89e9-99e99e99e999")
	userID := "test-user"

	sqlMock.ExpectQuery("SELECT e.tenant_id").
		WithArgs(execID, callerTenantID, userID, false).
		WillReturnRows(sqlmock.NewRows([]string{"tenant_id"}).AddRow(execTenantID))

	sqlMock.ExpectBegin()
	sqlMock.ExpectExec("SELECT set_config").
		WithArgs(execTenantID.String()).
		WillReturnResult(sqlmock.NewResult(0, 0))
	sqlMock.ExpectQuery("SELECT id, execution_id, event, from_status, to_status, actor_id, detail, created_at").
		WithArgs(execID, nil, nil, 1000).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "execution_id", "event", "from_status", "to_status", "actor_id", "detail", "created_at",
		}).AddRow(uuid.New(), execID, "CREATED", nil, "pending", userID, []byte(`{}`), time.Now()))
	sqlMock.ExpectCommit()

	repo := reports.NewExecutionRepository(db)
	events, truncated, err := repo.ListExecutionEvents(context.Background(), execID, callerTenantID, userID, false, nil, 5000)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.False(t, truncated)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestListScheduleExecutions_GoldCopyScheduledExecution_VisibleToSchedulingTenant(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	scheduleID := uuid.New()
	callerTenantID := uuid.New()
	execTenantID := uuid.MustParse("99e99e99-99e9-49e9-89e9-99e99e99e999")
	userID := "test-user"
	execID := uuid.New()
	now := time.Now()

	sqlMock.ExpectQuery("SELECT e.id, e.tenant_id, e.template_id, e.schedule_id, e.report_key, e.status").
		WithArgs(scheduleID, callerTenantID, nil, nil, userID, false, 50).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "tenant_id", "template_id", "schedule_id", "report_key", "status",
			"parameters", "output_url", "output_size_bytes", "rows_processed",
			"execution_time_ms", "error_message", "workflow_id", "run_id",
			"requested_by", "triggered_by", "metadata", "created_at", "completed_at",
			"is_personal", "created_by_id",
		}).AddRow(
			execID, execTenantID, uuid.New(), &scheduleID, "core-report", "completed",
			[]byte(`{}`), "", 1024, 100, 50, nil, "wf-1", "run-1",
			"gold-owner", userID, []byte(`{}`), now, nil, false, "",
		))

	repo := reports.NewExecutionRepository(db)
	execs, err := repo.ListScheduleExecutions(context.Background(), scheduleID, callerTenantID, userID, false, nil, 50)
	require.NoError(t, err)
	require.Len(t, execs, 1)
	require.Equal(t, execTenantID, execs[0].TenantID)
	require.Equal(t, userID, execs[0].TriggeredBy.String)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestExecutionRepository_LiveAlpha_ListExecutions(t *testing.T) {
	db := getTestDB(t)
	repo := reports.NewExecutionRepository(db)

	tenantID := uuid.MustParse("70a1172c-9b06-4bc3-a0ac-13f7aaefdd90")
	userID := "test-user-live"
	isAdmin := false

	execs, err := repo.ListExecutions(context.Background(), tenantID, userID, isAdmin, nil, 50)
	require.NoError(t, err)
	t.Logf("Listed %d executions for tenant %s", len(execs), tenantID)
}

func TestExecutionRepository_LiveAlpha_ListExecutionEvents(t *testing.T) {
	db := getTestDB(t)
	repo := reports.NewExecutionRepository(db)

	execID := uuid.MustParse("2fe73389-fa55-4b87-8d50-78b59d0cbac0")
	tenantID := uuid.MustParse("70a1172c-9b06-4bc3-a0ac-13f7aaefdd90")
	userID := "test-user-live"
	isAdmin := false

	events, truncated, err := repo.ListExecutionEvents(context.Background(), execID, tenantID, userID, isAdmin, nil, 5000)
	if err != nil && err.Error() == "report not found" {
		t.Skip("Execution not found or no access - skipping live events test")
	}
	require.NoError(t, err)
	t.Logf("Listed %d events for execution %s (truncated=%v)", len(events), execID, truncated)
}

func TestExecutionRepository_LiveAlpha_ListScheduleExecutions(t *testing.T) {
	db := getTestDB(t)
	repo := reports.NewExecutionRepository(db)

	scheduleID := uuid.MustParse("715b3c96-f441-5c81-98ef-0b04cfe78ad1")
	tenantID := uuid.MustParse("70a1172c-9b06-4bc3-a0ac-13f7aaefdd90")
	userID := "test-user-live"
	isAdmin := false

	execs, err := repo.ListScheduleExecutions(context.Background(), scheduleID, tenantID, userID, isAdmin, nil, 50)
	require.NoError(t, err)
	t.Logf("Listed %d executions for schedule %s", len(execs), scheduleID)
}
