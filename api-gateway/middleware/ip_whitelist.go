package middleware

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type WhitelistCache struct {
	mu    sync.RWMutex
	items map[string]wlEntry
}

type wlEntry struct {
	patterns []string
	expires  time.Time
}

func NewWhitelistCache() *WhitelistCache {
	return &WhitelistCache{items: make(map[string]wlEntry)}
}

func (c *WhitelistCache) Get(key string) ([]string, bool) {
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

func (c *WhitelistCache) Delete(key string) {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
}

func (c *WhitelistCache) Clear() {
	c.mu.Lock()
	c.items = make(map[string]wlEntry)
	c.mu.Unlock()
}

func ipMatches(pattern, ip string) bool {
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

type IPWhitelistConfig struct {
	Enforce     bool
	LogDecisions bool
	CacheTTL    time.Duration
}

func IPWhitelistMiddleware(backendBase string, cache *WhitelistCache, apiKeys map[string]APIKey, cfg IPWhitelistConfig) gin.HandlerFunc {
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
		if !cfg.Enforce {
			if cfg.LogDecisions {
				log.Printf("[IP WHITELIST] Path: %s, ENFORCE DISABLED", c.Request.URL.Path)
			}
			c.Next()
			return
		}
		if cfg.LogDecisions {
			log.Printf("[IP WHITELIST] Path: %s, ENFORCING", c.Request.URL.Path)
		}
		path := c.Request.URL.Path
		if skipPaths(path) {
			c.Next()
			return
		}

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
			c.Next()
			return
		}

		patterns, ok := cache.Get(tenantID)
		if !ok {
			client := &http.Client{Timeout: 5 * time.Second}
			req, _ := http.NewRequest("GET", backendBase+"/api/tenants/"+tenantID+"/ip-whitelist", nil)
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

			patterns = plist
			cache.Set(tenantID, patterns, cfg.CacheTTL)
		}

		clientIP := c.ClientIP()
		allowed := false
		if len(patterns) == 0 {
			allowed = true
		} else {
			for _, pattern := range patterns {
				if ipMatches(pattern, clientIP) {
					allowed = true
					break
				}
			}
		}

		if !allowed {
			if cfg.LogDecisions {
				log.Printf("ip_whitelist deny tenant=%s ip=%s patterns=%d path=%s", tenantID, clientIP, len(patterns), path)
			}
			c.JSON(403, gin.H{"error": "forbidden_ip", "message": "Client IP is not allowed"})
			c.Abort()
			return
		}

		if cfg.LogDecisions {
			log.Printf("ip_whitelist allow tenant=%s ip=%s patterns=%d path=%s", tenantID, clientIP, len(patterns), path)
		}

		c.Next()
	}
}
