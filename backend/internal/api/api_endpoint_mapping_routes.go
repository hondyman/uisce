package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// EndpointEntityMapping represents a relationship between an API endpoint and an entity
type EndpointEntityMapping struct {
	ID               string    `json:"id"`
	APIEndpointID    string    `json:"api_endpoint_id"`
	EntityID         string    `json:"entity_id"`
	TenantID         string    `json:"tenant_id"`
	RelationshipType string    `json:"relationship_type"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// EndpointDatasourceMapping represents a relationship between an API endpoint and a datasource
type EndpointDatasourceMapping struct {
	ID               string    `json:"id"`
	APIEndpointID    string    `json:"api_endpoint_id"`
	DatasourceID     string    `json:"datasource_id"`
	TenantID         string    `json:"tenant_id"`
	RelationshipType string    `json:"relationship_type"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// RegisterEndpointMappingRoutes registers all endpoint mapping routes
func RegisterEndpointMappingRoutes(r chi.Router, db *sql.DB) {
	// Entity mappings
	r.Get("/api-endpoints/{endpoint-id}/entity-mappings", handleListEntityMappings(db))
	r.Post("/api-endpoints/{endpoint-id}/entity-mappings", handleCreateEntityMapping(db))
	r.Delete("/api-endpoints/{endpoint-id}/entity-mappings/{entity-id}", handleDeleteEntityMapping(db))

	// Datasource mappings
	r.Get("/api-endpoints/{endpoint-id}/datasource-mappings", handleListDatasourceMappings(db))
	r.Post("/api-endpoints/{endpoint-id}/datasource-mappings", handleCreateDatasourceMapping(db))
	r.Delete("/api-endpoints/{endpoint-id}/datasource-mappings/{datasource-id}", handleDeleteDatasourceMapping(db))

	// Reverse lookups
	r.Get("/entities/{entity-id}/api-endpoints", handleGetEntityEndpoints(db))
	r.Get("/datasources/{datasource-id}/api-endpoints", handleGetDatasourceEndpoints(db))
}

// handleListEntityMappings lists all entity mappings for an endpoint
func handleListEntityMappings(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.URL.Query().Get("tenant_id")
		endpointID := chi.URLParam(r, "endpoint-id")

		if tenantID == "" {
			writeJSONError(w, http.StatusBadRequest, "tenant_id is required", "missing_tenant", "")
			return
		}

		query := `
			SELECT id, api_endpoint_id, entity_id, tenant_id, relationship_type, created_at, updated_at
			FROM api_endpoint_entity_mappings
			WHERE api_endpoint_id = $1 AND tenant_id = $2
			ORDER BY created_at DESC
		`

		rows, err := db.Query(query, endpointID, tenantID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to fetch mappings", "db_error", err.Error())
			return
		}
		defer rows.Close()

		mappings := []EndpointEntityMapping{}
		for rows.Next() {
			var m EndpointEntityMapping
			if err := rows.Scan(&m.ID, &m.APIEndpointID, &m.EntityID, &m.TenantID, &m.RelationshipType, &m.CreatedAt, &m.UpdatedAt); err != nil {
				continue
			}
			mappings = append(mappings, m)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"endpoint_id": endpointID,
			"data":        mappings,
		})
	}
}

