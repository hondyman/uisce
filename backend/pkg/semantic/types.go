package semantic

import (
	"time"
)

// Cube represents a semantic data model
type Cube struct {
	ID              string                 `json:"id" db:"id"`
	TenantID        string                 `json:"tenant_id" db:"tenant_id"`
	Name            string                 `json:"name" db:"name"`
	DisplayName     string                 `json:"display_name" db:"display_name"`
	Description     string                 `json:"description" db:"description"`
	SQL             string                 `json:"sql" db:"sql"`
	SourceCubeID    *string                `json:"source_cube_id,omitempty" db:"source_cube_id"`
	IsSystem        bool                   `json:"is_system" db:"is_system"`
	RefreshKey      string                 `json:"refresh_key,omitempty" db:"refresh_key"`
	PreAggregations []PreAggregation       `json:"pre_aggregations,omitempty" db:"pre_aggregations"`
	Joins           []Join                 `json:"joins,omitempty" db:"joins"`
	Metadata        map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	Status          string                 `json:"status" db:"status"`
	Version         int                    `json:"version" db:"version"`
	CreatedBy       string                 `json:"created_by,omitempty" db:"created_by"`
	CreatedAt       time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at" db:"updated_at"`

	// Loaded relationships
	Dimensions []Dimension `json:"dimensions,omitempty" db:"-"`
	Measures   []Measure   `json:"measures,omitempty" db:"-"`
}

// Dimension represents a dimension in a cube
type Dimension struct {
	ID            string                 `json:"id" db:"id"`
	CubeID        string                 `json:"cube_id" db:"cube_id"`
	Name          string                 `json:"name" db:"name"`
	DisplayName   string                 `json:"display_name" db:"display_name"`
	Type          string                 `json:"type" db:"type"` // string, number, time, geo, boolean
	SQL           string                 `json:"sql" db:"sql"`
	Format        string                 `json:"format,omitempty" db:"format"`
	CaseSensitive bool                   `json:"case_sensitive" db:"case_sensitive"`
	PrimaryKey    bool                   `json:"primary_key" db:"primary_key"`
	Shown         bool                   `json:"shown" db:"shown"`
	Metadata      map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
}

// Measure represents a measure (metric) in a cube
type Measure struct {
	ID            string                 `json:"id" db:"id"`
	CubeID        string                 `json:"cube_id" db:"cube_id"`
	Name          string                 `json:"name" db:"name"`
	DisplayName   string                 `json:"display_name" db:"display_name"`
	Type          string                 `json:"type" db:"type"` // count, sum, avg, min, max, countDistinct, etc.
	SQL           string                 `json:"sql" db:"sql"`
	Format        string                 `json:"format,omitempty" db:"format"`
	RollingWindow string                 `json:"rolling_window,omitempty" db:"rolling_window"`
	DrillMembers  []string               `json:"drill_members,omitempty" db:"drill_members"`
	Filters       []Filter               `json:"filters,omitempty" db:"filters"`
	Metadata      map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
}

// PreAggregation represents a pre-aggregation for performance
type PreAggregation struct {
	ID                   string                 `json:"id" db:"id"`
	CubeID               string                 `json:"cube_id" db:"cube_id"`
	Name                 string                 `json:"name" db:"name"`
	Type                 string                 `json:"type" db:"type"`               // rollup, originalSql, rollupJoin
	Region               string                 `json:"region,omitempty" db:"region"` // region for which this pre-aggregation is valid
	Dimensions           []string               `json:"dimensions,omitempty" db:"dimensions"`
	Measures             []string               `json:"measures,omitempty" db:"measures"`
	Segments             []string               `json:"segments,omitempty" db:"segments"`
	TimeDimension        string                 `json:"time_dimension,omitempty" db:"time_dimension"`
	Granularity          string                 `json:"granularity,omitempty" db:"granularity"`
	PartitionGranularity string                 `json:"partition_granularity,omitempty" db:"partition_granularity"`
	RefreshKey           string                 `json:"refresh_key,omitempty" db:"refresh_key"`
	Indexes              []Index                `json:"indexes,omitempty" db:"indexes"`
	BuildRangeStart      string                 `json:"build_range_start,omitempty" db:"build_range_start"`
	BuildRangeEnd        string                 `json:"build_range_end,omitempty" db:"build_range_end"`
	Metadata             map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	LastBuiltAt          *time.Time             `json:"last_built_at,omitempty" db:"last_built_at"`
	Status               string                 `json:"status" db:"status"`
	CreatedAt            time.Time              `json:"created_at" db:"created_at"`
}

// Join represents a join between cubes
type Join struct {
	Name         string `json:"name"`
	SQL          string `json:"sql"`
	Relationship string `json:"relationship"` // hasOne, hasMany, belongsTo
}

