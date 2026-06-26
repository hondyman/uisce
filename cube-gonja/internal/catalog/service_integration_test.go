package catalog

import (
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCatalogService_Integration(t *testing.T) {
	// Skip integration tests if no database URL is provided
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("Skipping integration test: TEST_DATABASE_URL not set")
	}

	// Connect to test database
	db, err := sql.Open("postgres", dbURL)
	require.NoError(t, err)
	defer db.Close()

	// Ping database to ensure connection
	err = db.Ping()
	require.NoError(t, err)

	// Create test tables
	setupTestTables(t, db)

	// Clean up after test
	defer cleanupTestTables(t, db)

	// Create service
	service := NewCatalogService(db, "test_tenant", "test_datasource")

	t.Run("UpsertAndListScheduledJobs", func(t *testing.T) {
		// Create a test job
		job := ScheduledJob{
			ID:                "integration_test_job",
			TenantID:          "test_tenant",
			DatasourceID:      "test_datasource",
			CubeName:          "test_cube",
			PreName:           "test_pre_agg",
			CronExpr:          "0 0 * * *",
			Storage:           "s3://test-bucket/path",
			RefreshKey:        map[string]interface{}{"date": "2023-01-01"},
			LastRefreshKeyVal: "2023-01-01",
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		// Insert job
		err := service.UpsertScheduledJob(job)
		assert.NoError(t, err)

		// List jobs
		jobs, err := service.ListScheduledJobs()
		assert.NoError(t, err)
		assert.Len(t, jobs, 1)
		assert.Equal(t, job.ID, jobs[0].ID)
		assert.Equal(t, job.CubeName, jobs[0].CubeName)
		assert.Equal(t, job.PreName, jobs[0].PreName)
		assert.NotNil(t, jobs[0].RefreshKey)
		assert.Equal(t, "2023-01-01", jobs[0].RefreshKey["date"])
	})

	t.Run("RecordJobRun", func(t *testing.T) {
		started := time.Now()
		finished := time.Now().Add(time.Minute)
		message := "Integration test job run"

		// Record job run
		err := service.RecordJobRun("integration_test_job", started, &finished, true, message)
		assert.NoError(t, err)

		// Verify job run was recorded (we can't easily query it without more setup,
		// but at least verify no error occurred)
	})

	t.Run("DeleteScheduledJob", func(t *testing.T) {
		// Delete job
		err := service.DeleteScheduledJob("integration_test_job")
		assert.NoError(t, err)

		// Verify job was deleted
		jobs, err := service.ListScheduledJobs()
		assert.NoError(t, err)
		assert.Len(t, jobs, 0)
	})
}

func setupTestTables(t *testing.T, db *sql.DB) {
	// Create test tables
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS public.scheduled_jobs (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			datasource_id TEXT NOT NULL,
			cube_name TEXT NOT NULL,
			pre_name TEXT NOT NULL,
			cron_expr TEXT,
			storage TEXT,
			refresh_key JSONB,
			last_run TIMESTAMP WITH TIME ZONE,
			last_refresh_key_val TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
		);

		CREATE TABLE IF NOT EXISTS public.scheduled_job_runs (
			id SERIAL PRIMARY KEY,
			job_id TEXT NOT NULL,
			started_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
			finished_at TIMESTAMP WITH TIME ZONE,
			success BOOLEAN,
			message TEXT
		);
	`)
	require.NoError(t, err)
}

func cleanupTestTables(t *testing.T, db *sql.DB) {
	// Clean up test data
	_, err := db.Exec(`
		DROP TABLE IF EXISTS public.scheduled_job_runs;
		DROP TABLE IF EXISTS public.scheduled_jobs;
	`)
	if err != nil {
		t.Logf("Warning: Failed to cleanup test tables: %v", err)
	}
}
