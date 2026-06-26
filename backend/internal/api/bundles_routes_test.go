package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

type fakeBundleHandler struct{}

func (f *fakeBundleHandler) RegisterRoutes(r chi.Router) {
	r.Get("/bundles/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
}

func TestRegisterBundleRoutes(t *testing.T) {
	r := chi.NewRouter()
	routes := NewRoutes()
	routes.RegisterBundles(r, &fakeBundleHandler{})

	req := httptest.NewRequest(http.MethodGet, "/bundles/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
}
