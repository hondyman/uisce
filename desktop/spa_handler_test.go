package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestSPAHandler_StaticAssetAndFallback(t *testing.T) {
	mockFS := fstest.MapFS{
		"index.html": {
			Data: []byte("<!DOCTYPE html><html><head><title>Uisce</title></head><body><div id=\"root\"></div></body></html>"),
		},
		"assets/app.js": {
			Data: []byte("console.log('uisce app');"),
		},
		"assets/style.css": {
			Data: []byte("body { background: #000; }"),
		},
	}

	handler := NewSPAHandler(mockFS)

	// 1. Root path "/" should serve index.html
	reqRoot := httptest.NewRequest(http.MethodGet, "/", nil)
	recRoot := httptest.NewRecorder()
	handler.ServeHTTP(recRoot, reqRoot)

	if recRoot.Code != http.StatusOK {
		t.Fatalf("expected 200 for /, got %d", recRoot.Code)
	}
	if !strings.Contains(recRoot.Body.String(), "<title>Uisce</title>") || !strings.Contains(recRoot.Body.String(), "window.go.main.DeskWindowManager") {
		t.Fatalf("expected index.html body with injected bridge, got %s", recRoot.Body.String())
	}

	// 2. Physical static file "/assets/app.js" should serve JS content
	reqJS := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	recJS := httptest.NewRecorder()
	handler.ServeHTTP(recJS, reqJS)

	if recJS.Code != http.StatusOK {
		t.Fatalf("expected 200 for /assets/app.js, got %d", recJS.Code)
	}
	if recJS.Body.String() != "console.log('uisce app');" {
		t.Fatalf("expected JS file content, got %s", recJS.Body.String())
	}

	// 3. SPA Route "/view/page/orders" should fallback to index.html with 200
	reqPopout := httptest.NewRequest(http.MethodGet, "/view/page/orders", nil)
	recPopout := httptest.NewRecorder()
	handler.ServeHTTP(recPopout, reqPopout)

	if recPopout.Code != http.StatusOK {
		t.Fatalf("expected 200 for /view/page/orders, got %d", recPopout.Code)
	}
	if recPopout.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("expected text/html content-type, got %s", recPopout.Header().Get("Content-Type"))
	}
	if !strings.Contains(recPopout.Body.String(), "<div id=\"root\"></div>") {
		t.Fatalf("expected index.html content for SPA route, got %s", recPopout.Body.String())
	}

	// 4. SPA Route "/workspace" should also fallback to index.html with 200
	reqWorkspace := httptest.NewRequest(http.MethodGet, "/workspace", nil)
	recWorkspace := httptest.NewRecorder()
	handler.ServeHTTP(recWorkspace, reqWorkspace)

	if recWorkspace.Code != http.StatusOK {
		t.Fatalf("expected 200 for /workspace, got %d", recWorkspace.Code)
	}
}
