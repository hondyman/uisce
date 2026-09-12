package reports_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"

	"github.com/hondyman/uisce/backend/internal/reports"
)

func TestAdminListExecutions_FirstPageNoCursor(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := reports.NewAdminExecutionRepository(db)
	callerTenantID := uuid.MustParse("99e99e99-99e9-49e9-89e9-99e99e99e999")
	execID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "tenant_id", "template_id", "schedule_id", "report_key", "status",
		"parameters", "output_url", "output_size_bytes", "rows_processed",
		"execution_time_ms", "error_message", "workflow_id", "run_id",
		"requested_by", "triggered_by", "metadata", "created_at", "completed_at",
		"is_personal", "created_by_id",
	}).AddRow(
		execID, callerTenantID, uuid.New(), nil, "test-report", "completed",
		nil, "", nil, nil, nil, "", "", "",
		"user1", "user1", nil, now, nil,
		false, "",
	)

	sqlMock.ExpectQuery("SELECT e.id, e.tenant_id, e.template_id, e.schedule_id, e.report_key, e.status,").
		WithArgs(nil, nil, nil, nil, nil, nil, 50).
		WillReturnRows(rows)

	execs, nextCursor, err := repo.ListExecutions(context.Background(), nil, "", nil, nil, nil, 50)
	require.NoError(t, err)
	require.Len(t, execs, 1)
	require.Nil(t, nextCursor)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestAdminListExecutions_WithTenantFilter(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := reports.NewAdminExecutionRepository(db)
	tenantID := uuid.New()

	rows := sqlmock.NewRows([]string{
		"id", "tenant_id", "template_id", "schedule_id", "report_key", "status",
		"parameters", "output_url", "output_size_bytes", "rows_processed",
		"execution_time_ms", "error_message", "workflow_id", "run_id",
		"requested_by", "triggered_by", "metadata", "created_at", "completed_at",
		"is_personal", "created_by_id",
	})

	sqlMock.ExpectQuery("SELECT e.id, e.tenant_id, e.template_id, e.schedule_id, e.report_key, e.status,").
		WithArgs(nil, nil, tenantID, nil, nil, nil, 50).
		WillReturnRows(rows)

	execs, _, err := repo.ListExecutions(context.Background(), &tenantID, "", nil, nil, nil, 50)
	require.NoError(t, err)
	require.Len(t, execs, 0)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestAdminListExecutions_NotFound(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := reports.NewAdminExecutionRepository(db)
	execID := uuid.New()

	sqlMock.ExpectQuery("SELECT e.id, e.tenant_id, e.template_id, e.schedule_id, e.report_key, e.status,").
		WithArgs(execID).
		WillReturnError(context.DeadlineExceeded)

	_, err = repo.GetExecution(context.Background(), execID)
	require.Error(t, err)
	require.Equal(t, context.DeadlineExceeded, err)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestAdminGetExecution_NotFound(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := reports.NewAdminExecutionRepository(db)
	execID := uuid.New()

	sqlMock.ExpectQuery("SELECT e.id, e.tenant_id, e.template_id, e.schedule_id, e.report_key, e.status,").
		WithArgs(execID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "tenant_id", "template_id", "schedule_id", "report_key", "status",
			"parameters", "output_url", "output_size_bytes", "rows_processed",
			"execution_time_ms", "error_message", "workflow_id", "run_id",
			"requested_by", "triggered_by", "metadata", "created_at", "completed_at",
			"is_personal", "created_by_id",
		}))

	_, err = repo.GetExecution(context.Background(), execID)
	require.Error(t, err)
	require.Equal(t, reports.ErrNotFound, err)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestAdminListEvents_FirstPage(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := reports.NewAdminExecutionRepository(db)
	tenantID := uuid.New()
	execID := uuid.New()
	evID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "execution_id", "tenant_id", "event", "from_status", "to_status",
		"actor_id", "detail", "created_at",
	}).AddRow(evID, execID, tenantID, "STARTED", "pending", "running", "system:executor", nil, now)

	sqlMock.ExpectQuery("SELECT id, execution_id, tenant_id, event, from_status, to_status,").
		WithArgs(nil, nil, nil, nil, nil, 100).
		WillReturnRows(rows)

	events, nextCursor, err := repo.ListEvents(context.Background(), nil, nil, nil, nil, 100)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Nil(t, nextCursor)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func getAdminDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("ADMIN_READ_DSN")
	if dsn == "" {
		t.Skip("ADMIN_READ_DSN not set; skipping admin DB integration test")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("ADMIN_READ_DSN: failed to open: %v", err)
		return nil
	}
	err = db.Ping()
	if err != nil {
		t.Skipf("ADMIN_READ_DSN: failed to ping (check pg_hba.conf entry for app_admin_read): %v", err)
		_ = db.Close()
		return nil
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestAdminExecutionRepository_LiveAlpha_CrossTenantRead(t *testing.T) {
	adminDB := getAdminDB(t)
	repo := reports.NewAdminExecutionRepository(adminDB)

	execs, _, err := repo.ListExecutions(context.Background(), nil, "", nil, nil, nil, 10)
	require.NoError(t, err)
	t.Logf("Admin list: %d executions visible (cross-tenant)", len(execs))
	for _, e := range execs {
		t.Logf("  id=%s tenant=%s status=%s", e.ID, e.TenantID, e.Status)
	}
}

func TestAdminExecutionRepository_LiveAlpha_WithTenantFilter(t *testing.T) {
	adminDB := getAdminDB(t)
	repo := reports.NewAdminExecutionRepository(adminDB)

	tenantID := uuid.MustParse("70a1172c-9b06-4bc3-a0ac-13f7aaefdd90")
	execs, _, err := repo.ListExecutions(context.Background(), &tenantID, "", nil, nil, nil, 10)
	require.NoError(t, err)
	t.Logf("Admin list with tenant filter: %d executions visible", len(execs))
}
