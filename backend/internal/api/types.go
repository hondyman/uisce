package api

import "time"

// APIResponse represents a standardized API response
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// APIError represents an API error response
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Meta contains metadata for API responses
type Meta struct {
	RequestID      string    `json:"request_id"`
	Timestamp      time.Time `json:"timestamp"`
	ProcessingTime string    `json:"processing_time,omitempty"`
}
