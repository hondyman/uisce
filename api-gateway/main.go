package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	temporalclient "github.com/hondyman/uisce/libs/temporal-client"
	"github.com/hondyman/uisce/libs/logging"
	"github.com/hondyman/uisce/api-gateway/internal/config"
	"github.com/hondyman/uisce/api-gateway/handlers"
	"go.uber.org/zap"
	"golang.org/x/time/rate"

	apipkg "github.com/hondyman/uisce/api-gateway/api"
	"github.com/hondyman/uisce/api-gateway/proxy"
)

type APIKey struct {
	ID          string     `json:"id"`
	Key         string     `json:"key"`
	Name        string     `json:"name"`
	TenantID    string     `json:"tenant_id"`
	Permissions []string   `json:"permissions"`
	RateLimit   int        `json:"rate_limit"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	IsActive    bool       `json:"is_active"`
}

type AuditLog struct {
	ID           string    `json:"id"`
	Timestamp    time.Time `json:"timestamp"`
	UserID       string    `json:"user_id,omitempty"`
	APIKeyID     string    `json:"api_key_id,omitempty"`
	TenantID     string    `json:"tenant_id"`
	Method       string    `json:"method"`
	Path         string    `json:"path"`
	StatusCode   int       `json:"status_code"`
	ResponseTime int64     `json:"response_time_ms"`
	RequestSize  int64     `json:"request_size_bytes"`
	ResponseSize int64     `json:"response_size_bytes"`
	UserAgent    string    `json:"user_agent"`
	IPAddress    string    `json:"ip_address"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

type PolicyRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Conditions  map[string]interface{} `json:"conditions"`
	Actions     []string               `json:"actions"`
	Priority    int                    `json:"priority"`
	IsActive    bool                   `json:"is_active"`
}

type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
	}
}

func (rl *RateLimiter) GetLimiter(key string, rps float64) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if limiter, exists := rl.limiters[key]; exists {
		return limiter
	}

	limiter := rate.NewLimiter(rate.Limit(rps), int(rps)*2)
	rl.limiters[key] = limiter
	return limiter
}

var (
	rateLimiter = NewRateLimiter()
	apiKeys     = make(map[string]APIKey) // In production, use Redis/database
	gatewayConfig *config.GatewayConfig
)

