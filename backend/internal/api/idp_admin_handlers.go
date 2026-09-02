package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/hondyman/uisce/backend/internal/security"
)

// ============================================================================
// IdP & Group-Role Mapping Admin API
//
// Manages semantic.tenant_identity_providers / tenant_identity_provider_grants
// (which issuers are trusted, and which tenants they may claim — enforced at
// auth time by services.ValidateIssuerTenant) and security.idp_group_role_mappings
// (which IdP group claim maps to which bp_roles.role_key, per tenant — resolved
// per request by security.GroupRoleResolver). Mounted on RBACHandlers since both
// are part of the same access-control admin surface.
// ============================================================================

func (h *RBACHandlers) RegisterIDPRoutes(r chi.Router) {
	r.Route("/rbac/idps", func(r chi.Router) {
		r.Get("/", h.listIDPs)
		r.Post("/", h.createIDP)
		r.Put("/{idpId}", h.updateIDP)
		r.Delete("/{idpId}", h.deleteIDP)

		r.Get("/{idpId}/grants", h.listIDPGrants)
		r.Post("/{idpId}/grants", h.grantIDPToTenant)
		r.Delete("/{idpId}/grants/{tenantId}", h.revokeIDPFromTenant)
	})

	r.Route("/rbac/group-role-mappings", func(r chi.Router) {
		r.Get("/", h.listGroupRoleMappings)
		r.Post("/", h.createGroupRoleMapping)
		r.Delete("/{mappingId}", h.deleteGroupRoleMapping)
	})
}

// ---- Identity Providers ----------------------------------------------------

type idpDTO struct {
	ID            string `json:"id" db:"id"`
	Issuer        string `json:"issuer" db:"issuer"`
	JWKSURI       string `json:"jwks_uri" db:"jwks_uri"`
	DisplayName   string `json:"display_name" db:"display_name"`
	IsCrossTenant bool   `json:"is_cross_tenant" db:"is_cross_tenant"`
	IsActive      bool   `json:"is_active" db:"is_active"`
	CreatedAt     string `json:"created_at" db:"created_at"`
	CreatedBy     string `json:"created_by" db:"created_by"`
}

func (h *RBACHandlers) listIDPs(w http.ResponseWriter, r *http.Request) {
	var idps []idpDTO
	if err := h.db.Select(&idps, `
		SELECT id, issuer, jwks_uri, display_name, is_cross_tenant, is_active, created_at, created_by
		FROM semantic.tenant_identity_providers ORDER BY created_at DESC
	`); err != nil {
		http.Error(w, fmt.Sprintf("Failed to list IdPs: %v", err), http.StatusInternalServerError)
		return
	}
	respondJSONRBAC(w, r, idps, http.StatusOK)
}

func (h *RBACHandlers) createIDP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Issuer        string `json:"issuer"`
		JWKSURI       string `json:"jwks_uri"`
		DisplayName   string `json:"display_name"`
		IsCrossTenant bool   `json:"is_cross_tenant"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Issuer == "" || req.JWKSURI == "" {
		http.Error(w, "issuer and jwks_uri are required", http.StatusBadRequest)
		return
	}

	createdBy := "admin"
	if auth, ok := security.AuthInfoFromContext(r.Context()); ok && auth.UserID != "" {
		createdBy = auth.UserID
	}

	var idp idpDTO
	err := h.db.Get(&idp, `
		INSERT INTO semantic.tenant_identity_providers (issuer, jwks_uri, display_name, is_cross_tenant, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, issuer, jwks_uri, display_name, is_cross_tenant, is_active, created_at, created_by
	`, req.Issuer, req.JWKSURI, req.DisplayName, req.IsCrossTenant, createdBy)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create IdP (issuer must be unique): %v", err), http.StatusConflict)
		return
	}
	respondJSONRBAC(w, r, idp, http.StatusCreated)
}

func (h *RBACHandlers) updateIDP(w http.ResponseWriter, r *http.Request) {
	idpID := chi.URLParam(r, "idpId")
	var req struct {
		JWKSURI       *string `json:"jwks_uri"`
		DisplayName   *string `json:"display_name"`
		IsCrossTenant *bool   `json:"is_cross_tenant"`
		IsActive      *bool   `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	_, err := h.db.Exec(`
		UPDATE semantic.tenant_identity_providers SET
			jwks_uri = COALESCE($2, jwks_uri),
			display_name = COALESCE($3, display_name),
			is_cross_tenant = COALESCE($4, is_cross_tenant),
			is_active = COALESCE($5, is_active)
		WHERE id = $1
	`, idpID, req.JWKSURI, req.DisplayName, req.IsCrossTenant, req.IsActive)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update IdP: %v", err), http.StatusInternalServerError)
		return
	}
	respondJSONRBAC(w, r, map[string]string{"status": "updated"}, http.StatusOK)
}

