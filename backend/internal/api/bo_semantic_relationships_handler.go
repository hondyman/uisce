package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/jmoiron/sqlx"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// BOSemanticRelationshipsHandler handles HTTP requests for BO semantic relationships
type BOSemanticRelationshipsHandler struct {
	service *BOSemanticRelationshipsService
}

// NewBOSemanticRelationshipsHandler creates a new handler
func NewBOSemanticRelationshipsHandler(db *sqlx.DB) *BOSemanticRelationshipsHandler {
	return &BOSemanticRelationshipsHandler{
		service: NewBOSemanticRelationshipsService(db),
	}
}

// RegisterRoutes registers all routes for this handler
func (h *BOSemanticRelationshipsHandler) RegisterRoutes(r chi.Router) {
	r.Get("/business-objects/{boId}/foreign-keys", h.GetForeignKeyRelationships)
	r.Get("/business-objects/{boId}/related-semantic-terms", h.GetRelatedSemanticTerms)
	r.Post("/business-objects/{boId}/link-semantic-term", h.LinkSemanticTermToBO)
	r.Get("/business-objects/{boId}/semantic-join-paths", h.GetSemanticJoinPaths)
}

// GetForeignKeyRelationships returns all FK relationships for a BO's driving table
// GET /api/business-objects/{boId}/foreign-keys
func (h *BOSemanticRelationshipsHandler) GetForeignKeyRelationships(w http.ResponseWriter, r *http.Request) {
	boID := chi.URLParam(r, "boId")
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID

	if tenantID == "" {
		http.Error(w, "Missing X-Tenant-ID header", http.StatusBadRequest)
		return
	}

	if boID == "" {
		http.Error(w, "Missing boId parameter", http.StatusBadRequest)
		return
	}

	fks, err := h.service.DiscoverForeignKeyRelationshipsForBO(r.Context(), tenantID, boID)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Error discovering FK relationships: %v", err)
		http.Error(w, "Failed to discover foreign key relationships", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"business_object_id": boID,
		"foreign_keys":       fks,
		"count":              len(fks),
	})
}

// GetRelatedSemanticTerms returns available semantic terms from related tables
// GET /api/business-objects/{boId}/related-semantic-terms?limit=100
func (h *BOSemanticRelationshipsHandler) GetRelatedSemanticTerms(w http.ResponseWriter, r *http.Request) {
	boID := chi.URLParam(r, "boId")
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	limitStr := r.URL.Query().Get("limit")

	if tenantID == "" {
		http.Error(w, "Missing X-Tenant-ID header", http.StatusBadRequest)
		return
	}

	if boID == "" {
		http.Error(w, "Missing boId parameter", http.StatusBadRequest)
		return
	}

	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	terms, err := h.service.DiscoverSemanticTermsForRelatedTables(r.Context(), tenantID, boID, limit)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Error discovering semantic terms: %v", err)
		http.Error(w, "Failed to discover semantic terms", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"business_object_id":     boID,
		"related_semantic_terms": terms,
		"count":                  len(terms),
		"message":                "These semantic terms are available from related tables via foreign keys",
		"usage":                  "Link these to BO fields to enable joining related table data",
	})
}

// LinkSemanticTermToBO links a semantic term from a related table to a BO field
// POST /api/business-objects/{boId}/link-semantic-term
//
//	{
//	  "semantic_term_id": "...",
//	  "related_table_id": "...",
//	  "foreign_key_edge_id": "...",
//	  "role": "customer"
//	}
func (h *BOSemanticRelationshipsHandler) LinkSemanticTermToBO(w http.ResponseWriter, r *http.Request) {
	boID := chi.URLParam(r, "boId")
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID

	if tenantID == "" {
		http.Error(w, "Missing X-Tenant-ID header", http.StatusBadRequest)
		return
	}

	if boID == "" {
		http.Error(w, "Missing boId parameter", http.StatusBadRequest)
		return
	}

	var req BOSemanticLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.SemanticTermID == "" || req.ForeignKeyEdgeID == "" {
		http.Error(w, "Missing required fields: semantic_term_id, foreign_key_edge_id", http.StatusBadRequest)
		return
	}

	req.BusinessObjectID = boID
	err := h.service.LinkSemanticTermToBusinessObject(r.Context(), tenantID, &req)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Error linking semantic term: %v", err)
		http.Error(w, "Failed to link semantic term", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":             true,
		"message":             "Semantic term linked successfully",
		"business_object_id":  boID,
		"semantic_term_id":    req.SemanticTermID,
		"foreign_key_edge_id": req.ForeignKeyEdgeID,
	})
}

// GetSemanticJoinPaths returns the join paths needed to fetch semantic terms for a BO
// GET /api/business-objects/{boId}/semantic-join-paths
func (h *BOSemanticRelationshipsHandler) GetSemanticJoinPaths(w http.ResponseWriter, r *http.Request) {
	boID := chi.URLParam(r, "boId")
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID

	if tenantID == "" {
		http.Error(w, "Missing X-Tenant-ID header", http.StatusBadRequest)
		return
	}

	if boID == "" {
		http.Error(w, "Missing boId parameter", http.StatusBadRequest)
		return
	}

	joinPaths, err := h.service.GetBOSemanticJoinPaths(r.Context(), tenantID, boID)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Error getting join paths: %v", err)
		http.Error(w, "Failed to get join paths", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"business_object_id":  boID,
		"semantic_join_paths": joinPaths,
		"count":               len(joinPaths),
		"message":             "Use these join paths to construct queries that fetch semantic term data from related tables",
	})
}
