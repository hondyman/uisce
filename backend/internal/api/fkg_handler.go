package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// FKGHandler handles Financial Knowledge Graph API requests.
type FKGHandler struct {
	fkgService FKGServiceInterface
}

// FKGServiceInterface defines the interface for FKG operations.
type FKGServiceInterface interface {
	CreateEntity(ctx interface{}, tenantID string, entity interface{}) (interface{}, error)
	GetEntity(ctx interface{}, tenantID, entityID string) (interface{}, error)
	UpdateEntity(ctx interface{}, tenantID, entityID string, updates map[string]interface{}) error
	DeleteEntity(ctx interface{}, tenantID, entityID string) error
	ListEntities(ctx interface{}, tenantID string, entityType string, limit, offset int) ([]interface{}, error)
	FindSimilarEntities(ctx interface{}, tenantID, name string, threshold float64) ([]interface{}, error)
	CreateRelationship(ctx interface{}, tenantID string, rel interface{}) error
	GetUBOChain(ctx interface{}, tenantID, entityID string, maxDepth int) ([]interface{}, error)
	HybridSearchDocuments(ctx interface{}, tenantID, query string, embedding []float32, limit int) ([]interface{}, error)
}

// NewFKGHandler creates a new FKG handler.
func NewFKGHandler(fkgService FKGServiceInterface) *FKGHandler {
	return &FKGHandler{
		fkgService: fkgService,
	}
}

// RegisterFKGRoutes registers FKG routes on the router.
func (h *FKGHandler) RegisterFKGRoutes(r chi.Router) {
	r.Route("/fkg", func(r chi.Router) {
		// Entity endpoints
		r.Route("/entities", func(r chi.Router) {
			r.Get("/", h.ListEntities)
			r.Post("/", h.CreateEntity)
			r.Get("/search", h.SearchSimilarEntities)
			r.Get("/{entityID}", h.GetEntity)
			r.Put("/{entityID}", h.UpdateEntity)
			r.Delete("/{entityID}", h.DeleteEntity)
			r.Get("/{entityID}/ubo", h.GetUBOChain)
		})

		// Relationship endpoints
		r.Route("/relationships", func(r chi.Router) {
			r.Post("/", h.CreateRelationship)
			r.Get("/", h.ListRelationships)
		})

		// Document search endpoints
		r.Route("/documents", func(r chi.Router) {
			r.Post("/search", h.SearchDocuments)
		})
	})
}

