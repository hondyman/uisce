package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/jmoiron/sqlx"
)

const (
	pipelineTestTenant = "11111111-1111-1111-1111-111111111111"
	goldCopyTenant     = "00000000-0000-0000-0000-000000000001"
)

// newPipelineTestRouter builds the handler on a sqlmock DB with no expected
// queries, so any DB access by a refused request fails the test.
func newPipelineTestRouter(t *testing.T) (http.Handler, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	h := &PipelineHandler{db: sqlx.NewDb(db, "postgres")}
	r := chi.NewRouter()
	h.RegisterRoutes(r)
	return r, mock
}

func pipelineRequest(method, path string, auth *security.AuthInfo, tenantHeader string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(`{"formData":{},"name":"p","pipeline_json":{"nodes":[],"edges":[]}}`))
	if auth != nil {
		req = req.WithContext(security.WithAuthInfo(context.Background(), *auth))
	}
	if tenantHeader != "" {
		req.Header.Set("X-Tenant-ID", tenantHeader)
	}
	return req
}

func TestPipelineHandler_RefusesRequestsWithoutResolvableTenant(t *testing.T) {
	routes := []struct{ method, path string }{
		{http.MethodGet, "/api/v1/pipelines"},
		{http.MethodPost, "/api/v1/pipelines"},
		{http.MethodGet, "/api/v1/pipelines/p1"},
		{http.MethodPut, "/api/v1/pipelines/p1"},
		{http.MethodDelete, "/api/v1/pipelines/p1"},
		{http.MethodPost, "/api/v1/pipelines/p1/execute"},
		{http.MethodPost, "/api/v1/pipelines/p1/simulate"},
	}
	cases := []struct {
		name   string
		auth   *security.AuthInfo
		header string
		want   int
	}{
		{"no auth context", nil, "", http.StatusUnauthorized},
		{"authenticated but no tenant", &security.AuthInfo{UserID: "u1"}, "", http.StatusUnauthorized},
		{"tenant but no user", &security.AuthInfo{TenantIDs: []string{pipelineTestTenant}}, "", http.StatusUnauthorized},
		{"no tenant, gold-copy header", &security.AuthInfo{UserID: "u1"}, goldCopyTenant, http.StatusForbidden},
		{"foreign tenant header", &security.AuthInfo{UserID: "u1", TenantIDs: []string{pipelineTestTenant}}, goldCopyTenant, http.StatusForbidden},
	}
	for _, rt := range routes {
		for _, tc := range cases {
			t.Run(rt.method+" "+rt.path+"/"+tc.name, func(t *testing.T) {
				router, mock := newPipelineTestRouter(t)
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, pipelineRequest(rt.method, rt.path, tc.auth, tc.header))
				if rec.Code != tc.want {
					t.Fatalf("status = %d, want %d (body %q)", rec.Code, tc.want, rec.Body.String())
				}
				if err := mock.ExpectationsWereMet(); err != nil {
					t.Fatalf("unexpected DB interaction: %v", err)
				}
			})
		}
	}
}

func TestPipelineHandler_TenantScopedRequestSucceeds(t *testing.T) {
	auth := &security.AuthInfo{UserID: "u1", TenantIDs: []string{pipelineTestTenant}}

	t.Run("list", func(t *testing.T) {
		router, mock := newPipelineTestRouter(t)
		mock.ExpectQuery(`FROM pipelines`).
			WithArgs(pipelineTestTenant).
			WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "name"}).AddRow("p1", pipelineTestTenant, "P"))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, pipelineRequest(http.MethodGet, "/api/v1/pipelines", auth, ""))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body %q", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), pipelineTestTenant) {
			t.Fatalf("body missing tenant pipeline: %q", rec.Body.String())
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("delete with matching tenant header", func(t *testing.T) {
		router, mock := newPipelineTestRouter(t)
		mock.ExpectExec(`UPDATE pipelines SET is_active = false`).
			WithArgs("p1", pipelineTestTenant).
			WillReturnResult(sqlmock.NewResult(0, 1))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, pipelineRequest(http.MethodDelete, "/api/v1/pipelines/p1", auth, pipelineTestTenant))
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, body %q", rec.Code, rec.Body.String())
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
}
