package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

type fakeIPWhitelistHandler struct{}

func (f *fakeIPWhitelistHandler) RegisterRoutes(r chi.Router) {
	r.Get("/ipwhitelist/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestRegisterIPWhitelistRoutes(t *testing.T) {
	r := chi.NewRouter()
	routes := NewRoutes()
	routes.RegisterIPWhitelist(r, &fakeIPWhitelistHandler{})

	req := httptest.NewRequest(http.MethodGet, "/ipwhitelist/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
}
