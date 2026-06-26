package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

type fakeDomainHandler struct{}

func (f *fakeDomainHandler) RegisterRoutes(r chi.Router) {
	r.Get("/data-domains/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
}

func TestRegisterDomainRoutes(t *testing.T) {
	r := chi.NewRouter()
	routes := NewRoutes()
	routes.RegisterDomains(r, &fakeDomainHandler{})

	req := httptest.NewRequest(http.MethodGet, "/data-domains/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
}
