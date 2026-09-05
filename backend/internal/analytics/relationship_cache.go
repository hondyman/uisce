package analytics

import (
	"sync"

	"github.com/google/uuid"
)

// relationshipCacheKey scopes a cached BO relationship lookup to a tenant
// datasource and a single BO node, mirroring the (datasourceID, boNodeID)
// pair GetBORelationships is keyed on.
type relationshipCacheKey struct {
	DatasourceID uuid.UUID
	BONodeID     uuid.UUID
}

// RelationshipCache is an in-memory cache of the BO_RELATES_TO_BO graph.
// GetBORelationships previously hit catalog_edge on every call; Report
// Builder, Query Builder, and Page Studio all need to resolve "which BOs
// relate to this one" on the hot path of building a query, so repeated
// lookups for the same BO are served from memory instead. Any write that
// changes the graph (a new/updated edge) must call Invalidate.
type RelationshipCache struct {
	mu     sync.RWMutex
	boRels map[relationshipCacheKey][]BORelationshipEdge
}

func NewRelationshipCache() *RelationshipCache {
	return &RelationshipCache{boRels: make(map[relationshipCacheKey][]BORelationshipEdge)}
}

func (c *RelationshipCache) Get(datasourceID, boNodeID uuid.UUID) ([]BORelationshipEdge, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.boRels[relationshipCacheKey{DatasourceID: datasourceID, BONodeID: boNodeID}]
	return v, ok
}

func (c *RelationshipCache) Set(datasourceID, boNodeID uuid.UUID, edges []BORelationshipEdge) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.boRels[relationshipCacheKey{DatasourceID: datasourceID, BONodeID: boNodeID}] = edges
}

// InvalidateDatasource drops every cached entry for a datasource. Edges are
// undirected in practice (a BO shows up under both its own key and its
// partner's), so a single new/changed edge can affect any BO's cached list;
// clearing per-datasource is the cheap, correct invalidation.
func (c *RelationshipCache) InvalidateDatasource(datasourceID uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.boRels {
		if k.DatasourceID == datasourceID {
			delete(c.boRels, k)
		}
	}
}
