package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWebSocketEndToEndProfiler(t *testing.T) {
	// Setup server and hub
	srv := &Server{WsHub: newWebSocketHub()}
	go srv.WsHub.run()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/ws/token", srv.getWsToken)
	mux.HandleFunc("/ws/profiler/", srv.handleWebSocket)

	// Start httptest server
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// Create a profiling job manually and start it
	jobID := generateJobID()
	job := &ProfileJob{ID: jobID, Status: "pending", CreatedAt: time.Now(), Req: ProfileRequest{Schema: "public", Tables: []string{"table1"}}}
	srv.ProfileJobs.Store(jobID, job)
	go srv.runProfile(jobID)

	// Request a ws token
	tokenReq := map[string]interface{}{"jobId": jobID, "purpose": "profiler", "ttl_seconds": 60}
	b, _ := json.Marshal(tokenReq)
	resp, err := http.Post(ts.URL+"/api/ws/token", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("failed to request token: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("token endpoint returned %d", resp.StatusCode)
	}
	var js map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&js); err != nil {
		t.Fatalf("failed to decode token response: %v", err)
	}
	token := js["token"]
	if token == "" {
		t.Fatalf("no token in response")
	}

	// Dial websocket
	u, _ := url.Parse(ts.URL)
	wsScheme := "ws"
	if u.Scheme == "https" {
		wsScheme = "wss"
	}
	dialURL := fmt.Sprintf("%s://%s/ws/profiler/%s?token=%s", wsScheme, u.Host, jobID, url.QueryEscape(token))
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(dialURL, nil)
	if err != nil {
		t.Fatalf("failed to dial websocket: %v", err)
	}
	defer conn.Close()

	// Read messages until we receive completed
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	gotProgress := false
	for {
		var msg map[string]interface{}
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("error reading ws message: %v", err)
		}
		typeStr, _ := msg["type"].(string)
		if typeStr == "progress" {
			gotProgress = true
		}
		if typeStr == "completed" {
			// ensure results present
			if _, ok := msg["results"]; !ok {
				t.Fatalf("completed message missing results")
			}
			if !gotProgress {
				t.Fatalf("did not receive progress before completed")
			}
			break
		}
	}
}

// (we use bytes.NewReader above)
