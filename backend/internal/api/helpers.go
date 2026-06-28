package api

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// TenantContext represents extracted tenant context
type TenantContext struct {
	TenantID     string
	DatasourceID string
}

// extractTenantContext extracts tenant context from request headers and query params
func extractTenantContext(r *http.Request) (*TenantContext, error) {
	var tenantID string
	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil {
		tenantID = claims.TenantID
	}
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	// Fall back to query params
	if tenantID == "" {
		tenantID = r.URL.Query().Get("tenant_id")
	}
	if datasourceID == "" {
		datasourceID = r.URL.Query().Get("datasource_id")
	}

	if tenantID == "" || datasourceID == "" {
		return nil, fmt.Errorf("tenant context not found in headers or query params")
	}

	return &TenantContext{
		TenantID:     tenantID,
		DatasourceID: datasourceID,
	}, nil
}

// writeJSONError writes a structured JSON error response with the given status code.
func writeJSONError(w http.ResponseWriter, status int, msg string, errorCode string, details interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:     msg,
		Code:      status,
		ErrorCode: errorCode,
		Details:   details,
	})
}

// getEnv returns the environment variable value if set; otherwise returns defaultValue.
func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// respond is a small helper used across handlers to write JSON responses.
// It accepts a value (data) and an error; if error is non-nil it writes a
// structured JSON error response, otherwise it serializes the data as JSON.
func respond(w http.ResponseWriter, _r *http.Request, data interface{}, err error) {
	if err != nil {
		// If the error is an httpError with status, we could extract it —
		// keep it simple here and return 500 for now.
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "internal_error", nil)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if data == nil {
		// Write an empty JSON object for nil data
		json.NewEncoder(w).Encode(map[string]interface{}{})
		return
	}

	_ = json.NewEncoder(w).Encode(data)
}

// toTitleCase converts snake/camel/underscore names into a human-friendly title.
func toTitleCase(s string) string {
	if s == "" {
		return s
	}
	// Replace underscores/dashes with spaces
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "-", " ")
	parts := strings.Fields(s)
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
		}
	}
	return strings.Join(parts, " ")
}

// sanitizeViewPayload is a permissive passthrough for view payloads. The
// real implementation may trim or filter fields for client-safe responses;
// for now return the payload unchanged.
func sanitizeViewPayload(v interface{}) interface{} {
	return v
}

// fileETag returns a simple ETag string for file bytes and FileInfo. Use
// the file's modification time and size with a tiny fingerprint of the
// payload to detect changes.
func fileETag(b []byte, fi os.FileInfo) string {
	if fi == nil {
		return ""
	}
	h := ""
	if len(b) > 0 {
		// Use a short hex prefix of the content for a lightweight fingerprint
		prefix := 8
		if len(b) < prefix {
			prefix = len(b)
		}
		h = hex.EncodeToString(b[:prefix])
	}
	return fmt.Sprintf("%d-%d-%s", fi.ModTime().Unix(), fi.Size(), h)
}

// parseIntDefault parses a string into int, returning defaultVal on error.
func parseIntDefault(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return defaultVal
}

// errorsIs is a small wrapper around errors.Is for call sites that expect this helper
func errorsIs(err, target error) bool {
	return errors.Is(err, target)
}

// generateRandomToken returns a URL-safe random token
func generateRandomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return uuid.New().String()
	}
	return hex.EncodeToString(b)
}

// generateJobID returns a new UUID-based job id
func generateJobID() string {
	return uuid.New().String()
}

// nilIfNullInt64 returns a pointer to int64 if valid, otherwise nil
func nilIfNullInt64(n sql.NullInt64) *int64 {
	if !n.Valid {
		return nil
	}
	v := n.Int64
	return &v
}

// nilIfNullFloat64 returns a pointer to float64 if valid, otherwise nil
func nilIfNullFloat64(n sql.NullFloat64) *float64 {
	if !n.Valid {
		return nil
	}
	v := n.Float64
	return &v
}

// respondJSON responds with JSON
func respondJSON(w http.ResponseWriter, statusCode int, response interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// nullString returns a pointer to the string if it's not empty, otherwise nil
func nullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
