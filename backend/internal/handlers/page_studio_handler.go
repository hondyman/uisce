package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/pagestudio"
)

// ContextKey is a custom type for context keys to avoid collisions
type ContextKey string

const RegionContextKey ContextKey = "region"

// PageStudioHandler handles management operations for the Page Studio
type PageStudioHandler struct {
	service       *pagestudio.Service
	bundleService *pagestudio.PageBundleService
}

// NewPageStudioHandler creates a new handler
func NewPageStudioHandler(service *pagestudio.Service, bundleService *pagestudio.PageBundleService) *PageStudioHandler {
	return &PageStudioHandler{service: service, bundleService: bundleService}
}

// Routes returns the router for Page Studio management
func (h *PageStudioHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/pages", h.ListPages)
	r.Post("/pages", h.SavePage)
	r.Get("/pages/{id}", h.GetPage)
	r.Get("/pages/slug/{slug}", h.GetPageBySlug)
	r.Get("/pages/{id}/overlay", h.GetOverlay)
	r.Post("/pages/{id}/overlay", h.SaveOverlay)
	r.Post("/pages/{id}/overlay", h.SaveOverlay)
	r.Post("/ai/generate-layout", h.GenerateLayout)
	r.Get("/runtime/page-bundle/{slug}", h.GetPageBundle)
	return r
}

// ListPages lists all core pages
func (h *PageStudioHandler) ListPages(w http.ResponseWriter, r *http.Request) {
	env := r.URL.Query().Get("env")
	pages, err := h.service.GetRepository().ListPages(r.Context(), env)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(pages)
}

// GetPage retrieves a single core page
func (h *PageStudioHandler) GetPage(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	p, err := h.service.GetRepository().GetPage(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(p)
}

// GetPageBySlug retrieves a core page by slug
func (h *PageStudioHandler) GetPageBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	env := r.URL.Query().Get("env")
	p, err := h.service.GetRepository().GetPageBySlug(r.Context(), slug, env)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(p)
}

// SavePage creates or updates a core page
func (h *PageStudioHandler) SavePage(w http.ResponseWriter, r *http.Request) {
	var p pagestudio.CorePage
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	actor := r.Header.Get("X-User-ID")
	if actor == "" {
		actor = "system"
	}

	if err := h.service.SavePage(r.Context(), &p, actor); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(p)
}

// GetOverlay retrieves an overlay for a page and tenant
func (h *PageStudioHandler) GetOverlay(w http.ResponseWriter, r *http.Request) {
	parentID, _ := uuid.Parse(chi.URLParam(r, "id"))
	tenantID := r.URL.Query().Get("tenant_id")
	env := r.URL.Query().Get("env")

	o, err := h.service.GetRepository().GetOverlay(r.Context(), parentID, tenantID, env)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(o)
}

// SaveOverlay creates or updates an overlay
func (h *PageStudioHandler) SaveOverlay(w http.ResponseWriter, r *http.Request) {
	var o pagestudio.PageOverlay
	if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	actor := r.Header.Get("X-User-ID")
	if actor == "" {
		actor = "system"
	}

	if err := h.service.SaveOverlay(r.Context(), &o, actor); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(o)
}

// GenerateLayout generates a page layout using AI
func (h *PageStudioHandler) GenerateLayout(w http.ResponseWriter, r *http.Request) {
	var req pagestudio.AIGenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := h.service.GenerateLayout(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(resp)
}

// GetPageBundle retrieves the data bundle for a page
func (h *PageStudioHandler) GetPageBundle(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	env := r.URL.Query().Get("env")
	if env == "" {
		env = "production"
	}
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id required", http.StatusBadRequest)
		return
	}

	// Route params from query
	routeParams := make(map[string]string)
	for k, v := range r.URL.Query() {
		if len(v) > 0 {
			routeParams[k] = v[0]
		}
	}

	actor := r.Header.Get("X-User-ID")
	var actorPtr *string
	if actor != "" {
		actorPtr = &actor
	}

	region := ""
	if rg, ok := r.Context().Value(RegionContextKey).(string); ok {
		region = rg
	}

	results, err := h.bundleService.ExecuteBundle(r.Context(), tenantID, slug, routeParams, env, actorPtr, region)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return bundle structure
	resp := map[string]interface{}{
		"pageSlug": slug,
		"tenantID": tenantID,
		"data":     results,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
