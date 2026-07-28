package upgrade

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/libs/jwt-middleware"
	"github.com/jmoiron/sqlx"
)

type UpgradeConflict struct {
	PropertyPath    string      `json:"property_path"`
	Reason          string      `json:"reason"`
	AncestorValue   interface{} `json:"ancestor_value"`
	ModifiedValue   interface{} `json:"modified_value"`
	TargetValue     interface{} `json:"target_value"`
}

type UpgradeResult struct {
	MergedSpec []byte            `json:"merged_spec"`
	Conflicts  []UpgradeConflict `json:"conflicts"`
	Success    bool              `json:"success"`
}

type TenantDelta struct {
	TenantID        string                 `json:"tenant_id"`
	BaseVersion     string                 `json:"base_version"`
	CoreMasterSpec  map[string]interface{} `json:"core_master_spec"`
	TenantOverlay   map[string]interface{} `json:"tenant_overlay"`
	CustomFields    []map[string]interface{}`json:"custom_fields"`
	ModifiedCount   int                    `json:"modified_count"`
}

type Service struct {
	db *sqlx.DB
}

func NewService(db *sqlx.DB) *Service {
	return &Service{db: db}
}

// MergeLayoutSpec executes a structural 3-way merge on JSON layout specs
func MergeLayoutSpec(ancestorJSON, modifiedJSON, targetJSON []byte) ([]byte, []UpgradeConflict, error) {
	var ancestor, modified, target map[string]interface{}

	if err := json.Unmarshal(ancestorJSON, &ancestor); err != nil {
		return nil, nil, fmt.Errorf("failed to parse ancestor: %w", err)
	}
	if err := json.Unmarshal(modifiedJSON, &modified); err != nil {
		return nil, nil, fmt.Errorf("failed to parse modified: %w", err)
	}
	if err := json.Unmarshal(targetJSON, &target); err != nil {
		return nil, nil, fmt.Errorf("failed to parse target: %w", err)
	}

	var conflicts []UpgradeConflict
	merged := make(map[string]interface{})

	// Copy new target baseline properties
	for k, v := range target {
		merged[k] = v
	}

	// Apply non-conflicting client modifications from 'modified'
	for k, modVal := range modified {
		ancVal, hasAncestor := ancestor[k]
		targetVal, hasTarget := target[k]

		// If client changed it from ancestor, and target didn't change it, keep client version
		if hasAncestor && jsonEquals(modVal, ancVal) {
			// Client didn't change this field, safe to use target upgrade
			continue
		}

		if !hasTarget || jsonEquals(targetVal, ancVal) {
			// Target didn't change this field, but client did -> preserve client customization
			merged[k] = modVal
			continue
		}

		// Both client and new target changed the same field -> Conflict!
		if !jsonEquals(modVal, targetVal) {
			conflicts = append(conflicts, UpgradeConflict{
				PropertyPath:  k,
				Reason:        fmt.Sprintf("Conflict detected on property '%s': keeping target upgrade version, review required.", k),
				AncestorValue: ancVal,
				ModifiedValue: modVal,
				TargetValue:   targetVal,
			})
		}
	}

	resultBytes, err := json.MarshalIndent(merged, "", "  ")
	return resultBytes, conflicts, err
}

func jsonEquals(a, b interface{}) bool {
	aBytes, _ := json.Marshal(a)
	bBytes, _ := json.Marshal(b)
	return string(aBytes) == string(bBytes)
}

