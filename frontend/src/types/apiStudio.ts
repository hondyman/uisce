export interface APIEndpoint {
    id: string;
    env: string;
    tenant_id: string;
    name: string;
    path: string;
    method: 'GET' | 'POST' | 'PUT' | 'DELETE';
    type: 'rest' | 'graphql';
    bo_name: string;
    fields: string[]; // BO fields exposed
    filters: any;
    pagination: {
        type: 'offset' | 'cursor';
        default_limit: number;
    };
    auth_policy?: string;
    version: number; // Incremental version for persistence
    is_active: boolean;
    status: 'active' | 'deprecated' | 'retired';
    semantic_version: string; // "v1", "v2", etc.
    previous_version_id?: string;
    owner_team?: string;
    deprecated_at?: string;
    retired_at?: string;
    request_schema_id?: string;
    response_schema_id?: string;
    created_at: string;
    created_by: string;
}

export interface APICatalog {
    id: string;
    env: string;
    tenant_id: string;
    name: string;
    description: string;
    created_at: string;
    created_by: string;
}

export interface APITest {
    id: string;
    env: string;
    tenant_id: string;
    endpoint_id: string;
    name: string;
    type: 'contract' | 'latency' | 'pii' | 'regression';
    definition: any;
    created_at: string;
    created_by: string;
    enabled: boolean;
}

export interface APITestRun {
    id: string;
    api_test_id: string;
    env: string;
    tenant_id: string;
    status: 'pending' | 'running' | 'passed' | 'failed';
    started_at?: string;
    finished_at?: string;
    result: any;
    logs: string[];
}