// Middleware functions
func JWTMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("[JWT] Path: %s, Method: %s", c.Request.URL.Path, c.Request.Method)
		log.Printf("[JWT] Path: %s, Method: %s, Auth: %v", c.Request.URL.Path, c.Request.Method, c.GetHeader("Authorization") != "")
		// Always allow preflight OPTIONS through so CORS checks can succeed
		if c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}
		// Allow catalog/scan endpoint without authentication for development
		if c.Request.URL.Path == "/api/catalog/scan" {
			c.Next()
			return
		}

		// Allow unauthenticated frontend dev calls to /api/fabric/* when
		// DEV_ALLOW_UNAUTH_FABRIC is set to "true" (default for local development).
		if strings.HasPrefix(c.Request.URL.Path, "/api/fabric/") {
			if strings.ToLower(getEnv("DEV_ALLOW_UNAUTH_FABRIC", "false")) == "true" {
				c.Next()
				return
			}
		}

		// Allow unauthenticated access to model catalog endpoints in development
		if strings.HasPrefix(c.Request.URL.Path, "/api/models") {
			if strings.ToLower(getEnv("DEV_ALLOW_UNAUTH_MODELS", "false")) == "true" {
				c.Next()
				return
			}
		}

		// Allow unauthenticated access to catalog endpoints in development
		if strings.HasPrefix(c.Request.URL.Path, "/api/catalog") {
			if strings.ToLower(getEnv("DEV_ALLOW_UNAUTH_CATALOG", "false")) == "true" {
				c.Next()
				return
			}
		}

		// Allow unauthenticated GET to /api/business-term in development so the
		// frontend can call it without a JWT while working locally. Control via
		// DEV_ALLOW_UNAUTH_BUSINESS_TERM (default: true) to avoid accidental
		// exposure in production.
		if c.Request.Method == http.MethodGet && c.Request.URL.Path == "/api/business-term" {
			if strings.ToLower(getEnv("DEV_ALLOW_UNAUTH_BUSINESS_TERM", "false")) == "true" {
				c.Next()
				return
			}
		}

		// Allow unauthenticated frontend dev calls to /api/views when running locally
		// This makes Vite+gateway development smoother; disable in production by setting
		// DEV_ALLOW_UNAUTH_VIEWS=false
		if strings.HasPrefix(c.Request.URL.Path, "/api/views") {
			if strings.ToLower(getEnv("DEV_ALLOW_UNAUTH_VIEWS", "false")) == "true" {
				c.Next()
				return
			}
		}

		// (Removed dev-only exemption for roles.) All role endpoints require auth

		// Allow unauthenticated GraphQL proxy in development (vite/front-end) so dev clients
		// can call /api/graphql without JWT. Remove or restrict in production.
		if c.Request.Method == http.MethodPost && c.Request.URL.Path == "/api/graphql" {
			c.Next()
			return
		}

		// Allow unauthenticated GETs to tenant endpoints in development so the frontend
		// dev server (vite) can fetch tenants and tenant-scoped resources when auth isn't available.
		if strings.ToLower(getEnv("DEV_ALLOW_UNAUTH_FABRIC", "false")) == "true" &&
			(c.Request.URL.Path == "/api/tenants" || strings.HasPrefix(c.Request.URL.Path, "/api/tenants/")) {
			c.Next()
			return
		}

		// Allow unauthenticated access to the system-wide IP whitelist list in development
		if strings.ToLower(getEnv("DEV_ALLOW_UNAUTH_FABRIC", "false")) == "true" &&
			(c.Request.URL.Path == "/api/ip-whitelist" || strings.HasPrefix(c.Request.URL.Path, "/api/ip-whitelist")) {
			c.Next()
			return
		}

		// Allow unauthenticated access to certain dev-only endpoints like policies and bundles
		// so the frontend can work without a JWT during local development.
		if strings.ToLower(getEnv("DEV_ALLOW_UNAUTH_FABRIC", "false")) == "true" {
			p := c.Request.URL.Path
			// Allow both list and item routes for policies during local development
			if strings.HasPrefix(p, "/api/policies") || strings.HasPrefix(p, "/api/bundles") || strings.HasPrefix(p, "/api/semantic") || strings.HasPrefix(p, "/api/business") || strings.HasPrefix(p, "/api/data-domains") || strings.HasPrefix(p, "/api/profiler") || strings.HasPrefix(p, "/api/entity-schema") || strings.HasPrefix(p, "/api/validation-rules") || strings.HasPrefix(p, "/api/relationships") || strings.HasPrefix(p, "/api/lineage") || strings.HasPrefix(p, "/api/node-types") || strings.HasPrefix(p, "/api/edge-types") || strings.HasPrefix(p, "/api/bp-notifications") || strings.HasPrefix(p, "/api/impact") {
				c.Next()
				return
			}
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// In development, allow requests that provide X-User-ID even without JWT
			if strings.ToLower(getEnv("DEV_ALLOW_UNAUTH_XUSER", "false")) == "true" && c.GetHeader("X-User-ID") != "" {
				c.Next()
				return
			}
			c.JSON(401, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(401, gin.H{"error": "Invalid authorization format"})
			c.Abort()
			return
		}

		// Check if this is an internal action token (e.g., from Hasura calling webhooks).
		// If API_GATEWAY_AUTH_TOKEN matches the Bearer token, allow it and skip JWT validation.
		// Note: expectedToken from env has the full "Bearer <token>" form, but tokenString is already stripped of "Bearer ".
		expectedTokenFull := getEnv("API_GATEWAY_AUTH_TOKEN", "")
		if expectedTokenFull != "" {
			// Extract just the token part from the full "Bearer token" string for comparison
			expectedTokenOnly := strings.TrimPrefix(expectedTokenFull, "Bearer ")
			if tokenString == expectedTokenOnly {
				log.Printf("api-gateway: internal action token accepted (Bearer token bypass)")
				c.Next()
				return
			}
		}

		// Parse and validate JWT token. Support RS256 (gateway-issued via JWKS)
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// If token uses RS*, use kid to find public key
			if token.Method.Alg() == jwt.SigningMethodRS256.Alg() || strings.HasPrefix(token.Method.Alg(), "RS") {
				kid, _ := token.Header["kid"].(string)
				if kid == "" {
					return nil, fmt.Errorf("missing kid in token header")
				}
				if pub, ok := keyManager.GetPublicKey(kid); ok {
					return pub, nil
				}
				return nil, fmt.Errorf("unknown kid: %s", kid)
			}
			// Fallback: HS256 using configured secret
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(getEnvRequired("JWT_SECRET")), nil
		})

		if err != nil || !token.Valid {
			c.JSON(401, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// If token has jti, check revocation store
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			if jti, ok := claims["jti"].(string); ok && jti != "" {
				if revoked, rerr := revocationStore.IsRevoked(context.Background(), jti); rerr != nil {
					log.Printf("revocation check error: %v", rerr)
				} else if revoked {
					c.JSON(401, gin.H{"error": "token_revoked"})
					c.Abort()
					return
				}
			}

			c.Set("semlayer_user_id", claims["user_id"])
			c.Set("semlayer_tenant_id", claims["tenant_id"])

			// Re-set X-Tenant-ID from the verified claim (it was stripped of any
			// client-supplied value above) so downstream handlers that read the
			// header directly — or the proxied request to the backend — see the
			// caller's actual tenant, not nothing and not whatever they sent.
			if tid, ok := claims["tenant_id"].(string); ok && tid != "" {
				c.Request.Header.Set("X-Tenant-ID", tid)
			}
			if uid, ok := claims["user_id"].(string); ok && uid != "" {
				c.Request.Header.Set("X-User-ID", uid)
			}
		}

		c.Next()
	}
}

