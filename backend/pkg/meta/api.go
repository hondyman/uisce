package meta

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// API handlers for metadata management
type API struct {
	service   *Service
	hasuraGen *HasuraMetadataGenerator
}

func NewAPI(service *Service, hasuraGen *HasuraMetadataGenerator) *API {
	return &API{
		service:   service,
		hasuraGen: hasuraGen,
	}
}

func (api *API) RegisterRoutes(r chi.Router) {
	r.Get("/meta/business-objects", api.listBusinessObjects)
	r.Post("/meta/business-objects", api.createBusinessObject)
	r.Get("/meta/business-objects/{id}", api.getBusinessObject)
	r.Put("/meta/business-objects/{id}", api.updateBusinessObject)
	r.Delete("/meta/business-objects/{id}", api.deleteBusinessObject)
	r.Post("/meta/business-objects/{id}/hasura", api.generateHasuraMetadata)
}

func (api *API) listBusinessObjects(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id required", http.StatusBadRequest)
		return
	}

	objects, err := api.service.ListBusinessObjects(r.Context(), tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(objects)
}

func (api *API) createBusinessObject(w http.ResponseWriter, r *http.Request) {
	var bo BusinessObjectDefinition
	if err := json.NewDecoder(r.Body).Decode(&bo); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := api.service.CreateBusinessObject(r.Context(), &bo); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(bo)
}

func (api *API) getBusinessObject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	bo, err := api.service.GetBusinessObject(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bo)
}

func (api *API) updateBusinessObject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var bo BusinessObjectDefinition
	if err := json.NewDecoder(r.Body).Decode(&bo); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	bo.ID = id

	if err := api.service.UpdateBusinessObject(r.Context(), &bo); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bo)
}

func (api *API) deleteBusinessObject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := api.service.DeleteBusinessObject(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (api *API) generateHasuraMetadata(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	bo, err := api.service.GetBusinessObject(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if err := api.hasuraGen.GenerateAndApply(r.Context(), bo); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "metadata generated and applied",
	})
}
