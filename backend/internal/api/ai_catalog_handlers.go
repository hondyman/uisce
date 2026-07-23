package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type AICatalogHandler struct {
	repo GraphRepository
}

func NewAICatalogHandler(repo GraphRepository) *AICatalogHandler {
	return &AICatalogHandler{repo: repo}
}

// ... methods ...

// 2.4 Approve
func (h *AICatalogHandler) ApproveSuggestion(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// In a real app:
	// 1. Fetch Draft from DB (using new schema fields like hierarchy, pii_flag, etc.)
	// 2. Create CatalogNode (Business Term) properties={pii, residency, sensitivity, hierarchy...}
	// 3. Create CatalogEdges (IS_MAPPED_TO) to semantic terms
	// 4. Update Draft status to APPROVED

	// Mock graph creation for now
	// h.repo.CreateNode(ctx, "BUSINESS_TERM", draft.Name, map[string]interface{}{
	//     "piiFlag": draft.PIIFlag,
	//     "sensitivity": draft.Sensitivity,
	//     "hierarchy": draft.Hierarchy,
	// })
	// h.repo.CreateEdge(...)

	resp := map[string]string{"status": "approved", "businessTermId": "bt-" + id}
	json.NewEncoder(w).Encode(resp)
}

// 2.5 Reject
func (h *AICatalogHandler) RejectSuggestion(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Reason string `json:"reason"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	// Logic: Mark Draft REJECTED
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "rejected", "id": id})
}
