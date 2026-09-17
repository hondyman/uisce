package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"github.com/hondyman/uisce/backend/internal/auth"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
)

func createTestJWT(t *testing.T, tenantID, userID string) string {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "test-jwt-secret-key-1234567890"
		os.Setenv("JWT_SECRET", secret)
	}

	claims := &jwtmiddleware.JWTClaims{
		UserID:   userID,
		TenantID: tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to sign test JWT: %v", err)
	}
	return tokenString
}

func setupTestWsServer(t *testing.T, vaultCfg auth.WsTicketVaultConfig) (*Server, *httptest.Server) {
	vault := auth.NewWsTicketVault(vaultCfg)
	srv := &Server{
		WsHub:   newWebSocketHub(),
		WsVault: vault,
	}
	go srv.WsHub.run()

	r := chi.NewRouter()
	r.Post("/api/ws/ticket", srv.issueWsTicket)
	r.Get("/api/ws", srv.handleWebSocketTicketAndUpgrade)
	r.Get("/ws", srv.handleWebSocketTicketAndUpgrade)

	ts := httptest.NewServer(r)
	return srv, ts
}

// 1. Unauthenticated ticket issuance -> 401
func TestWsTicket_IssueUnauthenticated(t *testing.T) {
	srv, ts := setupTestWsServer(t, auth.DefaultWsTicketVaultConfig())
	defer ts.Close()
	defer srv.WsVault.Close()

	req, _ := http.NewRequest("POST", ts.URL+"/api/ws/ticket", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", resp.StatusCode)
	}

	var body map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body["error"] != "unauthorized" {
		t.Fatalf("expected error='unauthorized', got %v", body)
	}
}

// 2. Authenticated ticket issuance -> 200 with ticket and expires_in: 30
func TestWsTicket_IssueAuthenticated(t *testing.T) {
	srv, ts := setupTestWsServer(t, auth.DefaultWsTicketVaultConfig())
	defer ts.Close()
	defer srv.WsVault.Close()

	jwtToken := createTestJWT(t, "tenant-alpha", "user-123")

	req, _ := http.NewRequest("POST", ts.URL+"/api/ws/ticket", nil)
	req.Header.Set("Authorization", "Bearer "+jwtToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var body IssueTicketResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body.Ticket == "" {
		t.Fatalf("expected non-empty ticket")
	}
	if body.ExpiresIn != 30 {
		t.Fatalf("expected expires_in=30, got %d", body.ExpiresIn)
	}
}

// 3. Ticket rate limiting (10 per 15s) -> 11th yields 429
func TestWsTicket_RateLimiting(t *testing.T) {
	cfg := auth.DefaultWsTicketVaultConfig()
	cfg.RateLimitMax = 5
	srv, ts := setupTestWsServer(t, cfg)
	defer ts.Close()
	defer srv.WsVault.Close()

	jwtToken := createTestJWT(t, "tenant-rate", "user-rate")

	for i := 0; i < 5; i++ {
		req, _ := http.NewRequest("POST", ts.URL+"/api/ws/ticket", nil)
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("iteration %d failed: %v", i, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("iteration %d expected 200, got %d", i, resp.StatusCode)
		}
	}

	// 6th ticket should return 429
	req, _ := http.NewRequest("POST", ts.URL+"/api/ws/ticket", nil)
	req.Header.Set("Authorization", "Bearer "+jwtToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("rate limit request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected 429 Too Many Requests, got %d", resp.StatusCode)
	}
}

// 4. Hostile origin check: invalid origin fails without burning ticket
func TestWsTicket_HostileOriginDoesNotBurnTicket(t *testing.T) {
	srv, ts := setupTestWsServer(t, auth.DefaultWsTicketVaultConfig())
	defer ts.Close()
	defer srv.WsVault.Close()

	// Set ENVIRONMENT=production so origin validation is strictly enforced
	os.Setenv("ENVIRONMENT", "production")
	defer os.Setenv("ENVIRONMENT", "development")

	// Issue a valid ticket
	ticket, _, err := srv.WsVault.IssueTicket("tenant-prod", "user-prod")
	if err != nil {
		t.Fatalf("failed to issue ticket: %v", err)
	}

	// Connect with hostile origin (should fail CheckOrigin)
	u, _ := url.Parse(ts.URL)
	wsURL := fmt.Sprintf("ws://%s/api/ws?ticket=%s", u.Host, ticket)

	header := http.Header{}
	header.Set("Origin", "http://evil-attacker-origin.com")

	dialer := websocket.Dialer{}
	_, resp, err := dialer.Dial(wsURL, header)
	if err == nil {
		t.Fatalf("expected dial to fail due to hostile origin")
	}
	if resp != nil && resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for bad origin, got %d", resp.StatusCode)
	}

	// CRITICAL INVARIANT: The ticket was NOT burned because origin check failed first!
	// Now attempt connect with legitimate origin (matching host)
	validHeader := http.Header{}
	validHeader.Set("Origin", fmt.Sprintf("http://%s", u.Host))

	conn, validResp, err := dialer.Dial(wsURL, validHeader)
	if err != nil {
		t.Fatalf("ticket was improperly burned by hostile probe! valid dial failed: %v, resp: %v", err, validResp)
	}
	defer conn.Close()

	// Now that it succeeded once, a second connect with the same ticket MUST fail (single-use)
	_, secondResp, err := dialer.Dial(wsURL, validHeader)
	if err == nil {
		t.Fatalf("expected second connect with consumed ticket to fail")
	}
	if secondResp == nil || secondResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for replayed ticket, got %v", secondResp)
	}
}

