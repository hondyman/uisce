// Package dscreds resolves datasource connection credentials from the secrets
// store (Infisical in dev) instead of the metadata DB.
//
// A datasource's connection config (tenant_product_datasource.config, or the
// connection-details JSON built from a public.connections row) holds only a
// reference:
//
//	{"host": "...", "port": 5432, "database": "...", "secret_path": "/datasources/<tenant_id>/<datasource_id>"}
//
// and the secrets store folder at that path holds the credentials:
//
//	USERNAME, PASSWORD, PRIVATE_KEY (client TLS key, PEM), API_KEY
//
// Hydrate merges them back into the config JSON just before a connection is
// opened, so the existing DSN/TLS builders are unchanged.
//
// Tenant isolation: the stored secret_path is never trusted as-is. It must equal
// the canonical path derived from the row's own tenant and id, so a config copied
// or edited to point at another tenant's (or the gold copy's) secret is refused.
package dscreds

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/hondyman/uisce/backend/internal/secrets"
)

// RefKey is the config key that holds the secrets-store reference.
const RefKey = "secret_path"

// Secret keys inside a datasource's secrets-store folder.
const (
	KeyUsername   = "USERNAME"
	KeyPassword   = "PASSWORD"
	KeyPrivateKey = "PRIVATE_KEY"
	KeyAPIKey     = "API_KEY"
)

// Kind is the table that owns the credential; it is the first path segment.
type Kind string

const (
	// KindDatasource is a public.tenant_product_datasource row.
	KindDatasource Kind = "datasources"
	// KindConnection is a public.connections row.
	KindConnection Kind = "connections"
)

var (
	// ErrPlaintextCredentials is returned in strict mode for a config that still
	// carries inline credentials and no secret_path.
	ErrPlaintextCredentials = errors.New("datasource config holds plaintext credentials and no secret_path")
	// ErrRefMismatch is returned when secret_path is not the canonical path for
	// the row's own tenant and id.
	ErrRefMismatch = errors.New("datasource secret_path does not belong to this datasource")
	// ErrNoProvider is returned when a config references the secrets store but
	// no provider is configured. There is no fallback to inline credentials.
	ErrNoProvider = errors.New("datasource config references the secrets store but no secrets provider is configured")
)

// CanonicalPath is the only secrets-store path a given row may reference.
func CanonicalPath(kind Kind, tenantID, id string) (string, error) {
	if kind != KindDatasource && kind != KindConnection {
		return "", fmt.Errorf("unknown credential owner kind %q", kind)
	}
	t, err := uuid.Parse(strings.TrimSpace(tenantID))
	if err != nil {
		return "", fmt.Errorf("invalid tenant id for %s credentials: %w", kind, err)
	}
	i, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return "", fmt.Errorf("invalid %s id for credentials: %w", kind, err)
	}
	return fmt.Sprintf("/%s/%s/%s", kind, t, i), nil
}

// Resolver hydrates connection configs from the secrets store.
type Resolver struct {
	provider   secrets.Provider
	requireRef bool
	ttl        time.Duration
	now        func() time.Time

	mu    sync.Mutex
	cache map[string]cachedSecret
}

type cachedSecret struct {
	values  map[string]string
	expires time.Time
}

// Option configures a Resolver.
type Option func(*Resolver)

// WithRequireRef refuses configs that carry inline credentials without a
// secret_path. Turn on once every row has been migrated.
func WithRequireRef(require bool) Option { return func(r *Resolver) { r.requireRef = require } }

// WithCacheTTL sets how long fetched secrets are reused (0 disables caching).
func WithCacheTTL(ttl time.Duration) Option { return func(r *Resolver) { r.ttl = ttl } }

