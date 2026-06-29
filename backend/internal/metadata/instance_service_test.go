package metadata

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestCreateInstanceTx_InsertsIntoBoInstances(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	xdb := sqlx.NewDb(db, "postgres")
	svc := NewBusinessObjectService(xdb, nil, nil, nil)

	ctx := context.Background()
	mock.ExpectBegin()
	tx, err := db.Begin()
	require.NoError(t, err)
	defer tx.Rollback()

	tenantID := uuid.NewString()
	userID := "admin@example.com"
	instance := &models.BusinessObjectInstance{
		BusinessObjectKey: "Customer",
		BusinessObjectID:  uuid.NewString(),
		DatasourceID:      uuid.NewString(),
		CoreFieldValues:   map[string]any{"name": "Acme"},
	}

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO bo_instances")).
		WithArgs(
			sqlmock.AnyArg(), // id
			tenantID,
			instance.BusinessObjectID,
			instance.BusinessObjectKey,
			instance.DatasourceID,
			sql.NullString{},
			"",
			sqlmock.AnyArg(), // core_json
			sqlmock.AnyArg(), // custom_json
			sqlmock.AnyArg(), // created_at
			userID,
			sqlmock.AnyArg(), // last_modified_at
			userID,
			false,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	created, err := svc.CreateInstanceTx(ctx, tx, tenantID, userID, instance)
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)
	require.Equal(t, tenantID, created.TenantID)
	require.Equal(t, userID, created.CreatedBy)
}

func TestUpdateInstanceTx_UpdatesBoInstances(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	xdb := sqlx.NewDb(db, "postgres")
	svc := NewBusinessObjectService(xdb, nil, nil, nil)

	ctx := context.Background()
	mock.ExpectBegin()
	tx, err := db.Begin()
	require.NoError(t, err)
	defer tx.Rollback()

	tenantID := uuid.NewString()
	userID := "admin@example.com"
	instanceID := uuid.NewString()
	boKey := "Customer"

	coreJSON := []byte(`{"name":"Acme"}`)
	customJSON := []byte(`{}`)

	// Expect the read of the existing instance.
	rows := sqlmock.NewRows([]string{
		"id", "tenant_id", "business_object_id", "business_object_key", "datasource_id",
		"subtype_id", "subtype_key", "core_field_values", "custom_field_values",
		"created_at", "created_by", "last_modified_at", "last_modified_by", "is_deleted", "deleted_at",
	}).AddRow(
		instanceID, tenantID, uuid.NewString(), boKey, uuid.NewString(),
		sql.NullString{}, "", coreJSON, customJSON,
		time.Now(), userID, time.Now(), userID, false, sql.NullTime{},
	)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, tenant_id, business_object_id, business_object_key, datasource_id")).
		WithArgs(instanceID, tenantID).
		WillReturnRows(rows)

	// Expect the update.
	mock.ExpectExec(regexp.QuoteMeta("UPDATE bo_instances")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), userID, instanceID, tenantID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	updated, err := svc.UpdateInstanceTx(ctx, tx, tenantID, instanceID, userID,
		map[string]any{"name": "Acme Inc"},
		nil,
	)
	require.NoError(t, err)
	require.Equal(t, instanceID, updated.ID)
	require.Equal(t, "Acme Inc", updated.CoreFieldValues["name"])
}

func TestDeleteInstanceTx_SoftDeletesInstance(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	xdb := sqlx.NewDb(db, "postgres")
	svc := NewBusinessObjectService(xdb, nil, nil, nil)

	ctx := context.Background()
	mock.ExpectBegin()
	tx, err := db.Begin()
	require.NoError(t, err)
	defer tx.Rollback()

	tenantID := uuid.NewString()
	userID := "admin@example.com"
	instanceID := uuid.NewString()

	mock.ExpectExec(regexp.QuoteMeta("UPDATE bo_instances")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), userID, instanceID, tenantID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = svc.DeleteInstanceTx(ctx, tx, tenantID, instanceID, userID)
	require.NoError(t, err)
}

func TestDeleteInstanceTx_ReturnsErrorWhenNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	xdb := sqlx.NewDb(db, "postgres")
	svc := NewBusinessObjectService(xdb, nil, nil, nil)

	ctx := context.Background()
	mock.ExpectBegin()
	tx, err := db.Begin()
	require.NoError(t, err)
	defer tx.Rollback()

	tenantID := uuid.NewString()
	userID := "admin@example.com"
	instanceID := uuid.NewString()

	mock.ExpectExec(regexp.QuoteMeta("UPDATE bo_instances")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), userID, instanceID, tenantID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = svc.DeleteInstanceTx(ctx, tx, tenantID, instanceID, userID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

// allowNullTimeScan makes sqlmock accept sql.NullTime as a driver value.
func allowNullTimeScan(val sql.NullTime) driver.Value {
	if val.Valid {
		return val.Time
	}
	return nil
}
