package common

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// PaginationParams represents pagination query parameters
type PaginationParams struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Page   int `json:"page,omitempty"`
}

// FilterParams represents common filter parameters
type FilterParams struct {
	TenantID     string                 `json:"tenant_id"`
	DatasourceID string                 `json:"datasource_id"`
	Filters      map[string]interface{} `json:"filters,omitempty"`
}

// SortParams represents sorting parameters
type SortParams struct {
	SortBy    string `json:"sort_by"`
	SortOrder string `json:"sort_order"` // "asc" or "desc"
}

// Pagination represents pagination metadata in responses
type Pagination struct {
	Total      int  `json:"total"`
	Limit      int  `json:"limit"`
	Offset     int  `json:"offset"`
	Page       int  `json:"page,omitempty"`
	TotalPages int  `json:"total_pages,omitempty"`
	HasMore    bool `json:"has_more"`
}

// ResponseEnvelope wraps API responses with consistent structure
type ResponseEnvelope struct {
	Data       interface{}            `json:"data"`
	Pagination *Pagination            `json:"pagination,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Success    bool                   `json:"success"`
}

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message"`
	Code    int                    `json:"code"`
	Details map[string]interface{} `json:"details,omitempty"`
	Success bool                   `json:"success"`
}

// ParsePaginationParams extracts pagination parameters from request
// Default: limit=50, offset=0
func ParsePaginationParams(r *http.Request) PaginationParams {
	query := r.URL.Query()

	limit := 50
	if limitStr := query.Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
			if limit > 1000 {
				limit = 1000 // Cap at 1000
			}
		}
	}

	offset := 0
	if offsetStr := query.Get("offset"); offsetStr != "" {
		if parsed, err := strconv.Atoi(offsetStr); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	page := 0
	if pageStr := query.Get("page"); pageStr != "" {
		if parsed, err := strconv.Atoi(pageStr); err == nil && parsed > 0 {
			page = parsed
			offset = (page - 1) * limit
		}
	}

	return PaginationParams{
		Limit:  limit,
		Offset: offset,
		Page:   page,
	}
}

// ParseFilterParams extracts filter parameters from request
func ParseFilterParams(r *http.Request) FilterParams {
	query := r.URL.Query()

	filters := make(map[string]interface{})

	// Parse common filters
	for key, values := range query {
		if key == "limit" || key == "offset" || key == "page" ||
			key == "sort_by" || key == "sort_order" ||
			key == "tenant_id" || key == "datasource_id" {
			continue
		}

		if len(values) == 1 {
			filters[key] = values[0]
		} else if len(values) > 1 {
			filters[key] = values
		}
	}

	return FilterParams{
		TenantID:     query.Get("tenant_id"),
		DatasourceID: query.Get("datasource_id"),
		Filters:      filters,
	}
}

// ParseSortParams extracts sorting parameters from request
func ParseSortParams(r *http.Request) SortParams {
	query := r.URL.Query()

	sortBy := query.Get("sort_by")
	sortOrder := query.Get("sort_order")

	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "asc" // Default
	}

	return SortParams{
		SortBy:    sortBy,
		SortOrder: sortOrder,
	}
}

// GetTenantScope extracts tenant and datasource from headers or query
func GetTenantScope(r *http.Request) (tenantID, datasourceID string, err error) {
	// Try headers first
	tenantID = jwtmiddleware.GetClaimsFromContext(r).TenantID
	datasourceID = r.Header.Get("X-Tenant-Datasource-ID")

	// Fallback to query params
	if tenantID == "" {
		tenantID = r.URL.Query().Get("tenant_id")
	}
	if datasourceID == "" {
		datasourceID = r.URL.Query().Get("datasource_id")
	}

	if tenantID == "" {
		return "", "", fmt.Errorf("tenant_id is required")
	}
	if datasourceID == "" {
		return "", "", fmt.Errorf("datasource_id is required")
	}

	return tenantID, datasourceID, nil
}

// WrapResponse creates a standardized success response envelope
func WrapResponse(data interface{}, pagination *Pagination) ResponseEnvelope {
	return ResponseEnvelope{
		Data:       data,
		Pagination: pagination,
		Success:    true,
	}
}

// WrapResponseWithMetadata creates a response with additional metadata
func WrapResponseWithMetadata(data interface{}, pagination *Pagination, metadata map[string]interface{}) ResponseEnvelope {
	return ResponseEnvelope{
		Data:       data,
		Pagination: pagination,
		Metadata:   metadata,
		Success:    true,
	}
}

// CalculatePagination computes pagination metadata
func CalculatePagination(total, limit, offset int) *Pagination {
	hasMore := offset+limit < total

	totalPages := 0
	page := 0
	if limit > 0 {
		totalPages = (total + limit - 1) / limit
		page = (offset / limit) + 1
	}

	return &Pagination{
		Total:      total,
		Limit:      limit,
		Offset:     offset,
		Page:       page,
		TotalPages: totalPages,
		HasMore:    hasMore,
	}
}

// WriteJSON writes a JSON response
func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

// WriteSuccess writes a standardized success response
func WriteSuccess(w http.ResponseWriter, data interface{}, pagination *Pagination) {
	WriteJSON(w, http.StatusOK, WrapResponse(data, pagination))
}

// WriteCreated writes a 201 Created response
func WriteCreated(w http.ResponseWriter, data interface{}) {
	WriteJSON(w, http.StatusCreated, WrapResponse(data, nil))
}

// WriteNoContent writes a 204 No Content response
func WriteNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// HandleError writes a standardized error response
func HandleError(w http.ResponseWriter, err error, statusCode int) {
	errResp := ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: err.Error(),
		Code:    statusCode,
		Success: false,
	}
	WriteJSON(w, statusCode, errResp)
}

// HandleErrorWithDetails writes an error response with additional details
func HandleErrorWithDetails(w http.ResponseWriter, err error, statusCode int, details map[string]interface{}) {
	errResp := ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: err.Error(),
		Code:    statusCode,
		Details: details,
		Success: false,
	}
	WriteJSON(w, statusCode, errResp)
}

// HandleBadRequest writes a 400 Bad Request error
func HandleBadRequest(w http.ResponseWriter, err error) {
	HandleError(w, err, http.StatusBadRequest)
}

// HandleUnauthorized writes a 401 Unauthorized error
func HandleUnauthorized(w http.ResponseWriter, err error) {
	HandleError(w, err, http.StatusUnauthorized)
}

// HandleForbidden writes a 403 Forbidden error
func HandleForbidden(w http.ResponseWriter, err error) {
	HandleError(w, err, http.StatusForbidden)
}

// HandleNotFound writes a 404 Not Found error
func HandleNotFound(w http.ResponseWriter, err error) {
	HandleError(w, err, http.StatusNotFound)
}

// HandleInternalError writes a 500 Internal Server Error
func HandleInternalError(w http.ResponseWriter, err error) {
	log.Printf("Internal server error: %v", err)
	HandleError(w, err, http.StatusInternalServerError)
}

// ValidateRequiredFields checks if required fields are present
func ValidateRequiredFields(data map[string]interface{}, required []string) error {
	missing := []string{}

	for _, field := range required {
		if val, ok := data[field]; !ok || val == nil || val == "" {
			missing = append(missing, field)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required fields: %v", missing)
	}

	return nil
}

// ParseJSONBody parses JSON request body into the target struct
func ParseJSONBody(r *http.Request, target interface{}) error {
	if r.Body == nil {
		return fmt.Errorf("request body is empty")
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields() // Strict parsing

	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	return nil
}
