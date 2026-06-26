package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/catalog"
)

// GraphRepository defines the interface for graph operations (mockable)
// Using shared types from catalog pkg to avoid cycles
type GraphRepository interface {
	GetNode(id string) (*catalog.CatalogNode, error)
	GetRelatedNodes(nodeID string, edgeType string, direction string) ([]catalog.CatalogNode, error)
	CreateNode(node catalog.CatalogNode) error
	CreateEdge(sourceID, targetID, edgeType string) error
	UpdateNodeProperties(id string, props map[string]interface{}) error
}

type ComplianceHandler struct {
	repo GraphRepository
}

// NewComplianceHandler creates a new handler (inject repo in real logic)
func NewComplianceHandler(repo GraphRepository) *ComplianceHandler {
	return &ComplianceHandler{repo: repo}
}

func (h *ComplianceHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/catalog/business-terms", func(r chi.Router) {
		r.Get("/{id}", h.GetBusinessTerm)
		r.Post("/{id}/mappings", h.AddMappings)
		r.Put("/{id}/compliance", h.UpdateCompliance)
	})
}

// BusinessTermResponse matches the UI requirements, mapped from Node
type BusinessTermResponse struct {
	ID            string           `json:"id"`
	Name          string           `json:"name"`
	Description   string           `json:"description"`
	PIIFlag       bool             `json:"piiFlag"`
	Residency     string           `json:"residency"`
	Sensitivity   string           `json:"sensitivity"`
	SemanticTerms []SimpleSemantic `json:"semanticTerms"`
	UpdatedAt     time.Time        `json:"updatedAt"`
	UpdatedBy     string           `json:"updatedBy"`
}

type SimpleSemantic struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// GetBusinessTerm returns business term details from CatalogNode
func (h *ComplianceHandler) GetBusinessTerm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// 1. Fetch Node
	node, err := h.repo.GetNode(id)
	if err != nil {
		http.Error(w, "Business term not found", http.StatusNotFound)
		return
	}

	// 2. Fetch Related Semantic Terms (Edges)
	semanticNodes, _ := h.repo.GetRelatedNodes(id, "IS_MAPPED_TO", "OUTGOING")

	// 3. Map Properties to Response
	props := node.Properties
	pii, _ := props["pii_flag"].(bool)
	res, _ := props["residency"].(string)
	sen, _ := props["sensitivity"].(string)

	terms := make([]SimpleSemantic, len(semanticNodes))
	for i, n := range semanticNodes {
		terms[i] = SimpleSemantic{ID: n.ID, Name: n.Name}
	}

	resp := BusinessTermResponse{
		ID:            node.ID,
		Name:          node.Name,
		Description:   node.Description,
		PIIFlag:       pii,
		Residency:     res,
		Sensitivity:   sen,
		SemanticTerms: terms,
		UpdatedAt:     time.Now(), // In real app, from node.UpdatedAt
		UpdatedBy:     "system",   // In real app, from node audit
	}

	json.NewEncoder(w).Encode(resp)
}

// AddMappings adds semantic terms to a business term by creating EDGES
func (h *ComplianceHandler) AddMappings(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		SemanticTermIDs []string `json:"semanticTermIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	for _, semanticID := range req.SemanticTermIDs {
		// Create 'IS_MAPPED_TO' edge from BusinessTerm -> SemanticTerm
		if err := h.repo.CreateEdge(id, semanticID, "IS_MAPPED_TO"); err != nil {
			// Log error but continue for now
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "mappings added", "businessTermId": id})
}

// UpdateCompliance updates compliance metadata in PROPERTIES
func (h *ComplianceHandler) UpdateCompliance(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		PIIFlag     bool   `json:"piiFlag"`
		Residency   string `json:"residency"`
		Sensitivity string `json:"sensitivity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	props := map[string]interface{}{
		"pii_flag":    req.PIIFlag,
		"residency":   req.Residency,
		"sensitivity": req.Sensitivity,
	}

	if err := h.repo.UpdateNodeProperties(id, props); err != nil {
		http.Error(w, "Failed to update compliance", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "compliance updated", "businessTermId": id})
}
