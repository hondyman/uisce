package services

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/hondyman/uisce/backend/internal/analytics"
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
	rsaPublicKey    interface{}
	rsaPublicKeys   map[string]*rsa.PublicKey // keyed by JWKS "kid", for rotation support
	tokenDuration   time.Duration
	refreshDuration time.Duration
	mu              sync.RWMutex
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

// NewSecurityManager creates a new security manager
func NewSecurityManager(cache *analytics.CacheManager, metrics *analytics.MetricsCollector, jwtSecret []byte) *SecurityManager {
	return &SecurityManager{
		jwtManager:    NewJWTManager(jwtSecret),
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

// SetRSAPublicKeyPEM sets the RSA public key for RS256 token verification from PEM-encoded data
func (sm *SecurityManager) SetRSAPublicKeyPEM(pemBytes []byte) error {
	if sm == nil || sm.jwtManager == nil {
		return fmt.Errorf("security manager not initialized")
	}
	return sm.jwtManager.SetRSAPublicKeyPEM(pemBytes)
}

// SetRSAPublicKey sets the RSA public key for RS256 token verification
func (sm *SecurityManager) SetRSAPublicKey(pubKey *rsa.PublicKey) {
	if sm != nil && sm.jwtManager != nil {
		sm.jwtManager.SetRSAPublicKey(pubKey)
	}
}

// SetRSAPublicKeys replaces the full set of known JWKS signing keys, keyed by
// "kid" — see JWTManager.SetRSAPublicKeys for rotation semantics.
func (sm *SecurityManager) SetRSAPublicKeys(keys map[string]*rsa.PublicKey) {
	if sm != nil && sm.jwtManager != nil {
		sm.jwtManager.SetRSAPublicKeys(keys)
	}
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
	return &JWTManager{
		secretKey:       secret,
		tokenDuration:   15 * time.Minute,
		refreshDuration: 7 * 24 * time.Hour,
	}
}

// SignMapClaims signs the provided claims with HS256 and returns the token string.
func (jm *JWTManager) SignMapClaims(claims jwt.MapClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jm.secretKey)
}

func getStringClaim(claims jwt.MapClaims, key string) string {
	if v, ok := claims[key].(string); ok {
		return v
	}
	return ""
}

func getSubjectClaim(claims jwt.MapClaims) string {
	if v, ok := claims["sub"].(string); ok {
		return v
	}
	if v, ok := claims["user_id"].(string); ok {
		return v
	}
	return ""
}

func getExpiresAtClaim(claims jwt.MapClaims) *jwt.NumericDate {
	if v, ok := claims["exp"].(float64); ok {
		t := time.Unix(int64(v), 0)
		return jwt.NewNumericDate(t)
	}
	return nil
}

func getIssuedAtClaim(claims jwt.MapClaims) *jwt.NumericDate {
	if v, ok := claims["iat"].(float64); ok {
		t := time.Unix(int64(v), 0)
		return jwt.NewNumericDate(t)
	}
	return nil
}

// ParseMapClaims parses a token string and returns the MapClaims if valid.
func (jm *JWTManager) ParseMapClaims(tokenString string) (jwt.MapClaims, error) {
	mc := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, mc, func(token *jwt.Token) (interface{}, error) {
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
	return mc, nil
}

// SetRSAPublicKey sets the RSA public key for RS256 token verification
func (jm *JWTManager) SetRSAPublicKey(pubKey *rsa.PublicKey) {
	jm.mu.Lock()
	defer jm.mu.Unlock()
	jm.rsaPublicKey = pubKey
}

// SetRSAPublicKeys replaces the full set of known JWKS signing keys, keyed by
// "kid". Safe to call repeatedly (e.g. from a periodic JWKS refresh) — this
// is how the server picks up Keycloak key rotation without a restart. The
// first key in the set (if any) also becomes the fallback used for tokens
// that carry no "kid" header.
func (jm *JWTManager) SetRSAPublicKeys(keys map[string]*rsa.PublicKey) {
	jm.mu.Lock()
	defer jm.mu.Unlock()
	jm.rsaPublicKeys = keys
	for _, k := range keys {
		jm.rsaPublicKey = k
		break
	}
}

// SetRSAPublicKeyPEM sets the RSA public key from a PEM-encoded string or byte slice
func (jm *JWTManager) SetRSAPublicKeyPEM(pemBytes []byte) error {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return fmt.Errorf("failed to parse PEM block containing public key")
	}
	pub, err := jwt.ParseRSAPublicKeyFromPEM(pemBytes)
	if err != nil {
		return fmt.Errorf("failed to parse RSA public key: %w", err)
	}
	jm.SetRSAPublicKey(pub)
	return nil
}

// ValidateToken validates a JWT token and returns claims
// ValidateToken validates a JWT token and returns claims.
//
// SECURITY: this must always cryptographically verify the signature. There is
// intentionally no "fall back to unverified claims" path — a previous version
// of this function accepted ANY token whose signature failed to verify (wrong
// key, expired JWKS cache, malformed alg) by silently re-parsing it without
// verification. That accepted attacker-forged claims (arbitrary user_id,
// tenant_id, roles, including global-admin) on every signature failure and is
// never reintroduced, including for local development — use a real HS256
// JWT_SECRET or a real Keycloak JWKS endpoint instead.
func (jm *JWTManager) ValidateToken(tokenString string) (*JWTClaims, error) {
	mc := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(tokenString, mc, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); ok {
			jm.mu.RLock()
			defer jm.mu.RUnlock()
			if kid, _ := token.Header["kid"].(string); kid != "" {
				if key, ok := jm.rsaPublicKeys[kid]; ok {
					return key, nil
				}
			}
			if jm.rsaPublicKey != nil {
				return jm.rsaPublicKey, nil
			}
			return nil, fmt.Errorf("no RSA public key configured for RS256 token verification")
		}
		return jm.secretKey, nil
	})
	if err != nil {
		return nil, err
	}
	userID := getStringClaim(mc, "user_id")
	if userID == "" {
		userID = getSubjectClaim(mc)
	}
	if userID == "" {
		userID = getStringClaim(mc, "preferred_username")
	}
	if userID == "" {
		userID = getStringClaim(mc, "email")
	}
	if userID == "" {
		return nil, fmt.Errorf("token missing user identifier")
	}
	email := getStringClaim(mc, "email")
	emailVerified, _ := mc["email_verified"].(bool)
	tenantID := getStringClaim(mc, "tenant_id")
	var tenantIDs []string
	if tids, ok := mc["tenant_ids"].([]interface{}); ok {
		for _, tid := range tids {
			if s, ok := tid.(string); ok {
				tenantIDs = append(tenantIDs, s)
			}
		}
	}
	if tenantID != "" && len(tenantIDs) == 0 {
		tenantIDs = []string{tenantID}
	}
	var roles []string
	if rs, ok := mc["roles"].([]interface{}); ok {
		for _, r := range rs {
			if s, ok := r.(string); ok {
				roles = append(roles, s)
			}
		}
	}
	if realmAccess, ok := mc["realm_access"].(map[string]interface{}); ok {
		if rs, ok := realmAccess["roles"].([]interface{}); ok {
			for _, r := range rs {
				if s, ok := r.(string); ok {
					roles = append(roles, s)
				}
			}
		}
	}
	if groups, ok := mc["groups"].([]interface{}); ok {
		for _, g := range groups {
			if s, ok := g.(string); ok {
				roles = append(roles, s)
			}
		}
	}

	// idpGroups preserves the raw Keycloak group paths (e.g. "/clients/acme/admin")
	// separately from roles, so downstream code can resolve them through the
	// group->role mapping table (security.identity_profile_mappings) instead of
	// treating a literal group name as a role string.
	idpGroups := parseStringListClaim(mc["groups"])
	idpGroups = append(idpGroups, parseStringListClaim(mc["idp_groups"])...)

	return &JWTClaims{
		UserID:        userID,
		Email:         email,
		EmailVerified: emailVerified,
		TenantID:      tenantID,
		TenantIDs:     tenantIDs,
		Roles:         roles,
		IdpGroups:     idpGroups,
		IssuedAt:      time.Now(),
	}, nil
}

// JWTClaims represents JWT token claims
type JWTClaims struct {
	UserID        string
	Email         string
	EmailVerified bool
	TenantID      string
	TenantIDs     []string
	Roles         []string
	IdpGroups     []string
	IssuedAt      time.Time
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
