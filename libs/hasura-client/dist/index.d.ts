export interface HasuraConfig {
    endpoint: string;
    adminSecret?: string;
    headers?: Record<string, string>;
}
export interface QueryOptions {
    headers?: Record<string, string>;
}
export declare class HasuraClient {
    private client;
    constructor(config: HasuraConfig);
    query<T = any>(query: string, variables?: Record<string, any>, options?: QueryOptions): Promise<T>;
    mutate<T = any>(mutation: string, variables?: Record<string, any>, options?: QueryOptions): Promise<T>;
    getFabricModels(tenantId: string, datasourceId: string): Promise<FabricModel[]>;
    getFabricModel(id: string): Promise<FabricModel | null>;
    createFabricModel(model: Omit<FabricModel, 'id' | 'created_at' | 'updated_at'>): Promise<FabricModel>;
    updateFabricModel(id: string, updates: Partial<FabricModel>): Promise<FabricModel>;
    deleteFabricModel(id: string): Promise<{
        id: string;
    }>;
    getBusinessProcesses(tenantId: string, datasourceId: string): Promise<BusinessProcess[]>;
    getBusinessProcess(id: string): Promise<BusinessProcess | null>;
    createBusinessProcess(process: Omit<BusinessProcess, 'id' | 'created_at' | 'updated_at'>): Promise<BusinessProcess>;
    updateBusinessProcess(id: string, updates: Partial<BusinessProcess>): Promise<BusinessProcess>;
    healthCheck(): Promise<boolean>;
    batchQuery<T = any>(queries: Array<{
        query: string;
        variables?: Record<string, any>;
    }>): Promise<T[]>;
}
export declare const QUERIES: {
    GET_FABRIC_MODELS: string;
    GET_FABRIC_MODEL: string;
    GET_BUSINESS_PROCESSES: string;
    GET_BUSINESS_PROCESS: string;
    GET_UMA_ACCOUNTS: string;
    GET_ATTRIBUTION_RESULTS: string;
};
export declare const MUTATIONS: {
    CREATE_FABRIC_MODEL: string;
    UPDATE_FABRIC_MODEL: string;
    DELETE_FABRIC_MODEL: string;
    CREATE_BUSINESS_PROCESS: string;
    UPDATE_BUSINESS_PROCESS: string;
    UPDATE_UMA_STATUS: string;
    INSERT_ATTRIBUTION_RESULT: string;
};
export interface FabricModel {
    id: string;
    name: string;
    description: string;
    tenant_id: string;
    datasource_id: string;
    schema: Record<string, any>;
    created_at: string;
    updated_at: string;
}
export interface BusinessProcess {
    id: string;
    tenant_id: string;
    datasource_id: string;
    process_name: string;
    entity: string;
    description: string;
    steps: any[];
    is_active: boolean;
    created_by: string;
    created_at: string;
    updated_at?: string;
    version: number;
}
export interface UMAAccount {
    id: string;
    aum: number;
    tax_saved: number;
    status: string;
    holdings: Array<{
        symbol: string;
        quantity: number;
        price: number;
        value: number;
        weight: number;
    }>;
    created_at: string;
    updated_at: string;
}
export interface AttributionResult {
    id: string;
    total_return: number;
    benchmark_return: number;
    alpha: number;
    factors: any[];
    created_at: string;
}
//# sourceMappingURL=index.d.ts.map