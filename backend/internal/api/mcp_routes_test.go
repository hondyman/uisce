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
		low := strings.ToLower(route)
		if strings.Contains(low, "mcp") || strings.Contains(low, "agentic/proposals") {
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
	t.Cleanup(func() { mcp.RegisterHook = nil })

	router := api.SetupRouter(nil, nil, nil, nil, nil, nil, &mockResolver{}, nil, nil)
	t.Logf("MCP-REGISTER traces (%d): %v", len(traces), traces)

	for _, tr := range traces {
		if strings.Contains(tr, "path5") || strings.Contains(tr, "path1 call site") {
			t.Errorf("dead Path 1/5 registration must not run; got %q", tr)
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
		t.Error("expected POST /api/mcp/tools/call (Path 6 shim)")
	}
	if !has("POST", "/agentic/proposals") {
		t.Error("expected POST /api/agentic/proposals (canonical Path 6)")
	}

	// GET must not be the old Path 1 info descriptor.
	infoRec := getPath(router, "/api/mcp", "application/json")
	body := infoRec.Body.String()
	t.Logf("GET /api/mcp status=%d body=%s", infoRec.Code, truncate(body, 300))
	if strings.Contains(body, `"protocol":"json-rpc-2.0"`) {
		t.Fatal("GET /api/mcp still serves Path 1 info JSON; streamable must own GET")
	}

	assertPath6Auth := func(path string) *httptest.ResponseRecorder {
		rec := postMCPJSON(router, path, `{"jsonrpc":"2.0","id":"3","method":"tools/call","params":{"name":"x","arguments":{}}}`)
		body := rec.Body.String()
		t.Logf("POST %s status=%d body=%s", path, rec.Code, truncate(body, 300))
		if rec.Code == http.StatusNotFound {
			t.Fatalf("POST %s 404", path)
		}
		if !strings.Contains(body, "-32001") || !strings.Contains(body, "auth required") {
			t.Fatalf("%s: expected Path 6 -32001 auth, got %s", path, truncate(body, 400))
		}
		return rec
	}
	canon := assertPath6Auth("/api/agentic/proposals")
	shim := assertPath6Auth("/api/mcp/tools/call")
	if shim.Header().Get("Deprecation") != "true" {
		t.Fatal("shim missing Deprecation: true header")
	}
	if !strings.Contains(shim.Header().Get("Link"), "/api/agentic/proposals") {
		t.Fatalf("shim Link header=%q", shim.Header().Get("Link"))
	}
	_ = canon

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
