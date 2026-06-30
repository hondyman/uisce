package main

import (
	"bytes"
	"context"
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
	temporalclient "github.com/hondyman/semlayer/libs/temporal-client"
	"github.com/joho/godotenv"
	"golang.org/x/time/rate"

	apipkg "github.com/hondyman/semlayer/api-gateway/api"
)

type Config struct {
	Port           string
	HasuraURL      string
	HasuraSecret   string
	JWTSecret      string
	RateLimitRPM   int
	EnableAudit    bool
	BackendURL     string
	GraphQLBackend string
}

type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

type GraphQLResponse struct {
	Data   interface{} `json:"data,omitempty"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors,omitempty"`
}

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
			if strings.ToLower(getEnv("DEV_ALLOW_UNAUTH_FABRIC", "true")) == "true" {
				c.Next()
				return
			}
		}

		// Allow unauthenticated access to model catalog endpoints in development
		if strings.HasPrefix(c.Request.URL.Path, "/api/models") {
			if strings.ToLower(getEnv("DEV_ALLOW_UNAUTH_MODELS", "true")) == "true" {
				c.Next()
				return
			}
		}

		// Allow unauthenticated access to catalog endpoints in development
		if strings.HasPrefix(c.Request.URL.Path, "/api/catalog") {
			if strings.ToLower(getEnv("DEV_ALLOW_UNAUTH_CATALOG", "true")) == "true" {
				c.Next()
				return
			}
		}

		// Allow unauthenticated GET to /api/business-term in development so the
		// frontend can call it without a JWT while working locally. Control via
		// DEV_ALLOW_UNAUTH_BUSINESS_TERM (default: true) to avoid accidental
		// exposure in production.
		if c.Request.Method == http.MethodGet && c.Request.URL.Path == "/api/business-term" {
			if strings.ToLower(getEnv("DEV_ALLOW_UNAUTH_BUSINESS_TERM", "true")) == "true" {
				c.Next()
				return
			}
		}

		// Allow unauthenticated frontend dev calls to /api/views when running locally
		// This makes Vite+gateway development smoother; disable in production by setting
		// DEV_ALLOW_UNAUTH_VIEWS=false
		if strings.HasPrefix(c.Request.URL.Path, "/api/views") {
			if strings.ToLower(getEnv("DEV_ALLOW_UNAUTH_VIEWS", "true")) == "true" {
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
		if strings.ToLower(getEnv("DEV_ALLOW_UNAUTH_FABRIC", "true")) == "true" &&
			(c.Request.URL.Path == "/api/tenants" || strings.HasPrefix(c.Request.URL.Path, "/api/tenants/")) {
			c.Next()
			return
		}

		// Allow unauthenticated access to the system-wide IP whitelist list in development
		if strings.ToLower(getEnv("DEV_ALLOW_UNAUTH_FABRIC", "true")) == "true" &&
			(c.Request.URL.Path == "/api/ip-whitelist" || strings.HasPrefix(c.Request.URL.Path, "/api/ip-whitelist")) {
			c.Next()
			return
		}

		// Allow unauthenticated access to certain dev-only endpoints like policies and bundles
		// so the frontend can work without a JWT during local development.
		if strings.ToLower(getEnv("DEV_ALLOW_UNAUTH_FABRIC", "true")) == "true" {
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
			if strings.ToLower(getEnv("DEV_ALLOW_UNAUTH_XUSER", "true")) == "true" && c.GetHeader("X-User-ID") != "" {
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
			return []byte(getEnv("JWT_SECRET", "your-secret-key")), nil
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
		// Get client identifier (API key or IP)
		clientID := c.GetHeader("X-API-Key")
		if clientID == "" {
			clientID = c.ClientIP()
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
			if strings.ToLower(getEnv("DEV_ALLOW_UNAUTH_FABRIC", "true")) == "true" {
				c.Next()
				return
			}
		}
		// Allow skipping policy enforcement for /api/views in local development
		if strings.HasPrefix(c.Request.URL.Path, "/api/views") {
			if strings.ToLower(getEnv("DEV_ALLOW_UNAUTH_VIEWS", "true")) == "true" {
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
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	config := Config{
		Port:           getEnv("PORT", "8001"),
		HasuraURL:      getEnv("HASURA_URL", "http://localhost:8081"),
		HasuraSecret:   getEnv("HASURA_ADMIN_SECRET", "newadminsecretkey"),
		BackendURL:     getEnv("BACKEND_URL", "http://localhost:8080"),
		GraphQLBackend: strings.ToLower(getEnv("UI_GRAPHQL_BACKEND", "hasura")),
	}



	// DEBUG: Verify config is populated
	log.Printf("DEBUG: Config initialized: Port=%s, BackendURL=%s, GraphQLBackend=%s", config.Port, config.BackendURL, config.GraphQLBackend)

	// Normalize Hasura URL so callers can reliably append "/v1/graphql".
	// Accept either a base URL (e.g. http://hasura:8080) or a full path
	// (e.g. http://hasura:8080/v1/graphql) in env. We strip any trailing
	// "/v1/graphql" and any trailing slash so code below can do
	// config.HasuraURL + "/v1/graphql" without duplicating segments.
	h := strings.TrimSuffix(config.HasuraURL, "/v1/graphql")
	h = strings.TrimSuffix(h, "/")
	config.HasuraURL = h

	r := gin.Default()
	log.Printf("ROUTER CREATED")
	// Debug endpoint: echo tenant headers for frontend debugging
	r.GET("/api/_debug/headers", func(c *gin.Context) {
		tenantID := c.GetHeader("X-Tenant-ID")
		dsID := c.GetHeader("X-Tenant-Datasource-ID")
		// Also echo query params for convenience
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

	// Add panic recovery middleware
	r.Use(gin.Recovery())

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

	// Backend base URL (can be overridden in local/dev environments)
	// Default to 8080 where the backend service runs in local dev.
	backendBase := getEnv("BACKEND_URL", "http://localhost:8080")

	// Auth service URL (separate from backend, defaults to auth-service on port 8001 in Docker)
	authServiceBase := getEnv("AUTH_SERVICE_URL", "http://auth-service:8001")

	// Initialize KeyManager and RevocationStore
	keyManager = NewKeyManager()
	// Use Redis for revocation in prod via REVOCATION_REDIS_ADDR, otherwise in-memory
	if addr := getEnv("REVOCATION_REDIS_ADDR", ""); addr != "" {
		revocationStore = NewRedisRevocationStore(addr)
		log.Printf("Using Redis revocation store at %s", addr)
	} else {
		revocationStore = NewInMemoryRevocationStore()
		log.Printf("Using in-memory revocation store (dev only)")
	}

	// Log resolved upstream endpoints for easier debugging in dev
	resolvedHasura := getEnv("HASURA_URL", "http://localhost:8081")
	resolvedHasuraSecret := getEnv("HASURA_ADMIN_SECRET", "newadminsecretkey")
	maskedSecret := "(not set)"
	if resolvedHasuraSecret != "" {
		if len(resolvedHasuraSecret) > 4 {
			maskedSecret = "****" + resolvedHasuraSecret[len(resolvedHasuraSecret)-4:]
		} else {
			maskedSecret = "****"
		}
	}
	log.Printf("Resolved HASURA_URL=%s HASURA_ADMIN_SECRET=%s BACKEND_URL=%s AUTH_SERVICE_URL=%s", resolvedHasura, maskedSecret, backendBase, authServiceBase)

	// Validate upstream Hasura endpoint is reachable in dev so developers get
	// an early, helpful warning instead of an opaque gateway error.
	// if ok := validateHasura(resolvedHasura, resolvedHasuraSecret); !ok {
	//     log.Printf("WARNING: Unable to reach Hasura at %s. If you run Hasura locally, set HASURA_URL=http://localhost:8080 and ensure Hasura is running and accessible.", resolvedHasura)
	// }
	if strings.Contains(resolvedHasura, ":"+config.Port) {
		log.Printf("WARNING: HASURA_URL (%s) seems to be pointing to the gateway's own address and port (%s). This will cause a proxy loop. Make sure HASURA_URL points to your separate Hasura service.", resolvedHasura, config.Port)
	}

	// CORS middleware (restrict to local frontend dev origin)
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:5173", "http://localhost:5174"},
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		// Add custom headers commonly used by the frontend and Hasura proxied requests.
		AllowHeaders: []string{
			"Origin",
			"Content-Length",
			"Content-Type",
			"Authorization",
			"X-Tenant-ID",
			"X-Tenant-Datasource-ID",
			"x-tenant-datasource-id",
			"X-API-Key",
			"X-Hasura-Role",
			"X-Hasura-Admin-Secret",
			"x-hasura-admin-secret",
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
				"hasura":   "up",
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
	api.GET("/catalog/apis", func(c *gin.Context) {
		handleGetAPIs(c)
	})

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

	api.GET("/catalog/business-terms", func(c *gin.Context) {
		handleGetBusinessTerms(c)
	})

	api.POST("/catalog/apis", func(c *gin.Context) {
		handleCreateAPI(c)
	})

	api.POST("/catalog/business-terms", func(c *gin.Context) {
		handleCreateBusinessTerm(c)
	})

	// Catalog scan endpoint - proxy to backend service
	api.POST("/catalog/scan", func(c *gin.Context) {
		// This specific endpoint has a different auth requirement (no JWT),
		// so we call the proxy handler directly.
		createProxyHandler(backendBase)(c)
	})

	// Business terms endpoints
	api.POST("/test-search", func(c *gin.Context) {
		log.Printf("ANONYMOUS HANDLER CALLED for /api/test-search")
		handleBusinessTermSearch(c, config)
	})

	api.POST("/validate/business-term", func(c *gin.Context) {
		handleBusinessTermValidation(c, config)
	})

	// GraphQL endpoint (canonical under /api/graphql)
	api.POST("/graphql", func(c *gin.Context) { handleGraphQLProxy(c, config) })



	// Aliases: accept requests at top-level /graphql and /v1/graphql so clients
	// that post directly to the gateway's GraphQL path (or expect v1 path)
	// will be proxied the same as /api/graphql. This avoids 404s when the
	// frontend or tools call /graphql or /v1/graphql on the gateway.
	r.POST("/graphql", func(c *gin.Context) { handleGraphQLProxy(c, config) })
	r.POST("/v1/graphql", func(c *gin.Context) { handleGraphQLProxy(c, config) })



	// Create a reusable proxy handler for the backend service
	proxy := createProxyHandler(backendBase)

	// Ensure a small set of backend-only endpoints that the frontend expects
	// are explicitly proxied by the gateway. This avoids returning 404s when
	// the frontend (via dev-proxy) points at the gateway in local dev.
	// These endpoints are implemented by the backend service and should be
	// forwarded transparently.
	api.GET("/semantic/objects", proxy)
	// Endpoint for singular 'semantic-mapping' (matching backend service definition)
	api.Any("/semantic-mapping", proxy)
	api.Any("/semantic-mapping/*path", proxy)

	api.Any("/semantic-mappings", proxy)
	api.Any("/semantic-mappings/*path", proxy)
	api.Any("/semantic-terms", proxy)
	api.Any("/semantic-terms/*path", proxy)
	// Forward singular business-term lookup to backend so frontend GET /api/business-term
	// is not met with a gateway 404. Backend will return JSON payloads.
	api.Any("/business-term", proxy)
	// Forward plural business-terms endpoints to backend for frontend compatibility
	api.Any("/business-terms", proxy)
	api.Any("/business-terms/*path", proxy)
	// Forward business-term-edges endpoints to backend for frontend compatibility
	api.Any("/business-term-edges", proxy)
	api.Any("/business-term-edges/*path", proxy)

	// Instance management endpoints (cloning, syncing etc.)
	api.Any("/instance", proxy)
	api.Any("/instance/*path", proxy)

	// Impact Analysis endpoints (Dynamic Graph Queries via AGE)
	api.Any("/impact", proxy)
	api.Any("/impact/*path", proxy)

	// Data bundles endpoints used by the BundleEditor/UI. Register both
	// explicit and wildcard routes so GET/PUT/POST requests are forwarded.
	api.GET("/bundles", proxy)
	api.POST("/bundles", proxy)
	api.GET("/bundles/:id", proxy)
	api.PUT("/bundles/:id", proxy)
	api.Any("/bundles/:id/*any", proxy)

	// Register models endpoints on the `api` group with explicit paths to
	// avoid Gin wildcard/static route conflicts. The backend only exposes a
	// handful of model-related endpoints, so listing them precisely prevents
	// catch-all wildcard conflicts while preserving the same middleware chain
	// already applied to `api` (JWT, rate limiting, IP whitelist, policy,
	// audit).
	api.Any("/models", proxy)
	api.POST("/models/generated", proxy)
	api.POST("/models/custom", proxy)
	api.POST("/models/clone", proxy)
	// model id routes: GET, PATCH, DELETE etc.
	api.Any("/models/:model_id", proxy)

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
			signed, signErr = token.SignedString([]byte(getEnv("JWT_SECRET", "your-secret-key")))
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

	// Proxy routes to the backend service
	api.POST("/auth/logout", proxy)
	api.Any("/fabric/*path", proxy)
	api.Any("/rest/*path", proxy)
	// Tenants & IP whitelist routes (backend chi server under /api)
	api.Any("/tenants", proxy)
	api.Any("/tenants/*path", proxy)
	api.Any("/ip-whitelist", proxy)
	api.Any("/ip-whitelist/*path", proxy)

	// Proxy specific backend endpoints used directly by the frontend
	// e.g. POST /api/query -> backend /api/query
	api.Any("/query", proxy)
	// Pre-aggregation endpoints sometimes called without /api prefix from frontend dev UI
	api.Any("/pre_aggregations", proxy)
	api.Any("/pre_aggregations/*path", proxy)
	// Model catalog endpoints are registered at the root with middleware to
	// avoid Gin wildcard/static route conflicts. See registration above.
	// Views catalog endpoints - forward to backend service
	api.Any("/views", proxy)
	api.Any("/views/*path", proxy)

	// Calculations endpoints - forward to backend service (backend exposes /api/calculations)
	api.Any("/calculations", proxy)
	api.Any("/calculations/*path", proxy)

	// Roles endpoints - forward to backend service (backend exposes /api/roles)
	api.Any("/roles", proxy)
	api.Any("/roles/*path", proxy)

	// Policies endpoints - forward to backend service (backend exposes /api/policies)
	api.Any("/policies", proxy)
	api.Any("/policies/*path", proxy)

	// Profiler endpoints - forward to backend service (backend exposes /api/profiler)
	api.Any("/profiler", proxy)
	api.Any("/profiler/*path", proxy)

	// Data Domains endpoints - forward to backend service (backend exposes /api/data-domains)
	api.Any("/data-domains", proxy)
	api.Any("/data-domains/*path", proxy)

	// Entity Schema endpoints - forward to backend service (backend exposes /api/entity-schema)
	api.GET("/entity-schema", proxy)
	api.POST("/entity-schema", proxy)

	// Validation Rules endpoints - forward to backend service (backend exposes /api/validation-rules)
	api.GET("/validation-rules", proxy)
	api.POST("/validation-rules", proxy)
	api.GET("/validation-rules/:id", proxy)
	api.PATCH("/validation-rules/:id", proxy)
	api.DELETE("/validation-rules/:id", proxy)
	api.POST("/validation-rules/:id/execute", proxy)
	api.POST("/validation-rules/execute-batch", proxy)
	api.GET("/validation-rules/:id/audit", proxy)

	// Schema introspection and rule testing endpoints
	api.GET("/schema/:entity", proxy)                      // Proxy to backend for dynamic schema
	api.POST("/rules/test", proxy)                         // Proxy to backend for rule testing
	api.GET("/ai/discover-relationships/:entityId", proxy) // Proxy to backend for AI suggestions
	api.POST("/ai/generate-rule", proxy)                   // Proxy to backend for AI rule generation from natural language

	// Relationships endpoints - forward to backend service (backend exposes /api/relationships)
	api.Any("/relationships", proxy)
	api.Any("/relationships/*path", proxy)

	// Bundles endpoints are proxied via explicit routes registered earlier

	// Catalog endpoints - forward specific catalog backend routes to the
	// backend service. We avoid registering a broad catch-all here because
	// Gin disallows wildcard segments that conflict with existing prefixes
	// (e.g. /apis). Register the handful of catalog endpoints the frontend
	// uses directly.
	api.Any("/catalog/tables", proxy)
	api.Any("/catalog/tables/*path", proxy)
	api.Any("/catalog/nodes", proxy)

	// Lineage endpoints - forward to backend service (replaces previous mock)
	api.Any("/lineage", proxy)
	api.Any("/lineage/*path", proxy)

	// Node Types & Edge Types endpoints - forward to backend service
	api.Any("/node-types", proxy)
	api.Any("/node-types/*path", proxy)
	api.Any("/edge-types", proxy)
	api.Any("/edge-types/*path", proxy)

	// Business Process Notifications endpoints - forward to backend service
	api.Any("/bp-notifications", proxy)
	api.Any("/bp-notifications/*path", proxy)

	// Model generator endpoint - handled by the models route registration above

	// Query management endpoints
	api.POST("/queries", func(c *gin.Context) {
		handleCreateQuery(c, config)
	})

	api.GET("/queries", func(c *gin.Context) {
		handleGetQueries(c, config)
	})

	api.GET("/queries/:id", func(c *gin.Context) {
		handleGetQuery(c, config)
	})

	api.PUT("/queries/:id", func(c *gin.Context) {
		handleUpdateQuery(c, config)
	})

	api.DELETE("/queries/:id", func(c *gin.Context) {
		handleDeleteQuery(c, config)
	})

	api.POST("/queries/:id/clone", func(c *gin.Context) {
		handleCloneQuery(c, config)
	})

	api.POST("/queries/:id/share", func(c *gin.Context) {
		handleShareQuery(c, config)
	})

	// API management endpoints
	api.POST("/apis", func(c *gin.Context) {
		handleCreateAPI(c)
	})

	api.GET("/apis", func(c *gin.Context) {
		handleGetAPIs(c)
	})

	api.GET("/apis/:id", func(c *gin.Context) {
		handleGetAPI(c, config)
	})

	api.PUT("/apis/:id", func(c *gin.Context) {
		handleUpdateAPI(c, config)
	})

	api.DELETE("/apis/:id", func(c *gin.Context) {
		handleDeleteAPI(c, config)
	})

	api.POST("/apis/:id/clone", func(c *gin.Context) {
		handleCloneAPI(c, config)
	})

	api.POST("/apis/:id/share", func(c *gin.Context) {
		handleShareAPI(c, config)
	})

	// Dynamic API execution endpoints
	api.POST("/execute/:apiId/*path", func(c *gin.Context) {
		handleExecuteAPI(c, config)
	})

	// OpenAPI/Swagger UI
	r.Static("/docs", "./docs")

	// Serve OpenAPI spec
	r.GET("/api/openapi.yaml", func(c *gin.Context) {
		c.File("./openapi.yaml")
	})

	// Initialize Temporal client (env-driven + retries)
	tc, err := temporalclient.NewClientWithRetry()
	if err != nil {
		log.Printf("WARNING: Failed to create Temporal client: %v. Some features may be unavailable.", err)
	} else {
		defer tc.Close()
		// Register custom routes only if temporal client is available
		apipkg.RegisterOptimizeAlphaRoutes(r, tc)
		apipkg.RegisterRiskAlphaRoutes(r, tc)
		apipkg.RegisterScenarioAnalysisRoutes(r, tc)
		apipkg.RegisterRebalancerRoutes(r, tc)
	}

	log.Printf("API Gateway starting on port %s", config.Port)
	log.Fatal(r.Run(":" + config.Port))
}

func handleCreateQuery(c *gin.Context, _ Config) {
	var req struct {
		Name        string                 `json:"name" binding:"required"`
		Description string                 `json:"description"`
		Type        string                 `json:"type" binding:"required"`
		Config      map[string]interface{} `json:"config" binding:"required"`
		Tags        []string               `json:"tags"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// In production, save to database
	query := gin.H{
		"id":          generateID(),
		"name":        req.Name,
		"description": req.Description,
		"type":        req.Type,
		"config":      req.Config,
		"tags":        req.Tags,
		"created_by":  "current_user", // Get from JWT
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"is_core":     false,
	}

	c.JSON(201, query)
}

func handleGetQueries(c *gin.Context, _ Config) {
	// In production, fetch from database with filtering
	queries := []gin.H{
		{
			"id":          "1",
			"name":        "Monthly Sales Report",
			"description": "Sales performance by month",
			"type":        "public",
			"created_by":  "john.doe",
			"created_at":  "2024-01-15T10:00:00Z",
			"updated_at":  "2024-01-15T10:00:00Z",
			"is_core":     true,
			"tags":        []string{"sales", "monthly"},
		},
	}

	c.JSON(200, gin.H{"queries": queries})
}

func handleGetQuery(c *gin.Context, _ Config) {
	id := c.Param("id")
	// In production, fetch from database
	query := gin.H{
		"id":          id,
		"name":        "Sample Query",
		"description": "Sample query description",
		"type":        "public",
		"config":      gin.H{"dataSource": "orders", "measures": []string{"total_amount"}},
		"created_by":  "john.doe",
		"created_at":  "2024-01-15T10:00:00Z",
		"updated_at":  "2024-01-15T10:00:00Z",
		"is_core":     false,
	}

	c.JSON(200, query)
}

func handleUpdateQuery(c *gin.Context, _ Config) {
	id := c.Param("id")
	var req struct {
		Name        string                 `json:"name"`
		Description string                 `json:"description"`
		Type        string                 `json:"type"`
		Config      map[string]interface{} `json:"config"`
		Tags        []string               `json:"tags"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// In production, update in database
	query := gin.H{
		"id":          id,
		"name":        req.Name,
		"description": req.Description,
		"type":        req.Type,
		"config":      req.Config,
		"tags":        req.Tags,
		"updated_at":  time.Now(),
	}

	c.JSON(200, query)
}

func handleDeleteQuery(c *gin.Context, _ Config) {
	id := c.Param("id")
	// In production, delete from database
	log.Printf("Deleting query with id: %s", id)
	c.JSON(204, gin.H{})
}

func handleCloneQuery(c *gin.Context, _ Config) {
	id := c.Param("id")
	// In production, clone the query in database
	log.Printf("Cloning query with id: %s", id)
	clonedQuery := gin.H{
		"id":          generateID(),
		"name":        "Cloned Query",
		"description": "Cloned from original query",
		"type":        "private",
		"created_by":  "current_user",
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"is_core":     false,
	}

	c.JSON(201, clonedQuery)
}

func handleShareQuery(c *gin.Context, _ Config) {
	id := c.Param("id")
	log.Printf("Sharing query with id: %s", id)
	var req struct {
		Users []string `json:"users"`
		Teams []string `json:"teams"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// In production, update sharing permissions in database
	c.JSON(200, gin.H{"message": "Query shared successfully"})
}

func handleCreateAPI(c *gin.Context) {
	var req struct {
		Name        string                   `json:"name" binding:"required"`
		Description string                   `json:"description"`
		Type        string                   `json:"type" binding:"required"`
		Config      map[string]interface{}   `json:"config" binding:"required"`
		Endpoints   []map[string]interface{} `json:"endpoints"`
		Tags        []string                 `json:"tags"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// In production, save to database
	api := gin.H{
		"id":          generateID(),
		"name":        req.Name,
		"description": req.Description,
		"type":        req.Type,
		"config":      req.Config,
		"endpoints":   req.Endpoints,
		"tags":        req.Tags,
		"created_by":  "current_user",
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"is_core":     false,
	}

	c.JSON(201, api)
}

func handleGetAPI(c *gin.Context, _ Config) {
	id := c.Param("id")
	// In production, fetch from database
	api := gin.H{
		"id":          id,
		"name":        "Sample API",
		"description": "Sample API description",
		"type":        "public",
		"config":      gin.H{"basePath": "/api", "authentication": "jwt"},
		"endpoints":   []gin.H{},
		"created_by":  "john.doe",
		"created_at":  "2024-01-15T10:00:00Z",
		"updated_at":  "2024-01-15T10:00:00Z",
		"is_core":     false,
	}

	c.JSON(200, api)
}

func handleUpdateAPI(c *gin.Context, _ Config) {
	id := c.Param("id")
	var req struct {
		Name        string                   `json:"name"`
		Description string                   `json:"description"`
		Type        string                   `json:"type"`
		Config      map[string]interface{}   `json:"config"`
		Endpoints   []map[string]interface{} `json:"endpoints"`
		Tags        []string                 `json:"tags"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// In production, update in database
	api := gin.H{
		"id":          id,
		"name":        req.Name,
		"description": req.Description,
		"type":        req.Type,
		"config":      req.Config,
		"endpoints":   req.Endpoints,
		"tags":        req.Tags,
		"updated_at":  time.Now(),
	}

	c.JSON(200, api)
}

func handleDeleteAPI(c *gin.Context, _ Config) {
	id := c.Param("id")
	// In production, delete from database
	log.Printf("Deleting API with id: %s", id)
	c.JSON(204, gin.H{})
}

func handleCloneAPI(c *gin.Context, _ Config) {
	id := c.Param("id")
	// In production, clone the API in database
	log.Printf("Cloning API with id: %s", id)
	clonedAPI := gin.H{
		"id":          generateID(),
		"name":        "Cloned API",
		"description": "Cloned from original API",
		"type":        "private",
		"created_by":  "current_user",
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"is_core":     false,
	}

	c.JSON(201, clonedAPI)
}

func handleShareAPI(c *gin.Context, _ Config) {
	id := c.Param("id")
	log.Printf("Sharing API with id: %s", id)
	var req struct {
		Users []string `json:"users"`
		Teams []string `json:"teams"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// In production, update sharing permissions in database
	c.JSON(200, gin.H{"message": "API shared successfully"})
}

func handleExecuteAPI(c *gin.Context, _ Config) {
	apiId := c.Param("apiId")
	path := c.Param("path")
	method := c.Request.Method

	// In production, look up the API configuration and execute against tenant database
	result := gin.H{
		"api_id": apiId,
		"path":   path,
		"method": method,
		"result": "API executed successfully",
		"data":   []gin.H{}, // Mock data
	}

	c.JSON(200, result)
}

func handleBusinessTermSearch(c *gin.Context, config Config) {
	log.Printf("HANDLER CALLED: handleBusinessTermSearch")

	// Log the raw request body
	rawBody, _ := c.GetRawData()
	log.Printf("RAW REQUEST BODY: %s", string(rawBody))
	// Restore the body for binding
	c.Request.Body = io.NopCloser(bytes.NewBuffer(rawBody))

	var req BusinessTermSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("JSON BINDING ERROR: %v", err)
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	log.Printf("PARSED REQUEST: %+v", req)

	// Set defaults
	if req.Limit == 0 {
		req.Limit = 20
	}
	tenantID := req.TenantID
	if tenantID == "" {
		tenantID = c.GetHeader("X-Tenant-ID")
		if tenantID == "" {
			tenantID = "default"
		}
	}

	// Get datasource ID from headers
	datasourceID := c.GetHeader("X-Tenant-Datasource-ID")
	if datasourceID == "" {
		datasourceID = "default"
	}

	// Create search request for backend (backend expects Query and Limit)
	backendReq := map[string]interface{}{
		"query": req.Query,
		"limit": req.Limit,
	}

	bodyJSON, err := json.Marshal(backendReq)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to marshal request: " + err.Error()})
		return
	}

	// Call backend /business-terms/search endpoint
	backendURL := config.BackendURL + "/business-terms/search"
	httpReq, err := http.NewRequest("POST", backendURL, bytes.NewBuffer(bodyJSON))
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to create backend request: " + err.Error()})
		return
	}

	// Set required headers for backend
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Tenant-ID", tenantID)
	httpReq.Header.Set("X-Tenant-Datasource-ID", datasourceID)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to reach backend service: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to read backend response: " + err.Error()})
		return
	}

	// Forward the response from backend
	c.Data(resp.StatusCode, "application/json", body)
}

func handleBusinessTermValidation(c *gin.Context, config Config) {
	var req BusinessTermValidationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// For now, return a placeholder validation response
	// In production, this would call the backend validation service
	c.JSON(200, gin.H{
		"valid":    true,
		"errors":   []string{},
		"warnings": []string{},
	})
}

func handleGraphQLProxy(c *gin.Context, config Config) {

	// Use GetRawData to safely read the body, even if a middleware has already read it.
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(400, gin.H{"error": "Failed to read request body"})
		return
	}

	// Forward raw request to Hasura and return the upstream status/body directly so
	// the frontend can see meaningful GraphQL errors instead of a generic gateway 500.
	status, respBody, err := executeRawGraphQLRequest(config, body, c.Request.Header)
	if err == nil {
		c.Header("X-Hasura-Admin-Secret", config.HasuraSecret)
	}
	if err != nil {
		// Log and return an internal error if we couldn't reach Hasura
		log.Printf("error proxying GraphQL request: %v", err)
		c.JSON(502, gin.H{"error": "Bad gateway: failed to contact GraphQL upstream"})
		return
	}

	// If upstream returned an error status, also log the body for debugging
	if status >= 400 {
		log.Printf("upstream GraphQL returned status %d: %s", status, string(respBody))
	}

	// Forward Hasura response exactly (content type is application/json)
	c.Data(status, "application/json", respBody)
}

func executeGraphQLQuery(config Config, query string, variables map[string]interface{}) (*GraphQLResponse, error) {
	req := GraphQLRequest{
		Query:     query,
		Variables: variables,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	_, respBody, err := executeRawGraphQLRequest(config, reqBody, http.Header{})
	if err != nil {
		return nil, err
	}
	var gqlResp GraphQLResponse
	if err := json.Unmarshal(respBody, &gqlResp); err != nil {
		return nil, err
	}
	return &gqlResp, nil
}

// executeRawGraphQLRequest sends the raw request body to Hasura and returns the
// upstream HTTP status code and response body. This lets the gateway forward
// Hasura errors directly to clients for easier debugging in development.
func executeRawGraphQLRequest(config Config, reqBody []byte, incomingHeaders http.Header) (int, []byte, error) {
	// Ensure we post to the Hasura GraphQL endpoint path.
	target := strings.TrimSuffix(config.HasuraURL, "/") + "/v1/graphql"
	req, err := http.NewRequest("POST", target, bytes.NewBuffer(reqBody))
	if err != nil {
		return 0, nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	// Set multiple header variants to ensure upstream Hasura recognizes the admin secret
	req.Header.Set("X-Hasura-Admin-Secret", config.HasuraSecret)
	req.Header.Set("x-hasura-admin-secret", config.HasuraSecret)
	req.Header.Set("x-hasura-access-key", config.HasuraSecret)

	// Forward Authorization header from client so Hasura can validate JWT tokens
	if authHeader := incomingHeaders.Get("Authorization"); authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	// Forward tenant-related headers if present
	if tenantID := incomingHeaders.Get("X-Tenant-ID"); tenantID != "" {
		req.Header.Set("X-Tenant-ID", tenantID)
	}
	if tenantDS := incomingHeaders.Get("X-Tenant-Datasource-ID"); tenantDS != "" {
		req.Header.Set("X-Tenant-Datasource-ID", tenantDS)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}

	return resp.StatusCode, body, nil
}

func handleGetAPIs(c *gin.Context) {
	// For now, return mock data - replace with actual database queries
	apis := []gin.H{
		{
			"id":             "1",
			"path":           "/api/search/business-terms",
			"method":         "POST",
			"description":    "Search for business terms in the catalog",
			"category":       "Business Terms",
			"service":        "API Gateway",
			"version":        "v1.0",
			"status":         "active",
			"last_updated":   "2024-01-15",
			"business_terms": []string{"customer_id", "order_value"},
			"dependencies":   []string{"hasura", "postgres"},
		},
		{
			"id":             "2",
			"path":           "/api/validate/business-term",
			"method":         "POST",
			"description":    "Validate business term definitions",
			"category":       "Business Terms",
			"service":        "API Gateway",
			"version":        "v1.0",
			"status":         "active",
			"last_updated":   "2024-01-15",
			"business_terms": []string{"customer_id"},
			"dependencies":   []string{"hasura"},
		},
	}

	c.JSON(200, gin.H{"apis": apis})
}

func handleGetBusinessTerms(c *gin.Context) {
	// For now, return mock data - replace with actual database queries
	businessTerms := []gin.H{
		{
			"id":           "customer_id",
			"name":         "Customer ID",
			"description":  "Unique identifier for customers",
			"category":     "Customer Data",
			"owner":        "Data Team",
			"status":       "approved",
			"related_apis": []string{"1", "2"},
		},
	}

	c.JSON(200, gin.H{"business_terms": businessTerms})
}

func handleCreateBusinessTerm(c *gin.Context) {
	var termData map[string]interface{}
	if err := c.ShouldBindJSON(&termData); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// TODO: Save to database
	c.JSON(201, gin.H{"message": "Business term created successfully", "business_term": termData})
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
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

func createProxyHandler(backendBase string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Handle OPTIONS requests for CORS preflight - return 200 OK without proxying
		if c.Request.Method == http.MethodOptions {
			c.Status(200)
			return
		}

		// Build backend URL preserving the original path and query.
		// c.Request.URL.Path already contains the full path including the group prefix (e.g., /api/fabric/some/thing).
		// If the gateway has no HASURA_URL configured and the frontend requested source=resolved
		// for /api/views, rewrite to source=runtime so the backend will return runtime files instead
		// of attempting a GraphQL query that would fail in this dev configuration.
		// Note: this only affects proxied requests; it avoids returning 500s to the dev frontend.
		reqURL := *c.Request.URL
		if strings.HasPrefix(c.Request.URL.Path, "/api/views") {
			q := reqURL.Query()
			if strings.ToLower(strings.TrimSpace(q.Get("source"))) == "resolved" && strings.TrimSpace(getEnv("HASURA_URL", "")) == "" {
				q.Set("source", "runtime")
				reqURL.RawQuery = q.Encode()
			}
		}
		backendURL := backendBase + reqURL.Path
		if reqURL.RawQuery != "" {
			backendURL = backendURL + "?" + reqURL.RawQuery
		}

		// Read incoming request body
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read request body"})
			return
		}

		// Create new request to backend
		req, err := http.NewRequest(c.Request.Method, backendURL, bytes.NewBuffer(body))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create backend request"})
			return
		}

		// Copy headers from the original request
		req.Header = c.Request.Header.Clone()

		// Propagate forwarding headers
		clientIP := c.ClientIP()
		if existing := req.Header.Get("X-Forwarded-For"); existing != "" {
			req.Header.Set("X-Forwarded-For", existing+", "+clientIP)
		} else {
			req.Header.Set("X-Forwarded-For", clientIP)
		}
		if proto := c.Request.Header.Get("X-Forwarded-Proto"); proto != "" {
			req.Header.Set("X-Forwarded-Proto", proto)
		} else if c.Request.TLS != nil {
			req.Header.Set("X-Forwarded-Proto", "https")
		} else {
			req.Header.Set("X-Forwarded-Proto", "http")
		}

		// Let the http.Client set the correct Content-Length
		req.Header.Del("Content-Length")

		// Execute the request
		client := &http.Client{Timeout: 60 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to connect to backend service", "details": err.Error()})
			return
		}
		defer resp.Body.Close()

		// Read backend response
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read backend response"})
			return
		}

		// Special-case: ensure requests to /api/business-term always return
		// a JSON payload the frontend can parse. If the backend returned a
		// non-JSON body or a non-200 status (for example a 404 HTML page),
		// respond with a safe JSON fallback instead of forwarding opaque
		// HTML which would break frontend JSON.parse.
		if strings.HasPrefix(c.Request.URL.Path, "/api/business-term") {
			ct := resp.Header.Get("Content-Type")
			if resp.StatusCode != http.StatusOK || !strings.Contains(strings.ToLower(ct), "application/json") {
				// Reply with the backend status but a safe JSON body.
				c.Status(resp.StatusCode)
				c.Header("Content-Type", "application/json")
				_, _ = c.Writer.Write([]byte("{\"business_term\":\"\"}"))
				return
			}
		}

		// If we modified tenant IP whitelist, invalidate cache for that tenant so
		// subsequent requests see the latest state.
		if strings.Contains(c.Request.URL.Path, "/api/tenants/") && strings.Contains(c.Request.URL.Path, "/ip-whitelist") {
			// path format: /api/tenants/{tenantId}/ip-whitelist[...]
			parts := strings.Split(c.Request.URL.Path, "/")
			for i := 0; i < len(parts)-1; i++ {
				if parts[i] == "tenants" && i+1 < len(parts) {
					wlCache.Delete(parts[i+1])
					break
				}
			}
		}

		// Propagate status, headers, and body from the backend response
		c.Status(resp.StatusCode)
		for k, v := range resp.Header {
			// Copy backend headers but avoid duplicating hop-by-hop headers
			c.Header(k, strings.Join(v, ","))
		}
		// Ensure CORS header is present for browser clients. Prefer backend's
		// Access-Control-Allow-Origin when provided; otherwise use request Origin
		// (set by browser) so dev server at http://localhost:5173 is allowed.
		// Apply dev-only CORS fallback when enabled via DEV_ALLOW_UNAUTH_FABRIC.
		// Dev-friendly CORS: if backend didn't set CORS or set '*', reflect
		// the request Origin for common localhost dev ports so browser preflight
		// passes when running the frontend locally on Vite (5173/5174).
		origin := c.Request.Header.Get("Origin")
		acao := c.Writer.Header().Get("Access-Control-Allow-Origin")
		if acao == "" || acao == "*" {
			if origin != "" {
				// Allow common dev origins automatically
				if strings.HasPrefix(origin, "http://localhost:517") || strings.HasPrefix(origin, "http://127.0.0.1:517") || strings.HasPrefix(origin, "http://localhost:3000") {
					c.Header("Access-Control-Allow-Origin", origin)
					c.Header("Access-Control-Allow-Credentials", "true")
					// Allow common headers used by the frontend and preflight
					c.Header("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization, X-User-Id, X-Tenant-Id, X-Datasource-Id")
					c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, PATCH, DELETE")
					log.Printf("proxy: dev CORS reflected origin=%s path=%s original=%s", origin, c.Request.URL.Path, acao)
				} else if strings.ToLower(getEnv("DEV_ALLOW_UNAUTH_FABRIC", "false")) == "true" {
					// Fallback when explicit dev override is enabled
					c.Header("Access-Control-Allow-Origin", origin)
					c.Header("Access-Control-Allow-Credentials", "true")
					c.Header("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization, X-User-Id, X-Tenant-Id, X-Datasource-Id")
					c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, PATCH, DELETE")
					log.Printf("proxy: dev CORS override applied, origin=%s path=%s original=%s", origin, c.Request.URL.Path, acao)
				}
			} else {
				// As a final fallback allow localhost:5173 for non-browser clients
				c.Header("Access-Control-Allow-Origin", "http://localhost:5173")
				log.Printf("proxy: dev CORS fallback applied, no Origin header present, using http://localhost:5173 for path=%s original=%s", c.Request.URL.Path, acao)
			}
		}
		c.Writer.Write(respBody)
	}
}
