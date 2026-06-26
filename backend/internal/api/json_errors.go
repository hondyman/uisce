package api

// ErrorResponse is the standard JSON error payload returned by API endpoints.
// ErrorResponse is the standard JSON error payload returned by API endpoints.
type ErrorResponse struct {
	Error     string      `json:"error"`
	Code      int         `json:"code"`
	ErrorCode string      `json:"error_code,omitempty"`
	Details   interface{} `json:"details,omitempty"`
}

// Note: writeJSONError is implemented in helpers.go to provide a single
// canonical implementation. This file only defines the ErrorResponse
// payload type to keep JSON shape definitions colocated.
