package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hondyman/uisce/backend/internal/auth"
)

// dialWithTicket issues a fresh ticket for (tenantID, userID) and dials the
// streaming upgrade endpoint with a same-origin header, returning the live
// connection. Mirrors the valid-dial path in ws_ticket_test.go.
func dialWithTicket(t *testing.T, srv *Server, ts *httptest.Server, tenantID, userID string) *websocket.Conn {
	t.Helper()
	ticket, _, err := srv.WsVault.IssueTicket(tenantID, userID)
	if err != nil {
		t.Fatalf("failed to issue ticket: %v", err)
	}
	u, _ := url.Parse(ts.URL)
	wsURL := fmt.Sprintf("ws://%s/api/ws?ticket=%s", u.Host, ticket)
	header := http.Header{}
	header.Set("Origin", fmt.Sprintf("http://%s", u.Host))
	conn, resp, err := (&websocket.Dialer{}).Dial(wsURL, header)
	if err != nil {
		t.Fatalf("dial failed: %v (resp=%v)", err, resp)
	}
	return conn
}

// TestStreamingSubscribe_ConnectionLevelAuth proves the security property
// slice 1 actually exercises: no channel has a per-key ownership check yet
// (ChannelPrices is allow-all), so the property to prove is connection-level
// — an invalid/missing ticket must never reach the point of accepting a
// subscribe message at all. This validates the reused ticket-auth plumbing
// end to end before any tenant-owned channel is added on top of it.
func TestStreamingSubscribe_ConnectionLevelAuth(t *testing.T) {
	srv, ts := setupTestWsServer(t, auth.DefaultWsTicketVaultConfig())
	defer ts.Close()
	defer srv.WsVault.Close()

	u, _ := url.Parse(ts.URL)
	// No ?ticket= at all.
	wsURL := fmt.Sprintf("ws://%s/api/ws", u.Host)
	header := http.Header{}
	header.Set("Origin", fmt.Sprintf("http://%s", u.Host))

	_, resp, err := (&websocket.Dialer{}).Dial(wsURL, header)
	if err == nil {
		t.Fatalf("expected dial without ticket to fail")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing ticket, got %v", resp)
	}
}

