package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

type IDPMappingHandlers struct {
	DB  *sql.DB
	Svc *security.ProfileService
}

func NewIDPMappingHandlers(db *sql.DB, svc *security.ProfileService) *IDPMappingHandlers {
	return &IDPMappingHandlers{DB: db, Svc: svc}
}

func (h *IDPMappingHandlers) RegisterRoutes(r chi.Router) {
	r.Route("/admin/idp-mappings", func(r chi.Router) {
		r.Get("/", h.ListMappings)
		r.Post("/", h.CreateMapping)
		r.Delete("/{id}", h.DeleteMapping)
	})
}

type IDPMappingResponse struct {
	MappingID      string    `json:"mapping_id"`
	TenantID       string    `json:"tenant_id"`
	TenantName     string    `json:"tenant_name"`
	IDPClientID    string    `json:"idp_client_id"`
	IDPGroupID     string    `json:"idp_group_id"`
	FunctionalRole string    `json:"functional_role"`
	ClearanceLevel string    `json:"clearance_level"`
	CreatedAt      time.Time `json:"created_at"`
}

func (h *IDPMappingHandlers) ListMappings(w http.ResponseWriter, r *http.Request) {
	// Require admin or global admin scope (usually enforced by global middleware, but we can verify)
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	query := `
		SELECT m.mapping_id, m.tenant_id, COALESCE(t.display_name, t.name, 'Unknown Tenant') as tenant_name,
		       m.idp_client_id, m.idp_group_id, m.functional_role, m.clearance_level, m.created_at
		FROM security.identity_profile_mappings m
		LEFT JOIN public.tenants t ON m.tenant_id = t.id
		ORDER BY m.created_at DESC
	`

	rows, err := h.DB.QueryContext(r.Context(), query)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to query mappings: %v"}`, err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	mappings := []IDPMappingResponse{}
	for rows.Next() {
		var m IDPMappingResponse
		err := rows.Scan(&m.MappingID, &m.TenantID, &m.TenantName, &m.IDPClientID, &m.IDPGroupID, &m.FunctionalRole, &m.ClearanceLevel, &m.CreatedAt)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"failed to scan mapping: %v"}`, err), http.StatusInternalServerError)
			return
		}
		mappings = append(mappings, m)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mappings)
}

func (h *IDPMappingHandlers) CreateMapping(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req struct {
		TenantID       string `json:"tenant_id"`
		IDPClientID    string `json:"idp_client_id"`
		IDPGroupID     string `json:"idp_group_id"`
		FunctionalRole string `json:"functional_role"`
		ClearanceLevel string `json:"clearance_level"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON request body"}`, http.StatusBadRequest)
		return
	}

	// 1. Client-side/Handler validation: ensure values aren't empty
	if strings.TrimSpace(req.TenantID) == "" {
		http.Error(w, `{"error":"Tenant ID is required"}`, http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.IDPClientID) == "" {
		http.Error(w, `{"error":"IDP Client ID is required"}`, http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.IDPGroupID) == "" {
		http.Error(w, `{"error":"IDP Group ID is required"}`, http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.FunctionalRole) == "" {
		http.Error(w, `{"error":"Functional Role is required"}`, http.StatusBadRequest)
		return
	}

	// Default clearance level if empty
	clearance := req.ClearanceLevel
	if strings.TrimSpace(clearance) == "" {
		clearance = "L1"
	}

	tenantUUID, err := uuid.Parse(req.TenantID)
	if err != nil {
		http.Error(w, `{"error":"Invalid Tenant UUID format"}`, http.StatusBadRequest)
		return
	}

	// 2. Conflict Sanity Check: query if (idp_client_id, idp_group_id) is already mapped to any tenant
	var conflictingTenantID string
	var conflictingTenantName string
	conflictQuery := `
		SELECT m.tenant_id, COALESCE(t.display_name, t.name, 'Unknown Tenant') as tenant_name
		FROM security.identity_profile_mappings m
		LEFT JOIN public.tenants t ON m.tenant_id = t.id
		WHERE m.idp_client_id = $1 AND m.idp_group_id = $2
		LIMIT 1
	`
	err = h.DB.QueryRowContext(r.Context(), conflictQuery, req.IDPClientID, req.IDPGroupID).Scan(&conflictingTenantID, &conflictingTenantName)
	if err != nil && err != sql.ErrNoRows {
		http.Error(w, fmt.Sprintf(`{"error":"database error checking conflicts: %v"}`, err), http.StatusInternalServerError)
		return
	}

	if err == nil {
		// Found conflict
		http.Error(w, fmt.Sprintf(`{"error":"Conflict: IDP client '%s' and group '%s' is already mapped to tenant '%s' (%s)"}`, req.IDPClientID, req.IDPGroupID, conflictingTenantName, conflictingTenantID), http.StatusConflict)
		return
	}

	// 3. Insert new mapping
	mappingID := uuid.New()
	createdAt := time.Now()

	insertQuery := `
		INSERT INTO security.identity_profile_mappings (mapping_id, tenant_id, idp_client_id, idp_group_id, functional_role, clearance_level, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err = h.DB.ExecContext(r.Context(), insertQuery, mappingID, tenantUUID, req.IDPClientID, req.IDPGroupID, req.FunctionalRole, clearance, createdAt)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to create mapping: %v"}`, err), http.StatusInternalServerError)
		return
	}

	// Return created mapping
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"mapping_id":      mappingID.String(),
		"tenant_id":       tenantUUID.String(),
		"idp_client_id":   req.IDPClientID,
		"idp_group_id":    req.IDPGroupID,
		"functional_role": req.FunctionalRole,
		"clearance_level": clearance,
		"created_at":      createdAt,
	})
}

func (h *IDPMappingHandlers) DeleteMapping(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	mappingUUID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, `{"error":"Invalid mapping ID format"}`, http.StatusBadRequest)
		return
	}

	_, err = h.DB.ExecContext(r.Context(), "DELETE FROM security.identity_profile_mappings WHERE mapping_id = $1", mappingUUID)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to delete mapping: %v"}`, err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
