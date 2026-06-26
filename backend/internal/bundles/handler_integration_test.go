package bundles

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestReloadGuardrailsHandler_Integration(t *testing.T) {
	// create temp YAML file
	yaml := `sod_pairs:
  - ["a","b"]
certified:
  - "c"
`
	tmp, err := os.CreateTemp("", "guardrails-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write([]byte(yaml)); err != nil {
		t.Fatalf("write tmp: %v", err)
	}
	if err := tmp.Close(); err != nil {
		t.Fatalf("close tmp: %v", err)
	}

	// set env to point to tmp file so loadGuardrails will pick it up (when DB nil)
	op := os.Getenv("GUARDRAILS_PATH")
	defer os.Setenv("GUARDRAILS_PATH", op)
	os.Setenv("GUARDRAILS_PATH", tmp.Name())

	// Create router and mount handlers
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
	if len(resp.Cache.Config.SoDPairs) != 1 || len(resp.Cache.Config.Certified) != 1 {
		t.Fatalf("unexpected cache contents: %+v", resp.Cache.Config)
	}
	if resp.Cache.Source != "yaml" {
		t.Fatalf("expected source yaml, got %s", resp.Cache.Source)
	}
	if resp.Cache.LastLoaded.IsZero() {
		t.Fatalf("expected last_loaded to be set")
	}
}
