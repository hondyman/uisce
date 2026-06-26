// BO Term Metadata Validation Schema
// Validates metadata for semantic terms within a Business Object
package bo

// Business Object Term Metadata Validation Schema
bo_term_metadata: {
    bo_id:   string & !="" @go(BOID)
    term_id: string & !="" @go(TermID)

    display_name: string | *""  @go(DisplayName)
    description:  string | *""  @go(Description)
    group_name:   string | *""  @go(GroupName)

    required: bool | *false @go(Required)
    visible:  bool | *true  @go(Visible)

    // Formatting rules
    format: *"string" | "number" | "currency" | "percent" | "date" | "integer" | "boolean" @go(Format)

    // Precision only valid for numeric formats
    precision: int | *0 @go(Precision)
    
    // Precision constraints based on format
    if format == "number" || format == "currency" || format == "percent" {
        precision: >=0 & <=10
    }
    if format == "string" || format == "date" || format == "boolean" {
        precision: 0
    }

    // Currency code (only for currency format)
    currency_code: string | *"USD" @go(CurrencyCode)
    if format != "currency" {
        currency_code: "" | "USD"
    }

    // Date format (only for date format)
    date_format: string | *"YYYY-MM-DD" @go(DateFormat)
    if format != "date" {
        date_format: "" | "YYYY-MM-DD"
    }

    // Aggregation rules
    aggregation: *"none" | "sum" | "avg" | "min" | "max" | "count" @go(Aggregation)

    // Aggregation constraints: sum/avg/min/max only for numeric formats
    if aggregation == "sum" || aggregation == "avg" || aggregation == "min" || aggregation == "max" {
        format: "number" | "currency" | "percent" | "integer"
    }

    // Count aggregation forces precision to 0
    if aggregation == "count" {
        precision: 0
    }

    // Ordering
    sort_order: int | *0 @go(SortOrder)
}

// Semantic type aware validation extension
// Backend populates semantic_type from catalog before validation
bo_term_metadata_with_semantic_type: bo_term_metadata & {
    semantic_type: "string" | "number" | "date" | "boolean" | "json"

    // Enforce format compatibility with semantic type
    if semantic_type == "number" {
        format: "number" | "currency" | "percent" | "integer"
    }
    if semantic_type == "date" {
        format: "date"
    }
    if semantic_type == "boolean" {
        format: "boolean" | "string"
    }
}

// Calculation term extension
// Backend sets is_calculation based on node type
bo_term_metadata_for_calculation: bo_term_metadata & {
    is_calculation: bool | *false @go(IsCalculation)

    if is_calculation {
        // Calculations must have a format other than string (usually derived)
        format: "number" | "currency" | "percent" | "date" | "integer"
        
        // Calculations cannot be marked as required (they compute on demand)
        required: false
    }
}
