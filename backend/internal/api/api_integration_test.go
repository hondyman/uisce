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

// Test DB-backed views pagination via Hasura
func TestViewsPaginationHandler_DBOnly(t *testing.T) {
	tmp := t.TempDir()
	runtimeResolved := filepath.Join(tmp, "runtime", "views_resolved")
	if err := os.MkdirAll(runtimeResolved, 0o755); err != nil {
		t.Fatal(err)
	}

	gqlResp := map[string]interface{}{
		"data": map[string]interface{}{
			"views": []map[string]interface{}{
				{
					"name": "d1",
					"view": map[string]interface{}{
						"name":        "d1",
						"title":       "Demo View",
						"description": "Example view from Hasura",
						"cubes":       []map[string]interface{}{},
						"folders":     []map[string]interface{}{},
					},
					"updated_at": "2025-01-01T00:00:00Z",
				},
			},
			"views_aggregate": map[string]interface{}{
				"aggregate": map[string]interface{}{
					"count": 1,
				},
			},
		},
	}

	// Start an httptest server to act as Hasura
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := json.Marshal(gqlResp)
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	}))
	defer hs.Close()

	// Patch HASURA_URL env for the handler
	os.Setenv("HASURA_URL", hs.URL)
	defer os.Unsetenv("HASURA_URL")

	// Set SEMLAYER_RUNTIME_DIR to our tmp so handler reads runtime files from there
	os.Setenv("SEMLAYER_RUNTIME_DIR", tmp)
	defer os.Unsetenv("SEMLAYER_RUNTIME_DIR")

	router := SetupRouter(nil, nil, nil, nil, nil, nil, nil, nil, nil)

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