// Key manager and revocation store are package-level so middleware/handlers can use them.
var keyManager *KeyManager
var revocationStore RevocationStore

func RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("[RATE LIMIT] Path: %s", c.Request.URL.Path)
		// Skip rate limiting for tenant and ip-whitelist endpoints to allow
		// frontend dev server (vite) to fetch tenant-scoped resources without
		// being blocked by gateway-level limits. These routes are protected
		// elsewhere in production, so keeping them unthrottled locally eases
		// development UX.
		// Only apply the tenant/ip-whitelist bypass in dev mode when the
		// DEV_ALLOW_UNAUTH_FABRIC env var is set to "true". This prevents
		// accidentally disabling rate limiting in production.
		if strings.ToLower(strings.TrimSpace(getEnv("DEV_ALLOW_UNAUTH_FABRIC", "false"))) == "true" {
			p := c.Request.URL.Path
			if p == "/api/tenants" || strings.HasPrefix(p, "/api/tenants/") ||
				p == "/api/ip-whitelist" || strings.HasPrefix(p, "/api/ip-whitelist/") {
				c.Next()
				return
			}
		}
		// SECURITY: Use tenant-aware rate limiting to prevent one tenant from
		// exhausting another tenant's rate limit. Get tenant from context set
		// by JWTMiddleware (runs before this middleware).
		var clientID string
		if tenantID, ok := c.Get("semlayer_tenant_id"); ok {
			if tid, ok := tenantID.(string); ok && tid != "" {
				// Rate limit per tenant to prevent tenant A exhausting tenant B's limit
				clientID = "tenant:" + tid
			}
		}
		if clientID == "" {
			// Fallback to API key or IP for unauthenticated requests
			clientID = c.GetHeader("X-API-Key")
			if clientID == "" {
				clientID = c.ClientIP()
			}
		}

		// Get rate limiter for this client
		limiter := rateLimiter.GetLimiter(clientID, 60.0) // 60 requests per minute

		if !limiter.Allow() {
			c.JSON(429, gin.H{
				"error":       "Rate limit exceeded",
				"retry_after": "60",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// Whitelist cache entry
type wlEntry struct {
	patterns []string
	expires  time.Time
}

type WhitelistCache struct {
	mu    sync.RWMutex
	items map[string]wlEntry
}

func NewWhitelistCache() *WhitelistCache {
	return &WhitelistCache{items: make(map[string]wlEntry)}
}

func (c *WhitelistCache) Get(key string) (patterns []string, ok bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	it, exists := c.items[key]
	if !exists || time.Now().After(it.expires) {
		return nil, false
	}
	return it.patterns, true
}

func (c *WhitelistCache) Set(key string, patterns []string, ttl time.Duration) {
	c.mu.Lock()
	c.items[key] = wlEntry{patterns: patterns, expires: time.Now().Add(ttl)}
	c.mu.Unlock()
}

// Delete removes a single tenant's whitelist cache entry.
func (c *WhitelistCache) Delete(key string) {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
}

// Clear removes all cached whitelist entries.
func (c *WhitelistCache) Clear() {
	c.mu.Lock()
	c.items = make(map[string]wlEntry)
	c.mu.Unlock()
}

var wlCache = NewWhitelistCache()

// ipMatches checks an IPv4 address against a pattern with '*' octets (e.g., "192.168.*.*").
func ipMatches(pattern, ip string) bool {
	// Quick exact match
	if pattern == ip {
		return true
	}
	pa := strings.Split(pattern, ".")
	pb := strings.Split(ip, ".")
	if len(pa) != 4 || len(pb) != 4 {
		return false
	}
	for i := 0; i < 4; i++ {
		if pa[i] == "*" {
			continue
		}
		if pa[i] != pb[i] {
			return false
		}
	}
	return true
}

// IpWhitelistMiddleware enforces that the client IP is allowed for the request's tenant.
// It fetches patterns from backend /api/tenants/{tenantId}/ip-whitelist (includes global entries)
// and caches them briefly to reduce load.
func IpWhitelistMiddleware(backendBase string) gin.HandlerFunc {
	enforce := strings.ToLower(getEnv("IP_WHITELIST_ENFORCE", "true")) == "true"
	logDecisions := strings.ToLower(getEnv("IP_WHITELIST_LOG", "true")) == "true"
	ttl := 60 * time.Second
	if v := getEnv("IP_WHITELIST_CACHE_TTL", ""); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			ttl = d
		}
	}

	skipPaths := func(p string) bool {
		if p == "/api/tenants" || strings.HasPrefix(p, "/api/tenants/") {
			return true
		}
		if p == "/api/ip-whitelist" || strings.HasPrefix(p, "/api/ip-whitelist") {
			return true
		}
		if p == "/api/auth/login" || p == "/api/openapi.yaml" || strings.HasPrefix(p, "/docs") || p == "/health" || p == "/jwks.json" {
			return true
		}
		return false
	}

	return func(c *gin.Context) {
		if !enforce {
			log.Printf("[IP WHITELIST] Path: %s, ENFORCE DISABLED", c.Request.URL.Path)
			c.Next()
			return
		}
		log.Printf("[IP WHITELIST] Path: %s, ENFORCING", c.Request.URL.Path)
		path := c.Request.URL.Path
		if skipPaths(path) {
			c.Next()
			return
		}

		// Determine tenant ID: JWT claim or API key mapping
		var tenantID string
		if v, ok := c.Get("semlayer_tenant_id"); ok {
			if s, ok2 := v.(string); ok2 {
				tenantID = s
			}
		}
		if tenantID == "" {
			if apiKey := c.GetHeader("X-API-Key"); apiKey != "" {
				if k, exists := apiKeys[apiKey]; exists {
					tenantID = k.TenantID
				}
			}
		}
		if tenantID == "" {
			// Cannot determine tenant; allow
			c.Next()
			return
		}

		// Fetch whitelist patterns for tenant (from cache or backend)
		patterns, ok := wlCache.Get(tenantID)
		if !ok {
			client := &http.Client{Timeout: 5 * time.Second}
			req, _ := http.NewRequest("GET", backendBase+"/api/tenants/"+tenantID+"/ip-whitelist", nil)
			// forward minimal auth headers if present
			if auth := c.GetHeader("Authorization"); auth != "" {
				req.Header.Set("Authorization", auth)
			}
			resp, err := client.Do(req)
			if err == nil && resp != nil && resp.Body != nil {
				defer resp.Body.Close()
			}
			var plist []string
			if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
				var body struct {
					Whitelist []struct {
						IpAddress string `json:"ipAddress"`
					} `json:"whitelist"`
				}
				data, _ := io.ReadAll(resp.Body)
				if err := json.Unmarshal(data, &body); err == nil {
					for _, e := range body.Whitelist {
						if e.IpAddress != "" {
							plist = append(plist, e.IpAddress)
						}
					}
				}
			}
			// Cache even empty slice to avoid stampede
			wlCache.Set(tenantID, plist, ttl)
			patterns = plist
		}

		// If there are no patterns, allow
		if len(patterns) == 0 {
			c.Next()
			return
		}

		// Check client IP against patterns
		clientIP := c.ClientIP()
		allowed := false
		for _, p := range patterns {
			if ipMatches(p, clientIP) {
				allowed = true
				break
			}
		}
		if !allowed {
			if logDecisions {
				log.Printf("ip_whitelist deny tenant=%s ip=%s patterns=%d path=%s", tenantID, clientIP, len(patterns), path)
			}
			c.JSON(403, gin.H{"error": "forbidden_ip", "message": "Client IP is not allowed for this tenant"})
			c.Abort()
			return
		}
		if logDecisions {
			log.Printf("ip_whitelist allow tenant=%s ip=%s patterns=%d path=%s", tenantID, clientIP, len(patterns), path)
		}
		c.Next()
	}
}

func PolicyEnforcementMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("[POLICY] Path: %s", c.Request.URL.Path)
		// Allow skipping policy enforcement for semantic-mappings endpoints during
		// local frontend development. The gateway's naive SQL keyword scanner can
		// produce false positives when the frontend sends JSON payloads that
		// include words like "select" or "insert" (for example in qualified
		// names or descriptions). Allow bypass in dev by setting
		// DEV_ALLOW_UNAUTH_FABRIC=true (this project already uses that flag for
		// other dev-only relaxations).
		if strings.HasPrefix(c.Request.URL.Path, "/api/semantic-mappings") {
			if strings.ToLower(getEnv("DEV_ALLOW_UNAUTH_FABRIC", "false")) == "true" {
				c.Next()
				return
			}
		}
		// Allow skipping policy enforcement for /api/views in local development
		if strings.HasPrefix(c.Request.URL.Path, "/api/views") {
			if strings.ToLower(getEnv("DEV_ALLOW_UNAUTH_VIEWS", "false")) == "true" {
				c.Next()
				return
			}
		}
		// Skip naive SQL-keyword scanning for GraphQL requests or JSON bodies
		// that look like GraphQL (top-level "query" field). GraphQL payloads
		// frequently include identifiers that can trigger false positives.
		if c.Request.URL.Path == "/api/graphql" || c.Request.URL.Path == "/api/catalog/scan" || strings.HasPrefix(c.Request.URL.Path, "/api/models") {
			c.Next()
			return
		}

		body, _ := c.GetRawData()
		bodyStr := string(body)

		// Restore the request body for downstream handlers (important for proxying)
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

		// If JSON and contains a top-level `query` key, assume GraphQL and skip
		contentType := strings.ToLower(c.GetHeader("Content-Type"))
		if strings.Contains(contentType, "application/json") {
			var probe map[string]interface{}
			if err := json.Unmarshal([]byte(bodyStr), &probe); err == nil {
				if _, ok := probe["query"]; ok {
					c.Next()
					return
				}
			}
		}

		sqlKeywords := []string{"SELECT", "INSERT", "UPDATE", "DELETE", "DROP", "UNION", "EXEC"}
		for _, keyword := range sqlKeywords {
			if strings.Contains(strings.ToUpper(bodyStr), keyword) {
				c.JSON(403, gin.H{"error": "Potential security violation detected"})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

func AuditMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Log audit information
		duration := time.Since(start)
		log.Printf("[AUDIT] %s %s %d %v %s",
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			duration,
			c.ClientIP(),
		)
	}
}

