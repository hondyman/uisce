package region_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/region"
)

func TestRegionMiddleware_Presence(t *testing.T) {
	r := chi.NewRouter()
	r.Use(region.RegionValidationMiddleware(nil))
	r.Get("/test", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reg, ok := region.GetRegionFromContext(r.Context())
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("missing region in context"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"region": reg})
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set(region.RegionHeader, "eu-west")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
}
