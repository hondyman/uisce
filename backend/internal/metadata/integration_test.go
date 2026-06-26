package metadata

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestGetBusinessObjectIncludesChildIntegration_Container(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pg, err := postgres.RunContainer(ctx,
		postgres.WithDatabase("alpha"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
	)
	require.NoError(t, err)
	defer func() {
		_ = pg.Terminate(ctx)
	}()

	host, err := pg.Host(ctx)
	require.NoError(t, err)
	port, err := pg.MappedPort(ctx, "5432")
	require.NoError(t, err)
	dsn := fmt.Sprintf("postgres://postgres:postgres@%s:%s/alpha?sslmode=disable", host, port.Port())

	// Wait for DB readiness
	var db *sqlx.DB
	retryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	for {
		db, err = sqlx.Open("postgres", dsn)
		if err == nil {
			err = db.PingContext(retryCtx)
		}
		if err == nil {
			break
		}
		select {
		case <-time.After(500 * time.Millisecond):
			continue
		case <-retryCtx.Done():
			require.NoError(t, err)
		}
	}
	defer db.Close()

	// Create minimal tables (only what the service uses)
	_, err = db.ExecContext(ctx, `
	CREATE TABLE business_objects (
		id uuid primary key,
		tenant_id uuid,
		key text,
		name text,
		display_name text,
		technical_name text,
		description text DEFAULT '',
		icon text DEFAULT '',
		is_core boolean default false,
		clones_from text DEFAULT '',
		clone_parent_key text DEFAULT '',
		clone_parent_display_name text DEFAULT '',
		category text DEFAULT '',
		parent_id uuid,
		instance_count integer DEFAULT 0,
		driver_table_id uuid,
		driver_table_name text DEFAULT '',
		datasource_id uuid,
		created_at timestamptz default now(),
		created_by text DEFAULT '',
		last_modified_at timestamptz default now(),
		last_modified_by text DEFAULT '',
		is_active boolean default true,
		config jsonb default '{}'::jsonb
	)
	`)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `
	CREATE TABLE bo_fields (
		id uuid primary key,
		tenant_id uuid not null,
		bo_id uuid not null,
		field_name varchar(255) not null,
		display_label varchar(255),
		field_type varchar(50),
		is_required boolean default false,
		is_readonly boolean default false,
		is_searchable boolean default true,
		is_sortable boolean default true,
		display_order integer default 0
	)
	`)
	require.NoError(t, err)

	// Insert parent and child BOs and an old-schema field for the child
	parentID := uuid.NewString()
	childID := uuid.NewString()
	tenantID := "00000000-0000-0000-0000-000000000000"

	_, err = db.ExecContext(ctx, `INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, created_at) VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, NOW())`, parentID, tenantID, "e2e_parent_test", "E2E Parent", "E2E Parent", "e2e_parent")
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, parent_id, created_at) VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7::uuid, NOW())`, childID, tenantID, "e2e_child_test", "E2E Child", "E2E Child", "e2e_child", parentID)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `INSERT INTO bo_fields (id, tenant_id, bo_id, field_name, display_label, field_type, display_order) VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7)`, uuid.NewString(), tenantID, childID, "f1", "Field 1", "string", 1)
	require.NoError(t, err)

	svc := NewBusinessObjectService(db, nil, nil, nil)
	secCtx := &security.Context{TenantID: tenantID}
	bo, err := svc.GetBusinessObject(ctx, secCtx, parentID)
	require.NoError(t, err)
	require.Equal(t, 1, len(bo.Subtypes), "expected 1 subtype")
	sd, ok := bo.Subtypes["e2e_child_test"]
	require.True(t, ok)
	require.Equal(t, childID, sd.ID)
	require.Equal(t, 1, len(sd.SubtypeFields), "expected 1 subtype field from old schema")
	require.Equal(t, "f1", sd.SubtypeFields[0].Key)
}