// 5. No-Oracle rule: invalid, missing, expired, and already consumed tickets all return identical 401
func TestWsTicket_NoOracleResponses(t *testing.T) {
	cfg := auth.DefaultWsTicketVaultConfig()
	cfg.TTL = 50 * time.Millisecond
	cfg.SkewGrace = 10 * time.Millisecond
	srv, ts := setupTestWsServer(t, cfg)
	defer ts.Close()
	defer srv.WsVault.Close()

	u, _ := url.Parse(ts.URL)
	dialer := websocket.Dialer{}

	check401Body := func(name string, wsURL string) {
		t.Helper()
		_, resp, err := dialer.Dial(wsURL, nil)
		if err == nil {
			t.Fatalf("[%s] expected dial to fail", name)
		}
		if resp == nil {
			t.Fatalf("[%s] nil response", name)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("[%s] expected 401 Unauthorized, got %d", name, resp.StatusCode)
		}
		var body map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&body)
		if body["error"] != "unauthorized" {
			t.Fatalf("[%s] expected identical body error='unauthorized', got %v", name, body)
		}
	}

	// Missing ticket
	check401Body("missing_ticket", fmt.Sprintf("ws://%s/api/ws", u.Host))

	// Invalid ticket (syntactically random / non-existent)
	check401Body("non_existent_ticket", fmt.Sprintf("ws://%s/api/ws?ticket=completely-bogus-ticket-value", u.Host))

	// Expired ticket
	expiredTicket, _, _ := srv.WsVault.IssueTicket("tenant-exp", "user-exp")
	time.Sleep(100 * time.Millisecond) // exceeds TTL + skew grace
	check401Body("expired_ticket", fmt.Sprintf("ws://%s/api/ws?ticket=%s", u.Host, expiredTicket))

	// Consumed ticket
	consumedTicket, _, _ := srv.WsVault.IssueTicket("tenant-used", "user-used")
	conn, _, err := dialer.Dial(fmt.Sprintf("ws://%s/api/ws?ticket=%s", u.Host, consumedTicket), nil)
	if err != nil {
		t.Fatalf("first dial failed: %v", err)
	}
	conn.Close()
	// Replay same ticket
	check401Body("replayed_ticket", fmt.Sprintf("ws://%s/api/ws?ticket=%s", u.Host, consumedTicket))
}

// 6. Legacy JWT compatibility flag (WS_ALLOW_LEGACY_JWT)
func TestWsTicket_LegacyJWTOption(t *testing.T) {
	srv, ts := setupTestWsServer(t, auth.DefaultWsTicketVaultConfig())
	defer ts.Close()
	defer srv.WsVault.Close()

	jwtToken := createTestJWT(t, "tenant-legacy", "user-legacy")
	u, _ := url.Parse(ts.URL)
	dialer := websocket.Dialer{}

	// Case A: WS_ALLOW_LEGACY_JWT is true (default)
	os.Setenv("WS_ALLOW_LEGACY_JWT", "true")
	legacyURL := fmt.Sprintf("ws://%s/api/ws?token=%s", u.Host, jwtToken)
	conn, resp, err := dialer.Dial(legacyURL, nil)
	if err != nil {
		t.Fatalf("legacy ?token= should succeed when WS_ALLOW_LEGACY_JWT=true: %v, resp: %v", err, resp)
	}
	conn.Close()

	// Case B: WS_ALLOW_LEGACY_JWT is false -> rejected with 401
	os.Setenv("WS_ALLOW_LEGACY_JWT", "false")
	defer os.Unsetenv("WS_ALLOW_LEGACY_JWT")

	_, respDisallowed, err := dialer.Dial(legacyURL, nil)
	if err == nil {
		t.Fatalf("legacy ?token= should fail when WS_ALLOW_LEGACY_JWT=false")
	}
	if respDisallowed == nil || respDisallowed.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 when legacy jwt disabled, got %v", respDisallowed)
	}
}
