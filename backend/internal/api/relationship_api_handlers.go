package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hondyman/semlayer/backend/internal/logging"
)

// ============================================================================
// Relationship Discovery API Handlers
// ============================================================================

// postDiscoverRelationships discovers relationships for an entity
func (s *Server) postDiscoverRelationships(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract tenant context from request
	tenantContext, err := extractTenantContext(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("missing tenant context: %v", err), http.StatusBadRequest)
		return
	}

	// Parse request body
	var req struct {
		EntityAttributeID string `json:"entity_attribute_id"`
		IncludeMultiHop   bool   `json:"include_multi_hop"`
		MaxHopDepth       int    `json:"max_hop_depth"`
	}

	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.EntityAttributeID == "" {
		http.Error(w, "entity_attribute_id is required", http.StatusBadRequest)
		return
	}

	if req.MaxHopDepth == 0 {
		req.MaxHopDepth = 3 // Default to 3 hops
	}
	if req.MaxHopDepth > 5 {
		req.MaxHopDepth = 5 // Cap at 5 hops
	}

	// Initialize discovery service
	discoveryService := NewEnhancedRelationshipDiscoveryService(s.DB)

	// Try complex discovery first
	directRelationships, err := discoveryService.DiscoverLinkableEntitiesWithSemanticContext(
		ctx,
		tenantContext.TenantID,
		tenantContext.DatasourceID,
		req.EntityAttributeID,
	)

	// If complex discovery fails, fall back to simple business object discovery
	if err != nil {
		logging.GetLogger().Sugar().Warnf("Complex relationship discovery failed (%v), falling back to simple discovery", err)
		directRelationships, err = s.discoverSimpleBusinessObjectRelationships(
			ctx,
			tenantContext.TenantID,
			tenantContext.DatasourceID,
			req.EntityAttributeID,
		)
		if err != nil {
			logging.GetLogger().Sugar().Errorf("simple relationship discovery also failed: %v", err)
			http.Error(w, fmt.Sprintf("failed to discover relationships: %v", err), http.StatusInternalServerError)
			return
		}
	}

	response := map[string]interface{}{
		"entity_attribute_id":  req.EntityAttributeID,
		"direct_relationships": directRelationships,
		"multi_hop_paths":      []interface{}{},
	}

	// Discover multi-hop paths if requested
	if req.IncludeMultiHop && req.MaxHopDepth > 1 {
		multiHopPaths, err := discoveryService.DiscoverMultiHopPaths(
			ctx,
			tenantContext.TenantID,
			tenantContext.DatasourceID,
			req.EntityAttributeID,
			req.MaxHopDepth,
		)
		if err != nil {
			logging.GetLogger().Sugar().Warnf("failed to discover multi-hop paths: %v", err)
			// Don't fail the request, just skip multi-hop
		} else {
			response["multi_hop_paths"] = multiHopPaths
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// postGetExistingRelationships retrieves already-applied (linked) relationships for an entity
// This endpoint is called by the RelationshipDiscoveryModal to show which relationships
// have already been established for an entity attribute.
func (s *Server) postGetExistingRelationships(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract tenant context from request
	tenantContext, err := extractTenantContext(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("missing tenant context: %v", err), http.StatusBadRequest)
		return
	}

	// Parse request body
	var req struct {
		EntityAttributeID string `json:"entity_attribute_id"`
	}

	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.EntityAttributeID == "" {
		http.Error(w, "entity_attribute_id is required", http.StatusBadRequest)
		return
	}

	// Query existing relationships for this entity
	// These are relationships where the entity is the source
	query := `
		SELECT 
			CAST(bor.target_object_id AS TEXT) as entity_id,
			bo.name as entity_name,
			bo.display_name,
			'DIRECT_FK' as link_type,
			bor.cardinality,
			COALESCE(bor.confidence, 1.0) as confidence,
			'Established relationship' as confidence_reason,
			'' as foreign_key_path,
			NULL as semantic_term_name,
			CURRENT_TIMESTAMP as discovered_at
		FROM business_object_relationships bor
		JOIN business_objects bo ON bo.id = bor.target_object_id
		WHERE bor.tenant_id = $1
		  AND bor.source_object_id = $2
		  AND bor.is_user_applied = true
		ORDER BY bo.name
	`

	rows, err := s.DB.QueryContext(ctx, query, tenantContext.TenantID, req.EntityAttributeID)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to fetch existing relationships for %s: %v", req.EntityAttributeID, err)
		http.Error(w, fmt.Sprintf("failed to fetch relationships: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var existingRelationships []EnhancedRelatedEntity

	for rows.Next() {
		var (
			entity           EnhancedRelatedEntity
			displayName      string
			semanticTermName sql.NullString
		)

		if err := rows.Scan(
			&entity.EntityID,
			&entity.EntityName,
			&displayName,
			&entity.LinkType,
			&entity.Cardinality,
			&entity.Confidence,
			&entity.ConfidenceReason,
			&entity.ForeignKeyPath,
			&semanticTermName,
			&entity.DiscoveredAt,
		); err != nil {
			logging.GetLogger().Sugar().Warnf("Failed to scan relationship row: %v", err)
			continue
		}

		if semanticTermName.Valid {
			entity.SemanticTermName = semanticTermName.String
		}

		existingRelationships = append(existingRelationships, entity)
	}

	if err := rows.Err(); err != nil {
		logging.GetLogger().Sugar().Errorf("Error iterating existing relationships: %v", err)
		http.Error(w, fmt.Sprintf("error reading relationships: %v", err), http.StatusInternalServerError)
		return
	}

	logging.GetLogger().Sugar().Debugf(
		"Found %d existing relationships for entity %s in tenant %s",
		len(existingRelationships), req.EntityAttributeID, tenantContext.TenantID,
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"existing_relationships": existingRelationships,
	})
}

// postTriggerModelRegeneration triggers model regeneration
func (s *Server) postTriggerModelRegeneration(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract tenant context
	tenantContext, err := extractTenantContext(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("missing tenant context: %v", err), http.StatusBadRequest)
		return
	}

	// Parse request body
	var req struct {
		EntityAttributeID string `json:"entity_attribute_id"`
		TriggerType       string `json:"trigger_type"`
		Priority          int    `json:"priority"`
		Reason            string `json:"reason"`
	}

	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.EntityAttributeID == "" || req.TriggerType == "" {
		http.Error(w, "entity_attribute_id and trigger_type are required", http.StatusBadRequest)
		return
	}

	if req.Priority == 0 {
		req.Priority = 5 // Default priority
	}

	// Initialize regeneration service
	regenerationService := NewModelRegenerationService(s.DB)

	// Get user ID from context
	userID := ""
	if user := ctx.Value("user"); user != nil {
		if u, ok := user.(map[string]interface{}); ok {
			if id, ok := u["sub"].(string); ok {
				userID = id
			}
		}
	}

	// Trigger regeneration
	queueID, err := regenerationService.TriggerModelRegeneration(
		ctx,
		tenantContext.TenantID,
		tenantContext.DatasourceID,
		&ModelRegenerationRequest{
			EntityAttributeID: req.EntityAttributeID,
			TriggerType:       req.TriggerType,
			TriggerSource:     "API",
			Priority:          req.Priority,
			Reason:            req.Reason,
			RequestedBy:       userID,
		},
	)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("failed to trigger regeneration: %v", err)
		http.Error(w, fmt.Sprintf("failed to trigger regeneration: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"queue_id": queueID,
		"status":   "queued",
		"message":  "Model regeneration triggered",
		"priority": req.Priority,
	})
}

// getModelVersion retrieves a specific model version
func (s *Server) getModelVersion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract tenant context
	tenantContext, err := extractTenantContext(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("missing tenant context: %v", err), http.StatusBadRequest)
		return
	}

	// Get query parameters
	entityAttributeID := r.URL.Query().Get("entity_attribute_id")
	versionStr := r.URL.Query().Get("version")

	if entityAttributeID == "" {
		http.Error(w, "entity_attribute_id is required", http.StatusBadRequest)
		return
	}

	// Initialize regeneration service
	regenerationService := NewModelRegenerationService(s.DB)

	// Default to latest version if not specified
	if versionStr == "" || versionStr == "latest" {
		// Generate current model
		model, err := regenerationService.GenerateSemanticModel(
			ctx,
			tenantContext.TenantID,
			tenantContext.DatasourceID,
			entityAttributeID,
		)
		if err != nil {
			logging.GetLogger().Sugar().Errorf("failed to generate model: %v", err)
			http.Error(w, fmt.Sprintf("failed to generate model: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"model":   model,
			"version": "latest",
			"status":  "success",
		})
		return
	}

	// Parse version number
	var versionNumber int
	_, err = fmt.Sscanf(versionStr, "%d", &versionNumber)
	if err != nil {
		http.Error(w, "invalid version number", http.StatusBadRequest)
		return
	}

	// Get specific version
	model, err := regenerationService.GetModelVersion(ctx, entityAttributeID, versionNumber)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("failed to get model version: %v", err)
		http.Error(w, fmt.Sprintf("failed to get model version: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"model":   model,
		"version": versionNumber,
		"status":  "success",
	})
}

