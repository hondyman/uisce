package oauth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/hondyman/uisce/backend/internal/security"
	"golang.org/x/oauth2"
)

// newTestEncryptor returns a deterministic 32-byte AES encryptor for tests.
func newTestEncryptor(t *testing.T) *security.TokenEncryptor {
	t.Helper()
	enc, err := security.NewTokenEncryptor(bytes.Repeat([]byte("t"), 32))
	if err != nil {
		t.Fatalf("NewTokenEncryptor: %v", err)
	}
	return enc
}

// newTestRedis returns a redis.Client pointed at 127.0.0.1:6379 (the default
// dev Redis). Tests skip if Redis is not reachable.
func newTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		addr = "127.0.0.1:6379"
	}
	c := redis.NewClient(&redis.Options{Addr: addr, DialTimeout: 500 * time.Millisecond})
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	if err := c.Ping(ctx).Err(); err != nil {
		t.Skipf("redis unavailable at %s: %v", addr, err)
	}
	return c
}

// uniquePrefix isolates the cache keys for each test run so they do not
// collide across tests sharing the same Redis instance.
func uniquePrefix() string {
	return time.Now().Format("20060102150405.000000")
}

func TestSplitScopes(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"api", []string{"api"}},
		{"api refresh_token", []string{"api", "refresh_token"}},
		{"api,refresh_token", []string{"api", "refresh_token"}},
		{"  api   ,  refresh_token  ", []string{"api", "refresh_token"}},
		{"api\trefresh_token\nopenid", []string{"api", "refresh_token", "openid"}},
	}
	for _, c := range cases {
		got := splitScopes(c.in)
		if !stringSlicesEqual(got, c.want) {
			t.Errorf("splitScopes(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestCacheRoundTrip_SaveThenGet saves a token, fetches it, and verifies
// the round-trip preserves the access token, refresh token, and expiry.
func TestCacheRoundTrip_SaveThenGet(t *testing.T) {
	rdb := newTestRedis(t)
	enc := newTestEncryptor(t)
	p := &ApiOAuthProvider{
		redis:     rdb,
		encryptor: enc,
		keyPrefix: "oauth:api:test:" + uniquePrefix(),
	}

	expiry := time.Now().Add(1 * time.Hour).Truncate(time.Second)
	in := &oauth2.Token{
		AccessToken:  "ya29.access-1",
		RefreshToken: "1//refresh-1",
		TokenType:    "Bearer",
		Expiry:       expiry,
	}
	ctx := context.Background()
	if err := p.SaveToken(ctx, "salesforce", "tenant-a", "ds-x", in); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}
	t.Cleanup(func() {
		_ = p.DeleteCachedToken(ctx, "salesforce", "tenant-a", "ds-x")
	})

	out, err := p.GetCachedToken(ctx, "salesforce", "tenant-a", "ds-x")
	if err != nil {
		t.Fatalf("GetCachedToken: %v", err)
	}
	if out.AccessToken != in.AccessToken {
		t.Errorf("AccessToken = %q, want %q", out.AccessToken, in.AccessToken)
	}
	if out.RefreshToken != in.RefreshToken {
		t.Errorf("RefreshToken = %q, want %q", out.RefreshToken, in.RefreshToken)
	}
	if !out.Expiry.Equal(in.Expiry) {
		t.Errorf("Expiry = %v, want %v", out.Expiry, in.Expiry)
	}
	if !out.Valid() {
		t.Errorf("expected token to be valid; got %+v", out)
	}
}

// TestCacheRoundTrip_ExpiredYieldsMiss verifies that an expired token is
// not returned from the cache (caller should fall back to refresh).
func TestCacheRoundTrip_ExpiredYieldsMiss(t *testing.T) {
	rdb := newTestRedis(t)
	enc := newTestEncryptor(t)
	p := &ApiOAuthProvider{
		redis:     rdb,
		encryptor: enc,
		keyPrefix: "oauth:api:test:" + uniquePrefix(),
	}

	expired := &oauth2.Token{
		AccessToken:  "ya29.expired",
		RefreshToken: "1//refresh",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(-1 * time.Hour),
	}
	ctx := context.Background()
	if err := p.SaveToken(ctx, "servicenow", "tenant-b", "ds-y", expired); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}
	t.Cleanup(func() {
		_ = p.DeleteCachedToken(ctx, "servicenow", "tenant-b", "ds-y")
	})

	_, err := p.GetCachedToken(ctx, "servicenow", "tenant-b", "ds-y")
	if err != ErrNoCachedToken {
		t.Fatalf("expected ErrNoCachedToken for expired token, got %v", err)
	}
}

// TestCacheMiss_OnEmpty confirms GetCachedToken returns ErrNoCachedToken
// when no key has ever been written.
func TestCacheMiss_OnEmpty(t *testing.T) {
	rdb := newTestRedis(t)
	enc := newTestEncryptor(t)
	p := &ApiOAuthProvider{
		redis:     rdb,
		encryptor: enc,
		keyPrefix: "oauth:api:test:" + uniquePrefix(),
	}
	_, err := p.GetCachedToken(context.Background(), "ghost", "tenant-z", "ds-z")
	if err != ErrNoCachedToken {
		t.Fatalf("expected ErrNoCachedToken, got %v", err)
	}
}

// TestCache_NilRedisIsSafe confirms the provider never panics when Redis
// is unavailable (e.g. dev environment with no Redis sidecar).
func TestCache_NilRedisIsSafe(t *testing.T) {
	enc := newTestEncryptor(t)
	p := NewApiOAuthProvider(nil, enc)

	_, err := p.GetCachedToken(context.Background(), "x", "y", "z")
	if err != ErrNoCachedToken {
		t.Fatalf("expected ErrNoCachedToken with nil redis, got %v", err)
	}
	if err := p.SaveToken(context.Background(), "x", "y", "z", &oauth2.Token{AccessToken: "abc", Expiry: time.Now().Add(time.Hour)}); err != nil {
		t.Fatalf("SaveToken with nil redis should be no-op, got %v", err)
	}
	if err := p.DeleteCachedToken(context.Background(), "x", "y", "z"); err != nil {
		t.Fatalf("DeleteCachedToken with nil redis should be no-op, got %v", err)
	}
}

// TestRefresh_RejectsIncompleteCredentials verifies that RefreshWithConfig
// fails fast when any required field is missing, without making a network
// call.
func TestRefresh_RejectsIncompleteCredentials(t *testing.T) {
	enc := newTestEncryptor(t)
	p := NewApiOAuthProvider(nil, enc)
	cases := []TokenCredentials{
		{},                                                              // everything empty
		{ClientID: "client", RefreshToken: "rt", TokenURL: ""},          // missing token URL
		{ClientID: "client", TokenURL: "https://example.com/t", RefreshToken: ""}, // missing refresh
		{RefreshToken: "rt", TokenURL: "https://example.com/t"},         // missing client_id
	}
	for i, c := range cases {
		if _, err := p.RefreshWithConfig(context.Background(), c); err == nil {
			t.Errorf("case %d: expected error for incomplete creds %+v", i, c)
		}
	}
}

// TestRefresh_EndToEnd spins up a fake OAuth token endpoint, calls
// RefreshWithConfig against it, and verifies the returned token is correctly
// parsed. This is the only test that exercises the real oauth2 package.
func TestRefresh_EndToEnd(t *testing.T) {
	enc := newTestEncryptor(t)
	p := NewApiOAuthProvider(nil, enc)

	// Fake Salesforce-style token endpoint.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if r.Form.Get("grant_type") != "refresh_token" {
			http.Error(w, "unexpected grant_type", http.StatusBadRequest)
			return
		}
		if r.Form.Get("refresh_token") != "old-rt" {
			http.Error(w, "unexpected refresh_token", http.StatusBadRequest)
			return
		}
		// Note: Salesforce does NOT return a new refresh_token; provider must
		// preserve the original.
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "new-at",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer srv.Close()

	creds := TokenCredentials{
		ClientID:     "client",
		ClientSecret: "secret",
		RefreshToken: "old-rt",
		TokenURL:     srv.URL,
		Scopes:       "api refresh_token",
	}

	tok, err := p.RefreshWithConfig(context.Background(), creds)
	if err != nil {
		t.Fatalf("RefreshWithConfig: %v", err)
	}
	if tok.AccessToken != "new-at" {
		t.Errorf("AccessToken = %q, want new-at", tok.AccessToken)
	}
	if tok.RefreshToken != "old-rt" {
		t.Errorf("RefreshToken = %q, want old-rt (provider should preserve original)", tok.RefreshToken)
	}
	if tok.TokenType != "Bearer" {
		t.Errorf("TokenType = %q, want Bearer", tok.TokenType)
	}
	if !tok.Valid() {
		t.Errorf("expected token to be valid; got %+v", tok)
	}
}

