package boread

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func TestListSummaries_BindsTenantAndGold(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	tid := uuid.MustParse("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
	gold := uuid.MustParse("99999999-9999-4999-8999-999999999999")
	mock.ExpectQuery("FROM public.tenants").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(gold))
	mock.ExpectQuery("FROM public.business_objects").
		WithArgs(tid.String(), gold.String()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "display_name", "status"}).
			AddRow("1", "order", "Order", "ACTIVE"))
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
	mock.ExpectQuery("FROM public.tenants").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(gold))
	mock.ExpectQuery("FROM public.business_objects").
		WithArgs(boID.String(), "order", tid.String(), gold.String()).
		WillReturnRows(sqlmock.NewRows([]string{"name", "display_name", "status"}).
			AddRow("order", "Order", "ACTIVE"))
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
	mock.ExpectQuery("FROM public.tenants").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(gold))
	mock.ExpectQuery("FROM public.business_objects").
		WithArgs(tid.String(), gold.String(), "order").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "display_name"}).
			AddRow("1", "order", "Order"))
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
	mock.ExpectQuery("FROM public.business_object_fields").
		WithArgs(tid, "bo-1").
		WillReturnRows(sqlmock.NewRows([]string{"name", "display_name", "data_type"}))
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
