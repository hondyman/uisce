"use strict";
// Hasura GraphQL Client for SemLayer
// Provides typed GraphQL operations for data access
Object.defineProperty(exports, "__esModule", { value: true });
exports.MUTATIONS = exports.QUERIES = exports.HasuraClient = void 0;
const graphql_request_1 = require("graphql-request");
class HasuraClient {
    constructor(config) {
        const headers = {
            ...config.headers,
        };
        if (config.adminSecret) {
            headers['x-hasura-admin-secret'] = config.adminSecret;
        }
        this.client = new graphql_request_1.GraphQLClient(config.endpoint, {
            headers,
            timeout: 30000,
        });
    }
    async query(query, variables, options) {
        try {
            const headers = options?.headers || {};
            return await this.client.request(query, variables, headers);
        }
        catch (error) {
            console.error('Hasura query failed:', error);
            throw new Error(`Hasura query failed: ${error instanceof Error ? error.message : 'Unknown error'}`);
        }
    }
    async mutate(mutation, variables, options) {
        try {
            const headers = options?.headers || {};
            return await this.client.request(mutation, variables, headers);
        }
        catch (error) {
            console.error('Hasura mutation failed:', error);
            throw new Error(`Hasura mutation failed: ${error instanceof Error ? error.message : 'Unknown error'}`);
        }
    }
    // Convenience methods for common operations
    async getFabricModels(tenantId, datasourceId) {
        const result = await this.query(exports.QUERIES.GET_FABRIC_MODELS, { tenantId, datasourceId });
        return result.fabric_models || [];
    }
    async getFabricModel(id) {
        const result = await this.query(exports.QUERIES.GET_FABRIC_MODEL, { id });
        return result.fabric_models_by_pk || null;
    }
    async createFabricModel(model) {
        const result = await this.mutate(exports.MUTATIONS.CREATE_FABRIC_MODEL, { model });
        return result.insert_fabric_models_one;
    }
    async updateFabricModel(id, updates) {
        const result = await this.mutate(exports.MUTATIONS.UPDATE_FABRIC_MODEL, { id, updates });
        return result.update_fabric_models_by_pk;
    }
    async deleteFabricModel(id) {
        const result = await this.mutate(exports.MUTATIONS.DELETE_FABRIC_MODEL, { id });
        return result.delete_fabric_models_by_pk;
    }
    async getBusinessProcesses(tenantId, datasourceId) {
        const result = await this.query(exports.QUERIES.GET_BUSINESS_PROCESSES, { tenantId, datasourceId });
        return result.business_processes || [];
    }
    async getBusinessProcess(id) {
        const result = await this.query(exports.QUERIES.GET_BUSINESS_PROCESS, { id });
        return result.business_processes_by_pk || null;
    }
    async createBusinessProcess(process) {
        const result = await this.mutate(exports.MUTATIONS.CREATE_BUSINESS_PROCESS, { process });
        return result.insert_business_processes_one;
    }
    async updateBusinessProcess(id, updates) {
        const result = await this.mutate(exports.MUTATIONS.UPDATE_BUSINESS_PROCESS, { id, updates });
        return result.update_business_processes_by_pk;
    }
    // Health check method
    async healthCheck() {
        try {
            // Simple query to check if Hasura is responding
            await this.query((0, graphql_request_1.gql) `query { __typename }`);
            return true;
        }
        catch (error) {
            console.error('Hasura health check failed:', error);
            return false;
        }
    }
    // Batch operations
    async batchQuery(queries) {
        const results = [];
        for (const { query, variables } of queries) {
            const result = await this.query(query, variables);
            results.push(result);
        }
        return results;
    }
}
exports.HasuraClient = HasuraClient;
// Predefined queries for common operations
exports.QUERIES = {
    // Fabric/Semantic Model queries
    GET_FABRIC_MODELS: (0, graphql_request_1.gql) `
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
    GET_FABRIC_MODEL: (0, graphql_request_1.gql) `
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
    GET_BUSINESS_PROCESSES: (0, graphql_request_1.gql) `
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
    GET_BUSINESS_PROCESS: (0, graphql_request_1.gql) `
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
    GET_UMA_ACCOUNTS: (0, graphql_request_1.gql) `
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
    GET_ATTRIBUTION_RESULTS: (0, graphql_request_1.gql) `
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
exports.MUTATIONS = {
    // Fabric mutations
    CREATE_FABRIC_MODEL: (0, graphql_request_1.gql) `
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
    UPDATE_FABRIC_MODEL: (0, graphql_request_1.gql) `
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
    DELETE_FABRIC_MODEL: (0, graphql_request_1.gql) `
    mutation DeleteFabricModel($id: String!) {
      delete_fabric_models_by_pk(id: $id) {
        id
      }
    }
  `,
    // Business Process mutations
    CREATE_BUSINESS_PROCESS: (0, graphql_request_1.gql) `
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
    UPDATE_BUSINESS_PROCESS: (0, graphql_request_1.gql) `
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
    UPDATE_UMA_STATUS: (0, graphql_request_1.gql) `
    mutation UpdateUMAStatus($id: String!, $status: String!) {
      update_uma_accounts_by_pk(pk_columns: { id: $id }, _set: { status: $status }) {
        id
        status
        updated_at
      }
    }
  `,
    INSERT_ATTRIBUTION_RESULT: (0, graphql_request_1.gql) `
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
//# sourceMappingURL=index.js.map