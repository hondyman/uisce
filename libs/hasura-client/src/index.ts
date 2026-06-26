// Hasura GraphQL Client for SemLayer
// Provides typed GraphQL operations for data access

import { GraphQLClient, gql } from 'graphql-request';

export interface HasuraConfig {
  endpoint: string;
  adminSecret?: string;
  headers?: Record<string, string>;
}

export interface QueryOptions {
  headers?: Record<string, string>;
}

export class HasuraClient {
  private client: GraphQLClient;

  constructor(config: HasuraConfig) {
    const headers: Record<string, string> = {
      ...config.headers,
    };

    if (config.adminSecret) {
      headers['x-hasura-admin-secret'] = config.adminSecret;
    }

    this.client = new GraphQLClient(config.endpoint, {
      headers,
      timeout: 30000,
    });
  }

  async query<T = any>(query: string, variables?: Record<string, any>, options?: QueryOptions): Promise<T> {
    try {
      const headers = options?.headers || {};
      return await this.client.request(query, variables, headers);
    } catch (error) {
      console.error('Hasura query failed:', error);
      throw new Error(`Hasura query failed: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  }

  async mutate<T = any>(mutation: string, variables?: Record<string, any>, options?: QueryOptions): Promise<T> {
    try {
      const headers = options?.headers || {};
      return await this.client.request(mutation, variables, headers);
    } catch (error) {
      console.error('Hasura mutation failed:', error);
      throw new Error(`Hasura mutation failed: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  }

  // Convenience methods for common operations
  async getFabricModels(tenantId: string, datasourceId: string): Promise<FabricModel[]> {
    const result = await this.query(QUERIES.GET_FABRIC_MODELS, { tenantId, datasourceId });
    return result.fabric_models || [];
  }

  async getFabricModel(id: string): Promise<FabricModel | null> {
    const result = await this.query(QUERIES.GET_FABRIC_MODEL, { id });
    return result.fabric_models_by_pk || null;
  }

  async createFabricModel(model: Omit<FabricModel, 'id' | 'created_at' | 'updated_at'>): Promise<FabricModel> {
    const result = await this.mutate(MUTATIONS.CREATE_FABRIC_MODEL, { model });
    return result.insert_fabric_models_one;
  }

  async updateFabricModel(id: string, updates: Partial<FabricModel>): Promise<FabricModel> {
    const result = await this.mutate(MUTATIONS.UPDATE_FABRIC_MODEL, { id, updates });
    return result.update_fabric_models_by_pk;
  }

  async deleteFabricModel(id: string): Promise<{ id: string }> {
    const result = await this.mutate(MUTATIONS.DELETE_FABRIC_MODEL, { id });
    return result.delete_fabric_models_by_pk;
  }

  async getBusinessProcesses(tenantId: string, datasourceId: string): Promise<BusinessProcess[]> {
    const result = await this.query(QUERIES.GET_BUSINESS_PROCESSES, { tenantId, datasourceId });
    return result.business_processes || [];
  }

  async getBusinessProcess(id: string): Promise<BusinessProcess | null> {
    const result = await this.query(QUERIES.GET_BUSINESS_PROCESS, { id });
    return result.business_processes_by_pk || null;
  }

  async createBusinessProcess(process: Omit<BusinessProcess, 'id' | 'created_at' | 'updated_at'>): Promise<BusinessProcess> {
    const result = await this.mutate(MUTATIONS.CREATE_BUSINESS_PROCESS, { process });
    return result.insert_business_processes_one;
  }

  async updateBusinessProcess(id: string, updates: Partial<BusinessProcess>): Promise<BusinessProcess> {
    const result = await this.mutate(MUTATIONS.UPDATE_BUSINESS_PROCESS, { id, updates });
    return result.update_business_processes_by_pk;
  }

  // Health check method
  async healthCheck(): Promise<boolean> {
    try {
      // Simple query to check if Hasura is responding
      await this.query(gql`query { __typename }`);
      return true;
    } catch (error) {
      console.error('Hasura health check failed:', error);
      return false;
    }
  }

  // Batch operations
  async batchQuery<T = any>(queries: Array<{ query: string; variables?: Record<string, any> }>): Promise<T[]> {
    const results: T[] = [];
    for (const { query, variables } of queries) {
      const result = await this.query<T>(query, variables);
      results.push(result);
    }
    return results;
  }
}

// Predefined queries for common operations
export const QUERIES = {
  // Fabric/Semantic Model queries
  GET_FABRIC_MODELS: gql`
    query GetFabricModels($tenantId: String!, $datasourceId: String!) {
      fabric_models(
        where: {
          tenant_id: { _eq: $tenantId }
          datasource_id: { _eq: $datasourceId }
        }
      ) {
        id
        name
        description
        schema
        created_at
        updated_at
      }
    }
  `,

  GET_FABRIC_MODEL: gql`
    query GetFabricModel($id: String!) {
      fabric_models_by_pk(id: $id) {
        id
        name
        description
        tenant_id
        datasource_id
        schema
        created_at
        updated_at
      }
    }
  `,

  // Business Process queries
  GET_BUSINESS_PROCESSES: gql`
    query GetBusinessProcesses($tenantId: String!, $datasourceId: String!) {
      business_processes(
        where: {
          tenant_id: { _eq: $tenantId }
          datasource_id: { _eq: $datasourceId }
        }
      ) {
        id
        process_name
        entity
        description
        steps
        is_active
        created_by
        created_at
        updated_at
        version
      }
    }
  `,

  GET_BUSINESS_PROCESS: gql`
    query GetBusinessProcess($id: String!) {
      business_processes_by_pk(id: $id) {
        id
        tenant_id
        datasource_id
        process_name
        entity
        description
        steps
        is_active
        created_by
        created_at
        updated_at
        version
      }
    }
  `,

  // Wealth Management queries
  GET_UMA_ACCOUNTS: gql`
    query GetUMAAccounts($tenantId: String!) {
      uma_accounts(where: { tenant_id: { _eq: $tenantId } }) {
        id
        aum
        tax_saved
        status
        holdings {
          symbol
          quantity
          price
          value
          weight
        }
        created_at
        updated_at
      }
    }
  `,

  GET_ATTRIBUTION_RESULTS: gql`
    query GetAttributionResults($portfolioId: String!) {
      attribution_results(
        where: { portfolio_id: { _eq: $portfolioId } }
        order_by: { created_at: desc }
        limit: 10
      ) {
        id
        total_return
        benchmark_return
        alpha
        factors
        created_at
      }
    }
  `,
};

// Predefined mutations
export const MUTATIONS = {
  // Fabric mutations
  CREATE_FABRIC_MODEL: gql`
    mutation CreateFabricModel($model: fabric_models_insert_input!) {
      insert_fabric_models_one(object: $model) {
        id
        name
        description
        schema
        created_at
      }
    }
  `,

  UPDATE_FABRIC_MODEL: gql`
    mutation UpdateFabricModel($id: String!, $updates: fabric_models_set_input!) {
      update_fabric_models_by_pk(pk_columns: { id: $id }, _set: $updates) {
        id
        name
        description
        schema
        updated_at
      }
    }
  `,

  DELETE_FABRIC_MODEL: gql`
    mutation DeleteFabricModel($id: String!) {
      delete_fabric_models_by_pk(id: $id) {
        id
      }
    }
  `,

  // Business Process mutations
  CREATE_BUSINESS_PROCESS: gql`
    mutation CreateBusinessProcess($process: business_processes_insert_input!) {
      insert_business_processes_one(object: $process) {
        id
        process_name
        description
        steps
        created_at
      }
    }
  `,

  UPDATE_BUSINESS_PROCESS: gql`
    mutation UpdateBusinessProcess($id: String!, $updates: business_processes_set_input!) {
      update_business_processes_by_pk(pk_columns: { id: $id }, _set: $updates) {
        id
        process_name
        description
        steps
        updated_at
      }
    }
  `,

  // Wealth Management mutations
  UPDATE_UMA_STATUS: gql`
    mutation UpdateUMAStatus($id: String!, $status: String!) {
      update_uma_accounts_by_pk(pk_columns: { id: $id }, _set: { status: $status }) {
        id
        status
        updated_at
      }
    }
  `,

  INSERT_ATTRIBUTION_RESULT: gql`
    mutation InsertAttributionResult($result: attribution_results_insert_input!) {
      insert_attribution_results_one(object: $result) {
        id
        total_return
        benchmark_return
        alpha
        factors
        created_at
      }
    }
  `,
};

// Type definitions for responses
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