// deleteIDP deactivates rather than hard-deletes: a hard delete cascades to
// tenant_identity_provider_grants (FK ON DELETE CASCADE), instantly locking
// out every user of every tenant that issuer served — deactivation is the
// safer, reversible operation for an admin UI action.
func (h *RBACHandlers) deleteIDP(w http.ResponseWriter, r *http.Request) {
	idpID := chi.URLParam(r, "idpId")
	if _, err := h.db.Exec(`UPDATE semantic.tenant_identity_providers SET is_active = false WHERE id = $1`, idpID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to deactivate IdP: %v", err), http.StatusInternalServerError)
		return
	}
	respondJSONRBAC(w, r, map[string]string{"status": "deactivated"}, http.StatusOK)
}

// ---- Tenant Grants ----------------------------------------------------------

type idpGrantDTO struct {
	ID         string `json:"id" db:"id"`
	IdPID      string `json:"idp_id" db:"idp_id"`
	TenantID   string `json:"tenant_id" db:"tenant_id"`
	TenantName string `json:"tenant_name" db:"tenant_name"`
	GrantedAt  string `json:"granted_at" db:"granted_at"`
	GrantedBy  string `json:"granted_by" db:"granted_by"`
}

func (h *RBACHandlers) listIDPGrants(w http.ResponseWriter, r *http.Request) {
	idpID := chi.URLParam(r, "idpId")
	var grants []idpGrantDTO
	if err := h.db.Select(&grants, `
		SELECT g.id, g.idp_id, g.tenant_id, t.name AS tenant_name, g.granted_at, g.granted_by
		FROM semantic.tenant_identity_provider_grants g
		JOIN tenants t ON t.id = g.tenant_id
		WHERE g.idp_id = $1
		ORDER BY g.granted_at DESC
	`, idpID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to list grants: %v", err), http.StatusInternalServerError)
		return
	}
	respondJSONRBAC(w, r, grants, http.StatusOK)
}

