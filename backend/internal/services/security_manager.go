package services

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/hondyman/semlayer/backend/internal/analytics"
)

// RateLimiter provides advanced rate limiting capabilities
type RateLimiter struct {
	userLimits  map[string]*UserRateLimit
	globalLimit *TokenBucket
	mu          sync.RWMutex
	windowSize  time.Duration
	maxRequests int64
}

// UserRateLimit tracks per-user rate limiting
type UserRateLimit struct {
	UserID       string
	RequestCount int64
	WindowStart  time.Time
	LastRequest  time.Time
	BannedUntil  *time.Time
}

// TokenBucket implements token bucket algorithm for rate limiting
type TokenBucket struct {
	capacity   int64
	tokens     int64
	refillRate float64 // tokens per second
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(windowSize time.Duration, maxRequests int64, globalCapacity int64) *RateLimiter {
	return &RateLimiter{
		userLimits:  make(map[string]*UserRateLimit),
		globalLimit: NewTokenBucket(globalCapacity, 100.0), // 100 tokens per second
		windowSize:  windowSize,
		maxRequests: maxRequests,
	}
}

// NewTokenBucket creates a new token bucket
func NewTokenBucket(capacity int64, refillRate float64) *TokenBucket {
	return &TokenBucket{
		capacity:   capacity,
		tokens:     capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request should be allowed
func (rl *RateLimiter) Allow(userID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Check if user is banned
	if limit, exists := rl.userLimits[userID]; exists {
		if limit.BannedUntil != nil && time.Now().Before(*limit.BannedUntil) {
			return false
		}
	}

	// Check global rate limit
	if !rl.globalLimit.Allow() {
		return false
	}

	// Check user-specific rate limit
	return rl.checkUserLimit(userID)
}

// Allow for TokenBucket
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	timePassed := now.Sub(tb.lastRefill).Seconds()
	tokensToAdd := int64(timePassed * tb.refillRate)

	if tokensToAdd > 0 {
		tb.tokens = min(tb.capacity, tb.tokens+tokensToAdd)
		tb.lastRefill = now
	}

	if tb.tokens > 0 {
		tb.tokens--
		return true
	}

	return false
}

func (rl *RateLimiter) checkUserLimit(userID string) bool {
	now := time.Now()
	limit, exists := rl.userLimits[userID]

	if !exists {
		limit = &UserRateLimit{
			UserID:      userID,
			WindowStart: now,
		}
		rl.userLimits[userID] = limit
	}

	// Reset window if needed
	if now.Sub(limit.WindowStart) >= rl.windowSize {
		limit.RequestCount = 0
		limit.WindowStart = now
	}

	// Check if limit exceeded
	if limit.RequestCount >= rl.maxRequests {
		// Ban user for window duration
		banUntil := now.Add(rl.windowSize)
		limit.BannedUntil = &banUntil
		return false
	}

	limit.RequestCount++
	limit.LastRequest = now
	return true
}

// GetUserStats returns rate limiting statistics for a user
func (rl *RateLimiter) GetUserStats(userID string) map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	limit, exists := rl.userLimits[userID]
	if !exists {
		return map[string]interface{}{
			"request_count": 0,
			"banned":        false,
		}
	}

	banned := false
	if limit.BannedUntil != nil && time.Now().Before(*limit.BannedUntil) {
		banned = true
	}

	return map[string]interface{}{
		"request_count": limit.RequestCount,
		"window_start":  limit.WindowStart,
		"last_request":  limit.LastRequest,
		"banned":        banned,
		"ban_until":     limit.BannedUntil,
	}
}

// RateLimitMiddleware creates a standard net/http middleware for rate limiting
func (rl *RateLimiter) RateLimitMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := r.Header.Get("X-User-ID")
			if userID == "" {
				// Fallback to IP if no user ID
				host, _, err := net.SplitHostPort(r.RemoteAddr)
				if err == nil {
					userID = host
				} else {
					userID = r.RemoteAddr
				}
			}

			if !rl.Allow(userID) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":       "Rate limit exceeded",
					"retry_after": rl.windowSize.String(),
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// SecurityManager provides comprehensive security features
type SecurityManager struct {
	jwtManager    *JWTManager
	apiKeyManager *APIKeyManager
	apiKeyStore   APIKeyStore
	encryptionMgr *EncryptionManager
	auditLogger   *AuditLogger
}

// JWTManager handles JWT token operations
type JWTManager struct {
	secretKey       []byte
	tokenDuration   time.Duration
	refreshDuration time.Duration

	// Optional OpenID Connect / Keycloak JWKS configuration for validating
	// RS256 tokens issued by an external identity provider.
	jwksURL    string
	jwksCache  *jwksCache
	httpClient *http.Client
}

// jwksCache holds cached JWKS keys and their expiration.
type jwksCache struct {
	mu         sync.RWMutex
	keys       map[string]*rsa.PublicKey
	expiresAt  time.Time
	fetchError error
}

// JWKS represents the JSON Web Key Set structure returned by an OIDC provider.
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a single JSON Web Key.
type JWK struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
	Alg string `json:"alg"`
}

