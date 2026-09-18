package swift

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAdminServer_AuthGate(t *testing.T) {
	srv := NewAdminServer("127.0.0.1:8982", "test-token")
	mux := http.NewServeMux()
	mux.HandleFunc("/channels", srv.authWrap(srv.handleChannels))
	mux.HandleFunc("/channels/", srv.authWrap(srv.handleChannelAction))

	ts := httptest.NewServer(mux)
	defer ts.Close()

	// 1. Without token -> 401
	resp, err := http.Post(ts.URL+"/channels/cust123/send", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized, got %d", resp.StatusCode)
	}

	// 2. With token -> 200
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/channels/cust123/send", strings.NewReader(`{"msg_type":"MT541","raw":"test"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req.Header.Set("X-Swift-Admin-Token", "test-token")
	req.Header.Set("Content-Type", "application/json")

	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp2.StatusCode)
	}

	// 3. Health check -> 200
	hReq, _ := http.NewRequest(http.MethodGet, ts.URL+"/channels/cust123/health", nil)
	hReq.Header.Set("X-Swift-Admin-Token", "test-token")
	hResp, err := http.DefaultClient.Do(hReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer hResp.Body.Close()
	if hResp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for health, got %d", hResp.StatusCode)
	}
}
