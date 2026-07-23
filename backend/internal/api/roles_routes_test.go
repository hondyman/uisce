package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

type fakeRoleHandler struct{}

func (f *fakeRoleHandler) RegisterRoutes(r chi.Router) {
	r.Get("/roles/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
}

func TestRegisterRoleRoutes(t *testing.T) {
	r := chi.NewRouter()
	routes := NewRoutes()
	routes.RegisterRoles(r, &fakeRoleHandler{})

	req := httptest.NewRequest(http.MethodGet, "/roles/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
}
