// Business Object Relationship Validation Schema
// Validates BO → BO relationships and join paths
package bo

// Payload for validating BO relationships
bo_relationships: {
    source_bo_id: string & !="" @go(SourceBOID)
    tenant_id: string & !=""    @go(TenantID)

    relationships: [...#BORelationship] @go(Relationships)
}

// Individual BO relationship definition
#BORelationship: {
    // Target BO being linked
    target_bo_id: string & !="" @go(TargetBOID)
    
    // Human-readable relationship name
    relationship_name: string | *"relates_to" @go(RelationshipName)
    
    // Relationship type for semantic meaning
    relationship_type: *"RELATES_TO" | "PARENT_OF" | "CHILD_OF" | "DEPENDS_ON" | "CONTAINS" | "REFERENCES" @go(RelationshipType)
    
    // Cardinality for analytics
    cardinality: *"one_to_many" | "one_to_one" | "many_to_one" | "many_to_many" @go(Cardinality)

    // Join path between driving tables (FK chain)
    join_path: [...#JoinPathStep] @go(JoinPath)

    // Optional: denormalized columns to include
    denormalized_fields?: [...string] @go(DenormalizedFields)

    // Relationship metadata
    description?: string @go(Description)
    is_required?: bool | *false @go(IsRequired)
}

// Step in a join path
#JoinPathStep: {
    from_table_id: string & !="" @go(FromTableID)
    from_column?: string         @go(FromColumn)
    to_table_id: string & !=""   @go(ToTableID)
    to_column?: string           @go(ToColumn)
    fk_name: string              @go(FKName)
    join_type?: *"INNER" | "LEFT" | "RIGHT" @go(JoinType)
}

// Relationship validation constraints
bo_relationships: {
    // Validate each relationship
    for r in relationships {
        // No self-relationships
        if r.target_bo_id == source_bo_id {
            _self_rel: "SELF_RELATIONSHIP_NOT_ALLOWED: \(r.target_bo_id)"
        }

        // Join path must be non-empty for explicit relationships
        if len(r.join_path) == 0 && r.relationship_type != "RELATES_TO" {
            _join: "JOIN_PATH_REQUIRED_FOR_STRUCTURED_RELATIONSHIPS"
        }
    }

    // No duplicate target BOs
    for i, r in relationships {
        for j, s in relationships if i < j {
            if r.target_bo_id == s.target_bo_id && r.relationship_name == s.relationship_name {
                _dup: "DUPLICATE_RELATIONSHIP: \(r.target_bo_id)"
            }
        }
    }
}

// Single relationship validation (for edge queue processing)
bo_relationship_single: #BORelationship & {
    source_bo_id: string & !="" @go(SourceBOID)
    tenant_id: string & !=""    @go(TenantID)

    // Ensure source != target
    if target_bo_id == source_bo_id {
        _error: "SELF_RELATIONSHIP_NOT_ALLOWED"
    }
}

// Relationship cardinality inference rules
#CardinalityInference: {
    // Source cardinality (how many source records per target)
    source_cardinality: "one" | "many"
    
    // Target cardinality (how many target records per source)
    target_cardinality: "one" | "many"

    // Combined cardinality
    cardinality: "\(source_cardinality)_to_\(target_cardinality)"
}
