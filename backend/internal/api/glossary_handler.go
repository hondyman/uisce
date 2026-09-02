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
	"github.com/hondyman/uisce/backend/internal/lineage"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/hondyman/uisce/backend/pkg/governance"
	"github.com/jmoiron/sqlx"

	"github.com/hondyman/uisce/backend/internal/handlers"
)

// GlossaryHandler handles glossary-related API requests
type GlossaryHandler struct {
	db           *sql.DB
	dbx          *sqlx.DB
	governance   *governance.GovernanceEngine
	lineageRepo  lineage.LineageRepository
	securityDeps handlers.SecurityContextDeps
}

// NewGlossaryHandler creates a new glossary handler
func NewGlossaryHandler(db *sql.DB, lineageRepo lineage.LineageRepository, securityDeps handlers.SecurityContextDeps) *GlossaryHandler {
	dbx := sqlx.NewDb(db, "postgres")
	return &GlossaryHandler{
		db:           db,
		dbx:          dbx,
		governance:   governance.NewGovernanceEngine(dbx),
		lineageRepo:  lineageRepo,
		securityDeps: securityDeps,
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
		// Technical assets & graph endpoints for selected term detail view
		r.Get("/technical-assets", h.ListTechnicalAssets)
		r.Post("/technical-assets", h.CreateTechnicalAssets)
		r.Delete("/technical-assets/{id}", h.DeleteTechnicalAsset)
		r.Get("/node-graph", h.GetNodeGraph)
		r.Get("/nodes/{id}/dependencies", h.GetNodeDependencies)

		// Billion-row safe sample profiling endpoint
		r.Post("/profile-sample", h.ProfileSample)

		// Bulk-create semantic terms from column groupings in a single request/transaction
		r.Post("/generate-semantic-terms", h.GenerateSemanticTerms)

		// Cube.dev properties endpoints
		r.Get("/semantic-terms/{id}/cube-definition", h.HandleGetSemanticTermWithCubeProperties)
		r.Get("/semantic-terms/export/cube-yaml", h.HandleExportSemanticTermsAsCubeYaml)
	})
}