// handleCreateEntityMapping creates a new entity mapping for an endpoint
func handleCreateEntityMapping(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.URL.Query().Get("tenant_id")
		endpointID := chi.URLParam(r, "endpoint-id")

		if tenantID == "" {
			writeJSONError(w, http.StatusBadRequest, "tenant_id is required", "missing_tenant", "")
			return
		}

		var req struct {
			EntityID         string `json:"entity_id"`
			RelationshipType string `json:"relationship_type"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request body", "decode_error", err.Error())
			return
		}

		if req.EntityID == "" || req.RelationshipType == "" {
			writeJSONError(w, http.StatusBadRequest, "Missing required fields", "validation_error", "")
			return
		}

		id := uuid.New().String()
		now := time.Now()

		query := `
			INSERT INTO api_endpoint_entity_mappings
			(id, api_endpoint_id, entity_id, tenant_id, relationship_type, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (api_endpoint_id, entity_id, tenant_id, relationship_type) DO NOTHING
		`

		if err := db.QueryRow(query, id, endpointID, req.EntityID, tenantID, req.RelationshipType, now, now).Err(); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to create mapping", "db_error", err.Error())
			return
		}

		mapping := EndpointEntityMapping{
			ID:               id,
			APIEndpointID:    endpointID,
			EntityID:         req.EntityID,
			TenantID:         tenantID,
			RelationshipType: req.RelationshipType,
			CreatedAt:        now,
			UpdatedAt:        now,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(mapping)
	}
}

// handleDeleteEntityMapping deletes an entity mapping
func handleDeleteEntityMapping(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.URL.Query().Get("tenant_id")
		endpointID := chi.URLParam(r, "endpoint-id")
		entityID := chi.URLParam(r, "entity-id")

		if tenantID == "" {
			writeJSONError(w, http.StatusBadRequest, "tenant_id is required", "missing_tenant", "")
			return
		}

		result, err := db.Exec(
			"DELETE FROM api_endpoint_entity_mappings WHERE api_endpoint_id = $1 AND entity_id = $2 AND tenant_id = $3",
			endpointID, entityID, tenantID,
		)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to delete mapping", "db_error", err.Error())
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil || rowsAffected == 0 {
			writeJSONError(w, http.StatusNotFound, "Mapping not found", "not_found", "")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// handleListDatasourceMappings lists all datasource mappings for an endpoint
func handleListDatasourceMappings(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.URL.Query().Get("tenant_id")
		endpointID := chi.URLParam(r, "endpoint-id")

		if tenantID == "" {
			writeJSONError(w, http.StatusBadRequest, "tenant_id is required", "missing_tenant", "")
			return
		}

		query := `
			SELECT id, api_endpoint_id, datasource_id, tenant_id, relationship_type, created_at, updated_at
			FROM api_endpoint_datasource_mappings
			WHERE api_endpoint_id = $1 AND tenant_id = $2
			ORDER BY created_at DESC
		`

		rows, err := db.Query(query, endpointID, tenantID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to fetch mappings", "db_error", err.Error())
			return
		}
		defer rows.Close()

		mappings := []EndpointDatasourceMapping{}
		for rows.Next() {
			var m EndpointDatasourceMapping
			if err := rows.Scan(&m.ID, &m.APIEndpointID, &m.DatasourceID, &m.TenantID, &m.RelationshipType, &m.CreatedAt, &m.UpdatedAt); err != nil {
				continue
			}
			mappings = append(mappings, m)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"endpoint_id": endpointID,
			"data":        mappings,
		})
	}
}

// handleCreateDatasourceMapping creates a new datasource mapping for an endpoint
func handleCreateDatasourceMapping(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.URL.Query().Get("tenant_id")
		endpointID := chi.URLParam(r, "endpoint-id")

		if tenantID == "" {
			writeJSONError(w, http.StatusBadRequest, "tenant_id is required", "missing_tenant", "")
			return
		}

		var req struct {
			DatasourceID     string `json:"datasource_id"`
			RelationshipType string `json:"relationship_type"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request body", "decode_error", err.Error())
			return
		}

		if req.DatasourceID == "" || req.RelationshipType == "" {
			writeJSONError(w, http.StatusBadRequest, "Missing required fields", "validation_error", "")
			return
		}

		id := uuid.New().String()
		now := time.Now()

		query := `
			INSERT INTO api_endpoint_datasource_mappings
			(id, api_endpoint_id, datasource_id, tenant_id, relationship_type, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (api_endpoint_id, datasource_id, tenant_id, relationship_type) DO NOTHING
		`

		if err := db.QueryRow(query, id, endpointID, req.DatasourceID, tenantID, req.RelationshipType, now, now).Err(); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to create mapping", "db_error", err.Error())
			return
		}

		mapping := EndpointDatasourceMapping{
			ID:               id,
			APIEndpointID:    endpointID,
			DatasourceID:     req.DatasourceID,
			TenantID:         tenantID,
			RelationshipType: req.RelationshipType,
			CreatedAt:        now,
			UpdatedAt:        now,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(mapping)
	}
}

