// Cube.dev Model Schema
// This CUE schema validates semantic term definitions before Cube YAML generation

package cube

// CubeModel is the top-level structure for a generated Cube
#CubeModel: {
    cubes: [...#Cube]
}

// Cube represents a single Cube definition
#Cube: {
    name:       #Identifier
    sql?:       string
    sql_table?: string
    data_source?: string
    
    // At least one of sql or sql_table must be provided
    _hasSql: (sql != _|_) | (sql_table != _|_)
    
    dimensions?: [...#Dimension]
    measures?:   [...#Measure]
    joins?:      [...#Join]
    segments?:   [...#Segment]
    pre_aggregations?: [...#PreAggregation]
}

// Dimension definition
#Dimension: {
    name:        #Identifier
    sql:         string
    type:        #DimensionType
    title?:      string
    description?: string
    primary_key?: bool
    shown?:      bool
    meta?:       {...}
}

// Measure definition
#Measure: {
    name:        #Identifier
    sql?:        string
    type:        #MeasureType
    title?:      string
    description?: string
    filters?:    [...#Filter]
    rolling_window?: #RollingWindow
    drill_members?: [...string]
    meta?:       {...}
}

// Join definition
#Join: {
    name:         #Identifier
    sql:          string
    relationship: #Cardinality
}

// Segment definition
#Segment: {
    name: #Identifier
    sql:  string
}

// Pre-aggregation definition
#PreAggregation: {
    name:        #Identifier
    measures?:   [...string]
    dimensions?: [...string]
    time_dimension?: string
    granularity?: #Granularity
    partition_granularity?: #Granularity
    refresh_key?: #RefreshKey
    indexes?: [...#Index]
}

#Index: {
    name:    #Identifier
    columns: [...string]
}

#RefreshKey: {
    every?:      string
    sql?:        string
    incremental?: bool
}

#RollingWindow: {
    trailing: string
    leading?: string
    offset?:  string
}

#Filter: {
    sql: string
}

// Type enums
#DimensionType: "string" | "number" | "boolean" | "time" | "geo"

#MeasureType: "count" | "count_distinct" | "count_distinct_approx" |
              "sum" | "avg" | "min" | "max" |
              "number" | "string" | "boolean" | "time"

#Cardinality: "one_to_one" | "one_to_many" | "many_to_one" | "belongs_to" | "has_many" | "has_one"

#Granularity: "second" | "minute" | "hour" | "day" | "week" | "month" | "quarter" | "year"

// Identifier must start with letter and contain only letters, numbers, underscore
#Identifier: =~"^[a-zA-Z][a-zA-Z0-9_]*$"

// Validation rules
#DangerousSQLPattern: !~"(?i)(drop|truncate|delete|insert|update|alter|create)"
