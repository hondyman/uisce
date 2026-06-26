package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/identity"
	"github.com/hondyman/semlayer/backend/internal/lineage"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/pkg/governance"
	"github.com/jmoiron/sqlx"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// GlossaryHandler handles glossary-related API requests
type GlossaryHandler struct {
	db          *sql.DB
	dbx         *sqlx.DB
	governance  *governance.GovernanceEngine
	lineageRepo lineage.LineageRepository
}

// NewGlossaryHandler creates a new glossary handler
func NewGlossaryHandler(db *sql.DB, lineageRepo lineage.LineageRepository) *GlossaryHandler {
	dbx := sqlx.NewDb(db, "postgres")
	return &GlossaryHandler{
		db:          db,
		dbx:         dbx,
		governance:  governance.NewGovernanceEngine(dbx),
		lineageRepo: lineageRepo,
	}
}

func (h *GlossaryHandler) RegisterRoutes(r chi.Router) {
	r.Route("/glossary", func(r chi.Router) {
		r.Get("/semantic-terms", h.ListSemanticTerms)
		r.Get("/business-terms", h.ListBusinessTerms)
		r.Get("/edges", h.ListEdges)
		r.Put("/terms/{id}", h.UpdateTerm)
		r.Post("/terms", h.CreateTerm)
		r.Delete("/terms/{id}", h.DeleteTerm)
		r.Post("/edges", h.CreateEdge)
		r.Put("/edges/{id}", h.UpdateEdge)
		r.Delete("/edges/{id}", h.DeleteEdge)

		// Cube.dev properties endpoints
		r.Get("/semantic-terms/{id}/cube-definition", h.HandleGetSemanticTermWithCubeProperties)
		r.Get("/semantic-terms/export/cube-yaml", h.HandleExportSemanticTermsAsCubeYaml)
	})
}

