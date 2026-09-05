package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/handlers"
	)

type NodeTypesHandler struct {
	db           *sql.DB
	securityDeps handlers.SecurityContextDeps
}

func NewNodeTypesHandler(db *sql.DB, securityDeps handlers.SecurityContextDeps) *NodeTypesHandler {
	return &NodeTypesHandler{
		db:           db,
		securityDeps: securityDeps,
	}
}

// NodeType represents a catalog node type
type NodeType struct {
	ID              string                 `json:"id"`
	TenantID        string                 `json:"tenant_id"`
	CatalogTypeName string                 `json:"catalog_type_name"`
	Description     *string                `json:"description"`
	IsActive        *bool                  `json:"is_active"`
	Type            string                 `json:"type,omitempty"` // "core" if from gold_copy tenant, "custom" otherwise
	ParentTypeID    *string                `json:"parent_type_id,omitempty"`
	Config          map[string]interface{} `json:"config,omitempty"`
	Properties      []NodeProperty         `json:"properties,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// NodeProperty represents a property configuration for a node type
type NodeProperty struct {
	Name         string                 `json:"name"`
	Label        string                 `json:"label"`
	DataType     string                 `json:"data_type"` // string, integer, boolean, date, etc.
	Nullable     bool                   `json:"nullable"`
	DefaultValue interface{}            `json:"default_value,omitempty"`
	InputType    string                 `json:"input_type"`       // text, select, checkbox, date-picker, textarea, etc.
	Format       string                 `json:"format,omitempty"` // display format/validation
	Validation   map[string]interface{} `json:"validation,omitempty"`
	Options      []string               `json:"options,omitempty"` // for select/dropdown inputs
	Order        int                    `json:"order"`
	LookupID     *string                `json:"lookup_id,omitempty"`
	CascadeFrom  *string                `json:"cascade_from,omitempty"`
}

// unmarshalConfigJSON unmarshals a JSONB config field, handling both array and object formats
// Arrays are normalized to empty objects since config should be an object
func unmarshalConfigJSON(configJSON []byte) (map[string]interface{}, error) {
	if len(configJSON) == 0 {
		return make(map[string]interface{}), nil
	}

	var rawConfig interface{}
	if err := json.Unmarshal(configJSON, &rawConfig); err != nil {
		return nil, err
	}

	// If it's an array (legacy data), convert to empty object
	if _, isArray := rawConfig.([]interface{}); isArray {
		return make(map[string]interface{}), nil
	}

	// If it's an object, return it
	if configMap, isMap := rawConfig.(map[string]interface{}); isMap {
		return configMap, nil
	}

	// For any other type, return empty object
	return make(map[string]interface{}), nil
}

func RegisterNodeTypesRoutes(r chi.Router, db *sql.DB, securityDeps handlers.SecurityContextDeps) {
	h := NewNodeTypesHandler(db, securityDeps)
	r.Get("/node-types", h.handleListNodeTypes)
	r.Post("/node-types", h.handleCreateNodeType)
	r.Get("/node-types/{id}", h.handleGetNodeType)
	r.Patch("/node-types/{id}", h.handleUpdateNodeType)
	r.Delete("/node-types/{id}", h.handleDeleteNodeType)

	r.Get("/node-types/{id}/properties", h.handleGetNodeTypeProperties)
	r.Post("/node-types/{id}/properties", h.handleAddNodeTypeProperty)
	r.Patch("/node-types/{id}/properties/{propName}", h.handleUpdateNodeTypeProperty)
	r.Delete("/node-types/{id}/properties/{propName}", h.handleDeleteNodeTypeProperty)

	r.Get("/node-types/{id}/nodes", h.handleGetNodesForType)
}

func (h *NodeTypesHandler) handleListNodeTypes(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	tenantID := secCtx.TenantID

	qParam := r.URL.Query().Get("q")

	var rows *sql.Rows
	if qParam == "" {
		query := `
		     SELECT cnt.id, cnt.tenant_id, cnt.catalog_type_name, cnt.description, cnt.is_active,
				  cnt.parent_type_id, cnt.config, cnt.created_at, cnt.updated_at,
				  COALESCE(t.gold_copy, false) as is_core
				FROM catalog_node_type cnt
				JOIN tenants t ON cnt.tenant_id::uuid = t.id
				WHERE cnt.tenant_id = $1 OR cnt.tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1)
				ORDER BY CASE WHEN cnt.tenant_id = $1 THEN 0 ELSE 1 END, cnt.catalog_type_name
			`
		rows, err = h.db.Query(query, tenantID)
	} else {
		search := "%" + qParam + "%"
		query := `
				SELECT cnt.id, cnt.tenant_id, cnt.catalog_type_name, cnt.description, cnt.is_active,
					   cnt.parent_type_id, cnt.config, cnt.created_at, cnt.updated_at,
					   COALESCE(t.gold_copy, false) as is_core
				FROM catalog_node_type cnt
				JOIN tenants t ON cnt.tenant_id::uuid = t.id
				WHERE (cnt.tenant_id = $1 OR cnt.tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1))
				  AND (cnt.catalog_type_name ILIKE $2 OR COALESCE(cnt.description, '') ILIKE $2)
				ORDER BY CASE WHEN cnt.tenant_id = $1 THEN 0 ELSE 1 END, cnt.catalog_type_name
			`
		rows, err = h.db.Query(query, tenantID, search)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var nodeTypes []NodeType
	for rows.Next() {
		var nt NodeType
		var configJSON []byte
		var isCore bool
		err := rows.Scan(&nt.ID, &nt.TenantID, &nt.CatalogTypeName, &nt.Description,
			&nt.IsActive, &nt.ParentTypeID, &configJSON, &nt.CreatedAt, &nt.UpdatedAt, &isCore)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if isCore {
			nt.Type = "core"
		} else {
			nt.Type = "custom"
		}

		config, err := unmarshalConfigJSON(configJSON)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		nt.Config = config

		if props, ok := config["properties"].([]interface{}); ok {
			var propsArray []NodeProperty
			propsJSON, _ := json.Marshal(props)
			if err := json.Unmarshal(propsJSON, &propsArray); err == nil {
				nt.Properties = propsArray
			}
		}

		nodeTypes = append(nodeTypes, nt)
	}

	if nodeTypes == nil {
		nodeTypes = []NodeType{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nodeTypes)
}

func (h *NodeTypesHandler) handleCreateNodeType(w http.ResponseWriter, r *http.Request) {
	var nt NodeType
	if err := json.NewDecoder(r.Body).Decode(&nt); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if nt.CatalogTypeName == "" {
		http.Error(w, "catalog_type_name is required", http.StatusBadRequest)
		return
	}

	if nt.TenantID == "" {
		nt.TenantID, _ = TenantIDFromRequest(r)
	}
	if nt.TenantID == "" {
		http.Error(w, "tenant_id is required", http.StatusBadRequest)
		return
	}

	if nt.IsActive == nil {
		active := true
		nt.IsActive = &active
	}

	if nt.ID == "" {
		nt.ID = uuid.New().String()
	}

	if nt.Config == nil {
		nt.Config = make(map[string]interface{})
	}

	var goldCopy bool
	err := h.db.QueryRow("SELECT COALESCE(gold_copy, false) FROM tenants WHERE id::text = $1", nt.TenantID).Scan(&goldCopy)
	if err == nil {
		nt.Config["core"] = goldCopy
	}

	if len(nt.Properties) > 0 {
		nt.Config["properties"] = nt.Properties
	}

	configJSON, err := json.Marshal(nt.Config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	query := `
			INSERT INTO catalog_node_type
			(id, tenant_id, catalog_type_name, description, is_active, parent_type_id, config, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
			RETURNING id, created_at, updated_at
		`

	err = h.db.QueryRow(query, nt.ID, nt.TenantID, nt.CatalogTypeName, nt.Description,
		nt.IsActive, nt.ParentTypeID, configJSON).Scan(&nt.ID, &nt.CreatedAt, &nt.UpdatedAt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(nt)
}

func (h *NodeTypesHandler) handleGetNodeType(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	tenantID := secCtx.TenantID

	query := `
			SELECT cond.id, cond.tenant_id, cond.catalog_type_name, cond.description, cond.is_active,
				   cond.parent_type_id, cond.config, cond.created_at, cond.updated_at,
				   COALESCE(t.gold_copy, false) as is_core
			FROM catalog_node_type cond
			LEFT JOIN tenants t ON cond.tenant_id = t.id
			WHERE cond.id = $1 AND cond.tenant_id = $2
		`

	var nt NodeType
	var configJSON []byte
	var isCore bool
	err = h.db.QueryRow(query, id, tenantID).Scan(&nt.ID, &nt.TenantID, &nt.CatalogTypeName,
		&nt.Description, &nt.IsActive, &nt.ParentTypeID, &configJSON, &nt.CreatedAt, &nt.UpdatedAt, &isCore)
	if err == sql.ErrNoRows {
		http.Error(w, "Node type not found", http.StatusNotFound)
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
	nt.Config = config
	if isCore {
		nt.Type = "core"
	} else {
		nt.Type = "custom"
	}

	if props, ok := config["properties"].([]interface{}); ok {
		var propsArray []NodeProperty
		propsJSON, _ := json.Marshal(props)
		if err := json.Unmarshal(propsJSON, &propsArray); err == nil {
			nt.Properties = propsArray
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nt)
}

func (h *NodeTypesHandler) handleUpdateNodeType(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	tenantID := secCtx.TenantID

	var nt NodeType
	if err := json.NewDecoder(r.Body).Decode(&nt); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if nt.Config == nil {
		nt.Config = make(map[string]interface{})
	}

	if len(nt.Properties) > 0 {
		nt.Config["properties"] = nt.Properties
	}

	configJSON, err := json.Marshal(nt.Config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	query := `
			UPDATE catalog_node_type
			SET catalog_type_name = $1, description = $2, is_active = $3,
				parent_type_id = $4, config = $5, updated_at = NOW()
			WHERE id = $6 AND tenant_id = $7
			RETURNING updated_at
		`

	err = h.db.QueryRow(query, nt.CatalogTypeName, nt.Description, nt.IsActive,
		nt.ParentTypeID, configJSON, id, tenantID).Scan(&nt.UpdatedAt)
	if err == sql.ErrNoRows {
		http.Error(w, "Node type not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	nt.ID = id
	nt.TenantID = tenantID

	coreQuery := `SELECT COALESCE(t.gold_copy, false) FROM tenants t WHERE t.id = $1`
	var isCore bool
	if err := h.db.QueryRow(coreQuery, tenantID).Scan(&isCore); err == nil {
		if isCore {
			nt.Type = "core"
		} else {
			nt.Type = "custom"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nt)
}

func (h *NodeTypesHandler) handleDeleteNodeType(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	tenantID := secCtx.TenantID

	query := `DELETE FROM catalog_node_type WHERE id = $1 AND tenant_id = $2`
	result, err := h.db.Exec(query, id, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		http.Error(w, "Node type not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *NodeTypesHandler) handleGetNodeTypeProperties(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	tenantID := secCtx.TenantID

	query := `SELECT config FROM catalog_node_type WHERE id = $1 AND tenant_id = $2`
	var configJSON []byte
	err = h.db.QueryRow(query, id, tenantID).Scan(&configJSON)
	if err == sql.ErrNoRows {
		http.Error(w, "Node type not found", http.StatusNotFound)
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

func (h *NodeTypesHandler) handleAddNodeTypeProperty(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	tenantID := secCtx.TenantID

	var prop NodeProperty
	if err := json.NewDecoder(r.Body).Decode(&prop); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if prop.Name == "" || prop.DataType == "" {
		http.Error(w, "name and data_type are required", http.StatusBadRequest)
		return
	}

	query := `SELECT config FROM catalog_node_type WHERE id = $1 AND tenant_id = $2`
	var configJSON []byte
	err = h.db.QueryRow(query, id, tenantID).Scan(&configJSON)
	if err == sql.ErrNoRows {
		http.Error(w, "Node type not found", http.StatusNotFound)
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

	updateQuery := `UPDATE catalog_node_type SET config = $1, updated_at = NOW() WHERE id = $2 AND tenant_id = $3`
	_, err = h.db.Exec(updateQuery, newConfigJSON, id, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(prop)
}

func (h *NodeTypesHandler) handleUpdateNodeTypeProperty(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	propName := chi.URLParam(r, "propName")
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	tenantID := secCtx.TenantID

	var prop NodeProperty
	if err := json.NewDecoder(r.Body).Decode(&prop); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := `SELECT config FROM catalog_node_type WHERE id = $1 AND tenant_id = $2`
	var configJSON []byte
	err = h.db.QueryRow(query, id, tenantID).Scan(&configJSON)
	if err == sql.ErrNoRows {
		http.Error(w, "Node type not found", http.StatusNotFound)
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

	updateQuery := `UPDATE catalog_node_type SET config = $1, updated_at = NOW() WHERE id = $2 AND tenant_id = $3`
	_, err = h.db.Exec(updateQuery, newConfigJSON, id, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prop)
}

func (h *NodeTypesHandler) handleDeleteNodeTypeProperty(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	propName := chi.URLParam(r, "propName")
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	tenantID := secCtx.TenantID

	query := `SELECT config FROM catalog_node_type WHERE id = $1 AND tenant_id = $2`
	var configJSON []byte
	err = h.db.QueryRow(query, id, tenantID).Scan(&configJSON)
	if err == sql.ErrNoRows {
		http.Error(w, "Node type not found", http.StatusNotFound)
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

	updateQuery := `UPDATE catalog_node_type SET config = $1, updated_at = NOW() WHERE id = $2 AND tenant_id = $3`
	_, err = h.db.Exec(updateQuery, newConfigJSON, id, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *NodeTypesHandler) handleGetNodesForType(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	tenantID := secCtx.TenantID
	tenantDatasourceID := secCtx.DatasourceID

	query := `
			SELECT id, node_name, description, node_type_id, tenant_id, tenant_datasource_id, properties, config, created_at, updated_at
			FROM catalog_node
			WHERE node_type_id = $1 AND (tenant_id = $2 OR tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1))
		`
	args := []interface{}{id, tenantID}

	if tenantDatasourceID != "" {
		query += " AND (tenant_datasource_id = $4 OR tenant_datasource_id IS NULL)"
		args = append(args, tenantDatasourceID)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var nodes []map[string]interface{}
	for rows.Next() {
		var n struct {
			ID                 string
			NodeName           *string
			Description        *string
			NodeTypeID         *string
			TenantID           *string
			TenantDatasourceID *string
			Properties         []byte
			Config             []byte
			CreatedAt          time.Time
			UpdatedAt          time.Time
		}

		err := rows.Scan(&n.ID, &n.NodeName, &n.Description, &n.NodeTypeID, &n.TenantID, &n.TenantDatasourceID, &n.Properties, &n.Config, &n.CreatedAt, &n.UpdatedAt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var properties interface{}
		if len(n.Properties) > 0 {
			json.Unmarshal(n.Properties, &properties)
		}
		var config interface{}
		if len(n.Config) > 0 {
			json.Unmarshal(n.Config, &config)
		}

		node := map[string]interface{}{
			"id":                   n.ID,
			"node_name":            n.NodeName,
			"description":          n.Description,
			"node_type_id":         n.NodeTypeID,
			"tenant_id":            n.TenantID,
			"tenant_datasource_id": n.TenantDatasourceID,
			"properties":           properties,
			"config":               config,
			"created_at":           n.CreatedAt,
			"updated_at":           n.UpdatedAt,
		}
		nodes = append(nodes, node)
	}

	if nodes == nil {
		nodes = []map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nodes)
}
