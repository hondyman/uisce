package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ============================================================================
// Semantic Layer Models
// ============================================================================

type SemanticAsset struct {
	ID               string         `json:"id" db:"id"`
	TenantID         string         `json:"tenant_id" db:"tenant_id"`
	DatasourceID     string         `json:"datasource_id" db:"datasource_id"`
	BusinessEntityID string         `json:"business_entity_id" db:"business_entity_id"`
	CoreModelID      *string        `json:"core_model_id" db:"core_model_id"`
	CoreViewID       *string        `json:"core_view_id" db:"core_view_id"`
	CustomModelID    *string        `json:"custom_model_id" db:"custom_model_id"`
	CustomViewID     *string        `json:"custom_view_id" db:"custom_view_id"`
	SourceTables     pq.StringArray `json:"source_tables" db:"source_tables"`
	CreatedAt        time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at" db:"updated_at"`
}

// RelationshipSuggestionRecord is defined in relationships_chi.go to keep
// relationship-related models and handlers together.

// ============================================================================
// Semantic Layer Routes
// ============================================================================

// RegisterSemanticLayerRoutes registers all semantic layer routes
func (s *Server) RegisterSemanticLayerRoutes(router chi.Router) {
	router.Route("/business-entities", func(r chi.Router) {
		// Core model/view generation
		r.Post("/{entityID}/generate-core-model", s.handleGenerateCoreModel)
		r.Post("/{entityID}/generate-core-view", s.handleGenerateCoreView)
		r.Post("/{entityID}/create-custom-model", s.handleCreateCustomModel)
		r.Post("/{entityID}/create-custom-view", s.handleCreateCustomView)

		// Retrieve semantic assets
		r.Get("/{entityID}/semantic-assets", s.handleGetSemanticAssets)

		// Get related objects (semantic model relationships)
		r.Get("/{entityID}/related-objects", s.handleGetRelatedSemanticObjects)

		// Graph traversal
		r.Post("/{entityID}/traverse-graph", s.handleTraverseObjectGraph)
	})

	// Register semantic bundle routes to fix 404s
	// MOVED TO SemanticLayerHandler.RegisterRoutes to avoid double-mount panic on /semantic path
	// router.Route("/semantic", func(r chi.Router) { ... })
}

// ============================================================================
// Request Types
// ============================================================================

type GenerateCoreModelRequest struct {
	ModelName  string   `json:"model_name"`
	SourceKeys []string `json:"source_keys"`
}

type GenerateCoreViewRequest struct {
	ViewName        string   `json:"view_name"`
	SelectedColumns []string `json:"selected_columns"`
}

type CreateCustomModelRequest struct {
	ModelName  string   `json:"model_name"`
	Expression string   `json:"expression"`
	SourceKeys []string `json:"source_keys"`
}

type CreateCustomViewRequest struct {
	ViewName   string   `json:"view_name"`
	Expression string   `json:"expression"`
	SourceKeys []string `json:"source_keys"`
}

type ApplyRelationshipSuggestionRequest struct {
	SuggestionID string `json:"suggestion_id"`
}

type TraverseGraphRequest struct {
	StartNodeID string `json:"start_node_id"`
	DotPath     string `json:"dot_path"`
}

// ============================================================================
// Handlers
// ============================================================================

