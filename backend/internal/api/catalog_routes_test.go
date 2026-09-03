package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
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
	// when accessed anonymously or with invalid credentials.
	r := chi.NewRouter()
	
	fakeHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("X-User-ID") != "" {
			t.Errorf("expected X-User-ID to be stripped, got %q", req.Header.Get("X-User-ID"))
		}
		if req.Header.Get("X-Tenant-ID") != "" {
			t.Errorf("expected X-Tenant-ID to be stripped, got %q", req.Header.Get("X-Tenant-ID"))
		}
		w.WriteHeader(http.StatusOK)
	})

	secMgr := services.NewSecurityManager(nil, nil, []byte("test-secret-32-bytes-long-required!!"))
	r.With(appmid.AuthContextMiddleware(secMgr, nil)).Get("/api/catalog/scan/stream", fakeHandler)

	// 1. Spoofed headers with no credentials -> headers stripped downstream
	req := httptest.NewRequest(http.MethodGet, "/api/catalog/scan/stream", nil)
	req.Header.Set("X-User-ID", "attacker-id")
	req.Header.Set("X-Tenant-ID", "victim-tenant")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	// 2. Malformed credentials -> rejected with 401
	req401 := httptest.NewRequest(http.MethodGet, "/api/catalog/scan/stream", nil)
	req401.Header.Set("Authorization", "Bearer not.a.valid.jwt")
	w401 := httptest.NewRecorder()
	r.ServeHTTP(w401, req401)
	if w401.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", w401.Code)
	}
}
