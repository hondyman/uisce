package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/hondyman/uisce/backend/internal/handlers"
)

const (
	ntCaller = "11111111-1111-1111-1111-111111111111"
	ntOther  = "22222222-2222-2222-2222-222222222222"
	ntGold   = "99e99e99-99e9-49e9-89e9-99e99e99e999"
)

func postNodeType(t *testing.T, h *NodeTypesHandler, body, tenant string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/node-types", strings.NewReader(body))
	if tenant != "" {
		req = withAuthContext(req, tenant)
	}
	rec := httptest.NewRecorder()
	h.handleCreateNodeType(rec, req)
	return rec
}

// A caller cannot name another tenant in the body: catalog_node_type has no RLS, so this check is the
// only thing keeping a tenant out of another tenant's (or the gold copy's) node types.
func TestCreateNodeType_RejectsTenantFromBody(t *testing.T) {
	for _, target := range []string{ntOther, ntGold} {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatal(err)
		}
		h := NewNodeTypesHandler(db, handlers.SecurityContextDeps{})
		rec := postNodeType(t, h, `{"catalog_type_name":"x","tenant_id":"`+target+`"}`, ntCaller)
		if rec.Code != http.StatusForbidden {
			t.Errorf("body tenant %s: status %d, want 403", target, rec.Code)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("body tenant %s reached the database: %v", target, err)
		}
		db.Close()
	}
}

func TestCreateNodeType_RequiresAuthentication(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()
	rec := postNodeType(t, NewNodeTypesHandler(db, handlers.SecurityContextDeps{}), `{"catalog_type_name":"x"}`, "")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status %d, want 401", rec.Code)
	}
}

// Without a body tenant, and with a matching one, the row is created in the caller's own tenant.
func TestCreateNodeType_UsesCallerTenant(t *testing.T) {
	for _, body := range []string{
		`{"catalog_type_name":"x"}`,
		`{"catalog_type_name":"x","tenant_id":"` + ntCaller + `"}`,
	} {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatal(err)
		}
		mock.ExpectQuery("FROM tenants").WithArgs(ntCaller).
			WillReturnRows(sqlmock.NewRows([]string{"gold_copy"}).AddRow(false))
		mock.ExpectQuery("INSERT INTO catalog_node_type").
			WithArgs(sqlmock.AnyArg(), ntCaller, "x", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow("id1", time.Now(), time.Now()))
		rec := postNodeType(t, NewNodeTypesHandler(db, handlers.SecurityContextDeps{}), body, ntCaller)
		if rec.Code != http.StatusCreated {
			t.Errorf("%s: status %d (%s), want 201", body, rec.Code, rec.Body.String())
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("%s: %v", body, err)
		}
		db.Close()
	}
}