// handleGenerateCoreModel generates a core model from source tables
func (s *Server) handleGenerateCoreModel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	db := s.DB

	// Extract tenant context
	tenantContext, err := extractTenantContext(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("missing tenant context: %v", err), http.StatusBadRequest)
		return
	}

	entityID := chi.URLParam(r, "entityID")
	if entityID == "" {
		writeJSONError(w, http.StatusBadRequest, "Entity ID required", "invalid_request", "")
		return
	}

	var req GenerateCoreModelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body", "invalid_request", err.Error())
		return
	}

	if req.ModelName == "" {
		writeJSONError(w, http.StatusBadRequest, "Model name required", "invalid_request", "")
		return
	}

	// Create catalog node for core model
	modelID := uuid.New().String()
	now := time.Now()

	insertNodeQuery := `
		INSERT INTO catalog_node (
			id, tenant_id, datasource_id, node_name, node_type, 
			node_description, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	var createdID string
	err = db.QueryRowContext(ctx, insertNodeQuery,
		modelID, tenantContext.TenantID, tenantContext.DatasourceID,
		req.ModelName, "model", "Auto-generated core model", now, now,
	).Scan(&createdID)

	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to create model", "db_error", err.Error())
		return
	}

	// Update semantic assets with core model link
	assetID := uuid.New().String()
	updateAssetQuery := `
		INSERT INTO semantic_assets (
			id, tenant_id, datasource_id, business_entity_id, core_model_id, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (tenant_id, datasource_id, business_entity_id) 
		DO UPDATE SET core_model_id = $5, updated_at = $6
	`

	_, err = db.ExecContext(ctx, updateAssetQuery,
		assetID, tenantContext.TenantID, tenantContext.DatasourceID,
		entityID, createdID, now, now,
	)

	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to update semantic assets", "db_error", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"model_id":   createdID,
		"model_name": req.ModelName,
	})
}

// handleGenerateCoreView generates a core view from selected columns
func (s *Server) handleGenerateCoreView(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	db := s.DB

	// Extract tenant context
	tenantContext, err := extractTenantContext(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("missing tenant context: %v", err), http.StatusBadRequest)
		return
	}

	entityID := chi.URLParam(r, "entityID")
	if entityID == "" {
		writeJSONError(w, http.StatusBadRequest, "Entity ID required", "invalid_request", "")
		return
	}

	var req GenerateCoreViewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body", "invalid_request", err.Error())
		return
	}

	if req.ViewName == "" {
		writeJSONError(w, http.StatusBadRequest, "View name required", "invalid_request", "")
		return
	}

	// Create catalog node for core view
	viewID := uuid.New().String()
	now := time.Now()

	insertNodeQuery := `
		INSERT INTO catalog_node (
			id, tenant_id, datasource_id, node_name, node_type, 
			node_description, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	var createdID string
	err = db.QueryRowContext(ctx, insertNodeQuery,
		viewID, tenantContext.TenantID, tenantContext.DatasourceID,
		req.ViewName, "view", "Auto-generated core view", now, now,
	).Scan(&createdID)

	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to create view", "db_error", err.Error())
		return
	}

	// Update semantic assets with core view link
	assetID := uuid.New().String()
	updateAssetQuery := `
		INSERT INTO semantic_assets (
			id, tenant_id, datasource_id, business_entity_id, core_view_id, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (tenant_id, datasource_id, business_entity_id) 
		DO UPDATE SET core_view_id = $5, updated_at = $6
	`

	_, err = db.ExecContext(ctx, updateAssetQuery,
		assetID, tenantContext.TenantID, tenantContext.DatasourceID,
		entityID, createdID, now, now,
	)

	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to update semantic assets", "db_error", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"view_id":   createdID,
		"view_name": req.ViewName,
	})
}