// APIKeyManager manages API key authentication
type APIKeyManager struct {
	apiKeys map[string]*APIKey
	mu      sync.RWMutex
}

// APIKey represents an API key with permissions
type APIKey struct {
	Key         string
	UserID      string
	TenantID    string
	TenantIDs   []string
	Roles       []string
	Permissions []string
	CreatedAt   time.Time
	LastUsedAt  *time.Time
	ExpiresAt   *time.Time
	Active      bool
}

// EncryptionManager handles data encryption/decryption
type EncryptionManager struct {
	key []byte
}

// AuditLogger provides security event logging
type AuditLogger struct {
	events []SecurityEvent
	mu     sync.Mutex
}

// SecurityEvent represents a security-related event
type SecurityEvent struct {
	EventID   string
	EventType string
	UserID    string
	Resource  string
	Action    string
	Timestamp time.Time
	IPAddress string
	UserAgent string
	Success   bool
	Details   map[string]interface{}
}

// NewSecurityManager creates a new security manager.
// If the KEYCLOAK_JWKS_URL environment variable is set, the JWT manager will
// also validate RS256 tokens from that Keycloak realm.
func NewSecurityManager(cache *analytics.CacheManager, metrics *analytics.MetricsCollector, jwtSecret []byte) *SecurityManager {
	jwksURL := os.Getenv("KEYCLOAK_JWKS_URL")
	return &SecurityManager{
		jwtManager:    NewJWTManagerWithJWKS(jwtSecret, jwksURL),
		apiKeyManager: NewAPIKeyManager(),
		encryptionMgr: NewEncryptionManager(),
		auditLogger:   NewAuditLogger(),
	}
}

// SignToken signs the provided claims and returns a token string.
func (sm *SecurityManager) SignToken(claims jwt.MapClaims) (string, error) {
	if sm == nil || sm.jwtManager == nil {
		return "", fmt.Errorf("security manager not initialized")
	}
	return sm.jwtManager.SignMapClaims(claims)
}

// HasPermission checks whether a user has a given permission.
// This is a lightweight permission check used by HTTP handlers.
func (sm *SecurityManager) HasPermission(userID, permission string) bool {
	if sm == nil {
		return false
	}

	// In this simple implementation we'll allow a special core admin
	// or look up API keys. In the future this should query a user
	// store or an external RBAC service.
	if userID == "core_admin" {
		return true
	}

	// Check API keys permissions
	// (iterate known API keys for a match - this is in-memory demo)
	sm.apiKeyManager.mu.RLock()
	defer sm.apiKeyManager.mu.RUnlock()
	for _, ak := range sm.apiKeyManager.apiKeys {
		if ak.UserID == userID {
			for _, p := range ak.Permissions {
				if p == permission || p == "admin" {
					return true
				}
			}
		}
	}

	return false
}

