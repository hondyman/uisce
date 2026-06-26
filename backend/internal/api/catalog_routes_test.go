package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
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
