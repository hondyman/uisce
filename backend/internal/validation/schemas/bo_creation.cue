// Business Object Creation Validation Schema
// Validates BO creation payloads from the wizard
package bo

// Business Object creation payload
bo_creation: {
    // ID is optional on create, required on update
    bo_id?: string @go(BOID)

    // Name must be alphanumeric with underscores (technical key)
    name: string & !="" & =~"^[a-zA-Z][a-zA-Z0-9_]*$" @go(Name)
    
    // Display name for UI (human readable)
    display_name: string & !="" @go(DisplayName)
    
    // Description is optional
    description?: string @go(Description)

    // Driving table is required
    driver_table_id: string & !="" @go(DriverTableID)
    driver_table_name?: string     @go(DriverTableName)

    // Optional categorization
    category?: string @go(Category)
    owner?: string    @go(Owner)
    steward?: string  @go(Steward)

    // Lifecycle state
    state: *"draft" | "published" | "archived" | "deprecated" @go(State)

    // Tenant isolation
    tenant_id: string & !="" @go(TenantID)
    datasource_id?: string   @go(DatasourceID)

    // Selected semantic terms from the driving table
    terms: [...#BOTermReference] @go(Terms)

    // Related business objects to link
    related_bos?: [...#BORelationshipReference] @go(RelatedBOs)

    // Included terms from related tables (flattening)
    included_terms_from_tables?: [...#IncludedTableTerms] @go(IncludedTermsFromTables)
}

// Reference to a term being added to the BO
#BOTermReference: {
    term_id: string & !="" @go(TermID)
    
    // Optional initial metadata overrides
    display_name?: string  @go(DisplayName)
    group_name?: string    @go(GroupName)
    required?: bool | *false @go(Required)
    visible?: bool | *true   @go(Visible)
    sort_order?: int | *0    @go(SortOrder)
}

// Reference to a related BO
#BORelationshipReference: {
    target_bo_id: string & !=""       @go(TargetBOID)
    relationship_name?: string        @go(RelationshipName)
    relationship_type: *"RELATES_TO" | "PARENT_OF" | "CHILD_OF" | "DEPENDS_ON" @go(RelationshipType)
    cardinality?: "one_to_one" | "one_to_many" | "many_to_one" | "many_to_many" @go(Cardinality)
}

// Terms included from a related table (flattening into this BO)
#IncludedTableTerms: {
    table_id: string & !=""   @go(TableID)
    term_ids: [...string]     @go(TermIDs)
}

// Constraints and validations
bo_creation: {
    // Name validation: minimum 2 characters
    name: =~".{2,}"

    // Display name validation: minimum 2 characters
    display_name: =~".{2,}"

    // At least one term should be selected (warning, not hard fail)
    // Uncomment to enforce:
    // if len(terms) == 0 {
    //     _warning: "NO_TERMS_SELECTED"
    // }
}

// BO update payload (extends creation with required ID)
bo_update: bo_creation & {
    bo_id: string & !="" // Required for updates
}
