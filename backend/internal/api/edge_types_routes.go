package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/hondyman/uisce/backend/internal/handlers"
)

// EdgeType represents a catalog edge type
type EdgeType struct {
	ID                  string                 `json:"id"`
	TenantID            string                 `json:"tenant_id"`
	EdgeTypeName        string                 `json:"edge_type_name"`
	Description         *string                `json:"description"`
	IsActive            *bool                  `json:"is_active"`
	SubjectNodeTypeID   *string                `json:"subject_node_type_id,omitempty"`
	ObjectNodeTypeID    *string                `json:"object_node_type_id,omitempty"`
	SubjectNodeTypeName *string                `json:"subject_node_type_name,omitempty"` // Display name
	ObjectNodeTypeName  *string                `json:"object_node_type_name,omitempty"`  // Display name
	Type                *string                `json:"type,omitempty"`                   // "core" or "custom"
	IsDirected          bool                   `json:"is_directed"`
	Config              map[string]interface{} `json:"config,omitempty"`
	Properties          []NodeProperty         `json:"properties,omitempty"` // Reusing NodeProperty for edges
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
}

type edgeTypesHandler struct {
	securityDeps handlers.SecurityContextDeps
	db           *sql.DB
}

// RegisterEdgeTypesRoutes registers all edge type management routes
func RegisterEdgeTypesRoutes(r chi.Router, db *sql.DB, securityDeps handlers.SecurityContextDeps) {
	h := &edgeTypesHandler{
		securityDeps: securityDeps,
		db:           db,
	}
	r.Get("/edge-types", h.handleListEdgeTypes())
	r.Post("/edge-types", h.handleCreateEdgeType())
	r.Get("/edge-types/{id}", h.handleGetEdgeType())
	r.Patch("/edge-types/{id}", h.handleUpdateEdgeType())
	r.Delete("/edge-types/{id}", h.handleDeleteEdgeType())

	// Property management endpoints
	r.Get("/edge-types/{id}/properties", h.handleGetEdgeTypeProperties())
	r.Post("/edge-types/{id}/properties", h.handleAddEdgeTypeProperty())
	r.Patch("/edge-types/{id}/properties/{propName}", h.handleUpdateEdgeTypeProperty())
	r.Delete("/edge-types/{id}/properties/{propName}", h.handleDeleteEdgeTypeProperty())
}

