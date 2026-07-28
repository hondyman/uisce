package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/iceberg"
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
}

type OnboardTenantResponse struct {
	TenantID          string `json:"tenant_id"`
	TenantKey        string `json:"tenant_key"`
	PolarisCatalogURL string `json:"polaris_catalog_url"`
	Status            string `json:"status"`
}

func (h *OnboardingHandler) OnboardTenant(w http.ResponseWriter, r *http.Request) {
	var req OnboardTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
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
