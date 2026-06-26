package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/auth"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/services"
)

// BundleHandler encapsulates the handlers for the bundle API.
type BundleHandler struct {
	Service services.BundleService
}

// NewBundleHandler creates a new bundle handler.
func NewBundleHandler(service services.BundleService) *BundleHandler {
	return &BundleHandler{Service: service}
}

// RegisterRoutes adds the bundle routes to the router.
func (h *BundleHandler) RegisterRoutes(r chi.Router) {
	r.Route("/bundles", func(r chi.Router) {
		r.Post("/", h.createBundle)
		r.Get("/", h.listBundles)
		r.Route("/{bundleID}", func(r chi.Router) {
			r.Get("/", h.getBundle)
			r.Put("/", h.updateBundle)
			r.Put("/policies", h.updateBundlePolicies)
			r.Post("/certify", h.certifyBundle)
			r.Post("/publish", h.publishBundle)
			r.Post("/deprecate", h.deprecateBundle)
		})
	})
}

func (h *BundleHandler) createBundle(w http.ResponseWriter, r *http.Request) {
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

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	bundle, err := h.Service.CreateBundle(user, req.Name, req.Description)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(bundle)
}

func (h *BundleHandler) listBundles(w http.ResponseWriter, r *http.Request) {
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

	bundles, err := h.Service.ListBundles(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(bundles)
}

func (h *BundleHandler) getBundle(w http.ResponseWriter, r *http.Request) {
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
	bundleID := chi.URLParam(r, "bundleID")

	bundle, err := h.Service.GetBundle(user, bundleID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(bundle)
}

func (h *BundleHandler) updateBundle(w http.ResponseWriter, r *http.Request) {
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
	bundleID := chi.URLParam(r, "bundleID")

	var req struct {
		Measures   []models.SemanticObjectReference `json:"measures"`
		Dimensions []models.SemanticObjectReference `json:"dimensions"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	bundle, err := h.Service.UpdateBundle(user, bundleID, req.Measures, req.Dimensions)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(bundle)
}

func (h *BundleHandler) updateBundlePolicies(w http.ResponseWriter, r *http.Request) {
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

	bundleID := chi.URLParam(r, "bundleID")

	var req struct {
		RowPolicies    []models.BundleRowPolicy    `json:"rowPolicies"`
		ColumnPolicies []models.BundleColumnPolicy `json:"columnPolicies"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	bundle, err := h.Service.UpdateBundlePolicies(user, bundleID, req.RowPolicies, req.ColumnPolicies)
	if err != nil {
		var valErr *services.ValidationError
		if errors.As(err, &valErr) {
			w.WriteHeader(http.StatusUnprocessableEntity)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error":   "validation_failed",
				"message": valErr.Message,
				"details": valErr.Errors,
			})
			return
		}

		switch {
		case err.Error() == fmt.Sprintf("bundle with id %s not found", bundleID):
			http.Error(w, err.Error(), http.StatusNotFound)
		case strings.Contains(err.Error(), "permission"):
			http.Error(w, err.Error(), http.StatusForbidden)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	json.NewEncoder(w).Encode(bundle)
}

func (h *BundleHandler) certifyBundle(w http.ResponseWriter, r *http.Request) {
	h.handleStatusChange(w, r, h.Service.CertifyBundle)
}

func (h *BundleHandler) publishBundle(w http.ResponseWriter, r *http.Request) {
	h.handleStatusChange(w, r, h.Service.PublishBundle)
}

func (h *BundleHandler) deprecateBundle(w http.ResponseWriter, r *http.Request) {
	h.handleStatusChange(w, r, h.Service.DeprecateBundle)
}

// handleStatusChange is a helper for status transition handlers.
func (h *BundleHandler) handleStatusChange(w http.ResponseWriter, r *http.Request, statusChangeFunc func(models.User, string) (*models.DataBundle, error)) {
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
	bundleID := chi.URLParam(r, "bundleID")

	bundle, err := statusChangeFunc(user, bundleID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(bundle)
}

// getUserFromContext is a placeholder function. In a real application,
// this would be middleware that extracts user info from a JWT or session.
// getUserFromContext returns the user from the typed auth context if present,
// otherwise falls back to a prototype steward user for local development parity.
// NOTE: handlers now use auth.GetUserFromContext directly with an inline fallback steward user.
