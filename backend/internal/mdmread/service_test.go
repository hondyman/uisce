package mdmread

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func TestGetByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	tid := uuid.MustParse("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
	ex := uuid.MustParse("b1000000-0000-4000-8000-000000000001")
	mock.ExpectQuery("FROM catalog_mdm.universal_exception_queue").
		WithArgs(ex, tid).
		WillReturnError(sql.ErrNoRows)
	svc := NewService(sqlx.NewDb(db, "sqlmock"))
	_, err = svc.GetByID(context.Background(), tid, ex)
	if err != sql.ErrNoRows {
		t.Fatalf("expected ErrNoRows, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGetByID_Found(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	tid := uuid.MustParse("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
	ex := uuid.MustParse("b1000000-0000-4000-8000-000000000001")
	mock.ExpectQuery("FROM catalog_mdm.universal_exception_queue").
		WithArgs(ex, tid).
		WillReturnRows(sqlmock.NewRows([]string{"domain_key", "master_entity_sid", "field_name", "competing_values"}).
			AddRow("sec", "sid-1", "isin", []byte(`[]`)))
	svc := NewService(sqlx.NewDb(db, "sqlmock"))
	got, err := svc.GetByID(context.Background(), tid, ex)
	if err != nil {
		t.Fatal(err)
	}
	if got.FieldName != "isin" {
		t.Fatalf("%#v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
