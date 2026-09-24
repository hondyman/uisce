package goldcopy

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func TestResolveTenantID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	gold := uuid.MustParse("99999999-9999-4999-8999-999999999999")
	mock.ExpectQuery("uisce_gold_copy_tenant_id").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(gold))
	got := ResolveTenantID(context.Background(), sqlx.NewDb(db, "sqlmock"))
	if got != gold {
		t.Fatalf("got %s want %s", got, gold)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