// discoverSimpleBusinessObjectRelationships provides a simple fallback for discovering
// relationships between business objects when the complex entity attribute discovery fails.
// It returns all other business objects as potential relationship targets.
func (s *Server) discoverSimpleBusinessObjectRelationships(
	ctx context.Context,
	tenantID, datasourceID, sourceEntityID string,
) ([]EnhancedRelatedEntity, error) {
	query := `
		SELECT 
			bo.id as entity_id,
			bo.name as entity_name,
			bo.display_name as display_name,
			'ASSOCIATION' as link_type,
			'N:M' as cardinality,
			0.7 as confidence,
			'Business object available' as confidence_reason
		FROM business_objects bo
		WHERE bo.id != $1
		ORDER BY bo.name
		LIMIT 50
	`

	rows, err := s.DB.QueryContext(ctx, query, sourceEntityID)
	if err != nil {
		return nil, fmt.Errorf("failed to query business objects: %w", err)
	}
	defer rows.Close()

	var entities []EnhancedRelatedEntity
	for rows.Next() {
		var entity EnhancedRelatedEntity
		var displayName string
		if err := rows.Scan(
			&entity.EntityID,
			&entity.EntityName,
			&displayName,
			&entity.LinkType,
			&entity.Cardinality,
			&entity.Confidence,
			&entity.ConfidenceReason,
		); err != nil {
			return nil, fmt.Errorf("failed to scan entity: %w", err)
		}
		entity.ForeignKeyPath = ""
		entities = append(entities, entity)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return entities, nil
}

// (Helper functions such as TenantContext and extractTenantContext are
// implemented centrally in helpers.go — avoid duplicating them here.)
