// Business Object Validation Schemas
// Main entry point for all BO-related CUE validation
package bo

// This package provides comprehensive validation schemas for:
//
// 1. bo_term_metadata.cue
//    - Term metadata within BOs (display, format, aggregation)
//    - Semantic type awareness
//    - Calculation constraints
//
// 2. bo_creation.cue
//    - BO creation payloads
//    - Term references
//    - Relationship references
//    - Flattening configuration
//
// 3. bo_relationships.cue
//    - BO-to-BO relationships
//    - Join path validation
//    - Cardinality inference
//    - Self-relationship prevention
//
// 4. calculation_definition.cue
//    - Semantic calculations (not SQL)
//    - Dependencies
//    - Evaluation modes
//    - Materialization
//
// 5. semantic_term_physical_mapping.cue
//    - Term-to-column mappings
//    - 1:M resolution with tie-breakers
//    - Context-aware mappings
//    - Data type compatibility

// Usage from Go:
//
//   import "cuelang.org/go/cue/cuecontext"
//
//   ctx := cuecontext.New()
//   schema := ctx.CompileString(schemaContent)
//   data := ctx.Encode(payload)
//   unified := schema.Unify(data)
//   if err := unified.Validate(); err != nil {
//       return handleValidationError(err)
//   }

// Validation result structure
#ValidationResult: {
    valid: bool
    errors?: [...#ValidationError]
    warnings?: [...#ValidationWarning]
}

#ValidationError: {
    code: string
    field?: string
    message: string
    severity: "error"
}

#ValidationWarning: {
    code: string
    field?: string
    message: string
    severity: "warning"
}

// Common validation patterns
#UUID: =~"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"
#Identifier: =~"^[a-zA-Z][a-zA-Z0-9_]*$"
#NonEmptyString: string & !=""

// Tenant context (used across all validations)
#TenantContext: {
    tenant_id: #UUID
    datasource_id?: #UUID
    user_id?: #UUID
    role?: string
}
