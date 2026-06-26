package middleware

import (
	"net/http"
)

// SecurityHeaders adds common security headers to all responses
// Provides defense-in-depth against XSS, clickjacking, and MIME type sniffing
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent MIME type sniffing (X-Content-Type-Options)
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking (X-Frame-Options)
		w.Header().Set("X-Frame-Options", "DENY")

		// Enable XSS filter (X-XSS-Protection - deprecated in modern browsers but still useful)
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Content Security Policy - restrictive by default
		// Adjust for your needs: 'unsafe-inline' should only be used in development
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'; connect-src 'self'")

		// Referrer-Policy - prevent referrer leakage
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions-Policy (formerly Feature-Policy) - restrict dangerous APIs
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=(), payment=(), usb=(), magnetometer=(), gyroscope=(), accelerometer=()")

		// HSTS (HTTP Strict-Transport-Security) - only in production with HTTPS
		if r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}

		// Timing-Allow-Origin - prevent timing attacks
		w.Header().Set("Timing-Allow-Origin", "none")

		// X-Permitted-Cross-Domain-Policies - Flash/PDF policy
		w.Header().Set("X-Permitted-Cross-Domain-Policies", "none")

		next.ServeHTTP(w, r)
	})
}

// SecurityHeadersStrict is a more aggressive security header set for APIs
// Use this for backend APIs where you don't need to support embedding or external content
func SecurityHeadersStrict(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// All the standard headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Stricter CSP for APIs - only self
		w.Header().Set("Content-Security-Policy", "default-src 'none'; form-action 'none'")

		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=(), payment=(), usb=(), magnetometer=(), gyroscope=(), accelerometer=()")

		if r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
		}

		w.Header().Set("Timing-Allow-Origin", "none")
		w.Header().Set("X-Permitted-Cross-Domain-Policies", "none")

		// Remove Server header disclosure
		w.Header().Del("Server")

		next.ServeHTTP(w, r)
	})
}
