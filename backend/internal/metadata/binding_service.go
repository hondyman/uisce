package metadata

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/libs/jwt-middleware"
	"github.com/jmoiron/sqlx"
)

// BusinessObjectBinding mirrors the live public.business_object_bindings
// table (verified directly against the running database — see
// migration 20260901000005_business_object_binding_datasource_slot.sql and
// db/migrations/20261002_business_object_studio_engine.up.sql, the two
// migrations whose statements actually landed on this column set; several
// other migration files in this repo declare a different, never-applied
// shape for this table and must not be used as a reference). There is no
// free-text binding name column — display names are derived from
// alpha_product/alpha_datasource via the slot FK.
type BusinessObjectBinding struct {
	ID                string         `json:"id" db:"id"`
	TenantID          string         `json:"tenant_id" db:"tenant_id"`
	BOID              string         `json:"bo_id" db:"bo_id"`
	BackendID         *string        `json:"backend_id,omitempty" db:"backend_id"`
	BackendType       *string        `json:"backend_type,omitempty" db:"backend_type"`
	DrivingNodeID     *string        `json:"driving_node_id,omitempty" db:"driving_node_id"`
	IsDefault         bool           `json:"is_default" db:"is_default"`
	TemporalOverride  string         `json:"temporal_override" db:"temporal_override"`
	BaseSQL           *string        `json:"base_sql,omitempty" db:"base_sql"`
	AlphaProductID    sql.NullString `json:"alpha_product_id,omitempty" db:"alpha_product_id"`
	AlphaDatasourceID sql.NullString `json:"alpha_datasource_id,omitempty" db:"alpha_datasource_id"`
}

type BindingService struct {
	db *sqlx.DB
}

func NewBindingService(db *sqlx.DB) *BindingService {
	return &BindingService{db: db}
}

func (s *BindingService) SaveBinding(ctx context.Context, binding BusinessObjectBinding) error {
	if binding.ID == "" {
		binding.ID = uuid.New().String()
	}
	if binding.TemporalOverride == "" {
		binding.TemporalOverride = "NONE"
	}

	query := `
		INSERT INTO public.business_object_bindings (
			id, tenant_id, bo_id, backend_id, backend_type, driving_node_id,
			is_default, temporal_override, base_sql, alpha_product_id, alpha_datasource_id, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
		ON CONFLICT (id) DO UPDATE SET
			backend_id = EXCLUDED.backend_id,
			backend_type = EXCLUDED.backend_type,
			driving_node_id = EXCLUDED.driving_node_id,
			is_default = EXCLUDED.is_default,
			temporal_override = EXCLUDED.temporal_override,
			base_sql = EXCLUDED.base_sql,
			alpha_product_id = EXCLUDED.alpha_product_id,
			alpha_datasource_id = EXCLUDED.alpha_datasource_id,
			updated_at = NOW()
	`
	_, err := s.db.ExecContext(ctx, query,
		binding.ID, binding.TenantID, binding.BOID, binding.BackendID, binding.BackendType,
		binding.DrivingNodeID, binding.IsDefault, binding.TemporalOverride, binding.BaseSQL,
		binding.AlphaProductID, binding.AlphaDatasourceID,
	)
	return err
}

func (s *BindingService) GetBindingsForBO(ctx context.Context, tenantID, boID string) ([]BusinessObjectBinding, error) {
	query := `
		SELECT id, tenant_id, bo_id, backend_id, backend_type, driving_node_id,
		       is_default, temporal_override, base_sql, alpha_product_id, alpha_datasource_id
		FROM public.business_object_bindings
		WHERE tenant_id = $1 AND bo_id = $2
	`
	var bindings []BusinessObjectBinding
	err := s.db.SelectContext(ctx, &bindings, query, tenantID, boID)
	return bindings, err
}

// BindingResolvedView is the read-facing shape for GetBindingsHandler.
// AlphaProductID/AlphaDatasourceID are the binding's declared logical
// datasource slot — a datasource *type* shared across tenants, not a
// connection. Resolving a slot to this tenant's actual
// tenant_product_datasource.id (what the tenant-scope UI calls "the scoped
// datasource") is deliberately NOT done here: that resolution already
// exists, tested, at GET
// /business-objects/{id}/resolve-datasource?binding_id=... (which this
// type's AlphaProductID/AlphaDatasourceID are exactly what that endpoint
// needs) — duplicating its join logic here would just be a second place for
// the same resolution to drift out of sync.
//
// DisplayName is derived (alpha_product.product_name + " / " +
// alpha_datasource.name) since the live table has no free-text binding name
// column.
type BindingResolvedView struct {
	ID                string         `json:"id" db:"id"`
	BOID              string         `json:"bo_id" db:"bo_id"`
	BackendType       sql.NullString `json:"backend_type" db:"backend_type"`
	DrivingNodeID     sql.NullString `json:"driving_node_id" db:"driving_node_id"`
	AlphaProductID    sql.NullString `json:"alpha_product_id" db:"alpha_product_id"`
	AlphaDatasourceID sql.NullString `json:"alpha_datasource_id" db:"alpha_datasource_id"`
	DisplayName       sql.NullString `json:"display_name" db:"display_name"`
	IsDefault         bool           `json:"is_default" db:"is_default"`
}

// GetBindingsForBOResolved is GetBindingsForBO plus the display name
// BindingResolvedView needs — see its docstring for the tenant-datasource
// resolution step this deliberately leaves to resolve-datasource.
func (s *BindingService) GetBindingsForBOResolved(ctx context.Context, tenantID, boID string) ([]BindingResolvedView, error) {
	query := `
		SELECT
			bob.id,
			bob.bo_id,
			bob.backend_type,
			bob.driving_node_id::text AS driving_node_id,
			bob.alpha_product_id::text AS alpha_product_id,
			bob.alpha_datasource_id::text AS alpha_datasource_id,
			NULLIF(TRIM(BOTH ' / ' FROM COALESCE(ap.product_name, '') || ' / ' || COALESCE(ad.name, '')), '') AS display_name,
			bob.is_default
		FROM public.business_object_bindings bob
		LEFT JOIN public.alpha_product ap ON ap.id = bob.alpha_product_id
		LEFT JOIN public.alpha_datasource ad ON ad.id = bob.alpha_datasource_id
		WHERE bob.tenant_id = $1 AND bob.bo_id = $2
		ORDER BY bob.is_default DESC
	`
	var bindings []BindingResolvedView
	err := s.db.SelectContext(ctx, &bindings, query, tenantID, boID)
	return bindings, err
}

// HTTP Handlers

func (s *BindingService) SaveBindingHandler(w http.ResponseWriter, r *http.Request) {
	var binding BusinessObjectBinding
	if err := json.NewDecoder(r.Body).Decode(&binding); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil && claims.TenantID != "" {
		binding.TenantID = claims.TenantID
	}
	if binding.TenantID == "" {
		binding.TenantID = "core"
	}

	if err := s.SaveBinding(r.Context(), binding); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save binding: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "binding": binding})
}

func (s *BindingService) GetBindingsHandler(w http.ResponseWriter, r *http.Request) {
	boID := r.URL.Query().Get("bo_id")

	claims := jwtmiddleware.GetClaimsFromContext(r)
	tenantID := "core"
	if claims != nil && claims.TenantID != "" {
		tenantID = claims.TenantID
	}

	bindings, err := s.GetBindingsForBOResolved(r.Context(), tenantID, boID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch bindings: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"tenantId": tenantID, "boId": boID, "bindings": bindings})
}