// CreateEntityRequest represents the request to create an entity.
type CreateEntityRequest struct {
	EntityType  string                 `json:"entity_type"`
	Name        string                 `json:"name"`
	CanonicalID string                 `json:"canonical_id,omitempty"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
}

// CreateEntity handles POST /fkg/entities.
func (h *FKGHandler) CreateEntity(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id required", http.StatusBadRequest)
		return
	}

	var req CreateEntityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.EntityType == "" {
		http.Error(w, "name and entity_type required", http.StatusBadRequest)
		return
	}

	entity := map[string]interface{}{
		"entity_id":    uuid.New().String(),
		"entity_type":  req.EntityType,
		"name":         req.Name,
		"canonical_id": req.CanonicalID,
		"properties":   req.Properties,
	}

	result, err := h.fkgService.CreateEntity(r.Context(), tenantID, entity)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

// GetEntity handles GET /fkg/entities/{entityID}.
func (h *FKGHandler) GetEntity(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	entityID := chi.URLParam(r, "entityID")

	if tenantID == "" || entityID == "" {
		http.Error(w, "tenant_id and entityID required", http.StatusBadRequest)
		return
	}

	entity, err := h.fkgService.GetEntity(r.Context(), tenantID, entityID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entity)
}

// UpdateEntity handles PUT /fkg/entities/{entityID}.
func (h *FKGHandler) UpdateEntity(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	entityID := chi.URLParam(r, "entityID")

	if tenantID == "" || entityID == "" {
		http.Error(w, "tenant_id and entityID required", http.StatusBadRequest)
		return
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.fkgService.UpdateEntity(r.Context(), tenantID, entityID, updates); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteEntity handles DELETE /fkg/entities/{entityID}.
func (h *FKGHandler) DeleteEntity(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	entityID := chi.URLParam(r, "entityID")

	if tenantID == "" || entityID == "" {
		http.Error(w, "tenant_id and entityID required", http.StatusBadRequest)
		return
	}

	if err := h.fkgService.DeleteEntity(r.Context(), tenantID, entityID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListEntities handles GET /fkg/entities.
func (h *FKGHandler) ListEntities(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id required", http.StatusBadRequest)
		return
	}

	entityType := r.URL.Query().Get("entity_type")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	entities, err := h.fkgService.ListEntities(r.Context(), tenantID, entityType, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"entities": entities,
		"limit":    limit,
		"offset":   offset,
	})
}

// SearchSimilarEntities handles GET /fkg/entities/search.
func (h *FKGHandler) SearchSimilarEntities(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	name := r.URL.Query().Get("name")

	if tenantID == "" || name == "" {
		http.Error(w, "tenant_id and name required", http.StatusBadRequest)
		return
	}

	threshold := 0.85
	if t := r.URL.Query().Get("threshold"); t != "" {
		if parsed, err := strconv.ParseFloat(t, 64); err == nil {
			threshold = parsed
		}
	}

	entities, err := h.fkgService.FindSimilarEntities(r.Context(), tenantID, name, threshold)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"entities":  entities,
		"query":     name,
		"threshold": threshold,
	})
}

// CreateRelationshipRequest represents the request to create a relationship.
type CreateRelationshipRequest struct {
	SourceEntityID      string                 `json:"source_entity_id"`
	TargetEntityID      string                 `json:"target_entity_id"`
	RelationshipType    string                 `json:"relationship_type"`
	PercentageOwnership float64                `json:"percentage_ownership,omitempty"`
	VotingRights        float64                `json:"voting_rights,omitempty"`
	EffectiveDate       string                 `json:"effective_date,omitempty"`
	Properties          map[string]interface{} `json:"properties,omitempty"`
}

// CreateRelationship handles POST /fkg/relationships.
func (h *FKGHandler) CreateRelationship(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id required", http.StatusBadRequest)
		return
	}

	var req CreateRelationshipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.SourceEntityID == "" || req.TargetEntityID == "" || req.RelationshipType == "" {
		http.Error(w, "source_entity_id, target_entity_id, and relationship_type required", http.StatusBadRequest)
		return
	}

	rel := map[string]interface{}{
		"relationship_id":      uuid.New().String(),
		"source_entity_id":     req.SourceEntityID,
		"target_entity_id":     req.TargetEntityID,
		"relationship_type":    req.RelationshipType,
		"percentage_ownership": req.PercentageOwnership,
		"voting_rights":        req.VotingRights,
		"effective_date":       req.EffectiveDate,
		"properties":           req.Properties,
	}

	if err := h.fkgService.CreateRelationship(r.Context(), tenantID, rel); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// ListRelationships handles GET /fkg/relationships.
func (h *FKGHandler) ListRelationships(w http.ResponseWriter, r *http.Request) {
	// Placeholder - would implement relationship listing
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"relationships": []interface{}{},
	})
}

// GetUBOChain handles GET /fkg/entities/{entityID}/ubo.
func (h *FKGHandler) GetUBOChain(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	entityID := chi.URLParam(r, "entityID")

	if tenantID == "" || entityID == "" {
		http.Error(w, "tenant_id and entityID required", http.StatusBadRequest)
		return
	}

	maxDepth := 20
	if d := r.URL.Query().Get("max_depth"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 && parsed <= 50 {
			maxDepth = parsed
		}
	}

	chain, err := h.fkgService.GetUBOChain(r.Context(), tenantID, entityID, maxDepth)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"entity_id":       entityID,
		"ownership_chain": chain,
		"max_depth":       maxDepth,
	})
}

// SearchDocumentsRequest represents the request to search documents.
type SearchDocumentsRequest struct {
	Query     string    `json:"query"`
	Embedding []float32 `json:"embedding,omitempty"`
	Limit     int       `json:"limit,omitempty"`
	EntityID  string    `json:"entity_id,omitempty"`
}

// SearchDocuments handles POST /fkg/documents/search.
func (h *FKGHandler) SearchDocuments(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id required", http.StatusBadRequest)
		return
	}

	var req SearchDocumentsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Query == "" {
		http.Error(w, "query required", http.StatusBadRequest)
		return
	}

	if req.Limit <= 0 || req.Limit > 50 {
		req.Limit = 10
	}

	results, err := h.fkgService.HybridSearchDocuments(r.Context(), tenantID, req.Query, req.Embedding, req.Limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results": results,
		"query":   req.Query,
		"limit":   req.Limit,
	})
}
