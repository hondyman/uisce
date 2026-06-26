package api

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	dockertest "github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

// TestProfilerE2E starts a real Postgres container, creates a table with many
// columns and runs the real profiler against it to validate end-to-end
// behavior and batching performance differences.
func TestProfilerE2E(t *testing.T) {
	// Opt-in: only run this test when RUN_PROFILER_E2E=1 is set in the environment
	if os.Getenv("RUN_PROFILER_E2E") != "1" {
		t.Skip("skipping E2E profiler test; set RUN_PROFILER_E2E=1 to enable")
	}

	pool, err := dockertest.NewPool("")
	if err != nil {
		t.Fatalf("could not connect to docker: %v", err)
	}

	options := &dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "15-alpine",
		Env:        []string{"POSTGRES_DB=alpha", "POSTGRES_PASSWORD=postgres", "POSTGRES_USER=postgres"},
	}
	resource, err := pool.RunWithOptions(options, func(hostConfig *docker.HostConfig) {
		hostConfig.AutoRemove = true
		hostConfig.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		t.Fatalf("could not start resource: %v", err)
	}
	defer func() {
		_ = pool.Purge(resource)
	}()

	var db *sql.DB
	dsn := fmt.Sprintf("postgres://postgres:postgres@localhost:%s/alpha?sslmode=disable", resource.GetPort("5432/tcp"))
	// Wait for postgres to be up
	if err := pool.Retry(func() error {
		var err error
		db, err = sql.Open("postgres", dsn)
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		t.Fatalf("could not connect to docker postgres: %v", err)
	}
	defer db.Close()

	// Create sml schema and column_profiles table expected by profiler
	_, _ = db.Exec(`CREATE SCHEMA IF NOT EXISTS sml`)
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS sml.column_profiles (
        tenant_id text,
        datasource_id text,
        datasource text,
        schema text,
        table_name text,
        column_name text,
        data_type text,
        cardinality bigint,
        min_length integer,
        max_length integer,
        avg_length double precision,
        frequent_values text[],
        inferred_patterns text[],
        bloom_filter bytea,
        created_at timestamptz default now()
    )`)
	if err != nil {
		t.Fatalf("failed to create column_profiles: %v", err)
	}

	// Create a wide table with many columns to make batching matter
	colCount := 200
	colDefs := ""
	for i := 1; i <= colCount; i++ {
		if i > 1 {
			colDefs += ","
		}
		colDefs += fmt.Sprintf("c%d text", i)
	}
	_, err = db.Exec(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS public.wide_table (%s)`, colDefs))
	if err != nil {
		t.Fatalf("failed to create wide_table: %v", err)
	}

	// Insert a few rows
	values := ""
	for i := 0; i < 10; i++ {
		if i > 0 {
			values += ","
		}
		values += "('sample')"
	}
	if values != "" {
		_, _ = db.Exec(fmt.Sprintf("INSERT INTO public.wide_table VALUES %s", values))
	}

	// Now run the profiler with two different batch sizes and time them
	srv := &Server{WsHub: newWebSocketHub()}
	go srv.WsHub.run()
	// Allow runProfile to connect to this test Postgres
	_ = os.Setenv("ALPHA_DB_URL", dsn)

	// Create jobs
	smallJob := generateJobID()
	srv.ProfileJobs.Store(smallJob, &ProfileJob{ID: smallJob, Status: "pending", CreatedAt: time.Now(), Req: ProfileRequest{DataSource: dsn, Schema: "public", Tables: []string{"wide_table"}, BatchSize: 1}})
	largeJob := generateJobID()
	srv.ProfileJobs.Store(largeJob, &ProfileJob{ID: largeJob, Status: "pending", CreatedAt: time.Now(), Req: ProfileRequest{DataSource: dsn, Schema: "public", Tables: []string{"wide_table"}, BatchSize: 100}})

	startSmall := time.Now()
	srv.runProfile(smallJob)
	durSmall := time.Since(startSmall)

	startLarge := time.Now()
	srv.runProfile(largeJob)
	durLarge := time.Since(startLarge)

	t.Logf("durations small=%v large=%v", durSmall, durLarge)

	if durLarge >= durSmall {
		t.Fatalf("expected large batch to be faster than small batch")
	}

	// Verify profiles were written
	rows, err := db.Query("SELECT count(*) FROM sml.column_profiles WHERE table_name = 'wide_table'")
	if err != nil {
		t.Fatalf("failed to query column_profiles: %v", err)
	}
	var cnt int
	if rows.Next() {
		rows.Scan(&cnt)
	}
	if cnt == 0 {
		t.Fatalf("no profiles written to column_profiles")
	}
}
