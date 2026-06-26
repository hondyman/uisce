package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// init sets up trace proxy authentication configuration on package initialization
func init() {
	initTraceAuthConfig()
}

// traceAuthConfig is the global configuration for trace proxy authentication
// This is initialized at package load time and persists for the server lifetime
var traceAuthConfig *TraceAuthConfig

// initTraceAuthConfig initializes the trace authentication configuration
// with API keys loaded from environment variables
func initTraceAuthConfig() {
	cfg := DefaultTraceAuthConfig()

	// Default single key shortcuts for initial deployments
	if k := os.Getenv("TRACE_API_KEY_DEFAULT"); k != "" {
		cfg.APIKeys[k] = []string{"admin"}
	}
	if k := os.Getenv("TRACE_API_KEY_SRE"); k != "" {
		cfg.APIKeys[k] = []string{"sre"}
	}

	// Support a simple CSV: TRACE_API_KEYS=key1:admin,key2:sre
	if list := os.Getenv("TRACE_API_KEYS"); list != "" {
		pairs := strings.Split(list, ",")
		for _, p := range pairs {
			parts := strings.SplitN(strings.TrimSpace(p), ":", 2)
			if len(parts) == 2 {
				k := parts[0]
				roles := strings.Split(parts[1], "|")
				cfg.APIKeys[k] = roles
			}
		}
	}

	traceAuthConfig = cfg
}

// proxyTempoTraces proxies trace search requests (by plan_id, service name, etc.) to the
// configured trace backend (Tempo/Jaeger) while enforcing authentication and tenant isolation.
// This centralizes trace access control, prevents CORS issues, and ensures audit logging.
//
// See TRACE_PROXY_AUTHENTICATION.md for behavioral details.
func (s *Server) proxyTempoTraces(w http.ResponseWriter, r *http.Request) {
	// Validate authentication first - most restrictive gate
	authInfo, tenantID, authErr, authStatus := ValidateTraceAuth(r, traceAuthConfig)
	if authErr != nil {
		writeJSONError(w, authStatus, "Authorization or validation error", "error", nil)
		return
	}

	// Require trace backend configuration
	traceBackend := os.Getenv("TRACE_QUERY_URL")
	if traceBackend == "" {
		writeJSONError(w, http.StatusServiceUnavailable, "Trace backend not configured", "service_unavailable", map[string]string{"env_var": "TRACE_QUERY_URL"})
		return
	}

	// Extract and validate query parameters
	planID := r.URL.Query().Get("plan_id")
	traceID := r.URL.Query().Get("trace_id")

	if valErr, valStatus := ValidateTraceQueryParams(planID, traceID); valErr != nil {
		writeJSONError(w, valStatus, "Authorization or validation error", "error", nil)
		return
	}

	// Add authentication context for downstream processing
	ctx := security.WithAuthInfo(r.Context(), *authInfo)

	// Build upstream URL with all query parameters
	upstream := fmt.Sprintf("%s?%s", traceBackend, r.URL.RawQuery)

	// Create HTTP client with timeout to prevent hung requests
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, upstream, nil)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to construct upstream trace request", "request_construction_failed", map[string]string{"error": err.Error()})
		return
	}

	// Forward Authorization header to upstream trace backend
	if auth := r.Header.Get("Authorization"); auth != "" {
		req.Header.Set("Authorization", auth)
	}

	// Execute upstream request
	resp, err := client.Do(req)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "Failed to contact trace backend", "trace_backend_unreachable", map[string]string{"details": fmt.Sprintf("%v", err)})
		return
	}
	defer resp.Body.Close()

	// Read the response body for inspection and filtering
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "Failed to read trace backend response", "trace_response_read_failed", map[string]string{"details": fmt.Sprintf("%v", err)})
		return
	}

	// Parse response to apply tenant filtering if configured
	tracesResponse := interface{}(nil)
	if len(body) > 0 && resp.Header.Get("Content-Type") == "application/json" {
		if err := json.Unmarshal(body, &tracesResponse); err == nil {
			// Apply tenant-based filtering to spans
			tracesResponse = filterTraceResponseByTenant(tracesResponse, tenantID)

			// Re-marshal the filtered response
			if filteredBody, err := json.Marshal(tracesResponse); err == nil {
				body = filteredBody
			}
		}
	}

	// Write response headers with appropriate cache control for trace data
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("X-Tenant-ID", tenantID)
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