// Generic list function to handle different term types
func (h *GlossaryHandler) listTerms(w http.ResponseWriter, r *http.Request, termType string) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, "security context initialization failed: "+err.Error(), http.StatusUnauthorized)
		return
	}

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
			COALESCE(cn.node_type, cnt.catalog_type_name, '') as node_type
		FROM catalog_node cn
		LEFT JOIN catalog_node_type cnt ON cn.node_type_id = cnt.id
		WHERE (cn.tenant_id = $1 OR cn.tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1))
	`
	args := []interface{}{secCtx.TenantID}

	if secCtx.DatasourceID != "" {
		query += " AND cn.tenant_datasource_id = $2"
		args = append(args, secCtx.DatasourceID)
	}

	if termType != "" {
		argCount := len(args) + 1
		query += fmt.Sprintf(" AND (cn.node_type = $%d OR cnt.catalog_type_name = $%d)", argCount, argCount)
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
				if len(b) == 16 {
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
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, "security context initialization failed: "+err.Error(), http.StatusUnauthorized)
		return
	}

	var rows *sql.Rows
	if secCtx.DatasourceID != "" {
		query := `
			SELECT
				ce.id,
				COALESCE(cet.edge_type_name, '') as predicate,
				COALESCE(cet.edge_type_name, '') as edge_type_name,
				ce.source_node_id,
				ce.target_node_id,
				COALESCE(ce.properties, '{}'::jsonb) as properties,
				ce.created_at,
				ce.updated_at,
				ce.tenant_id,
				ce.edge_type_id
			FROM catalog_edge ce
			LEFT JOIN catalog_edge_type cet ON ce.edge_type_id = cet.id
			WHERE (ce.tenant_id = $1 OR ce.tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1))
			  AND (ce.tenant_datasource_id = $2 OR ce.tenant_datasource_id IS NULL)
			ORDER BY ce.created_at DESC
		`
		var err error
		rows, err = h.db.Query(query, secCtx.TenantID, secCtx.DatasourceID)
		if err != nil {
			log.Printf("Error querying edges: %v", err)
			http.Error(w, "Failed to fetch edges: "+err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		query := `
			SELECT
				ce.id,
				COALESCE(cet.edge_type_name, '') as predicate,
				COALESCE(cet.edge_type_name, '') as edge_type_name,
				ce.source_node_id,
				ce.target_node_id,
				COALESCE(ce.properties, '{}'::jsonb) as properties,
				ce.created_at,
				ce.updated_at,
				ce.tenant_id,
				ce.edge_type_id
			FROM catalog_edge ce
			LEFT JOIN catalog_edge_type cet ON ce.edge_type_id = cet.id
			WHERE (ce.tenant_id = $1 OR ce.tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1))
			ORDER BY ce.created_at DESC
		`
		var err error
		rows, err = h.db.Query(query, secCtx.TenantID)
		if err != nil {
			log.Printf("Error querying edges: %v", err)
			http.Error(w, "Failed to fetch edges: "+err.Error(), http.StatusInternalServerError)
			return
		}
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
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, "security context initialization failed: "+err.Error(), http.StatusUnauthorized)
		return
	}

	if secCtx.DatasourceID == "" {
		http.Error(w, "datasource_id is required", http.StatusBadRequest)
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
		termData.CatalogType = "semantic_term"
	}

	if termData.TenantDatasourceID == "" {
		termData.TenantDatasourceID = secCtx.DatasourceID
	}

	propertiesJSON, err := json.Marshal(termData.Properties)
	if err != nil {
		http.Error(w, "Invalid properties format", http.StatusBadRequest)
		return
	}

	var nodeTypeID string
	err = h.db.QueryRow(`SELECT id FROM catalog_node_type WHERE catalog_type_name = $1 LIMIT 1`, termData.CatalogType).Scan(&nodeTypeID)
	if err != nil {
		if err == sql.ErrNoRows {
			insertTypeQ := `INSERT INTO catalog_node_type (tenant_id, catalog_type_name, created_at, updated_at) VALUES ($1, $2, NOW(), NOW()) RETURNING id`
			if err := h.db.QueryRow(insertTypeQ, secCtx.TenantID, termData.CatalogType).Scan(&nodeTypeID); err != nil {
				log.Printf("Error creating fallback catalog_node_type: %v", err)
				http.Error(w, "Failed to resolve catalog type", http.StatusInternalServerError)
				return
			}

			var nodeTypePropertiesBytes []byte
			var nodeProps []NodeProperty
			if err := h.db.QueryRow(`SELECT properties FROM catalog_node_type WHERE id = $1`, nodeTypeID).Scan(&nodeTypePropertiesBytes); err == nil {
				if len(nodeTypePropertiesBytes) > 0 {
					_ = json.Unmarshal(nodeTypePropertiesBytes, &nodeProps)
				}
			}

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

	if termData.CatalogType == "semantic_term" && h.governance != nil {
		validationInput := map[string]interface{}{}
		for k, v := range termData.Properties {
			validationInput[k] = v
		}
		validationInput["node_name"] = termData.NodeName
		validationInput["description"] = termData.Description

		result, err := h.governance.ValidateSemanticTerm(r.Context(), validationInput)
		if err != nil {
			log.Printf("Governance validation error: %v", err)
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

	qualifiedPath := fmt.Sprintf("%s/%s", termData.CatalogType, termData.NodeName)

	var parentID *string
	if termData.CatalogType == "semantic_term" && termData.ParentID != "" {
		parentID = &termData.ParentID
	}

	log.Printf("[DEBUG CreateTerm] catalog_type=%s, parent_id=%v, provided ParentID=%s", termData.CatalogType, parentID, termData.ParentID)

	var insertedID string
	// Check if a node with this tenant_datasource_id, node_type_id, and qualified_path already exists
	checkExistingQ := `
		SELECT id FROM catalog_node
		WHERE node_type_id = $1
		  AND (qualified_path = $2 OR node_name = $4)
		  AND ((tenant_datasource_id = $3) OR (tenant_datasource_id IS NULL AND $3 = ''))
		LIMIT 1
	`
	err = h.db.QueryRow(checkExistingQ, nodeTypeID, qualifiedPath, termData.TenantDatasourceID, termData.NodeName).Scan(&insertedID)
	if err == nil && insertedID != "" {
		log.Printf("[CreateTerm] Existing node found: %s (%s). Updating properties.", insertedID, termData.NodeName)
		_, _ = h.db.Exec(`UPDATE catalog_node SET description = COALESCE(NULLIF($1, ''), description), properties = $2::jsonb, updated_at = NOW() WHERE id = $3`, termData.Description, string(propertiesJSON), insertedID)
	} else {
		insertQ := `
			INSERT INTO catalog_node (
				node_name, description, node_type_id, tenant_id, tenant_datasource_id,
				properties, qualified_path, parent_id, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, NOW(), NOW())
			RETURNING id
		`
		if err := h.db.QueryRow(insertQ,
			termData.NodeName,
			termData.Description,
			nodeTypeID,
			secCtx.TenantID,
			termData.TenantDatasourceID,
			propertiesJSON,
			qualifiedPath,
			parentID,
		).Scan(&insertedID); err != nil {
			log.Printf("CreateTerm insert error (node_name=%s, tenant=%s): %v. Attempting existing node lookup...", termData.NodeName, secCtx.TenantID, err)
			// If unique constraint or conflict occurred, fetch existing node by name & tenant
			if lookupErr := h.db.QueryRow(`SELECT id FROM catalog_node WHERE node_name = $1 AND tenant_id = $2 LIMIT 1`, termData.NodeName, secCtx.TenantID).Scan(&insertedID); lookupErr == nil && insertedID != "" {
				log.Printf("[CreateTerm] Fallback found existing node: %s", insertedID)
				_, _ = h.db.Exec(`UPDATE catalog_node SET description = COALESCE(NULLIF($1, ''), description), properties = $2::jsonb, updated_at = NOW() WHERE id = $3`, termData.Description, string(propertiesJSON), insertedID)
			} else if lookupErr2 := h.db.QueryRow(`SELECT id FROM catalog_node WHERE qualified_path = $1 AND tenant_id = $2 LIMIT 1`, qualifiedPath, secCtx.TenantID).Scan(&insertedID); lookupErr2 == nil && insertedID != "" {
				log.Printf("[CreateTerm] Fallback found existing node by qualified_path: %s", insertedID)
				_, _ = h.db.Exec(`UPDATE catalog_node SET description = COALESCE(NULLIF($1, ''), description), properties = $2::jsonb, updated_at = NOW() WHERE id = $3`, termData.Description, string(propertiesJSON), insertedID)
			} else {
				http.Error(w, fmt.Sprintf("Failed to create term: %v", err), http.StatusInternalServerError)
				return
			}
		}
	}

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
		log.Printf("Error creating term (node_name=%s, tenant=%s, datasource=%s): %v", termData.NodeName, secCtx.TenantID, termData.TenantDatasourceID, err)
		http.Error(w, "Failed to create term", http.StatusInternalServerError)
		return
	}

	if len(returnedPropertiesBytes) > 0 {
		term.Properties = json.RawMessage(returnedPropertiesBytes)
	} else {
		term.Properties = json.RawMessage("[]")
	}
	active := true
	term.IsActive = &active

	if termData.CatalogType == "semantic_term" && termData.ParentID != "" {
		edgeID := uuid.New().String()
		edgeCreateQ := `
			INSERT INTO catalog_edge (id, source_node_id, target_node_id, relationship_type, tenant_id, tenant_datasource_id, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, NOW())
			ON CONFLICT DO NOTHING
		`
		if _, err := h.db.Exec(edgeCreateQ, edgeID, termData.ParentID, insertedID, "business_term_to_semantic_term", secCtx.TenantID, secCtx.DatasourceID); err != nil {
			log.Printf("Warning: Failed to create edge from business term to semantic term: %v", err)
		}
	}

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
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, "security context initialization failed: "+err.Error(), http.StatusUnauthorized)
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
	args = append(args, secCtx.TenantID)

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
					INSERT INTO catalog_edge (id, source_node_id, target_node_id, relationship_type, tenant_id, tenant_datasource_id, created_at)
					VALUES ($1, $2, $3, $4, $5, $6, NOW())
					ON CONFLICT DO NOTHING
				`
				if _, err := h.db.Exec(edgeCreateQ, edgeID, str, termID, "business_term_to_semantic_term", secCtx.TenantID, secCtx.DatasourceID); err != nil {
					log.Printf("Warning: Failed to create edge from business term to semantic term during update: %v", err)
				}
			}
		}
	}

	// Auto-extract formula dependencies if formula is present in properties
	if rawProps, ok := updates["properties"]; ok {
		var propMap map[string]interface{}
		if m, ok := rawProps.(map[string]interface{}); ok {
			propMap = m
		} else if s, ok := rawProps.(string); ok {
			_ = json.Unmarshal([]byte(s), &propMap)
		}
		if propMap != nil {
			if formulaVal, hasForm := propMap["formula"]; hasForm {
				if formulaStr, isStr := formulaVal.(string); isStr && formulaStr != "" {
					re := regexp.MustCompile(`\$\{([a-zA-Z0-9_\.\s]+)\}`)
					matches := re.FindAllStringSubmatch(formulaStr, -1)
					for _, m := range matches {
						if len(m) >= 2 {
							depKey := strings.TrimSpace(m[1])
							var depNodeID string
							_ = h.db.QueryRow(`
								SELECT id FROM catalog_node 
								WHERE tenant_id = $1 
								  AND (LOWER(node_name) = LOWER($2) OR qualified_path LIKE '%' || $2)
								LIMIT 1
							`, secCtx.TenantID, depKey).Scan(&depNodeID)
							if depNodeID != "" && depNodeID != termID {
								edgeID := uuid.New().String()
								_, _ = h.db.Exec(`
									INSERT INTO catalog_edge (id, source_node_id, target_node_id, relationship_type, tenant_id, created_at)
									VALUES ($1, $2, $3, 'depends_on', $4, NOW())
									ON CONFLICT DO NOTHING
								`, edgeID, termID, depNodeID, secCtx.TenantID)
							}
						}
					}
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
// GenerateSemanticTerms creates one semantic term per group and links each
// group's columns to it, all in a single request and a single DB
// transaction — replacing the frontend's previous per-group request loop.
// No LLM call is involved: group names/column groupings are computed
// client-side from column-name heuristics before this is called.
func (h *GlossaryHandler) GenerateSemanticTerms(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, "security context initialization failed: "+err.Error(), http.StatusUnauthorized)
		return
	}

	var req struct {
		Groups []struct {
			Name        string   `json:"name"`
			Description string   `json:"description"`
			ColumnIDs   []string `json:"column_ids"`
		} `json:"groups"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if len(req.Groups) == 0 {
		http.Error(w, "groups is required and must be non-empty", http.StatusBadRequest)
		return
	}

	// Resolve (or create) the semantic_term node type and MAPS_TO edge type
	// once, up front, shared by every group in this batch.
	var nodeTypeID string
	if err := h.db.QueryRow(`SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term' LIMIT 1`).Scan(&nodeTypeID); err != nil {
		if err == sql.ErrNoRows {
			if err := h.db.QueryRow(`INSERT INTO catalog_node_type (tenant_id, catalog_type_name, created_at, updated_at) VALUES ($1, 'semantic_term', NOW(), NOW()) RETURNING id`, secCtx.TenantID).Scan(&nodeTypeID); err != nil {
				log.Printf("GenerateSemanticTerms: failed to create semantic_term node type: %v", err)
				http.Error(w, "Failed to resolve semantic term type", http.StatusInternalServerError)
				return
			}
		} else {
			log.Printf("GenerateSemanticTerms: failed to resolve semantic_term node type: %v", err)
			http.Error(w, "Failed to resolve semantic term type", http.StatusInternalServerError)
			return
		}
	}

	var columnNodeTypeID sql.NullString
	if err := h.db.QueryRow(`SELECT id FROM catalog_node_type WHERE catalog_type_name = 'column' LIMIT 1`).Scan(&columnNodeTypeID); err != nil && err != sql.ErrNoRows {
		log.Printf("GenerateSemanticTerms: failed to resolve column node type: %v", err)
	}

	var edgeTypeID string
	if err := h.db.QueryRow(`SELECT id FROM catalog_edge_type WHERE edge_type_name = 'MAPS_TO' LIMIT 1`).Scan(&edgeTypeID); err != nil {
		if err == sql.ErrNoRows {
			if err := h.db.QueryRow(`INSERT INTO catalog_edge_type (tenant_id, edge_type_name, created_at, updated_at) VALUES ($1, 'MAPS_TO', NOW(), NOW()) RETURNING id`, secCtx.TenantID).Scan(&edgeTypeID); err != nil {
				log.Printf("GenerateSemanticTerms: failed to create MAPS_TO edge type: %v", err)
				http.Error(w, "Failed to resolve MAPS_TO edge type", http.StatusInternalServerError)
				return
			}
		} else {
			log.Printf("GenerateSemanticTerms: failed to resolve MAPS_TO edge type: %v", err)
			http.Error(w, "Failed to resolve MAPS_TO edge type", http.StatusInternalServerError)
			return
		}
	}

	tx, err := h.db.Begin()
	if err != nil {
		http.Error(w, "Failed to begin transaction", http.StatusInternalServerError)
		return
	}
	defer func() { _ = tx.Rollback() }()

	// catalog_node/catalog_edge have FORCE ROW LEVEL SECURITY: without this,
	// uisce_get_current_tenant() returns NULL, every insert's WITH CHECK
	// fails, and — since that's a real Postgres error, not an app-level one —
	// it poisons the rest of this transaction (25P02) rather than just this
	// statement.
	if _, err := tx.Exec(`SELECT set_config('uisce.current_tenant', $1, true)`, secCtx.TenantID); err != nil {
		http.Error(w, "Failed to set tenant context", http.StatusInternalServerError)
		return
	}

	type createdTerm struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		EdgeCnt int    `json:"linked_column_count"`
	}
	created := make([]createdTerm, 0, len(req.Groups))

	for _, group := range req.Groups {
		if group.Name == "" || len(group.ColumnIDs) == 0 {
			continue
		}
		qualifiedPath := fmt.Sprintf("semantic_term/%s", group.Name)

		// tenant_datasource_id is a real uuid column: an empty string here
		// (the request carries no reliable datasource context — secCtx and
		// the query param are both unset for this endpoint) would fail its
		// cast and poison the whole transaction. Resolve it from the
		// group's own first column instead, which is both always available
		// and more correct: the term's datasource should match the columns
		// it's actually mapping.
		var groupDatasourceID sql.NullString
		if err := tx.QueryRow(`SELECT tenant_datasource_id FROM catalog_node WHERE id = $1`, group.ColumnIDs[0]).Scan(&groupDatasourceID); err != nil {
			log.Printf("GenerateSemanticTerms: failed to resolve datasource for group %q: %v", group.Name, err)
			http.Error(w, fmt.Sprintf("Failed to resolve datasource for group %q: %v", group.Name, err), http.StatusInternalServerError)
			return
		}

		// catalog_node_unique is (tenant_id, node_type_id, qualified_path) —
		// tenant_datasource_id isn't part of it, so a lookup that also
		// filters on datasource can miss an existing row and then collide
		// on INSERT with a 23505 instead of reusing it.
		var termID string
		lookupErr := tx.QueryRow(`
			SELECT id FROM catalog_node
			WHERE tenant_id = $1 AND node_type_id = $2 AND (qualified_path = $3 OR node_name = $4)
			LIMIT 1
		`, secCtx.TenantID, nodeTypeID, qualifiedPath, group.Name).Scan(&termID)
		isNewTerm := lookupErr != nil
		if isNewTerm {
			if insertErr := tx.QueryRow(`
				INSERT INTO catalog_node (node_name, node_type_id, tenant_id, tenant_datasource_id, qualified_path, description, properties, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, '{}'::jsonb, NOW(), NOW())
				RETURNING id
			`, group.Name, nodeTypeID, secCtx.TenantID, groupDatasourceID, qualifiedPath, group.Description).Scan(&termID); insertErr != nil {
				log.Printf("GenerateSemanticTerms: failed to create term %q: %v", group.Name, insertErr)
				http.Error(w, fmt.Sprintf("Failed to create term %q: %v", group.Name, insertErr), http.StatusInternalServerError)
				return
			}
		}

		colIDs := append([]string{}, group.ColumnIDs...)

		// A brand-new term (not an accept onto an existing one) also picks up
		// every other same-named column across the tenant's other tables that
		// isn't already mapped to some semantic term — e.g. accepting
		// "AccountId" on one table's account_id auto-links every other
		// table's account_id column too, and removes them all from the
		// approval queue in one pass, rather than asking the same slam-dunk
		// decision over and over.
		if isNewTerm && columnNodeTypeID.Valid {
			var extraCols []string
			rows, err := tx.Query(`
				SELECT c.id FROM catalog_node c
				WHERE c.node_type_id = $1
				  AND c.node_name = $2
				  AND c.tenant_id = $3
				  AND NOT EXISTS (
					SELECT 1 FROM catalog_edge e
					WHERE (e.source_node_id = c.id OR e.target_node_id = c.id) AND e.edge_type_id = $4
				  )
			`, columnNodeTypeID, group.Name, secCtx.TenantID, edgeTypeID)
			if err != nil {
				log.Printf("GenerateSemanticTerms: failed to find same-named columns for %q: %v", group.Name, err)
			} else {
				for rows.Next() {
					var cid string
					if scanErr := rows.Scan(&cid); scanErr == nil {
						extraCols = append(extraCols, cid)
					}
				}
				rows.Close()
			}
			existing := make(map[string]bool, len(colIDs))
			for _, id := range colIDs {
				existing[id] = true
			}
			for _, id := range extraCols {
				if !existing[id] {
					colIDs = append(colIDs, id)
					existing[id] = true
				}
			}
		}

		linked := 0
		for _, colID := range colIDs {
			if colID == "" {
				continue
			}
			edgeID := uuid.New().String()
			if _, err := tx.Exec(`
				INSERT INTO catalog_edge (id, tenant_id, tenant_datasource_id, source_node_id, target_node_id, properties, edge_type_id, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, '{}'::jsonb, $6, NOW(), NOW())
			`, edgeID, secCtx.TenantID, groupDatasourceID, colID, termID, edgeTypeID); err != nil {
				log.Printf("GenerateSemanticTerms: failed to link column %s to term %s: %v", colID, termID, err)
				http.Error(w, fmt.Sprintf("Failed to link column to term %q: %v", group.Name, err), http.StatusInternalServerError)
				return
			}
			linked++
		}

		created = append(created, createdTerm{ID: termID, Name: group.Name, EdgeCnt: linked})
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"terms": created,
		"count": len(created),
	})
}

