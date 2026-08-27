package security

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfileService_Unit(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	svc := &ProfileService{
		db: sqlx.NewDb(mockDB, "postgres"),
	}
	ctx := context.Background()
	tenantID := uuid.New()
	profileID := uuid.New()

	t.Run("FetchEffectiveProfile_Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"profile_id", "tenant_id", "parent_profile_id"}).
			AddRow(profileID, nil, nil)

		mock.ExpectQuery(`SELECT profile_id, tenant_id, parent_profile_id FROM security\.security_profiles`).
			WithArgs("northwind_sales_rep", tenantID).
			WillReturnRows(rows)

		res, err := svc.FetchEffectiveProfile(ctx, tenantID, "northwind_sales_rep")
		require.NoError(t, err)
		assert.Equal(t, "northwind_sales_rep", res.ProfileKey)
		assert.False(t, res.IsCustomized)
	})

	t.Run("FetchEffectiveProfile_NotFound", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"profile_id", "tenant_id", "parent_profile_id"})

		mock.ExpectQuery(`SELECT profile_id, tenant_id, parent_profile_id FROM security\.security_profiles`).
			WithArgs("non_existent", tenantID).
			WillReturnRows(rows)

		_, err := svc.FetchEffectiveProfile(ctx, tenantID, "non_existent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist in platform blueprint")
	})

	t.Run("EnrichSubjectAttributes_EmptyGroups", func(t *testing.T) {
		role, clearance, err := svc.EnrichSubjectAttributes(ctx, tenantID, "user-1", []string{})
		require.NoError(t, err)
		assert.Equal(t, "standard_guest", role)
		assert.Equal(t, "L1", clearance)
	})

	t.Run("EnrichSubjectAttributes_MatchFound", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"functional_role", "clearance_level"}).
			AddRow("compliance_officer", "L3")

		mock.ExpectQuery(`SELECT functional_role, clearance_level FROM security\.identity_profile_mappings`).
			WithArgs(tenantID, `{"GG-Compliance"}`).
			WillReturnRows(rows)

		role, clearance, err := svc.EnrichSubjectAttributes(ctx, tenantID, "user-1", []string{"GG-Compliance"})
		require.NoError(t, err)
		assert.Equal(t, "compliance_officer", role)
		assert.Equal(t, "L3", clearance)
	})

	t.Run("CreateProfile_Success", func(t *testing.T) {
		newProfile := &SecurityProfile{
			ProfileKey:  "custom_analyst",
			ProfileName: "Custom Analyst",
			TenantID:    &tenantID,
		}

		rows := sqlmock.NewRows([]string{"profile_id", "tenant_id", "profile_key", "profile_name", "parent_profile_id", "created_at", "updated_at"}).
			AddRow(profileID, &tenantID, "custom_analyst", "Custom Analyst", nil, time.Now(), time.Now())

		mock.ExpectQuery(`INSERT INTO security\.security_profiles`).
			WillReturnRows(rows)

		created, err := svc.CreateProfile(ctx, newProfile)
		require.NoError(t, err)
		assert.Equal(t, "custom_analyst", created.ProfileKey)
	})

	t.Run("CreateMapping_Success", func(t *testing.T) {
		mappingID := uuid.New()
		newMapping := &IdentityProfileMapping{
			TenantID:       tenantID,
			IDPGroupClaim:  "GG-Trading",
			FunctionalRole: "trader",
			ClearanceLevel: "L2",
		}

		rows := sqlmock.NewRows([]string{"mapping_id", "tenant_id", "idp_group_claim", "functional_role", "clearance_level", "created_at"}).
			AddRow(mappingID, tenantID, "GG-Trading", "trader", "L2", time.Now())

		mock.ExpectQuery(`INSERT INTO security\.identity_profile_mappings`).
			WillReturnRows(rows)

		created, err := svc.CreateMapping(ctx, newMapping)
		require.NoError(t, err)
		assert.Equal(t, "trader", created.FunctionalRole)
	})
}
