package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// APIEndpoint represents an API endpoint in the catalog
type APIEndpoint struct {
	ID             string                 `json:"id"`
	TenantID       string                 `json:"tenant_id"`
	DatasourceID   string                 `json:"datasource_id"`
	EndpointName   string                 `json:"endpoint_name"`
	Description    string                 `json:"description"`
	HTTPMethod     string                 `json:"http_method"` // GET, POST, PATCH, DELETE
	URLPath        string                 `json:"url_path"`
	Category       string                 `json:"category"` // validation, entity, relationship, rule, etc.
	Subcategory    string                 `json:"subcategory"`
	Purpose        string                 `json:"purpose"` // create, read, update, delete, execute, etc.
	RequestSchema  map[string]interface{} `json:"request_schema,omitempty"`
	ResponseSchema map[string]interface{} `json:"response_schema,omitempty"`
	Parameters     []EndpointParameter    `json:"parameters,omitempty"`
	Examples       []EndpointExample      `json:"examples,omitempty"`
	Tags           []string               `json:"tags,omitempty"`
	RequiresAuth   bool                   `json:"requires_auth"`
	IsActive       bool                   `json:"is_active"`
	Version        string                 `json:"version"` // API version
	CreatedBy      *string                `json:"created_by,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// EndpointParameter represents a parameter for an API endpoint
type EndpointParameter struct {
	Name        string      `json:"name"`
	In          string      `json:"in"` // path, query, header, body
	Required    bool        `json:"required"`
	Description string      `json:"description"`
	DataType    string      `json:"data_type"`
	Example     interface{} `json:"example,omitempty"`
}

// EndpointExample represents an example request/response
type EndpointExample struct {
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Request     map[string]interface{} `json:"request,omitempty"`
	Response    map[string]interface{} `json:"response,omitempty"`
	StatusCode  int                    `json:"status_code"`
}

// APIEndpointRequest represents the request payload for creating/updating an endpoint
type APIEndpointRequest struct {
	EndpointName   string                 `json:"endpoint_name" binding:"required"`
	Description    string                 `json:"description"`
	HTTPMethod     string                 `json:"http_method" binding:"required"`
	URLPath        string                 `json:"url_path" binding:"required"`
	Category       string                 `json:"category" binding:"required"`
	Subcategory    string                 `json:"subcategory"`
	Purpose        string                 `json:"purpose"`
	RequestSchema  map[string]interface{} `json:"request_schema,omitempty"`
	ResponseSchema map[string]interface{} `json:"response_schema,omitempty"`
	Parameters     []EndpointParameter    `json:"parameters,omitempty"`
	Examples       []EndpointExample      `json:"examples,omitempty"`
	Tags           []string               `json:"tags,omitempty"`
	RequiresAuth   bool                   `json:"requires_auth"`
	IsActive       bool                   `json:"is_active"`
	Version        string                 `json:"version"`
}

// RegisterAPIEndpointsCatalogRoutes registers all API endpoints catalog routes
func RegisterAPIEndpointsCatalogRoutes(r chi.Router, db *sql.DB) {
	r.Get("/api-endpoints", handleListAPIEndpoints(db))
	r.Post("/api-endpoints", handleCreateAPIEndpoint(db))
	r.Get("/api-endpoints/{id}", handleGetAPIEndpoint(db))
	r.Patch("/api-endpoints/{id}", handleUpdateAPIEndpoint(db))
	r.Delete("/api-endpoints/{id}", handleDeleteAPIEndpoint(db))

	// Search and filter endpoints
	r.Get("/api-endpoints/category/{category}", handleListAPIEndpointsByCategory(db))
	r.Get("/api-endpoints/search", handleSearchAPIEndpoints(db))

	// Documentation endpoints
	r.Get("/api-endpoints/openapi", handleGetOpenAPISpec(db))
	r.Get("/api-endpoints/{id}/documentation", handleGetEndpointDocumentation(db))
}

// handleListAPIEndpoints retrieves all API endpoints with pagination and filtering
func handleListAPIEndpoints(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.URL.Query().Get("tenant_id")
		datasourceID := r.URL.Query().Get("datasource_id")

		if tenantID == "" {
			writeJSONError(w, http.StatusBadRequest, "tenant_id is required", "missing_tenant", "")
			return
		}

		// Pagination
		page := 1
		if p := r.URL.Query().Get("page"); p != "" {
			if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
				page = parsed
			}
		}
		limit := 50
		if l := r.URL.Query().Get("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
				limit = parsed
			}
		}
		offset := (page - 1) * limit

		// Filters
		category := r.URL.Query().Get("category")
		method := r.URL.Query().Get("method")
		searchQuery := r.URL.Query().Get("search")
		activeOnly := r.URL.Query().Get("active_only") == "true"

		query := `
			SELECT id, tenant_id, datasource_id, endpoint_name, description,
			       http_method, url_path, category, subcategory, purpose,
			       request_schema, response_schema, parameters, examples, tags,
			       requires_auth, is_active, version, created_by, created_at, updated_at
			FROM api_endpoints_catalog
			WHERE tenant_id = $1
		`
		args := []interface{}{tenantID}
		argCount := 2

		if datasourceID != "" {
			query += fmt.Sprintf(" AND datasource_id = $%d", argCount)
			args = append(args, datasourceID)
			argCount++
		}

		if category != "" {
			query += fmt.Sprintf(" AND category = $%d", argCount)
			args = append(args, category)
			argCount++
		}

		if method != "" {
			query += fmt.Sprintf(" AND http_method = $%d", argCount)
			args = append(args, method)
			argCount++
		}

		if activeOnly {
			query += " AND is_active = true"
		}

		if searchQuery != "" {
			query += fmt.Sprintf(" AND (endpoint_name ILIKE $%d OR description ILIKE $%d OR url_path ILIKE $%d)", argCount, argCount, argCount)
			args = append(args, "%"+searchQuery+"%", "%"+searchQuery+"%", "%"+searchQuery+"%")
			argCount += 3
		}

		// Count total
		countQuery := strings.Replace(query, "SELECT id, tenant_id, datasource_id, endpoint_name, description, http_method, url_path, category, subcategory, purpose, request_schema, response_schema, parameters, examples, tags, requires_auth, is_active, version, created_by, created_at, updated_at", "SELECT COUNT(*)", 1)
		var total int
		if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to count endpoints", "db_error", err.Error())
			return
		}

		// Get paginated results
		query += " ORDER BY created_at DESC LIMIT $" + fmt.Sprintf("%d", argCount) + " OFFSET $" + fmt.Sprintf("%d", argCount+1)
		args = append(args, limit, offset)

		rows, err := db.Query(query, args...)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to fetch endpoints", "db_error", err.Error())
			return
		}
		defer rows.Close()

		endpoints := []APIEndpoint{}
		for rows.Next() {
			var ep APIEndpoint
			var requestSchemaJSON, responseSchemaJSON, parametersJSON, examplesJSON []byte
			var tagsJSON []byte

			if err := rows.Scan(
				&ep.ID, &ep.TenantID, &ep.DatasourceID, &ep.EndpointName, &ep.Description,
				&ep.HTTPMethod, &ep.URLPath, &ep.Category, &ep.Subcategory, &ep.Purpose,
				&requestSchemaJSON, &responseSchemaJSON, &parametersJSON, &examplesJSON, &tagsJSON,
				&ep.RequiresAuth, &ep.IsActive, &ep.Version, &ep.CreatedBy, &ep.CreatedAt, &ep.UpdatedAt,
			); err != nil {
				writeJSONError(w, http.StatusInternalServerError, "Failed to scan endpoint", "scan_error", err.Error())
				return
			}

			if requestSchemaJSON != nil {
				json.Unmarshal(requestSchemaJSON, &ep.RequestSchema)
			}
			if responseSchemaJSON != nil {
				json.Unmarshal(responseSchemaJSON, &ep.ResponseSchema)
			}
			if parametersJSON != nil {
				json.Unmarshal(parametersJSON, &ep.Parameters)
			}
			if examplesJSON != nil {
				json.Unmarshal(examplesJSON, &ep.Examples)
			}
			if tagsJSON != nil {
				json.Unmarshal(tagsJSON, &ep.Tags)
			}

			endpoints = append(endpoints, ep)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data":       endpoints,
			"pagination": map[string]int{"page": page, "limit": limit, "total": total},
		})
	}
}

// handleCreateAPIEndpoint creates a new API endpoint in the catalog
func handleCreateAPIEndpoint(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.URL.Query().Get("tenant_id")
		datasourceID := r.URL.Query().Get("datasource_id")

		if tenantID == "" {
			writeJSONError(w, http.StatusBadRequest, "tenant_id is required", "missing_tenant", "")
			return
		}

		var req APIEndpointRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request body", "decode_error", err.Error())
			return
		}

		// Validate required fields
		if req.EndpointName == "" || req.HTTPMethod == "" || req.URLPath == "" || req.Category == "" {
			writeJSONError(w, http.StatusBadRequest, "Missing required fields", "validation_error", "")
			return
		}

		id := uuid.New().String()
		now := time.Now()

		requestSchemaJSON, _ := json.Marshal(req.RequestSchema)
		responseSchemaJSON, _ := json.Marshal(req.ResponseSchema)
		parametersJSON, _ := json.Marshal(req.Parameters)
		examplesJSON, _ := json.Marshal(req.Examples)
		tagsJSON, _ := json.Marshal(req.Tags)

		query := `
			INSERT INTO api_endpoints_catalog
			(id, tenant_id, datasource_id, endpoint_name, description, http_method, url_path,
			 category, subcategory, purpose, request_schema, response_schema, parameters,
			 examples, tags, requires_auth, is_active, version, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
		`

		if err := db.QueryRow(query,
			id, tenantID, datasourceID, req.EndpointName, req.Description, req.HTTPMethod,
			req.URLPath, req.Category, req.Subcategory, req.Purpose,
			requestSchemaJSON, responseSchemaJSON, parametersJSON, examplesJSON, tagsJSON,
			req.RequiresAuth, req.IsActive, req.Version, now, now,
		).Err(); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to create endpoint", "db_error", err.Error())
			return
		}

		ep := APIEndpoint{
			ID:             id,
			TenantID:       tenantID,
			DatasourceID:   datasourceID,
			EndpointName:   req.EndpointName,
			Description:    req.Description,
			HTTPMethod:     req.HTTPMethod,
			URLPath:        req.URLPath,
			Category:       req.Category,
			Subcategory:    req.Subcategory,
			Purpose:        req.Purpose,
			RequestSchema:  req.RequestSchema,
			ResponseSchema: req.ResponseSchema,
			Parameters:     req.Parameters,
			Examples:       req.Examples,
			Tags:           req.Tags,
			RequiresAuth:   req.RequiresAuth,
			IsActive:       req.IsActive,
			Version:        req.Version,
			CreatedAt:      now,
			UpdatedAt:      now,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(ep)
	}
}

// handleGetAPIEndpoint retrieves a specific API endpoint
func handleGetAPIEndpoint(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.URL.Query().Get("tenant_id")
		id := chi.URLParam(r, "id")

		if tenantID == "" {
			writeJSONError(w, http.StatusBadRequest, "tenant_id is required", "missing_tenant", "")
			return
		}

		query := `
			SELECT id, tenant_id, datasource_id, endpoint_name, description,
			       http_method, url_path, category, subcategory, purpose,
			       request_schema, response_schema, parameters, examples, tags,
			       requires_auth, is_active, version, created_by, created_at, updated_at
			FROM api_endpoints_catalog
			WHERE id = $1 AND tenant_id = $2
		`

		var ep APIEndpoint
		var requestSchemaJSON, responseSchemaJSON, parametersJSON, examplesJSON, tagsJSON []byte

		if err := db.QueryRow(query, id, tenantID).Scan(
			&ep.ID, &ep.TenantID, &ep.DatasourceID, &ep.EndpointName, &ep.Description,
			&ep.HTTPMethod, &ep.URLPath, &ep.Category, &ep.Subcategory, &ep.Purpose,
			&requestSchemaJSON, &responseSchemaJSON, &parametersJSON, &examplesJSON, &tagsJSON,
			&ep.RequiresAuth, &ep.IsActive, &ep.Version, &ep.CreatedBy, &ep.CreatedAt, &ep.UpdatedAt,
		); err == sql.ErrNoRows {
			writeJSONError(w, http.StatusNotFound, "Endpoint not found", "not_found", "")
			return
		} else if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to fetch endpoint", "db_error", err.Error())
			return
		}

		if requestSchemaJSON != nil {
			json.Unmarshal(requestSchemaJSON, &ep.RequestSchema)
		}
		if responseSchemaJSON != nil {
			json.Unmarshal(responseSchemaJSON, &ep.ResponseSchema)
		}
		if parametersJSON != nil {
			json.Unmarshal(parametersJSON, &ep.Parameters)
		}
		if examplesJSON != nil {
			json.Unmarshal(examplesJSON, &ep.Examples)
		}
		if tagsJSON != nil {
			json.Unmarshal(tagsJSON, &ep.Tags)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ep)
	}
}

// handleUpdateAPIEndpoint updates an existing API endpoint
func handleUpdateAPIEndpoint(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.URL.Query().Get("tenant_id")
		id := chi.URLParam(r, "id")

		if tenantID == "" {
			writeJSONError(w, http.StatusBadRequest, "tenant_id is required", "missing_tenant", "")
			return
		}

		var req APIEndpointRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request body", "decode_error", err.Error())
			return
		}

		requestSchemaJSON, _ := json.Marshal(req.RequestSchema)
		responseSchemaJSON, _ := json.Marshal(req.ResponseSchema)
		parametersJSON, _ := json.Marshal(req.Parameters)
		examplesJSON, _ := json.Marshal(req.Examples)
		tagsJSON, _ := json.Marshal(req.Tags)

		query := `
			UPDATE api_endpoints_catalog
			SET endpoint_name = $1, description = $2, http_method = $3, url_path = $4,
			    category = $5, subcategory = $6, purpose = $7, request_schema = $8,
			    response_schema = $9, parameters = $10, examples = $11, tags = $12,
			    requires_auth = $13, is_active = $14, version = $15, updated_at = $16
			WHERE id = $17 AND tenant_id = $18
			RETURNING id, tenant_id, datasource_id, endpoint_name, description,
			         http_method, url_path, category, subcategory, purpose,
			         request_schema, response_schema, parameters, examples, tags,
			         requires_auth, is_active, version, created_by, created_at, updated_at
		`

		var ep APIEndpoint
		var requestSchemaJSON2, responseSchemaJSON2, parametersJSON2, examplesJSON2, tagsJSON2 []byte

		if err := db.QueryRow(query,
			req.EndpointName, req.Description, req.HTTPMethod, req.URLPath,
			req.Category, req.Subcategory, req.Purpose, requestSchemaJSON,
			responseSchemaJSON, parametersJSON, examplesJSON, tagsJSON,
			req.RequiresAuth, req.IsActive, req.Version, time.Now(),
			id, tenantID,
		).Scan(
			&ep.ID, &ep.TenantID, &ep.DatasourceID, &ep.EndpointName, &ep.Description,
			&ep.HTTPMethod, &ep.URLPath, &ep.Category, &ep.Subcategory, &ep.Purpose,
			&requestSchemaJSON2, &responseSchemaJSON2, &parametersJSON2, &examplesJSON2, &tagsJSON2,
			&ep.RequiresAuth, &ep.IsActive, &ep.Version, &ep.CreatedBy, &ep.CreatedAt, &ep.UpdatedAt,
		); err == sql.ErrNoRows {
			writeJSONError(w, http.StatusNotFound, "Endpoint not found", "not_found", "")
			return
		} else if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to update endpoint", "db_error", err.Error())
			return
		}

		if requestSchemaJSON2 != nil {
			json.Unmarshal(requestSchemaJSON2, &ep.RequestSchema)
		}
		if responseSchemaJSON2 != nil {
			json.Unmarshal(responseSchemaJSON2, &ep.ResponseSchema)
		}
		if parametersJSON2 != nil {
			json.Unmarshal(parametersJSON2, &ep.Parameters)
		}
		if examplesJSON2 != nil {
			json.Unmarshal(examplesJSON2, &ep.Examples)
		}
		if tagsJSON2 != nil {
			json.Unmarshal(tagsJSON2, &ep.Tags)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ep)
	}
}

// handleDeleteAPIEndpoint deletes an API endpoint
func handleDeleteAPIEndpoint(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.URL.Query().Get("tenant_id")
		id := chi.URLParam(r, "id")

		if tenantID == "" {
			writeJSONError(w, http.StatusBadRequest, "tenant_id is required", "missing_tenant", "")
			return
		}

		result, err := db.Exec(
			"DELETE FROM api_endpoints_catalog WHERE id = $1 AND tenant_id = $2",
			id, tenantID,
		)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to delete endpoint", "db_error", err.Error())
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil || rowsAffected == 0 {
			writeJSONError(w, http.StatusNotFound, "Endpoint not found", "not_found", "")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleListAPIEndpointsByCategory retrieves endpoints by category
func handleListAPIEndpointsByCategory(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.URL.Query().Get("tenant_id")
		category := chi.URLParam(r, "category")

		if tenantID == "" {
			writeJSONError(w, http.StatusBadRequest, "tenant_id is required", "missing_tenant", "")
			return
		}

		query := `
			SELECT id, tenant_id, datasource_id, endpoint_name, description,
			       http_method, url_path, category, subcategory, purpose,
			       request_schema, response_schema, parameters, examples, tags,
			       requires_auth, is_active, version, created_by, created_at, updated_at
			FROM api_endpoints_catalog
			WHERE tenant_id = $1 AND category = $2
			ORDER BY subcategory, endpoint_name
		`

		rows, err := db.Query(query, tenantID, category)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to fetch endpoints", "db_error", err.Error())
			return
		}
		defer rows.Close()

		endpoints := []APIEndpoint{}
		for rows.Next() {
			var ep APIEndpoint
			var requestSchemaJSON, responseSchemaJSON, parametersJSON, examplesJSON, tagsJSON []byte

			if err := rows.Scan(
				&ep.ID, &ep.TenantID, &ep.DatasourceID, &ep.EndpointName, &ep.Description,
				&ep.HTTPMethod, &ep.URLPath, &ep.Category, &ep.Subcategory, &ep.Purpose,
				&requestSchemaJSON, &responseSchemaJSON, &parametersJSON, &examplesJSON, &tagsJSON,
				&ep.RequiresAuth, &ep.IsActive, &ep.Version, &ep.CreatedBy, &ep.CreatedAt, &ep.UpdatedAt,
			); err != nil {
				continue
			}

			if requestSchemaJSON != nil {
				json.Unmarshal(requestSchemaJSON, &ep.RequestSchema)
			}
			if responseSchemaJSON != nil {
				json.Unmarshal(responseSchemaJSON, &ep.ResponseSchema)
			}
			if parametersJSON != nil {
				json.Unmarshal(parametersJSON, &ep.Parameters)
			}
			if examplesJSON != nil {
				json.Unmarshal(examplesJSON, &ep.Examples)
			}
			if tagsJSON != nil {
				json.Unmarshal(tagsJSON, &ep.Tags)
			}

			endpoints = append(endpoints, ep)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"category": category,
			"data":     endpoints,
		})
	}
}

// handleSearchAPIEndpoints searches for endpoints by multiple criteria
func handleSearchAPIEndpoints(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.URL.Query().Get("tenant_id")
		search := r.URL.Query().Get("q")
		method := r.URL.Query().Get("method")
		category := r.URL.Query().Get("category")

		if tenantID == "" {
			writeJSONError(w, http.StatusBadRequest, "tenant_id is required", "missing_tenant", "")
			return
		}

		query := `
			SELECT id, tenant_id, datasource_id, endpoint_name, description,
			       http_method, url_path, category, subcategory, purpose,
			       request_schema, response_schema, parameters, examples, tags,
			       requires_auth, is_active, version, created_by, created_at, updated_at
			FROM api_endpoints_catalog
			WHERE tenant_id = $1 AND is_active = true
		`
		args := []interface{}{tenantID}
		argCount := 2

		if search != "" {
			query += fmt.Sprintf(" AND (endpoint_name ILIKE $%d OR description ILIKE $%d OR url_path ILIKE $%d)", argCount, argCount, argCount)
			args = append(args, "%"+search+"%", "%"+search+"%", "%"+search+"%")
			argCount += 3
		}

		if method != "" {
			query += fmt.Sprintf(" AND http_method = $%d", argCount)
			args = append(args, method)
			argCount++
		}

		if category != "" {
			query += fmt.Sprintf(" AND category = $%d", argCount)
			args = append(args, category)
			argCount++
		}

		query += " ORDER BY endpoint_name"

		rows, err := db.Query(query, args...)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to search endpoints", "db_error", err.Error())
			return
		}
		defer rows.Close()

		endpoints := []APIEndpoint{}
		for rows.Next() {
			var ep APIEndpoint
			var requestSchemaJSON, responseSchemaJSON, parametersJSON, examplesJSON, tagsJSON []byte

			if err := rows.Scan(
				&ep.ID, &ep.TenantID, &ep.DatasourceID, &ep.EndpointName, &ep.Description,
				&ep.HTTPMethod, &ep.URLPath, &ep.Category, &ep.Subcategory, &ep.Purpose,
				&requestSchemaJSON, &responseSchemaJSON, &parametersJSON, &examplesJSON, &tagsJSON,
				&ep.RequiresAuth, &ep.IsActive, &ep.Version, &ep.CreatedBy, &ep.CreatedAt, &ep.UpdatedAt,
			); err != nil {
				continue
			}

			if requestSchemaJSON != nil {
				json.Unmarshal(requestSchemaJSON, &ep.RequestSchema)
			}
			if responseSchemaJSON != nil {
				json.Unmarshal(responseSchemaJSON, &ep.ResponseSchema)
			}
			if parametersJSON != nil {
				json.Unmarshal(parametersJSON, &ep.Parameters)
			}
			if examplesJSON != nil {
				json.Unmarshal(examplesJSON, &ep.Examples)
			}
			if tagsJSON != nil {
				json.Unmarshal(tagsJSON, &ep.Tags)
			}

			endpoints = append(endpoints, ep)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"search": search,
			"data":   endpoints,
			"count":  len(endpoints),
		})
	}
}

// handleGetOpenAPISpec generates an OpenAPI specification for all endpoints
func handleGetOpenAPISpec(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.URL.Query().Get("tenant_id")

		if tenantID == "" {
			writeJSONError(w, http.StatusBadRequest, "tenant_id is required", "missing_tenant", "")
			return
		}

		query := `
			SELECT category, COUNT(*) as count
			FROM api_endpoints_catalog
			WHERE tenant_id = $1 AND is_active = true
			GROUP BY category
		`

		rows, err := db.Query(query, tenantID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to fetch spec", "db_error", err.Error())
			return
		}
		defer rows.Close()

		paths := make(map[string]interface{})
		for rows.Next() {
			var category string
			var count int
			if err := rows.Scan(&category, &count); err != nil {
				continue
			}
			paths[category] = map[string]interface{}{
				"count":   count,
				"summary": "Endpoints for " + category,
			}
		}

		spec := map[string]interface{}{
			"openapi": "3.0.0",
			"info": map[string]interface{}{
				"title":   "API Endpoints Catalog",
				"version": "1.0.0",
			},
			"paths": paths,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(spec)
	}
}

// handleGetEndpointDocumentation retrieves documentation for a specific endpoint
func handleGetEndpointDocumentation(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.URL.Query().Get("tenant_id")
		id := chi.URLParam(r, "id")

		if tenantID == "" {
			writeJSONError(w, http.StatusBadRequest, "tenant_id is required", "missing_tenant", "")
			return
		}

		query := `
			SELECT id, endpoint_name, description, http_method, url_path,
			       category, subcategory, purpose, request_schema, response_schema,
			       parameters, examples, version
			FROM api_endpoints_catalog
			WHERE id = $1 AND tenant_id = $2
		`

		var ep struct {
			ID             string
			EndpointName   string
			Description    string
			HTTPMethod     string
			URLPath        string
			Category       string
			Subcategory    string
			Purpose        string
			RequestSchema  json.RawMessage
			ResponseSchema json.RawMessage
			Parameters     json.RawMessage
			Examples       json.RawMessage
			Version        string
		}

		if err := db.QueryRow(query, id, tenantID).Scan(
			&ep.ID, &ep.EndpointName, &ep.Description, &ep.HTTPMethod, &ep.URLPath,
			&ep.Category, &ep.Subcategory, &ep.Purpose, &ep.RequestSchema, &ep.ResponseSchema,
			&ep.Parameters, &ep.Examples, &ep.Version,
		); err == sql.ErrNoRows {
			writeJSONError(w, http.StatusNotFound, "Endpoint not found", "not_found", "")
			return
		} else if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to fetch documentation", "db_error", err.Error())
			return
		}

		doc := map[string]interface{}{
			"id":              ep.ID,
			"title":           ep.EndpointName,
			"description":     ep.Description,
			"method":          ep.HTTPMethod,
			"path":            ep.URLPath,
			"category":        ep.Category,
			"subcategory":     ep.Subcategory,
			"purpose":         ep.Purpose,
			"version":         ep.Version,
			"request_schema":  ep.RequestSchema,
			"response_schema": ep.ResponseSchema,
			"parameters":      ep.Parameters,
			"examples":        ep.Examples,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(doc)
	}
}
