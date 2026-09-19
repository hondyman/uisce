package driftread

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func TestListPending_BindsTenantOnly(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	tid := uuid.MustParse("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
	mock.ExpectQuery("FROM catalog_drift.schema_drift_proposals").
		WithArgs(tid).
		WillReturnRows(sqlmock.NewRows([]string{"proposal_id", "bo_name", "field_name", "proposed_column_name", "confidence_score"}))
	svc := NewService(sqlx.NewDb(db, "sqlmock"))
	got, err := svc.ListPending(context.Background(), tid)
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

// TestPackageIsReadOnly fails if driftread sources call Exec/INSERT/UPDATE/DELETE.
// Guards the map-review requirement: extract must not inherit DetectSchemaDrift writes.
func TestPackageIsReadOnly(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	forbidden := map[string]bool{
		"Exec": true, "ExecContext": true,
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		upper := strings.ToUpper(string(src))
		for _, kw := range []string{"INSERT ", "UPDATE ", "DELETE ", "UPSERT "} {
			if strings.Contains(upper, kw) {
				t.Errorf("%s contains write keyword %q — driftread must stay SELECT-only", name, strings.TrimSpace(kw))
			}
		}
		file, err := parser.ParseFile(fset, name, src, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if forbidden[sel.Sel.Name] {
				t.Errorf("%s:%d calls .%s — driftread must stay read-only", name, fset.Position(call.Pos()).Line, sel.Sel.Name)
			}
			return true
		})
	}
}
