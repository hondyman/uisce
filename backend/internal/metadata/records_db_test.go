package metadata

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestRecordsDBStrict(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	s := &BusinessObjectService{db: sqlx.NewDb(db, "sqlmock")}
	ctx := context.Background()

	// No bound backend: the records live in the metadata DB.
	mock.ExpectQuery("FROM public.business_object_binding").WillReturnError(sql.ErrNoRows)
	got, err := s.recordsDBStrict(ctx, "bo-1")
	if err != nil || got != s.db {
		t.Fatalf("unbound BO: got %v, %v", got, err)
	}

	// Bound to a datasource with no connection config: an error, never s.db.
	mock.ExpectQuery("FROM public.business_object_binding").
		WillReturnRows(sqlmock.NewRows([]string{"backend_id"}).AddRow("ds-9"))
	mock.ExpectQuery("FROM public.tenant_product_datasource").WillReturnError(sql.ErrNoRows)
	got, err = s.recordsDBStrict(ctx, "bo-2")
	if err == nil || got != nil || !strings.Contains(err.Error(), "no connection configuration") {
		t.Fatalf("unresolvable datasource: got %v, %v", got, err)
	}

	// Reads still degrade (unchanged behaviour); writes do not.
	mock.ExpectQuery("FROM public.business_object_binding").
		WillReturnRows(sqlmock.NewRows([]string{"backend_id"}).AddRow("ds-9"))
	mock.ExpectQuery("FROM public.tenant_product_datasource").WillReturnError(sql.ErrNoRows)
	if s.resolveRecordsDB(ctx, "bo-2") != s.db {
		t.Error("reads keep their fallback")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestEnforceWriteBatch_RefusesUnresolvableDatasource(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	s := &BusinessObjectService{db: sqlx.NewDb(db, "sqlmock")}
	mock.ExpectQuery("FROM public.business_objects bo").
		WillReturnRows(sqlmock.NewRows([]string{"id", "bo_key", "driver_table_name"}).AddRow("bo-2", "security", "/orm/security"))
	mock.ExpectQuery("FROM public.business_object_binding").
		WillReturnRows(sqlmock.NewRows([]string{"backend_id"}).AddRow("ds-9"))
	mock.ExpectQuery("FROM public.tenant_product_datasource").WillReturnError(sql.ErrNoRows)
	wrote := false
	_, err := s.EnforceWriteBatch(context.Background(), "t1", "security", 1, false, func(*sqlx.Tx, int) (map[string]interface{}, error) {
		wrote = true
		return nil, nil
	})
	if err == nil || wrote {
		t.Fatalf("err=%v wrote=%v: the write must be refused before any transaction", err, wrote)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err) // in particular: no BEGIN on the metadata DB
	}
}
