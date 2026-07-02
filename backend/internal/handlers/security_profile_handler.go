package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// SecurityProfileHandler manages API endpoints for Security Profiles and mappings.
type SecurityProfileHandler struct {
	svc *security.ProfileService
}

// NewSecurityProfileHandler creates a new SecurityProfileHandler.
func NewSecurityProfileHandler(svc *security.ProfileService) *SecurityProfileHandler {
	return &SecurityProfileHandler{svc: svc}
}

// RegisterRoutes registers routes on the router.
func (h *SecurityProfileHandler) RegisterRoutes(r chi.Router) {
	r.Route("/security", func(r chi.Router) {
		r.Route("/profiles", func(r chi.Router) {
			r.Post("/", h.CreateProfile)
			r.Get("/", h.ListProfiles)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", h.GetProfile)
				r.Put("/", h.UpdateProfile)
				r.Delete("/", h.DeleteProfile)
			})
		})
		r.Route("/mappings", func(r chi.Router) {
			r.Post("/", h.CreateMapping)
			r.Get("/", h.ListMappings)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", h.GetMapping)
				r.Put("/", h.UpdateMapping)
				r.Delete("/", h.DeleteMapping)
			})
		})
	})
}

// --- Profiles ---

func (h *SecurityProfileHandler) CreateProfile(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil || claims.TenantID == "" {
		http.Error(w, `{"error":"unauthorized or missing tenant scope"}`, http.StatusUnauthorized)
		return
	}
	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		http.Error(w, `{"error":"invalid tenant id"}`, http.StatusBadRequest)
		return
	}

	var req struct {
		ProfileKey      string     `json:"profile_key"`
		ProfileName     string     `json:"profile_name"`
		ParentProfileID *uuid.UUID `json:"parent_profile_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	p := &security.SecurityProfile{
		TenantID:        &tenantID,
		ProfileKey:      req.ProfileKey,
		ProfileName:     req.ProfileName,
		ParentProfileID: req.ParentProfileID,
	}

	res, err := h.svc.CreateProfile(r.Context(), p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(res)
}

func (h *SecurityProfileHandler) ListProfiles(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil || claims.TenantID == "" {
		http.Error(w, `{"error":"unauthorized or missing tenant scope"}`, http.StatusUnauthorized)
		return
	}
	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		http.Error(w, `{"error":"invalid tenant id"}`, http.StatusBadRequest)
		return
	}

	res, err := h.svc.ListProfiles(r.Context(), tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (h *SecurityProfileHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	res, err := h.svc.GetProfile(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (h *SecurityProfileHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil || claims.TenantID == "" {
		http.Error(w, `{"error":"unauthorized or missing tenant scope"}`, http.StatusUnauthorized)
		return
	}
	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		http.Error(w, `{"error":"invalid tenant id"}`, http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	var req struct {
		ProfileName     string     `json:"profile_name"`
		ParentProfileID *uuid.UUID `json:"parent_profile_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	p := &security.SecurityProfile{
		ProfileID:       id,
		TenantID:        &tenantID,
		ProfileName:     req.ProfileName,
		ParentProfileID: req.ParentProfileID,
	}

	if err := h.svc.UpdateProfile(r.Context(), p); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *SecurityProfileHandler) DeleteProfile(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil || claims.TenantID == "" {
		http.Error(w, `{"error":"unauthorized or missing tenant scope"}`, http.StatusUnauthorized)
		return
	}
	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		http.Error(w, `{"error":"invalid tenant id"}`, http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	if err := h.svc.DeleteProfile(r.Context(), id, tenantID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Mappings ---

func (h *SecurityProfileHandler) CreateMapping(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil || claims.TenantID == "" {
		http.Error(w, `{"error":"unauthorized or missing tenant scope"}`, http.StatusUnauthorized)
		return
	}
	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		http.Error(w, `{"error":"invalid tenant id"}`, http.StatusBadRequest)
		return
	}

	var req struct {
		IDPClientID    string `json:"idp_client_id"`
		IDPGroupID     string `json:"idp_group_id"`
		FunctionalRole string `json:"functional_role"`
		ClearanceLevel string `json:"clearance_level"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	m := &security.IdentityProfileMapping{
		TenantID:       tenantID,
		IDPClientID:    req.IDPClientID,
		IDPGroupID:     req.IDPGroupID,
		FunctionalRole: req.FunctionalRole,
		ClearanceLevel: req.ClearanceLevel,
	}

	res, err := h.svc.CreateMapping(r.Context(), m)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(res)
}

func (h *SecurityProfileHandler) ListMappings(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil || claims.TenantID == "" {
		http.Error(w, `{"error":"unauthorized or missing tenant scope"}`, http.StatusUnauthorized)
		return
	}
	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		http.Error(w, `{"error":"invalid tenant id"}`, http.StatusBadRequest)
		return
	}

	res, err := h.svc.ListMappings(r.Context(), tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (h *SecurityProfileHandler) GetMapping(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	res, err := h.svc.GetMapping(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (h *SecurityProfileHandler) UpdateMapping(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil || claims.TenantID == "" {
		http.Error(w, `{"error":"unauthorized or missing tenant scope"}`, http.StatusUnauthorized)
		return
	}
	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		http.Error(w, `{"error":"invalid tenant id"}`, http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	var req struct {
		IDPClientID    string `json:"idp_client_id"`
		IDPGroupID     string `json:"idp_group_id"`
		FunctionalRole string `json:"functional_role"`
		ClearanceLevel string `json:"clearance_level"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	m := &security.IdentityProfileMapping{
		MappingID:      id,
		TenantID:       tenantID,
		IDPClientID:    req.IDPClientID,
		IDPGroupID:     req.IDPGroupID,
		FunctionalRole: req.FunctionalRole,
		ClearanceLevel: req.ClearanceLevel,
	}

	if err := h.svc.UpdateMapping(r.Context(), m); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *SecurityProfileHandler) DeleteMapping(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil || claims.TenantID == "" {
		http.Error(w, `{"error":"unauthorized or missing tenant scope"}`, http.StatusUnauthorized)
		return
	}
	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		http.Error(w, `{"error":"invalid tenant id"}`, http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	if err := h.svc.DeleteMapping(r.Context(), id, tenantID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