// handleCreateCustomModel creates a custom model with expression
func (s *Server) handleCreateCustomModel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	db := s.DB

	// Extract tenant context
	tenantContext, err := extractTenantContext(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("missing tenant context: %v", err), http.StatusBadRequest)
		return
	}

	entityID := chi.URLParam(r, "entityID")
	if entityID == "" {
		writeJSONError(w, http.StatusBadRequest, "Entity ID required", "invalid_request", "")
		return
	}

	var req CreateCustomModelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body", "invalid_request", err.Error())
		return
	}

	if req.ModelName == "" || req.Expression == "" {
		writeJSONError(w, http.StatusBadRequest, "Model name and expression required", "invalid_request", "")
		return
	}

	// Create catalog node for custom model
	modelID := uuid.New().String()
	now := time.Now()
	description := fmt.Sprintf("Custom model: %s", req.Expression)

	insertNodeQuery := `
		INSERT INTO catalog_node (
			id, tenant_id, datasource_id, node_name, node_type, 
			node_description, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	var createdID string
	err = db.QueryRowContext(ctx, insertNodeQuery,
		modelID, tenantContext.TenantID, tenantContext.DatasourceID,
		req.ModelName, "model", description, now, now,
	).Scan(&createdID)

	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to create custom model", "db_error", err.Error())
		return
	}

	// Update semantic assets with custom model link
	assetID := uuid.New().String()
	updateAssetQuery := `
		INSERT INTO semantic_assets (
			id, tenant_id, datasource_id, business_entity_id, custom_model_id, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (tenant_id, datasource_id, business_entity_id) 
		DO UPDATE SET custom_model_id = $5, updated_at = $6
	`

	_, err = db.ExecContext(ctx, updateAssetQuery,
		assetID, tenantContext.TenantID, tenantContext.DatasourceID,
		entityID, createdID, now, now,
	)

	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to update semantic assets", "db_error", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"model_id":   createdID,
		"model_name": req.ModelName,
		"expression": req.Expression,
	})
}

// handleCreateCustomView creates a custom view with expression
func (s *Server) handleCreateCustomView(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	db := s.DB

	// Extract tenant context
	tenantContext, err := extractTenantContext(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("missing tenant context: %v", err), http.StatusBadRequest)
		return
	}

	entityID := chi.URLParam(r, "entityID")
	if entityID == "" {
		writeJSONError(w, http.StatusBadRequest, "Entity ID required", "invalid_request", "")
		return
	}

	var req CreateCustomViewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body", "invalid_request", err.Error())
		return
	}

	if req.ViewName == "" || req.Expression == "" {
		writeJSONError(w, http.StatusBadRequest, "View name and expression required", "invalid_request", "")
		return
	}

	// Create catalog node for custom view
	viewID := uuid.New().String()
	now := time.Now()
	description := fmt.Sprintf("Custom view: %s", req.Expression)

	insertNodeQuery := `
		INSERT INTO catalog_node (
			id, tenant_id, datasource_id, node_name, node_type, 
			node_description, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	var createdID string
	err = db.QueryRowContext(ctx, insertNodeQuery,
		viewID, tenantContext.TenantID, tenantContext.DatasourceID,
		req.ViewName, "view", description, now, now,
	).Scan(&createdID)

	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to create custom view", "db_error", err.Error())
		return
	}

	// Update semantic assets with custom view link
	assetID := uuid.New().String()
	updateAssetQuery := `
		INSERT INTO semantic_assets (
			id, tenant_id, datasource_id, business_entity_id, custom_view_id, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (tenant_id, datasource_id, business_entity_id) 
		DO UPDATE SET custom_view_id = $5, updated_at = $6
	`

	_, err = db.ExecContext(ctx, updateAssetQuery,
		assetID, tenantContext.TenantID, tenantContext.DatasourceID,
		entityID, createdID, now, now,
	)

	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to update semantic assets", "db_error", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"view_id":    createdID,
		"view_name":  req.ViewName,
		"expression": req.Expression,
	})
}

// handleGetSemanticAssets retrieves semantic assets for an entity
func (s *Server) handleGetSemanticAssets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	db := s.DB

	// Extract tenant context
	tenantContext, err := extractTenantContext(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("missing tenant context: %v", err), http.StatusBadRequest)
		return
	}

	entityID := chi.URLParam(r, "entityID")
	if entityID == "" {
		writeJSONError(w, http.StatusBadRequest, "Entity ID required", "invalid_request", "")
		return
	}

	query := `
		SELECT id, tenant_id, datasource_id, business_entity_id, 
		       core_model_id, core_view_id, custom_model_id, custom_view_id,
		       source_tables, created_at, updated_at
		FROM semantic_assets
		WHERE tenant_id = $1 AND datasource_id = $2 AND business_entity_id = $3
	`

	var asset SemanticAsset
	err = db.QueryRowContext(ctx, query,
		tenantContext.TenantID, tenantContext.DatasourceID, entityID,
	).Scan(
		&asset.ID, &asset.TenantID, &asset.DatasourceID, &asset.BusinessEntityID,
		&asset.CoreModelID, &asset.CoreViewID, &asset.CustomModelID, &asset.CustomViewID,
		&asset.SourceTables, &asset.CreatedAt, &asset.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// Create empty record if doesn't exist
		asset.ID = uuid.New().String()
		asset.TenantID = tenantContext.TenantID
		asset.DatasourceID = tenantContext.DatasourceID
		asset.BusinessEntityID = entityID
		asset.CreatedAt = time.Now()
		asset.UpdatedAt = time.Now()

		insertQuery := `
			INSERT INTO semantic_assets (id, tenant_id, datasource_id, business_entity_id, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (tenant_id, datasource_id, business_entity_id) DO NOTHING
		`
		db.ExecContext(ctx, insertQuery, asset.ID, asset.TenantID, asset.DatasourceID, asset.BusinessEntityID, asset.CreatedAt, asset.UpdatedAt)
	} else if err != nil {
		// If table doesn't exist or other DB error, return empty asset gracefully
		// This allows the feature to degrade gracefully if semantic_assets table is not yet created
		asset.ID = uuid.New().String()
		asset.TenantID = tenantContext.TenantID
		asset.DatasourceID = tenantContext.DatasourceID
		asset.BusinessEntityID = entityID
		asset.CreatedAt = time.Now()
		asset.UpdatedAt = time.Now()
		// Don't return error - just return empty asset
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(asset)
}

