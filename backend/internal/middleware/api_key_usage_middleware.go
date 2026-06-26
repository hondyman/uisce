package middleware

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/hondyman/semlayer/backend/internal/store"
)

// APIKeyUsageMiddleware logs API key usage to the audit trail
// This should be placed after authentication middleware
func APIKeyUsageMiddleware(usageStore store.APIKeyUsageStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authInfo, ok := security.AuthInfoFromContext(r.Context())
			if !ok {
				// No auth, skip logging
				next.ServeHTTP(w, r)
				return
			}

			// Extract API key ID if present
			apiKeyIDStr := r.Header.Get("X-API-Key-ID")
			if apiKeyIDStr == "" {
				// API key tracking not available, skip
				next.ServeHTTP(w, r)
				return
			}

			apiKeyID, err := uuid.Parse(apiKeyIDStr)
			if err != nil {
				// Invalid API key ID, skip
				next.ServeHTTP(w, r)
				return
			}

			// Extract client IP
			clientIP := extractClientIP(r)

			// Get user agent
			userAgent := r.UserAgent()

			// Extract region if present
			var region *string
			if regionStr := r.Header.Get("X-Region"); regionStr != "" {
				region = &regionStr
			}

			// Prepare usage record
			userID := uuid.Nil
			if authInfo.UserID != "" {
				if parsed, err := uuid.Parse(authInfo.UserID); err == nil {
					userID = parsed
				}
			}

			var tenantID *uuid.UUID
			if len(authInfo.TenantIDs) > 0 {
				if parsed, err := uuid.Parse(authInfo.TenantIDs[0]); err == nil {
					tenantID = &parsed
				}
			}

			// Log usage in background to not block request
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				var ip *net.IP
				if clientIP != "" {
					parsedIP := net.ParseIP(clientIP)
					ip = &parsedIP
				}

				var ua *string
				if userAgent != "" {
					ua = &userAgent
				}

				req := models.APIKeyUsageCreateRequest{
					APIKeyID:  apiKeyID,
					UserID:    userIDPtr(userID),
					TenantID:  tenantID,
					Path:      r.URL.Path,
					Method:    r.Method,
					Region:    region,
					IPAddress: ip,
					UserAgent: ua,
				}

				_ = usageStore.LogUsage(ctx, req)
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// extractClientIP extracts the client IP address from the request
func extractClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (behind reverse proxy)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}

	return r.RemoteAddr
}

// userIDPtr converts a UUID to a pointer if not nil
func userIDPtr(uid uuid.UUID) *uuid.UUID {
	if uid == uuid.Nil {
		return nil
	}
	return &uid
}
