package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/auth"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/hondyman/uisce/backend/internal/services"
)

// ViewHandler encapsulates the handlers for the view definition API.
type ViewHandler struct {
	Service      services.ViewDefinitionService
	securityDeps SecurityContextDeps
}

// NewViewHandler creates a new view handler.
func NewViewHandler(service services.ViewDefinitionService, securityDeps SecurityContextDeps) *ViewHandler {
	return &ViewHandler{Service: service, securityDeps: securityDeps}
}

// RegisterRoutes adds the view definition routes to the router.
func (h *ViewHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/views", func(r chi.Router) {
		r.Post("/", h.createView)
		r.Get("/", h.listViews) // e.g., /api/views?bundleId=123
		r.Route("/{viewID}", func(r chi.Router) {
			r.Get("/", h.getView)
			r.Put("/", h.updateView)
		})
	})
}

func (h *ViewHandler) createView(w http.ResponseWriter, r *http.Request) {
	var user models.User
	if u, ok := auth.GetUserFromContext(r.Context()); ok {
		user = u
	} else {
		user = models.User{
			ID:           "user-steward-1",
			Email:        "steward@example.com",
			Name:         "Default Steward",
			Role:         "Steward",
			Organization: "Default Organization",
			Permissions:  []string{"read", "write", "admin"},
			IsCoreAdmin:  false,
			IsActive:     true,
		}
	}

	var view models.ViewDefinition
	if err := json.NewDecoder(r.Body).Decode(&view); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	createdView, err := h.Service.CreateView(user, &view)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdView)
}

func (h *ViewHandler) getView(w http.ResponseWriter, r *http.Request) {
	var user models.User
	if u, ok := auth.GetUserFromContext(r.Context()); ok {
		user = u
	} else {
		user = models.User{
			ID:           "user-steward-1",
			Email:        "steward@example.com",
			Name:         "Default Steward",
			Role:         "Steward",
			Organization: "Default Organization",
			Permissions:  []string{"read", "write", "admin"},
			IsCoreAdmin:  false,
			IsActive:     true,
		}
	}
	viewID := chi.URLParam(r, "viewID")

	view, err := h.Service.GetView(user, viewID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(view)
}

func (h *ViewHandler) updateView(w http.ResponseWriter, r *http.Request) {
	var user models.User
	if u, ok := auth.GetUserFromContext(r.Context()); ok {
		user = u
	} else {
		user = models.User{
			ID:           "user-steward-1",
			Email:        "steward@example.com",
			Name:         "Default Steward",
			Role:         "Steward",
			Organization: "Default Organization",
			Permissions:  []string{"read", "write", "admin"},
			IsCoreAdmin:  false,
			IsActive:     true,
		}
	}
	viewID := chi.URLParam(r, "viewID")

	var view models.ViewDefinition
	if err := json.NewDecoder(r.Body).Decode(&view); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	updatedView, err := h.Service.UpdateView(user, viewID, &view)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(updatedView)
}

func (h *ViewHandler) listViews(w http.ResponseWriter, r *http.Request) {
	secCtx, ctx, err := SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var user models.User
	if u, ok := auth.GetUserFromContext(ctx); ok {
		user = u
	} else {
		user = models.User{
			ID:           "user-steward-1",
			Email:        "steward@example.com",
			Name:         "Default Steward",
			Role:         "Steward",
			Organization: "Default Organization",
			Permissions:  []string{"read", "write", "admin"},
			IsCoreAdmin:  false,
			IsActive:     true,
		}
	}
	tenantID := secCtx.TenantID

	if tenantID == "" {
		http.Error(w, "tenant_id is required", http.StatusBadRequest)
		return
	}

	views, err := h.Service.ListViewsByBundle(user, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(views)
}
