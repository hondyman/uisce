package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/component_extensibility/forms"
	"github.com/hondyman/uisce/backend/internal/component_extensibility/micro"
	"github.com/hondyman/uisce/backend/internal/component_extensibility/recipes"
	"github.com/hondyman/uisce/libs/jwt-middleware"
)

type ComponentExtensibilityHandler struct {
	recipeRegistry   *recipes.Registry
	compositeManager *micro.CompositeManager
	formGenerator    *forms.FormGenerator
}

func NewComponentExtensibilityHandler(
	rr *recipes.Registry,
	cm *micro.CompositeManager,
	fg *forms.FormGenerator,
) *ComponentExtensibilityHandler {
	return &ComponentExtensibilityHandler{
		recipeRegistry:   rr,
		compositeManager: cm,
		formGenerator:    fg,
	}
}

func (h *ComponentExtensibilityHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Recipes
	r.Get("/recipes", h.ListRecipes)
	r.Post("/recipes/{id}/instantiate", h.InstantiateRecipe)

	// Micro-Components
	r.Post("/micro/expand/{id}", h.ExpandMicroComponent)

	// Forms
	r.Post("/forms/generate", h.GenerateForm)

	return r
}

func (h *ComponentExtensibilityHandler) ListRecipes(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID // Middleware would handle this
	list, _ := h.recipeRegistry.List(r.Context(), tenantID)
	json.NewEncoder(w).Encode(list)
}

func (h *ComponentExtensibilityHandler) InstantiateRecipe(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	instance, _ := h.recipeRegistry.Instantiate(r.Context(), id)
	w.Header().Set("Content-Type", "application/json")
	w.Write(instance)
}

func (h *ComponentExtensibilityHandler) ExpandMicroComponent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	node, _ := h.compositeManager.Expand(r.Context(), id)
	json.NewEncoder(w).Encode(node)
}

func (h *ComponentExtensibilityHandler) GenerateForm(w http.ResponseWriter, r *http.Request) {
	var bo forms.BusinessObject
	if err := json.NewDecoder(r.Body).Decode(&bo); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	form, _ := h.formGenerator.Generate(r.Context(), bo)
	json.NewEncoder(w).Encode(form)
}
