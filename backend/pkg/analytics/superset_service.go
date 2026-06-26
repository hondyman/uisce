package analytics

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
)

type SupersetService struct {
	db             *sqlx.DB
	apiBaseURL     string
	coreAssetsPath string
}

type AnalyticsAsset struct {
	ID            string `db:"id"`
	TenantID      string `db:"tenant_id"`
	CoreAssetID   string `db:"core_asset_id"`
	ActualAssetID string `db:"actual_asset_id"`
	Version       string `db:"version"`
	IsCustomized  bool   `db:"is_customized"`
}

func NewSupersetService(db *sqlx.DB) *SupersetService {
	return &SupersetService{
		db:             db,
		apiBaseURL:     "http://superset:8088/api/v1", // Mock URL
		coreAssetsPath: "superset/core_assets",
	}
}

// ProvisionTenant sets up the initial analytics workspace for a tenant
func (s *SupersetService) ProvisionTenant(ctx context.Context, tenantID string) error {
	log.Printf("[Superset] Provisioning workspace for Tenant: %s", tenantID)

	// 1. Create Tenant Role (Mock)
	// POST /api/v1/security/roles
	log.Printf("[Superset] Created Role: tenant_%s_users", tenantID)

	// 2. Apply RLS (Mock)
	// POST /api/v1/security/rls
	log.Printf("[Superset] Applied RLS: tenant_id = '%s'", tenantID)

	// 3. Deploy Core Assets (v1 default)
	// Import from JSON
	assetFile := "trade_dashboard_v1.json"

	// Mock: We pretend Superset returned dashboard ID "1001"
	actualID := fmt.Sprintf("dash_%s_1001", tenantID)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO analytics_assets 
		(tenant_id, core_asset_id, actual_asset_id, version, is_customized)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (tenant_id, core_asset_id) DO NOTHING
	`, tenantID, "core-trade", actualID, "v1", false)

	if err != nil {
		return fmt.Errorf("failed to record asset: %w", err)
	}

	log.Printf("[Superset] Deployed Asset: %s (v1) -> %s", assetFile, actualID)
	return nil
}

// CustomizeDashboard enables "Copy-and-Own" for a tenant
func (s *SupersetService) CustomizeDashboard(ctx context.Context, tenantID string, coreAssetID string) error {
	var asset AnalyticsAsset
	err := s.db.GetContext(ctx, &asset, "SELECT * FROM analytics_assets WHERE tenant_id=$1 AND core_asset_id=$2", tenantID, coreAssetID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("asset not found for tenant")
		}
		return err
	}

	log.Printf("[Superset] Customizing Dashboard %s for Tenant %s", asset.ActualAssetID, tenantID)

	// 1. Duplicate Asset in Superset (Mock)
	// POST /api/v1/dashboard/{id}/copy
	newActualID := asset.ActualAssetID + "_custom"

	// 2. Mark as Customized
	_, err = s.db.ExecContext(ctx, `
		UPDATE analytics_assets 
		SET is_customized = true, actual_asset_id = $1
		WHERE id = $2
	`, newActualID, asset.ID)

	if err != nil {
		return fmt.Errorf("failed to update customization status: %w", err)
	}

	log.Printf("[Superset] Dashboard duplicated. Owner changed to Tenant. Link to Core broken.")
	return nil
}

// UpgradeResult describes the outcome of an upgrade check
type UpgradeResult struct {
	TenantID string
	Result   string // "UPGRADED", "NOTIFIED", "UP_TO_DATE"
}

// CheckForUpgrades handles the upgrade workflow
// This would typically be run by a batch job
func (s *SupersetService) CheckForUpgrades(ctx context.Context, coreAssetID string, latestVersion string) ([]UpgradeResult, error) {
	var assets []AnalyticsAsset
	err := s.db.SelectContext(ctx, &assets, "SELECT * FROM analytics_assets WHERE core_asset_id=$1", coreAssetID)
	if err != nil {
		return nil, err
	}

	var results []UpgradeResult

	for _, asset := range assets {
		if asset.Version == latestVersion {
			results = append(results, UpgradeResult{TenantID: asset.TenantID, Result: "UP_TO_DATE"})
			continue
		}

		if asset.IsCustomized {
			// NOTIFICATION PATH
			// Setup notification logic here...
			log.Printf("[Notification] Tenant %s: 'Core Trade Dashboard' has a new version (%s). You have a custom version. Click to Preview.", asset.TenantID, latestVersion)
			results = append(results, UpgradeResult{TenantID: asset.TenantID, Result: "NOTIFIED"})
		} else {
			// AUTO-UPGRADE PATH
			// Mock: POST /api/v1/dashboard/import (overwrite)
			log.Printf("[Superset] Auto-upgrading Tenant %s to %s...", asset.TenantID, latestVersion)

			_, err := s.db.ExecContext(ctx, "UPDATE analytics_assets SET version=$1 WHERE id=$2", latestVersion, asset.ID)
			if err != nil {
				log.Printf("Failed to update version: %v", err)
				continue
			}
			results = append(results, UpgradeResult{TenantID: asset.TenantID, Result: "UPGRADED"})
		}
	}

	return results, nil
}
