package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// sendJSON sends a JSON response
func sendJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

// sendError sends an error response
func sendError(w http.ResponseWriter, code int, message string) {
	sendJSON(w, code, map[string]string{"error": message})
}

// SendErrorResponse sends a detailed error response
func SendErrorResponse(w http.ResponseWriter, code int, title, detail string) {
	sendJSON(w, code, map[string]interface{}{
		"error":  title,
		"detail": detail,
	})
}

// normalizeTenantID normalizes tenant ID format
func normalizeTenantID(tenantID string) string {
	tenantID = strings.TrimSpace(tenantID)
	tenantID = strings.ToLower(tenantID)

	// Try to parse as UUID - if it works, it's already normalized
	if _, err := uuid.Parse(tenantID); err == nil {
		return tenantID
	}

	// If not a UUID, treat as tenant slug and create deterministic UUID
	return uuid.NewSHA1(uuid.NameSpaceDNS, []byte(tenantID)).String()
}

// setupAuthContext sets up authentication context
func setupAuthContext(ctx context.Context, tenantID string) context.Context {
	ctx = context.WithValue(ctx, "tenant_id", tenantID)
	ctx = context.WithValue(ctx, "app.current_tenant_id", tenantID)
	return ctx
}

// extractTenantFromContext extracts tenant ID from context
func extractTenantFromContext(ctx context.Context) (uuid.UUID, error) {
	tenantStr, ok := ctx.Value("tenant_id").(string)
	if !ok {
		tenantStr, ok = ctx.Value("app.current_tenant_id").(string)
		if !ok {
			return uuid.Nil, NewError("missing tenant context")
		}
	}

	return uuid.Parse(tenantStr)
}

// Error represents a handler error
type Error struct {
	Message string
}

// NewError creates a new error
func NewError(message string) *Error {
	return &Error{Message: message}
}

// Error implements error interface
func (e *Error) Error() string {
	return e.Message
}
