package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/analytics"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/jmoiron/sqlx"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

type BusinessTermsHandler struct {
	service *analytics.SemanticMappingService
	db      *sqlx.DB
}

func NewBusinessTermsHandler(service *analytics.SemanticMappingService, db *sqlx.DB) *BusinessTermsHandler {
	return &BusinessTermsHandler{
		service: service,
		db:      db,
	}
}

func (h *BusinessTermsHandler) RegisterRoutes(r chi.Router) {
	r.Get("/business-terms", h.FetchBusinessTerms)
	r.Post("/business-terms", h.CreateBusinessTerm)
	r.Post("/business-terms/search", h.SearchBusinessTerms)
	r.Put("/business-terms/{id}", h.UpdateBusinessTerm)
	r.Delete("/business-terms/{id}", h.DeleteBusinessTerm)

	r.Get("/business-term-edges", h.GetEdges)
	r.Post("/business-term-edges", h.CreateEdge)
	r.Delete("/business-term-edges/{id}", h.DeleteEdge)
	r.Delete("/business-term-edges", h.DeleteEdgeByNodes)
}

func (h *BusinessTermsHandler) FetchBusinessTerms(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	tenantDatasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if tenantID == "" || tenantDatasourceID == "" {
		http.Error(w, "X-Tenant-ID and X-Tenant-Datasource-ID headers are required", http.StatusBadRequest)
		return
	}

	terms, err := h.service.FetchBusinessTerms(r.Context(), tenantID, tenantDatasourceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": terms,
	})
}

func (h *BusinessTermsHandler) SearchBusinessTerms(w http.ResponseWriter, r *http.Request) {
	// Re-using fetch logic as search was unimplemented in original code (just returned all terms)
	h.FetchBusinessTerms(w, r)
}

func (h *BusinessTermsHandler) CreateBusinessTerm(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	tenantDatasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if tenantID == "" || tenantDatasourceID == "" {
		http.Error(w, "X-Tenant-ID and X-Tenant-Datasource-ID headers are required", http.StatusBadRequest)
		return
	}

	var req struct {
		TermName   string                 `json:"term_name"`
		Properties map[string]interface{} `json:"properties"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.TermName == "" || req.Properties == nil {
		http.Error(w, "term_name is required", http.StatusBadRequest)
		return
	}

	termName := toTitleCase(req.TermName)
	nodeID, err := h.service.CreateBusinessTerm(r.Context(), tenantID, tenantDatasourceID, termName, req.Properties)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create business term: %v", err), http.StatusInternalServerError)
		return
	}

	result := map[string]interface{}{
		"node_id":   nodeID,
		"term_name": termName,
	}
	json.NewEncoder(w).Encode(result)
}

func (h *BusinessTermsHandler) UpdateBusinessTerm(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	tenantDatasourceID := r.Header.Get("X-Tenant-Datasource-ID")
	termNodeID := chi.URLParam(r, "id")

	if tenantID == "" || tenantDatasourceID == "" {
		http.Error(w, "X-Tenant-ID and X-Tenant-Datasource-ID headers are required", http.StatusBadRequest)
		return
	}

	if termNodeID == "" {
		http.Error(w, "Business term ID is required", http.StatusBadRequest)
		return
	}

	var req struct {
		TermName    string                 `json:"term_name,omitempty"`
		Description string                 `json:"description,omitempty"`
		Category    string                 `json:"category,omitempty"`
		Owner       string                 `json:"owner,omitempty"`
		Properties  map[string]interface{} `json:"properties,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	updates := make(map[string]interface{})
	if req.TermName != "" {
		updates["term_name"] = req.TermName
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Category != "" {
		updates["category"] = req.Category
	}
	if req.Owner != "" {
		updates["owner"] = req.Owner
	}
	if req.Properties != nil {
		for k, v := range req.Properties {
			updates[k] = v
		}
	}

	if len(updates) == 0 {
		http.Error(w, "At least one field must be provided for update", http.StatusBadRequest)
		return
	}

	err := h.service.UpdateBusinessTerm(r.Context(), tenantID, tenantDatasourceID, termNodeID, updates)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "access denied") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if strings.Contains(err.Error(), "already exists") {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to update business term: %v", err), http.StatusInternalServerError)
		return
	}

	result := map[string]interface{}{
		"success": true,
		"message": "Business term updated successfully",
	}
	json.NewEncoder(w).Encode(result)
}

func (h *BusinessTermsHandler) DeleteBusinessTerm(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	tenantDatasourceID := r.Header.Get("X-Tenant-Datasource-ID")
	termNodeID := chi.URLParam(r, "id")

	if tenantID == "" || tenantDatasourceID == "" {
		http.Error(w, "X-Tenant-ID and X-Tenant-Datasource-ID headers are required", http.StatusBadRequest)
		return
	}

	if termNodeID == "" {
		http.Error(w, "Business term ID is required", http.StatusBadRequest)
		return
	}

	if _, err := uuid.Parse(termNodeID); err != nil {
		http.Error(w, "Invalid business term ID format", http.StatusBadRequest)
		return
	}

	_, err := h.db.ExecContext(r.Context(), `
		DELETE FROM public.catalog_node
		WHERE id = $1 AND tenant_id = $2 AND tenant_datasource_id = $3 AND node_type = 'business_term'
	`, termNodeID, tenantID, tenantDatasourceID)

	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to delete business term: %v", err)
		http.Error(w, fmt.Sprintf("Failed to delete business term: %v", err), http.StatusInternalServerError)
		return
	}

	result := map[string]interface{}{
		"success": true,
		"message": "Business term deleted successfully",
	}
	json.NewEncoder(w).Encode(result)
}

