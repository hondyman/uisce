package integration

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/ory/dockertest/v3"
)

// StartPostgres starts a Postgres container and returns a *sql.DB connection and a cleanup func.
func StartPostgres(t testing.TB) (*sql.DB, func()) {
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping docker integration tests in CI")
	}

	pool, err := dockertest.NewPool("")
	if err != nil {
		t.Fatalf("could not connect to docker: %v", err)
	}

	opts := &dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "15-alpine",
		Env: []string{
			"POSTGRES_USER=semlayer",
			"POSTGRES_PASSWORD=secret",
			"POSTGRES_DB=semlayer_test",
		},
		ExposedPorts: []string{"5432/tcp"},
		// rely on pool.Retry below to wait for DB readiness
	}

	resource, err := pool.RunWithOptions(opts)
	if err != nil {
		t.Fatalf("could not start resource: %v", err)
	}

	cleanup := func() {
		_ = pool.Purge(resource)
	}

	var db *sql.DB
	// exponential backoff-retry, try to connect
	err = pool.Retry(func() error {
		var err error
		dsn := fmt.Sprintf("postgres://semlayer:secret@localhost:%s/semlayer_test?sslmode=disable", resource.GetPort("5432/tcp"))
		db, err = sql.Open("postgres", dsn)
		if err != nil {
			return err
		}
		return db.Ping()
	})
	if err != nil {
		cleanup()
		t.Fatalf("could not connect to database: %v", err)
	}

	// set a reasonable connection lifetime

	db.SetConnMaxLifetime(time.Minute * 3)

	return db, cleanup
}
