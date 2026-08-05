package cube

import (
	"github.com/google/uuid"
)

type Cube struct {
	Name         string                      `json:"name"`
	SQL          string                      `json:"sql"`
	SQLTable     string                      `json:"sql_table,omitempty"`
	SQLAlias     string                      `json:"sql_alias,omitempty"`
	Extends      any                         `json:"extends,omitempty"`
	DataSource   string                      `json:"data_source,omitempty"`
	Dimensions   map[string]map[string]any   `json:"dimensions"`
	Measures     map[string]map[string]any   `json:"measures"`
	Joins        map[string]map[string]any   `json:"joins"`
	Segments     map[string]map[string]any   `json:"segments,omitempty"`
	Hierarchies  []map[string]any           `json:"hierarchies,omitempty"`
	Tags         []string                    `json:"tags,omitempty"`
	PreAggregations map[string]map[string]any `json:"pre_aggregations,omitempty"`
	AccessPolicy map[string]any              `json:"access_policy,omitempty"`
	RefreshKey   map[string]any              `json:"refresh_key,omitempty"`
	Title        string                      `json:"title,omitempty"`
	Description  string                      `json:"description,omitempty"`
	Public       *bool                       `json:"public,omitempty"`
	Meta         map[string]any              `json:"meta,omitempty"`
	Metadata     map[string]any              `json:"metadata,omitempty"`
	FabricDefnID *uuid.UUID                  `json:"fabric_defn_id,omitempty"`
}

type ViewMeta struct {
	Schema      string           `json:"schema"`
	Name        string           `json:"name"`
	Cubes       []string         `json:"cubes"`
	Filters     []map[string]any `json:"filters"`
	Description string           `json:"description,omitempty"`
	Tags        []string         `json:"tags,omitempty"`
	Owner       string           `json:"owner,omitempty"`
	Dimensions  []string         `json:"dimensions"`
	Measures    []string         `json:"measures"`
}

type Catalog struct {
	Cubes map[string]Cube    `json:"cubes"`
	Views map[string]ViewMeta `json:"views"`
}

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

type ValidationIssue struct {
	Level   string `json:"level"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Dialect interface {
	Quote(identifier string) string
	EscapeString(s string) string
}
