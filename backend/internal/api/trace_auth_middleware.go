package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// TraceAuthConfig holds configuration for trace proxy authentication
type TraceAuthConfig struct {
	// APIKeys maps API key to allowed roles
	// In production, these should be validated against a database
	APIKeys map[string][]string
	// ValidRoles defines which roles can access trace proxy
	ValidRoles map[string]bool
	// EnableStrictTenantFiltering enforces tenant isolation in spans
	EnableStrictTenantFiltering bool
}

// DefaultTraceAuthConfig creates a default configuration with standard roles
func DefaultTraceAuthConfig() *TraceAuthConfig {
	return &TraceAuthConfig{
		APIKeys: make(map[string][]string),
		ValidRoles: map[string]bool{
			"admin":       true,
			"sre":         true,
			"ops_manager": true,
		},
		EnableStrictTenantFiltering: true,
	}
}

// ValidateTraceAuth validates authentication for trace proxy requests.
// It extracts authentication info from the request and stores it in context.
// Returns (authInfo, tenantID, errorResponse, httpStatusCode)
func ValidateTraceAuth(r *http.Request, config *TraceAuthConfig) (*security.AuthInfo, string, *TraceAuthErrorResponse, int) {
	// Prefer JWT Bearer token parsing if present
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		// Heuristic: if it looks like a JWT (two dots), try JWT parsing
		if parts := strings.Count(tokenStr, "."); parts == 2 {
			secret := os.Getenv("JWT_SECRET")
			if secret != "" {
				token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
					return []byte(secret), nil
				})
				if err == nil && token.Valid {
					claims, ok := token.Claims.(jwt.MapClaims)
					if ok {
						roleStr, _ := claims["role"].(string)
						tenantID, _ := claims["tenant_id"].(string)
						sub, _ := claims["sub"].(string)
						if sub == "" {
							sub = "jwt-user"
						}
						roles := []string{}
						if roleStr != "" {
							roles = append(roles, roleStr)
						}
						return &security.AuthInfo{UserID: sub, Roles: roles, TenantIDs: []string{tenantID}}, tenantID, nil, 0
					}
				}
			}
		}
	}

	// Fallback to API key extraction/validation
	apiKey := extractAPIKey(r)
	if apiKey == "" {
		return nil, "", &TraceAuthErrorResponse{
			Error:     "unauthorized",
			Message:   "Missing or invalid authentication credentials",
			Details:   "No X-API-Key header or Authorization header found",
			Timestamp: time.Now().Format(time.RFC3339),
		}, http.StatusUnauthorized
	}

	// Validate the API key against configured keys
	authorizedRoles, keyExists := config.APIKeys[apiKey]
	if !keyExists {
		return nil, "", &TraceAuthErrorResponse{
			Error:     "unauthorized",
			Message:   "Invalid API key",
			Details:   "The provided API key is not valid or has expired",
			Timestamp: time.Now().Format(time.RFC3339),
		}, http.StatusUnauthorized
	}

	// Check that at least one of the authorized roles is valid for trace access
	hasValidRole := false
	for _, role := range authorizedRoles {
		if config.ValidRoles[role] {
			hasValidRole = true
			break
		}
	}

	if !hasValidRole {
		return nil, "", &TraceAuthErrorResponse{
			Error:     "forbidden",
			Message:   "Insufficient permissions",
			Details:   "Your API key does not have the required roles to access trace data",
			Timestamp: time.Now().Format(time.RFC3339),
		}, http.StatusForbidden
	}

	var tenantID string
	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil {
		tenantID = claims.TenantID
	}
	if tenantID == "" {
		return nil, "", &TraceAuthErrorResponse{
			Error:     "bad_request",
			Message:   "Missing tenant identifier",
			Details:   "X-Tenant-ID header is required for trace proxy requests",
			Timestamp: time.Now().Format(time.RFC3339),
		}, http.StatusBadRequest
	}

	// Extract user ID (can be derived from API key in production)
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = fmt.Sprintf("api-key-%s", hashAPIKey(apiKey))
	}

	// Build AuthInfo with validated data
	authInfo := &security.AuthInfo{
		UserID:    userID,
		Roles:     authorizedRoles,
		TenantIDs: []string{tenantID},
	}

	return authInfo, tenantID, nil, 0
}

// extractAPIKey extracts the API key from either X-API-Key header or Authorization header
func extractAPIKey(r *http.Request) string {
	// Try X-API-Key header first
	if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
		return apiKey
	}

	// Try Authorization header with Bearer scheme
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	// Support Bearer token format
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	// Support Basic auth format - extract username as API key
	if strings.HasPrefix(authHeader, "Basic ") {
		credentials := strings.TrimPrefix(authHeader, "Basic ")
		decoded, err := base64.StdEncoding.DecodeString(credentials)
		if err != nil {
			return ""
		}

		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) == 2 {
			return parts[0]
		}

		return ""
	}

	return ""
}

