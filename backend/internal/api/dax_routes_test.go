package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

type fakeDAXHandler struct{}

func (f *fakeDAXHandler) RegisterRoutes(r chi.Router) {
	r.Get("/dax/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestRegisterDAXRoutes(t *testing.T) {
	r := chi.NewRouter()
	routes := NewRoutes()
	routes.RegisterDAX(r, &fakeDAXHandler{})

	req := httptest.NewRequest(http.MethodGet, "/dax/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
}
