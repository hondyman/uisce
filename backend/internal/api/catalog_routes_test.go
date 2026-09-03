package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/handlers"
	appmid "github.com/hondyman/uisce/backend/internal/middleware"
	"github.com/hondyman/uisce/backend/internal/services"
)

type fakeCatalogHandler struct{}

func (f *fakeCatalogHandler) HandleCatalogScan(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	if strings.Contains(string(body), "scan") {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("scanned"))
		return
	}
	w.WriteHeader(http.StatusBadRequest)
}

func TestRegisterCatalogScanRoute(t *testing.T) {
	// Use chi router since RegisterCatalogScan expects chi.Router
	r := chi.NewRouter()
	routes := NewRoutes()
	routes.RegisterCatalogScan(r, &fakeCatalogHandler{})

	req := httptest.NewRequest(http.MethodPost, "/catalog/scan", strings.NewReader("scan=true"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestCatalogScanStream_AuthProtection(t *testing.T) {
	// Verify that /api/catalog/scan/stream on rootMux strips spoofed identity headers
	// and strictly enforces authentication with 401 Unauthorized.
	r := chi.NewRouter()

	handler := handlers.NewCatalogScanHandler(nil, handlers.SecurityContextDeps{})

	secMgr := services.NewSecurityManager(nil, nil, []byte("test-secret-32-bytes-long-required!!"))
	r.With(appmid.AuthContextMiddleware(secMgr, nil)).Get("/api/catalog/scan/stream", handler.HandleScanStream)

	// 1. Spoofed headers with no credentials -> stripped by middleware, rejected with 401 by handler
	req := httptest.NewRequest(http.MethodGet, "/api/catalog/scan/stream?datasource_id=00000000-0000-0000-0000-000000000001", nil)
	req.Header.Set("X-User-ID", "attacker-id")
	req.Header.Set("X-Tenant-ID", "victim-tenant")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for unauthenticated/spoofed request, got %d body=%s", w.Code, w.Body.String())
	}

	// 2. Malformed credentials -> rejected with 401 by middleware
	req401 := httptest.NewRequest(http.MethodGet, "/api/catalog/scan/stream", nil)
	req401.Header.Set("Authorization", "Bearer not.a.valid.jwt")
	w401 := httptest.NewRecorder()
	r.ServeHTTP(w401, req401)
	if w401.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", w401.Code)
	}
}