// ParseToken parses the token string and returns map claims.
func (sm *SecurityManager) ParseToken(tokenString string) (jwt.MapClaims, error) {
	if sm == nil || sm.jwtManager == nil {
		return nil, fmt.Errorf("security manager not initialized")
	}
	return sm.jwtManager.ParseMapClaims(tokenString)
}

// ValidateToken validates a JWT token and returns structured claims.
func (sm *SecurityManager) ValidateToken(tokenString string) (*JWTClaims, error) {
	if sm == nil || sm.jwtManager == nil {
		return nil, fmt.Errorf("security manager not initialized")
	}
	return sm.jwtManager.ValidateToken(tokenString)
}

// GetAPIKey retrieves the full APIKey details if the key is valid.
func (sm *SecurityManager) GetAPIKey(key string) (*APIKey, bool) {
	if sm == nil {
		return nil, false
	}
	return sm.GetAPIKeyWithContext(context.Background(), key)
}

// GetAPIKeyWithContext retrieves API key details, checking DB store if configured.
func (sm *SecurityManager) GetAPIKeyWithContext(ctx context.Context, key string) (*APIKey, bool) {
	if sm == nil || sm.apiKeyManager == nil {
		return nil, false
	}
	if apiKey, ok := sm.apiKeyManager.ValidateAPIKey(key); ok {
		return apiKey, true
	}
	if sm.apiKeyStore == nil {
		return nil, false
	}
	apiKey, err := sm.apiKeyStore.FindByKey(ctx, key)
	if err != nil || apiKey == nil {
		return nil, false
	}
	return apiKey, true
}

// ValidateAPIKey validates an API key and returns the associated user ID if valid.
// This is a convenience wrapper so external packages don't need to access apiKeyManager directly.
func (sm *SecurityManager) ValidateAPIKey(key string) (string, bool) {
	if sm == nil {
		return "", false
	}
	if key == "" {
		return "", false
	}
	ak, ok := sm.GetAPIKeyWithContext(context.Background(), key)
	if !ok || ak == nil {
		return "", false
	}
	return ak.UserID, true
}

// SetAPIKeyStore configures a DB-backed API key store for lookups.
func (sm *SecurityManager) SetAPIKeyStore(store APIKeyStore) {
	if sm == nil {
		return
	}
	sm.apiKeyStore = store
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(secret []byte) *JWTManager {
	return NewJWTManagerWithJWKS(secret, "")
}

// NewJWTManagerWithJWKS creates a JWT manager that can validate both HS256
// platform tokens and RS256 tokens from an external OIDC provider.
func NewJWTManagerWithJWKS(secret []byte, jwksURL string) *JWTManager {
	jm := &JWTManager{
		secretKey:       secret,
		tokenDuration:   15 * time.Minute,
		refreshDuration: 7 * 24 * time.Hour,
		jwksURL:         jwksURL,
		jwksCache:       &jwksCache{},
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
	return jm
}

// SignMapClaims signs the provided claims with HS256 and returns the token string.
func (jm *JWTManager) SignMapClaims(claims jwt.MapClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jm.secretKey)
}

// ParseMapClaims parses a token string and returns the MapClaims if valid.
func (jm *JWTManager) ParseMapClaims(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jm.secretKey, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid token claims")
}

// parseAndValidateRS256 validates an RS256 token using the configured JWKS.
func (jm *JWTManager) parseAndValidateRS256(tokenString, kid string) (*jwt.Token, error) {
	pubKey, err := jm.getJWKSKey(kid)
	if err != nil {
		return nil, err
	}
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return pubKey, nil
	})
}

