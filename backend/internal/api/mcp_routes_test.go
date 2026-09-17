package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/api"
	"github.com/hondyman/uisce/backend/internal/handlers"
	"github.com/hondyman/uisce/backend/internal/mcp"
)

// TestChi_DuplicateMethodPattern records chi v5.2.3's actual contract for
// two Post("/mcp") registrations on one router. Earlier workstreams assumed
// a startup panic; this test is the source of truth for PR B.
func TestChi_DuplicateMethodPattern(t *testing.T) {
	r := chi.NewRouter()
	r.Post("/mcp", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("first")) })
	defer func() {
		if rec := recover(); rec != nil {
			t.Logf("chi panicked on second POST /mcp: %v", rec)
		}
	}()
	r.Post("/mcp", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("second")) })

	req := httptest.NewRequest("POST", "/mcp", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	body := rec.Body.String()
	t.Logf("duplicate POST /mcp served %q (status %d)", body, rec.Code)
	if body != "second" {
		t.Fatalf("chi v5.2.3 last-registration-wins expected body %q, got %q", "second", body)
	}
}

type mcpRoute struct {
	Method string
	Route  string
}

func collectMCPRoutes(t *testing.T, router chi.Router) []mcpRoute {
	t.Helper()
	var out []mcpRoute
	if err := chi.Walk(router, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		low := strings.ToLower(route)
		if strings.Contains(low, "mcp") {
			out = append(out, mcpRoute{Method: method, Route: route})
		}
		return nil
	}); err != nil {
		t.Fatalf("chi.Walk: %v", err)
	}
	return out
}

