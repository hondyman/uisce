package bo

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
)

type Handler struct {
	discoverySvc *DiscoveryService
	boSvc        *BusinessObjectService
}

func NewHandler(d *DiscoveryService, b *BusinessObjectService) *Handler {
	return &Handler{discoverySvc: d, boSvc: b}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/api/business-objects", func(r chi.Router) {
		r.Post("/binding-context", h.GetBindingContext)
		r.Post("/save", h.SaveBusinessObject)
	})
}

func (h *Handler) GetBindingContext(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	tenantID, _ := uuid.Parse(claims.TenantID)

	var req BindingContextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.TenantID = tenantID

	resp, err := h.discoverySvc.ResolveBindingContext(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) SaveBusinessObject(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	tenantID, _ := uuid.Parse(claims.TenantID)

	var req SaveBusinessObjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request payload", http.StatusBadRequest)
		return
	}
	req.TenantID = tenantID

	actorRole := "USER"
	if len(claims.Roles) > 0 {
		actorRole = claims.Roles[0]
	}
	boID, err := h.boSvc.SaveBusinessObjectAtomic(r.Context(), claims.UserID, actorRole, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"boId":   boID,
		"status": "SUCCESS",
	})
}
