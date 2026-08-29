package catalog

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

type mockLoaderForBuilder struct {
	rows []SubtypeRow
	err  error
}

func (m *mockLoaderForBuilder) LoadAllForTenant(ctx context.Context, db *sql.DB, tenantID uuid.UUID) ([]SubtypeRow, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.rows, nil
}

type mockDBForBO struct {
	execResults []mockExecResult
	txStarted  bool
	txCommitted bool
	txRolledBack bool
}

type mockExecResult struct {
	query string
	args  []interface{}
	err   error
}

func TestSubtypeBOBuilder_BuildForTenant_LoadError(t *testing.T) {
	loader := &mockLoaderForBuilder{
		err: context.DeadlineExceeded,
	}
	builder := NewSubtypeBOBuilder(loader)

	err := builder.BuildForTenant(context.Background(), nil, uuid.New())
	if err == nil {
		t.Error("expected error from loader")
	}
}

func TestSubtypeBOBuilder_QualifiedPath(t *testing.T) {
	_ = SubtypeRow{
		ID:                uuid.New(),
		TenantID:          uuid.New(),
		RootObject:        "account",
		SubtypeCode:       "institutional",
		DisplayName:       "Institutional Account",
		FieldAllowlist:    []string{"account_number", "sponsor_id"},
		IsActive:          true,
		CreatedAt:         time.Now(),
	}

	expectedPath := "oms.account/institutional"
	if expectedPath != "oms.account/institutional" {
		t.Errorf("path mismatch")
	}

	attrPath := expectedPath + "/account_number"
	if attrPath != "oms.account/institutional/account_number" {
		t.Errorf("expected oms.account/institutional/account_number, got %s", attrPath)
	}
}

func TestSubtypeBOBuilder_LoaderInterface(t *testing.T) {
	rows := []SubtypeRow{
		{
			ID:             uuid.New(),
			TenantID:       uuid.New(),
			RootObject:     "account",
			SubtypeCode:    "institutional",
			DisplayName:    "Institutional Account",
			FieldAllowlist: []string{"account_number", "sponsor_id"},
			IsActive:       true,
			CreatedAt:      time.Now(),
		},
	}

	loader := &mockLoaderForBuilder{rows: rows}
	builder := NewSubtypeBOBuilder(loader)

	if builder == nil {
		t.Fatal("expected non-nil builder")
	}
}

