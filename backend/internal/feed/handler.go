package feed

import (
	"encoding/json"
	"net/http"
)

type Handler struct {
	curator *Curator
}

func NewHandler(curator *Curator) *Handler {
	return &Handler{
		curator: curator,
	}
}

func (h *Handler) GetFeed(w http.ResponseWriter, r *http.Request) {
	// In a real app, we'd extract tenant/client from auth context
	tenantID := "default"
	clientID := "c_12345"

	feed, err := h.curator.GenerateFeed(r.Context(), tenantID, clientID)
	if err != nil {
		http.Error(w, "Failed to generate feed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(feed)
}
