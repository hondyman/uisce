package handlers

import (
	"database/sql/driver"
	"net/http"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/jmoiron/sqlx"
)

// The pre-aggregation and calc-term id routes must be tenant-scoped. The ownership SQL itself is proven
// against a real Postgres (see the commit message); these tests pin the HTTP behaviour: a validated
// tenant is required before any query runs, and a rule the tenant does not own is a plain 404.

func newPreAggRouter(t *testing.T) (http.Handler, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	x := sqlx.NewDb(db, "postgres")
	r := chi.NewRouter()
	NewPreAggregationHandler(analytics.NewPreAggregationService(x, nil, nil)).RegisterRoutes(r)
	return r, mock
}

func TestPreAggRoutes_RequireATenantBeforeAnyQuery(t *testing.T) {
	for _, c := range []struct{ method, path, body string }{
		{"GET", "/preaggregations/" + vrRuleID, ""},
		{"DELETE", "/preaggregations/" + vrRuleID, ""},
		{"POST", "/preaggregations/" + vrRuleID + "/ddl", ""},
		{"POST", "/preaggregations/" + vrRuleID + "/refresh", ""},
		{"PUT", "/preaggregations/" + vrRuleID, `{}`},
	} {
		h, mock := newPreAggRouter(t)
		if w := do(h, c.method, c.path, c.body, ""); w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without a tenant = %d; want 401", c.method, c.path, w.Code)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("%s %s ran a query before the tenant was known: %v", c.method, c.path, err)
		}
	}
}

func TestPreAggRoutes_ForeignPreAggIsNotFoundAndNothingIsWritten(t *testing.T) {
	notOwned := func(m sqlmock.Sqlmock) {
		m.ExpectQuery(`SELECT EXISTS`).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	}
	t.Run("delete", func(t *testing.T) {
		h, mock := newPreAggRouter(t)
		mock.ExpectBegin()
		notOwned(mock)
		mock.ExpectRollback() // nothing deleted: no DELETE statement is expected, so one would fail the test
		if w := do(h, "DELETE", "/preaggregations/"+vrRuleID, "", vrTenant); w.Code != http.StatusNotFound {
			t.Errorf("delete of another tenant's pre-aggregation = %d; want 404", w.Code)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})
	for _, c := range []struct{ name, method, path string }{
		{"ddl", "POST", "/preaggregations/" + vrRuleID + "/ddl"},
		{"refresh", "POST", "/preaggregations/" + vrRuleID + "/refresh"},
	} {
		t.Run(c.name, func(t *testing.T) {
			h, mock := newPreAggRouter(t)
			notOwned(mock)
			if w := do(h, c.method, c.path, "", vrTenant); w.Code != http.StatusNotFound {
				t.Errorf("%s of another tenant's pre-aggregation = %d; want 404", c.name, w.Code)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("only the ownership check may run: %v", err)
			}
		})
	}
	t.Run("update", func(t *testing.T) {
		h, mock := newPreAggRouter(t)
		mock.ExpectExec(`UPDATE catalog_node SET`).WillReturnResult(driver.RowsAffected(0)) // scoped WHERE matched nothing
		if w := do(h, "PUT", "/preaggregations/"+vrRuleID, `{"bo_name":"party"}`, vrTenant); w.Code != http.StatusNotFound {
			t.Errorf("update of another tenant's pre-aggregation = %d; want 404", w.Code)
		}
	})
	t.Run("get", func(t *testing.T) {
		h, mock := newPreAggRouter(t)
		mock.ExpectQuery(`FROM catalog_node n`).WillReturnRows(sqlmock.NewRows([]string{"id", "node_name", "description", "properties", "config", "created_at", "updated_at"}))
		if w := do(h, "GET", "/preaggregations/"+vrRuleID, "", vrTenant); w.Code != http.StatusNotFound {
			t.Errorf("get of another tenant's pre-aggregation = %d; want 404", w.Code)
		}
	})
}

func TestCalcTermGetByID_RequiresATenantAndIsScoped(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	x := sqlx.NewDb(db, "postgres")
	r := chi.NewRouter()
	NewCalcTermHandler(analytics.NewCalcTermService(x)).RegisterRoutes(r)

	if w := do(r, "GET", "/calc-terms/"+vrRuleID, "", ""); w.Code != http.StatusUnauthorized {
		t.Errorf("calc-term get without a tenant = %d; want 401", w.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("no query may run before the tenant is known: %v", err)
	}
	mock.ExpectQuery(`FROM catalog_node n`).WithArgs(vrRuleID, vrTenant).
		WillReturnRows(sqlmock.NewRows([]string{"id", "node_name", "description", "properties", "config"}))
	if w := do(r, "GET", "/calc-terms/"+vrRuleID, "", vrTenant); w.Code != http.StatusNotFound {
		t.Errorf("calc-term get of another tenant's term = %d; want 404", w.Code)
	}
}
