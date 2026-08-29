package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/iceberg"
	"github.com/hondyman/uisce/backend/internal/security"
)

type OnboardingHandler struct {
	platformDB     *sql.DB
	polaris        *iceberg.PolarisProvisioner
}

func NewOnboardingHandler(platformDB *sql.DB, polaris *iceberg.PolarisProvisioner) *OnboardingHandler {
	return &OnboardingHandler{
		platformDB: platformDB,
		polaris:   polaris,
	}
}

func (h *OnboardingHandler) RegisterRoutes(r chi.Router) {
	r.Post("/tenants", h.OnboardTenant)
}

type OnboardTenantRequest struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	// Domain, if set, is bound to security.tenant_domains so the tenant's
	// users are auto-provisioned on first login (see
	// internal/middleware/tenant_provisioning.go). Optional: a tenant can be
	// created before its client domain is known and bound later.
	Domain string `json:"domain,omitempty"`
}

type OnboardTenantResponse struct {
	TenantID          string `json:"tenant_id"`
	TenantKey        string `json:"tenant_key"`
	PolarisCatalogURL string `json:"polaris_catalog_url"`
	Status            string `json:"status"`
}

func (h *OnboardingHandler) OnboardTenant(w http.ResponseWriter, r *http.Request) {
	// Provisioning a tenant creates real infrastructure (an Iceberg catalog)
	// and grants it access to gold-copy Business Objects/security profiles —
	// this had no auth check at all. Restricted to global admins.
	auth, ok := security.RequireAuth(w, r)
	if !ok {
		return
	}
	if !auth.IsGlobalAdmin {
		http.Error(w, "Forbidden: global admin role required to onboard a tenant", http.StatusForbidden)
		return
	}

	var req OnboardTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	domain := strings.ToLower(strings.TrimSpace(req.Domain))
	if h.platformDB == nil {
		http.Error(w, "platform DB not configured", http.StatusInternalServerError)
		return
	}
	if h.polaris == nil {
		http.Error(w, "Polaris provisioner not configured", http.StatusInternalServerError)
		return
	}

	tenantKey := req.Name
	tenantID := uuid.New().String()
	displayName := req.DisplayName
	if displayName == "" {
		displayName = tenantKey
	}

	tx, err := h.platformDB.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "failed to begin transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(r.Context(), `
		INSERT INTO tenants (id, code, name, is_active)
		VALUES ($1, $2, $3, true)
		ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, is_active = true
	`, tenantID, tenantKey, displayName)
	if err != nil {
		http.Error(w, "failed to insert tenant: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = h.polaris.Provision(r.Context(), tenantKey)
	if err != nil {
		http.Error(w, "Polaris provisioning failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if domain != "" {
		_, err = tx.ExecContext(r.Context(), `
			INSERT INTO security.tenant_domains (domain, tenant_id, is_active)
			VALUES ($1, $2, true)
			ON CONFLICT (domain) DO UPDATE SET tenant_id = EXCLUDED.tenant_id, is_active = true, updated_at = now()
		`, domain, tenantID)
		if err != nil {
			http.Error(w, "failed to bind domain: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// NOTE: gold-copy Business Objects, security profiles, semantic/business
	// terms, AND tenant_product / tenant_product_datasource all use the same
	// mechanism — they are NOT copied here, and should not be. Confirmed
	// against the live DB: tenant_product and tenant_product_datasource each
	// carry a self-referencing core_id column, mirroring business_objects.core_id
	// / bo_fields.core_id. A new tenant inherits the gold-copy tenant's
	// products/datasources by having zero override rows, the same as it
	// inherits BOs by having zero tenant_id-scoped rows — not by a copy step.
	// (This repo's two on-disk migration files for these tables,
	// 000028_datasource_enhancements.sql and 20260812_add_datasource_tables.sql,
	// are both stale relative to the live schema, which is a superset of both
	// plus core_id/tenant_instance_id columns added by later, unfiled
	// migrations — don't trust either file as the schema source of truth.)
	// The one thing a new tenant DOES need per this owner's own onboarding
	// flow is its own connection/config rows, which are inherently
	// tenant-specific and were never meant to be inherited — that's the
	// "of course I will configure the databases to the specific tenant"
	// step, done manually or by internal/tenantauto/provisioner.go after
	// this handler returns, not by OnboardTenant itself.

	if err := tx.Commit(); err != nil {
		http.Error(w, "failed to commit transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := OnboardTenantResponse{
		TenantID:          tenantID,
		TenantKey:        tenantKey,
		PolarisCatalogURL: fmt.Sprintf("http://uisce-polaris:8185/api/catalog/v1/%s", tenantKey),
		Status:            "provisioned",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
