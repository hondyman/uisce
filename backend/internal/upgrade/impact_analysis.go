package upgrade

import (
	"context"
	"fmt"
	"net/http"

	"encoding/json"

	"github.com/hondyman/uisce/libs/jwt-middleware"
	"github.com/jmoiron/sqlx"
)

type TenantImpactWarning struct {
	TenantID     string `json:"tenant_id"`
	BOID         string `json:"bo_id"`
	FieldName    string `json:"field_name"`
	Severity     string `json:"severity"` // CRITICAL, WARNING, INFO
	Message      string `json:"message"`
	ConflictPath string `json:"conflict_path"`
}

type StorageImpactReport struct {
	CitusDistributedDDLs []string `json:"citus_distributed_ddls"`
	IcebergEvolutions     []string `json:"iceberg_evolutions"`
	StarRocksRebuilds     []string `json:"starrocks_rebuilds"`
	CacheInvalidations    []string `json:"cache_invalidations"`
}

type ImpactSimulationReport struct {
	PackageID          string                `json:"package_id"`
	Version            string                `json:"version"`
	CanUpgrade         bool                  `json:"can_upgrade"`
	CoreDeltasCount    int                   `json:"core_deltas_count"`
	AffectedTenants    int                   `json:"affected_tenants"`
	TenantWarnings     []TenantImpactWarning `json:"tenant_warnings"`
	StorageImpact      StorageImpactReport   `json:"storage_impact"`
	PreFlightCheckTime string                `json:"preflight_check_time"`
}

type ImpactEngine struct {
	db *sqlx.DB
}

func NewImpactEngine(db *sqlx.DB) *ImpactEngine {
	return &ImpactEngine{db: db}
}

// RunPreFlightSimulation calculates the pre-flight impact matrix across all 3 dimensions
func (e *ImpactEngine) RunPreFlightSimulation(ctx context.Context, pkg UpgradePackageSpec) (*ImpactSimulationReport, error) {
	report := &ImpactSimulationReport{
		PackageID:       pkg.PackageID,
		Version:         pkg.Version,
		CanUpgrade:      true,
		CoreDeltasCount: len(pkg.CoreDeltas),
	}

	// 1. Analyze Tenant Customization Deltas against proposed core deltas
	var tenantWarnings []TenantImpactWarning
	tenantMap := make(map[string]bool)

	for _, delta := range pkg.CoreDeltas {
		if delta.ChangeType == "TYPE_CHANGED" && e.db != nil {
			// Query tenants that have custom attributes or formulas referencing this field
			query := `SELECT tenant_id, attribute_name FROM public.tenant_custom_attributes WHERE bo_id = $1 AND attribute_name = $2`
			rows, err := e.db.QueryContext(ctx, query, delta.TargetBOID, delta.FieldName)
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var tID, attrName string
					_ = rows.Scan(&tID, &attrName)
					tenantMap[tID] = true
					tenantWarnings = append(tenantWarnings, TenantImpactWarning{
						TenantID:     tID,
						BOID:         delta.TargetBOID,
						FieldName:    attrName,
						Severity:     "CRITICAL",
						Message:      fmt.Sprintf("Tenant %s has custom attribute '%s' bound to field %s.%s which changed type from %s to %s", tID, attrName, delta.TargetBOID, delta.FieldName, delta.OldType, delta.NewType),
						ConflictPath: fmt.Sprintf("%s.%s", delta.TargetBOID, attrName),
					})
					report.CanUpgrade = false
				}
			}
		}
	}

	report.AffectedTenants = len(tenantMap)
	report.TenantWarnings = tenantWarnings

	// 2. Analyze Heterogeneous Storage Impact
	var citusDDLs, icebergEvos, starrocksViews, cacheInvals []string

	for _, script := range pkg.SchemaScripts {
		switch script.StoreType {
		case "CITUS_POSTGRES":
			citusDDLs = append(citusDDLs, fmt.Sprintf("[Citus Master Node] %s", script.DDLStatement))
		case "ICEBERG":
			icebergEvos = append(icebergEvos, fmt.Sprintf("[Iceberg Catalog] %s", script.DDLStatement))
		case "STARROCKS":
			starrocksViews = append(starrocksViews, fmt.Sprintf("[StarRocks OLAP] %s", script.DDLStatement))
		}
	}

	for _, rule := range pkg.PreAggInvalidate {
		if rule.TargetStore == "STARROCKS" {
			starrocksViews = append(starrocksViews, fmt.Sprintf("[StarRocks MV Invalidation] Rebuilding %s", rule.ViewName))
		} else if rule.TargetStore == "REDIS" {
			cacheInvals = append(cacheInvals, fmt.Sprintf("[Redis AST Cache] Flushed namespace %s", rule.ViewName))
		}
	}

	report.StorageImpact = StorageImpactReport{
		CitusDistributedDDLs: citusDDLs,
		IcebergEvolutions:     icebergEvos,
		StarRocksRebuilds:     starrocksViews,
		CacheInvalidations:    cacheInvals,
	}

	return report, nil
}

// HTTP Handler

func (e *ImpactEngine) PreFlightSimulationHandler(w http.ResponseWriter, r *http.Request) {
	var pkg UpgradePackageSpec
	if err := json.NewDecoder(r.Body).Decode(&pkg); err != nil || pkg.Version == "" {
		pkg = CreateSampleManifest("v1.3.0")
	}

	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil && claims.TenantID != "" {
		// Use claims tenant context if needed
	}

	report, err := e.RunPreFlightSimulation(r.Context(), pkg)
	if err != nil {
		http.Error(w, fmt.Sprintf("Pre-flight simulation failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}