// ExecuteTenantUpgrade runs the 3-way merge upgrade pipeline for a tenant
func (s *Service) ExecuteTenantUpgrade(ctx context.Context, tenantID, layoutKey, targetVersion string) (*UpgradeResult, error) {
	// Fetch tenant's active modified layout spec
	var currentSpec struct {
		LayoutSpec string `db:"layout_spec"`
		Version    int    `db:"version"`
	}
	queryCurrent := `SELECT layout_spec::text, version FROM public.page_layouts WHERE tenant_id = $1 AND key = $2 ORDER BY version DESC LIMIT 1`
	err := s.db.GetContext(ctx, &currentSpec, queryCurrent, tenantID, layoutKey)

	// If no custom spec found for tenant, fallback to gold_copy master spec
	if err != nil {
		currentSpec.LayoutSpec = `{"id":"default","title":"Base Master","domain":"PORTFOLIO","target_bo_id":"customers","layout":[]}`
	}

	// Gold copy master ancestor and target base specs
	ancestorSpec := []byte(`{"id":"default","title":"Base Master v1.0","domain":"PORTFOLIO","target_bo_id":"customers","layout":[]}`)
	targetBaseSpec := []byte(fmt.Sprintf(`{"id":"default","title":"Base Master %s","domain":"PORTFOLIO","target_bo_id":"customers","layout":[]}`, targetVersion))

	mergedJSON, conflicts, err := MergeLayoutSpec(ancestorSpec, []byte(currentSpec.LayoutSpec), targetBaseSpec)
	if err != nil {
		return nil, fmt.Errorf("3-way merge failed: %w", err)
	}

	res := &UpgradeResult{
		MergedSpec: mergedJSON,
		Conflicts:  conflicts,
		Success:    len(conflicts) == 0,
	}

	// If conflicts exist, store in upgrade_exceptions queue
	if len(conflicts) > 0 {
		for _, c := range conflicts {
			ancB, _ := json.Marshal(c.AncestorValue)
			modB, _ := json.Marshal(c.ModifiedValue)
			tarB, _ := json.Marshal(c.TargetValue)

			_, _ = s.db.ExecContext(ctx, `
				INSERT INTO public.upgrade_exceptions (
					id, tenant_id, layout_key, ancestor_version, target_version, property_path,
					conflict_reason, ancestor_value, modified_value, target_value, status, created_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
			`, uuid.New().String(), tenantID, layoutKey, "v1.0.0", targetVersion, c.PropertyPath, c.Reason, ancB, modB, tarB, "PENDING_REVIEW")
		}
	} else {
		// Auto-promote merged layout spec
		_, _ = s.db.ExecContext(ctx, `
			INSERT INTO public.page_layouts (
				id, tenant_id, key, title, domain, target_bo_id, is_default, layout_spec, version, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, false, $7, $8, NOW())
			ON CONFLICT (tenant_id, key, version) DO UPDATE SET layout_spec = EXCLUDED.layout_spec, updated_at = NOW()
		`, uuid.New().String(), tenantID, layoutKey, "Upgraded Layout", "PORTFOLIO", "customers", mergedJSON, currentSpec.Version+1)
	}

	return res, nil
}

// ComputeTenantDelta calculates the difference between gold_copy master tenant and client overlay tenant
func (s *Service) ComputeTenantDelta(ctx context.Context, tenantID string) (*TenantDelta, error) {
	// Query custom fields for tenant
	var customAttrs []map[string]interface{}
	rows, err := s.db.QueryxContext(ctx, `SELECT attribute_name, display_name, data_type, jsonb_path FROM public.tenant_custom_attributes WHERE tenant_id = $1`, tenantID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			item := make(map[string]interface{})
			_ = rows.MapScan(item)
			customAttrs = append(customAttrs, item)
		}
	}

	delta := &TenantDelta{
		TenantID:    tenantID,
		BaseVersion: "v1.2.0",
		CoreMasterSpec: map[string]interface{}{
			"gold_copy":    true,
			"target_bo_id": "customers",
			"domain":       "PORTFOLIO",
		},
		TenantOverlay: map[string]interface{}{
			"gold_copy":      false,
			"custom_fields":  customAttrs,
			"modified_count": len(customAttrs),
		},
		CustomFields:  customAttrs,
		ModifiedCount: len(customAttrs),
	}

	return delta, nil
}

// HTTP Handlers

func (s *Service) UpgradeTenantHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil && claims.TenantID != "" {
			tenantID = claims.TenantID
		} else {
			tenantID = "core"
		}
	}

	targetVersion := r.URL.Query().Get("target_version")
	if targetVersion == "" {
		targetVersion = "v1.3.0"
	}

	res, err := s.ExecuteTenantUpgrade(r.Context(), tenantID, "account_master", targetVersion)
	if err != nil {
		http.Error(w, fmt.Sprintf("Upgrade failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tenantId":      tenantID,
		"targetVersion": targetVersion,
		"status":        func() string { if res.Success { return "UPG_SUCCESS" }; return "UPGRADE_PENDING_REVIEW" }(),
		"conflicts":     res.Conflicts,
	})
}

func (s *Service) GetTenantDeltaHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil && claims.TenantID != "" {
			tenantID = claims.TenantID
		} else {
			tenantID = "core"
		}
	}

	delta, err := s.ComputeTenantDelta(r.Context(), tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to compute tenant delta: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(delta)
}
