package audit

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/requestcontext"
	"go.uber.org/zap"
)

// Middleware provides audit logging middleware for Gin
type Middleware struct {
	auditService *Service
	logger       *zap.Logger
}

// NewMiddleware creates a new audit middleware
func NewMiddleware(auditService *Service) *Middleware {
	return &Middleware{
		auditService: auditService,
		logger:       logging.GetLogger(),
	}
}

// responseWriter is a wrapper for http.ResponseWriter that captures the status code
type responseWriter struct {
	http.ResponseWriter
	status      int
	size        int
	wroteHeader bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}
	rw.status = code
	rw.wroteHeader = true
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	return n, err
}

// AuditLoggingMiddleware logs all HTTP requests for audit purposes
func (m *Middleware) AuditLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		// Get request context
		requestContext := requestcontext.GetRequestContext(r.Context())
		if requestContext == nil {
			next.ServeHTTP(w, r)
			return
		}

		// Read request body for sensitive operations
		var requestBody []byte
		if shouldLogRequestBody(r.Method, r.URL.Path) {
			if r.Body != nil {
				requestBody, _ = io.ReadAll(r.Body)
				// Restore the request body for further processing
				r.Body = io.NopCloser(bytes.NewBuffer(requestBody))
			}
		}

		// Wrap response writer to capture status code
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		// Process request
		next.ServeHTTP(rw, r)

		// Log the audit event
		duration := time.Since(startTime)

		event := &models.AuditEvent{
			EventType:    m.determineEventType(r, rw.status),
			Severity:     m.determineSeverity(r, rw.status),
			UserID:       requestContext.UserID,
			TenantID:     requestContext.TenantID,
			SessionID:    m.extractSessionID(r),
			ResourceID:   m.extractResourceID(r),
			ResourceType: m.extractResourceType(r),
			Action:       r.Method,
			IPAddress:    requestContext.IPAddress,
			UserAgent:    requestContext.UserAgent,
			RequestID:    requestContext.RequestID,
			Details: map[string]interface{}{
				"path":         r.URL.Path,
				"method":       r.Method,
				"status_code":  rw.status,
				"duration_ms":  duration.Milliseconds(),
				"user_agent":   requestContext.UserAgent,
				"query_params": r.URL.Query(),
			},
			Success: rw.status < 400,
		}

		// Add error message if request failed
		if !event.Success {
			event.ErrorMessage = m.extractErrorMessage(r, rw.status)
		}

		// Add request body for sensitive operations
		if len(requestBody) > 0 && len(requestBody) < 10000 { // Limit size
			event.Details["request_body"] = string(requestBody)
		}

		// Add compliance flags
		event.ComplianceFlags = m.determineComplianceFlags(r, event)

		// Log the event asynchronously to avoid blocking the response
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Convert models.AuditEvent to UnifiedAuditRecord
			uar := UnifiedAuditRecord{
				EventType:  string(event.EventType),
				TenantID:   event.TenantID,
				ActorID:    event.UserID,
				ObjectType: event.ResourceType,
				ObjectID:   event.ResourceID,
				Narrative:  event.Action + " " + event.ResourceType,
				Status:     map[bool]string{true: "Success", false: "Failed"}[event.Success],
				ErrorCode:  event.ErrorMessage,
			}

			if err := m.auditService.LogEvent(ctx, uar); err != nil {
				m.logger.Error("Failed to log audit event", zap.Error(err))
			}
		}()
	})
}

// determineEventType determines the audit event type based on the request
func (m *Middleware) determineEventType(r *http.Request, statusCode int) models.AuditEventType {
	path := r.URL.Path
	method := r.Method

	// Authentication events
	if strings.Contains(path, "/auth/login") {
		if statusCode >= 400 {
			return models.EventLoginFailed
		}
		return models.EventLogin
	}
	if strings.Contains(path, "/auth/logout") {
		return models.EventLogout
	}

	// Data access events
	if strings.Contains(path, "/api/funds") || strings.Contains(path, "/api/metrics") {
		switch method {
		case "GET":
			return models.EventDataAccess
		case "POST", "PUT", "PATCH":
			return models.EventDataModify
		case "DELETE":
			return models.EventDataDelete
		}
	}

	// Configuration events
	if strings.Contains(path, "/api/bundles") || strings.Contains(path, "/api/config") {
		switch method {
		case "POST":
			return models.EventBundleCreate
		case "PUT", "PATCH":
			return models.EventBundleUpdate
		case "DELETE":
			return models.EventBundleDelete
		}
	}

	// Calculation events
	if strings.Contains(path, "/api/calculate") || strings.Contains(path, "/api/models") {
		return models.EventCalculationRun
	}

	// Default to data access
	return models.EventDataAccess
}

