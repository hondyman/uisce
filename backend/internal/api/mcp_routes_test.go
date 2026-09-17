package api_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/api"
	"github.com/hondyman/uisce/backend/internal/handlers"
	"github.com/hondyman/uisce/backend/internal/mcp"
)

// TestChi_DuplicateMethodPattern records chi v5.2.3 last-wins.
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
	if rec.Body.String() != "second" {
		t.Fatalf("chi v5.2.3 last-registration-wins expected %q, got %q", "second", rec.Body.String())
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
		if strings.Contains(strings.ToLower(route), "mcp") {
			out = append(out, mcpRoute{Method: method, Route: route})
		}
		return nil
	}); err != nil {
		t.Fatalf("chi.Walk: %v", err)
	}
	return out
}

func postMCPJSON(handler http.Handler, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func getPath(handler http.Handler, path string, accept string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", path, nil)
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// TestMCP_RouteTableDump is the structural guard for the verb-complete
// streamable cutover. Flip checklist: this test + stderr ROUTES-DUMP +
// admin GET /_routes + strings(binary)|grep CutoverMarker.
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
	t.Logf("MCP-REGISTER traces (%d): %v", len(traces), traces)

	for _, tr := range traces {
		if strings.Contains(tr, "path5") {
			t.Errorf("Path 5 must not register after cutover; got trace %q", tr)
		}
		if strings.Contains(tr, "path1 call site") {
			t.Errorf("Path 1 RegisterRoutes must not run after cutover; got trace %q", tr)
		}
	}
	hasStreamable := false
	for _, tr := range traces {
		if strings.Contains(tr, "streamable") {
			hasStreamable = true
		}
	}
	if !hasStreamable {
		t.Error("expected streamable call-site trace")
	}

	routes := collectMCPRoutes(t, router)
	for _, rt := range routes {
		t.Logf("MCP-ROUTE %s %s", rt.Method, rt.Route)
	}

	has := func(method, substr string) bool {
		for _, rt := range routes {
			if (method == "" || rt.Method == method) && strings.Contains(strings.ToLower(rt.Route), strings.ToLower(substr)) {
				return true
			}
		}
		return false
	}
	if !has("", "/mcp") {
		t.Error("expected /api/mcp on the mux")
	}
	if !has("POST", "/mcp/tools/call") {
		t.Error("expected POST /api/mcp/tools/call (Path 6)")
	}

	// Streamable tools/list (stateless).
	listRec := postMCPJSON(router, "/api/mcp", `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	t.Logf("POST /api/mcp tools/list status=%d body=%s", listRec.Code, truncate(listRec.Body.String(), 400))
	if listRec.Code != http.StatusOK && listRec.Code != http.StatusAccepted {
		t.Fatalf("tools/list status=%d body=%s", listRec.Code, listRec.Body.String())
	}
	if !strings.Contains(listRec.Body.String(), "get_business_object_contract") &&
		!strings.Contains(listRec.Body.String(), `"tools"`) {
		t.Fatalf("tools/list missing catalog: %s", truncate(listRec.Body.String(), 500))
	}

	// Path 5 protocol must not be the face.
	legacyRec := postMCPJSON(router, "/api/mcp", `{"jsonrpc":"2.0","id":2,"method":"mcp.list_tools"}`)
	t.Logf("POST /api/mcp mcp.list_tools status=%d body=%s", legacyRec.Code, truncate(legacyRec.Body.String(), 300))
	if strings.Contains(legacyRec.Body.String(), `"name":"get_node_schema"`) {
		t.Fatal("Path 5 tool catalog leaked onto /api/mcp")
	}

	// GET must not be the old Path 1 info descriptor.
	infoRec := getPath(router, "/api/mcp", "application/json")
	body := infoRec.Body.String()
	t.Logf("GET /api/mcp status=%d body=%s", infoRec.Code, truncate(body, 300))
	if strings.Contains(body, `"protocol":"json-rpc-2.0"`) {
		t.Fatal("GET /api/mcp still serves Path 1 info JSON; streamable must own GET")
	}

	path6 := postMCPJSON(router, "/api/mcp/tools/call", `{"jsonrpc":"2.0","id":"3","method":"tools/call","params":{"name":"x","arguments":{}}}`)
	if path6.Code == http.StatusNotFound {
		t.Error("POST /api/mcp/tools/call 404; Path 6 missing")
	}

	if got := postMCPJSON(router, "/mcp", `{"jsonrpc":"2.0","id":4,"method":"tools/list"}`); got.Code != http.StatusNotFound {
		t.Errorf("POST /mcp: want 404, got %d", got.Code)
	}
}

func TestMCP_CutoverMarkerPresent(t *testing.T) {
	if mcp.CutoverMarker != "mcp-cutover-streamable-v1" {
		t.Fatalf("CutoverMarker drifted: %q", mcp.CutoverMarker)
	}
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
