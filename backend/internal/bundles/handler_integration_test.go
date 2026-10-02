//go:build integration

package bundles

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestReloadGuardrailsHandler_Integration(t *testing.T) {
	// Guardrails are DB-only; there is no YAML file to stage. This test used
	// to write a temp guardrails.yaml and point GUARDRAILS_PATH at it, which
	// asserted a fallback path that no longer exists.
	r := chi.NewRouter()
	RegisterRoutes(r)

	// POST reload with admin header
	// Note: RegisterRoutes mounts at /bundles, so the path is /bundles/guardrails/reload
	reqInfo := httptest.NewRequest("POST", "/bundles/guardrails/reload", bytes.NewReader([]byte(`{}`)))
	reqInfo.Header.Set("X-User-Role", "admin")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, reqInfo)
	if w.Code != http.StatusOK {
		t.Fatalf("reload failed: %d %s", w.Code, w.Body.String())
	}

	// GET cache
	req2 := httptest.NewRequest("GET", "/bundles/guardrails/cache", nil)
	req2.Header.Set("X-User-Role", "admin")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("get cache failed: %d %s", w2.Code, w2.Body.String())
	}
	var resp struct {
		Cache GuardrailCache `json:"cache"`
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Cache.Config == nil {
		t.Fatalf("cache config nil")
	}
	// Source is never "yaml" now: the DB is the only supported source.
	if resp.Cache.Source == "yaml" {
		t.Fatalf("yaml must no longer be a possible source, got %q", resp.Cache.Source)
	}
	if resp.Cache.LastLoaded.IsZero() {
		t.Fatalf("expected last_loaded to be set")
	}
}