// handleListEdgeTypes retrieves all edge types for a tenant
func (h *edgeTypesHandler) handleListEdgeTypes() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
		if err != nil {
			http.Error(w, "security context initialization failed: "+err.Error(), http.StatusUnauthorized)
			return
		}

		qParam := r.URL.Query().Get("q")

		var rows *sql.Rows
		if qParam == "" {
			query := `
				SELECT cet.id, cet.tenant_id, cet.edge_type_name, cet.description, cet.is_active,
					   cet.source_node_type_id, cet.target_node_type_id, cet.is_directed, cet.config, cet.created_at, cet.updated_at,
					   COALESCE(cnt_subj.catalog_type_name, '') as subject_node_name,
					   COALESCE(cnt_obj.catalog_type_name, '') as object_node_name,
					   CASE WHEN t.gold_copy = true THEN 'core' ELSE 'custom' END as type
				FROM catalog_edge_type cet
				LEFT JOIN catalog_node_type cnt_subj ON cnt_subj.id = cet.source_node_type_id
				LEFT JOIN catalog_node_type cnt_obj ON cnt_obj.id = cet.target_node_type_id
				JOIN tenants t ON cet.tenant_id::uuid = t.id
				WHERE cet.tenant_id::text = $1
				ORDER BY cet.edge_type_name
			`
			rows, err = h.db.Query(query, secCtx.TenantID)
		} else {
			search := "%" + qParam + "%"
			query := `
				SELECT cet.id, cet.tenant_id, cet.edge_type_name, cet.description, cet.is_active,
					   cet.source_node_type_id, cet.target_node_type_id, cet.is_directed, cet.config, cet.created_at, cet.updated_at,
					   COALESCE(cnt_subj.catalog_type_name, '') as subject_node_name,
					   COALESCE(cnt_obj.catalog_type_name, '') as object_node_name,
					   CASE WHEN t.gold_copy = true THEN 'core' ELSE 'custom' END as type
				FROM catalog_edge_type cet
				LEFT JOIN catalog_node_type cnt_subj ON cnt_subj.id = cet.source_node_type_id
				LEFT JOIN catalog_node_type cnt_obj ON cnt_obj.id = cet.target_node_type_id
				JOIN tenants t ON cet.tenant_id::uuid = t.id
				WHERE cet.tenant_id::text = $1
				  AND (cet.edge_type_name ILIKE $2 OR cet.description ILIKE $2)
				ORDER BY cet.edge_type_name
			`
			rows, err = h.db.Query(query, secCtx.TenantID, search)
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var edgeTypes []EdgeType
		for rows.Next() {
			var et EdgeType
			var configJSON []byte
			var tenantIDUUID sql.NullString
			var description sql.NullString
			var sourceNodeTypeID sql.NullString
			var targetNodeTypeID sql.NullString
			var subjectNodeName, objectNodeName, typeClassification string
			err := rows.Scan(&et.ID, &tenantIDUUID, &et.EdgeTypeName, &description,
				&et.IsActive, &sourceNodeTypeID, &targetNodeTypeID, &et.IsDirected, &configJSON, &et.CreatedAt, &et.UpdatedAt,
				&subjectNodeName, &objectNodeName, &typeClassification)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if tenantIDUUID.Valid {
				et.TenantID = tenantIDUUID.String
			}
			if description.Valid {
				et.Description = &description.String
			}
			if sourceNodeTypeID.Valid {
				et.SubjectNodeTypeID = &sourceNodeTypeID.String
			}
			if targetNodeTypeID.Valid {
				et.ObjectNodeTypeID = &targetNodeTypeID.String
			}

			if subjectNodeName != "" {
				et.SubjectNodeTypeName = &subjectNodeName
			}
			if objectNodeName != "" {
				et.ObjectNodeTypeName = &objectNodeName
			}
			if typeClassification != "" {
				et.Type = &typeClassification
			}

			config, err := unmarshalConfigJSON(configJSON)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			et.Config = config

			edgeTypes = append(edgeTypes, et)
		}

		if edgeTypes == nil {
			edgeTypes = []EdgeType{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(edgeTypes)
	}
}

// handleCreateEdgeType creates a new edge type
func (h *edgeTypesHandler) handleCreateEdgeType() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
		if err != nil {
			http.Error(w, "security context initialization failed: "+err.Error(), http.StatusUnauthorized)
			return
		}

		var et EdgeType
		if err := json.NewDecoder(r.Body).Decode(&et); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if et.EdgeTypeName == "" {
			http.Error(w, "edge_type_name is required", http.StatusBadRequest)
			return
		}

		et.TenantID = secCtx.TenantID

		if et.IsActive == nil {
			active := true
			et.IsActive = &active
		}

		if et.ID == "" {
			et.ID = uuid.New().String()
		}

		if et.Config == nil {
			et.Config = make(map[string]interface{})
		}

		var goldCopy bool
		err = h.db.QueryRow("SELECT COALESCE(gold_copy, false) FROM tenants WHERE id::text = $1", et.TenantID).Scan(&goldCopy)
		if err == nil {
			if goldCopy {
				core := "core"
				et.Type = &core
			} else {
				custom := "custom"
				et.Type = &custom
			}
		}

		if len(et.Properties) > 0 {
			et.Config["properties"] = et.Properties
		}

		configJSON, _ := json.Marshal(et.Config)

		query := `
			INSERT INTO catalog_edge_type
			(id, tenant_id, edge_type_name, description, is_active, source_node_type_id, target_node_type_id, is_directed, config, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
			RETURNING id, created_at, updated_at
		`

		var createErr error
		if createErr = h.db.QueryRow(query, et.ID, et.TenantID, et.EdgeTypeName, et.Description, et.IsActive,
			et.SubjectNodeTypeID, et.ObjectNodeTypeID, et.IsDirected, configJSON).Scan(&et.ID, &et.CreatedAt, &et.UpdatedAt); createErr != nil {
			http.Error(w, createErr.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(et)
	}
}

// handleGetEdgeType retrieves a single edge type
func (h *edgeTypesHandler) handleGetEdgeType() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
		if err != nil {
			http.Error(w, "security context initialization failed: "+err.Error(), http.StatusUnauthorized)
			return
		}
		id := chi.URLParam(r, "id")

		query := `
			SELECT cet.id, cet.tenant_id, cet.edge_type_name, cet.description, cet.is_active,
				   cet.source_node_type_id, cet.target_node_type_id, cet.is_directed, cet.config,
				   cet.created_at, cet.updated_at,
				   COALESCE(cnt_subj.catalog_type_name, '') as subject_node_name,
				   COALESCE(cnt_obj.catalog_type_name, '') as object_node_name,
				   CASE WHEN t.gold_copy = true THEN 'core' ELSE 'custom' END as type
			FROM catalog_edge_type cet
			LEFT JOIN catalog_node_type cnt_subj ON cnt_subj.id = cet.source_node_type_id
			LEFT JOIN catalog_node_type cnt_obj ON cnt_obj.id = cet.target_node_type_id
			JOIN tenants t ON cet.tenant_id::uuid = t.id
			WHERE cet.id = $1 AND cet.tenant_id::text = $2
		`

		var et EdgeType
		var configJSON sql.NullString
		var edgeTypeName string
		var description sql.NullString
		var isActive sql.NullBool
		var sourceNodeTypeID sql.NullString
		var targetNodeTypeID sql.NullString
		var isDirected sql.NullBool
		var tenantIDStr sql.NullString
		var subjectNodeName, objectNodeName, typeClassification string
		err = h.db.QueryRow(query, id, secCtx.TenantID).Scan(&et.ID, &tenantIDStr, &edgeTypeName,
			&description, &isActive, &sourceNodeTypeID, &targetNodeTypeID, &isDirected, &configJSON, &et.CreatedAt, &et.UpdatedAt,
			&subjectNodeName, &objectNodeName, &typeClassification)
		if err == sql.ErrNoRows {
			http.Error(w, "Edge type not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		et.EdgeTypeName = edgeTypeName
		if tenantIDStr.Valid {
			et.TenantID = tenantIDStr.String
		}
		if description.Valid {
			et.Description = &description.String
		}
		if isActive.Valid {
			et.IsActive = &isActive.Bool
		}
		if sourceNodeTypeID.Valid {
			et.SubjectNodeTypeID = &sourceNodeTypeID.String
		}
		if targetNodeTypeID.Valid {
			et.ObjectNodeTypeID = &targetNodeTypeID.String
		}
		if isDirected.Valid {
			et.IsDirected = isDirected.Bool
		}

		if subjectNodeName != "" {
			et.SubjectNodeTypeName = &subjectNodeName
		}
		if objectNodeName != "" {
			et.ObjectNodeTypeName = &objectNodeName
		}
		if typeClassification != "" {
			et.Type = &typeClassification
		}

		var config map[string]interface{}
		if configJSON.Valid {
			config, err = unmarshalConfigJSON([]byte(configJSON.String))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		if config == nil {
			config = make(map[string]interface{})
		}
		et.Config = config

		var propsArray []NodeProperty
		if props, ok := config["properties"].([]interface{}); ok {
			propsJSON, _ := json.Marshal(props)
			if err := json.Unmarshal(propsJSON, &propsArray); err == nil {
				et.Properties = propsArray
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(et)
	}
}

// handleUpdateEdgeType updates an existing edge type (partial update)
func (h *edgeTypesHandler) handleUpdateEdgeType() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
		if err != nil {
			http.Error(w, "security context initialization failed: "+err.Error(), http.StatusUnauthorized)
			return
		}
		id := chi.URLParam(r, "id")

		var et EdgeType
		if err := json.NewDecoder(r.Body).Decode(&et); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if et.Config == nil {
			et.Config = make(map[string]interface{})
		}

		if len(et.Properties) > 0 {
			et.Config["properties"] = et.Properties
		}

		configJSON, _ := json.Marshal(et.Config)

		var updates []string
		var args []interface{}
		argCount := 1

		if et.EdgeTypeName != "" {
			updates = append(updates, fmt.Sprintf("edge_type_name = $%d", argCount))
			args = append(args, et.EdgeTypeName)
			argCount++
		}

		if et.Description != nil || len(et.Properties) > 0 {
			updates = append(updates, fmt.Sprintf("description = $%d", argCount))
			args = append(args, et.Description)
			argCount++
		}

		updates = append(updates, fmt.Sprintf("config = $%d", argCount))
		args = append(args, configJSON)
		argCount++

		if et.IsActive != nil {
			updates = append(updates, fmt.Sprintf("is_active = $%d", argCount))
			args = append(args, et.IsActive)
			argCount++
		}

		updates = append(updates, "updated_at = NOW()")

		args = append(args, id)
		args = append(args, secCtx.TenantID)

		if len(updates) == 1 {
			http.Error(w, "No fields to update", http.StatusBadRequest)
			return
		}

		query := fmt.Sprintf(`
			UPDATE catalog_edge_type
			SET %s
			WHERE id = $%d AND tenant_id::text = $%d
			RETURNING id, edge_type_name, description, is_active, source_node_type_id,
					  target_node_type_id, is_directed, config, created_at, updated_at, tenant_id
		`, strings.Join(updates, ", "), argCount, argCount+1)

		var configOut []byte
		var isDirected bool
		if err := h.db.QueryRow(query, args...).Scan(
			&et.ID, &et.EdgeTypeName, &et.Description, &et.IsActive,
			&et.SubjectNodeTypeID, &et.ObjectNodeTypeID, &isDirected, &configOut,
			&et.CreatedAt, &et.UpdatedAt, &et.TenantID); err == sql.ErrNoRows {
			http.Error(w, "Edge type not found", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		et.IsDirected = isDirected

		if err := json.Unmarshal(configOut, &et.Config); err != nil {
			et.Config = make(map[string]interface{})
		}

		if propsData, ok := et.Config["properties"]; ok {
			if propsJSON, err := json.Marshal(propsData); err == nil {
				json.Unmarshal(propsJSON, &et.Properties)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(et)
	}
}

// handleDeleteEdgeType deletes an edge type
func (h *edgeTypesHandler) handleDeleteEdgeType() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
		if err != nil {
			http.Error(w, "security context initialization failed: "+err.Error(), http.StatusUnauthorized)
			return
		}
		id := chi.URLParam(r, "id")

		query := `DELETE FROM catalog_edge_type WHERE id = $1 AND tenant_id = $2`
		result, err := h.db.Exec(query, id, secCtx.TenantID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			http.Error(w, "Edge type not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// handleGetEdgeTypeProperties retrieves properties for an edge type
func (h *edgeTypesHandler) handleGetEdgeTypeProperties() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
		if err != nil {
			http.Error(w, "security context initialization failed: "+err.Error(), http.StatusUnauthorized)
			return
		}
		id := chi.URLParam(r, "id")

		query := `SELECT config FROM catalog_edge_type WHERE id = $1 AND tenant_id = $2`
		var configJSON []byte
		err = h.db.QueryRow(query, id, secCtx.TenantID).Scan(&configJSON)
		if err == sql.ErrNoRows {
			http.Error(w, "Edge type not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		properties := []NodeProperty{}

		if len(configJSON) > 0 {
			var config map[string]interface{}
			if err := json.Unmarshal(configJSON, &config); err == nil {
				if props, ok := config["properties"].([]interface{}); ok && len(props) > 0 {
					for _, p := range props {
						propJSON, _ := json.Marshal(p)
						var prop NodeProperty
						if err := json.Unmarshal(propJSON, &prop); err == nil {
							properties = append(properties, prop)
						}
					}
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(properties)
	}
}

// handleAddEdgeTypeProperty adds a property to an edge type
func (h *edgeTypesHandler) handleAddEdgeTypeProperty() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
		if err != nil {
			http.Error(w, "security context initialization failed: "+err.Error(), http.StatusUnauthorized)
			return
		}
		id := chi.URLParam(r, "id")

		var prop NodeProperty
		if err := json.NewDecoder(r.Body).Decode(&prop); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if prop.Name == "" || prop.DataType == "" {
			http.Error(w, "name and data_type are required", http.StatusBadRequest)
			return
		}

		query := `SELECT config FROM catalog_edge_type WHERE id = $1 AND tenant_id = $2`
		var configJSON []byte
		err = h.db.QueryRow(query, id, secCtx.TenantID).Scan(&configJSON)
		if err == sql.ErrNoRows {
			http.Error(w, "Edge type not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		config, err := unmarshalConfigJSON(configJSON)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var properties []NodeProperty
		if props, ok := config["properties"].([]interface{}); ok {
			for _, p := range props {
				propJSON, _ := json.Marshal(p)
				var existingProp NodeProperty
				if err := json.Unmarshal(propJSON, &existingProp); err == nil {
					properties = append(properties, existingProp)
				}
			}
		}
		properties = append(properties, prop)
		config["properties"] = properties

		newConfigJSON, err := json.Marshal(config)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		updateQuery := `UPDATE catalog_edge_type SET config = $1, updated_at = NOW() WHERE id = $2 AND tenant_id = $3`
		_, err = h.db.Exec(updateQuery, newConfigJSON, id, secCtx.TenantID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(prop)
	}
}

// handleUpdateEdgeTypeProperty updates a property in an edge type
func (h *edgeTypesHandler) handleUpdateEdgeTypeProperty() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
		if err != nil {
			http.Error(w, "security context initialization failed: "+err.Error(), http.StatusUnauthorized)
			return
		}
		id := chi.URLParam(r, "id")
		propName := chi.URLParam(r, "propName")

		var prop NodeProperty
		if err := json.NewDecoder(r.Body).Decode(&prop); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		query := `SELECT config FROM catalog_edge_type WHERE id = $1 AND tenant_id = $2`
		var configJSON []byte
		err = h.db.QueryRow(query, id, secCtx.TenantID).Scan(&configJSON)
		if err == sql.ErrNoRows {
			http.Error(w, "Edge type not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		config, err := unmarshalConfigJSON(configJSON)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var properties []NodeProperty
		found := false
		if props, ok := config["properties"].([]interface{}); ok {
			for _, p := range props {
				propJSON, _ := json.Marshal(p)
				var existingProp NodeProperty
				if err := json.Unmarshal(propJSON, &existingProp); err == nil {
					if existingProp.Name == propName {
						properties = append(properties, prop)
						found = true
					} else {
						properties = append(properties, existingProp)
					}
				}
			}
		}

		if !found {
			http.Error(w, "Property not found", http.StatusNotFound)
			return
		}

		config["properties"] = properties

		newConfigJSON, err := json.Marshal(config)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		updateQuery := `UPDATE catalog_edge_type SET config = $1, updated_at = NOW() WHERE id = $2 AND tenant_id = $3`
		_, err = h.db.Exec(updateQuery, newConfigJSON, id, secCtx.TenantID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(prop)
	}
}

// handleDeleteEdgeTypeProperty deletes a property from an edge type
func (h *edgeTypesHandler) handleDeleteEdgeTypeProperty() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
		if err != nil {
			http.Error(w, "security context initialization failed: "+err.Error(), http.StatusUnauthorized)
			return
		}
		id := chi.URLParam(r, "id")
		propName := chi.URLParam(r, "propName")

		query := `SELECT config FROM catalog_edge_type WHERE id = $1 AND tenant_id = $2`
		var configJSON []byte
		err = h.db.QueryRow(query, id, secCtx.TenantID).Scan(&configJSON)
		if err == sql.ErrNoRows {
			http.Error(w, "Edge type not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		config, err := unmarshalConfigJSON(configJSON)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if len(config) == 0 {
			http.Error(w, "No properties found", http.StatusNotFound)
			return
		}

		var properties []NodeProperty
		found := false
		if props, ok := config["properties"].([]interface{}); ok {
			for _, p := range props {
				propJSON, _ := json.Marshal(p)
				var existingProp NodeProperty
				if err := json.Unmarshal(propJSON, &existingProp); err == nil {
					if existingProp.Name != propName {
						properties = append(properties, existingProp)
					} else {
						found = true
					}
				}
			}
		}

		if !found {
			http.Error(w, "Property not found", http.StatusNotFound)
			return
		}

		config["properties"] = properties

		newConfigJSON, err := json.Marshal(config)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		updateQuery := `UPDATE catalog_edge_type SET config = $1, updated_at = NOW() WHERE id = $2 AND tenant_id = $3`
		_, err = h.db.Exec(updateQuery, newConfigJSON, id, secCtx.TenantID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
