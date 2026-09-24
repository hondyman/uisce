package boread

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func expectTenantTx(mock sqlmock.Sqlmock, tenant, gold uuid.UUID) {
	mock.ExpectQuery("uisce_gold_copy_tenant_id").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(gold))
	mock.ExpectBegin()
	mock.ExpectExec("uisce\\.current_tenant").WithArgs(tenant.String()).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("app\\.tenant_id").WithArgs(tenant.String()).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("uisce\\.gold_tenant").WithArgs(gold.String()).WillReturnResult(sqlmock.NewResult(0, 0))
}

func TestListSummaries_BindsTenantAndGold(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	tid := uuid.MustParse("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
	gold := uuid.MustParse("99999999-9999-4999-8999-999999999999")
	expectTenantTx(mock, tid, gold)
	mock.ExpectQuery("FROM public.business_objects").
		WithArgs(tid.String(), gold.String()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "display_name", "status"}).
			AddRow("1", "order", "Order", "ACTIVE"))
	mock.ExpectCommit()
	svc := NewService(sqlx.NewDb(db, "sqlmock"))
	got, err := svc.ListSummaries(context.Background(), tid)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "order" {
		t.Fatalf("%#v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGetContract_BindsIDKeyTenantGold(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	tid := uuid.MustParse("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
	gold := uuid.MustParse("99999999-9999-4999-8999-999999999999")
	boID := uuid.MustParse("b611af7b-8689-407d-807a-eeb315065e7d")
	expectTenantTx(mock, tid, gold)
	mock.ExpectQuery("FROM public.business_objects").
		WithArgs(boID.String(), "order", tid.String(), gold.String()).
		WillReturnRows(sqlmock.NewRows([]string{"name", "display_name", "status"}).
			AddRow("order", "Order", "ACTIVE"))
	mock.ExpectCommit()
	svc := NewService(sqlx.NewDb(db, "sqlmock"))
	got, err := svc.GetContract(context.Background(), tid, boID, "order")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "order" {
		t.Fatalf("%#v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSearch_BindsTenantGoldAndQuery(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	tid := uuid.MustParse("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
	gold := uuid.MustParse("99999999-9999-4999-8999-999999999999")
	expectTenantTx(mock, tid, gold)
	mock.ExpectQuery("FROM public.business_objects").
		WithArgs(tid.String(), gold.String(), "order").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "display_name"}).
			AddRow("1", "order", "Order"))
	mock.ExpectCommit()
	svc := NewService(sqlx.NewDb(db, "sqlmock"))
	got, err := svc.Search(context.Background(), tid, "order")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "order" {
		t.Fatalf("%#v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestListFieldSchema_BindsTenantAndBOID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	tid := uuid.MustParse("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
	gold := uuid.MustParse("99999999-9999-4999-8999-999999999999")
	expectTenantTx(mock, tid, gold)
	mock.ExpectQuery("FROM public.business_object_fields").
		WithArgs(tid, "bo-1").
		WillReturnRows(sqlmock.NewRows([]string{"name", "display_name", "data_type"}))
	mock.ExpectCommit()
	svc := NewService(sqlx.NewDb(db, "sqlmock"))
	got, err := svc.ListFieldSchema(context.Background(), tid, "bo-1")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("expected empty slice")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
