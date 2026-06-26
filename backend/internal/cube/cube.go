package cube

import "github.com/google/uuid"

// Cube is a lightweight representation of a semantic cube used by the diff engine.
type Cube struct {
	Name string `json:"name"`
	// Either sql or sql_table can be used by generators
	SQL             string                    `json:"sql"`
	SQLTable        string                    `json:"sql_table,omitempty"`
	SQLAlias        string                    `json:"sql_alias,omitempty"`
	Extends         any                       `json:"extends,omitempty"`
	DataSource      string                    `json:"data_source,omitempty"`
	Dimensions      map[string]map[string]any `json:"dimensions"`
	Measures        map[string]map[string]any `json:"measures"`
	Joins           map[string]map[string]any `json:"joins"`
	Segments        map[string]map[string]any `json:"segments,omitempty"`
	Hierarchies     []map[string]any          `json:"hierarchies,omitempty"`
	DrillMembers    []string                  `json:"drill_members,omitempty"`
	Tags            []string                  `json:"tags,omitempty"`
	PreAggregations map[string]map[string]any `json:"pre_aggregations,omitempty"`
	AccessPolicy    map[string]any            `json:"access_policy,omitempty"`
	RefreshKey      map[string]any            `json:"refresh_key,omitempty"`
	Title           string                    `json:"title,omitempty"`
	Description     string                    `json:"description,omitempty"`
	Public          *bool                     `json:"public,omitempty"`
	Meta            map[string]any            `json:"meta,omitempty"`
	// Internal metadata for Semlayer (not part of Cube.js spec)
	Metadata map[string]any `json:"metadata,omitempty"`
	// Fabric definition ID that this cube originated from (for UUID-based extension references)
	FabricDefnID *uuid.UUID `json:"fabric_defn_id,omitempty"`
}

// Catalog represents the semantic model catalog.
type Catalog struct {
	Cubes map[string]Cube
	Views map[string]ViewMeta
}

// ViewMeta represents metadata for a view.
type ViewMeta struct {
	Schema      string
	Name        string
	Cubes       []string         `json:"cubes"`
	Filters     []map[string]any `json:"filters"`
	Description string           `json:"description,omitempty"`
	Tags        []string         `json:"tags,omitempty"`
	Owner       string           `json:"owner,omitempty"`
	Dimensions  []string         `json:"dimensions"`
	Measures    []string         `json:"measures"`
	// Add other fields as needed based on usage in the codebase
}

// QueryRequest is the format expected by the query engine.
type QueryRequest struct {
	Cubes      []string         `json:"cubes"`
	QueryType  string           `json:"queryType"`
	Measures   []string         `json:"measures"`
	Dimensions []string         `json:"dimensions"`
	Timezone   string           `json:"timezone"`
	Limit      *int             `json:"limit,omitempty"`
	Offset     *int             `json:"offset,omitempty"`
	Filters    []map[string]any `json:"filters"`
	Order      []any            `json:"order"`
}
