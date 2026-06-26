package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func writeView(t *testing.T, dir, name string, body any) {
	t.Helper()
	b, _ := json.Marshal(body)
	if err := os.WriteFile(filepath.Join(dir, name+".json"), b, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestViewsListAndETag(t *testing.T) {
	tmp := t.TempDir()
	// create two views
	writeView(t, tmp, "orders_view", map[string]any{"name": "orders_view", "cubes": []any{"orders"}})
	writeView(t, tmp, "customers_view", map[string]any{"name": "customers_view", "cubes": []any{"customers"}})

	h := NewViewsHandler(tmp, "")
	r := chi.NewRouter()
	r.Get("/api/views", h.ListViews)

	// first call
	req, _ := http.NewRequest("GET", "/api/views?page=1&page_size=1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status %d", w.Code)
	}
	tag := w.Header().Get("ETag")
	if tag == "" {
		t.Fatalf("missing ETag")
	}

	// second call with If-None-Match
	req2, _ := http.NewRequest("GET", "/api/views?page=1&page_size=1", nil)
	req2.Header.Set("If-None-Match", tag)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusNotModified {
		t.Fatalf("expected 304, got %d", w2.Code)
	}
}

func TestViewsSourceResolvedAndDownload(t *testing.T) {
	tmpGen := t.TempDir()
	tmpRes := t.TempDir()
	writeView(t, tmpGen, "users_view", map[string]any{"name": "users_view", "description": "gen"})
	writeView(t, tmpRes, "users_view", map[string]any{"name": "users_view", "description": "resolved"})

	h := NewViewsHandler(tmpGen, tmpRes)
	r := chi.NewRouter()
	r.Get("/api/views", h.ListViews)
	r.Get("/api/views/{name}", h.GetView)
	r.Get("/api/views/{name}/download", h.DownloadView)

	// resolved list
	req, _ := http.NewRequest("GET", "/api/views?source=resolved", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "users_view") {
		t.Fatalf("missing users_view")
	}

	// download should set content disposition
	req2, _ := http.NewRequest("GET", "/api/views/users_view/download?source=resolved", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("status %d", w2.Code)
	}
	if cd := w2.Header().Get("Content-Disposition"); cd == "" {
		t.Fatalf("missing Content-Disposition")
	}
}