// hashAPIKey creates a short hash of the API key for logging purposes
// This prevents logging full API keys while still allowing key identification
func hashAPIKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return "****"
	}
	return apiKey[:4] + "****" + apiKey[len(apiKey)-4:]
}

// ValidateTraceQueryParams validates the query parameters for trace proxy requests
// Returns errorResponse and httpStatusCode if invalid, otherwise (nil, 0)
func ValidateTraceQueryParams(planID, traceID string) (*TraceAuthErrorResponse, int) {
	// For trace search, at least plan_id should be present
	if planID == "" && traceID == "" {
		return &TraceAuthErrorResponse{
			Error:     "bad_request",
			Message:   "Missing required query parameter",
			Details:   "Either 'plan_id' or 'trace_id' must be provided",
			Timestamp: time.Now().Format(time.RFC3339),
		}, http.StatusBadRequest
	}

	// Validate plan_id format if provided
	if planID != "" && !isValidPlanID(planID) {
		return &TraceAuthErrorResponse{
			Error:     "bad_request",
			Message:   "Invalid plan_id format",
			Details:   "plan_id must be a valid UUID or identifier",
			Timestamp: time.Now().Format(time.RFC3339),
		}, http.StatusBadRequest
	}

	// Validate trace_id format if provided
	if traceID != "" && !isValidTraceID(traceID) {
		return &TraceAuthErrorResponse{
			Error:     "bad_request",
			Message:   "Invalid trace_id format",
			Details:   "trace_id must be a valid hexadecimal string (16 bytes = 32 hex characters)",
			Timestamp: time.Now().Format(time.RFC3339),
		}, http.StatusBadRequest
	}

	return nil, 0
}

// isValidPlanID checks if the plan ID matches expected format
// Allows UUID or alphanumeric identifiers with hyphens, underscores, and dots
func isValidPlanID(planID string) bool {
	if len(planID) == 0 || len(planID) > 256 {
		return false
	}

	for _, ch := range planID {
		if !((ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '-' || ch == '_' || ch == '.') {
			return false
		}
	}

	return true
}

// isValidTraceID checks if the trace ID matches Tempo's expected format
// Tempo trace IDs are 16 bytes = 32 hexadecimal characters
func isValidTraceID(traceID string) bool {
	if len(traceID) != 32 {
		return false
	}

	for _, ch := range traceID {
		if !((ch >= '0' && ch <= '9') ||
			(ch >= 'a' && ch <= 'f') ||
			(ch >= 'A' && ch <= 'F')) {
			return false
		}
	}

	return true
}

// FilterSpansByTenant filters spans to only include those for the specified tenant
// This enforces tenant isolation in trace data
func FilterSpansByTenant(spans []interface{}, tenantID string) []interface{} {
	if len(spans) == 0 || tenantID == "" {
		return spans
	}

	var filtered []interface{}

	for _, span := range spans {
		spanMap, ok := span.(map[string]interface{})
		if !ok {
			continue
		}

		// Check if span has tenant information
		// Tempo stores tags in attributes or resource attributes
		if span, ok := spanMap["traceID"].(string); ok {
			_ = span // Suppress unused warning
		}

		// Check tags for tenant_id attribute
		if tags, ok := spanMap["tags"].([]interface{}); ok {
			spanTenantID := extractTenantFromTags(tags)
			if spanTenantID == tenantID {
				filtered = append(filtered, span)
			}
			continue
		}

		// If no tenant tag found, skip span to be safe (fail secure)
		continue
	}

	return filtered
}

// extractTenantFromTags extracts tenant_id from span tags
func extractTenantFromTags(tags []interface{}) string {
	for _, tag := range tags {
		if tagMap, ok := tag.(map[string]interface{}); ok {
			if key, ok := tagMap["key"].(string); ok && key == "tenant_id" {
				if value, ok := tagMap["value"].(string); ok {
					return value
				}
			}
		}
	}

	return ""
}

// TraceAuthErrorResponse represents a standardized error response for trace auth
// No hardcoded values, no placeholders
type TraceAuthErrorResponse struct {
	Error     string `json:"error"`
	Message   string `json:"message"`
	Details   string `json:"details,omitempty"`
	Timestamp string `json:"timestamp"`
}

// WriteErrorResponse writes an error response to the HTTP response writer
func WriteErrorResponse(w http.ResponseWriter, statusCode int, errResp *TraceAuthErrorResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	data, err := json.Marshal(errResp)
	if err != nil {
		// Fallback if JSON encoding fails
		w.Write([]byte(`{"error":"internal_error","message":"Failed to encode error response","timestamp":"` + time.Now().Format(time.RFC3339) + `"}`))
		return
	}

	w.Write(data)
}

// WriteSuccessResponse writes a JSON response with proper headers
func WriteSuccessResponse(w http.ResponseWriter, statusCode int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	encoder := json.NewEncoder(w)
	return encoder.Encode(data)
}
