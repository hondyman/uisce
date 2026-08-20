package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/oauth2"
)

var (
	apiTokenSaveErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "api_oauth_token_save_errors_total",
		Help: "Total number of API OAuth token save errors",
	})
	apiTokenRefreshErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "api_oauth_token_refresh_errors_total",
		Help: "Total number of API OAuth token refresh errors",
	})
	apiTokenCacheHits = promauto.NewCounter(prometheus.CounterOpts{
		Name: "api_oauth_token_cache_hits_total",
		Help: "Total number of API OAuth token cache hits",
	})
	apiTokenCacheMisses = promauto.NewCounter(prometheus.CounterOpts{
		Name: "api_oauth_token_cache_misses_total",
		Help: "Total number of API OAuth token cache misses (cache empty or expired)",
	})
)

// ErrNoCachedToken signals that the cache does not contain a token for the
// given (service, tenant, datasource) tuple. The dispatcher should fall back
// to using the stored refresh_token to mint a new one.
var ErrNoCachedToken = errors.New("api oauth: no cached token")

// TokenCredentials is the set of per-tenant OAuth inputs the dispatcher needs
// to refresh an access token. It is loaded from tenant_api_connections rows
// by the dispatcher and passed to RefreshWithConfig.
type TokenCredentials struct {
	ClientID     string
	ClientSecret string
	RefreshToken string
	TokenURL     string
	Scopes       string
}

// ApiOAuthProvider stores and refreshes OAuth access tokens for the API
// dispatcher. Tokens are keyed by (serviceType, tenantID, datasourceID) and
// encrypted at rest in Redis using the same TokenEncryptor as the database
// columns. The provider never holds long-lived secrets (client_secret,
// refresh_token); those live in the tenant_api_connections row and are
// passed in by the dispatcher at call time.
type ApiOAuthProvider struct {
	redis     *redis.Client
	encryptor *security.TokenEncryptor
	keyPrefix string
}

// NewApiOAuthProvider builds a provider. If redis is nil, all cache methods
// become no-ops and return ErrNoCachedToken so the dispatcher can still run
// using only the static/refresh flow.
func NewApiOAuthProvider(redisClient *redis.Client, encryptor *security.TokenEncryptor) *ApiOAuthProvider {
	return &ApiOAuthProvider{
		redis:     redisClient,
		encryptor: encryptor,
		keyPrefix: "oauth:api",
	}
}

// cacheKey produces the Redis key for a given (service, tenant, datasource).
func (p *ApiOAuthProvider) cacheKey(serviceType, tenantID, datasourceID string) string {
	return fmt.Sprintf("%s:%s:%s:%s", p.keyPrefix, serviceType, tenantID, datasourceID)
}

// GetCachedToken returns the cached access token if present and not expired.
//   - (nil, ErrNoCachedToken) when no entry exists or the entry cannot be
//     decrypted (cache miss).
//   - (nil, other error) on a Redis or marshal failure (cache miss is
//     preferable to failing the dispatch).
//   - (token, nil) when a valid unexpired token is returned.
func (p *ApiOAuthProvider) GetCachedToken(ctx context.Context, serviceType, tenantID, datasourceID string) (*oauth2.Token, error) {
	if p.redis == nil || p.encryptor == nil {
		apiTokenCacheMisses.Inc()
		return nil, ErrNoCachedToken
	}
	raw, err := p.redis.Get(ctx, p.cacheKey(serviceType, tenantID, datasourceID)).Bytes()
	if err == redis.Nil {
		apiTokenCacheMisses.Inc()
		return nil, ErrNoCachedToken
	}
	if err != nil {
		apiTokenCacheMisses.Inc()
		return nil, fmt.Errorf("redis get: %w", err)
	}
	plaintext, err := p.encryptor.Decrypt(string(raw))
	if err != nil {
		apiTokenCacheMisses.Inc()
		return nil, fmt.Errorf("decrypt cached token: %w", err)
	}
	var blob storedToken
	if err := json.Unmarshal([]byte(plaintext), &blob); err != nil {
		apiTokenCacheMisses.Inc()
		return nil, fmt.Errorf("parse cached token: %w", err)
	}
	tok := blob.toOAuth2()
	if !tok.Valid() {
		apiTokenCacheMisses.Inc()
		return nil, ErrNoCachedToken
	}
	apiTokenCacheHits.Inc()
	return tok, nil
}

