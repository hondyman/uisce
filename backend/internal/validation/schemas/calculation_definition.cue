// Calculation Definition Validation Schema
// Validates semantic calculation terms (not raw SQL, but semantic expressions)
package bo

// Calculation term definition
calculation_definition: {
    // ID is optional on create
    calc_id?: string @go(CalcID)

    // Name must be alphanumeric with underscores
    name: string & !="" & =~"^[a-zA-Z][a-zA-Z0-9_]*$" @go(Name)
    
    // Human-readable display name
    display_name: string & !="" @go(DisplayName)
    
    // Description for documentation and LLM context
    description?: string @go(Description)

    // Semantic expression (NOT raw SQL)
    // Uses semantic term references, not physical column names
    expression: string & !="" @go(Expression)

    // Output data type
    output_type: "number" | "currency" | "percent" | "string" | "date" | "integer" | "boolean" @go(OutputType)

    // Precision for numeric outputs
    precision?: int | *2 @go(Precision)
    if output_type == "number" || output_type == "currency" || output_type == "percent" {
        precision: >=0 & <=10
    }
    if output_type == "string" || output_type == "date" || output_type == "boolean" || output_type == "integer" {
        precision: 0
    }

    // Evaluation mode
    evaluation_mode: *"live" | "pre_aggregated" | "hybrid" | "on_demand" @go(EvaluationMode)

    // Dependencies on other semantic terms
    dependencies: [...#CalcDependency] @go(Dependencies)

    // Optional materialization target
    materialization?: #MaterializationConfig @go(Materialization)

    // Aggregation behavior when used in rollups
    default_aggregation?: "sum" | "avg" | "min" | "max" | "count" | "none" @go(DefaultAggregation)

    // Tenant isolation
    tenant_id?: string @go(TenantID)

    // Ownership and governance
    owner?: string   @go(Owner)
    steward?: string @go(Steward)
    status?: *"draft" | "published" | "deprecated" @go(Status)
}

// Dependency on a semantic term
#CalcDependency: {
    term_id: string & !=""   @go(TermID)
    term_name?: string       @go(TermName)
    is_optional?: bool | *false @go(IsOptional)
}

// Materialization configuration
#MaterializationConfig: {
    target_type: "table" | "view" | "cube" | "metric" | "iceberg" @go(TargetType)
    target_name: string & !="" @go(TargetName)
    refresh_schedule?: string  @go(RefreshSchedule)
    partition_by?: [...string] @go(PartitionBy)
}

// Validation constraints
calculation_definition: {
    // Must have at least one dependency
    if len(dependencies) == 0 {
        _deps: "CALC_DEPENDENCIES_REQUIRED"
    }

    // Pre-aggregated calculations must have materialization
    if evaluation_mode == "pre_aggregated" {
        materialization: _
    }

    // Hybrid mode should have at least one dependency
    if evaluation_mode == "hybrid" && len(dependencies) < 2 {
        _hybrid: "HYBRID_MODE_REQUIRES_MULTIPLE_DEPENDENCIES"
    }
}

// Common calculation patterns
#RatioCalculation: calculation_definition & {
    output_type: "percent" | "number"
    // Ratio calculations typically need exactly 2 dependencies
    dependencies: [_, _]
}

#DifferenceCalculation: calculation_definition & {
    output_type: "number" | "currency"
    // Difference needs 2 dependencies
    dependencies: [_, _]
}

#AggregateCalculation: calculation_definition & {
    // Aggregations can be pre-computed
    evaluation_mode: "pre_aggregated" | "hybrid"
    default_aggregation: "sum" | "avg" | "min" | "max" | "count"
}

// Calculation with window function support
#WindowCalculation: calculation_definition & {
    window_config: {
        partition_by: [...string]
        order_by?: [...string]
        frame?: "rows" | "range"
        frame_start?: string
        frame_end?: string
    }
}