// getJWKSKey returns the RSA public key for the given key ID, fetching the
// JWKS from the configured URL and caching it for subsequent lookups.
func (jm *JWTManager) getJWKSKey(kid string) (*rsa.PublicKey, error) {
	jm.jwksCache.mu.RLock()
	key, ok := jm.jwksCache.keys[kid]
	expiresAt := jm.jwksCache.expiresAt
	fetchErr := jm.jwksCache.fetchError
	jm.jwksCache.mu.RUnlock()

	if ok && time.Now().Before(expiresAt) {
		return key, nil
	}
	if fetchErr != nil && time.Now().Before(expiresAt) {
		return nil, fetchErr
	}

	jm.jwksCache.mu.Lock()
	defer jm.jwksCache.mu.Unlock()

	// Double-check after acquiring write lock.
	if key, ok := jm.jwksCache.keys[kid]; ok && time.Now().Before(jm.jwksCache.expiresAt) {
		return key, nil
	}

	req, err := http.NewRequest("GET", jm.jwksURL, nil)
	if err != nil {
		jm.jwksCache.fetchError = err
		jm.jwksCache.expiresAt = time.Now().Add(30 * time.Second)
		return nil, err
	}
	resp, err := jm.httpClient.Do(req)
	if err != nil {
		jm.jwksCache.fetchError = err
		jm.jwksCache.expiresAt = time.Now().Add(30 * time.Second)
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		jm.jwksCache.fetchError = fmt.Errorf("JWKS endpoint returned %d: %s", resp.StatusCode, string(body))
		jm.jwksCache.expiresAt = time.Now().Add(30 * time.Second)
		return nil, jm.jwksCache.fetchError
	}

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		jm.jwksCache.fetchError = err
		jm.jwksCache.expiresAt = time.Now().Add(30 * time.Second)
		return nil, fmt.Errorf("failed to decode JWKS: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey, len(jwks.Keys))
	var firstErr error
	for _, jwk := range jwks.Keys {
		if jwk.Kty != "RSA" {
			continue
		}
		if jwk.Use != "" && jwk.Use != "sig" {
			continue
		}
		pub, err := jwkToRSAPublicKey(jwk)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		keys[jwk.Kid] = pub
	}
	if len(keys) == 0 && firstErr != nil {
		jm.jwksCache.fetchError = firstErr
		jm.jwksCache.expiresAt = time.Now().Add(30 * time.Second)
		return nil, firstErr
	}

	jm.jwksCache.keys = keys
	jm.jwksCache.fetchError = nil
	jm.jwksCache.expiresAt = time.Now().Add(15 * time.Minute)

	key, ok = keys[kid]
	if !ok {
		return nil, fmt.Errorf("JWKS does not contain key ID %q", kid)
	}
	return key, nil
}

// jwkToRSAPublicKey converts a JWK with base64url-encoded n/e to an RSA public key.
func jwkToRSAPublicKey(jwk JWK) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("invalid JWK modulus: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("invalid JWK exponent: %w", err)
	}
	if len(eBytes) == 0 {
		return nil, fmt.Errorf("empty JWK exponent")
	}
	n := new(big.Int).SetBytes(nBytes)
	e := int(new(big.Int).SetBytes(eBytes).Int64())
	if e == 0 {
		return nil, fmt.Errorf("invalid JWK exponent value")
	}
	return &rsa.PublicKey{N: n, E: e}, nil
}

// pemToRSAPublicKey parses a PEM-encoded RSA public key (used for static OIDC keys).
func pemToRSAPublicKey(pemBytes []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("PEM block is not an RSA public key")
	}
	return rsaPub, nil
}

