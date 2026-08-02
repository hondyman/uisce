package proxy

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type ProxyHandler struct {
	backendBase string
	hasuraURL   string
	whitelistCache interface {
		Delete(key string)
	}
	devBypass bool
}

func NewProxyHandler(backendBase string) *ProxyHandler {
	return &ProxyHandler{
		backendBase: backendBase,
		devBypass:  true,
	}
}

func (h *ProxyHandler) WithHasuraURL(url string) *ProxyHandler {
	h.hasuraURL = url
	return h
}

func (h *ProxyHandler) WithWhitelistCache(cache interface {
	Delete(key string)
}) *ProxyHandler {
	h.whitelistCache = cache
	return h
}

func (h *ProxyHandler) WithDevBypass(enabled bool) *ProxyHandler {
	h.devBypass = enabled
	return h
}

func (h *ProxyHandler) ServeHTTP() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodOptions {
			c.Status(200)
			return
		}

		reqURL := *c.Request.URL
		if strings.HasPrefix(c.Request.URL.Path, "/api/views") {
			q := reqURL.Query()
			if strings.ToLower(strings.TrimSpace(q.Get("source"))) == "resolved" && strings.TrimSpace(h.hasuraURL) == "" {
				q.Set("source", "runtime")
				reqURL.RawQuery = q.Encode()
			}
		}

		backendURL := h.backendBase + reqURL.Path
		if reqURL.RawQuery != "" {
			backendURL = backendURL + "?" + reqURL.RawQuery
		}

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read request body"})
			return
		}

		req, err := http.NewRequest(c.Request.Method, backendURL, bytes.NewBuffer(body))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create backend request"})
			return
		}

		req.Header = c.Request.Header.Clone()

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

		req.Header.Del("Content-Length")

		client := &http.Client{Timeout: 60 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to connect to backend service", "details": err.Error()})
			return
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read backend response"})
			return
		}

		if strings.HasPrefix(c.Request.URL.Path, "/api/business-term") {
			ct := resp.Header.Get("Content-Type")
			if resp.StatusCode != http.StatusOK || !strings.Contains(strings.ToLower(ct), "application/json") {
				c.Status(resp.StatusCode)
				c.Header("Content-Type", "application/json")
				_, _ = c.Writer.Write([]byte("{\"business_term\":\"\"}"))
				return
			}
		}

		if h.whitelistCache != nil && strings.Contains(c.Request.URL.Path, "/api/tenants/") && strings.Contains(c.Request.URL.Path, "/ip-whitelist") {
			parts := strings.Split(c.Request.URL.Path, "/")
			for i := 0; i < len(parts)-1; i++ {
				if parts[i] == "tenants" && i+1 < len(parts) {
					h.whitelistCache.Delete(parts[i+1])
					break
				}
			}
		}

		c.Status(resp.StatusCode)
		for k, v := range resp.Header {
			c.Header(k, strings.Join(v, ","))
		}

		origin := c.Request.Header.Get("Origin")
		acao := c.Writer.Header().Get("Access-Control-Allow-Origin")
		if acao == "" || acao == "*" {
			if origin != "" {
				if strings.HasPrefix(origin, "http://localhost:517") || strings.HasPrefix(origin, "http://127.0.0.1:517") || strings.HasPrefix(origin, "http://localhost:3000") {
					c.Header("Access-Control-Allow-Origin", origin)
					c.Header("Access-Control-Allow-Credentials", "true")
					c.Header("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization, X-User-Id, X-Tenant-Id, X-Datasource-Id")
					c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, PATCH, DELETE")
					log.Printf("proxy: dev CORS reflected origin=%s path=%s original=%s", origin, c.Request.URL.Path, acao)
				} else if h.devBypass {
					c.Header("Access-Control-Allow-Origin", origin)
					c.Header("Access-Control-Allow-Credentials", "true")
					c.Header("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization, X-User-Id, X-Tenant-Id, X-Datasource-Id")
					c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, PATCH, DELETE")
					log.Printf("proxy: dev CORS override applied, origin=%s path=%s original=%s", origin, c.Request.URL.Path, acao)
				}
			} else {
				c.Header("Access-Control-Allow-Origin", "http://localhost:5173")
				log.Printf("proxy: dev CORS fallback applied, no Origin header present, using http://localhost:5173 for path=%s original=%s", c.Request.URL.Path, acao)
			}
		}

		c.Writer.Write(respBody)
	}
}