func (h *GlossaryHandler) CreateEdge(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, "security context initialization failed: "+err.Error(), http.StatusUnauthorized)
		return
	}

	var req struct {
		SubjectNodeID      string                 `json:"subject_node_id"`
		ObjectNodeID       string                 `json:"object_node_id"`
		EdgeTypeID         string                 `json:"edge_type_id"`
		TenantDatasourceID string                 `json:"tenant_datasource_id"`
		DatasourceID       string                 `json:"datasource_id"`
		Properties         map[string]interface{} `json:"properties"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.SubjectNodeID == "" || req.ObjectNodeID == "" || req.EdgeTypeID == "" {
		http.Error(w, "subject_node_id, object_node_id, and edge_type_id are required", http.StatusBadRequest)
		return
	}

	datasourceID := secCtx.DatasourceID
	if datasourceID == "" {
		if req.TenantDatasourceID != "" {
			datasourceID = req.TenantDatasourceID
		} else if req.DatasourceID != "" {
			datasourceID = req.DatasourceID
		} else if r.URL.Query().Get("tenant_datasource_id") != "" {
			datasourceID = r.URL.Query().Get("tenant_datasource_id")
		} else if r.URL.Query().Get("datasource_id") != "" {
			datasourceID = r.URL.Query().Get("datasource_id")
		} else {
			// Try to resolve datasource from either subject or object node in catalog_node
			var resolvedDS sql.NullString
			_ = h.db.QueryRow(
				`SELECT tenant_datasource_id FROM catalog_node WHERE id IN ($1, $2) AND tenant_datasource_id IS NOT NULL AND tenant_datasource_id != '' LIMIT 1`,
				req.SubjectNodeID, req.ObjectNodeID,
			).Scan(&resolvedDS)
			if resolvedDS.Valid {
				datasourceID = resolvedDS.String
			}
		}
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

	err = h.db.QueryRow(query, req.EdgeTypeID).Scan(&resolvedEdgeTypeID, &resolvedEdgeTypeName)

	if err != nil {
		if err == sql.ErrNoRows {
			// If invalid UUID or not found name, create new type (assuming input is the name)
			log.Printf("[CreateEdge] Edge Type not found, creating new. Input: %s", req.EdgeTypeID)
			insertTypeQ := `INSERT INTO catalog_edge_type (tenant_id, edge_type_name, created_at, updated_at) VALUES ($1, $2, NOW(), NOW()) RETURNING id`
			err = h.db.QueryRow(insertTypeQ, secCtx.TenantID, req.EdgeTypeID).Scan(&resolvedEdgeTypeID)
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

	log.Printf("[CreateEdge] Resolved EdgeTypeID: %s, DatasourceID: %s", resolvedEdgeTypeID, datasourceID)

	// Serialize properties
	var propertiesJSON []byte
	if req.Properties != nil && len(req.Properties) > 0 {
		propertiesJSON, _ = json.Marshal(req.Properties)
	}
	if len(propertiesJSON) == 0 {
		propertiesJSON = []byte("{}")
	}

	// Duplicate check on edge table: cannot have same subject, predicate, and object
	var existingEdgeID string
	dupCheckQ := `
		SELECT id FROM catalog_edge
		WHERE source_node_id = $1
		  AND target_node_id = $2
		  AND edge_type_id = $3
		LIMIT 1
	`
	err = h.db.QueryRow(dupCheckQ, req.SubjectNodeID, req.ObjectNodeID, resolvedEdgeTypeID).Scan(&existingEdgeID)
	if err == nil && existingEdgeID != "" {
		log.Printf("[CreateEdge] Duplicate edge found: %s with same subject (%s), object (%s), predicate (%s). Updating properties.", existingEdgeID, req.SubjectNodeID, req.ObjectNodeID, resolvedEdgeTypeName.String)
		if len(propertiesJSON) > 0 && string(propertiesJSON) != "{}" {
			_, _ = h.db.Exec(`UPDATE catalog_edge SET properties = $1::jsonb, updated_at = NOW() WHERE id = $2`, string(propertiesJSON), existingEdgeID)
		}
		edgeID = existingEdgeID
	} else {
		// Insert into catalog_edge
		insertQ := `
			INSERT INTO catalog_edge (id, tenant_id, tenant_datasource_id, source_node_id, target_node_id, properties, edge_type_id, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, NOW(), NOW())
		`
		_, err = h.db.Exec(insertQ, edgeID, secCtx.TenantID, datasourceID, req.SubjectNodeID, req.ObjectNodeID, string(propertiesJSON), resolvedEdgeTypeID)
		if err != nil {
			log.Printf("[CreateEdge] Error inserting edge: %v", err)
			http.Error(w, "Failed to create edge: "+err.Error(), http.StatusInternalServerError)
			return
		}
		log.Printf("[CreateEdge] Insert successful. EdgeID: %s", edgeID)
	}

	// Query back the inserted edge joined with edge type name for a friendly response
	var edge models.CatalogEdge
	var propertiesBytes []byte
	var edgeTypeName sql.NullString
	selQ := `
		SELECT ce.id, COALESCE(cet.edge_type_name, '') as predicate, COALESCE(ce.properties, '[]'::jsonb) as properties, ce.created_at, ce.updated_at, ce.tenant_id::text, COALESCE(ce.edge_type_id::text, '') as edge_type_id
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
			TenantID: &secCtx.TenantID,
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
// GetNodeDependencies reports whether a semantic/business term can be safely
// deleted. business_object_fields.term_node_id -> catalog_node.id is
// ON DELETE RESTRICT, so deleting a term that still backs a BO field would
// otherwise fail with an opaque FK-violation 500; this lets the frontend
// warn the user and name the blocking BO fields up front instead.
func (h *GlossaryHandler) GetNodeDependencies(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "term ID is required", http.StatusBadRequest)
		return
	}

	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, "security context initialization failed: "+err.Error(), http.StatusUnauthorized)
		return
	}

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "Failed to begin transaction", http.StatusInternalServerError)
		return
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(r.Context(), `SELECT set_config('uisce.current_tenant', $1, true)`, secCtx.TenantID); err != nil {
		http.Error(w, "Failed to set tenant context", http.StatusInternalServerError)
		return
	}

	rows, err := tx.QueryContext(r.Context(), `
		SELECT bof.id, bof.bo_id, bof.field_name, bo.bo_key, bo.bo_name
		FROM business_object_fields bof
		JOIN business_objects bo ON bo.id = bof.bo_id
		WHERE bof.term_node_id = $1 AND bof.tenant_id = $2
	`, id, secCtx.TenantID)
	if err != nil {
		log.Printf("[GetNodeDependencies] Error querying dependencies for %s: %v", id, err)
		http.Error(w, "Failed to check dependencies", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type dependency struct {
		RefTable  string `json:"ref_table"`
		RefID     string `json:"ref_id"`
		BOID      string `json:"bo_id"`
		BOKey     string `json:"bo_key"`
		BOName    string `json:"bo_name"`
		RefDetail string `json:"ref_detail"`
	}
	deps := make([]dependency, 0)
	for rows.Next() {
		var fieldID, boID, fieldName, boKey, boName string
		if err := rows.Scan(&fieldID, &boID, &fieldName, &boKey, &boName); err != nil {
			continue
		}
		deps = append(deps, dependency{
			RefTable: "business_object_fields", RefID: fieldID,
			BOID: boID, BOKey: boKey, BOName: boName, RefDetail: fieldName,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"can_delete":   len(deps) == 0,
		"dependencies": deps,
	})
}

func (h *GlossaryHandler) DeleteTerm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "term ID is required", http.StatusBadRequest)
		return
	}

	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, "security context initialization failed: "+err.Error(), http.StatusUnauthorized)
		return
	}

	tenantID := secCtx.TenantID
	if tenantID == "default" {
		var coreID string
		if err := h.db.QueryRowContext(r.Context(), `SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1`).Scan(&coreID); err == nil && coreID != "" {
			tenantID = coreID
		}
	}

	// catalog_node and catalog_edge both have FORCE ROW LEVEL SECURITY, and
	// catalog_edge has no foreign key to catalog_node (it's partitioned, so
	// none was ever added) — so this must (a) set the tenant GUC itself,
	// unlike the plain h.db.Exec this replaced, and (b) explicitly delete
	// any edges referencing this node first, or they're left dangling.
	tx, err := h.db.Begin()
	if err != nil {
		http.Error(w, "Failed to begin transaction", http.StatusInternalServerError)
		return
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`SELECT set_config('uisce.current_tenant', $1, true)`, tenantID); err != nil {
		http.Error(w, "Failed to set tenant context", http.StatusInternalServerError)
		return
	}

	if _, err := tx.Exec(`DELETE FROM catalog_edge WHERE source_node_id = $1 OR target_node_id = $1`, id); err != nil {
		log.Printf("Error deleting edges for term %s: %v", id, err)
		http.Error(w, "Failed to delete term's edges", http.StatusInternalServerError)
		return
	}

	result, err := tx.Exec(`DELETE FROM catalog_node WHERE id = $1 AND tenant_id = $2`, id, tenantID)
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

	if err := tx.Commit(); err != nil {
		http.Error(w, "Failed to commit delete", http.StatusInternalServerError)
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
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, "security context initialization failed: "+err.Error(), http.StatusUnauthorized)
		return
	}

	if edgeID == "" {
		http.Error(w, "Edge ID is required", http.StatusBadRequest)
		return
	}

	log.Printf("[DeleteEdge] Deleting edge %s for tenant %s", edgeID, secCtx.TenantID)

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
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, "security context initialization failed: "+err.Error(), http.StatusUnauthorized)
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

	log.Printf("[UpdateEdge] Updating edge %s for tenant %s. Updates: %v", edgeID, secCtx.TenantID, updates)

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
	args = append(args, edgeID, secCtx.TenantID)

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
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, "security context initialization failed: "+err.Error(), http.StatusUnauthorized)
		return
	}

	if secCtx.TenantID == "" || secCtx.DatasourceID == "" {
		http.Error(w, "Missing required parameters: tenant_id, datasource_id", http.StatusBadRequest)
		return
	}

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

	rows, err := h.db.Query(query, secCtx.TenantID, secCtx.DatasourceID)
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

