package trading

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func TestLoadOrderDB_BindsIDAndTenant(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	tid := uuid.MustParse("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
	oid := "b1000000-0000-4000-8000-000000000003"
	mock.ExpectQuery(`FROM orm\."order"`).
		WithArgs(oid, tid).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "side", "target_qty", "leaves_qty", "limit_price", "sec_id", "status",
		}).AddRow(oid, "BUY", 100.0, 50.0, 10.5, "AAPL", "NEW"))
	got, err := LoadOrderDB(context.Background(), sqlx.NewDb(db, "sqlmock"), tid, oid)
	if err != nil {
		t.Fatal(err)
	}
	if got.OrderID != oid || got.Quantity != 50.0 || got.Symbol != "AAPL" {
		t.Fatalf("%#v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLoadOrderDB_WrongTenantBindShape(t *testing.T) {
	// Documents the fence: both args required; tenant is never omitted.
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	tid := uuid.MustParse("bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb")
	oid := "b1000000-0000-4000-8000-000000000003"
	mock.ExpectQuery(`FROM orm\."order"`).
		WithArgs(oid, tid).
		WillReturnError(sqlmock.ErrCancelled)
	_, err = LoadOrderDB(context.Background(), sqlx.NewDb(db, "sqlmock"), tid, oid)
	if err == nil {
		t.Fatal("expected error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
