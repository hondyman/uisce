package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/auth"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/services"
)

// RoleHandler encapsulates the handlers for the role management API.
type RoleHandler struct {
	Service services.RoleService
}

// NewRoleHandler creates a new role handler.
func NewRoleHandler(service services.RoleService) *RoleHandler {
	return &RoleHandler{Service: service}
}

// RegisterRoutes adds the role management routes to the router.
func (h *RoleHandler) RegisterRoutes(r chi.Router) {
	// Register routes relative to the /api group so they resolve to /api/roles
	r.Route("/roles", func(r chi.Router) {
		r.Get("/", h.listRoles)
		r.Post("/", h.createRole)
		r.Route("/{roleName}", func(r chi.Router) {
			r.Get("/", h.getRole)
			r.Put("/", h.updateRole)
			r.Delete("/", h.deleteRole)
			r.Get("/bundles", h.getBundlesForRole)
			r.Post("/bundles", h.assignBundleToRole)
			r.Delete("/bundles/{bundleID}", h.unassignBundleFromRole)
		})
	})
}

func (h *RoleHandler) listRoles(w http.ResponseWriter, r *http.Request) {
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

	roles, err := h.Service.ListRoles(r.Context(), user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(roles)
}

func (h *RoleHandler) createRole(w http.ResponseWriter, r *http.Request) {
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

	var input services.RoleCreateInput

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	role, err := h.Service.CreateRole(r.Context(), user, input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(role)
}

func (h *RoleHandler) getRole(w http.ResponseWriter, r *http.Request) {
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
	roleName := chi.URLParam(r, "roleName")

	role, err := h.Service.GetRole(r.Context(), user, roleName)
	if err != nil {
		if errors.Is(err, services.ErrRoleNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(role)
}

func (h *RoleHandler) updateRole(w http.ResponseWriter, r *http.Request) {
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

	roleName := chi.URLParam(r, "roleName")

	var input services.RoleUpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	updated, err := h.Service.UpdateRole(r.Context(), user, roleName, input)
	if err != nil {
		if errors.Is(err, services.ErrRoleNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(updated)
}

func (h *RoleHandler) deleteRole(w http.ResponseWriter, r *http.Request) {
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

	roleName := chi.URLParam(r, "roleName")

	if err := h.Service.DeleteRole(r.Context(), user, roleName); err != nil {
		if errors.Is(err, services.ErrRoleNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *RoleHandler) getBundlesForRole(w http.ResponseWriter, r *http.Request) {
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
	roleName := chi.URLParam(r, "roleName")

	bundleIDs, err := h.Service.GetBundleIDsForRole(r.Context(), user, roleName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(bundleIDs)
}

func (h *RoleHandler) assignBundleToRole(w http.ResponseWriter, r *http.Request) {
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
	roleName := chi.URLParam(r, "roleName")

	var req struct {
		BundleID string `json:"bundleId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := h.Service.AssignBundleToRole(r.Context(), user, roleName, req.BundleID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *RoleHandler) unassignBundleFromRole(w http.ResponseWriter, r *http.Request) {
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
	roleName := chi.URLParam(r, "roleName")
	bundleID := chi.URLParam(r, "bundleID")

	err := h.Service.UnassignBundleFromRole(r.Context(), user, roleName, bundleID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