func (s *Server) getAppliedRelationships(ctx context.Context, db *sql.DB, tenantID, datasourceID, entityID string) ([]map[string]interface{}, error) {
	query := `
		SELECT
			bor.id,
			bor.source_object_id,
			bor.target_object_id,
			bor.cardinality,
			bor.relationship_type,
			bor.description,
			bo_source.display_name as source_name,
			bo_target.display_name as target_name
		FROM business_object_relationships bor
		LEFT JOIN business_objects bo_source ON bo_source.id = bor.source_object_id
		  AND bo_source.tenant_id = bor.tenant_id
		LEFT JOIN business_objects bo_target ON bo_target.id = bor.target_object_id
		  AND bo_target.tenant_id = bor.tenant_id
		WHERE bor.tenant_id = $1 AND bor.source_object_id = $2::uuid AND bor.is_user_applied = true
	`
	rows, err := db.QueryContext(ctx, query, tenantID, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var objects []map[string]interface{}
	for rows.Next() {
		var (
			id           string
			sourceNodeID string
			targetNodeID string
			cardinality  sql.NullString
			edgeType     sql.NullString
			description  sql.NullString
			sourceName   sql.NullString
			targetName   sql.NullString
		)
		if err := rows.Scan(&id, &sourceNodeID, &targetNodeID, &cardinality, &edgeType, &description, &sourceName, &targetName); err != nil {
			continue
		}

		obj := map[string]interface{}{
			"id":           id,
			"sourceEntity": sourceNodeID,
			"targetEntity": targetNodeID,
			"cardinality":  cardinality.String,
			"edgeType":     edgeType.String,
			"description":  description.String,
			"sourceName":   sourceName.String,
			"targetName":   targetName.String,
			"isApplied":    true,
			"confidence":   1.0,
		}
		objects = append(objects, obj)
	}
	return objects, nil
}

func (s *Server) getSuggestedRelationships(ctx context.Context, db *sql.DB, tenantID, datasourceID, entityID string) ([]map[string]interface{}, error) {
	query := `
		SELECT rs.id, rs.tenant_id, rs.datasource_id, rs.source_entity_id, rs.target_entity_id,
		       rs.confidence, rs.rationale, rs.scoring_breakdown, rs.accepted, rs.accepted_at,
		       rs.created_at, rs.updated_at,
		       bo_source.display_name as source_name,
		       bo_target.display_name as target_name
		FROM relationship_suggestions rs
		LEFT JOIN business_objects bo_source ON bo_source.id = rs.source_entity_id
		  AND bo_source.tenant_id = rs.tenant_id
		LEFT JOIN business_objects bo_target ON bo_target.id = rs.target_entity_id
		  AND bo_target.tenant_id = rs.tenant_id
		WHERE rs.tenant_id = $1 AND rs.datasource_id = $2
		      AND rs.source_entity_id = $3::uuid
		ORDER BY rs.accepted DESC, rs.confidence DESC
		LIMIT 100
	`

	rows, err := db.QueryContext(ctx, query, tenantID, datasourceID, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var objects []map[string]interface{}
	for rows.Next() {
		var (
			id               string
			tenantID         string
			datasourceID     string
			sourceEntityID   string
			targetEntityID   string
			confidence       float64
			rationale        string
			scoringBreakdown interface{}
			accepted         bool
			acceptedAt       interface{}
			createdAt        interface{}
			updatedAt        interface{}
			sourceName       *string
			targetName       *string
		)

		err := rows.Scan(
			&id, &tenantID, &datasourceID, &sourceEntityID, &targetEntityID,
			&confidence, &rationale, &scoringBreakdown, &accepted, &acceptedAt,
			&createdAt, &updatedAt,
			&sourceName, &targetName,
		)
		if err != nil {
			continue
		}

		// Use display name if available, otherwise fallback to ID
		sourceDisplayName := sourceEntityID
		if sourceName != nil && *sourceName != "" {
			sourceDisplayName = *sourceName
		}

		targetDisplayName := targetEntityID
		if targetName != nil && *targetName != "" {
			targetDisplayName = *targetName
		}

		obj := map[string]interface{}{
			"id":           id,
			"sourceEntity": sourceEntityID,
			"sourceName":   sourceDisplayName,
			"targetEntity": targetEntityID,
			"targetName":   targetDisplayName,
			"cardinality":  "One-to-Many", // Default cardinality
			"description":  rationale,     // Use rationale as description
			"edgeType":     "entity_relationship",
			"confidence":   confidence,
			"rationale":    rationale,
			"accepted":     accepted,
			"acceptedAt":   acceptedAt,
			"isApplied":    accepted,
		}
		objects = append(objects, obj)
	}
	return objects, nil
}

// handleGetRelatedSemanticObjects retrieves related objects (semantic model relationships) for an entity
func (s *Server) handleGetRelatedSemanticObjects(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	db := s.DB

	tenantContext, err := extractTenantContext(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("missing tenant context: %v", err), http.StatusBadRequest)
		return
	}

	entityID := chi.URLParam(r, "entityID")
	if entityID == "" {
		writeJSONError(w, http.StatusBadRequest, "Entity ID required", "invalid_request", "")
		return
	}

	applied, err := s.getAppliedRelationships(ctx, db, tenantContext.TenantID, tenantContext.DatasourceID, entityID)
	if err != nil {
		log.Printf("handleGetRelatedObjects: getAppliedRelationships error: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to query applied relationships", "db_error", err.Error())
		return
	}

	suggested, err := s.getSuggestedRelationships(ctx, db, tenantContext.TenantID, tenantContext.DatasourceID, entityID)
	if err != nil {
		log.Printf("handleGetRelatedObjects: getSuggestedRelationships error: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to query suggested relationships", "db_error", err.Error())
		return
	}

	allObjects := append(applied, suggested...)

	log.Printf("handleGetRelatedObjects: found %d related entities for %s", len(allObjects), entityID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"relationships": allObjects,
		"count":         len(allObjects),
	})
}

