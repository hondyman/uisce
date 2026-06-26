package analytics

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/cube"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/internal/viewgen"
	"github.com/hondyman/semlayer/backend/internal/viewmerge"
	"github.com/hondyman/semlayer/backend/internal/viewmodel"
	"github.com/hondyman/semlayer/backend/models"
	"github.com/jmoiron/sqlx"
)

const globalScopeSentinel = "00000000-0000-0000-0000-000000000000"

// ModelProvider is responsible for loading and providing the active semantic model catalog.
type ModelProvider struct {
	db *sqlx.DB
}

// NewModelProvider creates a new ModelProvider.
func NewModelProvider(db *sqlx.DB) *ModelProvider { return &ModelProvider{db: db} }

// GetActiveCatalog loads all current, published fabric definitions and compiles them
// into a single catalog for the query engine.
// GetActiveCatalog loads active fabric definitions filtered by tenant/datasource (when provided)
// and composes cubes + views. If tenantID/datasourceID are blank, it falls back to global published current.
func (p *ModelProvider) GetActiveCatalog(ctx context.Context, tenantID string, datasourceID string) (*cube.Catalog, error) {
	logging.GetLogger().Sugar().Info("Loading active catalog from database...")
	var defns []models.FabricDefn
	// Filter by tenant/datasource if provided. fabric_defn contains tenant_id and tenant_datasource_id columns.
	base := `SELECT id, tenant_id, tenant_datasource_id, model_key, resolved_config FROM public.fabric_defn WHERE is_current = true AND status = 'published'`
	var args []any
	globalTenantExpr := "(tenant_id = '" + globalScopeSentinel + "' OR tenant_id IS NULL)"
	globalDatasourceExpr := "(tenant_datasource_id = '" + globalScopeSentinel + "' OR tenant_datasource_id IS NULL)"
	if tenantID != "" && datasourceID != "" {
		base += ` AND (` + globalTenantExpr + ` OR tenant_id = $1)`
		base += ` AND (` + globalDatasourceExpr + ` OR tenant_datasource_id = $2)`
		args = append(args, tenantID, datasourceID)
	} else if tenantID != "" {
		base += ` AND (` + globalTenantExpr + ` OR tenant_id = $1)`
		args = append(args, tenantID)
	} else if datasourceID != "" {
		base += ` AND (` + globalDatasourceExpr + ` OR tenant_datasource_id = $1)`
		args = append(args, datasourceID)
	}
	base += ` ORDER BY (` + globalTenantExpr + `)::int DESC, (` + globalDatasourceExpr + `)::int DESC, model_key`
	if err := p.db.SelectContext(ctx, &defns, base, args...); err != nil {
		return nil, fmt.Errorf("failed to load active fabric definitions: %w", err)
	}

	catalog := &cube.Catalog{
		Cubes: make(map[string]cube.Cube),
		Views: make(map[string]cube.ViewMeta),
	}
	rawCubesByName := map[string][]cube.Cube{}

	// First load raw cubes and views
	for _, defn := range defns {
		var config models.ResolvedModelConfig
		if err := json.Unmarshal(defn.ResolvedConfig, &config); err != nil {
			logging.GetLogger().Sugar().Warnf("WARN: skipping invalid model config for model_key %s: %v", defn.ModelKey, err)
			continue
		}
		for _, mc := range config.Cubes {
			// Set the fabric definition ID for UUID-based extension resolution
			mc.FabricDefnID = &defn.ID
			if mc.Metadata == nil {
				mc.Metadata = map[string]any{}
			}
			// Persist the originating model key to assist inheritance resolution (e.g. references like "/employees")
			mc.Metadata["model_key"] = defn.ModelKey
			if tid := defn.TenantID.String(); tid != "" {
				mc.Metadata["_fabric_tenant_id"] = tid
			}
			if dsid := defn.TenantDatasourceID.String(); dsid != "" {
				mc.Metadata["_fabric_datasource_id"] = dsid
			}
			rawCubesByName[mc.Name] = append(rawCubesByName[mc.Name], mc)
		}
		if len(config.Views) > 0 {
			// The `config.Views` is of type `[]any`, which after JSON unmarshal becomes `[]map[string]any`.
			// We need to convert this to `[]cube.ViewMeta`. The simplest way is to re-marshal and unmarshal.
			viewsJSON, err := json.Marshal(config.Views)
			if err != nil {
				logging.GetLogger().Sugar().Warnf("WARN: could not re-marshal views for model_key %s: %v", defn.ModelKey, err)
				continue
			}

			var views []cube.ViewMeta
			if err := json.Unmarshal(viewsJSON, &views); err != nil {
				logging.GetLogger().Sugar().Warnf("WARN: could not unmarshal views for model_key %s: %v", defn.ModelKey, err)
			} else {
				for _, view := range views {
					catalog.Views[view.Name] = view
				}
			}
		}
	}
	// Compose core and extensions
	core := map[string]cube.Cube{}
	ext := map[string]cube.Cube{}
	for name, variants := range rawCubesByName {
		for _, c := range variants {
			isCore := hasTag(c, "core") || (c.Metadata != nil && c.Metadata["read_only"] == true)
			if !isCore && strings.HasSuffix(name, "_core") {
				isCore = true
			}
			if isCore {
				if existing, exists := core[name]; !exists || preferScopedCube(existing, c) {
					core[name] = c
				}
				continue
			}
			if c.Extends != nil {
				if existing, exists := ext[name]; !exists || preferScopedCube(existing, c) {
					ext[name] = c
				}
				continue
			}
			if existing, exists := core[name]; !exists || preferScopedCube(existing, c) {
				core[name] = c
			}
		}
	}

	merged, issues := cube.ComposeCatalog(core, ext)
	for _, is := range issues {
		lvl := strings.ToUpper(is.Level)
		if lvl == "ERROR" {
			logging.GetLogger().Sugar().Errorf("Model compose error: [%s] %s", is.Code, is.Message)
		} else {
			logging.GetLogger().Sugar().Warnf("Model compose warning: [%s] %s", is.Code, is.Message)
		}
	}
	catalog.Cubes = merged

	// Auto-generate views from merged cubes
	// Convert map to slice
	cubesList := make([]cube.Cube, 0, len(merged))
	for _, c := range merged {
		cubesList = append(cubesList, c)
	}
	vres := viewgen.GenerateViews(cubesList, viewgen.Options{
		MaxDepth:            2,
		DefaultPublic:       true,
		PiiMetaKey:          "pii",
		PreferPrefixedJoins: true,
		ExcludeFields:       []string{"tenant_id"},
		EnableAdminVariant:  true,
		AdminViewRoles:      []string{"steward", "admin"},
	})
	// Persist full JSON for debugging/export. Use SEMLAYER_RUNTIME_DIR if provided
	runtimeBase := os.Getenv("SEMLAYER_RUNTIME_DIR")
	if strings.TrimSpace(runtimeBase) == "" {
		if cwd, err := os.Getwd(); err == nil {
			runtimeBase = cwd
		} else {
			runtimeBase = "."
		}
	}
	viewsDir := filepath.Join(runtimeBase, "runtime", "views")
	if err := viewgen.WriteViews(viewsDir, vres); err != nil {
		logging.GetLogger().Sugar().Warnf("WARN: failed to write generated views to %s: %v", viewsDir, err)
	}
	// Validate views
	vIssues := viewgen.ValidateViews(cubesList, vres.Views)
	for _, is := range vIssues {
		lvl := strings.ToUpper(is.Level)
		if lvl == "ERROR" {
			logging.GetLogger().Sugar().Errorf("View validation error: [%s] %s", is.Code, is.Message)
		} else {
			logging.GetLogger().Sugar().Warnf("View validation warning: [%s] %s", is.Code, is.Message)
		}
	}

	// Second pass: read overrides and resolve extends/overrides. Prefer DB overrides when tenant is provided.
	userDir := filepath.Join(".", "views_overrides")
	genMap := map[string]viewmodel.View{}
	for _, v := range vres.Views {
		genMap[v.Name] = v
	}
	overMap := map[string]viewmodel.View{}
	// Load DB-backed overrides for tenant/datasource if provided, else fall back to filesystem
	if tenantID != "" {
		// Build query: tenant + optional datasource
		q := `SELECT name, view FROM public.view_overrides WHERE tenant_id = $1`
		var rows *sql.Rows
		var err error
		if datasourceID != "" {
			q += ` AND (tenant_datasource_id = $2 OR tenant_datasource_id IS NULL)`
			rows, err = p.db.QueryContext(ctx, q, tenantID, datasourceID)
		} else {
			rows, err = p.db.QueryContext(ctx, q, tenantID)
		}
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var name string
				var viewJSON []byte
				if err := rows.Scan(&name, &viewJSON); err == nil {
					var v viewmodel.View
					if jerr := json.Unmarshal(viewJSON, &v); jerr == nil {
						overMap[name] = v
					}
				}
			}
		} else {
			logging.GetLogger().Sugar().Warnf("WARN: failed to read DB overrides for tenant %s: %v", tenantID, err)
		}
	}
	if len(overMap) == 0 {
		// still allow local overrides when no tenant overrides available
		if fsover, err := viewgen.ReadViewsDir(userDir); err == nil {
			overMap = fsover
		} else {
			logging.GetLogger().Sugar().Warnf("WARN: failed to read user overrides from %s: %v", userDir, err)
		}
	}
	// Merge with change tracking via viewmerge
	var coreList []viewmodel.View
	for _, v := range genMap {
		coreList = append(coreList, v)
	}
	var extList []viewmodel.View
	for _, v := range overMap {
		extList = append(extList, v)
	}
	mres, _ := viewmerge.MergeViews(viewmerge.Stores{Core: coreList, Extensions: extList}, viewmerge.Options{CoreVersion: "runtime"})
	resolvedList := mres.Merged
	// Validate resolved views
	resIssues := viewgen.ValidateViews(cubesList, resolvedList)
	for _, is := range resIssues {
		lvl := strings.ToUpper(is.Level)
		if lvl == "ERROR" {
			logging.GetLogger().Sugar().Errorf("Resolved view validation error: [%s] %s", is.Code, is.Message)
		} else {
			logging.GetLogger().Sugar().Warnf("Resolved view validation warning: [%s] %s", is.Code, is.Message)
		}
	}
	// Persist resolved views
	resolvedDir := filepath.Join(runtimeBase, "runtime", "views_resolved")
	if err := os.MkdirAll(resolvedDir, 0o755); err != nil {
		logging.GetLogger().Sugar().Warnf("WARN: failed to mkdir %s: %v", resolvedDir, err)
	} else {
		if err := viewgen.WriteViews(resolvedDir, viewgen.Result{Views: resolvedList}); err != nil {
			logging.GetLogger().Sugar().Warnf("WARN: failed to write resolved views: %v", err)
		}
	}

	// Store into catalog.Views as light metadata entries (use resolved set to reflect overrides)
	for _, v := range resolvedList {
		catalog.Views[v.Name] = cube.ViewMeta{Name: v.Name, Description: v.Description}
	}

	logging.GetLogger().Sugar().Infof("Loaded %d cubes into active catalog (merged).", len(catalog.Cubes))
	logging.GetLogger().Sugar().Infof("Loaded %d views into active catalog.", len(catalog.Views))
	return catalog, nil
}

func hasTag(c cube.Cube, tag string) bool {
	for _, t := range c.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

func preferScopedCube(existing cube.Cube, candidate cube.Cube) bool {
	return cubeScopeRank(candidate) > cubeScopeRank(existing)
}

func cubeScopeRank(c cube.Cube) int {
	rank := 0
	if v := metadataString(c, "_fabric_tenant_id"); v != "" && v != globalScopeSentinel {
		rank += 2
	}
	if v := metadataString(c, "_fabric_datasource_id"); v != "" && v != globalScopeSentinel {
		rank++
	}
	return rank
}

func metadataString(c cube.Cube, key string) string {
	if c.Metadata == nil {
		return ""
	}
	if v, ok := c.Metadata[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
