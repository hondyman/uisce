package metadata

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

var tenantColCheck = regexp.QuoteMeta("SELECT 1 FROM information_schema.columns")

func newTenantScopeMock(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return sqlx.NewDb(db, "postgres"), mock
}

func TestTenantScopePredicate_TenantedTableScopesToCaller(t *testing.T) {
	xdb, mock := newTenantScopeMock(t)
	secCtx := &security.Context{TenantID: "tenant-a"}

	mock.ExpectQuery(tenantColCheck).WithArgs("crm", "accounts").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	pred, args, err := tenantScopePredicate(context.Background(), xdb, "crm.accounts", secCtx, 1)
	require.NoError(t, err)
	require.Equal(t, `"tenant_id" = $1`, pred)
	require.Equal(t, []interface{}{"tenant-a"}, args)

	// The predicate and its bound arg must reach both the COUNT and the page
	// query exactly as QueryBORecords composes them.
	whereSQL := strings.Join([]string{"1=1", pred}, " AND ")
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM "crm"."accounts" WHERE 1=1 AND "tenant_id" = $1`)).
		WithArgs("tenant-a").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "crm"."accounts" WHERE 1=1 AND "tenant_id" = $1`)).
		WithArgs("tenant-a").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("r1"))

	var total int
	require.NoError(t, xdb.Get(&total, `SELECT COUNT(*) FROM "crm"."accounts" WHERE `+whereSQL, args...))
	rows, err := xdb.Queryx(`SELECT * FROM "crm"."accounts" WHERE `+whereSQL+"  LIMIT 50 OFFSET 0", args...)
	require.NoError(t, err)
	rows.Close()
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTenantScopePredicate_UsesGivenArgIndex(t *testing.T) {
	xdb, mock := newTenantScopeMock(t)
	mock.ExpectQuery(tenantColCheck).WithArgs("public", "orders").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	pred, _, err := tenantScopePredicate(context.Background(), xdb, "orders", &security.Context{TenantID: "t"}, 3)
	require.NoError(t, err)
	require.Equal(t, `"tenant_id" = $3`, pred)
}

func TestTenantScopePredicate_UntenantedTableNoPredicate(t *testing.T) {
	xdb, mock := newTenantScopeMock(t)
	mock.ExpectQuery(tenantColCheck).WithArgs("public", "orders").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	pred, args, err := tenantScopePredicate(context.Background(), xdb, "orders", &security.Context{TenantID: "t"}, 1)
	require.NoError(t, err)
	require.Empty(t, pred)
	require.Empty(t, args)
}

func TestTenantScopePredicate_FailsClosed(t *testing.T) {
	t.Run("introspection error", func(t *testing.T) {
		xdb, mock := newTenantScopeMock(t)
		mock.ExpectQuery(tenantColCheck).WillReturnError(errors.New("boom"))
		_, _, err := tenantScopePredicate(context.Background(), xdb, "orders", &security.Context{TenantID: "t"}, 1)
		require.Error(t, err)
	})
	t.Run("missing tenant on tenanted table", func(t *testing.T) {
		xdb, mock := newTenantScopeMock(t)
		mock.ExpectQuery(tenantColCheck).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		_, _, err := tenantScopePredicate(context.Background(), xdb, "orders", &security.Context{}, 1)
		require.Error(t, err)
	})
}
