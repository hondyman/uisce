package boresolver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// QueryDef is the frontend Query Builder contract. It is the only input the
// UI ever sends; SQL construction happens exclusively in the backend.
type QueryDef struct {
	Context QueryContext `json:"context"`
	Query   QueryRequest `json:"query"`
}

// QuerySubjectKind discriminates the analytical subject for Query / Report builders.
const (
	QuerySubjectBusinessObject = "business_object"
	QuerySubjectCube           = "cube"
)

// ContractVersionPin is a cube contract pin: a positive integer, or "latest"
// (wire) which unmarshals to Latest=true / Version=0.
type ContractVersionPin struct {
	Latest  bool
	Version int
}

// UnmarshalJSON accepts a JSON number or the string "latest".
func (p *ContractVersionPin) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || bytes.Equal(b, []byte("null")) {
		*p = ContractVersionPin{Latest: true}
		return nil
	}
	if bytes.Equal(b, []byte(`"latest"`)) {
		*p = ContractVersionPin{Latest: true}
		return nil
	}
	var n int
	if err := json.Unmarshal(b, &n); err != nil {
		return fmt.Errorf("contractVersion: want number or \"latest\": %w", err)
	}
	if n <= 0 {
		*p = ContractVersionPin{Latest: true}
		return nil
	}
	*p = ContractVersionPin{Version: n}
	return nil
}

// MarshalJSON emits "latest" or a positive integer.
func (p ContractVersionPin) MarshalJSON() ([]byte, error) {
	if p.Latest || p.Version <= 0 {
		return []byte(`"latest"`), nil
	}
	return json.Marshal(p.Version)
}

// QuerySubject is the shared analytical subject for Query Builder and Report
// Builder (CUBE-1.6). Absent Subject on QueryContext means a legacy BO-only
// row: synthesize business_object from boId/bindingId.
type QuerySubject struct {
	Kind string `json:"kind"`

	// business_object fields (also copied onto QueryContext.BOID/BindingID for
	// the compile path so existing generators keep working).
	BOID         string   `json:"boId,omitempty"`
	BindingID    string   `json:"bindingId,omitempty"`
	RelatedBOIDs []string `json:"relatedBoIds,omitempty"`

	// cube fields — pin cubeId + contractVersion (fail closed when unresolved).
	CubeID          string             `json:"cubeId,omitempty"`
	ContractVersion ContractVersionPin `json:"contractVersion,omitempty"`
}

// NormalizedKind returns the canonical subject kind, defaulting empty to
// business_object for backward compatibility.
func (s *QuerySubject) NormalizedKind() string {
	if s == nil {
		return QuerySubjectBusinessObject
	}
	switch strings.ToLower(strings.TrimSpace(s.Kind)) {
	case QuerySubjectCube:
		return QuerySubjectCube
	default:
		return QuerySubjectBusinessObject
	}
}

// QueryContext carries the security + binding scope for the query.
type QueryContext struct {
	BOID      string `json:"boId"`
	BindingID string `json:"bindingId"`
	TenantID  string `json:"tenantId"`
	// RelatedBOIDs lists additional Business Objects to join into the query
	// alongside BOID. Each must have a resolvable join path from BOID via
	// the relationship graph (analytics.RelationshipInferenceService) —
	// the server resolves and validates the join, the client never supplies
	// raw join SQL.
	RelatedBOIDs []string `json:"relatedBoIds,omitempty"`
	// Subject is the shared QuerySubject (business_object | cube). Optional for
	// backward compatibility: when omitted, callers treat the query as a BO
	// subject synthesized from BOID/BindingID/RelatedBOIDs.
	Subject *QuerySubject `json:"subject,omitempty"`
}

// QueryRequest is the user-intent portion of the QueryDef.
type QueryRequest struct {
	Dimensions []DimensionDef `json:"dimensions"`
	Measures   []MeasureDef   `json:"measures"`
	Filters    []FilterDef    `json:"filters"`
	GroupBy    []string       `json:"groupBy,omitempty"`
	Limit      int            `json:"limit,omitempty"`
}

