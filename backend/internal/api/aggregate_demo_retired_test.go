package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestRetiredAggregateDemoRoutes_Gone(t *testing.T) {
	r := chi.NewRouter()
	RegisterRetiredAggregateDemoRoutes(r)

	for _, tc := range []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/analytics/aggregates"},
		{http.MethodGet, "/analytics/aggregates"},
		{http.MethodDelete, "/analytics/aggregates"},
		{http.MethodPost, "/analytics/preview"},
		{http.MethodGet, "/analytics/preview"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusGone {
			t.Fatalf("%s %s: status = %d, want 410", tc.method, tc.path, rr.Code)
		}
		link := rr.Header().Get("Link")
		if !strings.Contains(link, "/api/cubes") || !strings.Contains(link, "successor-version") {
			t.Fatalf("%s %s: Link header = %q, want successor to /api/cubes", tc.method, tc.path, link)
		}
		var body map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatalf("%s %s: decode body: %v", tc.method, tc.path, err)
		}
		if body["error"] != "Gone" {
			t.Fatalf("%s %s: error = %v", tc.method, tc.path, body["error"])
		}
		if body["successor"] != "/api/cubes" {
			t.Fatalf("%s %s: successor = %v", tc.method, tc.path, body["successor"])
		}
	}
}
