package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/apistudio"
)

// APIStudioHandler handles management operations for the API Studio
type APIStudioHandler struct {
	service *apistudio.Service
}

// NewAPIStudioHandler creates a new handler
func NewAPIStudioHandler(service *apistudio.Service) *APIStudioHandler {
	return &APIStudioHandler{service: service}
}

// Routes returns the router for API Studio management
func (h *APIStudioHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/endpoints", h.ListEndpoints)
	r.Post("/endpoints", h.SaveEndpoint)
	r.Get("/endpoints/{id}", h.GetEndpoint)
	r.Post("/endpoints/{id}/deprecate", h.DeprecateEndpoint)
	r.Post("/endpoints/{id}/retire", h.RetireEndpoint)
	r.Get("/openapi", h.GetOpenAPI)
	r.Get("/sdk/{lang}", h.DownloadSDK)
	r.Post("/endpoints/ai", h.AIGenerateEndpoint)
	return r
}

// ListEndpoints lists all API endpoints
func (h *APIStudioHandler) ListEndpoints(w http.ResponseWriter, r *http.Request) {
	env := r.URL.Query().Get("env")
	tenantID := r.URL.Query().Get("tenant_id")

	eps, err := h.service.GetRepository().ListEndpoints(r.Context(), env, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(eps)
}

// GetEndpoint retrieves a single endpoint
func (h *APIStudioHandler) GetEndpoint(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	ep, err := h.service.GetRepository().GetEndpoint(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(ep)
}

// SaveEndpoint creates or updates an endpoint
func (h *APIStudioHandler) SaveEndpoint(w http.ResponseWriter, r *http.Request) {
	var ep apistudio.APIEndpoint
	if err := json.NewDecoder(r.Body).Decode(&ep); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if ep.ID == uuid.Nil {
		ep.ID = uuid.New()
	}

	// Use actor from context/header
	actor := r.Header.Get("X-User-ID")
	if actor == "" {
		actor = "system"
	}

	if err := h.service.SaveEndpoint(r.Context(), &ep, actor); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(ep)
}

// GetOpenAPI generates the OpenAPI spec for a tenant
func (h *APIStudioHandler) GetOpenAPI(w http.ResponseWriter, r *http.Request) {
	env := r.URL.Query().Get("env")
	tenantID := r.URL.Query().Get("tenant_id")

	eps, err := h.service.GetRepository().ListEndpoints(r.Context(), env, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	spec, err := apistudio.GenerateOpenAPI(env, tenantID, eps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(spec))
}

// DownloadSDK generates and returns a client SDK
func (h *APIStudioHandler) DownloadSDK(w http.ResponseWriter, r *http.Request) {
	lang := chi.URLParam(r, "lang")
	env := r.URL.Query().Get("env")
	tenantID := r.URL.Query().Get("tenant_id")

	eps, err := h.service.GetRepository().ListEndpoints(r.Context(), env, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var sdk string
	switch lang {
	case "typescript":
		sdk = apistudio.GenerateTypeScriptSDK(env, tenantID, eps)
		w.Header().Set("Content-Disposition", "attachment; filename=api-client.ts")
	default:
		http.Error(w, "Unsupported language", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(sdk))
}

// DeprecateEndpoint marks an endpoint as deprecated
func (h *APIStudioHandler) DeprecateEndpoint(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	// Use actor from context/header
	actor := r.Header.Get("X-User-ID")
	if actor == "" {
		actor = "system"
	}

	if err := h.service.DeprecateEndpoint(r.Context(), id.String(), actor); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch updated endpoint to return
	ep, err := h.service.GetRepository().GetEndpoint(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(ep)
}

// RetireEndpoint marks an endpoint as retired
func (h *APIStudioHandler) RetireEndpoint(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	// Use actor from context/header
	actor := r.Header.Get("X-User-ID")
	if actor == "" {
		actor = "system"
	}

	if err := h.service.RetireEndpoint(r.Context(), id.String(), actor); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch updated endpoint to return
	ep, err := h.service.GetRepository().GetEndpoint(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(ep)
}

// AIGenerateEndpoint uses AI to propose a new endpoint configuration
func (h *APIStudioHandler) AIGenerateEndpoint(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Prompt   string `json:"prompt"`
		TenantID string `json:"tenant_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ep, err := h.service.AIGenerateEndpoint(r.Context(), req.Prompt, req.TenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(ep)
}
