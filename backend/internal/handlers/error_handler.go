package handlers

import (
	"fmt"
	"net/http"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// ErrorHandler handles error responses with consistent formatting
type ErrorHandler struct{}

// NewErrorHandler creates a new error handler
func NewErrorHandler() *ErrorHandler {
	return &ErrorHandler{}
}

// ValidateHeaders validates required headers and writes error response if missing
func (eh *ErrorHandler) ValidateHeaders(w http.ResponseWriter, tenantID string) error {
	if tenantID == "" {
		eh.BadRequest(w, "Missing X-Tenant-ID header")
		return fmt.Errorf("missing tenant ID")
	}
	return nil
}

// BadRequest writes a 400 Bad Request error
func (eh *ErrorHandler) BadRequest(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	fmt.Fprintf(w, `{"error": "bad_request", "message": "%s"}`, message)
}

// NotFound writes a 404 Not Found error
func (eh *ErrorHandler) NotFound(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, `{"error": "not_found", "message": "%s"}`, message)
}

// Unauthorized writes a 401 Unauthorized error
func (eh *ErrorHandler) Unauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	fmt.Fprintf(w, `{"error": "unauthorized", "message": "%s"}`, message)
}

// Forbidden writes a 403 Forbidden error
func (eh *ErrorHandler) Forbidden(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	fmt.Fprintf(w, `{"error": "forbidden", "message": "%s"}`, message)
}

// Conflict writes a 409 Conflict error
func (eh *ErrorHandler) Conflict(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusConflict)
	fmt.Fprintf(w, `{"error": "conflict", "message": "%s"}`, message)
}

// InternalError writes a 500 Internal Server Error
func (eh *ErrorHandler) InternalError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintf(w, `{"error": "internal_error", "message": "%s"}`, message)
}

// CommandFailed writes an error response for command failures
func (eh *ErrorHandler) CommandFailed(w http.ResponseWriter, err error) {
	message := err.Error()

	// Determine appropriate status code based on error type
	statusCode := http.StatusInternalServerError
	errorType := "command_failed"

	// Could add more sophisticated error categorization here
	// For now, default to 500 Internal Server Error

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	fmt.Fprintf(w, `{"error": "%s", "message": "%s"}`, errorType, message)
}

// ValidationError writes an error response for validation failures
func (eh *ErrorHandler) ValidationError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnprocessableEntity)
	fmt.Fprintf(w, `{"error": "validation_error", "message": "%s"}`, message)
}

// RateLimitExceeded writes a 429 Too Many Requests error
func (eh *ErrorHandler) RateLimitExceeded(w http.ResponseWriter, retryAfter string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", retryAfter)
	w.WriteHeader(http.StatusTooManyRequests)
	fmt.Fprintf(w, `{"error": "rate_limit_exceeded", "message": "Too many requests, please try again later"}`)
}

// ServiceUnavailable writes a 503 Service Unavailable error
func (eh *ErrorHandler) ServiceUnavailable(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusServiceUnavailable)
	fmt.Fprintf(w, `{"error": "service_unavailable", "message": "%s"}`, message)
}
