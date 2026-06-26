package middleware

import (
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/hondyman/semlayer/backend/internal/requestcontext"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// RequestContextMiddleware adds request context to all requests
func RequestContextMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Generate unique request ID
			requestID := requestcontext.GenerateRequestID()

			// Extract user info from headers (would come from auth middleware)
			userID := r.Header.Get("X-User-ID")
			claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID

			// Extract IP
			ipAddress, _, _ := net.SplitHostPort(r.RemoteAddr)
			if ipAddress == "" {
				ipAddress = r.RemoteAddr
			}

			ctx := &requestcontext.RequestContext{
				RequestID: requestID,
				UserID:    userID,
				TenantID:  tenantID,
				IPAddress: ipAddress,
				UserAgent: r.Header.Get("User-Agent"),
				StartTime: time.Now(),
			}

			// Add request ID to response header
			w.Header().Set("X-Request-ID", requestID)

			// Add to request context for downstream use (typed key)
			reqCtx := requestcontext.WithRequestContext(r.Context(), ctx)
			next.ServeHTTP(w, r.WithContext(reqCtx))
		})
	}
}

// SecurityHeadersMiddleware adds security headers to responses
func SecurityHeadersMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			w.Header().Set("Content-Security-Policy", "default-src 'self'")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			next.ServeHTTP(w, r)
		})
	}
}

// CORSMiddleware configures CORS for cross-origin requests
func CORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			allowed := false
			for _, allowedOrigin := range allowedOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					allowed = true
					break
				}
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Request-ID, X-Tenant-Datasource-ID, x-tenant-datasource-id")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// HealthCheckMiddleware provides health check endpoint
func HealthCheckMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"status":    "healthy",
					"timestamp": time.Now().UTC(),
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ResponseCaptureWriter is a simple wrapper to capture status code
type ResponseCaptureWriter struct {
	http.ResponseWriter
	Status int
}

func (w *ResponseCaptureWriter) WriteHeader(code int) {
	w.Status = code
	w.ResponseWriter.WriteHeader(code)
}

// RequestLoggingMiddleware logs all requests with structured logging
func RequestLoggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			path := r.URL.Path
			raw := r.URL.RawQuery

			rcw := &ResponseCaptureWriter{ResponseWriter: w, Status: http.StatusOK}
			next.ServeHTTP(rcw, r)

			end := time.Now()
			latency := end.Sub(start)

			if raw != "" {
				path = path + "?" + raw
			}

			ctx := requestcontext.GetRequestContextFromRequest(r)
			requestID := ""
			if ctx != nil {
				requestID = ctx.RequestID
			}

			ipAddress, _, _ := net.SplitHostPort(r.RemoteAddr)
			if ipAddress == "" {
				ipAddress = r.RemoteAddr
			}

			logger.Info("request completed",
				"request_id", requestID,
				"method", r.Method,
				"path", path,
				"status", rcw.Status,
				"latency", latency,
				"ip", ipAddress,
				"user_agent", r.UserAgent(),
			)
		})
	}
}
