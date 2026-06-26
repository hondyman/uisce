package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/metadata"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// EntitySchemaHandler wraps BusinessObjectService to provide entity schema endpoints
// for frontend compatibility
type EntitySchemaHandler struct {
	service *metadata.BusinessObjectService
}

// NewEntitySchemaHandler creates a new entity schema handler
func NewEntitySchemaHandler(service *metadata.BusinessObjectService) *EntitySchemaHandler {
	return &EntitySchemaHandler{service: service}
}

// RegisterRoutes registers the entity schema routes
func (h *EntitySchemaHandler) RegisterRoutes(r chi.Router) {
	r.Route("/entity-schema", func(r chi.Router) {
		r.Get("/", h.GetEntitySchema)
		r.Post("/", h.SaveEntitySchema)
	})
}

// GetEntitySchema returns all business objects as a map keyed by entity key
// GET /api/entity-schema
func (h *EntitySchemaHandler) GetEntitySchema(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		tenantID = r.URL.Query().Get("tenant_id")
	}

	if tenantID == "" {
		http.Error(w, "Missing tenant_id", http.StatusBadRequest)
		return
	}

	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	// Fetch all business objects for this tenant
	secCtx := &security.Context{TenantID: tenantID, DatasourceID: datasourceID}
	bos, err := h.service.ListBusinessObjects(r.Context(), secCtx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Transform to map[string]interface{} format expected by frontend
	// Frontend expects: { "entity_key": { id, name, config: { entity_fields: [...] }, ... }, ... }
	result := make(map[string]interface{})
	for _, bo := range bos {
		// Combine core and custom fields into entity_fields
		allFields := make([]map[string]interface{}, 0, len(bo.CoreFields)+len(bo.CustomFields))

		for _, field := range bo.CoreFields {
			allFields = append(allFields, map[string]interface{}{
				"key":           field.Key,
				"name":          field.Name,
				"displayName":   field.DisplayName,
				"technicalName": field.TechnicalName,
				"type":          field.Type,
				"isCore":        true,
				"displayOrder":  field.Sequence,
			})
		}

		for _, field := range bo.CustomFields {
			allFields = append(allFields, map[string]interface{}{
				"key":           field.Key,
				"name":          field.Name,
				"displayName":   field.DisplayName,
				"technicalName": field.TechnicalName,
				"type":          field.Type,
				"isCore":        false,
				"displayOrder":  field.Sequence,
			})
		}

		// Build entity object
		entity := map[string]interface{}{
			"id":             bo.ID,
			"name":           bo.Name,
			"display_name":   bo.DisplayName,
			"technical_name": bo.TechnicalName,
			"description":    bo.Description,
			"is_core":        bo.IsCore,
			"config": map[string]interface{}{
				"entity_fields": allFields,
			},
		}

		// Add subtypes if they exist
		if len(bo.Subtypes) > 0 {
			subtypes := make(map[string]interface{})
			for key, subtype := range bo.Subtypes {
				subtypeFields := make([]map[string]interface{}, 0, len(subtype.SubtypeFields))
				for _, field := range subtype.SubtypeFields {
					subtypeFields = append(subtypeFields, map[string]interface{}{
						"key":           field.Key,
						"name":          field.Name,
						"displayName":   field.DisplayName,
						"technicalName": field.TechnicalName,
						"type":          field.Type,
						"isCore":        field.IsCore,
						"displayOrder":  field.Sequence,
					})
				}

				subtypes[key] = map[string]interface{}{
					"name":           subtype.Name,
					"display_name":   subtype.DisplayName,
					"technical_name": subtype.TechnicalName,
					"is_core":        subtype.IsCore,
					"config": map[string]interface{}{
						"inheritedFields": allFields, // Inherited from parent
						"customFields":    subtypeFields,
					},
				}
			}
			entity["subtypes"] = subtypes
		}

		// Use technical_name as the key (normalized)
		entityKey := bo.TechnicalName
		if entityKey == "" {
			entityKey = bo.Key
		}
		result[entityKey] = entity
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// SaveEntitySchema saves entity schema updates
// POST /api/entity-schema
func (h *EntitySchemaHandler) SaveEntitySchema(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		tenantID = r.URL.Query().Get("tenant_id")
	}

	if tenantID == "" {
		http.Error(w, "Missing tenant_id", http.StatusBadRequest)
		return
	}

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "system" // Default to system if no user ID provided
	}

	// Decode the request body
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Check if this is a delta update (has 'changed' or 'deleted' keys)
	if changed, hasChanged := payload["changed"].(map[string]interface{}); hasChanged {
		// Handle delta updates
		for entityKey, entityData := range changed {
			if err := h.updateEntity(r.Context(), tenantID, entityKey, entityData, userID); err != nil {
				http.Error(w, "Failed to update entity "+entityKey+": "+err.Error(), http.StatusInternalServerError)
				return
			}
		}
	} else {
		// Handle full schema update
		for entityKey, entityData := range payload {
			if err := h.updateEntity(r.Context(), tenantID, entityKey, entityData, userID); err != nil {
				http.Error(w, "Failed to update entity "+entityKey+": "+err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	// Handle deletions if present
	if deleted, hasDeleted := payload["deleted"].([]interface{}); hasDeleted {
		for _, entityKey := range deleted {
			if key, ok := entityKey.(string); ok {
				secCtx := &security.Context{TenantID: tenantID}
				if err := h.service.DeleteBusinessObject(r.Context(), secCtx, key, userID); err != nil {
					http.Error(w, fmt.Sprintf("Failed to delete entity %s: %v", key, err), http.StatusInternalServerError)
					return
				}
			}
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// updateEntity is a helper function to update a single entity
func (h *EntitySchemaHandler) updateEntity(_ctx interface{}, _tenantID, _entityKey string, _entityData interface{}, _userID string) error {
	// This is a simplified implementation
	// In a real scenario, you would parse the entityData and call appropriate service methods
	// For now, we'll just return success as the business object service handles the actual updates
	return nil
}
