package metadata

import (
	"context"
	"database/sql"
	"sync"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResolveCatalogNodeTypeID_ResolvesAndCaches verifies the tenant-scoped
// lookup replacing the former hardcoded UUID: it must query by name, key the
// cache per tenant (not globally — catalog_node_types.id for
// catalog_type_name='business_object' is confirmed to differ per tenant on
// live data), and only hit the DB once per tenant.
func TestResolveCatalogNodeTypeID_ResolvesAndCaches(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	tenantID := uuid.New().String()
	expected := uuid.New()

	mock.ExpectQuery("SELECT id FROM catalog_node_types").
		WithArgs(tenantID).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(expected.String()))

	s := &BusinessObjectService{db: sqlxDB}
	catalogNodeTypeIDCache = sync.Map{} // reset shared cache between tests

	got, err := s.resolveCatalogNodeTypeID(context.Background(), tenantID)
	require.NoError(t, err)
	assert.Equal(t, expected, got)

	// Second call for the same tenant must not hit the DB again — the mock
	// would fail the test if an unexpected query fired here.
	got2, err := s.resolveCatalogNodeTypeID(context.Background(), tenantID)
	require.NoError(t, err)
	assert.Equal(t, expected, got2)

	require.NoError(t, mock.ExpectationsWereMet())
}

// TestResolveCatalogNodeTypeID_FailsLoudlyWhenMissing is the load-bearing
// test for this fix: on a fresh environment where catalog_node_types has no
// 'business_object' row for the tenant, this must return an actionable
// error, never a silent fallback to a guessed constant — that fallback is
// exactly the portability bug being replaced.
func TestResolveCatalogNodeTypeID_FailsLoudlyWhenMissing(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	tenantID := uuid.New().String()

	mock.ExpectQuery("SELECT id FROM catalog_node_types").
		WithArgs(tenantID).
		WillReturnError(sql.ErrNoRows)

	s := &BusinessObjectService{db: sqlxDB}
	catalogNodeTypeIDCache = sync.Map{}

	_, err = s.resolveCatalogNodeTypeID(context.Background(), tenantID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "catalog_node_types")
	assert.Contains(t, err.Error(), tenantID)

	require.NoError(t, mock.ExpectationsWereMet())
}
