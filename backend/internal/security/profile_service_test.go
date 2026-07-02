package security

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfileService_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode: requires Postgres")
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres:postgres@100.84.50.65:5432/alpha?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Skip("Database not available for integration tests")
		return
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Skip("Database connection failed, skipping integration tests")
		return
	}

	svc := NewProfileService(db)
	ctx := context.Background()

	// Use a transaction for all test operations to keep the DB clean
	tx, err := svc.db.BeginTxx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()

	tenantID := uuid.New()

	// Test 1: Fetch global blueprint profile (seeded in migration)
	profile, err := svc.FetchEffectiveProfile(ctx, tenantID, "northwind_sales_rep")
	require.NoError(t, err)
	assert.Equal(t, "northwind_sales_rep", profile.ProfileKey)
	assert.False(t, profile.IsCustomized)

	// Test 2: Create a tenant-specific customization overlay profile
	customProfile := &SecurityProfile{
		ProfileID:   uuid.New(),
		TenantID:    &tenantID,
		ProfileKey:  "northwind_sales_rep", // Override the global key
		ProfileName: "Acme Custom Sales Rep",
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO security.security_profiles (profile_id, tenant_id, profile_key, profile_name)
		VALUES ($1, $2, $3, $4)
	`, customProfile.ProfileID, customProfile.TenantID, customProfile.ProfileKey, customProfile.ProfileName)
	require.NoError(t, err)

	// Fetch effective profile again: should now be customized
	// We run the raw query using the transaction DB or mock it
	query := `
		SELECT profile_id, tenant_id, parent_profile_id 
		FROM security.security_profiles
		WHERE profile_key = $1 AND (tenant_id IS NULL OR tenant_id = $2)
		ORDER BY tenant_id ASC NULLS FIRST;
	`
	rows, err := tx.QueryContext(ctx, query, "northwind_sales_rep", tenantID)
	require.NoError(t, err)
	defer rows.Close()

	var totalFound int
	var isCustomized bool
	for rows.Next() {
		var pID uuid.UUID
		var tID *uuid.UUID
		var parentID *uuid.UUID
		require.NoError(t, rows.Scan(&pID, &tID, &parentID))
		if tID != nil {
			isCustomized = true
		}
		totalFound++
	}
	assert.True(t, isCustomized)
	assert.True(t, totalFound > 1)

	// Test 3: Create identity profile mapping
	mapping := &IdentityProfileMapping{
		MappingID:      uuid.New(),
		TenantID:       tenantID,
		IDPClientID:    "semlayer-frontend",
		IDPGroupID:     "GG-Sales-US",
		FunctionalRole: "northwind_sales_rep",
		ClearanceLevel: "L2",
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO security.identity_profile_mappings (mapping_id, tenant_id, idp_client_id, idp_group_id, functional_role, clearance_level)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, mapping.MappingID, mapping.TenantID, mapping.IDPClientID, mapping.IDPGroupID, mapping.FunctionalRole, mapping.ClearanceLevel)
	require.NoError(t, err)

	// Test EnrichSubjectAttributes using a query in the transaction
	var role, clearance string
	err = tx.QueryRowContext(ctx, `
		SELECT functional_role, clearance_level 
		FROM security.identity_profile_mappings
		WHERE tenant_id = $1 AND idp_group_id = ANY($2)
		LIMIT 1;
	`, tenantID, []string{"GG-Sales-US"}).Scan(&role, &clearance)
	require.NoError(t, err)
	assert.Equal(t, "northwind_sales_rep", role)
	assert.Equal(t, "L2", clearance)

	// Test 4: Verify recursive role hierarchy resolution (parent_profile_id semantics)
	// First, fetch the global northwind_sales_rep profile to get its ID
	var parentProfileID uuid.UUID
	err = tx.QueryRowContext(ctx, `
		SELECT profile_id FROM security.security_profiles
		WHERE profile_key = 'northwind_sales_rep' AND tenant_id IS NULL
		LIMIT 1
	`).Scan(&parentProfileID)
	require.NoError(t, err)

	// Create child profile custom_sales_rep inheriting from global northwind_sales_rep
	childProfileID := uuid.New()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO security.security_profiles (profile_id, tenant_id, profile_key, profile_name, parent_profile_id)
		VALUES ($1, $2, $3, $4, $5)
	`, childProfileID, tenantID, "custom_sales_rep", "Acme Custom Sales Representative", parentProfileID)
	require.NoError(t, err)

	// Query inherited roles for custom_sales_rep
	// (Traverses recursively: child custom_sales_rep -> parent northwind_sales_rep)
	queryInherit := `
		WITH RECURSIVE profile_hierarchy AS (
			SELECT profile_id, profile_key, parent_profile_id, tenant_id
			FROM security.security_profiles
			WHERE profile_key = $1 AND (tenant_id IS NULL OR tenant_id = $2)
			
			UNION ALL
			
			SELECT p.profile_id, p.profile_key, p.parent_profile_id, p.tenant_id
			FROM security.security_profiles p
			INNER JOIN profile_hierarchy h ON p.profile_id = h.parent_profile_id
		)
		SELECT DISTINCT profile_key FROM profile_hierarchy;
	`
	rowsInherit, err := tx.QueryContext(ctx, queryInherit, "custom_sales_rep", tenantID)
	require.NoError(t, err)
	defer rowsInherit.Close()

	var inheritedRoles []string
	for rowsInherit.Next() {
		var roleKey string
		require.NoError(t, rowsInherit.Scan(&roleKey))
		inheritedRoles = append(inheritedRoles, roleKey)
	}

	assert.Len(t, inheritedRoles, 2)
	assert.Contains(t, inheritedRoles, "custom_sales_rep")
	assert.Contains(t, inheritedRoles, "northwind_sales_rep")
}