// ValidateToken validates a JWT token and returns claims.
// It supports platform-issued HS256 tokens and, when a JWKS URL is configured,
// RS256 tokens from an external OIDC provider such as Keycloak.
func (jm *JWTManager) ValidateToken(tokenString string) (*JWTClaims, error) {
	// Accept optional "Bearer " prefix
	if strings.HasPrefix(strings.ToLower(tokenString), "bearer ") {
		tokenString = strings.TrimSpace(tokenString[7:])
	}
	if tokenString == "" {
		return nil, fmt.Errorf("empty token")
	}

	// Peek at the header to decide which validation path to use without fully
	// trusting the unverified token.
	token, _, err := jwt.NewParser().ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	alg, _ := token.Header["alg"].(string)
	kid, _ := token.Header["kid"].(string)

	var claims jwt.MapClaims
	switch alg {
	case "HS256":
		claims, err = jm.ParseMapClaims(tokenString)
		if err != nil {
			return nil, err
		}
	case "RS256":
		if jm.jwksURL == "" {
			return nil, fmt.Errorf("RS256 token received but no JWKS URL configured")
		}
		parsed, err := jm.parseAndValidateRS256(tokenString, kid)
		if err != nil {
			return nil, err
		}
		if mc, ok := parsed.Claims.(jwt.MapClaims); ok {
			claims = mc
		} else {
			return nil, fmt.Errorf("invalid RS256 token claims")
		}
	default:
		return nil, fmt.Errorf("unsupported signing algorithm: %s", alg)
	}

	// Try common claim names for user id
	var userID string
	if sub, ok := claims["user_id"].(string); ok && sub != "" {
		userID = sub
	} else if sub, ok := claims["sub"].(string); ok && sub != "" {
		userID = sub
	} else if sub, ok := claims["uid"].(string); ok && sub != "" {
		userID = sub
	}

	if userID == "" {
		return nil, fmt.Errorf("token missing user identifier")
	}

	issuedAt := time.Now()
	if iat, ok := claims["iat"].(float64); ok {
		issuedAt = time.Unix(int64(iat), 0)
	}

	// Extract tenant_id if present
	var tenantID string
	if tid, ok := claims["tenant_id"].(string); ok {
		tenantID = tid
	}

	roles := parseStringListClaim(claims["roles"])
	// Extract Keycloak realm_access.roles if present
	if realmAccess, ok := claims["realm_access"].(map[string]interface{}); ok {
		roles = append(roles, parseStringListClaim(realmAccess["roles"])...)
	}
	// Extract resource_access roles as "client:role" pairs
	if resourceAccess, ok := claims["resource_access"].(map[string]interface{}); ok {
		for client, access := range resourceAccess {
			if accessMap, ok := access.(map[string]interface{}); ok {
				for _, r := range parseStringListClaim(accessMap["roles"]) {
					roles = addClaimItem(roles, map[string]struct{}{}, client+":"+r)
				}
			}
		}
	}

	// Extract Keycloak groups claim (raw enterprise/IdP groups).
	idpGroups := parseStringListClaim(claims["groups"])

	// Extract Uisce internal operator role from nested uisce_metadata claim.
	var operatorRole string
	if metadata, ok := claims["uisce_metadata"].(map[string]interface{}); ok {
		if or, ok := metadata["operator_role"].(string); ok {
			operatorRole = strings.TrimSpace(or)
		}
	}
	// Fallback: allow top-level claim name as well.
	if operatorRole == "" {
		if or, ok := claims["operator_role"].(string); ok {
			operatorRole = strings.TrimSpace(or)
		}
	}
	// Fallback: map from Keycloak groups if operatorRole is empty
	if operatorRole == "" {
		for _, group := range idpGroups {
			switch group {
			case "Uisce-Global-Admins":
				operatorRole = "global_admin"
			case "Uisce-Professional-Services":
				operatorRole = "professional_services"
			case "Uisce-Helpdesk":
				operatorRole = "helpdesk"
			}
			if operatorRole != "" {
				break
			}
		}
	}

	tenantIDs := parseStringListClaim(claims["tenant_ids"])
	if len(tenantIDs) == 0 && tenantID != "" {
		tenantIDs = []string{tenantID}
	}

	var email string
	if em, ok := claims["email"].(string); ok {
		email = strings.TrimSpace(em)
	}

	var clientID string
	if azp, ok := claims["azp"].(string); ok {
		clientID = azp
	}

	return &JWTClaims{
		UserID:       userID,
		Email:        email,
		TenantID:     tenantID,
		TenantIDs:    tenantIDs,
		Roles:        roles,
		IDPGroups:    idpGroups,
		OperatorRole: operatorRole,
		IssuedAt:     issuedAt,
		ClientID:     clientID,
	}, nil
}