// DimensionDef selects a semantic term to group/project by.
type DimensionDef struct {
	TermNodeID string `json:"termNodeId"`
	Alias      string `json:"alias"`
	// BOID is the Business Object this term belongs to. Empty means the
	// primary BO (QueryContext.BOID); otherwise it must be one of
	// QueryContext.RelatedBOIDs.
	BOID string `json:"boId,omitempty"`
}

// MeasureDef selects a semantic term with an aggregation.
type MeasureDef struct {
	TermNodeID  string `json:"termNodeId"`
	Alias       string `json:"alias"`
	Aggregation string `json:"agg"`
	BOID        string `json:"boId,omitempty"`
}

// FilterDef applies a predicate to a semantic term.
type FilterDef struct {
	TermNodeID string      `json:"termNodeId"`
	Operator   string      `json:"operator"`
	Value      interface{} `json:"value,omitempty"`
	BOID       string      `json:"boId,omitempty"`
}

// QueryPreviewResponse is returned by POST /api/query/preview.
type QueryPreviewResponse struct {
	SQL        string        `json:"sql"`
	Dialect    string        `json:"dialect,omitempty"`
	Parameters []interface{} `json:"parameters,omitempty"`
	// Columns is populated for multi-BO queries (QueryContext.RelatedBOIDs)
	// with each column's source BO and cardinality. Empty for single-BO
	// queries, where every column trivially belongs to the primary BO.
	Columns []QueryResultColumn `json:"columns,omitempty"`

	// CubeHit is set when the query was served from a cube materialization.
	// It is an observability field: the SQL in this response is the base
	// statement, and the router reports which materialization WOULD serve it
	// faster. Nil means the base path is authoritative.
	//
	// Declared as a local struct rather than the querybuilder type to keep
	// boresolver free of a dependency on the query builder package.
	CubeHit *CubeHitInfo `json:"cubeHit,omitempty"`
	// CubeMiss explains why no materialization was used. Empty when CubeHit is
	// set, and empty when cube routing is not installed at all.
	CubeMiss string `json:"cubeMiss,omitempty"`
}

// CubeHitInfo describes the materialization a cube router selected.
type CubeHitInfo struct {
	CubeID          string   `json:"cubeId"`
	CubeName        string   `json:"cubeName"`
	Materialization string   `json:"materialization"`
	Grain           []string `json:"grain"`
	// ServedFrom is the dual-tier route taken: "hot" (StarRocks MV), "cold"
	// (Iceberg), or empty when unset. Preview/Execute badges read this field.
	ServedFrom string `json:"servedFrom,omitempty"`
	// ContractVersion is the cube contract version that served the hit.
	ContractVersion int `json:"contractVersion,omitempty"`
	// Stale is true when the materialization is behind its source. The data is
	// still served, flagged, unless the cube's stalePolicy forces a fallback.
	Stale bool `json:"stale"`
}

// QueryExecuteResponse is returned by POST /api/query/execute.
type QueryExecuteResponse struct {
	SQL             string                   `json:"sql"`
	Columns         []QueryResultColumn      `json:"columns"`
	Rows            []map[string]interface{} `json:"rows"`
	RowCount        int                      `json:"rowCount"`
	ExecutionTimeMs int64                    `json:"executionTimeMs"`
	// CubeHit mirrors QueryPreviewResponse.CubeHit so the executed result
	// carries the same routing provenance as the preview that preceded it.
	CubeHit *CubeHitInfo `json:"cubeHit,omitempty"`
	// CubeMiss explains why no materialization served the query.
	CubeMiss string `json:"cubeMiss,omitempty"`
}

