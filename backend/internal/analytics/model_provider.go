package analytics

import (
	"context"

	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/hondyman/uisce/backend/models"
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
func (p *ModelProvider) GetActiveCatalog(ctx context.Context, tenantID string, datasourceID string) (*models.Catalog, error) {
	logging.GetLogger().Sugar().Info("Loading active catalog (stub)...")

	catalog := &models.Catalog{
		Cubes: make(map[string]models.Cube),
		Views: make(map[string]models.ViewMeta),
	}

	logging.GetLogger().Sugar().Infof("Loaded %d cubes into active catalog.", len(catalog.Cubes))
	logging.GetLogger().Sugar().Infof("Loaded %d views into active catalog.", len(catalog.Views))
	return catalog, nil
}

func hasTag(c models.Cube, tag string) bool {
	for _, t := range c.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

func preferScopedCube(existing models.Cube, candidate models.Cube) bool {
	return cubeScopeRank(candidate) > cubeScopeRank(existing)
}

func cubeScopeRank(c models.Cube) int {
	rank := 0
	if v := metadataString(c, "_fabric_tenant_id"); v != "" && v != globalScopeSentinel {
		rank += 2
	}
	if v := metadataString(c, "_fabric_datasource_id"); v != "" && v != globalScopeSentinel {
		rank++
	}
	return rank
}

func metadataString(c models.Cube, key string) string {
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
