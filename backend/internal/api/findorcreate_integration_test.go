//go:build integration

package api

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	dockertest "github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

func TestFindOrCreateTermNode_ConcurrentInsertRace(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("set RUN_INTEGRATION_TESTS=1 to run concurrent integration tests")
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
	if err := pool.Retry(func() error {
		db, err = sql.Open("postgres", dsn)
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		t.Fatalf("could not connect to docker postgres: %v", err)
	}
	defer db.Close()

	// Create minimal metadata schema and tables
	// TODO(shared-DDL): inline DDL drifts from the actual migration file. When
	// convenient, execute 20261021_001_catalog_node_tenant_path_unique.up.sql via
	// psql -f or file-read instead, so the integration suite also exercises the
	// real migration artifact.
	schemas := []string{
		`CREATE SCHEMA IF NOT EXISTS metadata`,
		`CREATE TABLE IF NOT EXISTS metadata.catalog_node_types (
			id uuid DEFAULT gen_random_uuid() NOT NULL,
			tenant_id uuid NOT NULL,
			catalog_type_name varchar(255) NOT NULL,
			description text,
			is_active boolean DEFAULT true,
			created_at timestamptz DEFAULT now(),
			updated_at timestamptz DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS metadata.catalog_node (
			id uuid DEFAULT gen_random_uuid() NOT NULL,
			node_type_id uuid NOT NULL,
			node_name varchar,
			description text,
			properties jsonb DEFAULT '{}',
			qualified_path varchar NOT NULL,
			parent_id uuid,
			created_at timestamptz DEFAULT now(),
			updated_at timestamptz DEFAULT now(),
			tenant_id uuid NOT NULL,
			is_active boolean DEFAULT true,
			tenant_datasource_id uuid,
			CONSTRAINT catalog_node_pkey PRIMARY KEY (id)
		)`,
	}
	for _, stmt := range schemas {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("schema setup failed: %v\nSQL: %s", err, stmt)
		}
	}

	// Add unique constraint (what the migration does)
	if _, err := db.Exec(`
		ALTER TABLE metadata.catalog_node
		ADD CONSTRAINT catalog_node_tenant_path_uniq
		UNIQUE (tenant_id, qualified_path)`); err != nil {
		t.Fatalf("could not add unique constraint: %v", err)
	}

	// Insert a node_type row so FK is satisfied
	nodeTypeID := uuid.New()
	tenantID := uuid.New()
	if _, err := db.Exec(`
		INSERT INTO metadata.catalog_node_types (id, tenant_id, catalog_type_name)
		VALUES ($1, $2, 'semantic_term')`,
		nodeTypeID, tenantID); err != nil {
		t.Fatalf("could not insert node type: %v", err)
	}

	// Create service pointing at the test db
	svc := NewGlossaryService(context.Background(), db, nil, NewInMemoryJobStore())

	const goroutines = 8
	qualifiedPrefix := "semantic_term"
	name := fmt.Sprintf("concurrent-test-%s", uuid.New().String())
	qualifiedPath := qualifiedPrefix + "/" + name

	resultIDs := make(chan string, goroutines)
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id, reused, err := svc.findOrCreateTermNode(
				context.Background(),
				tenantID.String(), "none", nodeTypeID.String(),
				qualifiedPrefix, name, "test definition",
			)
			if err != nil {
				t.Errorf("findOrCreateTermNode returned error: %v", err)
				return
			}
			resultIDs <- id
			_ = reused // may be true or false depending on scheduling
		}()
	}
	wg.Wait()
	close(resultIDs)

	// Collect all returned IDs
	var ids []string
	for id := range resultIDs {
		ids = append(ids, id)
	}

	// Assert: exactly one row in the table
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM metadata.catalog_node WHERE qualified_path = $1`, qualifiedPath).Scan(&count); err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected exactly 1 row for qualified_path=%q, got %d", qualifiedPath, count)
	}

	// Assert: all callers got the same ID
	seen := make(map[string]bool)
	var firstID string
	for _, id := range ids {
		if firstID == "" {
			firstID = id
		}
		if id != firstID {
			t.Errorf("goroutines saw different IDs: first=%s, this=%s", firstID, id)
		}
		seen[id] = true
	}
	if len(seen) != 1 {
		t.Errorf("expected 1 unique ID from all goroutines, got %d: %v", len(seen), seen)
	}
}