// NewResolver builds a Resolver. provider may be nil: configs without a
// secret_path still work (legacy inline credentials), configs with one fail.
func NewResolver(provider secrets.Provider, opts ...Option) *Resolver {
	r := &Resolver{
		provider: provider,
		ttl:      5 * time.Minute,
		now:      time.Now,
		cache:    make(map[string]cachedSecret),
	}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Hydrate returns config with credentials filled in from the secrets store.
// tenantID and id are the owning row's own tenant and primary key, read from
// the DB alongside config — never from the caller's request.
func (r *Resolver) Hydrate(ctx context.Context, kind Kind, tenantID, id string, config []byte) ([]byte, error) {
	cfg := map[string]any{}
	if trimmed := strings.TrimSpace(string(config)); trimmed != "" && trimmed != "null" {
		if err := json.Unmarshal(config, &cfg); err != nil {
			return nil, fmt.Errorf("parse %s %s connection config: %w", kind, id, err)
		}
	}

	ref, _ := cfg[RefKey].(string)
	if strings.TrimSpace(ref) == "" {
		if HasInlineSecrets(cfg) {
			if r.requireRef {
				return nil, fmt.Errorf("%s %s: %w", kind, id, ErrPlaintextCredentials)
			}
			logging.GetLogger().Sugar().Warnf("%s %s still stores credentials in the metadata DB; move them to the secrets store", kind, id)
		}
		return config, nil
	}

	want, err := CanonicalPath(kind, tenantID, id)
	if err != nil {
		return nil, err
	}
	if ref != want {
		// Do not echo either path's secret content; the paths themselves are not secret.
		return nil, fmt.Errorf("%s %s: %w (got %s, want %s)", kind, id, ErrRefMismatch, ref, want)
	}
	if r == nil || r.provider == nil {
		return nil, fmt.Errorf("%s %s: %w", kind, id, ErrNoProvider)
	}

	values, err := r.fetch(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("%s %s: read credentials from secrets store at %s: %w", kind, id, ref, err)
	}
	if values[KeyPassword] == "" && values[KeyPrivateKey] == "" && values[KeyAPIKey] == "" {
		return nil, fmt.Errorf("%s %s: secrets store folder %s has no %s, %s or %s: %w",
			kind, id, ref, KeyPassword, KeyPrivateKey, KeyAPIKey, secrets.ErrSecretNotFound)
	}

	// The store is authoritative once a reference exists: drop anything inline
	// so a stale plaintext value left behind mid-migration is never used.
	StripInlineSecrets(cfg)
	auth, _ := cfg["auth"].(map[string]any)
	if auth == nil {
		auth = map[string]any{}
	}
	basic, _ := auth["basic"].(map[string]any)
	if basic == nil {
		basic = map[string]any{}
	}
	if u := values[KeyUsername]; u != "" {
		basic["username"] = u
		if _, flat := cfg["username"]; flat {
			cfg["username"] = u
		}
	}
	if p := values[KeyPassword]; p != "" {
		basic["password"] = p
	}
	if len(basic) > 0 {
		auth["basic"] = basic
		cfg["auth"] = auth
	}
	if k := values[KeyPrivateKey]; k != "" {
		cfg["private_key"] = k
	}
	if k := values[KeyAPIKey]; k != "" {
		cfg["api_key"] = k
	}

	out, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("encode %s %s connection config: %w", kind, id, err)
	}
	return out, nil
}

func (r *Resolver) fetch(ctx context.Context, path string) (map[string]string, error) {
	if r.ttl > 0 {
		r.mu.Lock()
		c, ok := r.cache[path]
		r.mu.Unlock()
		if ok && r.now().Before(c.expires) {
			return c.values, nil
		}
	}
	values, err := r.provider.GetMap(ctx, path)
	if err != nil {
		return nil, err
	}
	if r.ttl > 0 {
		r.mu.Lock()
		r.cache[path] = cachedSecret{values: values, expires: r.now().Add(r.ttl)}
		r.mu.Unlock()
	}
	return values, nil
}

// Invalidate drops a cached secret, e.g. after a credential rotation.
func (r *Resolver) Invalidate(path string) {
	r.mu.Lock()
	delete(r.cache, path)
	r.mu.Unlock()
}

// inlineSecretKeys are the credential keys a connection config may carry inline.
var inlineSecretKeys = []string{"password", "private_key", "api_key", "client_secret", "dsn", "connection_string", "database_url"}

// HasInlineSecrets reports whether a decoded config carries a non-empty
// credential value in the metadata DB.
func HasInlineSecrets(cfg map[string]any) bool {
	for _, k := range inlineSecretKeys {
		if s, _ := cfg[k].(string); s != "" {
			return true
		}
	}
	if auth, ok := cfg["auth"].(map[string]any); ok {
		if basic, ok := auth["basic"].(map[string]any); ok {
			if s, _ := basic["password"].(string); s != "" {
				return true
			}
		}
	}
	return false
}

// StripInlineSecrets removes every inline credential (and the secret_path
// reference) from a decoded config in place. Non-secret fields — host, port,
// username, ca_cert, client_cert — are kept.
func StripInlineSecrets(cfg map[string]any) {
	for _, k := range inlineSecretKeys {
		delete(cfg, k)
	}
	if auth, ok := cfg["auth"].(map[string]any); ok {
		if basic, ok := auth["basic"].(map[string]any); ok {
			delete(basic, "password")
		}
	}
}

// StripForClone returns a copy of config JSON safe to copy into another tenant:
// no inline credentials and no secret_path (which would point at the source
// tenant's secret). Invalid JSON yields "{}".
func StripForClone(config []byte) []byte {
	cfg := map[string]any{}
	if len(config) > 0 {
		if err := json.Unmarshal(config, &cfg); err != nil || cfg == nil {
			return []byte(`{}`)
		}
	}
	StripInlineSecrets(cfg)
	delete(cfg, RefKey)
	out, err := json.Marshal(cfg)
	if err != nil {
		return []byte(`{}`)
	}
	return out
}