// Relationship suggestion APIs are implemented in the relationships_chi.go
// package file. To avoid duplicating routes/handlers we delegate that
// responsibility there.

// handleApplyRelationshipSuggestion applies a suggestion and creates an edge
func (s *Server) handleApplyRelationshipSuggestion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	db := s.DB

	// Extract tenant context
	tenantContext, err := extractTenantContext(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("missing tenant context: %v", err), http.StatusBadRequest)
		return
	}

	var req ApplyRelationshipSuggestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body", "invalid_request", err.Error())
		return
	}

	if req.SuggestionID == "" {
		writeJSONError(w, http.StatusBadRequest, "Suggestion ID required", "invalid_request", "")
		return
	}

	// Fetch suggestion
	var sourceID, targetID string
	fetchQuery := `
		SELECT source_entity_id, target_entity_id
		FROM relationship_suggestions
		WHERE id = $1 AND tenant_id = $2
	`

	err = db.QueryRowContext(ctx, fetchQuery, req.SuggestionID, tenantContext.TenantID).Scan(&sourceID, &targetID)
	if err == sql.ErrNoRows {
		writeJSONError(w, http.StatusNotFound, "Suggestion not found", "not_found", "")
		return
	}
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to fetch suggestion", "db_error", err.Error())
		return
	}

	// Create edge in catalog
	edgeID := uuid.New().String()
	now := time.Now()

	createEdgeQuery := `
		INSERT INTO catalog_edge 
		(id, tenant_id, datasource_id, source_node_id, target_node_id, edge_type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, 'suggests', $6, $7)
	`

	_, err = db.ExecContext(ctx, createEdgeQuery, edgeID, tenantContext.TenantID, tenantContext.DatasourceID, sourceID, targetID, now, now)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to create edge", "db_error", err.Error())
		return
	}

	// Mark suggestion as accepted
	updateQuery := `
		UPDATE relationship_suggestions
		SET accepted = true, accepted_at = $1, updated_at = $2
		WHERE id = $3
	`
	_, err = db.ExecContext(ctx, updateQuery, now, now, req.SuggestionID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to mark suggestion as accepted", "db_error", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Relationship suggestion applied",
	})
}