// JWTClaims represents JWT token claims
type JWTClaims struct {
	UserID         string
	Email          string
	TenantID       string
	TenantIDs      []string
	Roles          []string
	IDPGroups      []string
	OperatorRole   string // Uisce internal staff role from uisce_metadata.operator_role
	FunctionalRole string // Resolved abstract platform profile role
	ClearanceLevel string // Resolved clearance level
	IssuedAt       time.Time
	ClientID       string // Keycloak Authorized Party / Client ID
}

func parseStringListClaim(value interface{}) []string {
	result := []string{}
	seen := map[string]struct{}{}
	switch v := value.(type) {
	case []string:
		for _, item := range v {
			result = addClaimItem(result, seen, item)
		}
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = addClaimItem(result, seen, s)
			}
		}
	case string:
		result = addClaimItem(result, seen, v)
	}
	return result
}

func addClaimItem(result []string, seen map[string]struct{}, value string) []string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return result
	}
	if _, ok := seen[trimmed]; ok {
		return result
	}
	seen[trimmed] = struct{}{}
	return append(result, trimmed)
}

func normalizeList(values []string) []string {
	result := []string{}
	seen := map[string]struct{}{}
	for _, value := range values {
		result = addClaimItem(result, seen, value)
	}
	return result
}

// NewAPIKeyManager creates a new API key manager
func NewAPIKeyManager() *APIKeyManager {
	return &APIKeyManager{
		apiKeys: make(map[string]*APIKey),
	}
}

// GenerateAPIKey creates an API key for a user via the internal APIKeyManager.
func (sm *SecurityManager) GenerateAPIKey(userID string, tenantID string, permissions []string) string {
	if sm == nil || sm.apiKeyManager == nil {
		return ""
	}
	return sm.apiKeyManager.GenerateAPIKey(userID, tenantID, permissions)
}

// GenerateAPIKeyWithTenants creates an API key with an explicit tenant allow-list.
func (sm *SecurityManager) GenerateAPIKeyWithTenants(userID string, tenantIDs []string, roles []string) string {
	if sm == nil || sm.apiKeyManager == nil {
		return ""
	}
	return sm.apiKeyManager.GenerateAPIKeyWithTenants(userID, tenantIDs, roles)
}

// RegisterAPIKey registers a pre-generated API key for runtime use.
func (sm *SecurityManager) RegisterAPIKey(key string, userID string, tenantIDs []string, roles []string) {
	if sm == nil || sm.apiKeyManager == nil {
		return
	}
	sm.apiKeyManager.RegisterAPIKey(key, userID, tenantIDs, roles)
}