func postJSON(handler http.Handler, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func getPath(handler http.Handler, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", path, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// TestMCP_RouteTableDump walks the real SetupRouter mux and probes /mcp*
// endpoints. Structural guard: Path 1 must remain registered AFTER Path 5
// (chi v5.2.3 last-wins). Flip checklist: this test + stderr [ROUTES-DUMP]
// + admin GET /_routes.
func TestMCP_RouteTableDump(t *testing.T) {
	t.Setenv("DISABLE_BACKGROUND_JOBS", "true")
	t.Setenv("ENVIRONMENT", "test")

	var traces []string
	mcp.RegisterHook = func(source string) { traces = append(traces, source) }
	handlers.MCPRegisterHook = func(source string) { traces = append(traces, source) }
	t.Cleanup(func() {
		mcp.RegisterHook = nil
		handlers.MCPRegisterHook = nil
	})

	router := api.SetupRouter(nil, nil, nil, nil, nil, nil, &mockResolver{}, nil, nil)
	t.Logf("MCP-REGISTER traces (%d):", len(traces))
	for _, tr := range traces {
		t.Logf("  %s", tr)
	}
	hasTrace := func(substr string) bool {
		for _, tr := range traces {
			if strings.Contains(tr, substr) {
				return true
			}
		}
		return false
	}
	if !hasTrace("path5") {
		t.Error("Path 5 RegisterMCP/RegisterRoutes never ran (exciting: never-registered)")
	}
	if !hasTrace("path1") {
		t.Error("Path 1 MCPToolHandler.RegisterRoutes wiring never ran")
	}

	routes := collectMCPRoutes(t, router)
	t.Logf("mcp route count=%d", len(routes))
	for _, rt := range routes {
		t.Logf("MCP-ROUTE %s %s", rt.Method, rt.Route)
	}

	has := func(method, substr string) bool {
		for _, rt := range routes {
			if rt.Method == method && strings.Contains(strings.ToLower(rt.Route), strings.ToLower(substr)) {
				return true
			}
		}
		return false
	}

	if !has("POST", "/mcp") {
		t.Error("expected a POST route containing /mcp")
	}
	if !has("POST", "/mcp/tools/call") {
		t.Error("expected POST /api/mcp/tools/call (Path 6)")
	}

	// Path 1 vs Path 5 discriminator on POST /api/mcp.
	// Path 1 (HandleRPC): tools/list is public and returns a tools array;
	//   mcp.list_tools is Method not found (no auth gate on unknown methods).
	// Path 5 (HandleMCPRequest): every method including mcp.list_tools is
	//   auth-gated first (-32001) and does not implement tools/list.
	listRec := postJSON(router, "/api/mcp", `{"jsonrpc":"2.0","id":"1","method":"tools/list"}`)
	t.Logf("POST /api/mcp tools/list status=%d body=%s", listRec.Code, truncate(listRec.Body.String(), 400))

	legacyRec := postJSON(router, "/api/mcp", `{"jsonrpc":"2.0","id":"2","method":"mcp.list_tools"}`)
	t.Logf("POST /api/mcp mcp.list_tools status=%d body=%s", legacyRec.Code, truncate(legacyRec.Body.String(), 400))

	var listResp struct {
		Result map[string]interface{} `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("tools/list decode: %v body=%s", err, listRec.Body.String())
	}
	if listResp.Error != nil {
		t.Fatalf("tools/list RPC error (Path 1 should succeed unauthenticated): %+v", listResp.Error)
	}
	tools, _ := listResp.Result["tools"].([]interface{})
	if len(tools) < 2 {
		t.Fatalf("Path 1 tools/list expected >=2 tools, got %v", listResp.Result)
	}

	var legacyResp struct {
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(legacyRec.Body.Bytes(), &legacyResp); err != nil {
		t.Fatalf("mcp.list_tools decode: %v body=%s", err, legacyRec.Body.String())
	}
	if legacyResp.Error == nil {
		t.Fatal("mcp.list_tools: expected RPC error")
	}
	// Path 1: -32601 method not found. Path 5: -32001 auth required.
	if legacyResp.Error.Code == -32001 {
		t.Fatal("POST /api/mcp is Path 5 (auth gate on mcp.list_tools); expected Path 1 Method not found")
	}
	if legacyResp.Error.Code != -32601 {
		t.Fatalf("mcp.list_tools: expected Path 1 -32601, got %d %s", legacyResp.Error.Code, legacyResp.Error.Message)
	}

	infoRec := getPath(router, "/api/mcp")
	t.Logf("GET /api/mcp status=%d body=%s", infoRec.Code, truncate(infoRec.Body.String(), 300))

	path6 := postJSON(router, "/api/mcp/tools/call", `{"jsonrpc":"2.0","id":"3","method":"tools/call","params":{"name":"x","arguments":{}}}`)
	t.Logf("POST /api/mcp/tools/call status=%d body=%s", path6.Code, truncate(path6.Body.String(), 400))
	if path6.Code == http.StatusNotFound {
		t.Error("POST /api/mcp/tools/call 404; Path 6 missing from mux")
	}

	if got := postJSON(router, "/mcp", `{"jsonrpc":"2.0","id":"4","method":"tools/list"}`); got.Code != http.StatusNotFound {
		t.Errorf("POST /mcp: want 404 (Path 5 is not on the root mux), got %d", got.Code)
	}
	if got := getPath(router, "/api/v1/mcp/tools"); got.Code != http.StatusNotFound {
		t.Errorf("GET /api/v1/mcp/tools: want 404 (Path 4 mux never registered), got %d", got.Code)
	}
	if got := getPath(router, "/mcp/tools"); got.Code != http.StatusNotFound {
		t.Errorf("GET /mcp/tools: want 404, got %d", got.Code)
	}
	if got := getPath(router, "/api/mcp/tools"); got.Code != http.StatusNotFound {
		t.Errorf("GET /api/mcp/tools: want 404, got %d", got.Code)
	}

	// chi.Walk lists the surviving registration only. Path 5 also calls
	// r.Post("/mcp") earlier on the same /api group; Path 1 registers last
	// and wins (see TestChi_DuplicateMethodPattern). Walk therefore shows
	// one POST /api/mcp, not two.
	if len(routes) != 3 {
		t.Errorf("expected exactly 3 mcp routes (GET/POST /api/mcp, POST /api/mcp/tools/call), got %d: %+v", len(routes), routes)
	}
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