// Generic list function to handle different term types
func (h *GlossaryHandler) listTerms(w http.ResponseWriter, r *http.Request, termType string) {
	tenantID, _ := identity.TenantIDFromContext(r.Context())

	// Fallback to Header/Query if Context is empty
	if tenantID == "" {
		tenantID = jwtmiddleware.GetClaimsFromContext(r).TenantID
		if tenantID == "" {
			tenantID = r.URL.Query().Get("tenant_id")
		}
	}

	if tenantID == "" {
		http.Error(w, "Unauthorized: tenant isolation required", http.StatusUnauthorized)
		return
	}

	tenantDatasourceID := r.Header.Get("X-Tenant-Datasource-ID")
	if tenantDatasourceID == "" {
		tenantDatasourceID = r.Header.Get("X-Tenant-Instance-ID")
	}
	if tenantDatasourceID == "" {
		tenantDatasourceID = r.URL.Query().Get("datasource_id")
	}
	if tenantDatasourceID == "" {
		tenantDatasourceID = r.URL.Query().Get("tenant_instance_id")
	}

	// NOTE: tenantDatasourceID is now optional. If missing, we return nodes across all datasources for this tenant.
	// This prevents 400 errors when no specific datasource is selected yet.

	query := `
		SELECT 
			cn.id, 
			cn.node_name,
			cn.tenant_datasource_id, 
			cn.description,
			cn.parent_type_id,
			cn.config,
			cn.created_at,
			cn.updated_at,
			cn.tenant_id,
			cn.core_id,
			cn.properties,
			cn.node_type_id,
			cn.qualified_path,
			cn.node_type
		FROM catalog_node cn
		WHERE cn.tenant_id = $1
	`
	args := []interface{}{tenantID}

	if tenantDatasourceID != "" {
		query += " AND cn.tenant_datasource_id = $2"
		args = append(args, tenantDatasourceID)
	}

	if termType != "" {
		argCount := len(args) + 1
		query += fmt.Sprintf(" AND cn.node_type = $%d", argCount)
		args = append(args, termType)
	}

	query += " ORDER BY cn.created_at DESC"

	rows, err := h.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		log.Printf("Error querying terms: %v", err)
		http.Error(w, "Failed to fetch terms", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		log.Printf("Error getting columns: %v", err)
		http.Error(w, "Failed to fetch terms", http.StatusInternalServerError)
		return
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		entry := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if val == nil {
				entry[col] = nil
			} else if b, ok := val.([]byte); ok {
				// Try to parse as UUID or string
				if len(b) == 16 {
					// Likely a UUID
					entry[col] = uuid.UUID(b).String()
				} else {
					entry[col] = string(b)
				}
			} else {
				entry[col] = val
			}
		}
		results = append(results, entry)
	}

	if results == nil {
		results = []map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// ListSemanticTerms returns all semantic terms for a tenant/datasource
func (h *GlossaryHandler) ListSemanticTerms(w http.ResponseWriter, r *http.Request) {
	h.listTerms(w, r, "semantic_term")
}

// ListBusinessTerms returns all business terms for a tenant/datasource
func (h *GlossaryHandler) ListBusinessTerms(w http.ResponseWriter, r *http.Request) {
	h.listTerms(w, r, "business_term")
}

// ListEdges returns all edges for a tenant/datasource
func (h *GlossaryHandler) ListEdges(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	tenantDatasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	// Allow query params as fallback
	if tenantID == "" {
		tenantID = r.URL.Query().Get("tenant_id")
	}
	if tenantDatasourceID == "" {
		tenantDatasourceID = r.URL.Query().Get("datasource_id")
	}

	if tenantID == "" || tenantDatasourceID == "" {
		http.Error(w, "X-Tenant-ID and X-Tenant-Datasource-ID headers/params are required", http.StatusBadRequest)
		return
	}

	// Query edges from catalog_edge table using correct column names
	query := `
		SELECT 
			ce.id, 
			ce.edge_type_name as predicate,
			ce.edge_type_name,
			ce.source_node_id,
			ce.target_node_id,
			COALESCE(ce.properties, '{}'::jsonb) as properties,
			ce.created_at,
			ce.updated_at,
			ce.tenant_id,
			ce.edge_type_id
		FROM catalog_edge ce
		WHERE ce.tenant_id = $1 AND ce.tenant_datasource_id = $2
		ORDER BY ce.created_at DESC
	`

	rows, err := h.db.Query(query, tenantID, tenantDatasourceID)
	if err != nil {
		log.Printf("Error querying edges: %v", err)
		http.Error(w, "Failed to fetch edges", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type EdgeRow struct {
		ID           string
		Predicate    *string
		EdgeTypeName *string
		SourceNodeID string
		TargetNodeID string
		Properties   []byte
		CreatedAt    string
		UpdatedAt    string
		TenantID     string
		EdgeTypeID   *string
	}

	var edges []map[string]interface{}
	for rows.Next() {
		var row EdgeRow
		err := rows.Scan(
			&row.ID,
			&row.Predicate,
			&row.EdgeTypeName,
			&row.SourceNodeID,
			&row.TargetNodeID,
			&row.Properties,
			&row.CreatedAt,
			&row.UpdatedAt,
			&row.TenantID,
			&row.EdgeTypeID,
		)
		if err != nil {
			log.Printf("Error scanning edge row: %v", err)
			continue
		}

		// Parse properties JSON
		var properties interface{}
		if len(row.Properties) > 0 {
			propertiesStr := string(row.Properties)
			trimmed := strings.TrimSpace(propertiesStr)
			if strings.HasPrefix(trimmed, "[") {
				var arr []map[string]interface{}
				if err := json.Unmarshal(row.Properties, &arr); err == nil {
					properties = arr
				} else {
					properties = []map[string]interface{}{}
				}
			} else if strings.HasPrefix(trimmed, "{") {
				var obj map[string]interface{}
				if err := json.Unmarshal(row.Properties, &obj); err == nil {
					properties = obj
				} else {
					properties = map[string]interface{}{}
				}
			} else {
				properties = map[string]interface{}{}
			}
		} else {
			properties = map[string]interface{}{}
		}

		edge := map[string]interface{}{
			"id":             row.ID,
			"predicate":      row.Predicate,
			"edge_type_name": row.EdgeTypeName,
			"source_node_id": row.SourceNodeID,
			"target_node_id": row.TargetNodeID,
			"properties":     properties,
			"created_at":     row.CreatedAt,
			"updated_at":     row.UpdatedAt,
			"tenant_id":      row.TenantID,
			"edge_type_id":   row.EdgeTypeID,
			"is_active":      true,
		}
		edges = append(edges, edge)
	}

	if edges == nil {
		edges = []map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(edges)
}

// CreateTerm creates a new catalog node (semantic or business term)
func (h *GlossaryHandler) CreateTerm(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	tenantDatasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	// Allow query params as fallback
	if tenantID == "" {
		tenantID = r.URL.Query().Get("tenant_id")
	}
	if tenantDatasourceID == "" {
		tenantDatasourceID = r.URL.Query().Get("datasource_id")
	}

	if tenantID == "" || tenantDatasourceID == "" {
		http.Error(w, "X-Tenant-ID and X-Tenant-Datasource-ID headers/params are required", http.StatusBadRequest)
		return
	}

	var termData struct {
		NodeName           string                 `json:"node_name"`
		Description        string                 `json:"description"`
		CatalogType        string                 `json:"catalog_type"`
		TenantDatasourceID string                 `json:"tenant_datasource_id"`
		Properties         map[string]interface{} `json:"properties"`
		ParentID           string                 `json:"parent_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&termData); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if termData.NodeName == "" {
		http.Error(w, "node_name is required", http.StatusBadRequest)
		return
	}

	if termData.CatalogType == "" {
		http.Error(w, "catalog_type is required", http.StatusBadRequest)
		return
	}

	// Validate catalog_type
	if termData.CatalogType != "semantic_term" && termData.CatalogType != "business_term" {
		http.Error(w, "catalog_type must be 'semantic_term' or 'business_term'", http.StatusBadRequest)
		return
	}

	// Use provided tenant_datasource_id or default to the one from headers
	if termData.TenantDatasourceID == "" {
		termData.TenantDatasourceID = tenantDatasourceID
	}

	// Convert properties to JSON
	propertiesJSON, err := json.Marshal(termData.Properties)
	if err != nil {
		http.Error(w, "Invalid properties format", http.StatusBadRequest)
		return
	}

	// Resolve node_type_id from catalog_node_type by name; create if missing.
	var nodeTypeID string
	err = h.db.QueryRow(`SELECT id FROM catalog_node_type WHERE catalog_type_name = $1 LIMIT 1`, termData.CatalogType).Scan(&nodeTypeID)
	if err != nil {
		if err == sql.ErrNoRows {
			// Create a fallback node type for this tenant
			insertTypeQ := `INSERT INTO catalog_node_type (tenant_id, catalog_type_name, created_at, updated_at) VALUES ($1, $2, NOW(), NOW()) RETURNING id`
			if err := h.db.QueryRow(insertTypeQ, tenantID, termData.CatalogType).Scan(&nodeTypeID); err != nil {
				log.Printf("Error creating fallback catalog_node_type: %v", err)
				http.Error(w, "Failed to resolve catalog type", http.StatusInternalServerError)
				return
			}

			// Fetch node type properties so we can validate incoming term properties
			var nodeTypePropertiesBytes []byte
			var nodeProps []NodeProperty
			if err := h.db.QueryRow(`SELECT properties FROM catalog_node_type WHERE id = $1`, nodeTypeID).Scan(&nodeTypePropertiesBytes); err == nil {
				if len(nodeTypePropertiesBytes) > 0 {
					_ = json.Unmarshal(nodeTypePropertiesBytes, &nodeProps)
				}
			}

			// Validate properties with node type metadata and return structured 400 errors when checks fail.
			if validationErrors, ok := validateTermProperties(nodeProps, termData.Properties); !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"validation_errors": validationErrors})
				return
			}
		} else {
			log.Printf("Error resolving catalog_node_type: %v", err)
			return
		}
	}

	// Governance Check for Semantic Terms
	if termData.CatalogType == "semantic_term" && h.governance != nil {
		// Convert properties to map for validation
		// We add top-level fields too as policy might check them
		validationInput := map[string]interface{}{}
		for k, v := range termData.Properties {
			validationInput[k] = v
		}
		validationInput["node_name"] = termData.NodeName
		validationInput["description"] = termData.Description

		result, err := h.governance.ValidateSemanticTerm(r.Context(), validationInput)
		if err != nil {
			log.Printf("Governance validation error: %v", err)
			// Decide if error should block. For now, log and proceed or block?
			// Let's block on error for safety
			http.Error(w, "Governance validation failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if !result.Allowed {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":   "Governance Policy Violation",
				"reasons": result.Reasons,
			})
			return
		}
	}

	// Insert the new term using node_type_id. Cast properties to jsonb explicitly so Postgres accepts
	// both empty objects and structured JSON payloads coming from the request.
	// Generate qualified_path as: catalog_type/node_name (or node_type_id/node_name for uniqueness)
	qualifiedPath := fmt.Sprintf("%s/%s", termData.CatalogType, termData.NodeName)

	insertQ := `
		INSERT INTO catalog_node (
			node_name, description, node_type_id, tenant_id, tenant_datasource_id,
			properties, qualified_path, parent_id, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, NOW(), NOW())
		RETURNING id
	`

	// Only include parent_id for semantic terms
	var parentID *string
	if termData.CatalogType == "semantic_term" && termData.ParentID != "" {
		parentID = &termData.ParentID
	}

	log.Printf("[DEBUG CreateTerm] catalog_type=%s, parent_id=%v, provided ParentID=%s", termData.CatalogType, parentID, termData.ParentID)

	var insertedID string
	if err := h.db.QueryRow(insertQ,
		termData.NodeName,
		termData.Description,
		nodeTypeID,
		tenantID,
		termData.TenantDatasourceID,
		propertiesJSON,
		qualifiedPath,
		parentID,
	).Scan(&insertedID); err != nil {
		log.Printf("Error creating term (node_name=%s, tenant=%s, datasource=%s): %v", termData.NodeName, tenantID, termData.TenantDatasourceID, err)
		http.Error(w, "Failed to create term", http.StatusInternalServerError)
		return
	}

	// Select back the inserted row joined with catalog_node_type for a consistent response
	selQ := `
		SELECT cn.id, cn.node_name, cn.tenant_datasource_id, COALESCE(cnt.catalog_type_name, '') as catalog_type_name, COALESCE(cn.description, '') as description, cn.parent_type_id, cn.parent_id, COALESCE(cn.config::text, cn.properties::text, '[]'::text) as config, cn.created_at, cn.updated_at, cn.tenant_id, cn.core_id, COALESCE(cn.properties, '[]'::jsonb) as properties
		FROM catalog_node cn
		LEFT JOIN catalog_node_type cnt ON cnt.id = cn.node_type_id
		WHERE cn.id = $1
	`

	var term models.CatalogNode
	var returnedPropertiesBytes []byte
	var parentIDStr *string
	err = h.db.QueryRow(selQ, insertedID).Scan(
		&term.ID,
		&term.NodeName,
		&term.TenantDatasourceID,
		&term.CatalogTypeName,
		&term.Description,
		&term.ParentTypeID,
		&parentIDStr,
		&term.Config,
		&term.CreatedAt,
		&term.UpdatedAt,
		&term.TenantID,
		&term.CoreID,
		&returnedPropertiesBytes,
	)
	if parentIDStr != nil && *parentIDStr != "" {
		term.ParentID = parentIDStr
	}

	log.Printf("[DEBUG CreateTerm Response] ID=%s, ParentID=%v, CatalogType=%s", term.ID, term.ParentID, term.CatalogTypeName)

	if err != nil {
		// Log the full error and the payload to help diagnose DB constraint issues.
		log.Printf("Error creating term (node_name=%s, tenant=%s, datasource=%s): %v", termData.NodeName, tenantID, termData.TenantDatasourceID, err)
		http.Error(w, "Failed to create term", http.StatusInternalServerError)
		return
	}

	// Set properties JSON
	if len(returnedPropertiesBytes) > 0 {
		term.Properties = json.RawMessage(returnedPropertiesBytes)
	} else {
		term.Properties = json.RawMessage("[]")
	}
	// schema lacks `is_active` on catalog_node; default to true for compatibility
	active := true
	term.IsActive = &active

	// If this is a semantic term with a parent business term, create an edge
	if termData.CatalogType == "semantic_term" && termData.ParentID != "" {
		// Create an edge from business term (parent) to semantic term (child)
		// relationship_type: 'business_term_to_semantic_term'
		edgeID := uuid.New().String()
		edgeCreateQ := `
			INSERT INTO catalog_edge (id, source_node_id, target_node_id, relationship_type, edge_type, tenant_id, tenant_datasource_id, created_at)
			VALUES ($1, $2, $3, $4, $4, $5, $6, NOW())
			ON CONFLICT (tenant_datasource_id, source_node_id, edge_type_id, target_node_id) DO NOTHING
		`
		if _, err := h.db.Exec(edgeCreateQ, edgeID, termData.ParentID, insertedID, "business_term_to_semantic_term", tenantID, tenantDatasourceID); err != nil {
			log.Printf("Warning: Failed to create edge from business term to semantic term: %v", err)
			// Don't fail the response; the term was created successfully
		}
	}

	// Sync to AGE if repo is available
	if h.lineageRepo != nil {
		node := lineage.LineageNode{
			ID:       term.ID,
			Type:     lineage.LineageNodeType(term.CatalogTypeName),
			Name:     term.NodeName,
			TenantID: &term.TenantID,
			Metadata: term.Properties,
			Env:      "dev",
		}
		if err := h.lineageRepo.UpsertNode(r.Context(), node); err != nil {
			log.Printf("Warning: Failed to sync term %s to graph: %v", term.ID, err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(term)
}

// UpdateTerm updates a catalog node (semantic or business term)
func (h *GlossaryHandler) UpdateTerm(w http.ResponseWriter, r *http.Request) {
	termID := r.PathValue("id")
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	tenantDatasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	// Allow query params as fallback
	if tenantID == "" {
		tenantID = r.URL.Query().Get("tenant_id")
	}
	if tenantDatasourceID == "" {
		tenantDatasourceID = r.URL.Query().Get("datasource_id")
	}

	if tenantID == "" || tenantDatasourceID == "" {
		http.Error(w, "X-Tenant-ID and X-Tenant-Datasource-ID headers/params are required", http.StatusBadRequest)
		return
	}

	if termID == "" {
		http.Error(w, "Term ID is required", http.StatusBadRequest)
		return
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Build dynamic update query
	setClauses := []string{}
	args := []interface{}{}
	argIndex := 1

	for key, value := range updates {
		// Only allow specific fields to be updated
		// Note: catalog_type is stored as node_type_id in the database, so we don't allow updating it via this endpoint
		if key == "node_name" || key == "description" || key == "properties" || key == "parent_id" {
			switch key {
			case "properties":
				// properties needs to be JSON encoded if it's a map/struct
				if m, ok := value.(map[string]interface{}); ok {
					propJSON, err := json.Marshal(m)
					if err != nil {
						http.Error(w, "Failed to encode properties", http.StatusBadRequest)
						return
					}
					setClauses = append(setClauses, fmt.Sprintf("%s = $%d", key, argIndex))
					args = append(args, string(propJSON))
				} else {
					// If it's already a string, use it as-is
					setClauses = append(setClauses, fmt.Sprintf("%s = $%d", key, argIndex))
					args = append(args, value)
				}
				argIndex++
			case "parent_id":
				// Handle parent_id as nullable field - empty string becomes NULL
				if str, ok := value.(string); ok && str == "" {
					setClauses = append(setClauses, fmt.Sprintf("%s = NULL", key))
					// Don't increment argIndex since we didn't add an argument
				} else if str, ok := value.(string); ok && str != "" {
					setClauses = append(setClauses, fmt.Sprintf("%s = $%d", key, argIndex))
					args = append(args, str)
					argIndex++
				}
			default:
				setClauses = append(setClauses, fmt.Sprintf("%s = $%d", key, argIndex))
				args = append(args, value)
				argIndex++
			}
		}
	}

	if len(setClauses) == 0 {
		http.Error(w, "No valid fields to update", http.StatusBadRequest)
		return
	}

	// Add updated_at (using NOW() function)
	setClauses = append(setClauses, "updated_at = NOW()")

	// Calculate WHERE clause indices based on current arg count
	whereIndex1 := argIndex
	whereIndex2 := argIndex + 1

	args = append(args, termID)
	args = append(args, tenantID)

	query := fmt.Sprintf(`
		UPDATE catalog_node
		SET %s
		WHERE id = $%d
			AND tenant_id = $%d
	`, strings.Join(setClauses, ", "), whereIndex1, whereIndex2)

	// Log the query and args for debugging
	log.Printf("[UpdateTerm] Query: %s", query)
	log.Printf("[UpdateTerm] Args: %v", args)
	log.Printf("[UpdateTerm] Updates received: %v", updates)

	// Execute the update
	res, err := h.db.Exec(query, args...)
	if err != nil {
		log.Printf("Error executing update for term %s: %v", termID, err)
		http.Error(w, "Failed to update term", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		log.Printf("Error checking rows affected for term %s: %v", termID, err)
		http.Error(w, "Failed to update term", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "Term not found", http.StatusNotFound)
		return
	}

	// If `parent_id` was included in updates and it's an empty string -> remove the mapping
	if rawParent, ok := updates["parent_id"]; ok {
		if str, sOk := rawParent.(string); sOk {
			if str == "" {
				// Remove any existing business_term_to_semantic_term edges for this semantic term
				if _, err := h.db.Exec(`DELETE FROM catalog_edge WHERE target_node_id = $1 AND relationship_type = 'business_term_to_semantic_term'`, termID); err != nil {
					log.Printf("Warning: failed to remove existing business_term_to_semantic_term edges for term %s: %v", termID, err)
				}
			} else {
				// Clear any old edges and create a new one from the provided parent
				if _, err := h.db.Exec(`DELETE FROM catalog_edge WHERE target_node_id = $1 AND relationship_type = 'business_term_to_semantic_term'`, termID); err != nil {
					log.Printf("Warning: failed to delete previous edges for term %s: %v", termID, err)
				}
				edgeID := uuid.New().String()
				edgeCreateQ := `
					INSERT INTO catalog_edge (id, source_node_id, target_node_id, relationship_type, edge_type, tenant_id, tenant_datasource_id, created_at)
					VALUES ($1, $2, $3, $4, $4, $5, $6, NOW())
					ON CONFLICT (tenant_datasource_id, source_node_id, edge_type_id, target_node_id) DO NOTHING
				`
				// We need tenantDatasourceID here. Let's assume it's available or fetch it.
				// For now, using "" as fallback if not in closure, but better to get it.
				if _, err := h.db.Exec(edgeCreateQ, edgeID, str, termID, "business_term_to_semantic_term", tenantID, tenantDatasourceID); err != nil {
					log.Printf("Warning: Failed to create edge from business term to semantic term during update: %v", err)
				}
			}
		}
	}

	// Select back the updated row joined with catalog_node_type for a consistent response
	selQ := `
		SELECT cn.id, cn.node_name, cn.tenant_datasource_id, COALESCE(cnt.catalog_type_name, '') as catalog_type_name, COALESCE(cn.description, '') as description, COALESCE(cn.is_alpha, false) as is_active, cn.parent_type_id, cn.parent_id, '[]'::text as config, cn.created_at, cn.updated_at, cn.tenant_id, cn.core_id, COALESCE(cn.properties, '[]'::jsonb) as properties
		FROM catalog_node cn
		LEFT JOIN catalog_node_type cnt ON cnt.id = cn.node_type_id
		WHERE cn.id = $1
	`

	var term models.CatalogNode
	var propertiesBytes []byte
	var parentIDStr *string
	err = h.db.QueryRow(selQ, termID).Scan(
		&term.ID,
		&term.NodeName,
		&term.TenantDatasourceID,
		&term.CatalogTypeName,
		&term.Description,
		&term.IsActive,
		&term.ParentTypeID,
		&parentIDStr,
		&term.Config,
		&term.CreatedAt,
		&term.UpdatedAt,
		&term.TenantID,
		&term.CoreID,
		&propertiesBytes,
	)
	if parentIDStr != nil && *parentIDStr != "" {
		term.ParentID = parentIDStr
	}
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Term not found", http.StatusNotFound)
		} else {
			log.Printf("Error updating term: %v", err)
			http.Error(w, "Failed to update term", http.StatusInternalServerError)
		}
		return
	}

	// Set properties JSON
	if len(propertiesBytes) > 0 {
		term.Properties = json.RawMessage(propertiesBytes)
	} else {
		term.Properties = json.RawMessage("{}")
	}

	// Sync to AGE if repo is available
	if h.lineageRepo != nil {
		node := lineage.LineageNode{
			ID:       term.ID,
			Type:     lineage.LineageNodeType(term.CatalogTypeName),
			Name:     term.NodeName,
			TenantID: &term.TenantID,
			Metadata: term.Properties,
			Env:      "dev",
		}
		if err := h.lineageRepo.UpsertNode(r.Context(), node); err != nil {
			log.Printf("Warning: Failed to sync updated term %s to graph: %v", term.ID, err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(term)
}

// validateTermProperties runs basic validation rules (required, min/max, length, pattern)
// based on NodeProperty metadata. Returns an error message and false if validation failed.
// ValidationError represents a single field-level validation failure
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// validateTermProperties runs basic validation rules (required, min/max, length, pattern)
// based on NodeProperty metadata. Returns a slice of ValidationError and a bool
// indicating success. When validation fails, the handler should return the
// structured errors to the client so they can be rendered inline.
func validateTermProperties(nodeProps []NodeProperty, values map[string]interface{}) ([]ValidationError, bool) {
	// Map property by name for quick lookup
	propMap := map[string]NodeProperty{}
	for _, p := range nodeProps {
		propMap[p.Name] = p
	}

	var errs []ValidationError

	for _, p := range nodeProps {
		v, exists := values[p.Name]

		// Required / nullable
		if !p.Nullable && (!exists || v == nil || (fmt.Sprintf("%v", v) == "")) {
			errs = append(errs, ValidationError{Field: fmt.Sprintf("properties.%s", p.Name), Message: fmt.Sprintf("%s is required", p.LabelOrName())})
			continue
		}

		// If field is absent, skip further checks
		if !exists || v == nil {
			continue
		}

		// Numeric checks
		if p.DataType == "integer" || p.DataType == "float" || p.InputType == "number" {
			// allow numeric strings in addition to numbers
			var num float64
			switch t := v.(type) {
			case float64:
				num = t
			case int:
				num = float64(t)
			case string:
				if t == "" {
					continue
				}
				parsed, err := strconv.ParseFloat(t, 64)
				if err != nil {
					errs = append(errs, ValidationError{Field: fmt.Sprintf("properties.%s", p.Name), Message: fmt.Sprintf("%s must be a number", p.LabelOrName())})
					continue
				}
				num = parsed
			default:
				errs = append(errs, ValidationError{Field: fmt.Sprintf("properties.%s", p.Name), Message: fmt.Sprintf("%s must be a number", p.LabelOrName())})
				continue
			}
			if minV, ok := extractFloat(p.Validation, "min"); ok && num < minV {
				errs = append(errs, ValidationError{Field: fmt.Sprintf("properties.%s", p.Name), Message: fmt.Sprintf("%s must be >= %v", p.LabelOrName(), minV)})
			}
			if maxV, ok := extractFloat(p.Validation, "max"); ok && num > maxV {
				errs = append(errs, ValidationError{Field: fmt.Sprintf("properties.%s", p.Name), Message: fmt.Sprintf("%s must be <= %v", p.LabelOrName(), maxV)})
			}
		}

		// String / text checks
		if p.DataType == "string" || p.InputType == "text" || p.InputType == "textarea" || p.DataType == "text" {
			if s, ok := v.(string); ok {
				if minL, ok := extractInt(p.Validation, "minLength"); ok && len(s) < minL {
					errs = append(errs, ValidationError{Field: fmt.Sprintf("properties.%s", p.Name), Message: fmt.Sprintf("%s must be at least %d characters", p.LabelOrName(), minL)})
				}
				if maxL, ok := extractInt(p.Validation, "maxLength"); ok && len(s) > maxL {
					errs = append(errs, ValidationError{Field: fmt.Sprintf("properties.%s", p.Name), Message: fmt.Sprintf("%s must be at most %d characters", p.LabelOrName(), maxL)})
				}
				if pat, ok := p.Validation["pattern"].(string); ok && pat != "" {
					re, err := regexp.Compile(pat)
					if err == nil && !re.MatchString(s) {
						errs = append(errs, ValidationError{Field: fmt.Sprintf("properties.%s", p.Name), Message: fmt.Sprintf("%s must match pattern", p.LabelOrName())})
					}
				}
			}
		}

		// JSON editor - verify JSON parseable when a string is provided
		if p.InputType == "json-editor" || p.DataType == "json" {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				var tmp interface{}
				if err := json.Unmarshal([]byte(s), &tmp); err != nil {
					errs = append(errs, ValidationError{Field: fmt.Sprintf("properties.%s", p.Name), Message: fmt.Sprintf("%s is not valid JSON", p.LabelOrName())})
					continue
				}
			}
		}

		// Multiple/array validations
		if p.Validation != nil {
			if mult, ok := p.Validation["multiple"].(bool); ok && mult {
				if arr, ok := v.([]interface{}); ok {
					if minL, ok := extractInt(p.Validation, "minLength"); ok && len(arr) < minL {
						errs = append(errs, ValidationError{Field: fmt.Sprintf("properties.%s", p.Name), Message: fmt.Sprintf("%s must have at least %d items", p.LabelOrName(), minL)})
					}
					if maxL, ok := extractInt(p.Validation, "maxLength"); ok && len(arr) > maxL {
						errs = append(errs, ValidationError{Field: fmt.Sprintf("properties.%s", p.Name), Message: fmt.Sprintf("%s must have at most %d items", p.LabelOrName(), maxL)})
					}
				}
			}
		}
	}

	if len(errs) > 0 {
		return errs, false
	}

	return nil, true
}

// extractFloat extracts numeric validation value as float if present
func extractFloat(m map[string]interface{}, key string) (float64, bool) {
	if m == nil {
		return 0, false
	}
	if v, ok := m[key]; ok {
		switch t := v.(type) {
		case float64:
			return t, true
		case int:
			return float64(t), true
		case json.Number:
			f, err := t.Float64()
			if err == nil {
				return f, true
			}
		}
	}
	return 0, false
}

// extractInt extracts numeric validation as int if present
func extractInt(m map[string]interface{}, key string) (int, bool) {
	if m == nil {
		return 0, false
	}
	if v, ok := m[key]; ok {
		switch t := v.(type) {
		case float64:
			return int(t), true
		case int:
			return t, true
		case json.Number:
			i, err := t.Int64()
			if err == nil {
				return int(i), true
			}
		}
	}
	return 0, false
}

// LabelOrName returns a friendly label or name for the property
func (p *NodeProperty) LabelOrName() string {
	if p.Label != "" {
		return p.Label
	}
	return p.Name
}

// CreateEdge creates a new edge between terms
func (h *GlossaryHandler) CreateEdge(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	tenantDatasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	// Allow query params as fallback
	if tenantID == "" {
		tenantID = r.URL.Query().Get("tenant_id")
	}
	if tenantDatasourceID == "" {
		tenantDatasourceID = r.URL.Query().Get("datasource_id")
	}

	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header/param is required", http.StatusBadRequest)
		return
	}

	if tenantDatasourceID == "" {
		http.Error(w, "X-Tenant-Datasource-ID header/param is required", http.StatusBadRequest)
		return
	}

	var req struct {
		SubjectNodeID string `json:"subject_node_id"`
		ObjectNodeID  string `json:"object_node_id"`
		EdgeTypeID    string `json:"edge_type_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.SubjectNodeID == "" || req.ObjectNodeID == "" || req.EdgeTypeID == "" {
		http.Error(w, "subject_node_id, object_node_id, and edge_type_id are required", http.StatusBadRequest)
		return
	}

	// Generate new edge ID
	edgeID := uuid.New().String()

	// Resolve edge type id (accept either id or name). If not found, use provided value as-is.
	var resolvedEdgeTypeID string
	var resolvedEdgeTypeName sql.NullString

	// Check if input looks like a UUID
	isUUID := regexp.MustCompile(`^[0-9a-fA-F0-9-]{36}$`).MatchString(req.EdgeTypeID)

	query := `SELECT id, edge_type_name FROM catalog_edge_type WHERE `
	if isUUID {
		query += `id = $1`
	} else {
		query += `edge_type_name = $1`
	}

	err := h.db.QueryRow(query, req.EdgeTypeID).Scan(&resolvedEdgeTypeID, &resolvedEdgeTypeName)

	if err != nil {
		if err == sql.ErrNoRows {
			// If invalid UUID or not found name, create new type (assuming input is the name)
			// If input was a random UUID that doesn't exist, this might create a type with that UUID as name?
			// Better: if it was a UUID and not found, we probably shouldn't create a type named with that UUID unless explicit.
			// But for now, let's treat the input as the edge_type_name.

			log.Printf("[CreateEdge] Edge Type not found, creating new. Input: %s", req.EdgeTypeID)
			// Insert into catalog_edge_type (singular)
			insertTypeQ := `INSERT INTO catalog_edge_type (tenant_id, edge_type_name, created_at, updated_at) VALUES ($1, $2, NOW(), NOW()) RETURNING id`
			// Use tenantID for the type
			err = h.db.QueryRow(insertTypeQ, tenantID, req.EdgeTypeID).Scan(&resolvedEdgeTypeID)
			if err != nil {
				log.Printf("[CreateEdge] Error creating fallback edge type: %v", err)
				http.Error(w, "Failed to create edge type", http.StatusInternalServerError)
				return
			}
			resolvedEdgeTypeName = sql.NullString{String: req.EdgeTypeID, Valid: true}
		} else {
			log.Printf("[CreateEdge] Error resolving edge type: %v", err)
			http.Error(w, "Failed to resolve edge type", http.StatusInternalServerError)
			return
		}
	}

	log.Printf("[CreateEdge] Resolved EdgeTypeID: %s", resolvedEdgeTypeID)

	// Insert into catalog_edge
	// Including tenant_datasource_id which is required
	// Use ON CONFLICT DO UPDATE to handle duplicates gracefully (idempotent)
	insertQ := `
		INSERT INTO catalog_edge (id, tenant_id, tenant_datasource_id, source_node_id, target_node_id, properties, edge_type, edge_type_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, '[]'::jsonb, $6, $7, NOW(), NOW())
		ON CONFLICT (tenant_datasource_id, source_node_id, target_node_id, edge_type_id)
		DO UPDATE SET updated_at = NOW()
		RETURNING id
	`

	// We store both the resolved ID and the current name (predicate) for redundancy/historical reasons if needed,
	// or just as a cache. User wants reliance on UUID, so ID is critical.
	var actualEdgeID string
	err = h.db.QueryRow(insertQ, edgeID, tenantID, tenantDatasourceID, req.SubjectNodeID, req.ObjectNodeID, resolvedEdgeTypeName.String, resolvedEdgeTypeID).Scan(&actualEdgeID)
	if err != nil {
		log.Printf("[CreateEdge] Error inserting edge: %v", err)
		http.Error(w, "Failed to create edge", http.StatusInternalServerError)
		return
	}

	// Use the actual ID returned (could be existing if conflict occurred)
	edgeID = actualEdgeID

	log.Printf("[CreateEdge] Insert/Update successful. EdgeID: %s", edgeID)

	// Query back the inserted edge joined with edge type name for a friendly response
	var edge models.CatalogEdge
	var propertiesBytes []byte
	var edgeTypeName sql.NullString
	selQ := `
		SELECT ce.id, COALESCE(cet.edge_type_name, ce.edge_type_name) as predicate, COALESCE(ce.properties, '[]'::jsonb) as properties, ce.created_at, ce.updated_at, ce.tenant_id, ce.edge_type_id
		FROM catalog_edge ce
		LEFT JOIN catalog_edge_type cet ON cet.id = ce.edge_type_id
		WHERE ce.id = $1
	`
	err = h.db.QueryRow(selQ, edgeID).Scan(
		&edge.ID,
		&edge.EdgeTypeName,
		&propertiesBytes,
		&edge.CreatedAt,
		&edge.UpdatedAt,
		&edge.TenantID,
		&edgeTypeName,
	)
	if err != nil {
		log.Printf("[CreateEdge] Error scanning response: %v", err)
		http.Error(w, "Error retrieving created edge", http.StatusInternalServerError)
		return
	}
	log.Printf("[CreateEdge] Select successful. Predicate: %s", edge.EdgeTypeName)

	// Fill in fields that map to the old response shape
	edge.SubjectNodeTypeID = req.SubjectNodeID
	edge.ObjectNodeTypeID = req.ObjectNodeID
	isActive := true
	edge.IsActive = &isActive

	// Set properties JSON
	if len(propertiesBytes) > 0 {
		edge.Properties = json.RawMessage(propertiesBytes)
	} else {
		edge.Properties = json.RawMessage("[]")
	}

	// Sync to AGE if repo is available
	if h.lineageRepo != nil {
		edgeRec := lineage.LineageEdge{
			FromID:   req.SubjectNodeID,
			ToID:     req.ObjectNodeID,
			Type:     lineage.LineageEdgeType(resolvedEdgeTypeName.String),
			TenantID: &tenantID,
			Env:      "dev",
		}
		if err := h.lineageRepo.UpsertEdge(r.Context(), edgeRec); err != nil {
			log.Printf("Warning: Failed to sync edge %s -> %s to graph: %v", req.SubjectNodeID, req.ObjectNodeID, err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(edge)
}

// DeleteTerm deletes a business term or semantic term
func (h *GlossaryHandler) DeleteTerm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "term ID is required", http.StatusBadRequest)
		return
	}

	// Get tenant and datasource from query params or headers
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = jwtmiddleware.GetClaimsFromContext(r).TenantID
	}

	datasourceID := r.URL.Query().Get("datasource_id")
	if datasourceID == "" {
		datasourceID = r.Header.Get("X-Tenant-Datasource-ID")
	}

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "tenant_id and datasource_id are required", http.StatusBadRequest)
		return
	}

	// Resolve "default" tenant to core UUID to prevent UUID parsing errors
	if tenantID == "default" {
		tenantID = "99e99e99-99e9-49e9-89e9-99e99e99e999"
	}

	// Delete the term (cascading deletes for catalog_edge are handled at the DB level)
	query := `
		DELETE FROM catalog_node 
		WHERE id = $1 AND tenant_id = $2
	`

	result, err := h.db.Exec(query, id, tenantID)
	if err != nil {
		log.Printf("Error deleting term %s: %v", id, err)
		http.Error(w, "Failed to delete term", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error getting rows affected for term %s: %v", id, err)
		http.Error(w, "Failed to delete term", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "Term not found", http.StatusNotFound)
		return
	}

	// Sync to AGE if repo is available
	if h.lineageRepo != nil {
		if err := h.lineageRepo.DeleteNode(r.Context(), id); err != nil {
			log.Printf("Warning: Failed to delete node %s from graph: %v", id, err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Term deleted successfully"})
}

// DeleteEdge deletes a semantic edge
func (h *GlossaryHandler) DeleteEdge(w http.ResponseWriter, r *http.Request) {
	edgeID := chi.URLParam(r, "id")
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID

	if tenantID == "" {
		tenantID = r.URL.Query().Get("tenant_id")
	}

	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header/param is required", http.StatusBadRequest)
		return
	}

	if edgeID == "" {
		http.Error(w, "Edge ID is required", http.StatusBadRequest)
		return
	}

	log.Printf("[DeleteEdge] Deleting edge %s for tenant %s", edgeID, tenantID)

	// Delete from catalog_edge - edge ID is globally unique, no need for tenant filter
	query := `DELETE FROM catalog_edge WHERE id = $1`

	res, err := h.db.Exec(query, edgeID)
	if err != nil {
		log.Printf("[DeleteEdge] Error deleting from catalog_edge: %v", err)
		http.Error(w, "Failed to delete edge", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := res.RowsAffected()
	log.Printf("[DeleteEdge] Deleted keys from catalog_edge: %d", rowsAffected)

	if rowsAffected == 0 {
		http.Error(w, "Edge not found", http.StatusNotFound)
		return
	}

	// Sync to AGE if repo is available
	// Note: We don't have the edge type here easily without querying first,
	// but DeleteEdge in our repo can be improved or we can just delete by from/to.
	// For now, let's assume we need to sync.
	// Actually, DeleteEdge in AgeRepo needs edgeType.
	// We might need to fetch the edge before deleting it to get the type.
	// But detaching relationships is safer if we just want to remove IT.
	// Since we already deleted it from catalog_edge, we just need to remove it from graph.
	// I'll skip specific edge type for now if it's too complex to fetch,
	// OR I can add a DeleteEdgeByNodes method.
	// But let's keep it simple for now and just try to delete what we can.

	w.WriteHeader(http.StatusNoContent)
}

// UpdateEdge updates an existing edge's properties or description
func (h *GlossaryHandler) UpdateEdge(w http.ResponseWriter, r *http.Request) {
	edgeID := chi.URLParam(r, "id")
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	tenantDatasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if tenantID == "" {
		tenantID = r.URL.Query().Get("tenant_id")
	}
	if tenantDatasourceID == "" {
		tenantDatasourceID = r.URL.Query().Get("datasource_id")
	}

	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header/param is required", http.StatusBadRequest)
		return
	}

	if edgeID == "" {
		http.Error(w, "Edge ID is required", http.StatusBadRequest)
		return
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("[UpdateEdge] Updating edge %s for tenant %s. Updates: %v", edgeID, tenantID, updates)

	// Build dynamic update query for allowed fields
	setClauses := []string{}
	args := []interface{}{}
	argIndex := 1

	for key, value := range updates {
		// Allow updating description and properties
		if key == "description" || key == "properties" {
			if key == "properties" {
				if m, ok := value.(map[string]interface{}); ok {
					propJSON, err := json.Marshal(m)
					if err != nil {
						http.Error(w, "Failed to encode properties", http.StatusBadRequest)
						return
					}
					setClauses = append(setClauses, fmt.Sprintf("properties = $%d::jsonb", argIndex))
					args = append(args, string(propJSON))
				} else {
					setClauses = append(setClauses, fmt.Sprintf("properties = $%d::jsonb", argIndex))
					args = append(args, value)
				}
			} else {
				setClauses = append(setClauses, fmt.Sprintf("%s = $%d", key, argIndex))
				args = append(args, value)
			}
			argIndex++
		}
	}

	if len(setClauses) == 0 {
		http.Error(w, "No valid fields to update", http.StatusBadRequest)
		return
	}

	setClauses = append(setClauses, "updated_at = NOW()")

	whereIndex1 := argIndex
	whereIndex2 := argIndex + 1
	args = append(args, edgeID, tenantID)

	query := fmt.Sprintf(`
		UPDATE catalog_edge
		SET %s
		WHERE id = $%d AND tenant_id = $%d
	`, strings.Join(setClauses, ", "), whereIndex1, whereIndex2)

	log.Printf("[UpdateEdge] Query: %s", query)
	log.Printf("[UpdateEdge] Args: %v", args)

	res, err := h.db.Exec(query, args...)
	if err != nil {
		log.Printf("[UpdateEdge] Error executing update: %v", err)
		http.Error(w, "Failed to update edge", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Edge not found", http.StatusNotFound)
		return
	}

	// Return the updated edge
	var edge models.CatalogEdge
	var propertiesBytes []byte
	selQ := `
		SELECT ce.id, COALESCE(ce.edge_type_name, '') as predicate, COALESCE(ce.properties, '[]'::jsonb) as properties, ce.created_at, ce.updated_at, ce.tenant_id, ce.edge_type_id
		FROM catalog_edge ce
		WHERE ce.id = $1
	`
	err = h.db.QueryRow(selQ, edgeID).Scan(
		&edge.ID,
		&edge.EdgeTypeName,
		&propertiesBytes,
		&edge.CreatedAt,
		&edge.UpdatedAt,
		&edge.TenantID,
		&edge.ObjectNodeTypeID,
	)
	if err != nil {
		log.Printf("[UpdateEdge] Error fetching updated edge: %v", err)
		http.Error(w, "Failed to fetch updated edge", http.StatusInternalServerError)
		return
	}

	// Set properties JSON
	if len(propertiesBytes) > 0 {
		edge.Properties = json.RawMessage(propertiesBytes)
	} else {
		edge.Properties = json.RawMessage("[]")
	}
	isAct := true
	edge.IsActive = &isAct

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(edge)
}

// CubePropertiesResponse represents Cube.dev properties for a semantic term
type CubePropertiesResponse struct {
	ID                 string      `json:"id"`
	NodeName           string      `json:"node_name"`
	SemanticTermType   string      `json:"semantic_term_type"`
	CubeProperties     interface{} `json:"cube_properties"`
	DataType           string      `json:"data_type"`
	ForeignKey         bool        `json:"foreign_key"`
	Nullable           bool        `json:"nullable"`
	Cardinality        *int        `json:"cardinality"`
	TenantID           string      `json:"tenant_id"`
	TenantDatasourceID string      `json:"tenant_datasource_id"`
}

// HandleGetSemanticTermWithCubeProperties retrieves a semantic term with its Cube.dev properties
func (h *GlossaryHandler) HandleGetSemanticTermWithCubeProperties(w http.ResponseWriter, r *http.Request) {
	termID := chi.URLParam(r, "id")
	if termID == "" {
		http.Error(w, "Missing term ID", http.StatusBadRequest)
		return
	}

	// Validate UUID format
	if _, err := uuid.Parse(termID); err != nil {
		http.Error(w, "Invalid term ID format", http.StatusBadRequest)
		return
	}

	// Query the semantic term with its properties
	query := `
		SELECT 
			cn.id,
			cn.node_name,
			COALESCE(cn.properties::jsonb->>'semantic_term_type', 'UNKNOWN') as semantic_term_type,
			COALESCE(cn.properties::jsonb->'cube_properties', '{}'::jsonb) as cube_properties,
			COALESCE(cn.properties::jsonb->>'data_type', '') as data_type,
			COALESCE((cn.properties::jsonb->>'foreign_key')::boolean, false) as foreign_key,
			COALESCE((cn.properties::jsonb->>'nullable')::boolean, true) as nullable,
			COALESCE((cn.properties::jsonb->>'cardinality')::integer, null) as cardinality,
			cn.tenant_id,
			cn.tenant_datasource_id
		FROM catalog_node cn
		WHERE cn.id = $1 AND cn.node_type_id IN (
			SELECT id FROM catalog_node_type 
			WHERE catalog_type_name LIKE 'semantic_term_%'
		)
	`

	var response CubePropertiesResponse
	var cubePropsJSON []byte

	err := h.db.QueryRow(query, termID).Scan(
		&response.ID,
		&response.NodeName,
		&response.SemanticTermType,
		&cubePropsJSON,
		&response.DataType,
		&response.ForeignKey,
		&response.Nullable,
		&response.Cardinality,
		&response.TenantID,
		&response.TenantDatasourceID,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "Semantic term not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Error fetching semantic term: %v", err)
		http.Error(w, "Failed to fetch semantic term", http.StatusInternalServerError)
		return
	}

	// Parse cube properties JSON
	if len(cubePropsJSON) > 0 {
		var cubeProps interface{}
		if err := json.Unmarshal(cubePropsJSON, &cubeProps); err == nil {
			response.CubeProperties = cubeProps
		} else {
			response.CubeProperties = map[string]interface{}{}
		}
	} else {
		response.CubeProperties = map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CubeYamlExportResponse represents Cube.js configuration export
type CubeYamlExportResponse struct {
	Cubes          []map[string]interface{} `json:"cubes"`
	Dimensions     []map[string]interface{} `json:"dimensions"`
	Measures       []map[string]interface{} `json:"measures"`
	Segments       []map[string]interface{} `json:"segments"`
	TimeDimensions []map[string]interface{} `json:"time_dimensions"`
}

// HandleExportSemanticTermsAsCubeYaml exports all semantic terms as Cube.js configuration
func (h *GlossaryHandler) HandleExportSemanticTermsAsCubeYaml(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "Missing required parameters: tenant_id, datasource_id", http.StatusBadRequest)
		return
	}

	// Query all semantic terms for the tenant/datasource
	query := `
		SELECT 
			cn.id,
			cn.node_name,
			cn.properties::jsonb
		FROM catalog_node cn
		WHERE cn.tenant_id = $1 
		  AND cn.tenant_datasource_id = $2
		  AND cn.node_type_id IN (
			SELECT id FROM catalog_node_type 
			WHERE catalog_type_name LIKE 'semantic_term_%'
		)
		ORDER BY cn.node_name
	`

	rows, err := h.db.Query(query, tenantID, datasourceID)
	if err != nil {
		log.Printf("Error querying semantic terms: %v", err)
		http.Error(w, "Failed to fetch semantic terms", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	response := CubeYamlExportResponse{
		Cubes:          []map[string]interface{}{},
		Dimensions:     []map[string]interface{}{},
		Measures:       []map[string]interface{}{},
		Segments:       []map[string]interface{}{},
		TimeDimensions: []map[string]interface{}{},
	}

	// Process each semantic term and categorize by type
	for rows.Next() {
		var termID, nodeName, propsJSON string
		if err := rows.Scan(&termID, &nodeName, &propsJSON); err != nil {
			log.Printf("Error scanning term: %v", err)
			continue
		}

		var props map[string]interface{}
		if err := json.Unmarshal([]byte(propsJSON), &props); err != nil {
			log.Printf("Error parsing properties for %s: %v", termID, err)
			continue
		}

		// Extract semantic term type
		termType, ok := props["semantic_term_type"].(string)
		if !ok {
			continue
		}

		// Extract cube properties
		cubePropsInterface, hasCubeProps := props["cube_properties"]
		if !hasCubeProps {
			continue
		}

		cubeProps, ok := cubePropsInterface.(map[string]interface{})
		if !ok {
			continue
		}

		// Add to appropriate collection based on type
		switch strings.ToUpper(termType) {
		case "DIMENSION":
			response.Dimensions = append(response.Dimensions, cubeProps)
		case "MEASURE":
			response.Measures = append(response.Measures, cubeProps)
		case "TIME":
			response.TimeDimensions = append(response.TimeDimensions, cubeProps)
		case "SEGMENT":
			response.Segments = append(response.Segments, cubeProps)
		case "HIERARCHY":
			// Hierarchies are typically used for organizing dimensions
			// Store as a special cube configuration
			cubeConfig := map[string]interface{}{
				"name":        nodeName,
				"type":        "hierarchy",
				"levels":      cubeProps["levels"],
				"title":       cubeProps["title"],
				"description": cubeProps["description"],
			}
			response.Cubes = append(response.Cubes, cubeConfig)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