// NewEncryptionManager creates a new encryption manager
func NewEncryptionManager() *EncryptionManager {
	secret := os.Getenv("ENCRYPTION_SECRET")
	if secret == "" {
		// Log a warning in dev, but in a real app this should be mandatory
		secret = "dev-encryption-key-32-chars-long!"
	}
	return &EncryptionManager{key: []byte(secret)}
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger() *AuditLogger {
	return &AuditLogger{
		events: make([]SecurityEvent, 0),
	}
}

// GenerateAPIKey generates a new API key for a user
func (akm *APIKeyManager) GenerateAPIKey(userID string, tenantID string, permissions []string) string {
	akm.mu.Lock()
	defer akm.mu.Unlock()

	key := generateSecureKey()
	now := time.Now()

	apiKey := &APIKey{
		Key:         key,
		UserID:      userID,
		TenantID:    tenantID,
		TenantIDs:   normalizeList([]string{tenantID}),
		Roles:       normalizeList(permissions),
		Permissions: permissions,
		CreatedAt:   now,
		Active:      true,
	}

	akm.apiKeys[key] = apiKey
	return key
}

// GenerateAPIKeyWithTenants generates a new API key for a user with multiple tenants.
func (akm *APIKeyManager) GenerateAPIKeyWithTenants(userID string, tenantIDs []string, roles []string) string {
	akm.mu.Lock()
	defer akm.mu.Unlock()

	key := generateSecureKey()
	now := time.Now()

	apiKey := &APIKey{
		Key:       key,
		UserID:    userID,
		TenantIDs: normalizeList(tenantIDs),
		Roles:     normalizeList(roles),
		CreatedAt: now,
		Active:    true,
	}

	akm.apiKeys[key] = apiKey
	return key
}

// RegisterAPIKey registers an existing API key in the in-memory store.
func (akm *APIKeyManager) RegisterAPIKey(key string, userID string, tenantIDs []string, roles []string) {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return
	}

	akm.mu.Lock()
	defer akm.mu.Unlock()

	apiKey := &APIKey{
		Key:       trimmed,
		UserID:    userID,
		TenantIDs: normalizeList(tenantIDs),
		Roles:     normalizeList(roles),
		CreatedAt: time.Now(),
		Active:    true,
	}

	akm.apiKeys[trimmed] = apiKey
}

// ValidateAPIKey validates an API key and returns associated permissions
func (akm *APIKeyManager) ValidateAPIKey(key string) (*APIKey, bool) {
	akm.mu.RLock()
	defer akm.mu.RUnlock()

	apiKey, exists := akm.apiKeys[key]
	if !exists || !apiKey.Active {
		return nil, false
	}

	// Check expiration
	if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
		return nil, false
	}

	// Update last used time
	now := time.Now()
	apiKey.LastUsedAt = &now

	return apiKey, true
}

// RevokeAPIKey revokes an API key
func (akm *APIKeyManager) RevokeAPIKey(key string) bool {
	akm.mu.Lock()
	defer akm.mu.Unlock()

	if apiKey, exists := akm.apiKeys[key]; exists {
		apiKey.Active = false
		return true
	}
	return false
}

// LogSecurityEvent logs a security event
func (al *AuditLogger) LogSecurityEvent(event SecurityEvent) {
	al.mu.Lock()
	defer al.mu.Unlock()

	event.EventID = generateEventID()
	event.Timestamp = time.Now()

	al.events = append(al.events, event)

	// In production, this would write to a secure log store
	fmt.Printf("Security Event: %s - %s - %s - %s\n",
		event.EventType, event.UserID, event.Resource, event.Action)
}

// GetSecurityEvents returns security events for a user
func (al *AuditLogger) GetSecurityEvents(userID string, limit int) []SecurityEvent {
	al.mu.Lock()
	defer al.mu.Unlock()

	var userEvents []SecurityEvent
	for _, event := range al.events {
		if event.UserID == userID {
			userEvents = append(userEvents, event)
		}
	}

	// Return most recent events
	if len(userEvents) > limit {
		userEvents = userEvents[len(userEvents)-limit:]
	}

	return userEvents
}

// SecurityMiddleware creates a standard net/http middleware for security
func (sm *SecurityManager) SecurityMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := ""
			authMethod := ""

			// Check JWT token
			if token := r.Header.Get("Authorization"); token != "" {
				if claims, err := sm.jwtManager.ValidateToken(token); err == nil {
					userID = claims.UserID
					authMethod = "jwt"
				}
			}

			// Check API key
			if userID == "" {
				if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
					if apiKeyInfo, valid := sm.GetAPIKeyWithContext(r.Context(), apiKey); valid {
						userID = apiKeyInfo.UserID
						authMethod = "api_key"
					}
				}
			}

			// Capture IP for logging
			ip, _, _ := net.SplitHostPort(r.RemoteAddr)

			// Log authentication attempt
			event := SecurityEvent{
				EventType: "authentication",
				UserID:    userID,
				Resource:  r.URL.Path,
				Action:    r.Method,
				IPAddress: ip,
				UserAgent: r.Header.Get("User-Agent"),
				Success:   userID != "",
				Details: map[string]interface{}{
					"auth_method": authMethod,
				},
			}
			sm.auditLogger.LogSecurityEvent(event)

			if userID == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": "Authentication required",
				})
				return
			}

			// Store user info in context (Note: we use standard request context here)
			ctx := context.WithValue(r.Context(), "semlayer_user_id", userID)
			ctx = context.WithValue(ctx, "semlayer_auth_method", authMethod)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// HealthCheckMiddleware provides standard net/http health check endpoints
