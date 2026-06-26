package lineage

import (
	"encoding/json"
	"time"
)

// LineageNodeType defines the type of a lineage node
type LineageNodeType string

const (
	NodeBO          LineageNodeType = "bo"
	NodeBOField     LineageNodeType = "bo_field"
	NodePreAgg      LineageNodeType = "preagg"
	NodeTable       LineageNodeType = "table"
	NodeColumn      LineageNodeType = "column"
	NodeEntitlement LineageNodeType = "entitlement"
	NodeASOOpt      LineageNodeType = "aso_opt"
	NodeTenant      LineageNodeType = "tenant"
	NodeChangeSet   LineageNodeType = "changeset"
	NodePage        LineageNodeType = "page"
	NodeAPIEndpoint LineageNodeType = "api_endpoint"
)

// LineageEdgeType defines the relationship type between nodes
type LineageEdgeType string

const (
	EdgeDependsOn   LineageEdgeType = "depends_on"
	EdgeDerivedFrom LineageEdgeType = "derived_from"
	EdgeGovernedBy  LineageEdgeType = "governed_by"
	EdgeOptimizedBy LineageEdgeType = "optimized_by"
	EdgeBelongsTo   LineageEdgeType = "belongs_to"
	EdgeOverrides   LineageEdgeType = "overrides"
	EdgeIncludedIn  LineageEdgeType = "included_in"
)

// LineageNode represents a node in the lineage graph
type LineageNode struct {
	ID        string          `json:"id" db:"id"`
	Type      LineageNodeType `json:"type" db:"type"`
	Env       string          `json:"env" db:"env"`
	TenantID  *string         `json:"tenant_id" db:"tenant_id"`
	Name      string          `json:"name" db:"name"`
	Metadata  json.RawMessage `json:"metadata" db:"metadata"`
	CreatedAt time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt time.Time       `json:"updated_at" db:"updated_at"`
}

// LineageEdge represents a directed edge in the lineage graph
type LineageEdge struct {
	FromID    string          `json:"from_id" db:"from_id"`
	ToID      string          `json:"to_id" db:"to_id"`
	Type      LineageEdgeType `json:"type" db:"type"`
	Env       string          `json:"env" db:"env"`
	TenantID  *string         `json:"tenant_id" db:"tenant_id"`
	Metadata  json.RawMessage `json:"metadata" db:"metadata"`
	CreatedAt time.Time       `json:"created_at" db:"created_at"`
}

// ImpactReport summarizes the impact of a change
type ImpactReport struct {
	NodeID                   string        `json:"node_id"`
	AffectedBOs              []LineageNode `json:"affected_bos"`
	AffectedPreAggs          []LineageNode `json:"affected_preaggs"`
	AffectedEntitlements     []LineageNode `json:"affected_entitlements"`
	AffectedASOOptimizations []LineageNode `json:"affected_aso_optimizations"`
	AffectedPages            []LineageNode `json:"affected_pages"`
	AffectedAPIEndpoints     []LineageNode `json:"affected_api_endpoints"`
	AffectedTenants          []string      `json:"affected_tenants"`
}

// Graph represents a collection of lineage nodes and edges
type Graph struct {
	Nodes []LineageNode `json:"nodes"`
	Edges []LineageEdge `json:"edges"`
}

// mustMarshal marshals data to JSON or panics on error
func mustMarshal(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