// determineSeverity determines the severity level based on the request and response
func (m *Middleware) determineSeverity(r *http.Request, statusCode int) models.AuditEventSeverity {
	path := r.URL.Path

	// Critical for authentication failures
	if strings.Contains(path, "/auth/") && statusCode >= 400 {
		return models.SeverityHigh
	}

	// High for unauthorized access
	if statusCode == 403 || statusCode == 401 {
		return models.SeverityHigh
	}

	// Medium for errors
	if statusCode >= 400 {
		return models.SeverityMedium
	}

	// Low for successful operations
	return models.SeverityLow
}

// shouldLogRequestBody determines if the request body should be logged
func shouldLogRequestBody(method, path string) bool {
	// Log request bodies for write operations on sensitive endpoints
	if method == "POST" || method == "PUT" || method == "PATCH" {
		sensitivePaths := []string{
			"/api/auth",
			"/api/users",
			"/api/bundles",
			"/api/config",
		}

		for _, sensitivePath := range sensitivePaths {
			if strings.Contains(path, sensitivePath) {
				return true
			}
		}
	}

	return false
}

// extractSessionID extracts session ID from request
func (m *Middleware) extractSessionID(r *http.Request) string {
	// Try to get from header
	if sessionID := r.Header.Get("X-Session-ID"); sessionID != "" {
		return sessionID
	}

	// Try to get from cookie
	if cookie, err := r.Cookie("session_id"); err == nil {
		return cookie.Value
	}

	return ""
}

// extractResourceID extracts resource ID from request path
func (m *Middleware) extractResourceID(r *http.Request) string {
	path := r.URL.Path

	// Extract ID from common patterns
	if strings.Contains(path, "/api/funds/") {
		parts := strings.Split(path, "/")
		if len(parts) >= 4 {
			return parts[3]
		}
	}

	if strings.Contains(path, "/api/bundles/") {
		parts := strings.Split(path, "/")
		if len(parts) >= 4 {
			return parts[3]
		}
	}

	if strings.Contains(path, "/api/users/") {
		parts := strings.Split(path, "/")
		if len(parts) >= 4 {
			return parts[3]
		}
	}

	return ""
}

// extractResourceType extracts resource type from request path
func (m *Middleware) extractResourceType(r *http.Request) string {
	path := r.URL.Path

	if strings.Contains(path, "/api/funds") {
		return "fund"
	}
	if strings.Contains(path, "/api/bundles") {
		return "bundle"
	}
	if strings.Contains(path, "/api/users") {
		return "user"
	}
	if strings.Contains(path, "/api/metrics") {
		return "metric"
	}
	if strings.Contains(path, "/api/calculate") {
		return "calculation"
	}

	return "unknown"
}

// extractErrorMessage extracts error message from response
func (m *Middleware) extractErrorMessage(r *http.Request, statusCode int) string {
	// Try to get error from context
	if err := r.Context().Value("error"); err != nil {
		if errStr, ok := err.(string); ok {
			return errStr
		}
	}

	// Return generic error based on status code
	switch statusCode {
	case 400:
		return "Bad Request"
	case 401:
		return "Unauthorized"
	case 403:
		return "Forbidden"
	case 404:
		return "Not Found"
	case 500:
		return "Internal Server Error"
	default:
		return "Request failed"
	}
}

// determineComplianceFlags determines compliance flags for the event
func (m *Middleware) determineComplianceFlags(r *http.Request, event *models.AuditEvent) []string {
	var flags []string

	// Add flags based on event characteristics
	if event.Severity == models.SeverityHigh || event.Severity == models.SeverityCritical {
		flags = append(flags, "requires_review")
	}

	if !event.Success {
		flags = append(flags, "failed_operation")
	}

	if strings.Contains(r.URL.Path, "/api/auth") {
		flags = append(flags, "authentication")
	}

	if strings.Contains(r.URL.Path, "/api/funds") || strings.Contains(r.URL.Path, "/api/metrics") {
		flags = append(flags, "data_access")
	}

	if event.IPAddress != "" {
		flags = append(flags, "external_access")
	}

	return flags
}