// handleTraverseObjectGraph traverses a dot-notation path through relationships
func (s *Server) handleTraverseObjectGraph(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	db := s.DB

	// Extract tenant context
	tenantContext, err := extractTenantContext(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("missing tenant context: %v", err), http.StatusBadRequest)
		return
	}

	var req TraverseGraphRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body", "invalid_request", err.Error())
		return
	}

	if req.StartNodeID == "" || req.DotPath == "" {
		writeJSONError(w, http.StatusBadRequest, "Start node ID and dot path required", "invalid_request", "")
		return
	}

	// Simple traversal: follow edges matching segment names
	var nodes []string
	currentID := req.StartNodeID

	// Parse dot notation path
	segments := strings.Split(req.DotPath, ".")
	for _, segment := range segments {
		query := `
			SELECT target_node_id FROM catalog_edge
			WHERE source_node_id = $1 AND tenant_id = $2 AND datasource_id = $3
		`

		rows, err := db.QueryContext(ctx, query, currentID, tenantContext.TenantID, tenantContext.DatasourceID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to query edges", "db_error", err.Error())
			return
		}
		defer rows.Close()

		found := false
		for rows.Next() {
			var targetID string
			if err := rows.Scan(&targetID); err == nil {
				// Naive implementation: take first matching edge
				currentID = targetID
				nodes = append(nodes, currentID)
				found = true
				break
			}
		}

		if !found {
			writeJSONError(w, http.StatusNotFound, fmt.Sprintf("Path broken at segment: %s", segment), "path_not_found", "")
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"end_node_id": currentID,
		"path":        nodes,
	})
}

// handleGetSemanticBundle returns a map of semantic terms for a specific bundle
func (s *Server) handleGetSemanticBundle(w http.ResponseWriter, r *http.Request) {
	bundleId := chi.URLParam(r, "bundleId")
	boId := r.URL.Query().Get("bo_id")
	log.Printf("[DEBUG] handleGetSemanticBundle: bundleId='%s' (len=%d), bo_id='%s', condition check: bundleId=='by-id'=%v, bo_id!=''=%v\n", bundleId, len(bundleId), boId, bundleId == "by-id", boId != "")

	// Delegate to the getSemanticBundle handler when called with "by-id" parameter
	if bundleId == "by-id" || boId != "" {
		log.Printf("[DEBUG] Delegation condition TRUE - calling getSemanticBundle\n")
		s.getSemanticBundle(w, r)
		return
	}

	log.Printf("[DEBUG] Delegation condition FALSE - using mock response\n")

	bundleName := bundleId

	// Mock implementation to unblock frontend
	// In production this would query based on bundle/tag membership

	type SemanticTerm struct {
		ID           string `json:"id"`
		Name         string `json:"name"`
		BusinessName string `json:"businessName"`
		Type         string `json:"type"`
		Description  string `json:"description"`
	}

	mockTerms := make(map[string]SemanticTerm)

	// Seed with some common terms if they request standard bundles
	if bundleName == "banking" || bundleName == "financial_services" {
		mockTerms["customer_id"] = SemanticTerm{
			ID:           "customer_id",
			Name:         "customer_id",
			BusinessName: "Customer ID",
			Type:         "string",
			Description:  "Unique identifier for the customer",
		}
		mockTerms["account_balance"] = SemanticTerm{
			ID:           "account_balance",
			Name:         "account_balance",
			BusinessName: "Account Balance",
			Type:         "number",
			Description:  "Current balance of the account",
		}
	} else if bundleName == "insurance" {
		mockTerms["policy_number"] = SemanticTerm{
			ID:           "policy_number",
			Name:         "policy_number",
			BusinessName: "Policy Number",
			Type:         "string",
			Description:  "Unique number for the insurance policy",
		}
	} else if bundleName == "capital_markets" || bundleName == "fixed_income" {
		mockTerms["security_id"] = SemanticTerm{
			ID:           "security_id",
			Name:         "security_id",
			BusinessName: "Security ID (ISIN)",
			Type:         "string",
			Description:  "International Securities Identification Number",
		}
		mockTerms["yield"] = SemanticTerm{
			ID:           "yield",
			Name:         "yield",
			BusinessName: "Yield to Maturity",
			Type:         "number",
			Description:  "Expected return if held to maturity",
		}
	}

	// Always return 200 OK with what we have (even if empty) to valid JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mockTerms)
}
