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

// ConsumerDomain categorizes downstream artifacts by their consumption layer
type ConsumerDomain string

const (
	DomainReactDashboards        ConsumerDomain = "REACT_DASHBOARDS"
	DomainRegulatoryExporters    ConsumerDomain = "REGULATORY_EXPORTERS"
	DomainDownstreamTenantQueries ConsumerDomain = "DOWNSTREAM_TENANT_QUERIES"
	DomainInternalAPIs           ConsumerDomain = "INTERNAL_APIS"
	DomainBusinessObjects        ConsumerDomain = "BUSINESS_OBJECTS"
	DomainUnknown                ConsumerDomain = "UNKNOWN"
)

// Severity levels for blast radius reports
type BlastRadiusSeverity string

const (
	BlastSeverityCritical BlastRadiusSeverity = "CRITICAL"
	BlastSeverityHigh     BlastRadiusSeverity = "HIGH"
	BlastSeverityMedium   BlastRadiusSeverity = "MEDIUM"
	BlastSeverityLow      BlastRadiusSeverity = "LOW"
)

// ConsumerCriticalityWeights assigns a blast-radius weight per consumer domain
var ConsumerCriticalityWeights = map[ConsumerDomain]float64{
	DomainRegulatoryExporters:     10.0,
	DomainReactDashboards:          8.0,
	DomainDownstreamTenantQueries: 6.0,
	DomainInternalAPIs:            5.0,
	DomainBusinessObjects:         4.0,
	DomainUnknown:                 1.0,
}

// DefaultWeight is the weight used when a domain is not recognized
const DefaultWeight = 1.0

// WeightFor returns the criticality weight for a given consumer domain
func WeightFor(domain ConsumerDomain) float64 {
	if w, ok := ConsumerCriticalityWeights[domain]; ok {
		return w
	}
	return DefaultWeight
}

// BlastRadiusSummary holds the computed blast radius score and severity
type BlastRadiusSummary struct {
	TotalImpactedArtifacts int                   `json:"total_impacted_artifacts"`
	Severity                BlastRadiusSeverity   `json:"severity"`
	WeightedScore           float64              `json:"weighted_score"`
}

// ImpactedConsumer groups artifacts by their consumer domain
type ImpactedConsumer struct {
	Domain    ConsumerDomain `json:"domain"`
	Artifacts []string      `json:"artifacts"`
	Risk      string        `json:"risk"`
}

// BlastRadiusReport is the structured output of the Impact Simulator
type BlastRadiusReport struct {
	TargetNode          string             `json:"target_node"`
	Action              string             `json:"action"`
	BlastRadiusSummary  BlastRadiusSummary `json:"blast_radius_summary"`
	UpstreamNodes       []LineageNode     `json:"upstream_nodes"`
	DownstreamNodes     []LineageNode     `json:"downstream_nodes"`
	ImpactedConsumers   []ImpactedConsumer `json:"impacted_consumers"`
}

// ComputeBlastRadiusSeverity computes severity from a distance-decay weighted score
func ComputeBlastRadiusSeverity(score float64) BlastRadiusSeverity {
	switch {
	case score >= 8.0:
		return BlastSeverityCritical
	case score >= 4.0:
		return BlastSeverityHigh
	case score >= 1.0:
		return BlastSeverityMedium
	default:
		return BlastSeverityLow
	}
}