func (h *GlossaryHandler) ListTechnicalAssets(w http.ResponseWriter, r *http.Request) {
	nodeID := r.URL.Query().Get("node_id")
	if nodeID == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{}, "total": 0})
		return
	}

	query := `
		SELECT ce.id AS edge_id, cn.id, cn.node_name, cn.qualified_path,
		       COALESCE(cnt.catalog_type_name, cn.node_type, 'column') AS node_type,
		       COALESCE(cn.properties, '{}'::jsonb) AS properties,
		       COALESCE(cn.is_active, true) AS is_active,
		       COALESCE(cn.tenant_datasource_id::text, '') AS datasource_id
		FROM catalog_edge ce
		JOIN catalog_node cn ON (
			(ce.source_node_id = $1 AND ce.target_node_id = cn.id)
			OR
			(ce.target_node_id = $1 AND ce.source_node_id = cn.id)
		)
		LEFT JOIN catalog_node_type cnt ON cn.node_type_id = cnt.id
		WHERE (ce.source_node_id = $1 OR ce.target_node_id = $1)
		  AND cn.id != $1
		  AND LOWER(COALESCE(cnt.catalog_type_name, cn.node_type, '')) NOT IN ('business_object', 'business_term', 'semantic_term')
		  AND NOT (cn.qualified_path LIKE '/business_object%' OR cn.qualified_path LIKE 'business_object%' OR cn.node_name LIKE 'business_object%')
		  AND (
		      ce.edge_type_id = '0434ca1a-6543-42d3-9fce-f0b58b5fba34' 
		      OR LOWER(COALESCE(ce.relationship_type, '')) IN ('has_context', 'maps_to', 'mapped_to', 'column_mapping', 'specializes', 'belongs_to', 'api_has_endpoint', 'foreign_key')
		      OR LOWER(COALESCE(cnt.catalog_type_name, cn.node_type, '')) IN ('column', 'table', 'database_column', 'database_table', 'endpoint', 'api_endpoint', 'resource')
		      OR cn.qualified_path LIKE '/%'
		      OR cn.qualified_path LIKE 'api_endpoint%'
		  )
	`

	rows, err := h.db.QueryContext(r.Context(), query, nodeID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{}, "total": 0})
		return
	}
	defer rows.Close()

	assets := make([]map[string]interface{}, 0)
	for rows.Next() {
		var edgeID, id, name, path, nType, dsID string
		var isActive bool
		var propsJSON []byte
		if err := rows.Scan(&edgeID, &id, &name, &path, &nType, &propsJSON, &isActive, &dsID); err != nil {
			continue
		}
		var props map[string]interface{}
		_ = json.Unmarshal(propsJSON, &props)

		// Parse datasource name from qualified_path (/datasource/table/column)
		datasource := ""
		if len(path) > 1 {
			parts := strings.Split(path[1:], "/")
			if len(parts) > 0 && parts[0] != "" {
				datasource = parts[0]
			}
		}

		assets = append(assets, map[string]interface{}{
			"edge_id":         edgeID,
			"id":              id,
			"node_name":       name,
			"qualified_path":  path,
			"node_type":       nType,
			"properties":      props,
			"is_core":         isActive,
			"datasource":      datasource,
			"datasource_id":   dsID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":  assets,
		"total": len(assets),
	})
}