// TestRefresh_ReturnsNewRefreshTokenWhenProvided verifies that providers
// (e.g. Google) which DO return a new refresh_token in the response keep
// the new one instead of the old.
func TestRefresh_ReturnsNewRefreshTokenWhenProvided(t *testing.T) {
	enc := newTestEncryptor(t)
	p := NewApiOAuthProvider(nil, enc)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "new-at",
			"refresh_token": "new-rt",
			"token_type":    "Bearer",
			"expires_in":    3600,
		})
	}))
	defer srv.Close()

	creds := TokenCredentials{
		ClientID:     "client",
		ClientSecret: "secret",
		RefreshToken: "old-rt",
		TokenURL:     srv.URL,
	}
	tok, err := p.RefreshWithConfig(context.Background(), creds)
	if err != nil {
		t.Fatalf("RefreshWithConfig: %v", err)
	}
	if tok.RefreshToken != "new-rt" {
		t.Errorf("RefreshToken = %q, want new-rt (provider should keep the response's value when non-empty)", tok.RefreshToken)
	}
}

// TestCache_DeleteIsIdempotent verifies that DeleteCachedToken on a
// non-existent key is a no-op rather than an error.
func TestCache_DeleteIsIdempotent(t *testing.T) {
	rdb := newTestRedis(t)
	enc := newTestEncryptor(t)
	p := &ApiOAuthProvider{
		redis:     rdb,
		encryptor: enc,
		keyPrefix: "oauth:api:test:" + uniquePrefix(),
	}
	if err := p.DeleteCachedToken(context.Background(), "ghost", "tenant", "ds"); err != nil {
		t.Fatalf("expected no error deleting non-existent key, got %v", err)
	}
}
