export type Severity = "error" | "warning" | "info";
export type RuleType = "sql" | "dsl" | "wasm" | "cue" | "starlark";
export type InheritMode = "custom" | "extend" | "override";

export interface Rule {
    id: string;
    tenant_id: string;
    datasource_id?: string | null;

    rule_category?: string | null;
    description?: string | null;

    name: string;
    target_entity?: string | null;
    target_entity_id?: string | null;
    target_entity_ids?: string[] | null;
    field_path?: string[] | null;

    rule_type: RuleType;
    script_content?: string | null;
    condition_json?: any | null;
    compiled_sql?: string | null;
    compiled_wasm?: string | null;

    severity: Severity;
    is_active: boolean;
    evaluation_order: number;

    execute_client_side: boolean;
    execute_server_side: boolean;
    run_on_blur: boolean;
    run_on_change: boolean;
    run_on_submit: boolean;

    is_core: boolean;
    is_override?: boolean;
    can_edit?: boolean;
    can_delete?: boolean;
    can_override?: boolean;
    core_rule_id?: string | null;
    inherit_mode: InheritMode;
    extension_script_content?: string | null;
    extension_condition_json?: any | null;
    is_core_locked: boolean;

    version: number;
    status: string;
    parent_rule_id?: string | null;
    user_friendly_name?: string | null;
    user_friendly_description?: string | null;
    remediation_hint?: string | null;

    created_by?: string | null;
    created_at: string;
    updated_at: string;
}

export interface RulePreviewResult {
    classification: "sql" | "wasm" | "mixed" | "invalid";
    sql?: string;
    cueErrors?: string[];
    dslErrors?: string[];
}

export interface RuleSimulationResult {
    id: string;
    status: string;
    sampleSize: number;
    failure_count: number;
    startedAt: string;
    completedAt?: string;
}

// ValidationRule represents the resolved rule for the UI (Validations Tab)
// including inherited semantic-term rules and tenant overrides.
export interface ValidationRule {
    id: string;
    name: string;
    expression: string;
    severity: string;
    source: "bo" | "field" | "semantic_term" | "tenant_override";
    scope: "local" | "inherited" | "override";
    readOnly: boolean;
    overriddenCoreRuleId?: string;
    promotionStatus?: string;
}

export interface ValidationPreviewResult {
    generated_sql: string;
    driving_table: string;
    join_path: string;
}

export interface RuleDiff {
    base: RuleVersion;
    current: RuleVersion;
    diffs: DiffField[];
}

export interface RuleVersion {
    name: string;
    expression: string;
    severity: string;
    scope: string;
    version: number;
    archived_at: string;
}

export interface DiffField {
    field: string;
    old_value: string;
    new_value: string;
}

export interface RuleLineage {
    inherited_by_bos: string[];
    overrides: string[];
    semantic_terms: string[];
}

export interface ReferencedField {
    semantic_term_id: string;
    bo_field_id: string;
    business_object_id: string;
    field_path: string[];
}

export interface RuleTestMessage {
    type: string;
    message: string;
}

export interface RuleTestResponse {
    mode: "sql" | "wasm" | "";
    fired: boolean;
    messages: RuleTestMessage[];
    referenced_fields: ReferencedField[];
    debug?: any;
}

// Impact Analysis types
export interface CatalogNodeInfo {
    id: string;
    node_name: string;
    display_name: string;
}

export interface DependentRule {
    id: string;
    rule_name: string;
    link_type: string; // 'uses_field' | 'override'
}

export interface OverrideInfo {
    id: string;
    tenant_id: string;
    rule_name: string;
}

export interface ImpactResult {
    rule_id: string;
    fields: ReferencedField[];
    semantic_terms: CatalogNodeInfo[];
    business_objects: CatalogNodeInfo[];
    dependent_rules: DependentRule[];
    overrides: OverrideInfo[];
}