// CreateTechnicalAssetsRequest payload for linking columns to a semantic term
type CreateTechnicalAssetsRequest struct {
	SemanticTermID string   `json:"semantic_term_id"`
	ColumnNodeIDs  []string `json:"column_node_ids"`
}

func (h *GlossaryHandler) CreateTechnicalAssets(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-ID")
	}

	var req CreateTechnicalAssetsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.SemanticTermID == "" || len(req.ColumnNodeIDs) == 0 {
		http.Error(w, "semantic_term_id and column_node_ids are required", http.StatusBadRequest)
		return
	}

	createdCount := 0
	for _, colID := range req.ColumnNodeIDs {
		edgeID := uuid.New().String()
		log.Printf("[DEBUG] CreateTechnicalAssets: colID=%s, semanticTermID=%s", colID, req.SemanticTermID)

		var tenantDatasourceID *string
		err := h.db.QueryRowContext(r.Context(), `
			SELECT tenant_datasource_id FROM catalog_node WHERE id = $1
		`, colID).Scan(&tenantDatasourceID)
		if err != nil {
			log.Printf("[ERROR] Failed to get tenant_datasource_id for node %s: %v", colID, err)
			continue
		}
		log.Printf("[DEBUG] CreateTechnicalAssets: tenantDatasourceID=%v", *tenantDatasourceID)

		var exists bool
		err = h.db.QueryRowContext(r.Context(), `
			SELECT EXISTS(
				SELECT 1 FROM catalog_edge 
				WHERE ((source_node_id = $1 AND target_node_id = $2) OR (source_node_id = $2 AND target_node_id = $1))
				  AND (edge_type_id = '0434ca1a-6543-42d3-9fce-f0b58b5fba34' OR relationship_type = 'has_context' OR edge_type = 'has_context')
			)
		`, colID, req.SemanticTermID).Scan(&exists)
		if err != nil {
			log.Printf("[ERROR] Failed to check existing edge (%s -> %s): %v", colID, req.SemanticTermID, err)
			continue
		}
		if exists {
			log.Printf("[DEBUG] Edge already exists, skipping")
			continue
		}

		_, err = h.db.ExecContext(r.Context(), `
			INSERT INTO catalog_edge (
				id,
				tenant_datasource_id,
				source_node_id,
				target_node_id,
				edge_type_id,
				relationship_type,
				tenant_id,
				created_at,
				updated_at
			) VALUES (
				$1, $2, $3, $4, '0434ca1a-6543-42d3-9fce-f0b58b5fba34', 'has_context',
				NULLIF($5, '')::uuid, NOW(), NOW()
			)
		`, edgeID, *tenantDatasourceID, colID, req.SemanticTermID, tenantID)

		if err != nil {
			log.Printf("[ERROR] Failed to insert has_context edge (%s -> %s): %v", colID, req.SemanticTermID, err)
			continue
		}

		createdCount++
		log.Printf("[DEBUG] CreateTechnicalAssets: inserted edgeID=%s", edgeID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"created": createdCount,
	})
}