// SaveToken encrypts and stores the access token (and refresh token if
// provided). TTL is set to the time until the token expires plus a 5-minute
// safety window; if the token has no expiry, a 1-hour TTL is used.
func (p *ApiOAuthProvider) SaveToken(ctx context.Context, serviceType, tenantID, datasourceID string, tok *oauth2.Token) error {
	if p.redis == nil || p.encryptor == nil {
		return nil
	}
	if tok == nil || tok.AccessToken == "" {
		return fmt.Errorf("cannot save nil or empty access token")
	}
	blob := fromOAuth2(tok)
	plaintext, err := json.Marshal(blob)
	if err != nil {
		apiTokenSaveErrors.Inc()
		return fmt.Errorf("marshal token: %w", err)
	}
	ciphertext, err := p.encryptor.Encrypt(string(plaintext))
	if err != nil {
		apiTokenSaveErrors.Inc()
		return fmt.Errorf("encrypt token: %w", err)
	}
	ttl := 1 * time.Hour
	if !tok.Expiry.IsZero() {
		remaining := time.Until(tok.Expiry)
		if remaining > 0 {
			ttl = remaining + 5*time.Minute
		}
	}
	if err := p.redis.Set(ctx, p.cacheKey(serviceType, tenantID, datasourceID), ciphertext, ttl).Err(); err != nil {
		apiTokenSaveErrors.Inc()
		return fmt.Errorf("redis set: %w", err)
	}
	return nil
}

// DeleteCachedToken removes any cached token for the tuple. Used when a
// tenant revokes their connection or rotates their refresh token.
func (p *ApiOAuthProvider) DeleteCachedToken(ctx context.Context, serviceType, tenantID, datasourceID string) error {
	if p.redis == nil {
		return nil
	}
	return p.redis.Del(ctx, p.cacheKey(serviceType, tenantID, datasourceID)).Err()
}

// RefreshWithConfig uses the supplied client credentials and refresh token to
// mint a fresh oauth2.Token from the provider's token URL. The result is
// returned to the caller (who should then call SaveToken to populate the
// cache) and is never persisted by this method.
func (p *ApiOAuthProvider) RefreshWithConfig(ctx context.Context, creds TokenCredentials) (*oauth2.Token, error) {
	if creds.ClientID == "" {
		return nil, fmt.Errorf("refresh: client_id is required")
	}
	if creds.TokenURL == "" {
		return nil, fmt.Errorf("refresh: token_url is required")
	}
	if creds.RefreshToken == "" {
		return nil, fmt.Errorf("refresh: refresh_token is required")
	}

	scopes := splitScopes(creds.Scopes)
	cfg := &oauth2.Config{
		ClientID:     creds.ClientID,
		ClientSecret: creds.ClientSecret,
		Endpoint: oauth2.Endpoint{
			TokenURL: creds.TokenURL,
		},
		Scopes: scopes,
	}
	// Build a TokenSource seeded with only the refresh token; calling .Token()
	// forces oauth2 to perform the refresh_token grant against TokenURL.
	ts := cfg.TokenSource(ctx, &oauth2.Token{RefreshToken: creds.RefreshToken})
	tok, err := ts.Token()
	if err != nil {
		apiTokenRefreshErrors.Inc()
		return nil, fmt.Errorf("oauth refresh: %w", err)
	}
	// Some providers (Salesforce, ServiceNow) don't return a new refresh_token;
	// preserve the original so the next refresh still works.
	if tok.RefreshToken == "" {
		tok.RefreshToken = creds.RefreshToken
	}
	return tok, nil
}

// splitScopes parses a space-separated scope string into oauth2's expected
// []string. Empty/whitespace input returns nil.
func splitScopes(s string) []string {
	var out []string
	word := ""
	for _, r := range s {
		if r == ' ' || r == ',' || r == '\t' || r == '\n' {
			if word != "" {
				out = append(out, word)
				word = ""
			}
			continue
		}
		word += string(r)
	}
	if word != "" {
		out = append(out, word)
	}
	return out
}

// storedToken is the JSON wire format for tokens at rest in Redis. We don't
// store oauth2.Token directly because its time.Time serializes to a string
// in a non-roundtrip-friendly format on some platforms.
type storedToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type,omitempty"`
	Expiry       time.Time `json:"expiry"`
}

func (s storedToken) toOAuth2() *oauth2.Token {
	return &oauth2.Token{
		AccessToken:  s.AccessToken,
		RefreshToken: s.RefreshToken,
		TokenType:    s.TokenType,
		Expiry:       s.Expiry,
	}
}

func fromOAuth2(t *oauth2.Token) storedToken {
	return storedToken{
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		TokenType:    t.TokenType,
		Expiry:       t.Expiry,
	}
}
