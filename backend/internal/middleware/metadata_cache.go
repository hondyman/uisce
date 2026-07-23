package middleware

import (
	"net/http"
	"strings"
)

// MetadataCacheConfig holds configuration for the metadata caching middleware
type MetadataCacheConfig struct {
	// MaxAge is the browser cache duration in seconds (default: 60)
	MaxAge int
	// StaleWhileRevalidate is the duration for serving stale content while revalidating (default: 600)
	StaleWhileRevalidate int
	// CDNMaxAge is the CDN/Edge cache duration in seconds (default: 3600)
	CDNMaxAge int
	// PathPrefixes are the URL path prefixes to apply caching to
	PathPrefixes []string
}

// DefaultMetadataCacheConfig returns sensible defaults for metadata caching
func DefaultMetadataCacheConfig() MetadataCacheConfig {
	return MetadataCacheConfig{
		MaxAge:               60,   // 1 minute browser cache
		StaleWhileRevalidate: 600,  // 10 minutes stale-while-revalidate
		CDNMaxAge:            3600, // 1 hour CDN cache
		PathPrefixes: []string{
			"/layouts",
			"/api/layouts",
			"/api/metadata",
			"/api/schemas",
			"/api/definitions",
		},
	}
}

// MetadataCacheMiddleware returns chi-compatible middleware that applies aggressive
// caching policies for metadata/schema endpoints. This implements the "stale-while-revalidate"
// pattern to ensure instant responses for layout definitions.
//
// The strategy:
//   - Browser caches for MaxAge seconds (default 60s)
//   - During StaleWhileRevalidate period, browser serves stale content while fetching fresh
//   - CDN/Edge caches for CDNMaxAge seconds (default 1 hour)
//   - Cookies are stripped to ensure CDN cacheability
//
// This is critical for metadata-driven architectures where layout definitions
// are fetched frequently but change rarely. Without this, the "N+1" problem
// manifests as hundreds of layout fetch calls per page load.
func MetadataCacheMiddleware(config MetadataCacheConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only apply caching to GET requests for metadata paths
			if r.Method == http.MethodGet && shouldCachePath(r.URL.Path, config.PathPrefixes) {
				applyCacheHeaders(w, config)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// shouldCachePath checks if the request path matches any of the configured prefixes
func shouldCachePath(path string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// applyCacheHeaders sets the appropriate cache headers for metadata responses
func applyCacheHeaders(w http.ResponseWriter, config MetadataCacheConfig) {
	// 1. Browser Instruction:
	// "Cache locally for MaxAge seconds. Between MaxAge and MaxAge+StaleWhileRevalidate,
	// use the stale version but fetch a new one in the background."
	// This ensures the user NEVER sees a network spinner for metadata.
	w.Header().Set("Cache-Control", formatCacheControl(config.MaxAge, config.StaleWhileRevalidate))

	// 2. CDN Instruction (Surrogate-Control):
	// "Keep this at the Edge for CDNMaxAge seconds."
	// The Edge protects the Origin DB from traffic spikes.
	// This header is understood by CDNs like Fastly, Cloudflare, Akamai.
	w.Header().Set("Surrogate-Control", formatSurrogateControl(config.CDNMaxAge))

	// 3. Safety: Remove Cookies
	// CDNs often bypass cache if cookies are present to prevent leaking user data.
	// Metadata is structural/public within a tenant context, so cookies are unnecessary
	// for the schema itself. The tenant scoping is handled via headers.
	w.Header().Del("Set-Cookie")

	// 4. Vary header - ensure cache is keyed by tenant
	// This allows different tenants to have different cached versions
	w.Header().Set("Vary", "X-Tenant-ID, X-Tenant-Datasource-ID, Accept-Encoding")
}

// formatCacheControl builds the Cache-Control header value
func formatCacheControl(maxAge, staleWhileRevalidate int) string {
	return "public, max-age=" + itoa(maxAge) + ", stale-while-revalidate=" + itoa(staleWhileRevalidate)
}

// formatSurrogateControl builds the Surrogate-Control header value for CDNs
func formatSurrogateControl(maxAge int) string {
	return "max-age=" + itoa(maxAge)
}

// itoa converts an int to string without importing strconv for this simple case
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [20]byte
	pos := len(b)
	neg := i < 0
	if neg {
		i = -i
	}
	for i > 0 {
		pos--
		b[pos] = byte(i%10) + '0'
		i /= 10
	}
	if neg {
		pos--
		b[pos] = '-'
	}
	return string(b[pos:])
}

// MetadataCacheMiddlewareDefault returns the middleware with default configuration
func MetadataCacheMiddlewareDefault() func(http.Handler) http.Handler {
	return MetadataCacheMiddleware(DefaultMetadataCacheConfig())
}