func (sm *SecurityManager) HealthCheckMiddleware(cacheMgr *analytics.CacheManager, metricsCollector *analytics.MetricsCollector) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" {
				health := sm.getHealthStatus(cacheMgr, metricsCollector)
				status := http.StatusOK
				if !health.Healthy {
					status = http.StatusServiceUnavailable
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				json.NewEncoder(w).Encode(health)
				return
			}

			if r.URL.Path == "/ready" {
				ready := sm.getReadinessStatus(cacheMgr)
				status := http.StatusOK
				if !ready.Ready {
					status = http.StatusServiceUnavailable
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				json.NewEncoder(w).Encode(ready)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// HealthStatus represents the health check response
type HealthStatus struct {
	Healthy   bool                   `json:"healthy"`
	Timestamp time.Time              `json:"timestamp"`
	Services  map[string]bool        `json:"services"`
	Metrics   map[string]interface{} `json:"metrics"`
}

// ReadinessStatus represents the readiness check response
type ReadinessStatus struct {
	Ready     bool      `json:"ready"`
	Timestamp time.Time `json:"timestamp"`
	Checks    []string  `json:"checks"`
}

// RevokeAPIKey revokes a registered API key.
func (sm *SecurityManager) RevokeAPIKey(key string) bool {
	if sm == nil || sm.apiKeyManager == nil {
		return false
	}
	return sm.apiKeyManager.RevokeAPIKey(key)
}

// getHealthStatus performs comprehensive health checks
func (sm *SecurityManager) getHealthStatus(cacheMgr *analytics.CacheManager, metricsCollector *analytics.MetricsCollector) *HealthStatus {
	health := &HealthStatus{
		Healthy:   true,
		Timestamp: time.Now(),
		Services:  make(map[string]bool),
		Metrics:   make(map[string]interface{}),
	}

	// Check cache health
	cacheStats := cacheMgr.GetStats()
	health.Services["cache"] = true // Assume healthy if we can get stats
	health.Metrics["cache"] = cacheStats

	// Check metrics collector
	systemMetrics := metricsCollector.GetSystemMetrics()
	health.Services["metrics"] = true
	health.Metrics["system"] = systemMetrics

	// Check if system is overloaded
	if systemMetrics.CPUUsage > 90.0 {
		health.Healthy = false
		health.Services["cpu"] = false
	} else {
		health.Services["cpu"] = true
	}

	return health
}

// getReadinessStatus checks if the service is ready to accept traffic
func (sm *SecurityManager) getReadinessStatus(_ *analytics.CacheManager) *ReadinessStatus {
	ready := &ReadinessStatus{
		Ready:     true,
		Timestamp: time.Now(),
		Checks:    []string{},
	}

	// Check if cache is initialized
	ready.Checks = append(ready.Checks, "cache_initialized")

	// Check if security services are ready
	ready.Checks = append(ready.Checks, "security_services")

	// In a real implementation, you would check database connections,
	// external service dependencies, etc.

	return ready
}

// Helper functions
func generateSecureKey() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("failed to generate secure key: %v", err))
	}
	return hex.EncodeToString(b)
}

func generateEventID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("failed to generate event ID: %v", err))
	}
	return "evt_" + hex.EncodeToString(b)
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