// handleDeleteDatasourceMapping deletes a datasource mapping
func handleDeleteDatasourceMapping(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.URL.Query().Get("tenant_id")
		endpointID := chi.URLParam(r, "endpoint-id")
		datasourceID := chi.URLParam(r, "datasource-id")

		if tenantID == "" {
			writeJSONError(w, http.StatusBadRequest, "tenant_id is required", "missing_tenant", "")
			return
		}

		result, err := db.Exec(
			"DELETE FROM api_endpoint_datasource_mappings WHERE api_endpoint_id = $1 AND datasource_id = $2 AND tenant_id = $3",
			endpointID, datasourceID, tenantID,
		)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to delete mapping", "db_error", err.Error())
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil || rowsAffected == 0 {
			writeJSONError(w, http.StatusNotFound, "Mapping not found", "not_found", "")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// handleGetEntityEndpoints retrieves all API endpoints for a specific entity
func handleGetEntityEndpoints(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.URL.Query().Get("tenant_id")
		entityID := chi.URLParam(r, "entity-id")

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

		query := `
			SELECT aec.id, aec.endpoint_name, aec.description, aec.http_method, aec.url_path,
			       aec.category, aec.subcategory, aec.purpose, aec.version, aem.relationship_type,
			       aec.created_at, aec.updated_at
			FROM api_endpoints_catalog aec
			JOIN api_endpoint_entity_mappings aem ON aec.id = aem.api_endpoint_id
			WHERE aem.entity_id = $1 AND aem.tenant_id = $2 AND aec.is_active = true
			ORDER BY aec.category, aec.endpoint_name
			LIMIT $3 OFFSET $4
		`

		rows, err := db.Query(query, entityID, tenantID, limit, offset)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to fetch endpoints", "db_error", err.Error())
			return
		}
		defer rows.Close()

		type EndpointWithRelationship struct {
			ID               string    `json:"id"`
			EndpointName     string    `json:"endpoint_name"`
			Description      string    `json:"description"`
			HTTPMethod       string    `json:"http_method"`
			URLPath          string    `json:"url_path"`
			Category         string    `json:"category"`
			Subcategory      string    `json:"subcategory"`
			Purpose          string    `json:"purpose"`
			Version          string    `json:"version"`
			RelationshipType string    `json:"relationship_type"`
			CreatedAt        time.Time `json:"created_at"`
			UpdatedAt        time.Time `json:"updated_at"`
		}

		endpoints := []EndpointWithRelationship{}
		for rows.Next() {
			var ep EndpointWithRelationship
			if err := rows.Scan(&ep.ID, &ep.EndpointName, &ep.Description, &ep.HTTPMethod, &ep.URLPath,
				&ep.Category, &ep.Subcategory, &ep.Purpose, &ep.Version, &ep.RelationshipType,
				&ep.CreatedAt, &ep.UpdatedAt); err != nil {
				continue
			}
			endpoints = append(endpoints, ep)
		}

		// Get total count
		var total int
		countQuery := strings.Replace(query, "SELECT aec.id, aec.endpoint_name, aec.description, aec.http_method, aec.url_path, aec.category, aec.subcategory, aec.purpose, aec.version, aem.relationship_type, aec.created_at, aec.updated_at", "SELECT COUNT(*)", 1)
		countQuery = strings.Split(countQuery, "LIMIT")[0]
		db.QueryRow(countQuery, entityID, tenantID).Scan(&total)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"entity_id":  entityID,
			"data":       endpoints,
			"pagination": map[string]int{"page": page, "limit": limit, "total": total},
		})
	}
}

// handleGetDatasourceEndpoints retrieves all API endpoints for a specific datasource
func handleGetDatasourceEndpoints(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.URL.Query().Get("tenant_id")
		datasourceID := chi.URLParam(r, "datasource-id")

		if tenantID == "" {
			writeJSONError(w, http.StatusBadRequest, "tenant_id is required", "missing_tenant", "")
			return
		}

		query := `
			SELECT aec.id, aec.endpoint_name, aec.description, aec.http_method, aec.url_path,
			       aec.category, aec.subcategory, aec.purpose, aec.version, adm.relationship_type,
			       aec.created_at, aec.updated_at
			FROM api_endpoints_catalog aec
			JOIN api_endpoint_datasource_mappings adm ON aec.id = adm.api_endpoint_id
			WHERE adm.datasource_id = $1 AND adm.tenant_id = $2 AND aec.is_active = true
			ORDER BY aec.category, aec.endpoint_name
		`

		rows, err := db.Query(query, datasourceID, tenantID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to fetch endpoints", "db_error", err.Error())
			return
		}
		defer rows.Close()

		type EndpointWithRelationship struct {
			ID               string    `json:"id"`
			EndpointName     string    `json:"endpoint_name"`
			Description      string    `json:"description"`
			HTTPMethod       string    `json:"http_method"`
			URLPath          string    `json:"url_path"`
			Category         string    `json:"category"`
			Subcategory      string    `json:"subcategory"`
			Purpose          string    `json:"purpose"`
			Version          string    `json:"version"`
			RelationshipType string    `json:"relationship_type"`
			CreatedAt        time.Time `json:"created_at"`
			UpdatedAt        time.Time `json:"updated_at"`
		}

		endpoints := []EndpointWithRelationship{}
		for rows.Next() {
			var ep EndpointWithRelationship
			if err := rows.Scan(&ep.ID, &ep.EndpointName, &ep.Description, &ep.HTTPMethod, &ep.URLPath,
				&ep.Category, &ep.Subcategory, &ep.Purpose, &ep.Version, &ep.RelationshipType,
				&ep.CreatedAt, &ep.UpdatedAt); err != nil {
				continue
			}
			endpoints = append(endpoints, ep)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"datasource_id": datasourceID,
			"data":          endpoints,
		})
	}
}