func (h *RBACHandlers) grantIDPToTenant(w http.ResponseWriter, r *http.Request) {
	idpID := chi.URLParam(r, "idpId")
	var req struct {
		TenantID string `json:"tenant_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TenantID == "" {
		http.Error(w, "tenant_id is required", http.StatusBadRequest)
		return
	}

	grantedBy := "admin"
	if auth, ok := security.AuthInfoFromContext(r.Context()); ok && auth.UserID != "" {
		grantedBy = auth.UserID
	}

	_, err := h.db.Exec(`
		INSERT INTO semantic.tenant_identity_provider_grants (idp_id, tenant_id, granted_by)
		VALUES ($1, $2, $3)
		ON CONFLICT (idp_id, tenant_id) DO NOTHING
	`, idpID, req.TenantID, grantedBy)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to grant IdP to tenant: %v", err), http.StatusInternalServerError)
		return
	}
	respondJSONRBAC(w, r, map[string]string{"status": "granted"}, http.StatusCreated)
}

func (h *RBACHandlers) revokeIDPFromTenant(w http.ResponseWriter, r *http.Request) {
	idpID := chi.URLParam(r, "idpId")
	tenantID := chi.URLParam(r, "tenantId")
	if _, err := h.db.Exec(`
		DELETE FROM semantic.tenant_identity_provider_grants WHERE idp_id = $1 AND tenant_id = $2
	`, idpID, tenantID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to revoke grant: %v", err), http.StatusInternalServerError)
		return
	}
	respondJSONRBAC(w, r, map[string]string{"status": "revoked"}, http.StatusOK)
}

// ---- Group -> Role Mappings --------------------------------------------------

type groupRoleMappingDTO struct {
	ID            string         `json:"id" db:"id"`
	TenantID      string         `json:"tenant_id" db:"tenant_id"`
	TenantName    string         `json:"tenant_name" db:"tenant_name"`
	IdpGroupClaim string         `json:"idp_group_claim" db:"idp_group_claim"`
	RoleKey       string         `json:"role_key" db:"role_key"`
	RoleName      sql.NullString `json:"role_name" db:"role_name"`
	CreatedAt     string         `json:"created_at" db:"created_at"`
}

func (h *RBACHandlers) listGroupRoleMappings(w http.ResponseWriter, r *http.Request) {
	var mappings []groupRoleMappingDTO
	if err := h.db.Select(&mappings, `
		SELECT m.id, m.tenant_id, t.name AS tenant_name, m.idp_group_claim, m.role_key, r.role_name, m.created_at
		FROM security.idp_group_role_mappings m
		JOIN tenants t ON t.id = m.tenant_id
		LEFT JOIN bp_roles r ON r.role_key = m.role_key
		ORDER BY t.name, m.idp_group_claim
	`); err != nil {
		http.Error(w, fmt.Sprintf("Failed to list group role mappings: %v", err), http.StatusInternalServerError)
		return
	}
	respondJSONRBAC(w, r, mappings, http.StatusOK)
}

func (h *RBACHandlers) createGroupRoleMapping(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID      string `json:"tenant_id"`
		IdpGroupClaim string `json:"idp_group_claim"`
		RoleKey       string `json:"role_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.TenantID == "" || req.IdpGroupClaim == "" || req.RoleKey == "" {
		http.Error(w, "tenant_id, idp_group_claim, and role_key are required", http.StatusBadRequest)
		return
	}

	var roleExists bool
	if err := h.db.Get(&roleExists, `SELECT EXISTS(SELECT 1 FROM bp_roles WHERE role_key = $1 AND is_active = true)`, req.RoleKey); err != nil || !roleExists {
		http.Error(w, fmt.Sprintf("role_key %q does not match any active role", req.RoleKey), http.StatusBadRequest)
		return
	}

	var mapping groupRoleMappingDTO
	err := h.db.Get(&mapping, `
		INSERT INTO security.idp_group_role_mappings (tenant_id, idp_group_claim, role_key)
		VALUES ($1, $2, $3)
		ON CONFLICT (tenant_id, idp_group_claim, role_key) DO UPDATE SET idp_group_claim = EXCLUDED.idp_group_claim
		RETURNING id, tenant_id, (SELECT name FROM tenants WHERE id = tenant_id) AS tenant_name, idp_group_claim, role_key,
			(SELECT role_name FROM bp_roles WHERE role_key = $3) AS role_name, created_at
	`, req.TenantID, req.IdpGroupClaim, req.RoleKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create group role mapping: %v", err), http.StatusInternalServerError)
		return
	}
	respondJSONRBAC(w, r, mapping, http.StatusCreated)
}

func (h *RBACHandlers) deleteGroupRoleMapping(w http.ResponseWriter, r *http.Request) {
	mappingID := chi.URLParam(r, "mappingId")
	if _, err := h.db.Exec(`DELETE FROM security.idp_group_role_mappings WHERE id = $1`, mappingID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete mapping: %v", err), http.StatusInternalServerError)
		return
	}
	respondJSONRBAC(w, r, map[string]string{"status": "deleted"}, http.StatusOK)
}
