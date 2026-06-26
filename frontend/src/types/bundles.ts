export type BundleStatus = 'Draft' | 'Certified' | 'Published' | 'Deprecated';

export type AttributeCondition = {
    attribute: string;
    operator: string;
    values: string[];
};

export type BundleRowPolicy = {
    id: string;
    name: string;
    description: string;
    member: string;
    operator: string;
    values: string[];
    conditions: AttributeCondition[];
};

export type BundleColumnPolicy = {
    id: string;
    name: string;
    description: string;
    columns: string[];
    maskType: string;
    maskValue?: string;
    conditions: AttributeCondition[];
};

export interface SemanticObjectReference {
    id: string;
    modelId: string;
    type: 'measure' | 'dimension';
    name?: string;
    title?: string;
    description?: string;
}

export interface DataBundle {
    id: string;
    name: string;
    version: string;
    status: BundleStatus;
    description: string;
    owner: string;
    createdAt: string;
    updatedAt: string;
    measures: SemanticObjectReference[];
    dimensions: SemanticObjectReference[];
    filters: string[];
    rowPolicies: BundleRowPolicy[];
    columnPolicies: BundleColumnPolicy[];
    allowedRoles: string[];
}

// ---- Additional bundle shapes used across the frontend ----

// Shape used by the Views Catalog / Bundle Editor when creating a simple data bundle
export interface BundleViewRefByName { view_name: string; view_id?: string }
export interface BundleViewRefById { view_id: string; view_name?: string }
export type BundleViewRef = BundleViewRefByName | BundleViewRefById;

export interface BundleForm {
    name: string;
    description: string;
    audience: string[];
    view_refs: BundleViewRef[];
}

// Shape used by the semantic registry bundles (BundleExplorer / calculationLibrary)
export interface RegistryFunction {
    name: string;
    class: string;
    badge?: string;
    description?: string;
}

export interface RegistryMetric {
    node_id: string;
    category?: string;
    description?: string;
    badge?: string;
    function_class?: string;
    functions_used?: string[];
    governance?: { status: string };
}

export interface RegistryBundle {
    bundle_id: string;
    domain: string;
    audience: string[];
    version: string;
    owner: string;
    description?: string;
    tags: string[];
    functions?: RegistryFunction[];
    metrics: RegistryMetric[];
}

// Shape used by calculationLibrary's bundle JSON
export interface BundleLibrary {
    bundle_id: string;
    description: string;
    version: string;
    last_updated: string;
    engines: string[];
    total_metrics: number;
    domains: string[];
    metrics: any[];
    function_mapping: any[];
}

// Shape used in private-markets feature bundles
export interface PrivateMarketsBundle {
    id: string;
    name: string;
    audience: 'lp' | 'gp' | 'fof';
    version: string;
    modules: Array<{ id: string; name: string; type: string; config: any }>;
    metrics: Array<{ node_id: string; name: string; type: string; category?: string; subcategory?: string; financial_calc?: any; description?: string }>;
    governance?: any;
}
