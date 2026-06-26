package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/hondyman/semlayer/backend/models"
)

// AbacHandler handles API requests for the ABAC service.
type AbacHandler struct {
	svc *services.AbacService
}

// NewAbacHandler creates a new AbacHandler.
func NewAbacHandler(svc *services.AbacService) *AbacHandler {
	return &AbacHandler{svc: svc}
}

// RegisterRoutes mounts ABAC routes
func (h *AbacHandler) RegisterRoutes(r chi.Router) {
	r.Route("/abac", func(r chi.Router) {
		r.Post("/evaluate", h.Evaluate)
		r.Route("/policies", func(r chi.Router) {
			r.Post("/", h.CreatePolicy)
			r.Get("/", h.ListPolicies)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", h.GetPolicy)
				r.Put("/", h.UpdatePolicy)
				r.Delete("/", h.DeletePolicy)
			})
		})
		r.Route("/resources", func(r chi.Router) {
			r.Post("/", h.CreateResource)
			r.Get("/", h.ListResources)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", h.GetResource)
				r.Put("/", h.UpdateResource)
				r.Delete("/", h.DeleteResource)
			})
		})
	})
}

// Evaluate handles a real-time access evaluation request.
func (h *AbacHandler) Evaluate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Subject  map[string]any `json:"subject"`
		Action   string         `json:"action"`
		Resource map[string]any `json:"resource"`
		Env      map[string]any `json:"env"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	allowed, reason, err := h.svc.EvaluateAccess(r.Context(), req.Subject, req.Action, req.Resource, req.Env)
	if err != nil {
		http.Error(w, "Failed during access evaluation: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"allowed": allowed, "reason": reason})
}

// --- Policy Handlers ---

// CreatePolicy handles POST /policies
func (h *AbacHandler) CreatePolicy(w http.ResponseWriter, r *http.Request) {
	var policy models.Policy
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		http.Error(w, "Invalid request payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	newPolicy, err := h.svc.CreatePolicy(r.Context(), &policy)
	if err != nil {
		http.Error(w, "Failed to create policy: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newPolicy)
}

// ListPolicies handles GET /policies
func (h *AbacHandler) ListPolicies(w http.ResponseWriter, r *http.Request) {
	policies, err := h.svc.ListPolicies(r.Context())
	if err != nil {
		http.Error(w, "Failed to list policies: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(policies)
}

// GetPolicy handles GET /policies/:id
func (h *AbacHandler) GetPolicy(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid policy ID format", http.StatusBadRequest)
		return
	}

	policy, err := h.svc.GetPolicy(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(policy)
}

// UpdatePolicy handles PUT /policies/:id
func (h *AbacHandler) UpdatePolicy(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid policy ID format", http.StatusBadRequest)
		return
	}

	var policy models.Policy
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		http.Error(w, "Invalid request payload: "+err.Error(), http.StatusBadRequest)
		return
	}
	policy.ID = id // Ensure ID from URL is used

	if err := h.svc.UpdatePolicy(r.Context(), &policy); err != nil {
		http.Error(w, "Failed to update policy: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeletePolicy handles DELETE /policies/:id
func (h *AbacHandler) DeletePolicy(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid policy ID format", http.StatusBadRequest)
		return
	}

	if err := h.svc.DeletePolicy(r.Context(), id); err != nil {
		http.Error(w, "Failed to delete policy: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Resource Handlers ---

// CreateResource handles POST /resources
func (h *AbacHandler) CreateResource(w http.ResponseWriter, r *http.Request) {
	var resource models.Resource
	if err := json.NewDecoder(r.Body).Decode(&resource); err != nil {
		http.Error(w, "Invalid request payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	newResource, err := h.svc.CreateResource(r.Context(), &resource)
	if err != nil {
		http.Error(w, "Failed to create resource: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newResource)
}

// ListResources handles GET /resources
func (h *AbacHandler) ListResources(w http.ResponseWriter, r *http.Request) {
	resources, err := h.svc.ListResources(r.Context())
	if err != nil {
		http.Error(w, "Failed to list resources: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resources)
}

// GetResource handles GET /resources/:id
func (h *AbacHandler) GetResource(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid resource ID format", http.StatusBadRequest)
		return
	}

	resource, err := h.svc.GetResource(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resource)
}

// UpdateResource handles PUT /resources/:id
func (h *AbacHandler) UpdateResource(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid resource ID format", http.StatusBadRequest)
		return
	}

	var resource models.Resource
	if err := json.NewDecoder(r.Body).Decode(&resource); err != nil {
		http.Error(w, "Invalid request payload: "+err.Error(), http.StatusBadRequest)
		return
	}
	resource.ID = id // Ensure ID from URL is used

	if err := h.svc.UpdateResource(r.Context(), &resource); err != nil {
		http.Error(w, "Failed to update resource: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteResource handles DELETE /resources/:id
func (h *AbacHandler) DeleteResource(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid resource ID format", http.StatusBadRequest)
		return
	}

	if err := h.svc.DeleteResource(r.Context(), id); err != nil {
		http.Error(w, "Failed to delete resource: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
