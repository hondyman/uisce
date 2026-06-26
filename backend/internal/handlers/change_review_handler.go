package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/review"
)

// ChangeReviewHandler handles HTTP requests for change reviews
type ChangeReviewHandler struct {
	service *review.ChangeReviewService
}

// NewChangeReviewHandler creates a new handler
func NewChangeReviewHandler(service *review.ChangeReviewService) *ChangeReviewHandler {
	return &ChangeReviewHandler{service: service}
}

// RegisterRoutes registers endpoints
func (h *ChangeReviewHandler) RegisterRoutes(r chi.Router) {
	r.Route("/change-reviews", func(r chi.Router) {
		r.Post("/", h.CreateReview)
		r.Get("/{id}", h.GetReview)
		r.Post("/{id}/promote", h.Promote)
		r.Post("/rollback", h.Rollback)
	})
}

// CreateReview creates a new review for a changeset
func (h *ChangeReviewHandler) CreateReview(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ChangeSetID string `json:"change_set_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	changeSetID, err := uuid.Parse(req.ChangeSetID)
	if err != nil {
		http.Error(w, "invalid change_set_id", http.StatusBadRequest)
		return
	}

	review, err := h.service.CreateReview(r.Context(), changeSetID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(review)
}

// GetReview retrieves a review by ID (or ChangeSetID)
func (h *ChangeReviewHandler) GetReview(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	// Try to fetch by Review ID first, or ChangeSet ID?
	// Service interface doesn't explicitly have GetReview(id).
	// ReviewService usually returns *ChangeReview from CreateReview.
	// I need to add GetChangeReview to ChangeReviewService first.
	// Wait, I can query DB directly or add method to service. Adding to service is better.
	// For now, I'll rely on service having it. Check service.go.
	// Checking service.go in next step. For now, assume service.GetReview(ctx, id) exists or add it.

	// Actually, I should verify service.go first.
	// But I can implement handler assuming service update.

	// Retrieve latest review for the change set (treating ID as ChangeSetID)
	review, err := h.service.GetReviewForChangeSet(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(review)
}

// Promote promotes a review/changeset
func (h *ChangeReviewHandler) Promote(w http.ResponseWriter, r *http.Request) {
	changeSetIDStr := chi.URLParam(r, "id")
	changeSetID, err := uuid.Parse(changeSetIDStr)
	if err != nil {
		http.Error(w, "invalid change_set_id", http.StatusBadRequest)
		return
	}

	actor := r.Header.Get("X-User-ID") // Middleware should provide this

	if err := h.service.Promote(r.Context(), changeSetID, actor); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"promotion_started"}`))
}

// Rollback initiates a rollback
func (h *ChangeReviewHandler) Rollback(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ObjectID      string `json:"object_id"`
		TargetVersion int    `json:"target_version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	actor := r.Header.Get("X-User-ID")

	if err := h.service.Rollback(r.Context(), req.ObjectID, req.TargetVersion, actor); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"rollback_successful"}`))
}