// TestStreamingSubscribe_PricesEndToEnd proves the happy path: an
// authenticated connection can subscribe to ChannelPrices and receives a
// SubscriptionID it can later unsubscribe with.
func TestStreamingSubscribe_PricesEndToEnd(t *testing.T) {
	srv, ts := setupTestWsServer(t, auth.DefaultWsTicketVaultConfig())
	defer ts.Close()
	defer srv.WsVault.Close()

	conn := dialWithTicket(t, srv, ts, "tenant-a", "user-a")
	defer conn.Close()

	req := streamActionRequest{Action: streamActionSubscribe, Channel: ChannelPrices, Keys: []string{"AAPL", "MSFT"}}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("write subscribe: %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var resp streamActionResponse
	if err := conn.ReadJSON(&resp); err != nil {
		t.Fatalf("read subscribe response: %v", err)
	}
	if resp.Error != "" {
		t.Fatalf("expected successful subscribe, got error: %s", resp.Error)
	}
	if resp.SubscriptionID == "" {
		t.Fatalf("expected non-empty subscription id")
	}

	unsubReq := streamActionRequest{Action: streamActionUnsubscribe, SubscriptionID: resp.SubscriptionID}
	if err := conn.WriteJSON(unsubReq); err != nil {
		t.Fatalf("write unsubscribe: %v", err)
	}
	var unsubResp streamActionResponse
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if err := conn.ReadJSON(&unsubResp); err != nil {
		t.Fatalf("read unsubscribe response: %v", err)
	}
	if unsubResp.Error != "" {
		t.Fatalf("expected successful unsubscribe, got error: %s", unsubResp.Error)
	}
}

// TestStreamingSubscribe_LargeKeyListDoesNotBreakReadLimit is the regression
// test for the bug the review specifically flagged: the original 512-byte
// SetReadLimit would fail any subscribe message carrying a realistic number
// of instrument keys (a blotter grid subscribes per-instrument, not
// per-row), killing the connection instead of returning a clean response.
func TestStreamingSubscribe_LargeKeyListDoesNotBreakReadLimit(t *testing.T) {
	srv, ts := setupTestWsServer(t, auth.DefaultWsTicketVaultConfig())
	defer ts.Close()
	defer srv.WsVault.Close()

	conn := dialWithTicket(t, srv, ts, "tenant-a", "user-a")
	defer conn.Close()

	keys := make([]string, 500)
	for i := range keys {
		keys[i] = fmt.Sprintf("INSTRUMENT-%04d", i)
	}
	req := streamActionRequest{Action: streamActionSubscribe, Channel: ChannelPrices, Keys: keys}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("write subscribe: %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var resp streamActionResponse
	if err := conn.ReadJSON(&resp); err != nil {
		t.Fatalf("read subscribe response (500 keys should not exceed read limit): %v", err)
	}
	if resp.Error != "" {
		t.Fatalf("expected successful subscribe with 500 keys, got error: %s", resp.Error)
	}
	if len(resp.Keys) != 500 {
		t.Fatalf("expected 500 keys echoed back, got %d", len(resp.Keys))
	}
}

// TestStreamingSubscribe_UnknownChannelDenied checks the allow-list gate and
// the no-oracle response shape (generic "subscription denied", not a
// channel-specific message).
func TestStreamingSubscribe_UnknownChannelDenied(t *testing.T) {
	srv, ts := setupTestWsServer(t, auth.DefaultWsTicketVaultConfig())
	defer ts.Close()
	defer srv.WsVault.Close()

	conn := dialWithTicket(t, srv, ts, "tenant-a", "user-a")
	defer conn.Close()

	req := streamActionRequest{Action: streamActionSubscribe, Channel: StreamChannel("order-updates"), Keys: []string{"ORD-1"}}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("write subscribe: %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var resp streamActionResponse
	if err := conn.ReadJSON(&resp); err != nil {
		t.Fatalf("read subscribe response: %v", err)
	}
	if resp.Error != "subscription denied" {
		t.Fatalf("expected generic 'subscription denied' for unregistered channel, got %q", resp.Error)
	}
}

// TestFanOutPriceUpdate_DropsWhenSendBufferFull pins the latest-value-wins
// contract server-side: a slow/blocked consumer must not cause the hub to
// block or queue unboundedly. A superseded tick is expected to be dropped,
// matching what the client-side ring buffer/quote table already assume.
func TestFanOutPriceUpdate_DropsWhenSendBufferFull(t *testing.T) {
	hub := newWebSocketHub()
	go hub.run()

	client := &WebSocketClient{
		conn: nil, // not needed: this test only exercises sendJSON's channel write
		send: make(chan []byte, 1),
		hub:  hub,
	}

	sub := &subscription{
		id:      nextSubscriptionID(),
		client:  client,
		channel: ChannelPrices,
		keys:    map[string]struct{}{"AAPL": {}},
	}
	client.subscriptions = map[SubscriptionID]*subscription{sub.id: sub}
	hub.subIndex[channelKey{channel: ChannelPrices, key: "AAPL"}] = map[SubscriptionID]*subscription{sub.id: sub}

	// Fill the send buffer (capacity 1) so the next fan-out must drop.
	client.send <- []byte(`{"stale":"tick"}`)

	hub.FanOutPriceUpdate("AAPL", map[string]float64{"price": 101.5})

	// The buffer should still hold exactly the original stale message —
	// proof the new tick was dropped rather than blocking or growing the
	// channel.
	select {
	case msg := <-client.send:
		var decoded map[string]string
		_ = json.Unmarshal(msg, &decoded)
		if decoded["stale"] != "tick" {
			t.Fatalf("expected the pre-existing stale message to remain, got: %s", msg)
		}
	default:
		t.Fatalf("expected the send buffer to still contain the original message")
	}

	select {
	case <-client.send:
		t.Fatalf("expected no second message queued — fan-out should have dropped it")
	default:
	}
}
