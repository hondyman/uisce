package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/hondyman/uisce/backend/internal/services"
	"github.com/jmoiron/sqlx"
)

const domID = "00000000-0000-4000-8000-0000000000d1"

func newDomainRouter(t *testing.T) (http.Handler, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	r := chi.NewRouter()
	NewDomainHandler(services.NewDomainService(sqlx.NewDb(db, "postgres"))).RegisterRoutes(r)
	return r, mock
}

func doAs(h http.Handler, method, path, body string, auth *security.AuthInfo) int {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if auth != nil {
		req = req.WithContext(security.WithAuthInfo(req.Context(), *auth))
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w.Code
}

// data_domain is one global taxonomy: every tenant reads it, only gold-copy (core) administrators change it.
func TestDataDomainWrites_RequireACoreAdmin(t *testing.T) {
	writes := []struct{ method, path, body string }{
		{"POST", "/data-domains/", `{"name":"Risk"}`},
		{"PUT", "/data-domains/" + domID + "/", `{"name":"Risk"}`},
		{"DELETE", "/data-domains/" + domID + "/", ""},
	}
	denied := map[string]*security.AuthInfo{
		"a regular tenant user":       {UserID: "u", TenantIDs: []string{vrTenant}, Roles: []string{"user"}},
		"a tenant administrator":      {UserID: "u", TenantIDs: []string{vrTenant}, Roles: []string{"tenant_admin"}},
		"global_ops (support role)":   {UserID: "u", TenantIDs: []string{vrTenant}, Roles: []string{"global_ops"}, IsGlobalAdmin: true},
		"helpdesk (support role)":     {UserID: "u", TenantIDs: []string{vrTenant}, Roles: []string{"helpdesk"}},
		"an admin with no admin role": {UserID: "u", TenantIDs: []string{vrTenant}},
	}
	for who, auth := range denied {
		for _, c := range writes {
			h, mock := newDomainRouter(t)
			if code := doAs(h, c.method, c.path, c.body, auth); code != http.StatusForbidden {
				t.Errorf("%s: %s %s = %d; want 403", who, c.method, c.path, code)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("%s: %s %s reached the database: %v", who, c.method, c.path, err)
			}
		}
	}
	for _, c := range writes {
		h, mock := newDomainRouter(t)
		if code := doAs(h, c.method, c.path, c.body, nil); code != http.StatusUnauthorized {
			t.Errorf("unauthenticated %s %s = %d; want 401", c.method, c.path, code)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unauthenticated %s %s reached the database: %v", c.method, c.path, err)
		}
	}
}

func TestDataDomainWrites_CoreAdminsMayWrite(t *testing.T) {
	for _, role := range []string{"core_admin", "is_core_admin", "global_admin", "GLOBAL_ADMIN"} {
		auth := &security.AuthInfo{UserID: "u", TenantIDs: []string{vrGold}, Roles: []string{role}}

		h, mock := newDomainRouter(t)
		mock.ExpectExec(`INSERT INTO public.data_domain`).WillReturnResult(sqlmock.NewResult(1, 1))
		if code := doAs(h, "POST", "/data-domains/", `{"name":"Risk"}`, auth); code != http.StatusCreated {
			t.Errorf("%s create = %d; want 201", role, code)
		}
		h, mock = newDomainRouter(t)
		mock.ExpectExec(`UPDATE public.data_domain`).WillReturnResult(sqlmock.NewResult(0, 1))
		if code := doAs(h, "PUT", "/data-domains/"+domID+"/", `{"name":"Risk"}`, auth); code != http.StatusOK {
			t.Errorf("%s update = %d; want 200", role, code)
		}
		h, mock = newDomainRouter(t)
		mock.ExpectExec(`DELETE FROM public.data_domain`).WillReturnResult(sqlmock.NewResult(0, 1))
		if code := doAs(h, "DELETE", "/data-domains/"+domID+"/", "", auth); code != http.StatusNoContent {
			t.Errorf("%s delete = %d; want 204", role, code)
		}
	}
}

func TestDataDomainReads_AreOpenToEveryTenant(t *testing.T) {
	tenantUser := &security.AuthInfo{UserID: "u", TenantIDs: []string{vrTenant}, Roles: []string{"user"}}
	h, mock := newDomainRouter(t)
	mock.ExpectQuery(`FROM public.data_domain`).WillReturnRows(sqlmock.NewRows([]string{"id", "name", "slug", "parent_id", "level", "description", "created_by", "created_at", "updated_at"}))
	if code := doAs(h, "GET", "/data-domains/", "", tenantUser); code != http.StatusOK {
		t.Errorf("a regular tenant listing data domains = %d; want 200", code)
	}
}
