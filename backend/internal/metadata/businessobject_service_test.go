package metadata

import (
	"context"
	"database/sql/driver"
	"fmt"
	"regexp"
	"testing"

	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func expectBoFieldsDisplayNameColumn(mock sqlmock.Sqlmock, schema string, exists bool) {
	query := "SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = $1 AND column_name = $2"
	args := []driver.Value{"bo_fields", "display_name"}
	if schema != "" {
		query += " AND table_schema = $3"
		args = append(args, schema)
	}
	query += ")"

	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(args...).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(exists))
}

func TestLoadBOSubtypesAndFields_PopulatesSubtypes(t *testing.T) {
	// Setup sqlmock
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	xdb := sqlx.NewDb(db, "postgres")
	svc := NewBusinessObjectService(xdb, nil, nil, nil)

	ctx := context.Background()
	bo := &models.BusinessObjectDefinition{
		ID:       uuid.NewString(),
		TenantID: uuid.NewString(),
		Key:      "parent_key",
	}

	childID := uuid.NewString()
	childKey := "child_key"
	// Expect query for child BOs
	childRows := sqlmock.NewRows([]string{"id", "key", "name", "display_name", "technical_name", "description", "is_core"}).
		AddRow(childID, childKey, "Child", "Child", "child", "desc", false)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, key, name, display_name, COALESCE(technical_name, '') AS technical_name, \n\t\t       COALESCE(description, '') AS description, is_core\n\t\tFROM business_objects\n\t\tWHERE parent_id = $1::uuid AND tenant_id = $2::uuid\n\t\tORDER BY name")).
		WithArgs(bo.ID, bo.TenantID).
		WillReturnRows(childRows)

	// Expect fields query for the child
	fieldRows := sqlmock.NewRows([]string{"id", "key", "name", "display_name", "technical_name", "type", "is_core", "is_required", "is_system", "description", "reference_entity", "sequence", "created_at", "created_by", "last_modified_at", "last_modified_by"}).
		AddRow(uuid.NewString(), "f1", "Field1", "Field 1", "field1", "string", true, false, false, "", "", 1, time.Now(), "", time.Now(), "")

	expectBoFieldsDisplayNameColumn(mock, "ag_catalog", true)
	mock.ExpectQuery("SELECT .* FROM .*bo_fields .*").
		WithArgs(childID, bo.TenantID).
		WillReturnRows(fieldRows)

	// Expect legacy subtypes query (return none)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, key, name, display_name, COALESCE(technical_name, '') AS technical_name, \n\t\t       COALESCE(description, '') AS description, is_core, based_on_entity, \n\t\t       COALESCE(clone_parent_key, '') AS clone_parent_key, sequence, created_at, \n\t\t       COALESCE(created_by, '') AS created_by, last_modified_at, COALESCE(last_modified_by, '') AS last_modified_by\n\t\tFROM bo_subtypes\n\t\tWHERE business_object_id = $1::uuid\n\t\tORDER BY sequence")).
		WithArgs(bo.ID).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	// Expect entity-level fields query (return none)
	expectBoFieldsDisplayNameColumn(mock, "", true)
	mock.ExpectQuery("SELECT .* FROM .*bo_fields .*").
		WithArgs(bo.ID, bo.TenantID).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	// Expect entity-level fields query (old schema fallback)
	mock.ExpectQuery("SELECT .* FROM bo_fields .*").
		WithArgs(bo.ID).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	// Run
	err = svc.loadBOSubtypesAndFields(ctx, bo, bo.TenantID)
	require.NoError(t, err)

	// Verify
	require.Equal(t, 1, len(bo.Subtypes))
	sd, ok := bo.Subtypes[childKey]
	require.True(t, ok)
	require.Equal(t, childID, sd.ID)

	// Ensure all expectations were met
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLoadBOSubtypesAndFields_FallbackOldSchema(t *testing.T) {
	// Setup sqlmock
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	xdb := sqlx.NewDb(db, "postgres")
	svc := NewBusinessObjectService(xdb, nil, nil, nil)

	ctx := context.Background()
	bo := &models.BusinessObjectDefinition{
		ID:       uuid.NewString(),
		TenantID: uuid.NewString(),
		Key:      "parent_key",
	}

	childID := uuid.NewString()
	childKey := "child_key"
	// Expect query for child BOs
	childRows := sqlmock.NewRows([]string{"id", "key", "name", "display_name", "technical_name", "description", "is_core"}).
		AddRow(childID, childKey, "Child", "Child", "child", "desc", false)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, key, name, display_name, COALESCE(technical_name, '') AS technical_name, \n\t\t       COALESCE(description, '') AS description, is_core\n\t\tFROM business_objects\n\t\tWHERE parent_id = $1::uuid AND tenant_id = $2::uuid\n\t\tORDER BY name")).
		WithArgs(bo.ID, bo.TenantID).
		WillReturnRows(childRows)

	// Expect fields query for the child to fail (new schema not present)
	expectBoFieldsDisplayNameColumn(mock, "ag_catalog", true)
	mock.ExpectQuery("SELECT .* FROM .*bo_fields .*").WithArgs(childID, bo.TenantID).WillReturnError(fmt.Errorf("pq: column \"key\" does not exist"))

	// Expect old schema query to succeed
	oldRows := sqlmock.NewRows([]string{"id", "bo_id", "field_name", "display_label", "field_type", "is_required", "is_readonly", "is_searchable", "is_sortable", "display_order"}).
		AddRow(uuid.NewString(), childID, "f1", "Field 1", "string", false, false, true, true, 1)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, bo_id, field_name, display_label, field_type, is_required, is_readonly, is_searchable, is_sortable, display_order\n\t\t\tFROM bo_fields\n\t\t\tWHERE bo_id = $1\n\t\t\tORDER BY display_order")).
		WithArgs(childID).
		WillReturnRows(oldRows)

	// Expect legacy subtypes query (return none)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, key, name, display_name, COALESCE(technical_name, '') AS technical_name, \n\t\t       COALESCE(description, '') AS description, is_core, based_on_entity, \n\t\t       COALESCE(clone_parent_key, '') AS clone_parent_key, sequence, created_at, \n\t\t       COALESCE(created_by, '') AS created_by, last_modified_at, COALESCE(last_modified_by, '') AS last_modified_by\n\t\tFROM bo_subtypes\n\t\tWHERE business_object_id = $1::uuid\n\t\tORDER BY sequence")).
		WithArgs(bo.ID).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	// Expect entity-level fields query (return none)
	expectBoFieldsDisplayNameColumn(mock, "", true)
	mock.ExpectQuery("SELECT .* FROM .*bo_fields .*").
		WithArgs(bo.ID, bo.TenantID).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	// Expect entity-level fields query (old schema fallback)
	mock.ExpectQuery("SELECT .* FROM bo_fields .*").
		WithArgs(bo.ID).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	// Run
	err = svc.loadBOSubtypesAndFields(ctx, bo, bo.TenantID)
	require.NoError(t, err)

	// Verify
	require.Equal(t, 1, len(bo.Subtypes))
	sd, ok := bo.Subtypes[childKey]
	require.True(t, ok)
	require.Equal(t, childID, sd.ID)
	require.Equal(t, 1, len(sd.SubtypeFields))
	require.Equal(t, "f1", sd.SubtypeFields[0].Key)

	// Ensure all expectations were met
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLoadBOSubtypesAndFields_LoadsEntityFieldsFromNormalizedBoFields(t *testing.T) {
	// Setup sqlmock
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	xdb := sqlx.NewDb(db, "postgres")
	svc := NewBusinessObjectService(xdb, nil, nil, nil)

	ctx := context.Background()
	bo := &models.BusinessObjectDefinition{
		ID:       uuid.NewString(),
		TenantID: uuid.NewString(),
		Key:      "customer",
	}

	// Expect no child BOs
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, key, name, display_name, COALESCE(technical_name, '') AS technical_name, \n\t\t\t       COALESCE(description, '') AS description, is_core\n\t\tFROM business_objects\n\t\tWHERE parent_id = $1::uuid AND tenant_id = $2::uuid\n\t\tORDER BY name")).
		WithArgs(bo.ID, bo.TenantID).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	// Expect legacy subtypes empty
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, key, name, display_name, COALESCE(technical_name, '') AS technical_name, \n\t\t\t       COALESCE(description, '') AS description, is_core, based_on_entity, \n\t\t\t       COALESCE(clone_parent_key, '') AS clone_parent_key, sequence, created_at, \n\t\t\t       COALESCE(created_by, '') AS created_by, last_modified_at, COALESCE(last_modified_by, '') AS last_modified_by\n\t\tFROM bo_subtypes\n\t\tWHERE business_object_id = $1::uuid\n\t\tORDER BY sequence")).
		WithArgs(bo.ID).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	// Expect entity-level fields (normalized) to return one row
	fieldRows := sqlmock.NewRows([]string{"id", "key", "name", "display_name", "technical_name", "type", "is_core", "is_required", "is_system", "description", "reference_entity", "sequence", "created_at", "created_by", "last_modified_at", "last_modified_by", "semantic_term_id"}).
		AddRow(uuid.NewString(), "cust_id", "Customer ID", "Customer ID", "customer_id", "string", false, false, false, "", "", 1, time.Now(), "", time.Now(), "", uuid.NewString())

	expectBoFieldsDisplayNameColumn(mock, "", true)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, key, name, display_name, COALESCE(technical_name, '') AS technical_name, type,\n\t\t       COALESCE(is_core, false) AS is_core, COALESCE(is_required, false) AS is_required,\n\t\t       COALESCE(is_system, false) AS is_system, COALESCE(description, '') AS description,\n\t\t       COALESCE(reference_entity, '') AS reference_entity, COALESCE(sequence, 0) AS sequence,\n\t\t       created_at, COALESCE(CAST(created_by AS text), '') AS created_by, last_modified_at, \n\t\t       COALESCE(CAST(last_modified_by AS text), '') AS last_modified_by,\n\t\t\t   COALESCE(CAST(semantic_term_id AS text), '') AS semantic_term_id\n\t\tFROM bo_fields\n\t\tWHERE business_object_id::text = $1 AND tenant_id::text = $2 AND subtype_id IS NULL\n\t\tORDER BY sequence")).
		WithArgs(bo.ID, bo.TenantID).
		WillReturnRows(fieldRows)

	// Run
	err = svc.loadBOSubtypesAndFields(ctx, bo, bo.TenantID)
	require.NoError(t, err)

	// Verify entity fields loaded into CustomFields since not core
	require.Equal(t, 1, len(bo.CustomFields))
	require.Equal(t, "cust_id", bo.CustomFields[0].Key)

	// Ensure expectations met
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteSubtype_LookupById(t *testing.T) {
	// Setup sqlmock
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Allow the expectations for this test to match in any order. DeleteSubtype
	// performs multiple independent queries and updates; enforcing strict order
	// makes the test brittle.
	mock.MatchExpectationsInOrder(false)

	xdb := sqlx.NewDb(db, "postgres")
	svc := NewBusinessObjectService(xdb, nil, nil, nil)

	ctx := context.Background()
	tenantID := uuid.NewString()
	parentID := uuid.NewString()
	subtypeID := uuid.NewString()
	subtypeKey := "child_key"

	// Mock GetBusinessObject
	boJSON := fmt.Sprintf(`{"subtypes": {"%s": {"id": "%s", "key": "%s", "displayName": "Child"}}}`, subtypeKey, subtypeID, subtypeKey)
	parentRows := sqlmock.NewRows([]string{"id", "tenant_id", "key", "name", "display_name", "technical_name", "description", "icon", "is_core", "clones_from", "clone_parent_key", "clone_parent_display_name", "category", "parent_id", "instance_count", "created_at", "created_by", "last_modified_at", "last_modified_by", "is_active", "config"}).
		AddRow(parentID, tenantID, "parent", "Parent", "Parent", "parent", "", "", false, "", "", "", "", "", 0, time.Now(), "", time.Now(), "", true, []byte(boJSON))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, tenant_id, key, name, display_name, COALESCE(technical_name, '') AS technical_name")).
		WithArgs(tenantID, parentID, true).
		WillReturnRows(parentRows)

	// Mock loading sub-resources (multi-strategy)

	// 1. Child BOs query
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, key, name, display_name, COALESCE(technical_name, '') AS technical_name")).
		WithArgs(parentID, tenantID).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	// 2. Fallback child BOs query (if first fails or returns 0)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, key, name, display_name, COALESCE(technical_name, '') AS technical_name")).
		WithArgs(parentID).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	// 3. Legacy subtypes query
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, key, name, display_name, COALESCE(technical_name, '') AS technical_name")).
		WithArgs(parentID).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	// 4. Entity-level fields query
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, key, name, display_name, COALESCE(technical_name, '') AS technical_name")).
		WithArgs(parentID).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	// Expect UPDATE
	mock.ExpectExec(regexp.QuoteMeta("UPDATE business_objects")).
		WithArgs("{}", sqlmock.AnyArg(), "user", parentID, tenantID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Expect DELETE from bo_subtypes
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM bo_subtypes")).
		WithArgs(subtypeID, parentID).
		WillReturnResult(sqlmock.NewResult(1, 0))

	// Expect DELETE from business_objects
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM business_objects")).
		WithArgs(subtypeID, subtypeID, parentID).
		WillReturnResult(sqlmock.NewResult(1, 0))

	// 5. Final GetBusinessObject to return updated state
	parentRowsFinal := sqlmock.NewRows([]string{"id", "tenant_id", "key", "name", "display_name", "technical_name", "description", "icon", "is_core", "clones_from", "clone_parent_key", "clone_parent_display_name", "category", "parent_id", "instance_count", "created_at", "created_by", "last_modified_at", "last_modified_by", "is_active", "config"}).
		AddRow(parentID, tenantID, "parent", "Parent", "Parent", "parent", "", "", false, "", "", "", "", "", 0, time.Now(), "", time.Now(), "", true, []byte("{}"))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, tenant_id, key, name, display_name, COALESCE(technical_name, '') AS technical_name")).
		WithArgs(tenantID, parentID, true).
		WillReturnRows(parentRowsFinal)

	// Mock loading sub-resources for the final Get
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, key, name")).WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, key, name")).WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, key, name")).WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, key, name")).WillReturnRows(sqlmock.NewRows([]string{"id"}))

	// Run
	secCtx := &security.Context{TenantID: tenantID}
	_, err = svc.DeleteSubtype(ctx, secCtx, parentID, subtypeID, "user")
	require.NoError(t, err)

	// Ensure all expectations were met
	require.NoError(t, mock.ExpectationsWereMet())
}