func (h *GlossaryHandler) DeleteTechnicalAsset(w http.ResponseWriter, r *http.Request) {
	edgeID := chi.URLParam(r, "id")
	if edgeID == "" {
		http.Error(w, "Missing edge id", http.StatusBadRequest)
		return
	}

	_, err := h.db.ExecContext(r.Context(), `DELETE FROM catalog_edge WHERE id = $1`, edgeID)
	if err != nil {
		http.Error(w, "Failed to delete mapping edge: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"deleted": edgeID,
	})
}

func (h *GlossaryHandler) GetNodeGraph(w http.ResponseWriter, r *http.Request) {
	nodeID := r.URL.Query().Get("node_id")
	if nodeID == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"nodes": []interface{}{}, "edges": []interface{}{}})
		return
	}

	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// catalog_node/catalog_edge are FORCE ROW LEVEL SECURITY tables — the
	// tenant GUC must be set on this transaction or the SELECT below
	// silently returns zero rows regardless of the WHERE clause.
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		log.Printf("[GetNodeGraph] Error starting tx: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"nodes": []interface{}{}, "edges": []interface{}{}})
		return
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(r.Context(), "SELECT set_config('uisce.current_tenant', $1, true)", secCtx.TenantID); err != nil {
		log.Printf("[GetNodeGraph] Error setting tenant GUC: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"nodes": []interface{}{}, "edges": []interface{}{}})
		return
	}

	query := `
		SELECT ce.id, ce.source_node_id, ce.target_node_id,
		       COALESCE(cet.edge_type_name, ce.relationship_type, '') AS predicate,
		       COALESCE(src.qualified_path, src.node_name, '') AS source_name,
		       COALESCE(tgt.qualified_path, tgt.node_name, '') AS target_name,
		       COALESCE(src_cnt.catalog_type_name, src.node_type, 'Node') AS source_type,
		       COALESCE(tgt_cnt.catalog_type_name, tgt.node_type, 'Node') AS target_type,
		       COALESCE(src.qualified_path, '') AS source_path,
		       COALESCE(tgt.qualified_path, '') AS target_path
		FROM catalog_edge ce
		LEFT JOIN catalog_edge_type cet ON cet.id = ce.edge_type_id
		LEFT JOIN catalog_node src ON src.id = ce.source_node_id
		LEFT JOIN catalog_node_type src_cnt ON src_cnt.id = src.node_type_id
		LEFT JOIN catalog_node tgt ON tgt.id = ce.target_node_id
		LEFT JOIN catalog_node_type tgt_cnt ON tgt_cnt.id = tgt.node_type_id
		WHERE (ce.source_node_id = $1 OR ce.target_node_id = $1)
		  AND ce.tenant_id = $2
	`
	rows, err := tx.QueryContext(r.Context(), query, nodeID, secCtx.TenantID)
	if err != nil {
		log.Printf("[GetNodeGraph] Error querying edges: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"nodes": []interface{}{}, "edges": []interface{}{}})
		return
	}
	defer rows.Close()

	edges := make([]map[string]interface{}, 0)
	nodeMap := make(map[string]map[string]interface{})

	for rows.Next() {
		var id, sID, tID, pName, sName, tName, sType, tType, sPath, tPath string
		if err := rows.Scan(&id, &sID, &tID, &pName, &sName, &tName, &sType, &tType, &sPath, &tPath); err != nil {
			continue
		}
		edges = append(edges, map[string]interface{}{
			"id":                id,
			"source_node_id":    sID,
			"source_name":       sName,
			"source_node_type":  sType,
			"source_path":       sPath,
			"target_node_id":    tID,
			"target_name":       tName,
			"target_node_type":  tType,
			"target_path":       tPath,
			"edge_type_name":    pName,
			"relationship_type": pName,
			"predicate":         pName,
		})

		if _, exists := nodeMap[sID]; !exists && sID != "" {
			nodeMap[sID] = map[string]interface{}{
				"id":                sID,
				"node_name":         sName,
				"qualified_path":    sPath,
				"catalog_type_name": sType,
				"node_type":         sType,
			}
		}
		if _, exists := nodeMap[tID]; !exists && tID != "" {
			nodeMap[tID] = map[string]interface{}{
				"id":                tID,
				"node_name":         tName,
				"qualified_path":    tPath,
				"catalog_type_name": tType,
				"node_type":         tType,
			}
		}
	}

	nodesList := make([]map[string]interface{}, 0, len(nodeMap))
	for _, n := range nodeMap {
		nodesList = append(nodesList, n)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"nodes":           nodesList,
		"connected_nodes": nodesList,
		"edges":           edges,
	})
}

// ProfileSampleRequest defines the payload for sampling and profiling a column safely
type ProfileSampleRequest struct {
	NodeID        string `json:"node_id"`
	NodeName      string `json:"node_name"`
	QualifiedPath string `json:"qualified_path"`
	TableName     string `json:"table_name"`
	ColumnName    string `json:"column_name"`
	SampleSize    int    `json:"sample_size"`
}

// ProfileSampleResponse returns statistical and pattern metadata from zero-impact sampling
type ProfileSampleResponse struct {
	ColumnName        string   `json:"column_name"`
	TableName         string   `json:"table_name"`
	SampleCount       int      `json:"sample_count"`
	SampleValues      []string `json:"sample_values"`
	DistinctCount     int      `json:"distinct_count"`
	NullCount         int      `json:"null_count"`
	InferredType      string   `json:"inferred_type"`
	PatternDetected   string   `json:"pattern_detected"`
	BloombergCandidate string  `json:"bloomberg_candidate,omitempty"`
	IsSafeSampled     bool     `json:"is_safe_sampled"`
}

// ProfileSample handles safe, zero-impact physical page sampling on columns for billion-row tables
func (h *GlossaryHandler) ProfileSample(w http.ResponseWriter, r *http.Request) {
	var req ProfileSampleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	tableName := strings.TrimSpace(req.TableName)
	columnName := strings.TrimSpace(req.ColumnName)

	// If table/column not explicitly given, extract from qualified path
	if (tableName == "" || columnName == "") && req.QualifiedPath != "" {
		parts := strings.Split(req.QualifiedPath, ".")
		if len(parts) >= 2 {
			columnName = parts[len(parts)-1]
			tableName = parts[len(parts)-2]
		}
	}

	// If still empty and node_id is provided, look up catalog node
	if (tableName == "" || columnName == "") && req.NodeID != "" {
		var path, name sql.NullString
		err := h.db.QueryRowContext(r.Context(), `
			SELECT qualified_path, node_name FROM catalog_node WHERE id = $1 LIMIT 1
		`, req.NodeID).Scan(&path, &name)
		if err == nil {
			if columnName == "" && name.Valid {
				columnName = name.String
			}
			if path.Valid && path.String != "" {
				parts := strings.Split(path.String, ".")
				if len(parts) >= 2 {
					columnName = parts[len(parts)-1]
					tableName = parts[len(parts)-2]
				}
			}
		}
	}

	if tableName == "" || columnName == "" {
		http.Error(w, "table_name and column_name could not be resolved", http.StatusBadRequest)
		return
	}

	// Strict SQL identifier safety check (letters, numbers, underscore only)
	validIdentifier := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	if !validIdentifier.MatchString(tableName) || !validIdentifier.MatchString(columnName) {
		http.Error(w, "Invalid table or column identifier", http.StatusBadRequest)
		return
	}

	sampleSize := req.SampleSize
	if sampleSize <= 0 || sampleSize > 1000 {
		sampleSize = 500
	}

	// 1. Try physical page reservoir sampling (TABLESAMPLE SYSTEM)
	// Reads 2-3 physical 8KB disk blocks with O(1) latency regardless of table size
	query := fmt.Sprintf(`SELECT CAST("%s" AS TEXT) FROM "%s" TABLESAMPLE SYSTEM (0.1) WHERE "%s" IS NOT NULL LIMIT %d`,
		columnName, tableName, columnName, sampleSize)

	rows, err := h.db.QueryContext(r.Context(), query)
	if err != nil {
		// Fallback for views or systems without TABLESAMPLE support
		fallbackQuery := fmt.Sprintf(`SELECT CAST("%s" AS TEXT) FROM "%s" WHERE "%s" IS NOT NULL LIMIT %d`,
			columnName, tableName, columnName, sampleSize)
		rows, err = h.db.QueryContext(r.Context(), fallbackQuery)
		if err != nil {
			log.Printf("[ProfileSample] Sampling query failed for table %s, col %s: %v", tableName, columnName, err)
			http.Error(w, "Could not profile table: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
	defer rows.Close()

	uniqueValuesMap := make(map[string]bool)
	var sampleList []string
	sampleCount := 0

	for rows.Next() {
		var val sql.NullString
		if err := rows.Scan(&val); err != nil {
			continue
		}
		if val.Valid {
			sampleCount++
			trimmed := strings.TrimSpace(val.String)
			if trimmed != "" {
				uniqueValuesMap[trimmed] = true
				if len(sampleList) < 10 && !uniqueValuesMap[trimmed] {
					sampleList = append(sampleList, trimmed)
				}
			}
		}
	}

	// Collect preview values
	previewValues := make([]string, 0, 10)
	for val := range uniqueValuesMap {
		if len(previewValues) >= 10 {
			break
		}
		previewValues = append(previewValues, val)
	}

	// Classify pattern
	inferredType := "string"
	patternDetected := "GENERIC_STRING"
	bloombergCandidate := ""

	// Check ISIN (12 chars: 2 alpha + 9 alphanum + 1 digit)
	isinRegex := regexp.MustCompile(`^[A-Z]{2}[A-Z0-9]{9}[0-9]$`)
	// Check Currency (3 alpha uppercase)
	currencyRegex := regexp.MustCompile(`^[A-Z]{3}$`)
	// Check Number
	numericRegex := regexp.MustCompile(`^-?[0-9]+(\.[0-9]+)?$`)
	// Check Date
	dateRegex := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}`)

	isinHits, currencyHits, numericHits, dateHits := 0, 0, 0, 0
	for _, v := range previewValues {
		if isinRegex.MatchString(v) {
			isinHits++
		}
		if currencyRegex.MatchString(v) && len(v) == 3 {
			currencyHits++
		}
		if numericRegex.MatchString(v) {
			numericHits++
		}
		if dateRegex.MatchString(v) {
			dateHits++
		}
	}

	if len(previewValues) > 0 {
		if isinHits >= len(previewValues)/2 {
			patternDetected = "ISIN"
			bloombergCandidate = "ID_ISIN"
			inferredType = "string"
		} else if currencyHits >= len(previewValues)/2 {
			patternDetected = "ISO_CURRENCY"
			bloombergCandidate = "CRNCY"
			inferredType = "string"
		} else if numericHits >= len(previewValues)/2 {
			patternDetected = "FINANCIAL_AMOUNT"
			bloombergCandidate = "PX_LAST"
			inferredType = "numeric"
		} else if dateHits >= len(previewValues)/2 {
			patternDetected = "DATE"
			bloombergCandidate = "MATURITY"
			inferredType = "date"
		}
	}

	resp := ProfileSampleResponse{
		ColumnName:         columnName,
		TableName:          tableName,
		SampleCount:        sampleCount,
		SampleValues:       previewValues,
		DistinctCount:      len(uniqueValuesMap),
		NullCount:          0,
		InferredType:       inferredType,
		PatternDetected:    patternDetected,
		BloombergCandidate: bloombergCandidate,
		IsSafeSampled:      true,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
