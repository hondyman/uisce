// Semantic Term Physical Mapping Validation Schema
// Validates mappings between semantic terms and physical columns
package bo

// Mapping of a semantic term to physical column(s)
semantic_term_physical_mapping: {
    term_id: string & !="" @go(TermID)
    term_name?: string     @go(TermName)

    // Semantic metadata
    semantic_type: "string" | "number" | "date" | "boolean" | "json" @go(SemanticType)

    // Mappings to physical columns (1:M support)
    mappings: [...#PhysicalMapping] @go(Mappings)

    // Default mapping ID (when multiple exist)
    default_mapping_id?: string @go(DefaultMappingID)

    // Tie-breaker configuration for 1:M resolution
    tie_breaker?: #TieBreakerConfig @go(TieBreaker)
}

// Individual physical mapping
#PhysicalMapping: {
    mapping_id?: string @go(MappingID)

    // Context in which this mapping applies
    context_type: "table" | "business_object" | "datasource" | "tenant" @go(ContextType)
    context_id: string & !="" @go(ContextID)

    // Physical column reference
    datasource_id?: string      @go(DatasourceID)
    table_id: string & !=""     @go(TableID)
    table_name?: string         @go(TableName)
    column_id: string & !=""    @go(ColumnID)
    column_name?: string        @go(ColumnName)

    // Optional expression for derived physical values
    expression?: string @go(Expression)

    // Physical data type
    physical_data_type?: string @go(PhysicalDataType)

    // Priority for 1:M resolution (lower = higher priority)
    priority: int | *0 @go(Priority)

    // Is this the primary/default mapping?
    is_default?: bool | *false @go(IsDefault)

    // Governance
    status?: *"active" | "deprecated" | "pending" @go(Status)
}

// Tie-breaker configuration for 1:M semantic-to-physical
#TieBreakerConfig: {
    strategy: "precedence" | "latest_timestamp" | "custom" | "priority" @go(Strategy)
    
    // For precedence strategy: ordered list of context IDs
    precedence?: [...string] @go(Precedence)
    
    // For timestamp strategy: column to use
    timestamp_column?: string @go(TimestampColumn)
    
    // For custom strategy: expression
    custom_expression?: string @go(CustomExpression)
    
    description?: string @go(Description)
}

// Validation constraints
semantic_term_physical_mapping: {
    // At least one mapping required
    if len(mappings) == 0 {
        _maps: "MAPPINGS_REQUIRED"
    }

    // No duplicate (context_type, context_id) pairs for same term
    for i, m in mappings {
        for j, n in mappings if i < j {
            if m.context_type == n.context_type && m.context_id == n.context_id {
                _dup: "DUPLICATE_CONTEXT_MAPPING: \(m.context_type)/\(m.context_id)"
            }
        }
    }

    // If multiple mappings, tie_breaker should be defined
    if len(mappings) > 1 && tie_breaker == _|_ {
        _tie: "TIE_BREAKER_RECOMMENDED_FOR_MULTIPLE_MAPPINGS"
    }

    // Exactly one default mapping if multiple exist
    let defaultCount = len([for m in mappings if m.is_default { m }])
    if len(mappings) > 1 && defaultCount != 1 {
        _default: "EXACTLY_ONE_DEFAULT_MAPPING_REQUIRED"
    }
}

// Mapping batch for bulk operations
semantic_term_mapping_batch: {
    tenant_id: string & !=""    @go(TenantID)
    datasource_id?: string      @go(DatasourceID)
    
    term_mappings: [...semantic_term_physical_mapping] @go(TermMappings)
}

// Physical column reference (minimal)
#PhysicalColumnRef: {
    table_id: string & !=""  @go(TableID)
    column_id: string & !="" @go(ColumnID)
    qualified_name?: string  @go(QualifiedName)
}

// Data type compatibility matrix
#DataTypeCompatibility: {
    semantic_type: "string" | "number" | "date" | "boolean" | "json"
    
    allowed_physical_types: [...string]
    
    if semantic_type == "number" {
        allowed_physical_types: ["integer", "bigint", "smallint", "numeric", "decimal", "float", "double", "real"]
    }
    if semantic_type == "string" {
        allowed_physical_types: ["varchar", "text", "char", "string", "nvarchar"]
    }
    if semantic_type == "date" {
        allowed_physical_types: ["date", "timestamp", "timestamptz", "datetime", "time"]
    }
    if semantic_type == "boolean" {
        allowed_physical_types: ["boolean", "bool", "bit"]
    }
    if semantic_type == "json" {
        allowed_physical_types: ["json", "jsonb", "variant", "object"]
    }
}
