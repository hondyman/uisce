package middleware

import (
	"net/http"
	"strings"
)

// MetadataCacheMiddleware applies aggressive caching policies for metadata endpoints.
// This enables CDN/edge caching and browser caching with stale-while-revalidate strategy.
//
// Key behaviors:
//   - Browser: Cache for 60s, serve stale for 600s while revalidating
//   - CDN: Cache for 1 hour at the edge
//   - Removes Set-Cookie to enable CDN caching
//   - Adds surrogate keys for targeted cache invalidation
func MetadataCacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only apply to GET requests (metadata reads)
		if r.Method != http.MethodGet {
			next.ServeHTTP(w, r)
			return
		}

		// 1. Browser Cache Instruction
		// "Cache locally for 60s. Between 60s and 600s (10 min), use the stale version
		// but fetch a new one in the background."
		// This ensures users NEVER see a network spinner for metadata.
		w.Header().Set("Cache-Control", "public, max-age=60, stale-while-revalidate=600")

		// 2. CDN/Edge Cache Instruction (Surrogate-Control)
		// "Keep this at the Edge for 1 hour (3600s)."
		// The Edge protects the Origin from traffic spikes.
		// Note: Fastly, Cloudflare, and similar CDNs respect this header.
		w.Header().Set("Surrogate-Control", "max-age=3600")

		// 3. Surrogate Key Tagging
		// Extract the resource type from the path for targeted invalidation.
		// Examples:
		//   /api/glossary/terms/schema -> key: metadata-glossary
		//   /api/semantic/models/123 -> key: metadata-semantic
		surrogateKey := extractSurrogateKey(r.URL.Path)
		if surrogateKey != "" {
			w.Header().Set("Surrogate-Key", surrogateKey)
		}

		// 4. Safety: Remove Cookies
		// CDNs often bypass cache if Set-Cookie is present to prevent leaking user data.
		// Metadata is public/structural, so cookies are unnecessary.
		// We'll remove Set-Cookie after the handler runs.
		defer func() {
			w.Header().Del("Set-Cookie")
		}()

		// 5. Vary Header Management
		// Ensure we don't vary on User-Agent or other high-cardinality headers.
		// This would create separate cache entries per user/browser, destroying efficiency.
		// Only vary on Authorization if metadata is tenant-specific.
		if w.Header().Get("Vary") == "" {
			w.Header().Set("Vary", "Accept-Encoding") // Only vary on compression
		}

		next.ServeHTTP(w, r)
	})
}

// extractSurrogateKey derives a cache invalidation key from the URL path.
// This allows targeted purging: when a glossary term is updated, we can purge
// only "metadata-glossary" without invalidating semantic schemas.
func extractSurrogateKey(path string) string {
	path = strings.TrimPrefix(path, "/api/")

	switch {
	case strings.HasPrefix(path, "glossary"):
		return "metadata-glossary"
	case strings.HasPrefix(path, "semantic"):
		return "metadata-semantic"
	case strings.HasPrefix(path, "layouts"):
		return "metadata-layouts"
	case strings.HasPrefix(path, "rules") && strings.Contains(path, "schema"):
		return "metadata-rules"
	default:
		return "metadata-generic"
	}
}

// CachePurgeHandler provides an API endpoint for cache invalidation.
// This is called when an admin updates metadata schemas.
//
// Example: POST /api/cache/purge {"surrogate_key": "metadata-glossary"}
func CachePurgeHandler(w http.ResponseWriter, r *http.Request) {
	// This is a placeholder. In production, this would call your CDN's purge API:
	// - Fastly: DELETE https://api.fastly.com/service/{service_id}/purge/{surrogate_key}
	// - Cloudflare: POST https://api.cloudflare.com/client/v4/zones/{zone}/purge_cache

	// var req struct {
	// 	SurrogateKey string `json:"surrogate_key"`
	// }

	// Parse request (simplified)
	// decoder := json.NewDecoder(r.Body)
	// if err := decoder.Decode(&req); err != nil {
	// 	http.Error(w, "Invalid request", http.StatusBadRequest)
	// 	return
	// }

	// TODO: Call CDN API
	// err := purgeCDNCache(req.SurrogateKey)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok","message":"Cache purge initiated"}`))
}