// QueryResultColumn describes one result column.
type QueryResultColumn struct {
	Name string `json:"name"`
	Type string `json:"type"`
	// BOID is the Business Object this column was selected from.
	BOID string `json:"boId,omitempty"`
	// Cardinality is "one" or "many", describing how a row of this column's
	// BO relates to the primary BO's row: "one" means the join is a lookup
	// (1:1/M:1) and the column flattens safely into the primary row; "many"
	// means the join is 1:M/M:M (PeopleSoft calls this a child "scroll
	// level") — the UI must render it as a nested/grouped child collection
	// rather than flattening it, or results silently fan out duplicate
	// primary rows. Empty/omitted means the column belongs to the primary
	// BO itself.
	Cardinality string `json:"cardinality,omitempty"`
	// RootOwnership is "unique" or "shared", describing whether a row of
	// this column's BO is reachable from more than one primary-BO row.
	// "unique" means every hop back to the primary BO is 1:1/1:M (no
	// "M:1"/"M:M" anywhere), so this column's value belongs to exactly
	// one primary row. "shared" means at least one hop back is M:1/M:M -
	// e.g. many orders referencing the same region - so the SAME target
	// row (and its column value) can appear under more than one primary
	// row. This is independent of Cardinality: a plain lookup column
	// (Cardinality "one", e.g. order -> region) can still be "shared".
	//
	// The distinction matters entirely downstream of this generator: a
	// consumer that sums or averages this column ACROSS the returned
	// rows will double-count whenever two rows share the same target row
	// - correct per-row values, wrong rolled-up total. This field exists
	// so that check can be made without re-deriving it from the join
	// path. Empty/omitted means unique (no related-BO join, or a path
	// this generator hasn't computed ownership for).
	RootOwnership string `json:"rootOwnership,omitempty"`
	// Aggregation is the SQL aggregation the generator wrapped around
	// this column's expression - "sum" | "avg" | "count" | "min" |
	// "max" | "count_distinct", lowercased at construction so the wire
	// shape is a column-fact and consumers never re-derive it from the
	// query's MeasureDef. Empty/omitted means this column is NOT
	// aggregated (a dimension or a non-aggregated measure) - the unsafe
	// sentinel on the consumption side, not a placeholder string.
	//
	// Distinct from Cardinality/RootOwnership: those say "is every row's
	// value representable once at this grain?" (a fan-out/duplicate-row
	// question). Aggregation says "is `+` a meaningful way to combine
	// these values across rows at all?" (a linearity question). Both
	// questions are independent and either alone can produce a wrong
	// roll-up; consumed together as a two-signal AND at the call site.
	// See frontend isSafeToRollUpAcrossRows (grain/ownership) and
	// isAdditiveSafe (linearity) for the split.
	Aggregation string `json:"aggregation,omitempty"`
}

// SemanticTermView is the shape exposed by GET /api/business-objects/{boId}/terms.
type SemanticTermView struct {
	TermNodeID         string `json:"termNodeId"`
	TermKey            string `json:"termKey"`
	TermName           string `json:"termName"`
	DisplayName        string `json:"displayName"`
	Description        string `json:"description,omitempty"`
	DataType           string `json:"dataType"`
	Role               string `json:"role"`
	BindingStatus      string `json:"bindingStatus"`
	DefaultAggregation string `json:"defaultAggregation,omitempty"`
	// TermType is "calculated" for calc-term catalog nodes (vm.Expression
	// backed), empty for physical-column terms. The frontend uses this to
	// avoid auto-populating aggregation for calc terms — their compiled
	// expression defines any aggregation internally.
	TermType string `json:"termType,omitempty"`
	// DrillPath lists the term node ids of successive drill-down levels
	// configured once on this field's semantic term (catalog_node.properties
	// ->'drill_path') and inherited by every BO field bound to that term —
	// e.g. region -> country -> state. Empty means this field isn't part of
	// a drill hierarchy; widgets fall back to plain cross-filtering on click.
	DrillPath []string `json:"drillPath,omitempty"`
}

// BOBinding captures the physical binding context for a Business Object.
// db tags are required: sqlx's default column mapper only lowercases field
// names (DrivingTable -> "drivingtable"), it doesn't insert underscores, so
// without these tags a snake_case query alias like "driving_table" never
// matches its destination field and struct-scans silently fail to populate it.
type BOBinding struct {
	ID               string `db:"id"`
	Name             string `db:"name"`
	DialectName      string `db:"dialect_name"`
	ConnectionString string `db:"connection_string"`
	BindingID        string `db:"binding_id"`
	BOID             string `db:"bo_id"`
	DatasourceID     string `db:"datasource_id"`
	DrivingTable     string `db:"driving_table"`
	DrivingTableID   string `db:"driving_table_id"`
}