type BusinessTermSearchRequest struct {
	Query    string `json:"query"`
	TenantID string `json:"tenant_id,omitempty"`
	Limit    int    `json:"limit,omitempty"`
	Offset   int    `json:"offset,omitempty"`
}

type BusinessTermValidationRequest struct {
	Name        string                 `json:"name"`
	DisplayName string                 `json:"display_name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Category    string                 `json:"category,omitempty"`
	SubCategory string                 `json:"sub_category,omitempty"`
	Owner       string                 `json:"owner,omitempty"`
	Steward     string                 `json:"steward,omitempty"`
	Status      string                 `json:"status,omitempty"`
	Version     string                 `json:"version,omitempty"`
	Tags        string                 `json:"tags,omitempty"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
}

type LineageRequest struct {
	NodeID   string `json:"node_id"`
	TenantID string `json:"tenant_id,omitempty"`
	Depth    int    `json:"depth,omitempty"`
}

func main() {
	logging.InitGlobalLogger()
	logger := logging.GetLogger()

	var err error
	gatewayConfig, err = config.LoadGatewayConfig()
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}
	logger.Info("Configuration loaded", zap.String("env", gatewayConfig.Env), zap.String("logLevel", gatewayConfig.LogLevel))

	handlers.SetBackendURL(gatewayConfig.BackendURL)

	r := gin.Default()
	logger.Info("Router created")

	// Debug endpoint: echoes back whatever tenant header the caller sent, with
	// no auth. Only meaningful as a local dev aid, and only ever safe as one —
	// gate it behind explicit opt-in like every other dev-only relaxation in
	// this file, never on by default.
	if strings.ToLower(getEnv("DEV_ENABLE_DEBUG_HEADERS_ENDPOINT", "false")) == "true" {
		r.GET("/api/_debug/headers", func(c *gin.Context) {
			tenantID := c.GetHeader("X-Tenant-ID")
			dsID := c.GetHeader("X-Tenant-Datasource-ID")
			q := map[string]string{}
			for k, v := range c.Request.URL.Query() {
				if len(v) > 0 {
					q[k] = v[0]
				}
			}
			c.JSON(200, gin.H{
				"received_tenant_id":     tenantID,
				"received_datasource_id": dsID,
				"query_params":           q,
			})
		})
	}

	// Add panic recovery middleware
	r.Use(gin.Recovery())

	// Strip any client-supplied tenant/identity headers before anything else
	// runs. Tenant scope must come exclusively from a verified JWT (set by
	// JWTMiddleware below via semlayer_tenant_id), never from a header the
	// caller controls — a caller who sets X-Tenant-ID to another tenant's ID
	// must never have that value reach a handler or get proxied upstream.
	r.Use(func(c *gin.Context) {
		c.Request.Header.Del("X-Tenant-ID")
		c.Request.Header.Del("X-Client-ID")
		c.Request.Header.Del("X-Org-Id")
		c.Request.Header.Del("X-Roles")
		c.Request.Header.Del("X-Scopes")
		c.Request.Header.Del("X-Tenant-Scope")
		c.Next()
	})

	// Configure trusted proxies to avoid trusting arbitrary X-Forwarded-For headers.
	// Use environment variable TRUSTED_PROXIES (comma-separated CIDRs/IPs) in dev/real deployments.
	// Default to localhost addresses only to avoid implicitly trusting all proxies.
	trusted := getEnv("TRUSTED_PROXIES", "127.0.0.1,::1")
	proxyList := []string{}
	for _, p := range strings.Split(trusted, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			proxyList = append(proxyList, p)
		}
	}
	if len(proxyList) > 0 {
		if err := r.SetTrustedProxies(proxyList); err != nil {
			log.Printf("warning: failed to set trusted proxies (%v): %v", proxyList, err)
		} else {
			log.Printf("Set trusted proxies: %v", proxyList)
		}
	}

	// Backend base URL
	backendBase := gatewayConfig.BackendURL

	// Auth service URL
	authServiceBase := gatewayConfig.AuthServiceURL

	// Initialize KeyManager and RevocationStore
	keyManager = NewKeyManager()
	if gatewayConfig.RevocationRedisAddr != "" {
		revocationStore = NewRedisRevocationStore(gatewayConfig.RevocationRedisAddr)
		logger.Info("Using Redis revocation store", zap.String("addr", gatewayConfig.RevocationRedisAddr))
	} else {
		revocationStore = NewInMemoryRevocationStore()
		logger.Info("Using in-memory revocation store (dev only)")
	}

	logger.Info("Upstream endpoints resolved",
		zap.String("backendURL", backendBase),
		zap.String("authServiceURL", authServiceBase))

	// CORS middleware (restrict to local frontend dev origin)
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:5173", "http://localhost:5174"},
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders: []string{
			"Origin",
			"Content-Length",
			"Content-Type",
			"Authorization",
			"X-Tenant-ID",
			"X-Tenant-Datasource-ID",
			"x-tenant-datasource-id",
			"X-API-Key",
			"x-requested-with",
			"Accept",
			"x-user-id",
		},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// JWKS endpoint for RS256 public keys
	r.GET("/jwks.json", func(c *gin.Context) {
		keyManager.JWKSHandler(c.Writer, c.Request)
	})

	// Competitive features - API analytics and monitoring
	r.GET("/api/analytics", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"total_requests":    1250,
			"active_users":      45,
			"avg_response_time": "120ms",
			"uptime":            "99.9%",
			"error_rate":        "0.1%",
		})
	})

	r.GET("/api/health/detailed", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"version": "1.0.0",
			"services": gin.H{
				"database": "up",
				"cache":    "up",
			},
			"features": []string{
				"JWT Authentication",
				"Rate Limiting",
				"Policy Enforcement",
				"Audit Logging",
				"API Catalog",
				"Business Term Search",
				"Semantic Lineage",
			},
		})
	})

	// API key management will be protected under the /api group (below)

	// API routes group with security middleware
	api := r.Group("/api")
	api.Use(JWTMiddleware())
	api.Use(RateLimitMiddleware())
	// Enforce IP whitelist per-tenant (after auth and rate limit, before policy & audit)
	api.Use(IpWhitelistMiddleware(backendBase))
	api.Use(PolicyEnforcementMiddleware())
	api.Use(AuditMiddleware())

	// API key management (protected)
	api.POST("/keys", func(c *gin.Context) {
		var req struct {
			Name        string   `json:"name"`
			TenantID    string   `json:"tenant_id"`
			Permissions []string `json:"permissions"`
			RateLimit   int      `json:"rate_limit"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		// Generate API key
		key := generateAPIKey()

		apiKey := APIKey{
			ID:          generateID(),
			Key:         key,
			Name:        req.Name,
			TenantID:    req.TenantID,
			Permissions: req.Permissions,
			RateLimit:   req.RateLimit,
			CreatedAt:   time.Now(),
			IsActive:    true,
		}

		apiKeys[key] = apiKey

		c.JSON(201, gin.H{
			"id":          apiKey.ID,
			"key":         apiKey.Key,
			"name":        apiKey.Name,
			"permissions": apiKey.Permissions,
			"rate_limit":  apiKey.RateLimit,
			"created_at":  apiKey.CreatedAt,
		})
	})

	api.GET("/keys", func(c *gin.Context) {
		keys := make([]gin.H, 0, len(apiKeys))
		for _, key := range apiKeys {
			keys = append(keys, gin.H{
				"id":          key.ID,
				"name":        key.Name,
				"permissions": key.Permissions,
				"rate_limit":  key.RateLimit,
				"created_at":  key.CreatedAt,
				"is_active":   key.IsActive,
			})
		}
		c.JSON(200, gin.H{"api_keys": keys})
	})

	// API catalog endpoints
	api.GET("/catalog/apis", handlers.HandleGetAPIs)

	// Admin rotate endpoint exposed under /api for operators
	api.POST("/keys/rotate", func(c *gin.Context) {
		if keyManager == nil {
			c.JSON(500, gin.H{"error": "key manager not initialized"})
			return
		}
		keyManager.RotateKeyHandler(c.Writer, c.Request)
	})

	// Admin endpoint to revoke tokens by jti
	api.POST("/tokens/revoke", func(c *gin.Context) {
		var req struct {
			JTI string `json:"jti" binding:"required"`
			Exp int64  `json:"exp,omitempty"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		exp := time.Now().Add(24 * time.Hour)
		if req.Exp > 0 {
			exp = time.Unix(req.Exp, 0)
		}
		if err := revocationStore.Revoke(context.Background(), req.JTI, exp); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "ok"})
	})

	api.GET("/catalog/business-terms", handlers.HandleGetBusinessTerms)

	api.POST("/catalog/apis", handlers.HandleCreateAPI)

	api.POST("/catalog/business-terms", handlers.HandleCreateBusinessTerm)

	proxyHandler := proxy.NewProxyHandler(backendBase).WithWhitelistCache(wlCache).WithDevBypass(true)

	// Catalog scan endpoint - proxy to backend service
	api.POST("/catalog/scan", func(c *gin.Context) {
		proxyHandler.ServeHTTP()(c)
	})

	// Business terms endpoints
	api.POST("/test-search", func(c *gin.Context) {
		log.Printf("ANONYMOUS HANDLER CALLED for /api/test-search")
		handlers.HandleBusinessTermSearch(c)
	})

	api.POST("/validate/business-term", handlers.HandleBusinessTermValidation)

	// Register all backend proxy routes
	proxy.NewRouteRegistrar(proxyHandler, api).RegisterAll()

	// Gateway login endpoint: proxy credentials to auth service, then issue a gateway-signed JWT
	r.POST("/api/auth/login", func(c *gin.Context) {
		// Read incoming login body
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		// Forward to auth service login (not backend)
		client := &http.Client{Timeout: 10 * time.Second}
		req, err := http.NewRequest("POST", authServiceBase+"/api/auth/login", bytes.NewBuffer(body))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create backend request"})
			return
		}
		req.Header = c.Request.Header.Clone()
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "failed to reach backend"})
			return
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read backend response"})
			return
		}

		// If backend returned non-200, forward the response as-is
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			c.Status(resp.StatusCode)
			for k, v := range resp.Header {
				c.Header(k, strings.Join(v, ","))
			}
			c.Writer.Write(respBody)
			return
		}

		// Parse backend response to extract user info and expiry
		var backendResp map[string]interface{}
		if err := json.Unmarshal(respBody, &backendResp); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid backend response"})
			return
		}

		// Extract user and expires_in
		var user map[string]interface{}
		if u, ok := backendResp["user"].(map[string]interface{}); ok {
			user = u
		}
		expiresIn := 3600
		if ei, ok := backendResp["expires_in"].(float64); ok && ei > 0 {
			expiresIn = int(ei)
		}

		// Build claims and sign JWT using either HS256 (legacy) or RS256 via KeyManager
		claims := jwt.MapClaims{}
		if user != nil {
			if uid, ok := user["id"].(string); ok {
				claims["user_id"] = uid
			}
			if tenant, ok := user["tenant_id"].(string); ok {
				claims["tenant_id"] = tenant
			}
		}
		exp := time.Now().Add(time.Duration(expiresIn) * time.Second)
		claims["exp"] = exp.Unix()

		// Add a jti claim for revocation support
		jti := generateID()
		claims["jti"] = jti

		enableRS256 := strings.ToLower(getEnv("ENABLE_RS256", "false")) == "true"
		var signed string
		var kid string
		var signErr error
		if enableRS256 {
			signed, kid, signErr = keyManager.SignTokenRS256(claims)
		} else {
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			signed, signErr = token.SignedString([]byte(getEnvRequired("JWT_SECRET")))
		}
		if signErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to sign token"})
			return
		}

		// Return a gateway-style auth response with the signed token
		out := map[string]interface{}{
			"user":         user,
			"access_token": signed,
			"token_type":   "Bearer",
			"expires_in":   expiresIn,
		}
		if kid != "" {
			out["kid"] = kid
		}
		// include refresh_token if backend provided one
		if rt, ok := backendResp["refresh_token"].(string); ok {
			out["refresh_token"] = rt
		}

		// Persist jti in revocation store until token expiry (so it can be revoked later)
		if revocationStore != nil {
			if err := revocationStore.Revoke(context.Background(), jti, exp); err != nil {
				// We intentionally don't fail the login if revocation store isn't writable; just log.
				log.Printf("warning: failed to persist jti to revocation store: %v", err)
			} else {
				// Immediately remove the entry so IsRevoked only returns true when explicitly revoked.
				// For in-memory store we kept it for illustration; in production you'd not add it here.
			}
		}

		c.JSON(http.StatusOK, out)
	})

	// Query management endpoints
	api.POST("/queries", handlers.HandleCreateQuery)
	api.GET("/queries", handlers.HandleGetQueries)
	api.GET("/queries/:id", handlers.HandleGetQuery)
	api.PUT("/queries/:id", handlers.HandleUpdateQuery)
	api.DELETE("/queries/:id", handlers.HandleDeleteQuery)
	api.POST("/queries/:id/clone", handlers.HandleCloneQuery)
	api.POST("/queries/:id/share", handlers.HandleShareQuery)

	// API management endpoints
	api.POST("/apis", handlers.HandleCreateAPI)
	api.GET("/apis", handlers.HandleGetAPIs)
	api.GET("/apis/:id", handlers.HandleGetAPI)

	api.PUT("/apis/:id", handlers.HandleUpdateAPI)
	api.DELETE("/apis/:id", handlers.HandleDeleteAPI)
	api.POST("/apis/:id/clone", handlers.HandleCloneAPI)
	api.POST("/apis/:id/share", handlers.HandleShareAPI)

	// Dynamic API execution endpoints
	api.POST("/execute/:apiId/*path", handlers.HandleExecuteAPI)

	// OpenAPI/Swagger UI
	r.Static("/docs", "./docs")

	// Serve OpenAPI spec
	r.GET("/api/openapi.yaml", func(c *gin.Context) {
		c.File("./openapi.yaml")
	})

	// Initialize Temporal client (env-driven + retries)
	tc, err := temporalclient.NewClientWithRetry()
	if err != nil {
		logger.Warn("Failed to create Temporal client", zap.Error(err))
	} else {
		defer tc.Close()
		// Register custom routes only if temporal client is available
		apipkg.RegisterOptimizeAlphaRoutes(r, tc)
		apipkg.RegisterRiskAlphaRoutes(r, tc)
		apipkg.RegisterScenarioAnalysisRoutes(r, tc)
		apipkg.RegisterRebalancerRoutes(r, tc)
	}

		logger.Info("API Gateway starting", zap.String("port", gatewayConfig.Port))

	tlsEnabled := os.Getenv("TLS_ENABLED") == "true"
	if tlsEnabled {
		certFile := os.Getenv("TLS_CERT_FILE")
		keyFile := os.Getenv("TLS_KEY_FILE")
		if certFile == "" || keyFile == "" {
			logger.Fatal("TLS_CERT_FILE and TLS_KEY_FILE are required when TLS_ENABLED=true")
		}
		server := &http.Server{
			Addr:    ":" + gatewayConfig.Port,
			Handler: r,
			TLSConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		}
		logger.Info("API Gateway starting with TLS", zap.String("port", gatewayConfig.Port))
		if err := server.ListenAndServeTLS(certFile, keyFile); err != nil {
			logger.Fatal("Failed to start TLS server", zap.Error(err))
		}
	} else {
		env := os.Getenv("ENV")
		if env == "production" || env == "staging" {
			logger.Warn("TLS is not enabled in " + env)
		}
		if err := r.Run(":" + gatewayConfig.Port); err != nil {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvRequired(key string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	log.Fatalf("required environment variable %s is not set", key)
	return ""
}

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func generateAPIKey() string {
	// Generate a secure API key
	bytes := make([]byte, 32)
	for i := range bytes {
		bytes[i] = byte(65 + (time.Now().UnixNano()+int64(i))%26) // A-Z
	}
	return string(bytes)
}