func (h *BusinessTermsHandler) GetEdges(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	tenantDatasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if tenantID == "" || tenantDatasourceID == "" {
		http.Error(w, "X-Tenant-ID and X-Tenant-Datasource-ID headers are required", http.StatusBadRequest)
		return
	}

	query := `
		SELECT
			id,
			source_node_id as source_node_id,
			target_node_id as target_node_id,
			edge_type_id,
			relationship_type,
			properties,
			created_at,
			updated_at
		FROM public.catalog_edge
		WHERE tenant_id = $1
			AND tenant_datasource_id = $2
			AND edge_type_id = '3be9d6ae-1598-4628-a3dd-b606921a9193'
		ORDER BY created_at DESC
	`

	rows, err := h.db.QueryContext(r.Context(), query, tenantID, tenantDatasourceID)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to query business term edges: %v", err)
		http.Error(w, fmt.Sprintf("Failed to query business term edges: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var edges []map[string]interface{}
	for rows.Next() {
		var id, sourceNodeID, targetNodeID, edgeTypeID, relationshipType string
		var properties []byte
		var createdAt, updatedAt time.Time

		if err := rows.Scan(&id, &sourceNodeID, &targetNodeID, &edgeTypeID, &relationshipType, &properties, &createdAt, &updatedAt); err != nil {
			logging.GetLogger().Sugar().Errorf("Failed to scan business term edge: %v", err)
			http.Error(w, fmt.Sprintf("Failed to scan business term edge: %v", err), http.StatusInternalServerError)
			return
		}

		edge := map[string]interface{}{
			"id":                id,
			"source_node_id":    sourceNodeID,
			"target_node_id":    targetNodeID,
			"edge_type_id":      edgeTypeID,
			"relationship_type": relationshipType,
			"created_at":        createdAt,
			"updated_at":        updatedAt,
		}

		if len(properties) > 0 {
			var props map[string]interface{}
			if err := json.Unmarshal(properties, &props); err == nil {
				edge["properties"] = props
			}
		}

		edges = append(edges, edge)
	}

	if err := rows.Err(); err != nil {
		logging.GetLogger().Sugar().Errorf("Error iterating business term edges: %v", err)
		http.Error(w, fmt.Sprintf("Error iterating business term edges: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(edges)
}

func (h *BusinessTermsHandler) CreateEdge(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var body struct {
		SubjectNodeID    string `json:"subject_node_id"`   // Business term ID (legacy)
		ObjectNodeID     string `json:"object_node_id"`    // Semantic term ID (legacy)
		EdgeTypeID       string `json:"edge_type_id"`      // Optional explicit edge type id
		RelationshipType string `json:"relationship_type"` // business_term_to_semantic_term

		// Frontend payload shape
		SourceNodeID string  `json:"source_node_id"` // business term id from frontend
		TargetNodeID string  `json:"target_node_id"` // semantic term id from frontend
		EdgeType     string  `json:"edge_type"`      // e.g. "business_term_mapping"
		Confidence   float64 `json:"confidence"`     // optional
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid payload"})
		return
	}

	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	tenantDatasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if tenantID == "" || tenantDatasourceID == "" {
		http.Error(w, "X-Tenant-ID and X-Tenant-Datasource-ID headers are required", http.StatusBadRequest)
		return
	}

	// Normalize
	sourceID := body.SourceNodeID
	if sourceID == "" {
		sourceID = body.SubjectNodeID
	}
	targetID := body.TargetNodeID
	if targetID == "" {
		targetID = body.ObjectNodeID
	}

	if sourceID == "" || targetID == "" {
		http.Error(w, "source_node_id and target_node_id are required", http.StatusBadRequest)
		return
	}

	// Create via service (reusing CreateBusinessTermEdge logic for now, or new method if distinct)
	// The original code called CreateBusinessTermEdge for /semantic-mappings/business-term-edges,
	// but this endpoint was new/distinct in the orphaned block.
	// We'll use CreateBusinessTermEdge as it seems to handle the business<->semantic link.
	// IF this is a generic edge, we might need a different service method.
	// But given the context, it's likely business term to semantic term.

	created, err := h.service.CreateBusinessTermEdge(r.Context(), tenantID, tenantDatasourceID, targetID, sourceID) // Note arg order: semantic, business
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create edge: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"created": created,
	})
}

func (h *BusinessTermsHandler) DeleteEdge(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	tenantDatasourceID := r.Header.Get("X-Tenant-Datasource-ID")
	edgeID := chi.URLParam(r, "id")

	if tenantID == "" || tenantDatasourceID == "" {
		http.Error(w, "X-Tenant-ID and X-Tenant-Datasource-ID headers are required", http.StatusBadRequest)
		return
	}

	if edgeID == "" {
		http.Error(w, "Edge ID is required", http.StatusBadRequest)
		return
	}

	if _, err := uuid.Parse(edgeID); err != nil {
		http.Error(w, "Invalid edge ID format", http.StatusBadRequest)
		return
	}

	_, err := h.db.ExecContext(r.Context(), `
		DELETE FROM public.catalog_edge
		WHERE id = $1 AND tenant_id = $2 AND tenant_datasource_id = $3
	`, edgeID, tenantID, tenantDatasourceID)

	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to delete business term edge: %v", err)
		http.Error(w, fmt.Sprintf("Failed to delete business term edge: %v", err), http.StatusInternalServerError)
		return
	}

	result := map[string]interface{}{
		"success": true,
		"message": "Business term edge deleted successfully",
	}
	json.NewEncoder(w).Encode(result)
}

func (h *BusinessTermsHandler) DeleteEdgeByNodes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	tenantDatasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	semanticTermID := r.URL.Query().Get("semantic_term_id")
	businessTermID := r.URL.Query().Get("business_term_id")

	if tenantID == "" || tenantDatasourceID == "" {
		http.Error(w, "X-Tenant-ID and X-Tenant-Datasource-ID headers are required", http.StatusBadRequest)
		return
	}

	if semanticTermID == "" || businessTermID == "" {
		http.Error(w, "semantic_term_id and business_term_id query parameters are required", http.StatusBadRequest)
		return
	}

	if _, err := uuid.Parse(semanticTermID); err != nil {
		http.Error(w, "Invalid semantic_term_id format", http.StatusBadRequest)
		return
	}
	if _, err := uuid.Parse(businessTermID); err != nil {
		http.Error(w, "Invalid business_term_id format", http.StatusBadRequest)
		return
	}

	_, err := h.db.ExecContext(r.Context(), `
		DELETE FROM public.catalog_edge
		WHERE tenant_id = $1 
		AND tenant_datasource_id = $2 
		AND source_node_id = $3 
		AND target_node_id = $4
		AND edge_type_id = 'has_semantic_edge'
	`, tenantID, tenantDatasourceID, businessTermID, semanticTermID)

	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to delete business term edge: %v", err)
		http.Error(w, fmt.Sprintf("Failed to delete business term edge: %v", err), http.StatusInternalServerError)
		return
	}

	result := map[string]interface{}{
		"success": true,
		"message": "Business term edge deleted successfully",
	}
	json.NewEncoder(w).Encode(result)
}
