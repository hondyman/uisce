//go:build integration

package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// Test DB-backed views pagination
func TestViewsPaginationHandler_DBOnly(t *testing.T) {
	tmp := t.TempDir()
	runtimeResolved := filepath.Join(tmp, "runtime", "views_resolved")
	if err := os.MkdirAll(runtimeResolved, 0o755); err != nil {
		t.Fatal(err)
	}

	// Set SEMLAYER_RUNTIME_DIR to our tmp so handler reads runtime files from there
	os.Setenv("SEMLAYER_RUNTIME_DIR", tmp)
	defer os.Unsetenv("SEMLAYER_RUNTIME_DIR")

	router := SetupRouter(nil, nil, nil, nil, nil, nil, nil, nil, nil, "")

	// Issue request to /api/views?source=resolved&page=1&page_size=10
	req := httptest.NewRequest(http.MethodGet, "/api/views?source=resolved&page=1&page_size=10", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d body=%s", rr.Code, rr.Body.String())
	}
	// parse body
	var resp map[string]interface{}
	if err := json.NewDecoder(bytes.NewReader(rr.Body.Bytes())).Decode(&resp); err != nil {
		t.Fatalf("failed decode: %v", err)
	}
	total, _ := resp["total"].(float64)
	if int(total) != 1 {
		t.Fatalf("expected total 1 got %v", resp["total"])
	}

	views, ok := resp["views"].([]any)
	if !ok || len(views) != 1 {
		t.Fatalf("expected exactly one view entry got %v", resp["views"])
	}
}