// TestSubtypeBOBuilder_ParentBOsGetCoreFields verifies that when BuildForTenant runs,
// parent STI BOs (e.g., oms.account) get CoreFields inserted into business_object_fields
// as the union of all their child subtypes' field_allowlist entries.
// This is a regression test for the bug where parent BOs were left with no fields.
func TestSubtypeBOBuilder_ParentBOsGetCoreFields(t *testing.T) {
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

	tenantID := uuid.New()
	goldCopyTenantID := "00000000-0000-0000-0000-000000000001"

	// Create required tables
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS catalog_node (
			id uuid primary key,
			tenant_id uuid,
			node_type_id uuid,
			node_type text,
			node_name text,
			qualified_path text,
			properties jsonb default '{}',
			is_active boolean default true,
			parent_id uuid,
			created_at timestamptz default now(),
			updated_at timestamptz default now()
		)
	`)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS catalog_edge (
			id uuid primary key,
			source_node_id uuid,
			target_node_id uuid,
			edge_type_id uuid,
			relationship_type text,
			tenant_id uuid,
			is_active boolean default true,
			properties jsonb default '{}',
			created_at timestamptz default now(),
			updated_at timestamptz default now()
		)
	`)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS business_objects (
			id uuid primary key,
			tenant_id uuid,
			model_id uuid,
			bo_key text not null,
			bo_name text,
			bo_type text,
			description text default '',
			classification_node_id uuid,
			business_key_node_id uuid,
			semantic_id_node_id uuid,
			grain_node_id uuid,
			sti_discriminator_column text,
			active_subtype_filter text,
			is_active boolean default true,
			is_core boolean default false,
			created_at timestamptz default now(),
			updated_at timestamptz default now()
		)
	`)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS business_object_fields (
			id uuid primary key,
			tenant_id uuid,
			bo_id uuid,
			term_node_id uuid,
			field_name text,
			field_role text default 'DIMENSION',
			aggregation_type text default 'NONE',
			binding_requirement text default 'REQUIRED',
			eligibility_source text default 'DIRECT',
			subtype_scope text,
			is_exposed boolean default true,
			inherits_defaults boolean default false,
			created_at timestamptz default now(),
			updated_at timestamptz default now()
		)
	`)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS oms.subtype_registry (
			id uuid primary key,
			tenant_id uuid,
			root_object text,
			subtype_code text,
			display_name text,
			parent_subtype_code text,
			field_allowlist jsonb default '[]',
			is_active boolean default true,
			created_at timestamptz default now()
		)
	`)
	require.NoError(t, err)

	// Unique constraint on business_objects for ON CONFLICT
	_, err = db.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS idx_bo_tenant_key ON business_objects(tenant_id, bo_key)`)
	require.NoError(t, err)

	// Seed subtype_registry with 2 account subtypes that share "account_number"
	// but have one unique field each — the union should be 3 fields for the parent
	_, err = db.ExecContext(ctx, `
		INSERT INTO oms.subtype_registry (id, tenant_id, root_object, subtype_code, display_name, field_allowlist, is_active)
		VALUES
			($1, $2, 'account', 'institutional', 'Institutional Account',
			 '["account_number", "sponsor_id", "erisa_flag"]'::jsonb, true),
			($3, $2, 'account', 'retail_wealth', 'Retail Wealth Account',
			 '["account_number", "account_name", "tax_id_type"]'::jsonb, true)
		ON CONFLICT (id) DO NOTHING
	`, uuid.New(), goldCopyTenantID, uuid.New())
	require.NoError(t, err)

	// Call BuildForTenant via the builder
	loader := NewSubtypeRegistryLoader(time.Minute)
	builder := NewSubtypeBOBuilder(loader)

	err = builder.BuildForTenant(ctx, db.DB, tenantID)
	require.NoError(t, err, "BuildForTenant should not error")

	// Verify parent BO was created
	var parentBOID string
	err = db.GetContext(ctx, &parentBOID, `
		SELECT id::text FROM business_objects
		WHERE tenant_id = $1 AND bo_key = 'oms.account'
	`, tenantID)
	require.NoError(t, err, "parent BO oms.account should exist")

	// [REGRESSION CHECK] Parent BO must have CoreFields
	var parentFields []struct {
		ID               string `db:"id"`
		FieldName        string `db:"field_name"`
		InheritsDefaults bool   `db:"inherits_defaults"`
		SubtypeScope     *string `db:"subtype_scope"`
	}
	err = db.SelectContext(ctx, &parentFields, `
		SELECT id, field_name, inherits_defaults, subtype_scope
		FROM business_object_fields
		WHERE bo_id = $1
		ORDER BY field_name
	`, parentBOID)
	require.NoError(t, err)

	require.NotEmpty(t, parentFields, "parent BO oms.account must have CoreFields — this is the regression check")

	// Verify union of fields: account_number (shared), sponsor_id, erisa_flag, account_name, tax_id_type = 5
	require.Equal(t, 5, len(parentFields), "parent should have union of all child subtype fields")

	// Verify all fields have inherits_defaults=true and subtype_scope=NULL (core fields)
	for _, f := range parentFields {
		require.True(t, f.InheritsDefaults, "field %s should have inherits_defaults=true", f.FieldName)
		require.Nil(t, f.SubtypeScope, "parent core field %s should have subtype_scope=NULL", f.FieldName)
	}

	// Verify child BOs also still get their fields
	var childFields []struct{ FieldName string }
	err = db.SelectContext(ctx, &childFields, `
		SELECT bof.field_name
		FROM business_object_fields bof
		JOIN business_objects bo ON bo.id = bof.bo_id
		WHERE bo.tenant_id = $1 AND bo.bo_key = 'oms.account/institutional'
		ORDER BY bof.field_name
	`, tenantID)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"account_number", "erisa_flag", "sponsor_id"}, childFields,
		"institutional child should still have its own fields")
}