var (
	defaultOnce     sync.Once
	defaultResolver *Resolver
	defaultMu       sync.RWMutex
)

// Default returns the process-wide Resolver, built on first use from env:
//
//	SECRETS_PROVIDER                          infisical | vault | memory (unset: no provider)
//	INFISICAL_SITE_URL (or INFISICAL_DOMAIN)  e.g. http://100.84.50.65:8085
//	INFISICAL_CLIENT_ID / INFISICAL_CLIENT_SECRET  universal-auth machine identity
//	INFISICAL_TOKEN                           access token, if no machine identity
//	INFISICAL_PROJECT_ID, INFISICAL_ENVIRONMENT
//	DATASOURCE_CREDENTIALS_REQUIRE_SECRET_REF=true  refuse inline plaintext
//
// A provider that fails to initialise is logged and left nil, so every
// datasource that references the store fails closed rather than falling back.
func Default() *Resolver {
	defaultMu.RLock()
	r := defaultResolver
	defaultMu.RUnlock()
	if r != nil {
		return r
	}
	defaultOnce.Do(func() {
		built := fromEnv()
		defaultMu.Lock()
		if defaultResolver == nil {
			defaultResolver = built
		}
		defaultMu.Unlock()
	})
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	return defaultResolver
}

// SetDefault replaces the process-wide Resolver (tests, or explicit wiring in main).
func SetDefault(r *Resolver) {
	defaultMu.Lock()
	defaultResolver = r
	defaultMu.Unlock()
}

func fromEnv() *Resolver {
	require := strings.EqualFold(os.Getenv("DATASOURCE_CREDENTIALS_REQUIRE_SECRET_REF"), "true")
	p, err := ProviderFromEnv()
	if err != nil {
		logging.GetLogger().Sugar().Errorf("datasource credentials: secrets provider unavailable, datasources with a secret_path will fail to connect: %v", err)
		return NewResolver(nil, WithRequireRef(require))
	}
	return NewResolver(p, WithRequireRef(require))
}

// ProviderFromEnv builds the secrets provider described in Default's doc
// comment. It returns (nil, nil) when SECRETS_PROVIDER is unset.
func ProviderFromEnv() (secrets.Provider, error) {
	typ := strings.TrimSpace(os.Getenv("SECRETS_PROVIDER"))
	if typ == "" {
		return nil, nil
	}
	site := os.Getenv("INFISICAL_SITE_URL")
	if site == "" {
		site = strings.TrimSuffix(strings.TrimSuffix(os.Getenv("INFISICAL_DOMAIN"), "/"), "/api")
	}
	p, err := secrets.NewProvider(secrets.Config{
		Type:                  typ,
		VaultAddr:             os.Getenv("VAULT_ADDR"),
		VaultToken:            os.Getenv("VAULT_TOKEN"),
		InfisicalURL:          site,
		InfisicalClientID:     os.Getenv("INFISICAL_CLIENT_ID"),
		InfisicalClientSecret: os.Getenv("INFISICAL_CLIENT_SECRET"),
		InfisicalProjectID:    os.Getenv("INFISICAL_PROJECT_ID"),
		InfisicalEnvironment:  os.Getenv("INFISICAL_ENVIRONMENT"),
		InfisicalServiceToken: os.Getenv("INFISICAL_TOKEN"),
	})
	if err != nil {
		return nil, fmt.Errorf("secrets provider %q: %w", typ, err)
	}
	return p, nil
}

// ErrConflictingCredentials is returned by ExtractInline when a config holds two
// different non-empty values for one credential (e.g. auth.basic.password and a
// flat password). Readers disagree on which wins, so migration refuses to guess.
var ErrConflictingCredentials = errors.New("config holds conflicting values for one credential")

// ExtractInline collects a decoded config's inline credentials, keyed by the
// secrets-store key names, for migration into the store. Empty values are
// omitted. The error names the credential, never its values.
func ExtractInline(cfg map[string]any) (map[string]string, error) {
	var basic map[string]any
	if auth, ok := cfg["auth"].(map[string]any); ok {
		basic, _ = auth["basic"].(map[string]any)
	}
	str := func(m map[string]any, k string) string {
		s, _ := m[k].(string)
		return s
	}
	out := map[string]string{}
	for _, c := range []struct {
		key        string
		candidates []string
	}{
		{KeyUsername, []string{str(basic, "username"), str(cfg, "username")}},
		{KeyPassword, []string{str(basic, "password"), str(cfg, "password")}},
		{KeyPrivateKey, []string{str(cfg, "private_key")}},
		{KeyAPIKey, []string{str(cfg, "api_key")}},
	} {
		for _, v := range c.candidates {
			if v == "" {
				continue
			}
			if prev, ok := out[c.key]; ok && prev != v {
				return nil, fmt.Errorf("%s: %w", c.key, ErrConflictingCredentials)
			}
			out[c.key] = v
		}
	}
	return out, nil
}
