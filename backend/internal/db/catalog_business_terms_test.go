package db

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

func TestUpsertBusinessTermsFromGold_SkipsForGoldTenant(t *testing.T) {
	dbSQL, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer dbSQL.Close()
	dbx := sqlx.NewDb(dbSQL, "sqlmock")

	tenantID := uuid.New()
	datasourceID := uuid.New()
	btType := uuid.New()

	// Expect the gold_copy check to return true (gold tenant)
	mock.ExpectQuery(`SELECT\s+gold_copy\s+FROM\s+public\.tenants\s+WHERE\s+id\s*=\s*\$1`).
		WithArgs(tenantID).
		WillReturnRows(sqlmock.NewRows([]string{"gold_copy"}).AddRow(true))

	count, err := UpsertBusinessTermsFromGold(context.Background(), dbx, tenantID, datasourceID, btType)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertBusinessTermsFromGold_UpsertsCount(t *testing.T) {
	dbSQL, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer dbSQL.Close()
	dbx := sqlx.NewDb(dbSQL, "sqlmock")

	tenantID := uuid.New()
	datasourceID := uuid.New()
	btType := uuid.New()

	// Non-gold tenant
	mock.ExpectQuery(`SELECT\s+gold_copy\s+FROM\s+public\.tenants\s+WHERE\s+id\s*=\s*\$1`).
		WithArgs(tenantID).
		WillReturnRows(sqlmock.NewRows([]string{"gold_copy"}).AddRow(false))

		// Upsert query returns count 3
	mock.ExpectQuery(`WITH\s+gold_terms[\s\S]*SELECT\s+COUNT\(\*\)\s+FROM\s+upserted;?`).
		WithArgs(btType, tenantID, datasourceID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	count, err := UpsertBusinessTermsFromGold(context.Background(), dbx, tenantID, datasourceID, btType)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), count)
	assert.NoError(t, mock.ExpectationsWereMet())
}
