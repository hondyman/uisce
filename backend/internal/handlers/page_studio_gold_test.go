package handlers

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// TestPageStudio_GoldCopyID_UsesSharedResolver is the HTTP-surface receipt for
// the gold-copy widen: PageStudioHandler must resolve gold via goldcopy, not a
// private inline query that can drift from MCP/pagestudio.
func TestPageStudio_GoldCopyID_UsesSharedResolver(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no caller")
	}
	src, err := os.ReadFile(filepath.Join(filepath.Dir(file), "page_studio_handler.go"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(src)
	if !strings.Contains(body, "goldcopy.ResolveTenantID") {
		t.Fatal("PageStudioHandler.goldCopyID must call goldcopy.ResolveTenantID")
	}
	// Private inline tenants lookup must not remain as the gold source.
	if strings.Contains(body, "SELECT id FROM public.tenants WHERE gold_copy = true ORDER BY created_at LIMIT 1") {
		t.Fatal("handler must not keep a private gold-copy SQL copy")
	}
}

// TestPageStudio_GoldCopyID_ReturnsResolvedGold proves the shared path binds
// the tenants.gold_copy lookup (same query MCP/pagestudio use).
func TestPageStudio_GoldCopyID_ReturnsResolvedGold(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	gold := uuid.MustParse("99999999-9999-4999-8999-999999999999")
	mock.ExpectQuery("FROM public.tenants").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(gold))
	h := &PageStudioHandler{db: sqlx.NewDb(db, "sqlmock")}
	got := h.goldCopyID(context.Background())
	if got != gold {
		t.Fatalf("got %s want %s", got, gold)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