// proxyTempoGetTrace proxies fetching a specific trace by ID from the trace backend
// while enforcing authentication and tenant isolation.
func (s *Server) proxyTempoGetTrace(w http.ResponseWriter, r *http.Request) {
	// Validate authentication first - most restrictive gate
	authInfo, tenantID, authErr, authStatus := ValidateTraceAuth(r, traceAuthConfig)
	if authErr != nil {
		writeJSONError(w, authStatus, "Authorization or validation error", "error", nil)
		return
	}

	// Require trace backend configuration
	traceBackend := os.Getenv("TRACE_QUERY_URL")
	if traceBackend == "" {
		writeJSONError(w, http.StatusServiceUnavailable, "Trace backend not configured", "service_unavailable", map[string]string{"env_var": "TRACE_QUERY_URL"})
		return
	}

	// Extract trace ID from URL path parameter
	traceID := chi.URLParam(r, "traceId")
	if traceID == "" {
		writeJSONError(w, http.StatusBadRequest, "Missing trace identifier", "bad_request", map[string]string{"param": "traceId"})
		return
	}

	// Validate trace ID format
	if !isValidTraceID(traceID) {
		writeJSONError(w, http.StatusBadRequest, "Invalid trace ID format", "bad_request", map[string]string{"expected": "32 hex chars"})
		return
	}

	// Add authentication context for downstream processing
	ctx := security.WithAuthInfo(r.Context(), *authInfo)

	// Build upstream URL for fetching specific trace
	upstream := fmt.Sprintf("%s/%s", traceBackend, traceID)

	// Create HTTP client with timeout
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, upstream, nil)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to construct upstream trace request", "request_construction_failed", map[string]string{"error": err.Error()})
		return
	}

	// Forward Authorization header to upstream trace backend
	if auth := r.Header.Get("Authorization"); auth != "" {
		req.Header.Set("Authorization", auth)
	}

	// Execute upstream request
	resp, err := client.Do(req)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "Failed to contact trace backend", "trace_backend_unreachable", map[string]string{"details": fmt.Sprintf("%v", err)})
		return
	}
	defer resp.Body.Close()

	// Handle trace not found at backend
	if resp.StatusCode == http.StatusNotFound {
		writeJSONError(w, http.StatusNotFound, "Trace not found", "not_found", map[string]string{"trace_id": traceID})
		return
	}

	// Read the response body for inspection and filtering
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "Failed to read trace backend response", "trace_response_read_failed", map[string]string{"details": fmt.Sprintf("%v", err)})
		return
	}

	// Parse and filter response to apply tenant isolation
	if len(body) > 0 && resp.Header.Get("Content-Type") == "application/json" {
		var traceData interface{}
		if err := json.Unmarshal(body, &traceData); err == nil {
			// Apply tenant-based filtering
			traceData = filterTraceResponseByTenant(traceData, tenantID)

			// Re-marshal the filtered response
			if filteredBody, err := json.Marshal(traceData); err == nil {
				body = filteredBody
			}
		}
	}

	// Write response headers with cache control
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("X-Tenant-ID", tenantID)
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

// filterTraceResponseByTenant filters trace response data to only include spans/data for the specified tenant
// This enforces strict tenant isolation - no cross-tenant data leakage
func filterTraceResponseByTenant(traceData interface{}, tenantID string) interface{} {
	// Map response structure and filter spans list if present
	switch data := traceData.(type) {
	case map[string]interface{}:
		filtered := make(map[string]interface{})

		for key, value := range data {
			switch key {
			case "spans", "traceSpans", "trace_spans":
				// These are lists of span objects - apply tenant filtering
				if spans, ok := value.([]interface{}); ok {
					filtered[key] = FilterSpansByTenant(spans, tenantID)
				} else {
					filtered[key] = value
				}

			case "traces", "data":
				// Recursively filter nested trace objects
				filtered[key] = filterTraceResponseByTenant(value, tenantID)

			default:
				// Pass through other fields unchanged
				filtered[key] = value
			}
		}

		return filtered

	case []interface{}:
		// For arrays, recursively filter each element
		var filtered []interface{}
		for _, item := range data {
			filtered = append(filtered, filterTraceResponseByTenant(item, tenantID))
		}
		return filtered

	default:
		// Primitive types pass through unchanged
		return traceData
	}
}
