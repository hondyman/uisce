package catalog

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCatalogService_UpsertScheduledJob(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewCatalogService(db, "tenant1", "datasource1")

	job := ScheduledJob{
		ID:                "job1",
		TenantID:          "tenant1",
		DatasourceID:      "datasource1",
		CubeName:          "test_cube",
		PreName:           "test_pre_agg",
		CronExpr:          "0 0 * * *",
		Storage:           "s3://bucket/path",
		RefreshKey:        map[string]interface{}{"date": "2023-01-01"},
		LastRun:           &time.Time{},
		LastRefreshKeyVal: "2023-01-01",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	refreshKeyJSON, _ := json.Marshal(job.RefreshKey)

	mock.ExpectExec(`INSERT INTO public.scheduled_jobs`).
		WithArgs(
			job.ID, job.TenantID, job.DatasourceID, job.CubeName, job.PreName,
			job.CronExpr, job.Storage, refreshKeyJSON, job.LastRun, job.LastRefreshKeyVal,
			job.CreatedAt, job.UpdatedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = service.UpsertScheduledJob(job)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCatalogService_UpsertScheduledJob_NilRefreshKey(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewCatalogService(db, "tenant1", "datasource1")

	job := ScheduledJob{
		ID:                "job1",
		TenantID:          "tenant1",
		DatasourceID:      "datasource1",
		CubeName:          "test_cube",
		PreName:           "test_pre_agg",
		CronExpr:          "0 0 * * *",
		Storage:           "s3://bucket/path",
		RefreshKey:        nil,
		LastRun:           nil,
		LastRefreshKeyVal: "",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	mock.ExpectExec(`INSERT INTO public.scheduled_jobs`).
		WithArgs(
			job.ID, job.TenantID, job.DatasourceID, job.CubeName, job.PreName,
			job.CronExpr, job.Storage, []byte("null"), job.LastRun, job.LastRefreshKeyVal,
			job.CreatedAt, job.UpdatedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = service.UpsertScheduledJob(job)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCatalogService_DeleteScheduledJob(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewCatalogService(db, "tenant1", "datasource1")

	mock.ExpectExec(`DELETE FROM public.scheduled_jobs WHERE id = \$1`).
		WithArgs("job1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = service.DeleteScheduledJob("job1")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCatalogService_ListScheduledJobs(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewCatalogService(db, "tenant1", "datasource1")

	refreshKey := map[string]interface{}{"date": "2023-01-01"}
	refreshKeyJSON, _ := json.Marshal(refreshKey)
	lastRun := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "tenant_id", "datasource_id", "cube_name", "pre_name",
		"cron_expr", "storage", "refresh_key", "last_run", "last_refresh_key_val",
		"created_at", "updated_at",
	}).
		AddRow(
			"job1", "tenant1", "datasource1", "test_cube", "test_pre_agg",
			"0 0 * * *", "s3://bucket/path", string(refreshKeyJSON), lastRun, "2023-01-01",
			time.Now(), time.Now(),
		)

	mock.ExpectQuery(`SELECT .* FROM public.scheduled_jobs WHERE datasource_id = \$1`).
		WithArgs("datasource1").
		WillReturnRows(rows)

	jobs, err := service.ListScheduledJobs()
	assert.NoError(t, err)
	assert.Len(t, jobs, 1)
	assert.Equal(t, "job1", jobs[0].ID)
	assert.Equal(t, "tenant1", jobs[0].TenantID)
	assert.Equal(t, "datasource1", jobs[0].DatasourceID)
	assert.Equal(t, "test_cube", jobs[0].CubeName)
	assert.Equal(t, "test_pre_agg", jobs[0].PreName)
	assert.Equal(t, "0 0 * * *", jobs[0].CronExpr)
	assert.Equal(t, "s3://bucket/path", jobs[0].Storage)
	assert.NotNil(t, jobs[0].RefreshKey)
	assert.Equal(t, "2023-01-01", jobs[0].RefreshKey["date"])
	assert.NotNil(t, jobs[0].LastRun)
	assert.Equal(t, "2023-01-01", jobs[0].LastRefreshKeyVal)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCatalogService_ListScheduledJobs_NullValues(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewCatalogService(db, "tenant1", "datasource1")

	rows := sqlmock.NewRows([]string{
		"id", "tenant_id", "datasource_id", "cube_name", "pre_name",
		"cron_expr", "storage", "refresh_key", "last_run", "last_refresh_key_val",
		"created_at", "updated_at",
	}).
		AddRow(
			"job1", "tenant1", "datasource1", "test_cube", "test_pre_agg",
			"0 0 * * *", "s3://bucket/path", nil, nil, nil,
			time.Now(), time.Now(),
		)

	mock.ExpectQuery(`SELECT .* FROM public.scheduled_jobs WHERE datasource_id = \$1`).
		WithArgs("datasource1").
		WillReturnRows(rows)

	jobs, err := service.ListScheduledJobs()
	assert.NoError(t, err)
	assert.Len(t, jobs, 1)
	assert.Equal(t, "job1", jobs[0].ID)
	assert.Nil(t, jobs[0].RefreshKey)
	assert.Nil(t, jobs[0].LastRun)
	assert.Equal(t, "", jobs[0].LastRefreshKeyVal)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCatalogService_RecordJobRun(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewCatalogService(db, "tenant1", "datasource1")

	started := time.Now()
	finished := time.Now().Add(time.Minute)
	message := "Job completed successfully"

	mock.ExpectExec(`INSERT INTO public.scheduled_job_runs`).
		WithArgs("job1", started, &finished, true, message).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = service.RecordJobRun("job1", started, &finished, true, message)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCatalogService_RecordJobRun_NilFinished(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewCatalogService(db, "tenant1", "datasource1")

	started := time.Now()
	message := "Job started"

	mock.ExpectExec(`INSERT INTO public.scheduled_job_runs`).
		WithArgs("job1", started, nil, false, message).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = service.RecordJobRun("job1", started, nil, false, message)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
