package pagestudio

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func TestListSummaries_BindsTenantOnly(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	tid := uuid.MustParse("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
	mock.ExpectQuery("FROM public.page_definitions").
		WithArgs(tid).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "slug", "status"}).
			AddRow("p1", "Page", "slug", "draft"))
	svc := NewService(sqlx.NewDb(db, "sqlmock"))
	got, err := svc.ListSummaries(context.Background(), tid)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Slug != "slug" {
		t.Fatalf("unexpected: %#v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGetByIDOrSlug_BindsTenantAndID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	tid := uuid.MustParse("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
	mock.ExpectQuery("FROM public.page_definitions").
		WithArgs(tid, "pid", "slug").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "slug", "status", "layout", "components", "data_sources", "presentation_events", "filter_bar",
		}).AddRow("pid", "N", "slug", "draft", []byte(`{}`), []byte(`[]`), []byte(`[]`), []byte(`[]`), []byte(`{}`)))
	svc := NewService(sqlx.NewDb(db, "sqlmock"))
	got, err := svc.GetByIDOrSlug(context.Background(), tid, "pid", "slug")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "pid" {
		t.Fatalf("got %#v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestListBySlugs_BindsTenantAndTwoSlugs(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	tid := uuid.MustParse("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
	mock.ExpectQuery("FROM public.page_definitions").
		WithArgs(tid, "a", "b").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "slug", "status"}))
	svc := NewService(sqlx.NewDb(db, "sqlmock"))
	got, err := svc.ListBySlugs(context.Background(), tid, "a", "b")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("expected empty slice, not nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
