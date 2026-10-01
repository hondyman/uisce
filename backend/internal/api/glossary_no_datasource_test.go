package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/hondyman/uisce/backend/internal/handlers"
	"github.com/hondyman/uisce/backend/internal/security"
)

type noDatasourceResolver struct{}

func (noDatasourceResolver) Resolve(context.Context, string) (*security.ResolvedDatasource, error) {
	return nil, errors.New("no datasource expected")
}

// A tenant-only request (no datasource selected) must list the tenant's
// terms, not bind the "none" placeholder into the uuid datasource column.
func TestListTerms_NoDatasourceSelected(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	const tenant = "11111111-1111-4111-8111-111111111111"
	mock.ExpectQuery(`FROM catalog_node cn`).
		WithArgs(tenant, "semantic_term").
		WillReturnRows(sqlmock.NewRows([]string{"id", "node_name"}).AddRow("n1", "Fund AUM"))

	h := &GlossaryHandler{db: db, securityDeps: handlers.SecurityContextDeps{Resolver: noDatasourceResolver{}}}
	req := httptest.NewRequest(http.MethodGet, "/api/glossary/semantic-terms", nil)
	req = req.WithContext(security.WithAuthInfo(req.Context(), security.AuthInfo{UserID: "u1", TenantIDs: []string{tenant}}))
	rec := httptest.NewRecorder()
	h.listTerms(rec, req, "semantic_term")

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestScopedDatasourceID(t *testing.T) {
	cases := map[string]string{"none": "", "": "", "25b5dce3-27d9-4773-933e-6ee29a42871f": "25b5dce3-27d9-4773-933e-6ee29a42871f"}
	for in, want := range cases {
		if got := (&security.Context{DatasourceID: in}).ScopedDatasourceID(); got != want {
			t.Errorf("ScopedDatasourceID(%q) = %q, want %q", in, got, want)
		}
	}
	if got := (*security.Context)(nil).ScopedDatasourceID(); got != "" {
		t.Errorf("nil context = %q", got)
	}
}
