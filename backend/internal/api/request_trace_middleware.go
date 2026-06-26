package api

import (
	"fmt"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// responseCapture wraps http.ResponseWriter to capture status and response headers
type responseCapture struct {
	http.ResponseWriter
	status int
}

func (rc *responseCapture) WriteHeader(status int) {
	rc.status = status
	rc.ResponseWriter.WriteHeader(status)
}

// RequestTracingMiddleware logs method, path, tenant header, request id, status, and X-BO-Handler header
func RequestTracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simple proof-of-execution log
		fmt.Fprintf(os.Stderr, "[MW-EXEC] Middleware executing for %s %s\n", r.Method, r.URL.Path)

		// Ensure request id
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.NewString()
			r.Header.Set("X-Request-ID", reqID)
			w.Header().Set("X-Request-ID", reqID)
		}

		tenant := jwtmiddleware.GetClaimsFromContext(r).TenantID

		// Log entry (structured and to stderr for immediate visibility)
		logging.GetLogger().Sugar().Infof("RequestTrace Start: id=%s method=%s path=%s tenant=%s", reqID, r.Method, r.URL.Path, tenant)
		fmt.Fprintf(os.Stderr, "[TRACE START] id=%s method=%s path=%s tenant=%s\n", reqID, r.Method, r.URL.Path, tenant)

		rc := &responseCapture{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rc, r)

		handler := rc.Header().Get("X-BO-Handler")
		logging.GetLogger().Sugar().Infof("RequestTrace End: id=%s method=%s path=%s tenant=%s status=%d handler=%s", reqID, r.Method, r.URL.Path, tenant, rc.status, handler)
		fmt.Fprintf(os.Stderr, "[TRACE END] id=%s method=%s path=%s tenant=%s status=%d handler=%s\n", reqID, r.Method, r.URL.Path, tenant, rc.status, handler)
	})
}