// Index represents an index on a pre-aggregation
type Index struct {
	Columns []string `json:"columns"`
	Type    string   `json:"type,omitempty"` // btree, hash, etc.
}

// Filter represents a filter condition
type Filter struct {
	Member   string   `json:"member"`
	Operator string   `json:"operator"` // equals, notEquals, contains, gt, gte, lt, lte, inDateRange
	Values   []string `json:"values"`
}

// Query represents a semantic query
type Query struct {
	Measures       []string          `json:"measures"`
	Dimensions     []string          `json:"dimensions,omitempty"`
	TimeDimensions []TimeDimension   `json:"timeDimensions,omitempty"`
	Filters        []Filter          `json:"filters,omitempty"`
	Segments       []string          `json:"segments,omitempty"`
	Order          map[string]string `json:"order,omitempty"` // member -> asc/desc
	Limit          int               `json:"limit,omitempty"`
	Offset         int               `json:"offset,omitempty"`
	Timezone       string            `json:"timezone,omitempty"`
}

// TimeDimension represents a time-based dimension in a query
type TimeDimension struct {
	Dimension   string   `json:"dimension"`
	Granularity string   `json:"granularity,omitempty"` // hour, day, week, month, quarter, year
	DateRange   []string `json:"dateRange,omitempty"`   // [start, end] or relative like "last 7 days"
}

// QueryResult represents the result of a query execution
type QueryResult struct {
	Data          []map[string]interface{} `json:"data"`
	Annotation    QueryAnnotation          `json:"annotation"`
	ExecutionTime int64                    `json:"executionTime"` // milliseconds
	CacheHit      bool                     `json:"cacheHit"`
	PreAggUsed    string                   `json:"preAggUsed,omitempty"`
}

// QueryAnnotation provides metadata about the query execution
type QueryAnnotation struct {
	Measures       map[string]MemberAnnotation `json:"measures"`
	Dimensions     map[string]MemberAnnotation `json:"dimensions"`
	TimeDimensions map[string]MemberAnnotation `json:"timeDimensions"`
	GeneratedSQL   string                      `json:"generatedSQL,omitempty"`
}

// MemberAnnotation provides metadata about a query member
type MemberAnnotation struct {
	Title      string `json:"title"`
	ShortTitle string `json:"shortTitle"`
	Type       string `json:"type"`
	Format     string `json:"format,omitempty"`
}

// QueryCache represents a cached query result
type QueryCache struct {
	ID              string                   `json:"id" db:"id"`
	TenantID        string                   `json:"tenant_id" db:"tenant_id"`
	QueryHash       string                   `json:"query_hash" db:"query_hash"`
	Query           Query                    `json:"query" db:"query"`
	Result          []map[string]interface{} `json:"result" db:"result"`
	ResultRows      int                      `json:"result_rows" db:"result_rows"`
	ExecutionTimeMs int                      `json:"execution_time_ms" db:"execution_time_ms"`
	CacheKey        string                   `json:"cache_key" db:"cache_key"`
	CreatedAt       time.Time                `json:"created_at" db:"created_at"`
	ExpiresAt       time.Time                `json:"expires_at" db:"expires_at"`
	LastAccessedAt  time.Time                `json:"last_accessed_at" db:"last_accessed_at"`
	AccessCount     int                      `json:"access_count" db:"access_count"`
}

// QueryHistory represents a query execution record
type QueryHistory struct {
	ID              string    `json:"id" db:"id"`
	TenantID        string    `json:"tenant_id" db:"tenant_id"`
	UserID          string    `json:"user_id,omitempty" db:"user_id"`
	CubeName        string    `json:"cube_name,omitempty" db:"cube_name"`
	Query           Query     `json:"query" db:"query"`
	GeneratedSQL    string    `json:"generated_sql,omitempty" db:"generated_sql"`
	ExecutionTimeMs int       `json:"execution_time_ms" db:"execution_time_ms"`
	ResultRows      int       `json:"result_rows" db:"result_rows"`
	CacheHit        bool      `json:"cache_hit" db:"cache_hit"`
	PreAggUsed      string    `json:"pre_agg_used,omitempty" db:"pre_agg_used"`
	Error           string    `json:"error,omitempty" db:"error"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// CubeMetadata represents compiled cube metadata for caching
type CubeMetadata struct {
	TenantID        string           `json:"tenant_id" db:"tenant_id"`
	CubeName        string           `json:"cube_name" db:"cube_name"`
	Metadata        Cube             `json:"metadata" db:"metadata"`
	Dimensions      []Dimension      `json:"dimensions" db:"dimensions"`
	Measures        []Measure        `json:"measures" db:"measures"`
	PreAggregations []PreAggregation `json:"pre_aggregations" db:"pre_aggregations"`
	CachedAt        time.Time        `json:"cached_at" db:"cached_at"`
}
