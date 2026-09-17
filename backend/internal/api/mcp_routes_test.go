package api_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/api"
	"github.com/hondyman/uisce/backend/internal/handlers"
	"github.com/hondyman/uisce/backend/internal/mcp"
	"github.com/hondyman/uisce/backend/internal/middleware"
	"github.com/hondyman/uisce/backend/internal/services"
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

	// GET must not be the old Path 1 info descriptor.
	infoRec := getPath(router, "/api/mcp", "application/json")
	body := infoRec.Body.String()
	t.Logf("GET /api/mcp status=%d body=%s", infoRec.Code, truncate(body, 300))
	if strings.Contains(body, `"protocol":"json-rpc-2.0"`) {
		t.Fatal("GET /api/mcp still serves Path 1 info JSON; streamable must own GET")
	}

	// Path 6 subpath must NOT be swallowed by streamable /mcp.
	// Discriminator: unauthenticated Path 6 returns JSON-RPC -32001 with
	// the maker-checker auth message — not a streamable MCP parse/session error.
	path6 := postMCPJSON(router, "/api/mcp/tools/call", `{"jsonrpc":"2.0","id":"3","method":"tools/call","params":{"name":"x","arguments":{}}}`)
	p6body := path6.Body.String()
	t.Logf("POST /api/mcp/tools/call status=%d body=%s", path6.Code, truncate(p6body, 300))
	if path6.Code == http.StatusNotFound {
		t.Fatal("POST /api/mcp/tools/call 404; Path 6 missing (streamable may have swallowed subpath)")
	}
	if !strings.Contains(p6body, "-32001") || !strings.Contains(p6body, "auth required") {
		t.Fatalf("Path 6 discriminator failed (expected -32001 auth); streamable may own /mcp/tools/call: %s", truncate(p6body, 400))
	}
	if !has("POST", "/mcp/tools/call") {
		t.Error("Walk missing POST /api/mcp/tools/call while streamable owns /api/mcp")
	}

	if got := postMCPJSON(router, "/mcp", `{"jsonrpc":"2.0","id":4,"method":"tools/list"}`); got.Code != http.StatusNotFound {
		t.Errorf("POST /mcp: want 404, got %d", got.Code)
	}
}

// TestMCP_InitializeCapableProbe is the harness form of the flip-checklist
// live probe: mcp-go client initialize → tools/list → tools/call → refuse.
// Uses a minimal mux (not full SetupRouter) so nil-DB region middleware
// cannot panic under a tenant-bearing JWT.
func TestMCP_InitializeCapableProbe(t *testing.T) {
	const secret = "initialize-probe-secret"
	sm := services.NewSecurityManager(nil, nil, []byte(secret))
	token, err := sm.MintDevToken(services.DevTokenInput{
		UserID:    "init-probe",
		TenantIDs: []string{"99e99e99-99e9-49e9-89e9-99e99e99e999"},
		Roles:     []string{"portfolio_manager"},
	})
	if err != nil {
		t.Fatalf("MintDevToken: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/api/mcp", middleware.AuthContextMiddleware(sm)(mcp.NewServer(nil).HTTPHandler()))
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	probe, err := mcp.ProbeStreamable(ctx, ts.URL+"/api/mcp", token)
	if err != nil {
		t.Fatalf("ProbeStreamable: %v", err)
	}
	if probe.ToolCount < 7 {
		t.Fatalf("tools=%d", probe.ToolCount)
	}
	if !strings.Contains(probe.CallText, "99e99e99-99e9-49e9-89e9-99e99e99e999") {
		t.Fatalf("call missing tenant_id field: %s", truncate(probe.CallText, 300))
	}
	refused := probe.RefuseIsErr || strings.Contains(probe.RefuseText, "refused") || strings.Contains(probe.RefuseText, "unknown")
	if !refused {
		t.Fatalf("expected refuse, got %q", probe.RefuseText)
	}
	t.Logf("initialize-capable probe ok session=%s tools=%d", mcp.SessionMode, probe.ToolCount)
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
